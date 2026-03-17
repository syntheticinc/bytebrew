//go:build lsp

package lsp

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const testGoMain = `package main

type Repository interface {
	GetByID(id string) string
}

type InMemoryRepo struct{}

func (r *InMemoryRepo) GetByID(id string) string {
	return "item-" + id
}

func NewRepo() Repository {
	return &InMemoryRepo{}
}

func main() {
	repo := NewRepo()
	_ = repo.GetByID("123")
}
`

const testGoMod = `module testproject

go 1.24
`

// setupTestGoProject creates a temporary Go project and returns its root path.
func setupTestGoProject(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()

	require.NoError(t, os.WriteFile(filepath.Join(dir, "go.mod"), []byte(testGoMod), 0644))
	require.NoError(t, os.WriteFile(filepath.Join(dir, "main.go"), []byte(testGoMain), 0644))

	return dir
}

// requireGopls skips the test if gopls is not available.
func requireGopls(t *testing.T) {
	t.Helper()
	if whichBin("gopls") == "" {
		t.Skip("gopls not available")
	}
}

// startGoplsClient directly creates and initializes a gopls client for the given project root.
// This bypasses Service to have more control over initialization and retries.
func startGoplsClient(t *testing.T, projectRoot string) *Client {
	t.Helper()

	bin := whichBin("gopls")
	require.NotEmpty(t, bin, "gopls must be available")

	cmd := exec.Command(bin, "serve")
	cmd.Dir = projectRoot

	client, err := NewClient("go", projectRoot, cmd)
	require.NoError(t, err)

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	err = client.Initialize(ctx, projectRoot)
	require.NoError(t, err)

	// Open the main file so gopls indexes it
	mainPath := filepath.Join(projectRoot, "main.go")
	err = client.DidOpen(ctx, mainPath)
	require.NoError(t, err)

	// gopls may or may not send $/progress for small projects.
	// Wait briefly, but don't rely on it.
	client.WaitForReady(5 * time.Second)

	return client
}

// retryDefinition retries a definition request until it returns a non-empty result or timeout.
// gopls may need time to index even after initialization.
func retryDefinition(t *testing.T, client *Client, uri string, pos Position, timeout time.Duration) []Location {
	t.Helper()
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		locations, err := client.Definition(ctx, uri, pos)
		cancel()
		if err == nil && len(locations) > 0 {
			return locations
		}
		t.Logf("retryDefinition: attempt got err=%v, locations=%d, retrying...", err, len(locations))
		time.Sleep(2 * time.Second)
	}
	t.Fatalf("retryDefinition: timed out after %v", timeout)
	return nil
}

// retryReferences retries a references request until it returns enough results or timeout.
func retryReferences(t *testing.T, client *Client, uri string, pos Position, minCount int, timeout time.Duration) []Location {
	t.Helper()
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		locations, err := client.References(ctx, uri, pos)
		cancel()
		if err == nil && len(locations) >= minCount {
			return locations
		}
		t.Logf("retryReferences: attempt got err=%v, locations=%d, retrying...", err, len(locations))
		time.Sleep(2 * time.Second)
	}
	t.Fatalf("retryReferences: timed out after %v waiting for >= %d results", timeout, minCount)
	return nil
}

// retryImplementation retries an implementation request until it returns a non-empty result or timeout.
func retryImplementation(t *testing.T, client *Client, uri string, pos Position, timeout time.Duration) []Location {
	t.Helper()
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		locations, err := client.Implementation(ctx, uri, pos)
		cancel()
		if err == nil && len(locations) > 0 {
			return locations
		}
		t.Logf("retryImplementation: attempt got err=%v, locations=%d, retrying...", err, len(locations))
		time.Sleep(2 * time.Second)
	}
	t.Fatalf("retryImplementation: timed out after %v", timeout)
	return nil
}

// findLineAndCharacter returns the 0-based line and character of the first occurrence
// of target in the source code.
func findLineAndCharacter(source, target string) (int, int) {
	lines := strings.Split(source, "\n")
	for i, line := range lines {
		idx := strings.Index(line, target)
		if idx >= 0 {
			return i, idx
		}
	}
	return -1, -1
}

