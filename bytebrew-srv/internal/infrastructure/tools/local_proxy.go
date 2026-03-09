package tools

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"time"

	pb "github.com/syntheticinc/bytebrew/bytebrew-srv/api/proto/gen"
	"github.com/syntheticinc/bytebrew/bytebrew-srv/internal/infrastructure/indexing"
	"github.com/syntheticinc/bytebrew/bytebrew-srv/internal/infrastructure/lsp"
	"github.com/syntheticinc/bytebrew/bytebrew-srv/internal/infrastructure/shell"
)

// LocalClientOperationsProxy implements ClientOperationsProxy with local filesystem operations.
// Used when tools execute on the server directly instead of proxying to CLI via gRPC.
type LocalClientOperationsProxy struct {
	projectRoot  string
	shellManager *shell.SessionManager
	chunkStore   *indexing.ChunkStore
	embedder     *indexing.EmbeddingsClient
	lspService   *lsp.Service
}

// LocalProxyOption configures optional dependencies for LocalClientOperationsProxy.
type LocalProxyOption func(*LocalClientOperationsProxy)

// WithChunkStore sets the chunk store for symbol and code search.
func WithChunkStore(store *indexing.ChunkStore) LocalProxyOption {
	return func(p *LocalClientOperationsProxy) { p.chunkStore = store }
}

// WithEmbedder sets the embeddings client for semantic search.
func WithEmbedder(embedder *indexing.EmbeddingsClient) LocalProxyOption {
	return func(p *LocalClientOperationsProxy) { p.embedder = embedder }
}

// WithLspService sets the LSP service for code navigation.
func WithLspService(svc *lsp.Service) LocalProxyOption {
	return func(p *LocalClientOperationsProxy) { p.lspService = svc }
}

// NewLocalClientOperationsProxy creates a proxy that operates on the local filesystem.
func NewLocalClientOperationsProxy(projectRoot string, opts ...LocalProxyOption) *LocalClientOperationsProxy {
	p := &LocalClientOperationsProxy{
		projectRoot:  projectRoot,
		shellManager: shell.NewSessionManager(),
	}
	for _, opt := range opts {
		opt(p)
	}
	return p
}

// EditFile reads a file, applies a fuzzy replacement, and writes the result back.
// Replace errors are returned as result strings (not Go errors) so the LLM can read them.
func (p *LocalClientOperationsProxy) EditFile(ctx context.Context, _, filePath, oldString, newString string, replaceAll bool) (string, error) {
	resolved := p.resolvePath(filePath)

	data, err := os.ReadFile(resolved)
	if os.IsNotExist(err) {
		return fmt.Sprintf("[ERROR] File not found: %s. Use write_file to create new files.", filePath), nil
	}
	if err != nil {
		return "", fmt.Errorf("read file %s: %w", filePath, err)
	}

	content := string(data)
	newContent, replaceErr := Replace(content, oldString, newString, replaceAll)
	if replaceErr != nil {
		return replaceErr.Error(), nil
	}

	if err := os.WriteFile(resolved, []byte(newContent), 0644); err != nil {
		return "", fmt.Errorf("write file %s: %w", filePath, err)
	}

	oldLineCount := strings.Count(content, "\n")
	newLineCount := strings.Count(newContent, "\n")
	diff := newLineCount - oldLineCount

	diffStr := "±0"
	if diff > 0 {
		diffStr = fmt.Sprintf("+%d", diff)
	} else if diff < 0 {
		diffStr = fmt.Sprintf("%d", diff)
	}

	relPath := p.relativePath(resolved)
	slog.InfoContext(ctx, "edited file", "path", relPath, "diff_lines", diffStr)
	return fmt.Sprintf("Edit applied: %s (%s lines)", relPath, diffStr), nil
}

// SearchCode is implemented in local_symbol_ops.go
// SymbolSearch is implemented in local_symbol_ops.go
// GrepSearch is implemented in grep_search.go

// ExecuteSubQueries runs sub-queries (symbol, vector, grep) locally in parallel.
func (p *LocalClientOperationsProxy) ExecuteSubQueries(ctx context.Context, _ string, subQueries []*pb.SubQuery) ([]*pb.SubResult, error) {
	results := make([]*pb.SubResult, len(subQueries))

	var wg sync.WaitGroup
	for i, sq := range subQueries {
		wg.Add(1)
		go func(idx int, q *pb.SubQuery) {
			defer wg.Done()
			results[idx] = p.executeSubQuery(ctx, q)
		}(i, sq)
	}
	wg.Wait()

	return results, nil
}

