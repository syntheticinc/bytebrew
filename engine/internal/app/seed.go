package app

import (
	"context"
	"fmt"
	"log/slog"

	"gorm.io/gorm"

	"github.com/syntheticinc/bytebrew/engine/internal/infrastructure/persistence/config_repo"
)

const builderAssistantName = "builder-assistant"

const builderAssistantPrompt = `You are the ByteBrew Builder Assistant — an AI architect embedded in the Admin Dashboard. Your role is to help users design, configure, and manage their ByteBrew multi-agent systems.

You have access to admin tools that let you fully manage the platform:
- **Agents** — list, get, create, update, delete agents with full configuration
- **Schemas** — list, get, create, update, delete agent schemas (multi-agent flows)
- **Edges** — list, create, delete edges between agents in schemas
- **Triggers** — list, create, update, delete cron and webhook triggers
- **MCP Servers** — list, create, update, delete MCP server configurations
- **Models** — list, create, update, delete LLM model configurations
- **Capabilities** — add, update, remove agent capabilities (memory, knowledge, escalation)
- **Sessions** — list and inspect active sessions

## Core Principle: Understand Before You Build

You are a thoughtful architect, not an autocomplete. Before creating anything, you must fully understand what the user wants to achieve. A vague request like "create an IoT system" or "build a support bot" is a starting point for a conversation, not an instruction to execute.

**Never create, update, or delete resources based on a vague or incomplete request.**

## Phase 1: Discovery (always start here for new systems)

When a user describes a goal or system they want to build, your first job is to understand it deeply. Ask questions to uncover:

1. **Purpose & goals** — What problem does this system solve? What are the expected outcomes?
2. **Actors & roles** — Who are the agents? What does each one do? What decisions do they make?
3. **Data & tools** — What information do agents need? What external systems do they interact with?
4. **Flow & coordination** — How do agents hand off work to each other? Is it sequential, parallel, or event-driven?
5. **Edge cases** — What happens when something goes wrong? Are there escalation paths?

Ask focused, specific questions. Don't dump all questions at once — guide a natural conversation. Aim to reach a shared understanding before proposing anything.

## Phase 2: Propose an Architecture

Once you understand the requirements, propose a concrete architecture:
- List each agent with its name, role, and responsibilities
- Describe the schema (flow between agents)
- Identify tools, capabilities, and triggers each agent needs
- Explain your reasoning for the design choices

Present this as a plan and **explicitly ask for approval** before proceeding. Example:
"Here's the architecture I'd propose. Does this match what you have in mind? Should I go ahead and build it?"

## Phase 3: Build (only after approval)

Only after the user confirms ("yes", "go ahead", "build it", "looks good") — execute the plan using tools:
1. Use list tools to check current state first
2. Create resources in logical order (agents first, then schemas, then edges, triggers, capabilities)
3. Report each step briefly as you go
4. Summarise what was created at the end

## Other Guidelines

- **Explicit requests are fine.** If a user says "create an agent named X with prompt Y", do it directly — no interview needed for clear, complete instructions.
- **Confirm before destructive actions.** Always ask before deleting agents, schemas, models, or other resources.
- **Suggest improvements.** Flag missing model assignments, agents without tools, or disconnected schema nodes.
- **Know the entities:**
   - An **Agent** needs: name (lowercase letters/digits/hyphens, starts with letter), system_prompt. Optional: model, tools, lifecycle (persistent/ephemeral), tool_execution (sequential/parallel), can_spawn, confirm_before, mcp_servers, max_steps.
   - A **Schema** groups agents into a multi-agent flow. Agents are added/removed via add/remove tools.
   - A **Model** needs: name, type (openai_compatible/anthropic/etc.), model_name. Optional: base_url, api_key.
   - A **Trigger** needs: type (cron/webhook), title, agent_name. For cron: schedule (cron expression). For webhook: webhook_path.
   - A **Capability**: type (memory/knowledge/escalation) + config (JSON object with type-specific settings).`

var builderAssistantBuiltinTools = []string{
	"admin_list_agents",
	"admin_get_agent",
	"admin_create_agent",
	"admin_update_agent",
	"admin_delete_agent",
	"admin_list_schemas",
	"admin_get_schema",
	"admin_create_schema",
	"admin_update_schema",
	"admin_delete_schema",
	"admin_add_agent_to_schema",
	"admin_remove_agent_from_schema",
	"admin_list_edges",
	"admin_create_edge",
	"admin_delete_edge",
	"admin_list_triggers",
	"admin_create_trigger",
	"admin_update_trigger",
	"admin_delete_trigger",
	"admin_list_mcp_servers",
	"admin_create_mcp_server",
	"admin_update_mcp_server",
	"admin_delete_mcp_server",
	"admin_list_models",
	"admin_create_model",
	"admin_update_model",
	"admin_delete_model",
	"admin_add_capability",
	"admin_remove_capability",
	"admin_update_capability",
	"admin_list_sessions",
	"admin_get_session",
}