// TC-L-01: Definition of GetByID method call in main() should resolve to InMemoryRepo.GetByID.
func TestLSP_Definition_Go(t *testing.T) {
	requireGopls(t)

	projectRoot := setupTestGoProject(t)
	client := startGoplsClient(t, projectRoot)
	defer func() {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		_ = client.Shutdown(ctx)
	}()

	mainPath := filepath.Join(projectRoot, "main.go")
	mainURI := pathToURI(mainPath)

	// Find "repo.GetByID" call in main() — position on "GetByID" in the call
	line, char := findLineAndCharacter(testGoMain, `repo.GetByID("123")`)
	require.NotEqual(t, -1, line, "could not find repo.GetByID call in test source")
	char += len("repo.")

	pos := Position{Line: line, Character: char}
	locations := retryDefinition(t, client, mainURI, pos, 60*time.Second)

	// repo has type Repository (interface), so gopls resolves definition to the interface method.
	loc := locations[0]
	assert.Contains(t, loc.URI, "main.go", "definition should be in main.go")
	assert.Greater(t, loc.Range.Start.Line, 0, "definition line should be > 0")

	// Definition should point to GetByID in the Repository interface declaration.
	interfaceLine, _ := findLineAndCharacter(testGoMain, "GetByID(id string) string")
	assert.Equal(t, interfaceLine, loc.Range.Start.Line,
		"definition of repo.GetByID should point to interface method (repo is typed as Repository)")
}

// TC-L-02: References on GetByID should find at least 2 references
// (interface declaration, implementation, call site).
func TestLSP_References_Go(t *testing.T) {
	requireGopls(t)

	projectRoot := setupTestGoProject(t)
	client := startGoplsClient(t, projectRoot)
	defer func() {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		_ = client.Shutdown(ctx)
	}()

	mainPath := filepath.Join(projectRoot, "main.go")
	mainURI := pathToURI(mainPath)

	// Position on GetByID in the interface declaration
	line, char := findLineAndCharacter(testGoMain, "GetByID(id string) string")
	require.NotEqual(t, -1, line, "could not find GetByID in interface")

	pos := Position{Line: line, Character: char}
	locations := retryReferences(t, client, mainURI, pos, 2, 60*time.Second)

	assert.GreaterOrEqual(t, len(locations), 2,
		"expected at least 2 references for GetByID (interface decl + impl + call), got %d", len(locations))
}

// TC-L-03: Implementation on Repository interface should find InMemoryRepo.
func TestLSP_Implementation_Go(t *testing.T) {
	requireGopls(t)

	projectRoot := setupTestGoProject(t)
	client := startGoplsClient(t, projectRoot)
	defer func() {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		_ = client.Shutdown(ctx)
	}()

	mainPath := filepath.Join(projectRoot, "main.go")
	mainURI := pathToURI(mainPath)

	// Position on "Repository" in `type Repository interface {`
	line, char := findLineAndCharacter(testGoMain, "type Repository interface")
	require.NotEqual(t, -1, line, "could not find Repository interface")
	char += len("type ")

	pos := Position{Line: line, Character: char}
	locations := retryImplementation(t, client, mainURI, pos, 60*time.Second)

	// One of the locations should point to InMemoryRepo
	inMemoryLine, _ := findLineAndCharacter(testGoMain, "type InMemoryRepo struct")
	found := false
	for _, loc := range locations {
		if strings.Contains(loc.URI, "main.go") && loc.Range.Start.Line == inMemoryLine {
			found = true
			break
		}
	}
	assert.True(t, found,
		"expected InMemoryRepo in implementation results at line %d, got: %v", inMemoryLine, locations)
}

// TC-L-06: Service for unsupported file extension returns "no LSP server available".
func TestLSP_NotInstalled(t *testing.T) {
	svc := NewService(t.TempDir())
	defer svc.Dispose()

	ctx := context.Background()
	_, err := svc.Definition(ctx, "/tmp/project/file.xyz", 0, 0)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "no LSP server available")
}

// TC-L-07: First request after gopls start should succeed (WaitForReady works).
func TestLSP_GoplsWarmup(t *testing.T) {
	requireGopls(t)

	projectRoot := setupTestGoProject(t)

	svc := NewService(projectRoot)
	defer svc.Dispose()

	mainPath := filepath.Join(projectRoot, "main.go")

	// First request triggers gopls startup. Even if gopls isn't fully indexed,
	// the request should complete without error (may return empty results).
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	line, char := findLineAndCharacter(testGoMain, "func main()")
	require.NotEqual(t, -1, line)
	char += len("func ")

	// The key assertion is that this does NOT return an error — gopls started and responds.
	_, err := svc.Definition(ctx, mainPath, line, char)
	require.NoError(t, err, "first request to gopls should succeed (no error)")

	// Verify client is now active
	assert.True(t, svc.HasActiveClients(), "expected active LSP client after warmup")
}

// TC-L-09: resolveSymbolCharacter is tested via tools/lsp_position_test.go (same build tag).
// See internal/infrastructure/tools/lsp_position_test.go for that test.

// TC-L-10: WholeWordMatch is tested via tools/lsp_position_test.go (same build tag).
// See internal/infrastructure/tools/lsp_position_test.go for that test.

// ---------------------------------------------------------------------------
// Python test source for TC-L-05
// ---------------------------------------------------------------------------

