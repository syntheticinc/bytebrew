package mcp

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/syntheticinc/bytebrew/engine/internal/domain"
)

const testCatalogYAML = `
catalog_version: "1.0"
servers:
  - name: "tavily-web-search"
    display: "Tavily Web Search"
    description: "AI-optimized web search"
    category: "search"
    verified: true
    packages:
      - type: "stdio"
        command: "npx"
        args: ["-y", "@mcptools/mcp-tavily"]
        env_vars:
          - name: "TAVILY_API_KEY"
            description: "Get key at tavily.com"
            required: true
            secret: true
    provided_tools:
      - name: "tavily_search"
        description: "Search the web"

  - name: "brave-search"
    display: "Brave Search"
    description: "Privacy-focused web search"
    category: "search"
    verified: true
    packages:
      - type: "stdio"
        command: "npx"
        args: ["-y", "@anthropic/mcp-brave-search"]

  - name: "github"
    display: "GitHub"
    description: "Create issues, PRs, search code"
    category: "dev-tools"
    verified: true
    packages:
      - type: "stdio"
        command: "npx"
        args: ["-y", "@modelcontextprotocol/server-github"]

  - name: "slack"
    display: "Slack"
    description: "Send messages, read channels"
    category: "communication"
    verified: true
    packages:
      - type: "stdio"
        command: "npx"
        args: ["-y", "@anthropic/mcp-slack"]
`

func TestCatalogService_List(t *testing.T) {
	svc, err := NewCatalogServiceFromData([]byte(testCatalogYAML))
	require.NoError(t, err)

	entries := svc.List()
	assert.Len(t, entries, 4)
	assert.Equal(t, "1.0", svc.Version())
}

func TestCatalogService_ListByCategory(t *testing.T) {
	svc, err := NewCatalogServiceFromData([]byte(testCatalogYAML))
	require.NoError(t, err)

	search := svc.ListByCategory(domain.MCPCategorySearch)
	assert.Len(t, search, 2)

	devTools := svc.ListByCategory(domain.MCPCategoryDevTools)
	assert.Len(t, devTools, 1)
	assert.Equal(t, "github", devTools[0].Name)

	payments := svc.ListByCategory(domain.MCPCategoryPayments)
	assert.Len(t, payments, 0)
}

func TestCatalogService_Search(t *testing.T) {
	svc, err := NewCatalogServiceFromData([]byte(testCatalogYAML))
	require.NoError(t, err)

	results := svc.Search("search")
	assert.Len(t, results, 3) // tavily + brave + github (all have "search" in name/desc)

	results = svc.Search("github")
	assert.Len(t, results, 1)
	assert.Equal(t, "github", results[0].Name)

	results = svc.Search("nonexistent")
	assert.Len(t, results, 0)
}

func TestCatalogService_GetByName(t *testing.T) {
	svc, err := NewCatalogServiceFromData([]byte(testCatalogYAML))
	require.NoError(t, err)

	entry, ok := svc.GetByName("tavily-web-search")
	require.True(t, ok)
	assert.Equal(t, "Tavily Web Search", entry.Display)
	assert.True(t, entry.Verified)
	assert.Equal(t, domain.MCPCategorySearch, entry.Category)
	require.Len(t, entry.Packages, 1)
	assert.Equal(t, "stdio", entry.Packages[0].Type)
	require.Len(t, entry.ProvidedTools, 1)
	assert.Equal(t, "tavily_search", entry.ProvidedTools[0].Name)

	_, ok = svc.GetByName("nonexistent")
	assert.False(t, ok)
}

func TestCatalogService_EnvVars(t *testing.T) {
	svc, err := NewCatalogServiceFromData([]byte(testCatalogYAML))
	require.NoError(t, err)

	entry, ok := svc.GetByName("tavily-web-search")
	require.True(t, ok)
	require.Len(t, entry.Packages[0].EnvVars, 1)

	env := entry.Packages[0].EnvVars[0]
	assert.Equal(t, "TAVILY_API_KEY", env.Name)
	assert.True(t, env.Required)
	assert.True(t, env.Secret)
}

func TestCatalogService_InvalidYAML(t *testing.T) {
	_, err := NewCatalogServiceFromData([]byte("invalid: [yaml: broken"))
	require.Error(t, err)
}
