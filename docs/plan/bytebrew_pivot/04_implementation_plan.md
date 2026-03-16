# ByteBrew Pivot — Implementation Plan

**Дата:** 15 марта 2026
**Цель:** Трансформация ByteBrew 1.x (coding agent) → universal autonomous agent engine
**Базовый документ:** `02_requirements.md`

---

## Принципы

- **Переиспользовать максимум** — 60%+ кода остаётся as-is
- **Абстрагировать, не переписывать** — hardcoded → configurable
- **Engine не знает про домены** — coding-специфика выносится в developer kit
- **Каждый этап — работающий продукт** — нет "сломано пока рефакторим"
- **Backward compatible** — текущий coding agent продолжает работать на каждом этапе
- **Тесты обновляются вместе с кодом** — каждый этап включает обновление затронутых тестов. Новые struct'ы/interfaces → новые unit tests. Рефакторинг → адаптация существующих тестов. Если тест ломается при рефакторинге — чинить сразу, не откладывать

---

## Этап 1: Config Engine + Agent Abstraction

**Цель:** агенты определяются в YAML, не захардкожены в коде.

### Текущее состояние

**`pkg/config/config.go`:**
```go
type Config struct {
    // ...
    Agent       AgentConfig        // ОДИН глобальный конфиг на всех агентов
    // ...
}

type AgentConfig struct {
    MaxSteps       int
    MaxContextSize int
    Prompts        *PromptsConfig   // system, supervisor, code, reviewer, researcher
}

type PromptsConfig struct {
    SystemPrompt     string         // Один system prompt
    SupervisorPrompt string         // Hardcoded роли
    CodeAgentPrompt  string
    ResearcherPrompt string
    ReviewerPrompt   string
    UrgencyWarning   string
}
```

**`internal/domain/flow.go`:** hardcoded flow types:
```go
const (
    FlowTypeSupervisor = "supervisor"
    FlowTypeCoder      = "coder"
    FlowTypeReviewer   = "reviewer"
    FlowTypeResearcher = "researcher"
)

type Flow struct {
    Type           FlowType
    ToolNames      []string     // Фиксированный набор tools per flow type
    MaxSteps       int
    Lifecycle      LifecyclePolicy
    Spawn          SpawnPolicy  // AllowedFlows, MaxConcurrent
}
```

**Промпты загружаются из:**
- `internal/embedded/prompts.yaml` (embedded в бинарник)
- `prompts.yaml` в dataDir (создаётся из embedded при первом запуске)
- Merge с `config.yaml` через Viper

### Целевое состояние

**Новый формат `bytebrew.yaml`:**
```go
type Config struct {
    Engine    EngineConfig         // host, port, data_dir, logging
    Models    ModelsConfig         // providers[]
    Agents    []AgentDefinition    // МАССИВ агентов
    Triggers  []TriggerDefinition  // cron, webhooks (global)
    Bridge    BridgeConfig         // Без изменений
    License   LicenseConfig        // Без изменений
}

type AgentDefinition struct {
    Name           string              // "supervisor", "code-agent", "sales"
    Model          string              // Ссылка на providers[].name
    SystemPrompt   string              // Per-agent prompt
    Flow           *FlowConfig         // steps, escalation
    Tools          ToolsConfig         // builtin[], mcp_servers{}, custom[]
    Kit            string              // "developer" или "" (опционально)
    Knowledge      string              // Путь к папке (опционально)
    Rules          []string            // Ограничения (inject в prompt)
    ConfirmBefore  []string            // Tools requiring confirmation
    CanSpawn       []string            // Whitelist agent names
    Lifecycle      string              // "persistent" | "spawn" (default: "persistent")
    ToolExecution  string              // "sequential" | "parallel" (default: "sequential")
    MaxSteps       int                 // Override global (default: 50)
    MaxContextSize int                 // Override global (default: 16000)
    Triggers       []TriggerDefinition // Per-agent triggers
}

type FlowConfig struct {
    Steps      []string             // Инструкции (inject в prompt)
    Escalation *EscalationConfig    // triggers[], action, webhook
}

type ToolsConfig struct {
    Builtin    []string                    // ["ask_user", "web_search", "manage_tasks"]
    MCPServers map[string]MCPServerConfig  // name → {type, url/command}
    Custom     []CustomToolConfig          // Declarative HTTP tools
}

type TriggerDefinition struct {
    Type     string     // "cron" | "webhook"
    Schedule string     // Cron expression (for cron)
    Path     string     // HTTP path (for webhook)
    Job      JobConfig  // title, description, agent
}
```

### Маппинг: старое → новое

| Старое | Новое | Действие |
|--------|-------|----------|
| `Config.Agent.Prompts.SystemPrompt` | `AgentDefinition.SystemPrompt` | Per-agent |
| `Config.Agent.Prompts.SupervisorPrompt` | `AgentDefinition.SystemPrompt` (для supervisor agent) | Объединить с system_prompt |
| `Config.Agent.Prompts.CodeAgentPrompt` | `AgentDefinition.SystemPrompt` (для code-agent) | Объединить |
| `Config.Agent.Prompts.ReviewerPrompt` | `AgentDefinition.SystemPrompt` (для reviewer agent) | Объединить |
| `Config.Agent.Prompts.ResearcherPrompt` | `AgentDefinition.SystemPrompt` (для researcher agent) | Объединить |
| `Config.Agent.Prompts.UrgencyWarning` | Остаётся в engine (глобальный) | Без изменений |
| `Config.Agent.MaxSteps` | `AgentDefinition.MaxSteps` (per-agent) + global default | Per-agent override |
| `Config.Agent.MaxContextSize` | `AgentDefinition.MaxContextSize` (per-agent) + global default | Per-agent override |
| `domain.Flow.Type` | `AgentDefinition.Name` | Убрать enum, agent name = identity |
| `domain.Flow.ToolNames` | `AgentDefinition.Tools.Builtin` | Per-agent из YAML |
| `domain.Flow.Spawn.AllowedFlows` | `AgentDefinition.CanSpawn` | Per-agent |
| `domain.Flow.Lifecycle` | `AgentDefinition.Lifecycle` | Per-agent |
| `embedded/prompts.yaml` | `examples/developer.yaml` | Промпты → пример конфига |
| `embedded/flows.yaml` | Убрать | Flows определяются agents[] |