// executeSubQuery dispatches a single sub-query to the appropriate local search.
func (p *LocalClientOperationsProxy) executeSubQuery(ctx context.Context, q *pb.SubQuery) *pb.SubResult {
	result := &pb.SubResult{Type: q.Type}

	switch q.Type {
	case "symbol":
		output, err := p.SymbolSearch(ctx, "", q.Query, q.Limit, nil)
		if err != nil {
			result.Error = err.Error()
			return result
		}
		result.Result = output
		result.Count = int32(strings.Count(output, "\n\n") + 1)
		if isNoResultsMessage(output) {
			result.Count = 0
		}

	case "vector":
		data, err := p.SearchCode(ctx, "", q.Query, "", q.Limit, 0)
		if err != nil {
			result.Error = err.Error()
			return result
		}
		result.Result = string(data)
		// Count entries in JSON array
		result.Count = int32(strings.Count(result.Result, `"file_path"`))

	case "grep":
		output, err := p.GrepSearch(ctx, "", q.Query, q.Limit, nil, false)
		if err != nil {
			result.Error = err.Error()
			return result
		}
		result.Result = output
		result.Count = int32(strings.Count(output, "\n  "))
		if output == "" || isNoResultsMessage(output) {
			result.Count = 0
		}

	default:
		result.Error = fmt.Sprintf("unknown sub-query type: %s", q.Type)
	}

	return result
}

// ExecuteCommand runs a foreground command with the given parameters.
func (p *LocalClientOperationsProxy) ExecuteCommand(ctx context.Context, _, command, cwd string, timeout int32) (string, error) {
	return p.ExecuteCommandFull(ctx, "", map[string]string{
		"command": command,
		"cwd":     cwd,
		"timeout": fmt.Sprintf("%d", timeout),
	})
}

// ExecuteCommandFull routes command execution based on arguments:
// bg_action present -> background management;
// background=true -> spawn background process;
// otherwise -> foreground execution in persistent shell.
func (p *LocalClientOperationsProxy) ExecuteCommandFull(ctx context.Context, sessionID string, arguments map[string]string) (string, error) {
	bgAction := arguments["bg_action"]
	if bgAction != "" {
		return p.handleBgAction(bgAction, arguments["bg_id"])
	}

	if arguments["background"] == "true" {
		return p.handleBackground(arguments["command"], arguments["cwd"])
	}

	return p.handleForeground(ctx, sessionID, arguments)
}

// AskUserQuestionnaire in headless mode auto-selects the first option for each question.
func (p *LocalClientOperationsProxy) AskUserQuestionnaire(ctx context.Context, _, questionsJSON string) (string, error) {
	type question struct {
		Text    string   `json:"text"`
		Options []string `json:"options"`
	}

	var questions []question
	if err := json.Unmarshal([]byte(questionsJSON), &questions); err != nil {
		return "", fmt.Errorf("parse questions: %w", err)
	}

	type answer struct {
		Question string `json:"question"`
		Answer   string `json:"answer"`
	}

	answers := make([]answer, 0, len(questions))
	for _, q := range questions {
		selected := ""
		if len(q.Options) > 0 {
			selected = q.Options[0]
		}
		answers = append(answers, answer{
			Question: q.Text,
			Answer:   selected,
		})
		slog.InfoContext(ctx, "ask_user auto-answered (headless)", "question", q.Text, "answer", selected)
	}

	result, err := json.Marshal(answers)
	if err != nil {
		return "", fmt.Errorf("marshal answers: %w", err)
	}
	return string(result), nil
}

func (p *LocalClientOperationsProxy) LspRequest(ctx context.Context, _, symbolName, operation string) (string, error) {
	if p.lspService == nil {
		return "LSP not available.", nil
	}

	// Find symbol position via chunk store
	var filePath string
	var line, character int

	if p.chunkStore != nil {
		chunks, err := p.chunkStore.GetByName(ctx, symbolName)
		if err == nil && len(chunks) > 0 {
			chunk := chunks[0]
			filePath = chunk.FilePath
			line = chunk.StartLine - 1 // LSP uses 0-based lines
			character = resolveSymbolCharacter(chunk.Content, symbolName)
		}
	}

	if filePath == "" {
		return fmt.Sprintf("Symbol %q not found in index. Index the project first.", symbolName), nil
	}

	// Call LSP operation
	locations, err := p.callLspOperation(ctx, operation, filePath, line, character)
	if err != nil {
		return fmt.Sprintf("[ERROR] LSP request failed: %v", err), nil
	}

	// Retry once if empty (server might need warmup)
	if len(locations) == 0 && p.lspService.HasActiveClients() {
		time.Sleep(2 * time.Second)
		locations, err = p.callLspOperation(ctx, operation, filePath, line, character)
		if err != nil {
			return fmt.Sprintf("[ERROR] LSP retry failed: %v", err), nil
		}
	}

	if len(locations) == 0 {
		return fmt.Sprintf("No %s found for %q", operation, symbolName), nil
	}

	return formatLspLocations(operation, symbolName, filePath, line, locations, p.projectRoot), nil
}

