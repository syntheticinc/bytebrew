package mcp

import (
	"context"
	"encoding/json"
	"log/slog"
	"time"

	infrastructure_mcp "github.com/syntheticinc/bytebrew/engine/internal/infrastructure/mcp"
	"github.com/syntheticinc/bytebrew/engine/internal/infrastructure/persistence/models"
)

// ConnectAll iterates the persisted MCP server records and dials each one,
// registering successful clients with the supplied registry. Failures are
// logged and skipped so a single broken endpoint does not abort the boot sequence.
// policy is consulted before opening stdio transports; Cloud deployments pass a
// RestrictedTransportPolicy that blocks stdio (host code execution is forbidden
// in multi-tenant builds).
func ConnectAll(
	ctx context.Context,
	servers []models.MCPServerModel,
	registry *infrastructure_mcp.ClientRegistry,
	policy TransportPolicy,
) {
	for _, srv := range servers {
		var forwardHeaders []string
		if srv.ForwardHeaders != nil && *srv.ForwardHeaders != "" {
			if err := json.Unmarshal([]byte(*srv.ForwardHeaders), &forwardHeaders); err != nil {
				slog.WarnContext(ctx, "mcp connector: failed to parse forward_headers", "name", srv.Name, "error", err)
				continue
			}
		}

		var transport infrastructure_mcp.Transport
		switch srv.Type {
		case "stdio":
			if err := policy.IsAllowed("stdio"); err != nil {
				slog.WarnContext(ctx, "MCP stdio transport blocked by policy", "name", srv.Name, "reason", err.Error())
				continue
			}
			var args []string
			if srv.Args != nil && *srv.Args != "" {
				if err := json.Unmarshal([]byte(*srv.Args), &args); err != nil {
					slog.WarnContext(ctx, "mcp connector: failed to parse args", "name", srv.Name, "error", err)
					continue
				}
			}
			transport = infrastructure_mcp.NewStdioTransport(srv.Command, args, nil, forwardHeaders)
		case "http":
			transport = infrastructure_mcp.NewHTTPTransport(srv.URL, forwardHeaders)
		case "sse":
			transport = infrastructure_mcp.NewSSETransport(srv.URL, forwardHeaders)
		case "streamable-http":
			transport = infrastructure_mcp.NewStreamableHTTPTransport(srv.URL, forwardHeaders)
		default:
			slog.WarnContext(ctx, "unknown MCP server type, skipping", "name", srv.Name, "type", srv.Type)
			continue
		}

		client := infrastructure_mcp.NewClient(srv.Name, transport)
		connectCtx, connectCancel := context.WithTimeout(ctx, 10*time.Second)
		if err := client.Connect(connectCtx); err != nil {
			slog.WarnContext(ctx, "MCP server unavailable, skipping", "name", srv.Name, "error", err)
			connectCancel()
			continue
		}
		connectCancel()

		tools := client.ListTools()
		slog.InfoContext(ctx, "MCP server connected", "name", srv.Name, "tools", len(tools))
		registry.Register(srv.Name, client)
	}
}