### Файлы: изменения

| Файл | Изменения |
|------|-----------|
| **`pkg/config/config.go`** | Добавить `AgentDefinition`, `ToolsConfig`, `FlowConfig`, `TriggerDefinition`. Сохранить старый `Config.Agent` как fallback для backward compat. Парсинг: если `agents:` есть → новый формат, если нет → legacy (один agent из Config.Agent) |
| **`internal/domain/flow.go`** | Оставить `Flow` struct, но убрать hardcoded constants. `FlowType` → `string` (agent name). Добавить `Flow.FromAgentDefinition(def AgentDefinition) *Flow` — конвертер |
| **`internal/embedded/prompts.yaml`** | Оставить как fallback. Добавить комментарий "legacy, use bytebrew.yaml agents[]" |
| **`internal/embedded/flows.yaml`** | Оставить как fallback |
| **`cmd/server/main.go`** | После config.Load(): если `cfg.Agents` не пуст → создать AgentRegistry из них. Если пуст → legacy path (создать agents из Config.Agent + flows.yaml) |

### Файлы: новые

| Файл | Содержимое |
|------|-----------|
| **`internal/infrastructure/agent_registry.go`** | `type AgentRegistry struct { agents map[string]*domain.Flow }`. Методы: `Get(name) *Flow`, `List() []string`, `FromConfig([]config.AgentDefinition)`. Конвертирует AgentDefinition → domain.Flow. Валидация: уникальные имена, can_spawn ссылается на существующих агентов, lifecycle валиден |

### Wiring

**Текущий:**
```
main.go → NewInfraComponents(cfg) → FlowManager (из flows.yaml) → Engine
  → FlowManager.GetFlow(flowType) → Flow (hardcoded)
  → Engine.Execute(ExecutionConfig{Flow: flow, ...})
```

**Новый:**
```
main.go → config.Load() → cfg.Agents[]
  → AgentRegistry.FromConfig(cfg.Agents) → registry
  → NewInfraComponents(cfg, registry) → Engine
  → registry.Get(agentName) → Flow (из YAML)
  → Engine.Execute(ExecutionConfig{Flow: flow, ...})
```

**`infrastructure/agent_service.go` NewInfraComponents:**
- Добавить параметр `registry *AgentRegistry`
- FlowManager → заменить на AgentRegistry (или FlowManager делегирует к AgentRegistry)
- AgentPool → использует registry для получения flow spawned агентов

### Backward compatibility

Если `bytebrew.yaml` не содержит `agents:` → fallback на текущее поведение:
1. Читаем Config.Agent + prompts.yaml + flows.yaml
2. Создаём agents из legacy конфига (supervisor, coder, reviewer, researcher)
3. AgentRegistry заполняется из legacy

Это позволяет не ломать существующих пользователей.

### System prompt composition

AgentDefinition содержит system_prompt, rules, flow.steps — всё inject'ится в один system prompt:

```go
func BuildSystemPrompt(def *AgentDefinition) string {
    var sb strings.Builder
    sb.WriteString(def.SystemPrompt)

    if len(def.Flow.Steps) > 0 {
        sb.WriteString("\n\n## Workflow\n")
        for i, step := range def.Flow.Steps {
            sb.WriteString(fmt.Sprintf("%d. %s\n", i+1, step))
        }
    }

    if len(def.Rules) > 0 {
        sb.WriteString("\n\n## Rules (MUST follow)\n")
        for _, rule := range def.Rules {
            sb.WriteString("- " + rule + "\n")
        }
    }

    if len(def.ConfirmBefore) > 0 {
        sb.WriteString("\n\n## Confirmation required\nAsk user before calling: ")
        sb.WriteString(strings.Join(def.ConfirmBefore, ", "))
    }

    return sb.String()
}
```

Этот composed prompt подаётся в `MessageModifier` вместо текущего `config.AgentConfig.Prompts.SystemPrompt`. MessageModifier остаётся без изменений — он просто inject'ит system prompt.

### Edge cases

- Agent name содержит спецсимволы → валидация: `^[a-z][a-z0-9-]*$`
- `can_spawn` ссылается на несуществующего агента → ошибка при старте
- Два агента с одинаковым именем → ошибка при старте
- Agent без `system_prompt` → ошибка при старте (обязательное поле)
- Agent с `lifecycle: "spawn"` в top-level (не spawned другим) → предупреждение (кто его создаст?)

### Валидация этапа

- [ ] `examples/developer.yaml` → coding agent работает через новый конфиг
- [ ] `bytebrew.yaml` с двумя агентами → оба загружены, видны через `registry.List()`
- [ ] Legacy config.yaml без `agents:` → текущее поведение сохранено
- [ ] Невалидный YAML → понятная ошибка при старте

---

## Этап 2: Dynamic Tool Registry

**Цель:** каждый агент видит только свои tools. Tools определяются в YAML.

### Текущее состояние

**`infrastructure/tools/resolver.go`:**
```go
func (r *DefaultToolResolver) Resolve(ctx context.Context, toolNames []string, deps ToolDependencies) ([]tool.InvokableTool, error) {
    // switch по 20+ hardcoded names
    // Все зависимости через ToolDependencies struct
}
```

**`infrastructure/tools/deps_provider.go`:**
```go
type ToolDependencies struct {
    SessionID, ProjectKey, ProjectRoot string
    Proxy          ClientOperationsProxy
    TaskManager    TaskManager
    SubtaskManager SubtaskManager
    AgentPool      AgentPoolForTool
    WebSearchTool, WebFetchTool  tool.InvokableTool
    ChunkStore     *indexing.ChunkStore
    Embedder       *indexing.EmbeddingsClient
}
```

Tools разрешаются по строковым именам, все зависимости передаются глобально. Агент получает **все** tools которые перечислены в его Flow.ToolNames.

### Целевое состояние

