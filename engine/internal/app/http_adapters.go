package app

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"sync/atomic"
	"time"

	"gorm.io/gorm"

	deliveryhttp "github.com/syntheticinc/bytebrew/engine/internal/delivery/http"
	"github.com/syntheticinc/bytebrew/engine/internal/infrastructure/agentregistry"
	"github.com/syntheticinc/bytebrew/engine/internal/infrastructure/audit"
	"github.com/syntheticinc/bytebrew/engine/internal/infrastructure/mcp"
	"github.com/syntheticinc/bytebrew/engine/internal/infrastructure/persistence/configrepo"
	"github.com/syntheticinc/bytebrew/engine/internal/infrastructure/persistence/models"
)

// agentCounterHTTPAdapter bridges AgentRegistry to the http.AgentCounter interface.
type agentCounterHTTPAdapter struct {
	registry *agentregistry.AgentRegistry
}

func (a *agentCounterHTTPAdapter) Count() int {
	return a.registry.Count()
}

// auditHTTPAdapter bridges audit.Logger to the http.AuditLogger interface.
type auditHTTPAdapter struct {
	logger *audit.Logger
}

func (a *auditHTTPAdapter) Log(ctx context.Context, entry deliveryhttp.AuditEntry) error {
	return a.logger.Log(ctx, audit.Entry{
		Timestamp: entry.Timestamp,
		ActorType: entry.ActorType,
		ActorID:   entry.ActorID,
		Action:    entry.Action,
		Resource:  entry.Resource,
		Details:   entry.Details,
		SessionID: entry.SessionID,
	})
}

// agentListerHTTPAdapter bridges AgentRegistry to the http.AgentLister interface.
type agentListerHTTPAdapter struct {
	registry *agentregistry.AgentRegistry
}

func (a *agentListerHTTPAdapter) ListAgents(_ context.Context) ([]deliveryhttp.AgentInfo, error) {
	agents := a.registry.GetAll()
	result := make([]deliveryhttp.AgentInfo, 0, len(agents))
	for _, agent := range agents {
		result = append(result, deliveryhttp.AgentInfo{
			Name:         agent.Record.Name,
			ToolsCount:   len(agent.Record.BuiltinTools) + len(agent.Record.CustomTools),
		})
	}
	return result, nil
}

func (a *agentListerHTTPAdapter) GetAgent(_ context.Context, name string) (*deliveryhttp.AgentDetail, error) {
	agent, err := a.registry.Get(name)
	if err != nil {
		return nil, nil
	}
	rec := agent.Record
	tools := make([]string, 0, len(rec.BuiltinTools)+len(rec.CustomTools))
	tools = append(tools, rec.BuiltinTools...)
	for _, ct := range rec.CustomTools {
		tools = append(tools, ct.Name)
	}
	return &deliveryhttp.AgentDetail{
		AgentInfo: deliveryhttp.AgentInfo{
			Name:         rec.Name,
			ToolsCount:   len(tools),
			IsSystem:     rec.IsSystem,
		},
		ModelID:         rec.ModelID,
		SystemPrompt:    rec.SystemPrompt,
		Tools:           tools,
		CanSpawn:        rec.CanSpawn,
		Lifecycle:       rec.Lifecycle,
		ToolExecution:   rec.ToolExecution,
		MaxSteps:        rec.MaxSteps,
		MaxContextSize:  rec.MaxContextSize,
		MaxTurnDuration: rec.MaxTurnDuration,
		Temperature:     rec.Temperature,
		TopP:            rec.TopP,
		MaxTokens:       rec.MaxTokens,
		StopSequences:   rec.StopSequences,
		ConfirmBefore:   rec.ConfirmBefore,
		MCPServers:      rec.MCPServers,
	}, nil
}

// tokenRepoHTTPAdapter bridges GORMAPITokenRepository to the http.TokenRepository interface.
type tokenRepoHTTPAdapter struct {
	repo *configrepo.GORMAPITokenRepository
}

func (a *tokenRepoHTTPAdapter) Create(ctx context.Context, name, tokenHash string, scopesMask int) (string, error) {
	return a.repo.Create(ctx, name, tokenHash, scopesMask)
}

func (a *tokenRepoHTTPAdapter) List(ctx context.Context) ([]deliveryhttp.TokenInfo, error) {
	tokens, err := a.repo.List(ctx)
	if err != nil {
		return nil, err
	}
	result := make([]deliveryhttp.TokenInfo, 0, len(tokens))
	for _, t := range tokens {
		result = append(result, deliveryhttp.TokenInfo{
			ID:         t.ID,
			Name:       t.Name,
			ScopesMask: t.ScopesMask,
			CreatedAt:  t.CreatedAt,
			LastUsedAt: t.LastUsedAt,
		})
	}
	return result, nil
}

