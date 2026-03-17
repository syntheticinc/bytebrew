//go:build prompt

package prompt_regression

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"runtime"

	"github.com/syntheticinc/bytebrew/bytebrew-srv/internal/infrastructure/agents"
)

// Fixture represents a test fixture wrapping a context snapshot
type Fixture struct {
	Name        string                 `json:"name"`
	Description string                 `json:"description"`
	Snapshot    agents.ContextSnapshot `json:"snapshot"`
}

// LoadFixture loads a fixture from fixtures/{name}.json
func LoadFixture(name string) (*Fixture, error) {
	// Get current file directory using runtime.Caller
	_, filename, _, ok := runtime.Caller(0)
	if !ok {
		return nil, fmt.Errorf("failed to get current file path")
	}
	currentDir := filepath.Dir(filename)

	// Construct path to fixture file
	fixturePath := filepath.Join(currentDir, "fixtures", name+".json")

	// Read fixture file
	data, err := os.ReadFile(fixturePath)
	if err != nil {
		return nil, fmt.Errorf("read fixture file: %w", err)
	}

	// Parse JSON
	var fixture Fixture
	if err := json.Unmarshal(data, &fixture); err != nil {
		return nil, fmt.Errorf("parse fixture JSON: %w", err)
	}

	return &fixture, nil
}