```go
type ToolRegistry struct {
    builtinFactories map[string]BuiltinToolFactory  // name → factory func
    mcpClients       map[string]*mcp.Client          // server name → MCP client
    kitRegistry      *KitRegistry                    // для получения kit.Tools(session)
}

type BuiltinToolFactory func(deps ToolDependencies) tool.InvokableTool

// ResolveContext — per-session контекст для resolve
type ResolveContext struct {
    AgentDef   *AgentDefinition
    Deps       ToolDependencies
    KitSession *domain.KitSession  // nil если нет kit
    ActiveKit  domain.Kit          // nil если нет kit
}

func (r *ToolRegistry) ResolveForAgent(rc ResolveContext) ([]tool.InvokableTool, error) {
    var tools []tool.InvokableTool

    // 1. Built-in tools (из agentDef.Tools.Builtin whitelist)
    for _, name := range rc.AgentDef.Tools.Builtin {
        if factory, ok := r.builtinFactories[name]; ok {
            tools = append(tools, factory(rc.Deps))
        }
    }

    // 2. MCP tools (из agentDef.Tools.MCPServers)
    for serverName := range rc.AgentDef.Tools.MCPServers {
        if client, ok := r.mcpClients[serverName]; ok {
            tools = append(tools, client.Tools()...)
        }
    }

    // 3. Custom declarative tools (из agentDef.Tools.Custom)
    for _, custom := range rc.AgentDef.Tools.Custom {
        tools = append(tools, NewDeclarativeTool(custom))
    }

    // 4. Kit tools (per-session, не cached)
    if rc.ActiveKit != nil && rc.KitSession != nil {
        kitTools := rc.ActiveKit.Tools(*rc.KitSession)
        // Wrap kit-enrichable tools (file_edit, file_write etc.) с KitEnrichmentWrapper
        for i, t := range tools {
            tools[i] = NewKitEnrichmentWrapper(t, rc.ActiveKit, *rc.KitSession)
        }
        tools = append(tools, kitTools...)
    }

    // 5. Spawn tools (из agentDef.CanSpawn)
    for _, spawnTarget := range rc.AgentDef.CanSpawn {
        tools = append(tools, NewSpawnTool(spawnTarget, rc.Deps.AgentPool, rc.Deps.SessionID))
    }

    // Wrap all with Safe + Cancellable
    return wrapAll(tools), nil
}
```

### Файлы: изменения

| Файл | Изменения |
|------|-----------|
| **`infrastructure/tools/resolver.go`** | Рефакторинг: switch → `builtinFactories` map. Каждый case из switch → factory function. `Resolve()` → deprecated (для legacy). Новый метод `ResolveForAgent()` |
| **`infrastructure/tools/deps_provider.go`** | `ToolDependencies` — добавить `AgentRegistry *AgentRegistry`. Убрать `ChunkStore`, `Embedder` (переедут в Kit). Остальное без изменений |
| **`infrastructure/turn_executor_factory.go`** | `CreateForSession()` → использовать `ToolRegistry.ResolveForAgent(agentDef, deps)` вместо `resolver.Resolve(flow.ToolNames, deps)` |
| **`service/engine/engine.go`** | `ExecutionConfig.Tools` заполняется из `ToolRegistry.ResolveForAgent()`, не из `ToolResolver.Resolve()` |

### Файлы: новые

| Файл | Содержимое |
|------|-----------|
| **`infrastructure/tools/tool_registry.go`** | `ToolRegistry` struct. Регистрация built-in factories при создании. `ResolveForAgent()` |
| **`infrastructure/tools/builtin_factories.go`** | Все built-in tool factories извлечённые из switch в resolver.go |
| **`infrastructure/tools/spawn_tool.go`** | Generic `SpawnTool(targetAgentName)`. Заменяет `SpawnCodeAgentTool`. Actions: spawn, wait, status, list, stop |
| **`infrastructure/tools/declarative_tool.go`** | `DeclarativeTool` — YAML endpoint → HTTP request → tool response |

### Регистрация built-in tools

Из текущего switch в resolver.go извлекаем factories:

```go
func registerBuiltinFactories(registry *ToolRegistry) {
    registry.RegisterBuiltin("read_file", func(d ToolDependencies) tool.InvokableTool {
        return NewReadFileTool(d.Proxy, d.SessionID)
    })
    registry.RegisterBuiltin("write_file", func(d ToolDependencies) tool.InvokableTool {
        return NewWriteFileTool(/* ... */)
    })
    registry.RegisterBuiltin("edit_file", func(d ToolDependencies) tool.InvokableTool {
        return NewEditFileTool(/* ... */)
    })
    // ... все 19 built-in tools
    registry.RegisterBuiltin("manage_tasks", func(d ToolDependencies) tool.InvokableTool {
        return NewManageTasksTool(d.TaskManager, d.Proxy)
    })
    // manage_subtasks объединяем в manage_tasks
    registry.RegisterBuiltin("ask_user", func(d ToolDependencies) tool.InvokableTool {
        return NewAskUserTool(/* ... */)
    })
}
```

### Security: whitelist-only

Agent видит **только** tools из своего конфига. Если `builtin: [ask_user]` — видит только ask_user. `shell_exec` не появится если не указан. Это critical security — sales agent не должен иметь доступ к shell.

### Валидация этапа

- [ ] Agent с `builtin: [ask_user, web_search]` → видит только 2 tools
- [ ] Agent с `can_spawn: ["reviewer"]` → видит tool `spawn_reviewer`
- [ ] Agent без `shell_exec` в конфиге → не может выполнять shell команды
- [ ] Legacy resolver всё ещё работает для backward compat

---

## Этап 3: Agent Spawn Refactor

**Цель:** generic spawn через `can_spawn` вместо hardcoded `spawn_code_agent`.

### Текущее состояние

**`tools/spawn_code_agent_tool.go`:**
- Hardcoded `AgentPoolForTool` interface с `Spawn(sessionID, projectKey, subtaskID, blocking)`
- Знает про subtaskID, projectKey, FlowType
- Actions: spawn, wait, status, list, stop, restart