func (a *tokenRepoHTTPAdapter) Delete(ctx context.Context, id string) error {
	return a.repo.Delete(ctx, id)
}

func (a *tokenRepoHTTPAdapter) VerifyToken(ctx context.Context, tokenHash string) (string, int, error) {
	return a.repo.VerifyToken(ctx, tokenHash)
}

// userResolverHTTPAdapter bridges GORMUserRepository to the http.UserResolver interface.
type userResolverHTTPAdapter struct {
	repo *configrepo.GORMUserRepository
}

func (a *userResolverHTTPAdapter) GetOrCreate(ctx context.Context, tenantID, externalID string) (string, error) {
	user, err := a.repo.GetOrCreate(ctx, tenantID, externalID)
	if err != nil {
		return "", err
	}
	return user.ID, nil
}

// configReloaderHTTPAdapter bridges AgentRegistry and MCP reconnection to the http.ConfigReloader interface.
type configReloaderHTTPAdapter struct {
	registry            *agentregistry.AgentRegistry
	mcpRegistry         *mcp.ClientRegistry
	db                  *gorm.DB
	forwardHeadersStore *atomic.Value // shared with ChatHandler for dynamic forward header updates
}

func (a *configReloaderHTTPAdapter) Reload(ctx context.Context) error {
	if err := a.registry.Reload(ctx); err != nil {
		return err
	}

	a.reconnectMCPServers(ctx)
	return nil
}

func (a *configReloaderHTTPAdapter) reconnectMCPServers(ctx context.Context) {
	if a.mcpRegistry == nil || a.db == nil {
		return
	}

	mcpServerRepo := configrepo.NewGORMMCPServerRepository(a.db)
	mcpServers, err := mcpServerRepo.List(ctx)
	if err != nil {
		slog.ErrorContext(ctx, "failed to load MCP servers for reload", "error", err)
		return
	}

	a.mcpRegistry.CloseAll()
	connectMCPServers(ctx, mcpServers, a.mcpRegistry)

	// Update forward headers so ChatHandler picks up changes immediately
	if a.forwardHeadersStore != nil {
		a.forwardHeadersStore.Store(collectForwardHeaders(mcpServers))
		slog.InfoContext(ctx, "forward headers updated after config reload")
	}

	slog.InfoContext(ctx, "MCP servers reconnected after config reload", "count", len(mcpServers))
}

func (a *configReloaderHTTPAdapter) AgentsCount() int {
	return a.registry.Count()
}

// collectForwardHeaders returns the deduplicated union of forward_headers
// configured across all MCP servers.
func collectForwardHeaders(mcpServers []models.MCPServerModel) []string {
	seen := make(map[string]bool)
	var headers []string
	for _, srv := range mcpServers {
		if srv.ForwardHeaders == "" {
			continue
		}
		var fh []string
		if err := json.Unmarshal([]byte(srv.ForwardHeaders), &fh); err != nil {
			continue
		}
		for _, h := range fh {
			if !seen[h] {
				seen[h] = true
				headers = append(headers, h)
			}
		}
	}
	return headers
}

// configImportExportHTTPAdapter and its YAML types live in config_import_export_http_adapter.go.

// auditServiceHTTPAdapter bridges GORMAuditRepository to the http.AuditService interface.
type auditServiceHTTPAdapter struct {
	repo *configrepo.GORMAuditRepository
}

func (a *auditServiceHTTPAdapter) ListAuditLogs(ctx context.Context, actorType, action, resource string, from, to *time.Time, page, perPage int) ([]deliveryhttp.AuditResponse, int64, error) {
	filters := configrepo.AuditFilters{
		ActorType: actorType,
		Action:    action,
		Resource:  resource,
		From:      from,
		To:        to,
	}

	logs, total, err := a.repo.List(ctx, filters, page, perPage)
	if err != nil {
		return nil, 0, err
	}

	result := make([]deliveryhttp.AuditResponse, 0, len(logs))
	for _, l := range logs {
		actorID := ""
		if l.ActorUserID != nil {
			actorID = *l.ActorUserID
		}
		result = append(result, deliveryhttp.AuditResponse{
			ID:        l.ID,
			Timestamp: l.OccurredAt.Format(time.RFC3339),
			ActorType: l.ActorType,
			ActorID:   actorID,
			Action:    l.Action,
			Resource:  l.Resource,
			Details:   l.Details,
		})
	}
	return result, total, nil
}

