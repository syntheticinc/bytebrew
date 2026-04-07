package mcp

import (
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"strings"

	"github.com/syntheticinc/bytebrew/engine/internal/domain"
	"gopkg.in/yaml.v3"
)

// CatalogService loads and manages the MCP server catalog.
type CatalogService struct {
	catalog *domain.MCPCatalog
}

// NewCatalogService creates a catalog service by loading the catalog from YAML.
// It searches for mcp-catalog.yaml alongside the binary or in the working directory.
func NewCatalogService() (*CatalogService, error) {
	catalog, err := loadCatalog()
	if err != nil {
		slog.Warn("[MCPCatalog] failed to load catalog, using empty", "error", err)
		catalog = &domain.MCPCatalog{
			CatalogVersion: "1.0",
			Servers:        []domain.MCPCatalogEntry{},
		}
	}

	slog.Info("[MCPCatalog] loaded", "version", catalog.CatalogVersion, "servers", len(catalog.Servers))
	return &CatalogService{catalog: catalog}, nil
}

// NewCatalogServiceFromData creates a catalog service from raw YAML data (for testing).
func NewCatalogServiceFromData(data []byte) (*CatalogService, error) {
	catalog, err := parseCatalog(data)
	if err != nil {
		return nil, fmt.Errorf("parse catalog: %w", err)
	}
	return &CatalogService{catalog: catalog}, nil
}

// List returns all catalog entries.
func (s *CatalogService) List() []domain.MCPCatalogEntry {
	return s.catalog.Servers
}

// ListByCategory returns catalog entries filtered by category.
func (s *CatalogService) ListByCategory(category domain.MCPCatalogCategory) []domain.MCPCatalogEntry {
	var result []domain.MCPCatalogEntry
	for _, entry := range s.catalog.Servers {
		if entry.Category == category {
			result = append(result, entry)
		}
	}
	return result
}

// Search returns catalog entries matching the query (name or description).
func (s *CatalogService) Search(query string) []domain.MCPCatalogEntry {
	q := strings.ToLower(query)
	var result []domain.MCPCatalogEntry
	for _, entry := range s.catalog.Servers {
		if strings.Contains(strings.ToLower(entry.Name), q) ||
			strings.Contains(strings.ToLower(entry.Display), q) ||
			strings.Contains(strings.ToLower(entry.Description), q) {
			result = append(result, entry)
		}
	}
	return result
}

// GetByName returns a specific catalog entry by name.
func (s *CatalogService) GetByName(name string) (*domain.MCPCatalogEntry, bool) {
	for _, entry := range s.catalog.Servers {
		if entry.Name == name {
			return &entry, true
		}
	}
	return nil, false
}

// Version returns the catalog version string.
func (s *CatalogService) Version() string {
	return s.catalog.CatalogVersion
}

func loadCatalog() (*domain.MCPCatalog, error) {
	// Search paths: alongside binary, then working directory
	paths := []string{}

	// Path relative to binary
	if exe, err := os.Executable(); err == nil {
		paths = append(paths, filepath.Join(filepath.Dir(exe), "mcp-catalog.yaml"))
	}

	// Working directory
	if wd, err := os.Getwd(); err == nil {
		paths = append(paths, filepath.Join(wd, "mcp-catalog.yaml"))
	}

	for _, p := range paths {
		data, err := os.ReadFile(p)
		if err != nil {
			continue
		}
		slog.Info("[MCPCatalog] loading from", "path", p)
		return parseCatalog(data)
	}

	return nil, fmt.Errorf("mcp-catalog.yaml not found in: %v", paths)
}

func parseCatalog(data []byte) (*domain.MCPCatalog, error) {
	var catalog domain.MCPCatalog
	if err := yaml.Unmarshal(data, &catalog); err != nil {
		return nil, fmt.Errorf("parse YAML: %w", err)
	}
	return &catalog, nil
}