**`service/agent/agent_pool.go`:**
- `RunningAgent` имеет `SubtaskID`, `ProjectKey`, `flowType domain.FlowType`
- `Spawn()` принимает subtaskID, привязывается к subtask lifecycle
- `SpawnWithDescription()` принимает flowType (coder/researcher/reviewer)
- Session-scoped contexts: агенты выживают при отмене supervisor turn

### Целевое состояние

**Новый `SpawnTool` (generic):**
```go
type SpawnTool struct {
    targetAgentName string              // Из can_spawn конфига
    pool            GenericAgentPool    // Новый interface
    sessionID       string
}

// LLM вызывает: spawn_code_agent(description: "...", task_id: "task-1")
// task_id — опциональный. Если указан, engine связывает spawn с task для auto-update.
type SpawnArgs struct {
    Description string `json:"description"`          // Обязательный
    TaskID      string `json:"task_id,omitempty"`    // Опциональный — связь с manage_tasks
}

type GenericAgentPool interface {
    SpawnAgent(ctx context.Context, params SpawnParams) (string, error)
    WaitForAllSessionAgents(ctx context.Context, sessionID string) (WaitResult, error)
    HasBlockingWait(sessionID string) bool
    NotifyUserMessage(sessionID, message string)
    GetStatusInfo(agentID string) (*AgentInfo, bool)
    GetAllAgentInfos() []AgentInfo
    StopAgent(agentID string) error
}

type SpawnParams struct {
    SessionID   string
    AgentName   string
    Description string
    TaskID      string  // Опциональный — если указан, engine auto-updates task при completion
    Blocking    bool
}
```

**Связь spawn ↔ task:** LLM сам передаёт task_id при spawn (через промпт: "при делегировании указывай task_id"). Engine при завершении spawn-агента проверяет: если TaskID != "" → обновить task status. Это сохраняет generic подход (engine не знает про tasks обязательно) + даёт auto-update когда связь есть.
```

**Рефакторинг AgentPool:**
```go
type AgentPool struct {
    // Сохраняется:
    agents           map[string]*RunningAgent
    mu               sync.RWMutex
    sessionContexts  map[string]context.Context   // Session-scoped contexts
    sessionCancels   map[string]context.CancelFunc
    interrupt        *InterruptManager
    eventBus         *orchestrator.SessionEventBus

    // Меняется:
    agentRegistry    *AgentRegistry      // ВМЕСТО flowProvider FlowProvider
    // modelSelector → берёт model из AgentDefinition
    // subtaskManager → опциональный (только если agent использует manage_tasks)

    // Сохраняется:
    engine           AgentEngine
    toolRegistry     *ToolRegistry       // ВМЕСТО toolResolver + toolDeps
    agentConfig      *config.AgentConfig // Global defaults
}

func (p *AgentPool) SpawnAgent(ctx, sessionID, agentName, description string, blocking bool) (string, error) {
    // 1. agentDef := p.agentRegistry.Get(agentName)
    // 2. Создать RunningAgent с session-scoped context
    // 3. Emit EventTypeAgentSpawned
    // 4. Goroutine → runAgentWithEngine(agentDef, description)
    // 5. Return agentID
}

type RunningAgent struct {
    ID           string
    AgentName    string          // ВМЕСТО flowType — имя из YAML
    SessionID    string
    Description  string          // Task description
    TaskID       string          // Опциональный — связь с manage_tasks для auto-update
    Status       string          // running | completed | failed | stopped
    Result       string
    Error        string
    Lifecycle    string          // "persistent" | "spawn"
    StartedAt    time.Time
    Cancel       context.CancelFunc
    completionCh chan struct{}
    blockingSpawn bool
}
```

### Файлы: изменения

| Файл | Изменения |
|------|-----------|
| **`service/agent/agent_pool.go`** | Убрать `subtaskManager`, `flowProvider`, `modelSelector` fields. Добавить `agentRegistry`, `toolRegistry`. `Spawn()` → legacy wrapper. Новый `SpawnAgent()`. RunningAgent: убрать SubtaskID/ProjectKey/flowType, добавить AgentName/Description/Lifecycle |
| **`service/agent/agent_pool_adapter.go`** | Адаптировать под новый `GenericAgentPool` interface. Упростить |
| **`domain/flow.go`** | `FlowType` → `string` (оставить type alias для backward compat). Убрать const block с hardcoded types. `Flow.FromAgentDefinition()` |
| **`domain/active_flow.go`** | Адаптировать под string-based flow type |

### Файлы: удаляемые

| Файл | Причина |
|------|---------|
| **`tools/spawn_code_agent_tool.go`** | Заменяется generic `spawn_tool.go` |

### Как spawn_tool знает какого агента создать

SpawnTool создаётся ToolRegistry при resolve для конкретного агента:

```go
// В ToolRegistry.ResolveForAgent():
for _, spawnTarget := range agentDef.CanSpawn {
    tools = append(tools, NewSpawnTool(spawnTarget, deps.AgentPool, deps.SessionID))
}
```

LLM видит tool `spawn_reviewer(description)` → вызывает → SpawnTool.Execute() → pool.SpawnAgent(sessionID, "reviewer", description, blocking).

### Валидация этапа

- [ ] `can_spawn: ["code-agent"]` → supervisor может спаунить code-agent
- [ ] `can_spawn: ["review", "e2e-test"]` → code-agent может спаунить оба
- [ ] Agent без `can_spawn` → нет spawn tools
- [ ] `lifecycle: "spawn"` → agent умирает после return
- [ ] `lifecycle: "persistent"` → agent живёт до scope end
- [ ] Blocking spawn + user interrupt → WaitResult.Interrupted = true (сохранено)
- [ ] Session-scoped contexts → агенты выживают при отмене supervisor turn (сохранено)

---

## Этап 4: MCP Client

**Цель:** агент может использовать внешние MCP серверы как tools.

### Файлы: новые

| Файл | Содержимое |
|------|-----------|
| **`internal/infrastructure/mcp/client.go`** | MCP Client: connect, listTools, callTool. Implements MCP protocol (JSON-RPC 2.0 over stdio/HTTP/SSE) |
| **`internal/infrastructure/mcp/stdio_transport.go`** | Запуск MCP сервера как subprocess (stdin/stdout). Lifecycle: start on first call, stop on shutdown |
| **`internal/infrastructure/mcp/http_transport.go`** | HTTP transport: POST to MCP server URL |
| **`internal/infrastructure/mcp/sse_transport.go`** | SSE transport: connect, receive events |
| **`internal/infrastructure/mcp/tool_adapter.go`** | MCP tool → Eino tool.InvokableTool adapter. Конвертирует MCP tool schema → Eino tool definition |
| **`internal/infrastructure/tools/declarative_tool.go`** | YAML custom tool → HTTP request. `DeclarativeTool{Name, Description, Endpoint, Params, Auth, ConfirmationRequired}`. Execute: build HTTP request from LLM args → call endpoint → return response |

### MCP Protocol

Используем Go MCP SDK (существует `github.com/mark3labs/mcp-go` или аналог). Если нет зрелого — реализуем minimal client:

```go
type MCPClient struct {
    transport Transport  // stdio | http | sse
    tools     []MCPTool  // Discovered tools
}

