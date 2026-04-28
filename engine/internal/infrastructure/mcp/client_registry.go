package mcp

import (
	"fmt"
	"sync"

	"github.com/cloudwego/eino/components/tool"
)

// ClientRegistry manages connected MCP clients and provides tools by server name.
// Implements tools.MCPClientProvider.
type ClientRegistry struct {
	mu      sync.RWMutex
	clients map[string]*Client
}

// NewClientRegistry creates a new ClientRegistry.
func NewClientRegistry() *ClientRegistry {
	return &ClientRegistry{clients: make(map[string]*Client)}
}

// Register adds a connected client to the registry.
func (r *ClientRegistry) Register(name string, client *Client) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.clients[name] = client
}

// GetMCPTools returns Eino-compatible tools for the named MCP server.
// Returns nil, nil if the server is not registered or not connected.
func (r *ClientRegistry) GetMCPTools(name string) ([]tool.InvokableTool, error) {
	r.mu.RLock()
	client, ok := r.clients[name]
	r.mu.RUnlock()

	if !ok {
		return nil, fmt.Errorf("mcp server %q not registered", name)
	}
	if !client.IsConnected() {
		return nil, nil
	}

	mcpTools := client.ListTools()
	result := make([]tool.InvokableTool, 0, len(mcpTools))
	for _, mt := range mcpTools {
		result = append(result, AdaptMCPTool(client, mt))
	}
	return result, nil
}

// CloseAll closes all registered clients.
func (r *ClientRegistry) CloseAll() {
	r.mu.Lock()
	defer r.mu.Unlock()

	for name, client := range r.clients {
		_ = client.Close()
		delete(r.clients, name)
	}
}

// Names returns all registered server names.
func (r *ClientRegistry) Names() []string {
	r.mu.RLock()
	defer r.mu.RUnlock()

	names := make([]string, 0, len(r.clients))
	for name := range r.clients {
		names = append(names, name)
	}
	return names
}