// callLspOperation dispatches to the appropriate LSP method.
func (p *LocalClientOperationsProxy) callLspOperation(ctx context.Context, operation, filePath string, line, character int) ([]lsp.Location, error) {
	switch operation {
	case "definition":
		return p.lspService.Definition(ctx, filePath, line, character)
	case "references":
		return p.lspService.References(ctx, filePath, line, character)
	case "implementation":
		return p.lspService.Implementation(ctx, filePath, line, character)
	default:
		return nil, fmt.Errorf("unknown operation: %s", operation)
	}
}

// resolveSymbolCharacter finds the column of symbolName within the first line of content.
func resolveSymbolCharacter(content, symbolName string) int {
	firstLine, _, _ := strings.Cut(content, "\n")
	idx := strings.Index(firstLine, symbolName)
	if idx < 0 {
		return 0
	}
	return idx
}

// formatLspLocations formats LSP locations for display.
func formatLspLocations(operation, symbolName, sourceFile string, sourceLine int, locations []lsp.Location, projectRoot string) string {
	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("%s of %q (from %s:%d):\n",
		capitalizeFirst(operation), symbolName, relativeTo(sourceFile, projectRoot), sourceLine+1))
	for _, loc := range locations {
		relPath := uriToRelativePath(loc.URI, projectRoot)
		sb.WriteString(fmt.Sprintf("  %s:%d\n", relPath, loc.Range.Start.Line+1))
	}
	return sb.String()
}

// capitalizeFirst capitalizes the first letter of a string.
func capitalizeFirst(s string) string {
	if s == "" {
		return s
	}
	return strings.ToUpper(s[:1]) + s[1:]
}

// relativeTo returns a forward-slash path relative to base.
func relativeTo(path, base string) string {
	rel, err := filepath.Rel(base, path)
	if err != nil {
		return path
	}
	return filepath.ToSlash(rel)
}

// uriToRelativePath converts a file:// URI to a path relative to projectRoot.
func uriToRelativePath(uri, projectRoot string) string {
	path := strings.TrimPrefix(uri, "file:///")
	path = strings.TrimPrefix(path, "file://")

	// On Windows, URI may be file:///C:/...
	absPath, err := filepath.Abs(path)
	if err != nil {
		return path
	}
	rel, err := filepath.Rel(projectRoot, absPath)
	if err != nil {
		return absPath
	}
	return filepath.ToSlash(rel)
}

// Dispose cleans up all shell sessions, background processes, and LSP clients.
func (p *LocalClientOperationsProxy) Dispose() {
	if p.shellManager != nil {
		p.shellManager.DisposeAll()
	}
	if p.lspService != nil {
		p.lspService.Dispose()
	}
}

// handleForeground executes a command in a persistent shell session.
func (p *LocalClientOperationsProxy) handleForeground(ctx context.Context, sessionID string, arguments map[string]string) (string, error) {
	command := arguments["command"]
	if command == "" {
		return "[ERROR] command is required", nil
	}

	agentID := sessionID
	session := p.shellManager.GetAvailableSession(p.projectRoot, agentID)
	if session == nil {
		return "All shell sessions are busy (3/3). Wait for a command to finish or use background=true.", nil
	}

	timeout := parseTimeout(arguments["timeout"])

	// Resolve CWD: if provided, make it relative to projectRoot
	cwd := arguments["cwd"]
	if cwd != "" {
		resolved := p.resolveCwd(cwd)
		// cd into the directory before running the command
		command = fmt.Sprintf("cd %s && %s", shellQuote(resolved), command)
	}

	slog.InfoContext(ctx, "executing foreground command",
		"command", arguments["command"],
		"cwd", cwd,
		"timeout", timeout)

	result, err := session.Execute(ctx, command, time.Duration(timeout)*time.Second)
	if err != nil {
		return "", fmt.Errorf("execute command: %w", err)
	}

	return formatResult(result, timeout), nil
}