func (c *MCPClient) Connect(ctx context.Context) error       // Initialize, listTools
func (c *MCPClient) ListTools() []MCPTool                     // Cached after connect
func (c *MCPClient) CallTool(ctx context.Context, name string, args map[string]any) (string, error)
func (c *MCPClient) Close() error
```

### Интеграция с ToolRegistry

**`infrastructure/tools/tool_registry.go`:**
```go
func (r *ToolRegistry) InitMCPServers(servers map[string]config.MCPServerConfig) error {
    for name, cfg := range servers {
        client := mcp.NewClient(cfg.Type, cfg.URL, cfg.Command, cfg.Args, cfg.Env)
        if err := client.Connect(ctx); err != nil {
            log.Warn("MCP server unavailable, skipping", "name", name, "err", err)
            continue  // Graceful degradation
        }
        r.mcpClients[name] = client
    }
}
```

MCP tools добавляются агенту автоматически при ResolveForAgent().

### Валидация этапа

- [ ] MCP stdio server → tools discovered → agent can call
- [ ] MCP HTTP server → tools discovered → agent can call
- [ ] Declarative tool (YAML endpoint) → HTTP request → response
- [ ] MCP server unavailable → agent works without it (graceful degradation)
- [ ] MCP tool timeout → error returned to agent
- [ ] `confirmation_required: true` → agent asks user before calling tool

---

## Этап 5: Developer Kit

**Цель:** LSP + code indexing вынесены в kit. Engine чистый.

### Текущее состояние

LSP и indexing захардкожены:
- `infrastructure/lsp/` — 5 файлов (service, client, installer, bin_directory, server_configs)
- `infrastructure/indexing/` — 5 файлов (chunker, scanner, indexer, store, embeddings)
- `infrastructure/tools/` — lsp_tool, get_function_tool, get_class_tool, get_file_structure_tool, search_code_tool, smart_search_tool (6 файлов)
- Зависимости в ToolDependencies: ChunkStore, Embedder
- Tools разрешаются в resolver.go switch

### Целевое состояние

```
kits/
└── developer/
    ├── kit.go              # Kit interface implementation
    ├── lsp.go              # Из infrastructure/lsp/* (объединить)
    ├── lsp_installer.go    # Из infrastructure/lsp/installer.go
    ├── lsp_configs.go      # Из infrastructure/lsp/server_configs.go
    ├── indexer.go           # Из infrastructure/indexing/* (объединить)
    ├── tools.go            # lsp_tool, symbol tools, search tools
    └── tools_test.go
```

### Kit Interface

**`internal/domain/kit.go`:**
```go
type Kit interface {
    Name() string

    // Session lifecycle
    OnSessionStart(ctx context.Context, session KitSession) error
    OnSessionEnd(ctx context.Context, session KitSession) error

    // Tools provided by kit
    Tools(session KitSession) []tool.InvokableTool

    // Auto-enrichment after tool calls
    PostToolCall(ctx context.Context, session KitSession, toolName string, result string) *Enrichment
}

type KitSession struct {
    SessionID   string
    ProjectRoot string      // Для developer kit: путь к проекту
    ProjectKey  string
}

type Enrichment struct {
    Content string          // Добавляется в контекст перед следующим LLM call
}
```

### Kit Registration

**`internal/infrastructure/kit_registry.go`:**
```go
type KitRegistry struct {
    kits map[string]domain.Kit
}

func NewKitRegistry() *KitRegistry {
    r := &KitRegistry{kits: make(map[string]domain.Kit)}
    // Compile-time registration
    r.Register(developer.NewKit())
    return r
}
```

**`cmd/server/main.go`:**
```go
import "bytebrew-srv/kits/developer"

kitRegistry := infrastructure.NewKitRegistry()
// developer kit auto-registered via NewKitRegistry()
```

### Developer Kit Implementation

**`kits/developer/kit.go`:**
```go
type DeveloperKit struct {
    sessions map[string]*devSession  // sessionID → per-session state
    mu       sync.RWMutex
}

type devSession struct {
    lspService  *LSPService
    indexer     *CodeIndexer
    watcher     *fsnotify.Watcher
    chunkStore  *ChunkStore
    embedder    *EmbeddingsClient
}

func (k *DeveloperKit) Name() string { return "developer" }

func (k *DeveloperKit) OnSessionStart(ctx context.Context, s KitSession) error {
    // 1. Создать LSP service (start language servers)
    // 2. Создать code indexer (scan project, build embeddings)
    // 3. Запустить file watcher
    // 4. Сохранить в k.sessions[s.SessionID]
}

func (k *DeveloperKit) OnSessionEnd(ctx context.Context, s KitSession) error {
    // 1. Stop LSP servers
    // 2. Stop file watcher
    // 3. Cleanup index
    // 4. Delete from k.sessions
}

