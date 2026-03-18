package tools

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGetToolMetadata_KnownTool(t *testing.T) {
	tests := []struct {
		name         string
		toolName     string
		wantZone     SecurityZone
		wantWarning  bool
	}{
		{"safe tool", "ask_user", ZoneSafe, false},
		{"safe tool web_search", "web_search", ZoneSafe, false},
		{"caution tool", "web_fetch", ZoneCaution, true},
		{"caution tool glob", "glob", ZoneCaution, true},
		{"dangerous tool read_file", "read_file", ZoneDangerous, true},
		{"dangerous tool execute_command", "execute_command", ZoneDangerous, true},
		{"dangerous tool write_file", "write_file", ZoneDangerous, true},
		{"dangerous tool edit_file", "edit_file", ZoneDangerous, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			meta := GetToolMetadata(tt.toolName)
			assert.Equal(t, tt.toolName, meta.Name)
			assert.Equal(t, tt.wantZone, meta.SecurityZone)
			assert.NotEmpty(t, meta.Description)
			if tt.wantWarning {
				assert.NotEmpty(t, meta.RiskWarning)
			}
		})
	}
}

func TestGetToolMetadata_UnknownTool(t *testing.T) {
	meta := GetToolMetadata("nonexistent_tool")
	assert.Equal(t, "nonexistent_tool", meta.Name)
	assert.Equal(t, ZoneCaution, meta.SecurityZone)
	assert.Equal(t, "Custom tool", meta.Description)
}

func TestGetAllToolMetadata(t *testing.T) {
	all := GetAllToolMetadata()
	require.NotEmpty(t, all)

	// Verify all zones are represented
	zones := map[SecurityZone]int{}
	for _, m := range all {
		zones[m.SecurityZone]++
		assert.NotEmpty(t, m.Name, "every tool must have a name")
		assert.NotEmpty(t, m.Description, "every tool must have a description")
	}
	assert.Greater(t, zones[ZoneSafe], 0, "should have safe tools")
	assert.Greater(t, zones[ZoneCaution], 0, "should have caution tools")
	assert.Greater(t, zones[ZoneDangerous], 0, "should have dangerous tools")
}

func TestExecuteCommand_HasStrictestWarning(t *testing.T) {
	meta := GetToolMetadata("execute_command")
	assert.Contains(t, meta.RiskWarning, "CRITICAL")
	assert.Contains(t, meta.RiskWarning, "ARBITRARY")
}