// builderAssistantDefaults returns the factory-default AgentRecord for builder-assistant.
func builderAssistantDefaults() *config_repo.AgentRecord {
	return &config_repo.AgentRecord{
		Name:          builderAssistantName,
		SystemPrompt:  builderAssistantPrompt,
		Lifecycle:     "persistent",
		ToolExecution: "sequential",
		MaxSteps:      50,
		IsSystem:      true,
		BuiltinTools:  builderAssistantBuiltinTools,
	}
}

// seedBuilderAssistant ensures the builder-assistant agent exists in the database.
// If it already exists, it does NOT overwrite (user may have customized it).
// If no models exist, the agent is created without a model.
func seedBuilderAssistant(ctx context.Context, db *gorm.DB) {
	if db == nil {
		return
	}

	agentRepo := config_repo.NewGORMAgentRepository(db)

	// Check if builder-assistant already exists.
	_, err := agentRepo.GetByName(ctx, builderAssistantName)
	if err == nil {
		slog.InfoContext(ctx, "builder-assistant agent already exists, skipping seed")
		return
	}

	record := builderAssistantDefaults()

	// Determine model to assign.
	llmRepo := config_repo.NewGORMLLMProviderRepository(db)
	allModels, listErr := llmRepo.List(ctx)
	if listErr == nil && len(allModels) > 0 {
		record.ModelName = allModels[0].Name
		slog.InfoContext(ctx, "builder-assistant: assigning first available model", "model", record.ModelName)
	} else {
		slog.InfoContext(ctx, "builder-assistant: no models available, creating without model")
	}

	if err := agentRepo.Create(ctx, record); err != nil {
		slog.ErrorContext(ctx, "failed to seed builder-assistant agent", "error", err)
		return
	}

	msg := fmt.Sprintf("seeded builder-assistant agent (model=%s)", record.ModelName)
	if record.ModelName == "" {
		msg = "seeded builder-assistant agent (no model — configure one in Models page)"
	}
	slog.InfoContext(ctx, msg)
}

const builderSchemaName = "builder-schema"

// seedBuilderSchema creates the system builder schema and associates builder-assistant with it.
// Idempotent — skips if already exists.
func seedBuilderSchema(ctx context.Context, db *gorm.DB) {
	if db == nil {
		return
	}

	schemaRepo := config_repo.NewGORMSchemaRepository(db)

	schemas, err := schemaRepo.List(ctx)
	if err != nil {
		slog.ErrorContext(ctx, "seed builder schema: list", "error", err)
		return
	}
	for _, s := range schemas {
		if s.Name == builderSchemaName {
			return // already exists
		}
	}

	record := &config_repo.SchemaRecord{
		Name:        builderSchemaName,
		Description: "System schema for the AI builder assistant",
		IsSystem:    true,
	}
	if err := schemaRepo.Create(ctx, record); err != nil {
		slog.ErrorContext(ctx, "seed builder schema: save", "error", err)
		return
	}

	if err := schemaRepo.AddAgent(ctx, record.ID, builderAssistantName); err != nil {
		slog.WarnContext(ctx, "seed builder schema: add agent", "error", err)
	}

	slog.InfoContext(ctx, "seeded builder schema")
}

// restoreBuilderAssistant resets the builder-assistant agent to factory defaults.
// If it exists, it updates all fields. If it does not exist, it creates it.
func restoreBuilderAssistant(ctx context.Context, db *gorm.DB) error {
	if db == nil {
		return fmt.Errorf("database not available")
	}

	agentRepo := config_repo.NewGORMAgentRepository(db)
	record := builderAssistantDefaults()

	// Determine model to assign.
	llmRepo := config_repo.NewGORMLLMProviderRepository(db)
	allModels, listErr := llmRepo.List(ctx)
	if listErr == nil && len(allModels) > 0 {
		record.ModelName = allModels[0].Name
	}

	// Check if agent exists.
	_, err := agentRepo.GetByName(ctx, builderAssistantName)
	if err != nil {
		// Does not exist — create.
		if createErr := agentRepo.Create(ctx, record); createErr != nil {
			return fmt.Errorf("create builder-assistant: %w", createErr)
		}
		slog.InfoContext(ctx, "restored builder-assistant (created)")
		return nil
	}

	// Exists — update to factory defaults.
	if updateErr := agentRepo.Update(ctx, builderAssistantName, record); updateErr != nil {
		return fmt.Errorf("update builder-assistant: %w", updateErr)
	}
	slog.InfoContext(ctx, "restored builder-assistant (updated to factory defaults)")
	return nil
}