func (k *DeveloperKit) Tools(s KitSession) []tool.InvokableTool {
    session := k.sessions[s.SessionID]
    return []tool.InvokableTool{
        NewLspTool(session.lspService),
        NewSearchCodeTool(session.chunkStore, session.embedder),
        NewSmartSearchTool(session.chunkStore, session.embedder),
        NewGetFunctionTool(session.chunkStore, session.embedder),
        NewGetClassTool(session.chunkStore, session.embedder),
        NewGetFileStructureTool(session.chunkStore, session.embedder),
    }
}

func (k *DeveloperKit) PostToolCall(ctx context.Context, s KitSession, toolName string, result string) *Enrichment {
    if toolName == "edit_file" || toolName == "write_file" {
        session := k.sessions[s.SessionID]
        diagnostics := session.lspService.GetDiagnostics(ctx)
        if diagnostics != "" {
            return &domain.Enrichment{Content: "LSP Diagnostics:\n" + diagnostics}
        }
    }
    return nil
}
```

### Интеграция с Engine

**Kit tools — per-session.** Kit.Tools() возвращает tools привязанные к конкретной session (LSP service, ChunkStore). Поэтому kit tools **не** хранятся в ToolRegistry. Вместо этого ToolRegistry.ResolveForAgent() вызывает kit.Tools(session) при каждом resolve.

**PostToolCall enrichment — через append к tool result.** Engine не может inject'ить между шагами Eino ReAct loop (управление внутри Eino). Поэтому enrichment append'ится к tool result **перед** возвратом в Eino. LLM видит result + enrichment как единый ответ tool.

**`service/engine/engine.go` — Execute():**
```go
// В начале Execute():
var activeKit domain.Kit
if agentDef.Kit != "" {
    activeKit = kitRegistry.Get(agentDef.Kit)
    if activeKit != nil {
        session := domain.KitSession{SessionID: cfg.SessionID, ProjectRoot: cfg.ProjectRoot, ProjectKey: cfg.ProjectKey}
        activeKit.OnSessionStart(ctx, session)
        defer activeKit.OnSessionEnd(ctx, session)

        // Kit tools добавляются к agent tools
        kitTools := activeKit.Tools(session)
        cfg.Tools = append(cfg.Tools, kitTools...)
    }
}
```

**PostToolCall — в tool wrapper (НЕ в engine):**
```go
// infrastructure/tools/kit_enrichment_wrapper.go
type KitEnrichmentWrapper struct {
    inner tool.InvokableTool
    kit   domain.Kit
    session domain.KitSession
}

func (w *KitEnrichmentWrapper) Invoke(ctx context.Context, args string) (string, error) {
    result, err := w.inner.Invoke(ctx, args)
    if err != nil {
        return result, err
    }
    // Append kit enrichment к результату
    enrichment := w.kit.PostToolCall(ctx, w.session, w.inner.Info().Name, result)
    if enrichment != nil {
        result = result + "\n\n" + enrichment.Content
    }
    return result, nil
}
```

Engine оборачивает tools агента с kit в KitEnrichmentWrapper. Eino видит обогащённый result, не зная про kit.

### Новые файлы

| Файл | Содержимое |
|------|-----------|
| **`internal/infrastructure/tools/kit_enrichment_wrapper.go`** | Wrapper: оборачивает tool, append'ит kit PostToolCall enrichment к result |
| **`internal/domain/kit.go`** | Kit interface |
| **`internal/infrastructure/kit_registry.go`** | KitRegistry: name → Kit |
| **`kits/developer/kit.go`** | Developer Kit implementation |
| **`kits/developer/lsp.go`** | LSP logic |
| **`kits/developer/lsp_installer.go`** | Auto-install |
| **`kits/developer/lsp_configs.go`** | Server configs |
| **`kits/developer/indexer.go`** | Code indexing |
| **`kits/developer/tools.go`** | Kit tools |

### Файлы: перемещение

| Откуда | Куда |
|--------|------|
| `infrastructure/lsp/service.go` | `kits/developer/lsp.go` |
| `infrastructure/lsp/client.go` | `kits/developer/lsp.go` |
| `infrastructure/lsp/installer.go` | `kits/developer/lsp_installer.go` |
| `infrastructure/lsp/bin_directory.go` | `kits/developer/lsp_installer.go` |
| `infrastructure/lsp/server_configs.go` | `kits/developer/lsp_configs.go` |
| `infrastructure/indexing/chunker.go` | `kits/developer/indexer.go` |
| `infrastructure/indexing/scanner.go` | `kits/developer/indexer.go` |
| `infrastructure/indexing/indexer.go` | `kits/developer/indexer.go` |
| `infrastructure/indexing/store.go` | `kits/developer/indexer.go` |
| `infrastructure/indexing/embeddings.go` | `kits/developer/indexer.go` |
| `infrastructure/tools/lsp_tool.go` | `kits/developer/tools.go` |
| `infrastructure/tools/get_function_tool.go` | `kits/developer/tools.go` |
| `infrastructure/tools/get_class_tool.go` | `kits/developer/tools.go` |
| `infrastructure/tools/get_file_structure_tool.go` | `kits/developer/tools.go` |
| `infrastructure/tools/search_code_tool.go` | `kits/developer/tools.go` |
| `infrastructure/tools/smart_search_tool.go` | `kits/developer/tools.go` |

### Файлы: удаляемые после переноса

| Файл | Причина |
|------|---------|
| `infrastructure/lsp/` (вся директория) | Перенесено в kits/developer/ |
| `infrastructure/indexing/` (вся директория) | Перенесено в kits/developer/ |
| 6 tool файлов в infrastructure/tools/ | Перенесено в kits/developer/tools.go |

### Файлы: изменения

| Файл | Изменения |
|------|-----------|
| **`infrastructure/tools/resolver.go`** | Убрать cases: lsp, get_function, get_class, get_file_structure, search_code, smart_search. Они теперь в kit |
| **`infrastructure/tools/deps_provider.go`** | Убрать `ChunkStore`, `Embedder` fields. Они теперь внутри kit |
| **`infrastructure/agent_service.go`** | Убрать создание LSP service и indexing. Добавить KitRegistry initialization |

### Валидация этапа

- [ ] Agent с `kit: "developer"` → LSP работает, symbol search работает, diagnostics inject работает
- [ ] Agent без kit → engine чистый, нет LSP tools
- [ ] `examples/developer.yaml` → полный coding agent работает
- [ ] `examples/sales.yaml` → agent работает без kit, без LSP

**CHECKPOINT 1: Coding agent работает через новую архитектуру**

---

## Этап 6: Job System

**Цель:** единая точка входа (Job) для всех источников задач.

### Новые файлы

| Файл | Содержимое |
|------|-----------|
| **`internal/domain/job.go`** | `type Job struct { ID, Title, Description, AgentName string; Source JobSource; Status JobStatus; Mode JobMode; Result *JobResult; CreatedAt, CompletedAt time.Time }`. Statuses: created, running, completed, failed, needs_input. Sources: chat, cron, webhook, api. Modes: interactive, background |
| **`internal/service/job/queue.go`** | `type Queue struct`. Methods: `Submit(job)`, `Next(agentName) *Job`, `UpdateStatus()`, `List()`, `Get(id)`, `Cancel(id)`. PostgreSQL-backed. Priority queue per agent |
| **`internal/service/job/scheduler.go`** | Cron scheduler: парсит `triggers[].schedule` из YAML → `robfig/cron` → создаёт Job по расписанию |
| **`internal/service/job/webhook_handler.go`** | HTTP handler: `POST /api/webhooks/{path}` → парсит body → создаёт Job из trigger config + webhook body |
| **`internal/infrastructure/persistence/postgres_job_storage.go`** | Jobs table в PostgreSQL. CRUD + status transitions |

### Job ↔ Session relationship

| Источник | Job ↔ Session | Описание |
|----------|:------------:|----------|
| **Chat (CLI/mobile/WS)** | Session содержит много Jobs | Пользователь отправляет сообщения внутри сессии. Каждое сообщение = Job. Session = контейнер диалога |
| **Cron / Webhook / API** | Job создаёт новую Session | Каждый trigger = новый Job = новая Session. Нет предыдущего контекста |

**Chat flow (существующая Session):**
```
WS: create_session → sessionID
WS: send_message(sessionID, message)
  → Job{Source: "chat", SessionID: sessionID, AgentName: "sales", Description: message}
  → processor.ProcessJob(job) — синхронно, внутри существующей Session
  → events стримятся по sessionID (как сейчас)