const testPythonMain = `class Repository:
    def get_by_id(self, id: str) -> str:
        return "item-" + id

def main():
    repo = Repository()
    result = repo.get_by_id("123")
`

// setupTestPythonProject creates a temporary Python project and returns its root path.
func setupTestPythonProject(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()

	require.NoError(t, os.WriteFile(filepath.Join(dir, "requirements.txt"), []byte(""), 0644))
	require.NoError(t, os.WriteFile(filepath.Join(dir, "main.py"), []byte(testPythonMain), 0644))

	return dir
}

// requirePyright skips the test if pyright-langserver is not available.
func requirePyright(t *testing.T) {
	t.Helper()
	if whichBin("pyright-langserver") == "" {
		t.Skip("pyright-langserver not available")
	}
}

// startPyrightClient creates and initializes a pyright client for the given project root.
func startPyrightClient(t *testing.T, projectRoot string) *Client {
	t.Helper()

	bin := whichBin("pyright-langserver")
	require.NotEmpty(t, bin, "pyright-langserver must be available")

	cmd := exec.Command(bin, "--stdio")
	cmd.Dir = projectRoot

	client, err := NewClient("python", projectRoot, cmd)
	require.NoError(t, err)

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	err = client.Initialize(ctx, projectRoot)
	require.NoError(t, err)

	mainPath := filepath.Join(projectRoot, "main.py")
	err = client.DidOpen(ctx, mainPath)
	require.NoError(t, err)

	client.WaitForReady(5 * time.Second)

	return client
}

// TC-L-05: Definition of get_by_id call in Python main() should resolve to Repository.get_by_id.
func TestLSP_Definition_Python(t *testing.T) {
	requirePyright(t)

	projectRoot := setupTestPythonProject(t)
	client := startPyrightClient(t, projectRoot)
	defer func() {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		_ = client.Shutdown(ctx)
	}()

	mainPath := filepath.Join(projectRoot, "main.py")
	mainURI := pathToURI(mainPath)

	// Find "repo.get_by_id" call in main() — position on "get_by_id" in the call
	line, char := findLineAndCharacter(testPythonMain, `repo.get_by_id("123")`)
	require.NotEqual(t, -1, line, "could not find repo.get_by_id call in test source")
	char += len("repo.")

	pos := Position{Line: line, Character: char}
	locations := retryDefinition(t, client, mainURI, pos, 60*time.Second)

	require.NotEmpty(t, locations, "definition should return at least one location")

	loc := locations[0]
	assert.Contains(t, loc.URI, "main.py", "definition should be in main.py")

	// Definition should point to the get_by_id method in the Repository class
	defLine, _ := findLineAndCharacter(testPythonMain, "def get_by_id(self")
	assert.Equal(t, defLine, loc.Range.Start.Line,
		"definition of repo.get_by_id should point to class method definition (line %d), got line %d",
		defLine, loc.Range.Start.Line)
}

// TC-L-08: Retry on empty — requesting definition at a position with no navigable symbol
// should complete without error, returning empty results. This exercises the retry path
// in LocalClientOperationsProxy (line 259-266): if locations are empty and active clients
// exist, it waits 2s and retries — both attempts return empty for a non-symbol position.
func TestLSP_RetryOnEmpty(t *testing.T) {
	requireGopls(t)

	projectRoot := setupTestGoProject(t)

	svc := NewService(projectRoot)
	defer svc.Dispose()

	mainPath := filepath.Join(projectRoot, "main.go")

	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	// First, do a valid request to warm up gopls.
	line, char := findLineAndCharacter(testGoMain, "func main()")
	require.NotEqual(t, -1, line)
	_, err := svc.Definition(ctx, mainPath, line, char+len("func "))
	require.NoError(t, err, "warmup request should succeed")
	require.True(t, svc.HasActiveClients(), "gopls should be active after warmup")

	// Now request definition at line 0, char 0 — the "package" keyword.
	// gopls returns empty locations for this (no navigable definition).
	locations, err := svc.Definition(ctx, mainPath, 0, 0)
	require.NoError(t, err, "definition request should not error even for non-symbol position")

	assert.Empty(t, locations,
		"definition at main.go:0:0 (package keyword) should return empty")

	// The LSP client should remain active after returning empty results.
	assert.True(t, svc.HasActiveClients(),
		"LSP client should remain active after returning empty results")
}

// dumpLocations formats locations for debugging.
func dumpLocations(locations []Location) string {
	var parts []string
	for _, loc := range locations {
		parts = append(parts, fmt.Sprintf("%s:%d:%d", loc.URI, loc.Range.Start.Line+1, loc.Range.Start.Character))
	}
	return strings.Join(parts, "\n")
}
