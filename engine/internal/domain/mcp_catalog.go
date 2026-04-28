package domain

// MCPCatalogRecord is the system-wide catalog entry for an MCP server.
//
// V2 Commit Group C (§5.5): the catalog moved from runtime YAML reads into a
// dedicated `mcp_catalog` table. Rows are seeded from `mcp-catalog.yaml` at
// engine startup (see `seedMCPCatalog`). There is no FK from `mcp_servers`
// back to this table — installing from the catalog copies the selected
// package fields into a new tenant-scoped `mcp_servers` row, after which the
// instance is independent of the catalog.
//
// This is a pure domain entity (no GORM tags). Packages and ProvidedTools are
// stored as JSON/jsonb by the GORM model and round-trip here as structured
// slices.
type MCPCatalogRecord struct {
	ID            string
	Name          string // stable catalog key, unique (e.g. "tavily-web-search")
	Display       string
	Description   string
	Category      MCPCatalogCategory
	Verified      bool
	Packages      []MCPCatalogPackage
	ProvidedTools []MCPCatalogTool
}