```

**Cron/Webhook flow (новая Session):**
```
Cron trigger fires
  → Job{Source: "cron", AgentName: "iot-monitor", Description: "Утренний отчёт"}
  → jobQueue.Submit(job)
  → queue worker: session = createSession(job.AgentName)
  → processor.ProcessJob(job) — в background
  → events стримятся по sessionID (подписчики: mobile push, webhook callback)
```

Для chat — Session и Job co-exist, events привязаны к sessionID (без изменений в WS/EventStore).
Для background — Job создаёт ephemeral Session, которая живёт пока Job не завершён.

### Файлы: изменения

| Файл | Изменения |
|------|-----------|
| **`delivery/ws/connection.go`** | `send_message` → создаёт Job (interactive), не вызывает processor напрямую |
| **`service/session_processor/processor.go`** | Новый метод `ProcessJob(job)`. Определяет agent из job.AgentName, создаёт session, выполняет agent. ask_user: если interactive → блокирует, если background → needs_input |
| **`cmd/server/main.go`** | Инициализация JobQueue, Scheduler, WebhookHandler |

### Валидация этапа

- [ ] CLI send_message → Job → agent works → result
- [ ] Cron trigger → Job создаётся автоматически → agent works
- [ ] Webhook POST → Job → agent works
- [ ] Job status API: list, get, cancel
- [ ] Background job + ask_user → needs_input → notification

---

## Этап 7: REST API + SSE

**Цель:** HTTP API для embed интеграции.

### Новые файлы

| Файл | Содержимое |
|------|-----------|
| **`internal/delivery/http/server.go`** | HTTP server (net/http + chi router). Mounts on configurable port |
| **`internal/delivery/http/agent_handler.go`** | `POST /api/v1/agents/{name}/chat` → creates Job(interactive) → SSE stream events. `GET /api/v1/agents` → list agents |
| **`internal/delivery/http/job_handler.go`** | `POST /api/v1/jobs` → create job. `GET /api/v1/jobs` → list. `GET /api/v1/jobs/{id}` → details. `DELETE /api/v1/jobs/{id}` → cancel |
| **`internal/delivery/http/config_handler.go`** | `POST /api/v1/config/reload` → hot-reload YAML. `GET /api/v1/health` → health check |
| **`internal/delivery/http/webhook_handler.go`** | `POST /api/v1/webhooks/{path}` → creates job from trigger config |
| **`internal/delivery/http/auth_middleware.go`** | Bearer token validation middleware |
| **`internal/delivery/http/sse_writer.go`** | SSE event formatting + streaming |

### SSE Event Format

```
event: thinking
data: {"content": "Мне нужно найти товары..."}

event: tool_call
data: {"tool": "search_products", "args": {"query": "ноутбук"}}

event: tool_result
data: {"tool": "search_products", "result": "3 товара найдено"}

event: message
data: {"content": "Вот что я нашёл..."}

event: done
data: {"job_id": "job-123", "status": "completed"}
```

### Валидация этапа

- [ ] `POST /api/v1/agents/sales/chat` → SSE stream
- [ ] `POST /api/v1/jobs` → job created
- [ ] `GET /api/v1/health` → 200 OK
- [ ] Bearer token → 401 без токена

**CHECKPOINT 2: Два use case работают (coding + sales/IoT)**

---

## Этап 8: Knowledge (RAG)

**Цель:** agent knowledge из папки с документами.

### Файлы: новые

| Файл | Содержимое |
|------|-----------|
| **`internal/infrastructure/knowledge/document_chunker.go`** | Чанкер для документов (markdown по заголовкам/параграфам, txt по абзацам, PDF по страницам). Отличается от code chunker (тот по функциям/классам) |
| **`internal/infrastructure/knowledge/store.go`** | Per-agent knowledge store. Scan folder → chunk → embed → pgvector/SQLite. Search(query) → []Chunk |
| **`internal/infrastructure/tools/knowledge_search_tool.go`** | `knowledge_search(query)` → store.Search() → top-N chunks |

### Переиспользование

Embeddings client и vector store logic переиспользуются из существующего code indexing (теперь в developer kit). Общий код выносится:

```
internal/infrastructure/embeddings/  ← shared embeddings client
  ├── client.go                       ← из indexing/embeddings.go
  └── store.go                        ← vector storage interface
