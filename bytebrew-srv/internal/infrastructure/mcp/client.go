package mcp

import (
	"context"
	"fmt"
	"sync"
)

// Client connects to an MCP server and provides tools.
type Client struct {
	name      string
	transport Transport
	tools     []MCPTool
	mu        sync.RWMutex
	connected bool
	nextID    int64
}

// NewClient creates a new MCP client with the given name and transport.
func NewClient(name string, transport Transport) *Client {
	return &Client{name: name, transport: transport}
}

// Connect initializes the connection and fetches available tools.
func (c *Client) Connect(ctx context.Context) error {
	if err := c.transport.Start(ctx); err != nil {
		return fmt.Errorf("start transport: %w", err)
	}

	initReq := &Request{
		JSONRPC: "2.0",
		ID:      c.nextRequestID(),
		Method:  "initialize",
		Params: map[string]interface{}{
			"protocolVersion": "2024-11-05",
			"capabilities":   map[string]interface{}{},
			"clientInfo": map[string]interface{}{
				"name":    "bytebrew-engine",
				"version": "1.0.0",
			},
		},
	}
	if _, err := c.transport.Send(ctx, initReq); err != nil {
		return fmt.Errorf("initialize: %w", err)
	}

	notif := &Request{
		JSONRPC: "2.0",
		Method:  "notifications/initialized",
	}
	c.transport.Notify(ctx, notif)

	toolsReq := &Request{
		JSONRPC: "2.0",
		ID:      c.nextRequestID(),
		Method:  "tools/list",
	}
	resp, err := c.transport.Send(ctx, toolsReq)
	if err != nil {
		return fmt.Errorf("tools/list: %w", err)
	}

	tools, err := parseToolsFromResponse(resp)
	if err != nil {
		return fmt.Errorf("parse tools: %w", err)
	}

	c.mu.Lock()
	c.tools = tools
	c.connected = true
	c.mu.Unlock()

	return nil
}

// Name returns the client name.
func (c *Client) Name() string { return c.name }

// ListTools returns the tools available from the MCP server.
func (c *Client) ListTools() []MCPTool {
	c.mu.RLock()
	defer c.mu.RUnlock()
	result := make([]MCPTool, len(c.tools))
	copy(result, c.tools)
	return result
}

// IsConnected returns whether the client is connected.
func (c *Client) IsConnected() bool {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.connected
}

// CallTool invokes a tool on the MCP server.
func (c *Client) CallTool(ctx context.Context, name string, args map[string]interface{}) (string, error) {
	req := &Request{
		JSONRPC: "2.0",
		ID:      c.nextRequestID(),
		Method:  "tools/call",
		Params: map[string]interface{}{
			"name":      name,
			"arguments": args,
		},
	}
	resp, err := c.transport.Send(ctx, req)
	if err != nil {
		return "", fmt.Errorf("call tool %q: %w", name, err)
	}
	return extractToolResult(resp)
}

// Close closes the transport connection.
func (c *Client) Close() error {
	c.mu.Lock()
	c.connected = false
	c.mu.Unlock()
	return c.transport.Close()
}

func (c *Client) nextRequestID() int64 {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.nextID++
	return c.nextID
}
