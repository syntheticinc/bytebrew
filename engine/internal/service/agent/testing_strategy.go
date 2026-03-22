package agent

import (
	"fmt"

	"gopkg.in/yaml.v3"
)

// TestingStrategy represents project-level testing configuration
type TestingStrategy struct {
	Testing TestingConfig `yaml:"testing"`
}

// TestingConfig holds testing commands and metadata
type TestingConfig struct {
	Build       *CommandEntry `yaml:"build"`
	Unit        *CommandEntry `yaml:"unit"`
	Integration *CommandEntry `yaml:"integration"`
	Lint        *CommandEntry `yaml:"lint"`
	Notes       string        `yaml:"notes"`
}

// CommandEntry represents a single testing command with optional metadata
type CommandEntry struct {
	Command     string `yaml:"command"`
	Description string `yaml:"description,omitempty"`
	Pattern     string `yaml:"pattern,omitempty"`
	Framework   string `yaml:"framework,omitempty"`
}

// ParseTestingStrategy parses YAML content into TestingStrategy
func ParseTestingStrategy(yamlContent string) (*TestingStrategy, error) {
	var strategy TestingStrategy
	if err := yaml.Unmarshal([]byte(yamlContent), &strategy); err != nil {
		return nil, fmt.Errorf("parse testing strategy: %w", err)
	}
	return &strategy, nil
}