```

Developer kit и Knowledge store оба используют `embeddings.Client`.

### Интеграция

Engine при создании агента: если `agentDef.Knowledge != ""` → создать KnowledgeStore для папки → добавить `knowledge_search` tool автоматически.

### Валидация этапа

- [ ] Agent с `knowledge: "./docs/"` → knowledge_search tool available
- [ ] Query по markdown docs → relevant chunks returned
- [ ] Agent без knowledge → tool не available

---

## Этап 9: manage_tasks Engine Integration

**Цель:** engine-level lifecycle для manage_tasks.

### Файлы: изменения

| Файл | Изменения |
|------|-----------|
| **`infrastructure/tools/manage_tasks_tool.go`** | Объединить manage_tasks + manage_subtasks в один tool. Subtasks опциональны. Actions: create, list, get, complete, fail + create_subtask, list_subtasks (если нужны) |
| **`infrastructure/tools/manage_plan_tool.go`** | **УДАЛИТЬ**. manage_tasks покрывает |
| **`infrastructure/persistence/sqlite_plan_storage.go`** | **УДАЛИТЬ** (или оставить deprecated) |
| **`service/agent/agent_pool.go`** | После завершения spawn-агента: проверить manage_tasks → обновить task status → inject pending tasks в контекст parent |
| **`infrastructure/agents/context_reminders.go`** | Убрать PlanReminderProvider. Добавить TaskReminderProvider: если agent имеет manage_tasks → inject pending tasks |

### Логика engine lifecycle

В `agent_pool.go` после agent completion:
```go
func (p *AgentPool) onAgentCompleted(agent *RunningAgent) {
    // ... existing logic ...

    // NEW: если parent agent использует manage_tasks
    if parentHasManageTasks(agent.SessionID) {
        // Inject pending tasks в контекст parent через eventBus
        pendingTasks := p.taskManager.GetPendingTasks(agent.SessionID)
        if len(pendingTasks) > 0 {
            p.eventBus.Publish(agent.SessionID, &domain.AgentEvent{
                Type:    "task_status_update",
                Content: formatPendingTasks(pendingTasks),
            })
        }
    }
}
```

### Domain events

Добавить event types:
```go
const (
    EventTypeTaskCreated    = "task_created"
    EventTypeTaskCompleted  = "task_completed"
    EventTypeTaskFailed     = "task_failed"
    EventTypeTasksRemaining = "tasks_remaining"  // Injected by engine
)
```

### Валидация этапа

- [ ] Supervisor создал tasks → spawn agent → agent completed → task auto-updated
- [ ] Pending tasks injected в контекст supervisor
- [ ] Все tasks completed → engine signals completion
- [ ] manage_plan tool → NOT FOUND (удалён)

---

## Horizontal Scaling — НЕ В SCOPE

Архитектурно заложено (GORM для DB abstraction, interfaces для event broadcasting и job queue), но не реализуется. Реализация — когда будет реальная нагрузка и реальные клиенты.

---

## Этап 10: CLI Abstraction

**Цель:** CLI работает с любым агентом.

### Файлы: изменения (bytebrew-cli)

| Файл | Изменения |
|------|-----------|
| **`src/presentation/`** | Убрать coding-specific UI elements. Generic chat view |
| **WS connection** | Передавать `agent_name` при `create_session`. Новый параметр |
| **CLI args** | `--agent NAME` (обязательный или default из конфига) |
| **`bytebrew agents`** | Новая команда: `GET /api/v1/agents` → список агентов |
| **`bytebrew task`** | Новая команда: `POST /api/v1/jobs` → создать job |

### Mobile app

| Изменение | Описание |
|-----------|----------|
| Agent selector | Список агентов при старте / в drawer. API: `GET /api/v1/agents` |
| Generic chat | Тот же chat view, но без coding-specific widgets для non-coding agents |
| Job list | Новый экран: список jobs с статусами |

---

## Порядок и зависимости

```
Этап 1: Config Engine              [фундамент]
  ↓
Этап 2+3: Tool Registry + Spawn   [per-agent tools + generic spawn]
  ↓
Этап 4: MCP Client                 [внешние tools]
  ↓
Этап 5: Developer Kit              [LSP/indexing → kit]
  ↓
═══ CHECKPOINT 1: coding agent на новой архитектуре ═══
  ↓
Этап 6: Job System                 [cron, webhooks]
  ↓
Этап 7: REST API + SSE             [embed API]
  ↓
═══ CHECKPOINT 2: второй use case работает ═══
  ↓
Этап 8: Knowledge (RAG)
Этап 9: manage_tasks Integration
Этап 10: CLI + Mobile Abstraction
```

**Horizontal Scaling — НЕ В SCOPE.** Архитектурно заложено, реализуется позже.

### Оценка сроков

| Этап | Срок (1 dev + AI) |
|------|:-----------------:|
| 1. Config Engine | 1-2 недели |
| 2+3. Tool Registry + Spawn | 1-2 недели |
| 4. MCP Client | 1-2 недели |
| 5. Developer Kit | 1-2 недели |
| **Checkpoint 1** | **4-8 недель** |
| 6. Job System | 1-2 недели |
| 7. REST API + SSE | 1 неделя |
| **Checkpoint 2** | **2-3 недели** |
| 8. Knowledge (RAG) | 1-2 недели |
| 9. manage_tasks Integration | 1 неделя |
| 10. CLI + Mobile Abstraction | 1 неделя |
| **Итого** | **~9-14 недель** |