// toolCallLogHTTPAdapter bridges ToolCallEventRepository to the http.ToolCallEventQuerier interface.
type toolCallLogHTTPAdapter struct {
	repo *configrepo.ToolCallEventRepository
}

func (a *toolCallLogHTTPAdapter) QueryToolCalls(ctx context.Context, filters deliveryhttp.ToolCallFilters, page, perPage int) ([]deliveryhttp.ToolCallEntry, int64, error) {
	repoFilters := configrepo.ToolCallFilters{
		SessionID: filters.SessionID,
		AgentName: filters.AgentName,
		ToolName:  filters.ToolName,
		Status:    filters.Status,
		UserID:    filters.UserID,
		From:      filters.From,
		To:        filters.To,
	}

	entries, total, err := a.repo.QueryToolCalls(ctx, repoFilters, page, perPage)
	if err != nil {
		return nil, 0, err
	}

	result := make([]deliveryhttp.ToolCallEntry, 0, len(entries))
	for _, e := range entries {
		result = append(result, deliveryhttp.ToolCallEntry{
			ID:         e.ID,
			SessionID:  e.SessionID,
			AgentName:  e.AgentName,
			ToolName:   e.ToolName,
			Input:      e.Input,
			Output:     e.Output,
			Status:     e.Status,
			DurationMs: e.DurationMs,
			UserID:     e.UserID,
			CreatedAt:  e.CreatedAt,
		})
	}
	return result, total, nil
}

// agentSchemaIDResolver resolves the primary schema UUID for an agent.
// BUG-007: memory/knowledge tools need schema_id (UUID) to scope data.
type agentSchemaIDResolver struct {
	db *gorm.DB
}

func (r *agentSchemaIDResolver) ResolveSchemaID(ctx context.Context, agentName string) (string, error) {
	// Q.5: agent_relations uses agent UUIDs. Resolve name → id first,
	// then find the schema where this agent participates.
	var agentID string
	if err := r.db.WithContext(ctx).Raw(
		"SELECT id FROM agents WHERE name = ? LIMIT 1", agentName).Scan(&agentID).Error; err != nil || agentID == "" {
		return "", fmt.Errorf("no schema for agent %q", agentName)
	}
	var schemaID string
	if err := r.db.WithContext(ctx).Raw(
		`SELECT schema_id FROM agent_relations
			WHERE source_agent_id = ? OR target_agent_id = ?
			LIMIT 1`, agentID, agentID).Scan(&schemaID).Error; err != nil || schemaID == "" {
		return "", fmt.Errorf("no schema for agent %q", agentName)
	}
	return schemaID, nil
}

// taskServiceHTTPAdapter and its helpers live in task_http_adapter.go.

// knowledgeStatsHTTPAdapter bridges GORMKnowledgeRepository to the http.KnowledgeStats interface.
// Resolves agent name → linked KB IDs, then aggregates stats across KBs.
type knowledgeStatsHTTPAdapter struct {
	repo   *configrepo.GORMKnowledgeRepository
	kbRepo *configrepo.GORMKnowledgeBaseRepository
}

func (a *knowledgeStatsHTTPAdapter) GetStats(ctx context.Context, agentName string) (int, int, *time.Time, error) {
	if a.kbRepo == nil {
		return 0, 0, nil, nil
	}
	kbIDs, err := a.kbRepo.ListKBsByAgentName(ctx, agentName)
	if err != nil || len(kbIDs) == 0 {
		return 0, 0, nil, nil
	}
	return a.repo.GetStatsByKBs(ctx, kbIDs)
}

// knowledgeReindexerHTTPAdapter bridges knowledge.Indexer to the http.KnowledgeReindexer interface.
type knowledgeReindexerHTTPAdapter struct {
	indexer  knowledgeIndexer
	registry *agentregistry.AgentRegistry
}

// knowledgeIndexer is the consumer-side interface for indexing.
type knowledgeIndexer interface {
	IndexFolder(ctx context.Context, kbID string, folderPath string) error
}

func (a *knowledgeReindexerHTTPAdapter) Reindex(ctx context.Context, agentName string) error {
	agent, err := a.registry.Get(agentName)
	if err != nil {
		return fmt.Errorf("agent not found: %s", agentName)
	}
	_ = agent
	return fmt.Errorf("agent %s: legacy knowledge_path reindex is no longer supported; use capability Knowledge + knowledge base instead", agentName)
}

// chatServiceHTTPAdapter and chatTriggerCheckerAdapter live in chat_http_adapter.go.
