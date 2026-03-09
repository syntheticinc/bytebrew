package lsp

import (
	"context"
	"fmt"
	"log/slog"
	"os/exec"
	"sync"
	"time"
)

// Service manages per-project, per-language LSP clients.
type Service struct {
	clients     map[string]*Client // key: serverID (e.g., "go", "typescript")
	projectRoot string
	installer   *Installer
	mu          sync.Mutex
}

// NewService creates a new LSP service for the given project root.
func NewService(projectRoot string) *Service {
	return &Service{
		clients:     make(map[string]*Client),
		projectRoot: projectRoot,
		installer:   NewInstaller(),
	}
}

// Definition finds the definition of a symbol at the given position.
func (s *Service) Definition(ctx context.Context, filePath string, line, character int) ([]Location, error) {
	client, err := s.getOrCreateClient(ctx, filePath)
	if err != nil {
		return nil, err
	}

	if err := client.DidOpen(ctx, filePath); err != nil {
		slog.WarnContext(ctx, "LSP didOpen failed", "file", filePath, "error", err)
	}

	uri := pathToURI(filePath)
	pos := Position{Line: line, Character: character}
	return client.Definition(ctx, uri, pos)
}

// References finds all references to a symbol at the given position.
func (s *Service) References(ctx context.Context, filePath string, line, character int) ([]Location, error) {
	client, err := s.getOrCreateClient(ctx, filePath)
	if err != nil {
		return nil, err
	}

	if err := client.DidOpen(ctx, filePath); err != nil {
		slog.WarnContext(ctx, "LSP didOpen failed", "file", filePath, "error", err)
	}

	uri := pathToURI(filePath)
	pos := Position{Line: line, Character: character}
	return client.References(ctx, uri, pos)
}

// Implementation finds implementations of an interface at the given position.
func (s *Service) Implementation(ctx context.Context, filePath string, line, character int) ([]Location, error) {
	client, err := s.getOrCreateClient(ctx, filePath)
	if err != nil {
		return nil, err
	}

	if err := client.DidOpen(ctx, filePath); err != nil {
		slog.WarnContext(ctx, "LSP didOpen failed", "file", filePath, "error", err)
	}

	uri := pathToURI(filePath)
	pos := Position{Line: line, Character: character}
	return client.Implementation(ctx, uri, pos)
}

// HasActiveClients returns true if any LSP clients are running.
func (s *Service) HasActiveClients() bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	return len(s.clients) > 0
}

// Dispose shuts down all LSP clients.
func (s *Service) Dispose() {
	s.mu.Lock()
	clients := make(map[string]*Client, len(s.clients))
	for k, v := range s.clients {
		clients[k] = v
	}
	s.clients = make(map[string]*Client)
	s.mu.Unlock()

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	for id, client := range clients {
		slog.Info("shutting down LSP client", "server", id)
		if err := client.Shutdown(ctx); err != nil {
			slog.Warn("LSP shutdown error", "server", id, "error", err)
		}
	}
}

// getOrCreateClient returns an existing client or creates a new one for the given file.
func (s *Service) getOrCreateClient(ctx context.Context, filePath string) (*Client, error) {
	config := ConfigForFile(filePath)
	if config == nil {
		return nil, fmt.Errorf("no LSP server available for file type: %s", filePath)
	}

	s.mu.Lock()
	if client, ok := s.clients[config.ID]; ok {
		s.mu.Unlock()
		return client, nil
	}
	s.mu.Unlock()

	// Find workspace root
	root, err := config.FindRoot(filePath, s.projectRoot)
	if err != nil {
		root = s.projectRoot
		slog.WarnContext(ctx, "LSP: workspace root not found, using project root",
			"server", config.ID, "error", err)
	}

	// Spawn server (try auto-install if binary not found)
	name, args, err := config.SpawnCommand(root)
	if err != nil && config.Install != nil {
		slog.InfoContext(ctx, "LSP binary not found, attempting auto-install",
			"server", config.ID)
		if installErr := s.installer.Install(ctx, config.ID, *config.Install); installErr != nil {
			return nil, fmt.Errorf("spawn %s: %w (auto-install also failed: %v)", config.ID, err, installErr)
		}
		// Retry after install
		name, args, err = config.SpawnCommand(root)
	}
	if err != nil {
		return nil, fmt.Errorf("spawn %s: %w", config.ID, err)
	}

	cmd := exec.Command(name, args...)
	cmd.Dir = root

	client, err := NewClient(config.ID, root, cmd)
	if err != nil {
		return nil, fmt.Errorf("create LSP client %s: %w", config.ID, err)
	}

	// Initialize
	if err := client.Initialize(ctx, root); err != nil {
		_ = client.Close()
		return nil, fmt.Errorf("initialize LSP %s: %w", config.ID, err)
	}

	// Wait briefly for readiness (some servers index quickly)
	client.WaitForReady(5 * time.Second)

	s.mu.Lock()
	// Double-check: another goroutine may have created the client
	if existing, ok := s.clients[config.ID]; ok {
		s.mu.Unlock()
		_ = client.Shutdown(ctx)
		return existing, nil
	}
	s.clients[config.ID] = client
	s.mu.Unlock()

	slog.InfoContext(ctx, "LSP client created", "server", config.ID, "root", root)
	return client, nil
}