// handleBackground spawns a command as a background process.
func (p *LocalClientOperationsProxy) handleBackground(command, cwd string) (string, error) {
	if command == "" {
		return "[ERROR] command is required for background execution", nil
	}

	resolvedCwd := p.resolveCwd(cwd)
	proc, err := p.shellManager.BackgroundManager().Spawn(command, resolvedCwd)
	if err != nil {
		return "", fmt.Errorf("spawn background process: %w", err)
	}

	return fmt.Sprintf(
		"Started %s (PID: %d)\n\nUse execute_command with bg_action to manage:\n"+
			"- Read: {\"bg_action\":\"read\",\"bg_id\":\"%s\"}\n"+
			"- Kill: {\"bg_action\":\"kill\",\"bg_id\":\"%s\"}\n"+
			"- List: {\"bg_action\":\"list\"}",
		proc.ID, proc.PID, proc.ID, proc.ID,
	), nil
}

// handleBgAction manages background processes (list, read, kill).
func (p *LocalClientOperationsProxy) handleBgAction(action, bgID string) (string, error) {
	bgm := p.shellManager.BackgroundManager()

	switch action {
	case "list":
		return p.formatBgList(bgm.List()), nil

	case "read":
		if bgID == "" {
			return "[ERROR] bg_id is required for read action", nil
		}
		output, err := bgm.ReadOutput(bgID)
		if err != nil {
			return fmt.Sprintf("[ERROR] %v", err), nil
		}
		procs := bgm.List()
		statusLine := ""
		for _, proc := range procs {
			if proc.ID == bgID {
				statusLine = fmt.Sprintf("\n[Status: %s", proc.Status)
				if proc.Status == "exited" {
					statusLine += fmt.Sprintf(", exit code: %d", proc.ExitCode)
				}
				statusLine += "]"
				break
			}
		}
		if output == "" {
			return "(no output)" + statusLine, nil
		}
		return output + statusLine, nil

	case "kill":
		if bgID == "" {
			return "[ERROR] bg_id is required for kill action", nil
		}
		if err := bgm.Kill(bgID); err != nil {
			return fmt.Sprintf("[ERROR] Failed to kill %s: %v", bgID, err), nil
		}
		return fmt.Sprintf("Process %s killed.", bgID), nil

	default:
		return fmt.Sprintf("[ERROR] Unknown bg_action: %s. Use 'list', 'read', or 'kill'.", action), nil
	}
}

// formatBgList formats the background process list for display.
func (p *LocalClientOperationsProxy) formatBgList(procs []*shell.BackgroundProcess) string {
	if len(procs) == 0 {
		return "No background processes running."
	}

	var sb strings.Builder
	sb.WriteString("Background processes:\n")
	for _, proc := range procs {
		sb.WriteString(fmt.Sprintf("  %s [%s] PID=%d cmd=%q\n",
			proc.ID, proc.Status, proc.PID, proc.Command))
	}
	return sb.String()
}

// resolveCwd resolves a working directory relative to projectRoot.
func (p *LocalClientOperationsProxy) resolveCwd(cwd string) string {
	if cwd == "" {
		return p.projectRoot
	}
	if filepath.IsAbs(cwd) {
		return cwd
	}
	return filepath.Join(p.projectRoot, cwd)
}

// parseTimeout parses a timeout string, defaulting to 30s, max 120s.
func parseTimeout(s string) int {
	if s == "" {
		return 30
	}
	t, err := strconv.Atoi(s)
	if err != nil || t <= 0 {
		return 30
	}
	if t > 120 {
		return 120
	}
	return t
}

// formatResult converts a ShellResult to a display string.
func formatResult(result *shell.ShellResult, timeout int) string {
	if !result.Completed {
		output := result.Stdout
		if output == "" {
			output = "(no output captured)"
		}
		return fmt.Sprintf(
			"%s\n[Command timed out after %ds — interrupted]\n"+
				"[Use background=true for servers, watchers, and long-running processes]",
			output, timeout,
		)
	}

	output := result.Stdout
	if output == "" && result.ExitCode == 0 {
		return "[Command completed successfully with no output]"
	}

	if result.ExitCode != 0 {
		return fmt.Sprintf("%s\n[Exit code: %d]", output, result.ExitCode)
	}

	return output
}

// shellQuote wraps a path in single quotes for shell safety.
func shellQuote(s string) string {
	return "'" + strings.ReplaceAll(s, "'", "'\\''") + "'"
}
