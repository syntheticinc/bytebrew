package app

import (
	"context"
	"fmt"
	"log/slog"

	"gorm.io/gorm"

	"github.com/syntheticinc/bytebrew/engine/internal/infrastructure/persistence/config_repo"
	"github.com/syntheticinc/bytebrew/engine/internal/infrastructure/persistence/models"
)

const builderAssistantName = "builder-assistant"

const builderAssistantPrompt = `You are the ByteBrew Builder Assistant — an AI agent embedded in the Admin Dashboard that helps users configure and manage their ByteBrew Engine instance.

You have access to MCP tools from the "admin-api" server that let you manage:
- **Agents** — list, create, update, delete agents
- **Models** — list, create, update, delete LLM model configurations
- **Triggers** — list, create, update, delete triggers (cron schedules, webhooks)
- **MCP Servers** — list configured MCP servers
- **Tools** — list available builtin tools with security zones
- **Config** — export/import full Engine configuration as YAML

## Guidelines

1. **Be helpful and proactive.** When a user asks to configure something, use the appropriate tool immediately. Don't just describe what to do — do it.

2. **Confirm before destructive actions.** Always ask the user for confirmation before deleting agents, models, or triggers.

3. **Explain what you did.** After creating or modifying something, briefly describe the change.

4. **Suggest improvements.** If you notice an agent has no model assigned, suggest one. If a trigger references a non-existent agent, flag it.

5. **Know the entities:**
   - An **Agent** needs at minimum: name and system_prompt. Optionally: model, tools, mcp_servers, lifecycle (persistent/ephemeral), tool_execution (sequential/parallel), can_spawn, confirm_before.
   - A **Model** needs: name, type (openai_compatible, anthropic, etc.), model_name. Optionally: base_url, api_key.
   - A **Trigger** needs: type (cron/webhook), title, agent_id. For cron: schedule. For webhook: webhook_path.

6. **Use list tools first** to understand current state before making changes.

7. **Config export/import** uses YAML format with upsert semantics — imports create or update, never delete.`

// seedBuilderAssistant ensures the builder-assistant agent exists in the database.
// If it already exists, it does NOT overwrite (user may have customized it).
// If no models exist, the agent is created without a model.
func seedBuilderAssistant(ctx context.Context, db *gorm.DB) {
	if db == nil {
		return
	}

	// Ensure admin-api MCP server record exists in DB (required for junction table).
	var adminMCP models.MCPServerModel
	if err := db.Where("name = ?", "admin-api").First(&adminMCP).Error; err != nil {
		adminMCP = models.MCPServerModel{
			Name:        "admin-api",
			Type:        "in_process",
			IsWellKnown: true,
		}
		if err := db.Create(&adminMCP).Error; err != nil {
			slog.ErrorContext(ctx, "failed to seed admin-api MCP server record", "error", err)
		} else {
			slog.InfoContext(ctx, "seeded admin-api MCP server record")
		}
	}

	agentRepo := config_repo.NewGORMAgentRepository(db)

	// Check if builder-assistant already exists.
	_, err := agentRepo.GetByName(ctx, builderAssistantName)
	if err == nil {
		slog.InfoContext(ctx, "builder-assistant agent already exists, skipping seed")
		return
	}

	// Determine model to assign.
	llmRepo := config_repo.NewGORMLLMProviderRepository(db)
	allModels, listErr := llmRepo.List(ctx)

	var modelName string
	if listErr == nil && len(allModels) > 0 {
		modelName = allModels[0].Name
		slog.InfoContext(ctx, "builder-assistant: assigning first available model", "model", modelName)
	} else {
		slog.InfoContext(ctx, "builder-assistant: no models available, creating without model")
	}

	// Build the agent record.
	record := &config_repo.AgentRecord{
		Name:          builderAssistantName,
		SystemPrompt:  builderAssistantPrompt,
		ModelName:     modelName,
		Lifecycle:     "persistent",
		ToolExecution: "sequential",
		MCPServers:    []string{"admin-api"},
	}

	if err := agentRepo.Create(ctx, record); err != nil {
		slog.ErrorContext(ctx, "failed to seed builder-assistant agent", "error", err)
		return
	}

	msg := fmt.Sprintf("seeded builder-assistant agent (model=%s)", modelName)
	if modelName == "" {
		msg = "seeded builder-assistant agent (no model — configure one in Models page)"
	}
	slog.InfoContext(ctx, msg)
}
