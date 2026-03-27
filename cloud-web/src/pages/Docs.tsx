import React, { useState } from 'react';

/* ------------------------------------------------------------------ */
/*  Navigation structure                                               */
/* ------------------------------------------------------------------ */

const NAV_SECTIONS = [
  {
    title: 'Getting Started',
    items: [
      { id: 'quick-start', label: 'Quick Start' },
      { id: 'configuration', label: 'Configuration' },
      { id: 'api-reference', label: 'API Reference' },
    ],
  },
  {
    title: 'Admin Dashboard',
    items: [
      { id: 'admin-login', label: 'Login' },
      { id: 'admin-agents', label: 'Agents' },
      { id: 'admin-models', label: 'Models' },
      { id: 'admin-mcp', label: 'MCP Servers' },
      { id: 'admin-tasks', label: 'Tasks' },
      { id: 'admin-triggers', label: 'Triggers' },
      { id: 'admin-api-keys', label: 'API Keys' },
      { id: 'admin-settings', label: 'Settings' },
      { id: 'admin-config', label: 'Config Management' },
      { id: 'admin-audit', label: 'Audit Log' },
    ],
  },
  {
    title: 'Core Concepts',
    items: [
      { id: 'concept-agents', label: 'Agents & Lifecycle' },
      { id: 'concept-multi-agent', label: 'Multi-Agent' },
      { id: 'concept-tools', label: 'Tools' },
      { id: 'concept-tasks', label: 'Tasks & Jobs' },
      { id: 'concept-rag', label: 'Knowledge / RAG' },
      { id: 'concept-triggers', label: 'Triggers' },
    ],
  },
  {
    title: 'Examples',
    items: [
      { id: 'example-sales', label: 'Sales Agent' },
      { id: 'example-support', label: 'Support Agent' },
      { id: 'example-devops', label: 'DevOps Monitor' },
      { id: 'example-iot', label: 'IoT Analyzer' },
    ],
  },
];

/* ------------------------------------------------------------------ */
/*  Content map                                                        */
/* ------------------------------------------------------------------ */

const CONTENT_MAP: Record<string, () => React.JSX.Element> = {
  'quick-start': QuickStartContent,
  'configuration': ConfigurationContent,
  'api-reference': ApiReferenceContent,
  'admin-login': AdminLoginContent,
  'admin-agents': AdminAgentsContent,
  'admin-models': AdminModelsContent,
  'admin-mcp': AdminMcpContent,
  'admin-tasks': AdminTasksContent,
  'admin-triggers': AdminTriggersContent,
  'admin-api-keys': AdminApiKeysContent,
  'admin-settings': AdminSettingsContent,
  'admin-config': AdminConfigContent,
  'admin-audit': AdminAuditContent,
  'concept-agents': ConceptAgentsContent,
  'concept-multi-agent': ConceptMultiAgentContent,
  'concept-tools': ConceptToolsContent,
  'concept-tasks': ConceptTasksContent,
  'concept-rag': ConceptRagContent,
  'concept-triggers': ConceptTriggersContent,
  'example-sales': ExampleSalesContent,
  'example-support': ExampleSupportContent,
  'example-devops': ExampleDevopsContent,
  'example-iot': ExampleIotContent,
};

/* ------------------------------------------------------------------ */
/*  Main page                                                          */
/* ------------------------------------------------------------------ */

export function DocsPage() {
  const [activeSection, setActiveSection] = useState('quick-start');
  const [sidebarOpen, setSidebarOpen] = useState(false);

  const ContentComponent = CONTENT_MAP[activeSection] ?? QuickStartContent;

  return (
    <div className="min-h-[calc(100vh-56px)] bg-surface">
      {/* Mobile overlay backdrop */}
      {sidebarOpen && (
        <div
          className="fixed inset-0 z-30 bg-black/50 md:hidden"
          onClick={() => setSidebarOpen(false)}
        />
      )}

      {/* Centered container: sidebar + content */}
      <div className="max-w-6xl mx-auto flex">
        {/* Sidebar */}
        <DocsSidebar
          activeSection={activeSection}
          onSelect={(id) => {
            setActiveSection(id);
            setSidebarOpen(false);
          }}
          isOpen={sidebarOpen}
        />

        {/* Content */}
        <main className="flex-1 min-w-0">
          <div className="max-w-3xl px-6 sm:px-8 py-8 mx-auto">
          {/* Mobile menu button */}
          <button
            className="md:hidden mb-6 flex items-center gap-2 rounded-[8px] border border-border px-3 py-2 text-sm text-text-secondary hover:text-text-primary hover:border-border-hover transition-colors"
            onClick={() => setSidebarOpen(!sidebarOpen)}
          >
            <svg width="16" height="16" viewBox="0 0 16 16" fill="none" className="shrink-0">
              <path d="M2 4h12M2 8h12M2 12h12" stroke="currentColor" strokeWidth="1.5" strokeLinecap="round" />
            </svg>
            Menu
          </button>

          <ContentComponent />
        </div>
        </main>
      </div>
    </div>
  );
}

/* ------------------------------------------------------------------ */
/*  Sidebar                                                            */
/* ------------------------------------------------------------------ */

function DocsSidebar({
  activeSection,
  onSelect,
  isOpen,
}: {
  activeSection: string;
  onSelect: (id: string) => void;
  isOpen: boolean;
}) {
  return (
    <aside
      className={`
        fixed z-40 top-0 left-0 h-full w-60
        md:sticky md:top-0 md:z-0 md:h-[calc(100vh-56px)]
        bg-surface border-r border-border
        overflow-y-auto overscroll-contain
        transition-transform duration-200 ease-in-out
        ${isOpen ? 'translate-x-0' : '-translate-x-full md:translate-x-0'}
        shrink-0
      `}
    >
      <div className="px-4 pt-6 pb-8">
        {/* Mobile close button */}
        <button
          className="md:hidden mb-4 text-text-tertiary hover:text-text-primary transition-colors"
          onClick={() => onSelect(activeSection)}
        >
          <svg width="20" height="20" viewBox="0 0 20 20" fill="none">
            <path d="M6 6l8 8M14 6l-8 8" stroke="currentColor" strokeWidth="1.5" strokeLinecap="round" />
          </svg>
        </button>

        {NAV_SECTIONS.map((section) => (
          <div key={section.title} className="mb-5">
            <h3 className="text-[10px] font-bold uppercase tracking-wider text-text-tertiary mb-2 px-2">
              {section.title}
            </h3>
            <ul className="space-y-0.5">
              {section.items.map((item) => {
                const isActive = activeSection === item.id;
                return (
                  <li key={item.id}>
                    <button
                      onClick={() => onSelect(item.id)}
                      className={`
                        w-full text-left px-2 py-1.5 text-sm rounded-[8px] transition-colors
                        ${isActive
                          ? 'text-brand-accent bg-brand-accent/10 font-medium'
                          : 'text-text-tertiary hover:text-text-primary'
                        }
                      `}
                    >
                      {item.label}
                    </button>
                  </li>
                );
              })}
            </ul>
          </div>
        ))}
      </div>
    </aside>
  );
}

/* ================================================================== */
/*  CONTENT SECTIONS                                                   */
/* ================================================================== */

/* ------------------------------------------------------------------ */
/*  Getting Started > Quick Start                                      */
/* ------------------------------------------------------------------ */

function QuickStartContent() {
  return (
    <div>
      <PageTitle>Quick Start</PageTitle>
      <p className="text-sm text-text-tertiary mb-4">
        Get ByteBrew Engine running with Docker in under 5 minutes. By the end of this guide,
        you will have a working AI agent that responds to messages over a REST API.
      </p>

      <Callout type="info" title="Prerequisites">
        You need <Ic>docker</Ic> and <Ic>docker compose</Ic> installed. ByteBrew Engine runs on
        Linux, macOS, and Windows (WSL2). Minimum 2 GB RAM for the engine + PostgreSQL.
      </Callout>

      <QuickStartStep n={1} title="Start the Engine">
        <p className="text-sm text-text-tertiary mb-3">
          Download the Docker Compose file and start the engine. This spins up two containers:
          the ByteBrew Engine and a PostgreSQL database.
        </p>
        <CodeBlock>{`curl -fsSL https://bytebrew.ai/releases/docker-compose.yml -o docker-compose.yml
docker compose up -d`}</CodeBlock>
        <p className="text-sm text-text-tertiary mt-3">
          The engine starts on port <Ic>8080</Ic> (API) and <Ic>8443</Ic> (Admin Dashboard).
          Verify it is running:
        </p>
        <CodeBlock>{`curl http://localhost:8080/api/v1/health
# {"status":"ok","version":"1.0.0","agents_count":0}`}</CodeBlock>
      </QuickStartStep>

      <QuickStartStep n={2} title="Create your first agent">
        <p className="text-sm text-text-tertiary mb-3">
          Create an <Ic>agents.yaml</Ic> file in the same directory as your <Ic>docker-compose.yml</Ic>.
          This file defines your agents, models, and tools:
        </p>
        <CodeBlock>{`# agents.yaml
agents:
  my-agent:
    model: glm-5
    system: "You are a helpful assistant for our product."
    tools:
      - knowledge_search

models:
  glm-5:
    provider: openai
    api_key: \${OPENAI_API_KEY}`}</CodeBlock>
        <Callout type="tip" title="Environment variables">
          The <Ic>{'${OPENAI_API_KEY}'}</Ic> syntax references an environment variable. Set it in
          your shell (<Ic>export OPENAI_API_KEY=sk-...</Ic>) or in a <Ic>.env</Ic> file next
          to <Ic>docker-compose.yml</Ic>. Never hardcode secrets in YAML.
        </Callout>

        <Callout type="tip" title="Prefer a visual editor?">
          Skip the YAML file and use the Admin Dashboard instead.
          Open <Ic>http://localhost:8443/admin</Ic>, log in, and click <strong className="text-text-secondary">Create Agent</strong>.
          The dashboard lets you configure everything visually — model, system prompt,
          tools, security zones, spawn rules, and more.
          <img
            src="/screenshots/admin-agents.png"
            alt="Admin Dashboard — Agents list with Create Agent button"
            className="mt-3 w-full max-w-4xl rounded-[2px] border border-border overflow-hidden shadow-2xl shadow-brand-accent/5"
            loading="lazy"
          />
        </Callout>
      </QuickStartStep>

      <QuickStartStep n={3} title="Send your first message">
        <p className="text-sm text-text-tertiary mb-3">
          Use the REST API to talk to your agent. The response streams back as Server-Sent Events (SSE),
          so you see tokens as they are generated:
        </p>
        <CodeBlock>{`curl -N http://localhost:8080/api/v1/agents/my-agent/chat \\
  -H "Content-Type: application/json" \\
  -d '{"message": "Hello, what can you do?"}'`}</CodeBlock>
      </QuickStartStep>

      <QuickStartStep n={4} title="See the response">
        <p className="text-sm text-text-tertiary mb-3">
          The engine returns a stream of SSE events. Each event has a <Ic>type</Ic> field that tells you
          what kind of data it contains:
        </p>
        <CodeBlock>{`event: content
data: {"text":"Hello! I'm your product assistant. "}

event: content
data: {"text":"I can help you with product questions, "}

event: content
data: {"text":"documentation search, and more."}

event: done
data: {"session_id":"a1b2c3d4","tokens":42}`}</CodeBlock>
        <p className="text-sm text-text-tertiary mt-3">
          The <Ic>session_id</Ic> in the <Ic>done</Ic> event lets you continue the conversation.
          Pass it in subsequent requests to maintain context:
        </p>
        <CodeBlock>{`curl -N http://localhost:8080/api/v1/agents/my-agent/chat \\
  -H "Content-Type: application/json" \\
  -d '{"message": "Tell me more about that", "session_id": "a1b2c3d4"}'`}</CodeBlock>
      </QuickStartStep>

      <QuickStartStep n={5} title="Open the Admin Dashboard">
        <p className="text-sm text-text-tertiary mb-3">
          Navigate to <Ic>http://localhost:8443/admin</Ic> in your browser to manage agents,
          models, tools, and triggers through a visual interface. Default credentials
          are configured via <Ic>ADMIN_USER</Ic> and <Ic>ADMIN_PASSWORD</Ic> environment variables.
        </p>
        <img
          src="/screenshots/admin-health.png"
          alt="Admin Dashboard — Health page showing engine status and connected agents"
          className="mt-3 w-full max-w-4xl rounded-[2px] border border-border overflow-hidden shadow-2xl shadow-brand-accent/5"
          loading="lazy"
        />
      </QuickStartStep>

      <SectionDivider />
      <WhatNext items={[
        { label: 'Configuration Reference', id: 'configuration' },
        { label: 'API Reference', id: 'api-reference' },
        { label: 'Core Concepts: Agents', id: 'concept-agents' },
        { label: 'Example: Sales Agent', id: 'example-sales' },
      ]} />
    </div>
  );
}

/* ------------------------------------------------------------------ */
/*  Getting Started > Configuration                                    */
/* ------------------------------------------------------------------ */

function ConfigurationContent() {
  return (
    <div>
      <PageTitle>Configuration Reference</PageTitle>
      <p className="text-sm text-text-tertiary mb-4">
        ByteBrew Engine is configured through YAML files or the Admin Dashboard. Both methods
        write to the same PostgreSQL database -- YAML is just a convenient bootstrap format.
        This reference covers every configuration option in detail.
      </p>

      <Callout type="info" title="Two ways to configure">
        You can define everything in a single <Ic>agents.yaml</Ic> file (great for version control
        and GitOps), or use the Admin Dashboard for a visual editor. Changes made in the dashboard
        are persisted to the database immediately. Use <Ic>POST /api/v1/config/import</Ic> to
        sync YAML into the database, or <Ic>GET /api/v1/config/export</Ic> to export the current
        state as YAML.
      </Callout>

      {/* ---- Agent Configuration ---- */}
      <SubSection title="Agent Configuration">
        <p className="text-sm text-text-tertiary mb-4">
          Agents are the core building blocks of ByteBrew. Each agent is an LLM-powered entity
          with its own identity, behavior, tools, and memory. You define agents under
          the <Ic>agents:</Ic> key, where each key is the agent&apos;s unique name.
        </p>

        <ParamTable params={[
          { name: 'model', required: true, default: '--', desc: 'References a model defined in the models: section. Determines which LLM the agent uses for reasoning.' },
          { name: 'system', required: false, default: '--', desc: 'Inline system prompt string that defines the agent\'s personality, role, and behavior rules.' },
          { name: 'system_file', required: false, default: '--', desc: 'Path to a text file containing the system prompt. Mutually exclusive with system:. Useful for long prompts.' },
          { name: 'lifecycle', required: false, default: 'persistent', desc: 'persistent keeps context across sessions. spawn creates a fresh instance per invocation and terminates after.' },
          { name: 'kit', required: false, default: 'none', desc: 'Preset tool bundle. developer adds code-related tools (read_file, edit_file, bash, etc.).' },
          { name: 'tool_execution', required: false, default: 'sequential', desc: 'sequential runs tool calls one at a time. parallel runs independent tool calls concurrently.' },
          { name: 'max_steps', required: false, default: '50', desc: 'Maximum number of reasoning iterations (1-500). Prevents infinite loops in complex tasks.' },
          { name: 'max_context_size', required: false, default: '16000', desc: 'Maximum context window in tokens (1,000-200,000). Older messages are compressed when exceeded.' },
          { name: 'tools', required: false, default: '[]', desc: 'List of built-in tools and custom tool names available to this agent.' },
          { name: 'knowledge', required: false, default: '--', desc: 'Path to a folder of documents for RAG. The engine auto-indexes files at startup.' },
          { name: 'mcp_servers', required: false, default: '[]', desc: 'List of MCP server names (defined in mcp_servers: section) available to this agent.' },
          { name: 'can_spawn', required: false, default: '[]', desc: 'List of agent names this agent can create at runtime. The engine auto-generates spawn_<name> tools.' },
          { name: 'confirm_before', required: false, default: '[]', desc: 'List of tool names that require user confirmation before execution.' },
        ]} />

        <CodeBlock>{`agents:
  sales-agent:
    model: glm-5                       # Required: model from models: section
    system: |                          # Multi-line system prompt
      You are a sales consultant for Acme Corp.
      Always be professional and helpful.
      Never discuss competitor products.
    lifecycle: persistent              # Keep conversation history
    tool_execution: parallel           # Run independent tools concurrently
    max_steps: 100                     # Allow complex multi-step tasks
    max_context_size: 32000            # Larger context for long conversations
    tools:
      - knowledge_search               # Search product docs
      - web_search                     # Search the internet
      - create_order                   # Custom HTTP tool
    knowledge: "./docs/products/"      # Auto-indexed product catalog
    mcp_servers:
      - crm-api                        # CRM integration via MCP
    can_spawn:
      - researcher                     # Can delegate research tasks
    confirm_before:
      - create_order                   # Ask user before placing orders`}</CodeBlock>
      </SubSection>

      {/* ---- System Prompts: Best Practices ---- */}
      <SubSection title="System Prompts: Best Practices">
        <p className="text-sm text-text-tertiary mb-4">
          The system prompt is the most important configuration for an agent. It defines personality,
          capabilities, constraints, and output format. A well-written prompt dramatically improves
          agent reliability.
        </p>

        <h4 className="text-sm font-semibold text-text-primary mt-4 mb-2">Structure of an effective prompt</h4>
        <BulletList items={[
          <><strong className="text-text-secondary">Role definition</strong> -- who the agent is and what organization it belongs to.</>,
          <><strong className="text-text-secondary">Capabilities</strong> -- what tools are available and when to use each one.</>,
          <><strong className="text-text-secondary">Constraints</strong> -- what the agent must never do (guardrails).</>,
          <><strong className="text-text-secondary">Output format</strong> -- how to structure responses (markdown, JSON, bullet points).</>,
          <><strong className="text-text-secondary">Escalation rules</strong> -- when to ask the user vs. act autonomously.</>,
        ]} />

        <CodeBlock>{`# Good: specific role, clear boundaries, actionable instructions
system: |
  You are a customer support agent for ByteStore, an online electronics retailer.

  ## Your capabilities
  - Search the knowledge base for product information and policies
  - Look up order status by order ID
  - Create support tickets for issues you cannot resolve

  ## Rules
  - Always greet the customer by name if available
  - Never share internal pricing or margin data
  - If asked about a competitor, redirect to our product advantages
  - For refund requests over $500, escalate to a human agent

  ## Response format
  - Keep responses concise (2-3 paragraphs max)
  - Use bullet points for lists of options
  - Always end with a follow-up question or next step`}</CodeBlock>

        <Callout type="warning" title="Common mistakes">
          Avoid vague prompts like &quot;You are a helpful assistant.&quot; The more specific your prompt,
          the more consistent the agent&apos;s behavior. Always tell the agent what it should NOT do --
          LLMs are eager to please and will attempt tasks outside their scope unless explicitly told not to.
        </Callout>

        <p className="text-sm text-text-tertiary mt-4 mb-3">
          For long prompts, use <Ic>system_file</Ic> to load from an external file.
          This keeps your YAML clean and lets you version-control prompts separately:
        </p>
        <CodeBlock>{`agents:
  support-bot:
    model: glm-5
    system_file: "./prompts/support-bot.txt"   # Loaded at startup`}</CodeBlock>
      </SubSection>

      {/* ---- Security Zones Explained ---- */}
      <SubSection title="Security Zones Explained">
        <p className="text-sm text-text-tertiary mb-4">
          Every tool in ByteBrew is assigned a security zone that indicates its risk level.
          This helps operators understand what an agent can do and enforce appropriate safeguards.
        </p>

        <ParamTable params={[
          { name: 'Safe', required: false, default: '--', desc: 'Read-only operations with no side effects. Examples: knowledge_search, web_search, list_files. No confirmation needed.' },
          { name: 'Caution', required: false, default: '--', desc: 'Operations that modify state but are reversible. Examples: edit_file, create_ticket, send_email. Consider adding to confirm_before.' },
          { name: 'Dangerous', required: false, default: '--', desc: 'Operations with irreversible side effects. Examples: bash, delete_file, create_order. Strongly recommended for confirm_before.' },
        ]} />

        <Callout type="tip" title="Defense in depth">
          Use <Ic>confirm_before</Ic> for any Caution or Dangerous tool in production.
          This pauses execution and returns a <Ic>needs_input</Ic> event to the client,
          allowing a human to approve or reject the action before it executes.
        </Callout>

        <CodeBlock>{`agents:
  devops-bot:
    model: glm-5
    tools:
      - web_search              # Safe: read-only
      - edit_file               # Caution: modifies files
      - bash                    # Dangerous: arbitrary commands
    confirm_before:
      - bash                    # Require human approval
      - edit_file               # Require human approval`}</CodeBlock>
      </SubSection>

      {/* ---- Environment Variables ---- */}
      <SubSection title="Environment Variables">
        <p className="text-sm text-text-tertiary mb-4">
          ByteBrew supports <Ic>{'${VAR_NAME}'}</Ic> syntax for referencing environment variables
          anywhere in your YAML configuration. Variables are expanded at engine startup, so the
          YAML file never contains actual secrets.
        </p>

        <h4 className="text-sm font-semibold text-text-primary mt-4 mb-2">How it works</h4>
        <BulletList items={[
          <>The engine reads the YAML file and replaces every <Ic>{'${VAR_NAME}'}</Ic> with the value of that environment variable.</>,
          <>If a referenced variable is not set, the engine logs a warning and leaves the placeholder empty.</>,
          <>You can use variables in any string value: URLs, API keys, file paths, even system prompts.</>,
          <>Variables are expanded once at startup (or on hot-reload). They are not re-evaluated per-request.</>,
        ]} />

        <CodeBlock>{`# .env file (loaded by Docker Compose automatically)
OPENAI_API_KEY=sk-proj-abc123
CATALOG_API=https://api.mystore.com/v2
WEBHOOK_SECRET=whsec_xyz789
CRM_API_KEY=crm_live_456

# agents.yaml — references variables, never contains secrets
models:
  glm-5:
    provider: openai
    api_key: \${OPENAI_API_KEY}

tools:
  search_products:
    type: http
    url: "\${CATALOG_API}/products/search"

triggers:
  order-webhook:
    secret: \${WEBHOOK_SECRET}`}</CodeBlock>

        <Callout type="warning" title="Never hardcode secrets">
          If your YAML file is checked into version control (recommended for GitOps),
          all secrets must use <Ic>{'${VAR}'}</Ic> syntax. The engine will refuse to start
          if it detects bare API keys in the configuration file.
        </Callout>
      </SubSection>

      {/* ---- Model Configuration ---- */}
      <SubSection title="Model Configuration">
        <p className="text-sm text-text-tertiary mb-4">
          Models define the LLM backends your agents use. ByteBrew supports any OpenAI-compatible API,
          Anthropic, and local models via Ollama. You can configure multiple models and assign
          different ones to different agents.
        </p>

        <ParamTable params={[
          { name: 'provider', required: true, default: '--', desc: 'LLM provider type: openai (any OpenAI-compatible API), anthropic, or ollama.' },
          { name: 'model', required: false, default: '--', desc: 'Model name as expected by the provider API (e.g., gpt-4o, claude-sonnet-4-20250514, llama3.2).' },
          { name: 'base_url', required: false, default: 'Provider default', desc: 'Custom API endpoint. Required for Ollama and third-party OpenAI-compatible providers.' },
          { name: 'api_key', required: false, default: '--', desc: 'API key for the provider. Use ${VAR} syntax. Not required for Ollama.' },
        ]} />

        <h4 className="text-sm font-semibold text-text-primary mt-6 mb-2">Ollama (local models)</h4>
        <p className="text-sm text-text-tertiary mb-3">
          Run models locally with zero API costs. Install Ollama, pull a model, and point ByteBrew at it:
        </p>
        <CodeBlock>{`# 1. Install Ollama (https://ollama.com)
curl -fsSL https://ollama.com/install.sh | sh

# 2. Pull a model
ollama pull llama3.2
ollama pull qwen2.5-coder:32b

# 3. Configure in ByteBrew
models:
  llama-local:
    provider: ollama
    model: llama3.2
    base_url: "http://localhost:11434/v1"
    api_key: "ollama"              # Ollama ignores the key, but the field is required

  qwen-coder:
    provider: ollama
    model: qwen2.5-coder:32b
    base_url: "http://localhost:11434/v1"
    api_key: "ollama"`}</CodeBlock>

        <Callout type="tip" title="GPU acceleration">
          Ollama uses GPU automatically if available. For 32B+ parameter models, you need at least
          24 GB VRAM (RTX 4090 or A100). Smaller models like llama3.2 (3B) run on 4 GB VRAM or even CPU.
        </Callout>

        <h4 className="text-sm font-semibold text-text-primary mt-6 mb-2">OpenAI-compatible providers</h4>
        <p className="text-sm text-text-tertiary mb-3">
          Any API that follows the OpenAI chat completions format works out of the box.
          Just change the <Ic>base_url</Ic>:
        </p>

        <ParamTable params={[
          { name: 'OpenAI', required: false, default: '--', desc: 'base_url: https://api.openai.com/v1 (default, can be omitted)' },
          { name: 'DeepInfra', required: false, default: '--', desc: 'base_url: https://api.deepinfra.com/v1/openai' },
          { name: 'Together AI', required: false, default: '--', desc: 'base_url: https://api.together.xyz/v1' },
          { name: 'Groq', required: false, default: '--', desc: 'base_url: https://api.groq.com/openai/v1' },
          { name: 'vLLM', required: false, default: '--', desc: 'base_url: http://localhost:8000/v1 (self-hosted)' },
          { name: 'LiteLLM', required: false, default: '--', desc: 'base_url: http://localhost:4000/v1 (proxy)' },
        ]} />

        <CodeBlock>{`models:
  # DeepInfra — pay-per-token cloud inference
  qwen-3-32b:
    provider: openai
    model: Qwen/Qwen3-32B
    base_url: "https://api.deepinfra.com/v1/openai"
    api_key: \${DEEPINFRA_API_KEY}

  # Groq — ultra-fast inference
  llama-groq:
    provider: openai
    model: llama-3.3-70b-versatile
    base_url: "https://api.groq.com/openai/v1"
    api_key: \${GROQ_API_KEY}

  # Self-hosted vLLM
  local-vllm:
    provider: openai
    model: meta-llama/Llama-3.2-8B-Instruct
    base_url: "http://gpu-server:8000/v1"
    api_key: "not-needed"`}</CodeBlock>

        <h4 className="text-sm font-semibold text-text-primary mt-6 mb-2">Anthropic</h4>
        <p className="text-sm text-text-tertiary mb-3">
          Native Anthropic API support with automatic message formatting:
        </p>
        <CodeBlock>{`models:
  claude-sonnet-4:
    provider: anthropic
    model: claude-sonnet-4-20250514
    api_key: \${ANTHROPIC_API_KEY}`}</CodeBlock>
      </SubSection>

      {/* ---- Tool Configuration ---- */}
      <SubSection title="Tool Configuration (Declarative YAML)">
        <p className="text-sm text-text-tertiary mb-4">
          Declarative HTTP tools let you connect agents to any REST API without writing code.
          You define the endpoint, parameters, and authentication in YAML -- the engine handles
          the HTTP request and passes the result back to the LLM.
        </p>

        <ParamTable params={[
          { name: 'type', required: true, default: '--', desc: 'Tool type. Currently only http is supported for declarative tools.' },
          { name: 'method', required: true, default: '--', desc: 'HTTP method: GET, POST, PUT, PATCH, DELETE.' },
          { name: 'url', required: true, default: '--', desc: 'Endpoint URL. Supports ${VAR} for env vars and {{param}} for LLM-provided values.' },
          { name: 'params', required: false, default: '--', desc: 'Query parameters as key-value pairs. Values can use {{param}} placeholders.' },
          { name: 'body', required: false, default: '--', desc: 'Request body (POST/PUT/PATCH). Keys and values can use {{param}} placeholders.' },
          { name: 'headers', required: false, default: '--', desc: 'Additional HTTP headers as key-value pairs.' },
          { name: 'auth', required: false, default: '--', desc: 'Authentication block: type (bearer, basic, header), token/username/password/name/value.' },
          { name: 'confirmation_required', required: false, default: 'false', desc: 'When true, pauses execution and asks the user before making the request.' },
          { name: 'description', required: false, default: '--', desc: 'Human-readable description shown to the LLM. Helps the model decide when to use this tool.' },
        ]} />

        <CodeBlock>{`tools:
  # GET with query parameters
  search_products:
    type: http
    method: GET
    url: "\${CATALOG_API}/products/search"
    description: "Search the product catalog by keyword"
    params:
      query: "{{search_term}}"
      limit: "10"
    auth:
      type: bearer
      token: \${API_TOKEN}

  # POST with JSON body
  create_order:
    type: http
    method: POST
    url: "\${ORDER_API}/orders"
    description: "Create a new order for a customer"
    body:
      customer_id: "{{customer_id}}"
      items: "{{items}}"
      notes: "{{notes}}"
    confirmation_required: true       # Human approval before execution
    auth:
      type: bearer
      token: \${ORDER_API_TOKEN}

  # Basic auth example
  legacy_erp:
    type: http
    method: GET
    url: "\${ERP_URL}/api/inventory/{{sku}}"
    auth:
      type: basic
      username: \${ERP_USER}
      password: \${ERP_PASSWORD}

  # Custom header auth
  internal_api:
    type: http
    method: GET
    url: "http://internal:3000/data"
    auth:
      type: header
      name: "X-Internal-Key"
      value: \${INTERNAL_KEY}`}</CodeBlock>

        <Callout type="tip" title="Placeholders vs environment variables">
          <Ic>{'${VAR}'}</Ic> is expanded at startup from environment variables (static).
          <Ic>{'{{param}}'}</Ic> is filled by the LLM at runtime (dynamic). Use <Ic>{'${}'}</Ic> for
          secrets and base URLs, <Ic>{'{{}}'}</Ic> for user-specific values like search queries and IDs.
        </Callout>
      </SubSection>

      {/* ---- MCP Server Configuration ---- */}
      <SubSection title="MCP Server Configuration">
        <p className="text-sm text-text-tertiary mb-4">
          Model Context Protocol (MCP) servers extend agent capabilities with external tools.
          ByteBrew supports two transport types: <strong className="text-text-secondary">stdio</strong> (the
          engine spawns a local process) and <strong className="text-text-secondary">HTTP/SSE</strong> (the
          engine connects to a remote server).
        </p>

        <ParamTable params={[
          { name: 'command', required: false, default: '--', desc: 'For stdio transport: the command to run (e.g., npx, python, node).' },
          { name: 'args', required: false, default: '[]', desc: 'Command-line arguments for the stdio process.' },
          { name: 'env', required: false, default: '{}', desc: 'Environment variables passed to the stdio process. Supports ${VAR} syntax.' },
          { name: 'type', required: false, default: 'stdio', desc: 'Transport type: http or sse. Omit for stdio (default).' },
          { name: 'url', required: false, default: '--', desc: 'For HTTP/SSE transport: the server URL to connect to.' },
        ]} />

        <CodeBlock>{`mcp_servers:
  # Stdio: Engine spawns the process and communicates over stdin/stdout
  github:
    command: npx
    args: ["-y", "@modelcontextprotocol/server-github"]
    env:
      GITHUB_TOKEN: \${GITHUB_TOKEN}

  # Stdio: Python-based MCP server
  database:
    command: python
    args: ["-m", "mcp_server_postgres"]
    env:
      DATABASE_URL: \${DATABASE_URL}

  # HTTP: Engine connects to a running server
  analytics:
    type: http
    url: "http://analytics-service:3000/mcp"

  # SSE: Engine connects via Server-Sent Events
  realtime-data:
    type: sse
    url: "http://localhost:4000/sse"`}</CodeBlock>

        <Callout type="info" title="Tool discovery">
          When an MCP server connects, the engine discovers its available tools automatically.
          These tools appear in the agent&apos;s tool palette alongside built-in tools. You can
          see discovered tools and their descriptions in the Admin Dashboard under MCP Servers.
        </Callout>
      </SubSection>

      {/* ---- Trigger Configuration ---- */}
      <SubSection title="Trigger Configuration">
        <p className="text-sm text-text-tertiary mb-4">
          Triggers let agents run autonomously without user interaction. Cron triggers
          execute on a schedule; webhook triggers fire when an external service sends an HTTP request.
          Both types create background tasks that the agent processes independently.
        </p>

        <h4 className="text-sm font-semibold text-text-primary mt-4 mb-2">Cron expression reference</h4>
        <ParamTable params={[
          { name: '* * * * *', required: false, default: '--', desc: 'Every minute' },
          { name: '*/5 * * * *', required: false, default: '--', desc: 'Every 5 minutes' },
          { name: '0 */2 * * *', required: false, default: '--', desc: 'Every 2 hours' },
          { name: '0 9 * * 1-5', required: false, default: '--', desc: 'Weekdays at 9:00 AM' },
          { name: '0 9,17 * * *', required: false, default: '--', desc: 'Daily at 9:00 AM and 5:00 PM' },
          { name: '0 0 * * *', required: false, default: '--', desc: 'Every day at midnight' },
          { name: '0 0 * * 0', required: false, default: '--', desc: 'Every Sunday at midnight' },
          { name: '0 0 1 * *', required: false, default: '--', desc: 'First day of each month at midnight' },
          { name: '0 0 1 1 *', required: false, default: '--', desc: 'January 1st at midnight (yearly)' },
        ]} />

        <CodeBlock>{`triggers:
  # Cron trigger — agent runs on a schedule
  morning-report:
    cron: "0 9 * * 1-5"               # Weekdays at 9 AM
    agent: supervisor
    message: "Generate the daily report from all data sources."

  # Webhook trigger — agent responds to external events
  order-webhook:
    type: webhook
    path: /webhooks/orders             # Exposed at POST /api/v1/webhooks/orders
    agent: sales-agent
    secret: \${WEBHOOK_SECRET}          # HMAC-SHA256 signature verification

  # Webhook without signature verification (not recommended for production)
  internal-events:
    type: webhook
    path: /webhooks/internal
    agent: ops-bot`}</CodeBlock>

        <h4 className="text-sm font-semibold text-text-primary mt-6 mb-2">Webhook security</h4>
        <p className="text-sm text-text-tertiary mb-3">
          When a <Ic>secret</Ic> is configured, the engine verifies incoming requests using
          HMAC-SHA256 signature verification. The external service must include the signature
          in the <Ic>X-Webhook-Secret</Ic> header:
        </p>
        <CodeBlock>{`# Sending a verified webhook request
curl -X POST http://localhost:8080/api/v1/webhooks/orders \\
  -H "X-Webhook-Secret: whsec_your_secret_here" \\
  -H "Content-Type: application/json" \\
  -d '{"order_id": "12345", "event": "created", "total": 99.99}'`}</CodeBlock>

        <Callout type="warning" title="Production webhooks">
          Always configure a <Ic>secret</Ic> for production webhook triggers.
          Without signature verification, anyone who knows the URL can trigger your agent.
        </Callout>
      </SubSection>

      {/* ---- Environment Variables note ---- */}
      <SectionDivider />
      <WhatNext items={[
        { label: 'API Reference', id: 'api-reference' },
        { label: 'Admin Dashboard: Agents', id: 'admin-agents' },
        { label: 'Core Concepts: Tools', id: 'concept-tools' },
      ]} />
    </div>
  );
}

/* ------------------------------------------------------------------ */
/*  Getting Started > API Reference                                    */
/* ------------------------------------------------------------------ */

function ApiReferenceContent() {
  return (
    <div>
      <PageTitle>API Reference</PageTitle>
      <p className="text-sm text-text-tertiary mb-4">
        Complete REST API reference for the ByteBrew Engine. All endpoints
        return JSON (except SSE streams) and accept JSON request bodies.
      </p>

      {/* ---- Authentication ---- */}
      <SubSection title="Authentication">
        <p className="text-sm text-text-tertiary mb-4">
          All API requests must include a valid API token in the <Ic>Authorization</Ic> header.
          Tokens are created through the Admin Dashboard and are scoped to specific capabilities.
        </p>

        <h4 className="text-sm font-semibold text-text-primary mt-4 mb-2">Creating an API token</h4>
        <BulletList items={[
          <>Navigate to <strong className="text-text-secondary">Admin Dashboard</strong> &rarr; <strong className="text-text-secondary">API Keys</strong></>,
          <>Click &quot;Create API Key&quot; and select the scopes you need</>,
          <>Copy the token immediately -- it is shown only once and cannot be recovered</>,
          <>Tokens are prefixed with <Ic>bb_</Ic> for easy identification in logs and config</>,
        ]} />

        <h4 className="text-sm font-semibold text-text-primary mt-4 mb-2">Using the token</h4>
        <CodeBlock>{`# curl
curl http://localhost:8080/api/v1/agents \\
  -H "Authorization: Bearer bb_your_api_token"

# JavaScript (fetch)
const response = await fetch('http://localhost:8080/api/v1/agents', {
  headers: { 'Authorization': 'Bearer bb_your_api_token' },
});

# Python (requests)
import requests
response = requests.get(
    'http://localhost:8080/api/v1/agents',
    headers={'Authorization': 'Bearer bb_your_api_token'},
)`}</CodeBlock>

        <h4 className="text-sm font-semibold text-text-primary mt-4 mb-2">Token scopes</h4>
        <ParamTable params={[
          { name: 'chat', required: false, default: '--', desc: 'Send messages to agents (POST /agents/{name}/chat)' },
          { name: 'tasks', required: false, default: '--', desc: 'Create, list, cancel tasks and provide input' },
          { name: 'agents:read', required: false, default: '--', desc: 'List and inspect agent configurations' },
          { name: 'config', required: false, default: '--', desc: 'Reload, export, and import configuration' },
          { name: 'admin', required: false, default: '--', desc: 'Full access to all endpoints including API key management' },
        ]} />

        <Callout type="tip" title="Least privilege">
          Create separate tokens for different integrations. A chatbot frontend only
          needs <Ic>chat</Ic> scope. A CI/CD pipeline might need <Ic>config</Ic> for hot-reload.
          Use <Ic>admin</Ic> scope only for the Admin Dashboard and management scripts.
        </Callout>

        <h4 className="text-sm font-semibold text-text-primary mt-4 mb-2">Error responses</h4>
        <CodeBlock>{`# 401 Unauthorized — missing or invalid token
{"error": "unauthorized", "message": "Invalid or expired API token"}

# 403 Forbidden — token lacks required scope
{"error": "forbidden", "message": "Token does not have 'config' scope"}`}</CodeBlock>
      </SubSection>

      <div className="mb-6 space-y-2">
        <div className="flex items-baseline gap-2">
          <span className="text-sm font-medium text-text-secondary">Base URL:</span>
          <Ic>http://localhost:8080/api/v1</Ic>
        </div>
        <div className="flex items-baseline gap-2 flex-wrap">
          <span className="text-sm font-medium text-text-secondary">Content-Type:</span>
          <Ic>application/json</Ic>
        </div>
      </div>

      {/* ---- Chat ---- */}
      <SubSection title="Chat (SSE Streaming)">
        <p className="text-sm text-text-tertiary mb-3">
          Send a message to an agent and receive a stream of Server-Sent Events. This is the
          primary endpoint for building conversational interfaces.
        </p>

        <CodeBlock>{`POST /api/v1/agents/{name}/chat`}</CodeBlock>

        <h4 className="text-sm font-semibold text-text-primary mt-4 mb-2">Request body</h4>
        <ParamTable params={[
          { name: 'message', required: true, default: '--', desc: 'The user message to send to the agent.' },
          { name: 'session_id', required: false, default: 'auto-generated', desc: 'Session ID for continuing a conversation. Omit to start a new session.' },
        ]} />

        <h4 className="text-sm font-semibold text-text-primary mt-4 mb-2">Full example</h4>
        <CodeBlock>{`# Start a new conversation
curl -N http://localhost:8080/api/v1/agents/sales-agent/chat \\
  -H "Authorization: Bearer bb_your_token" \\
  -H "Content-Type: application/json" \\
  -d '{"message": "What laptops do you have under $1000?"}'`}</CodeBlock>

        <h4 className="text-sm font-semibold text-text-primary mt-4 mb-2">SSE event types</h4>
        <ParamTable params={[
          { name: 'content', required: false, default: '--', desc: 'Text chunk from the agent. Concatenate all content events for the full response.' },
          { name: 'tool_call', required: false, default: '--', desc: 'Agent is calling a tool. Contains tool name and input parameters.' },
          { name: 'tool_result', required: false, default: '--', desc: 'Result returned from the tool execution.' },
          { name: 'error', required: false, default: '--', desc: 'An error occurred during processing.' },
          { name: 'done', required: false, default: '--', desc: 'Stream is complete. Contains session_id and token count.' },
        ]} />

        <h4 className="text-sm font-semibold text-text-primary mt-4 mb-2">Example response stream</h4>
        <CodeBlock>{`event: content
data: {"text":"I found several laptops under $1000. "}

event: tool_call
data: {"tool":"search_products","input":{"query":"laptops under 1000","limit":"5"}}

event: tool_result
data: {"tool":"search_products","output":"[{\"name\":\"ProBook 450\",\"price\":849}...]"}

event: content
data: {"text":"Here are the top options:\\n\\n1. **ProBook 450** — $849..."}

event: done
data: {"session_id":"a1b2c3d4-e5f6-7890-abcd-ef1234567890","tokens":234}`}</CodeBlock>

        <h4 className="text-sm font-semibold text-text-primary mt-4 mb-2">Continue the conversation</h4>
        <CodeBlock>{`curl -N http://localhost:8080/api/v1/agents/sales-agent/chat \\
  -H "Authorization: Bearer bb_your_token" \\
  -H "Content-Type: application/json" \\
  -d '{"message": "Tell me more about the ProBook 450", "session_id": "a1b2c3d4-e5f6-7890-abcd-ef1234567890"}'`}</CodeBlock>
      </SubSection>

      {/* ---- Agents ---- */}
      <SubSection title="Agents">
        <p className="text-sm text-text-tertiary mb-3">
          List and inspect configured agents. Requires <Ic>agents:read</Ic> or <Ic>admin</Ic> scope.
        </p>
        <CodeBlock>{`# List all agents
curl http://localhost:8080/api/v1/agents \\
  -H "Authorization: Bearer bb_your_token"

# Response
{
  "agents": [
    {
      "name": "sales-agent",
      "model": "glm-5",
      "lifecycle": "persistent",
      "tools_count": 5,
      "has_knowledge": true
    }
  ]
}

# Get agent details
curl http://localhost:8080/api/v1/agents/sales-agent \\
  -H "Authorization: Bearer bb_your_token"

# Response
{
  "name": "sales-agent",
  "model": "glm-5",
  "lifecycle": "persistent",
  "tools": ["knowledge_search", "search_products", "create_order"],
  "mcp_servers": ["crm-api"],
  "can_spawn": ["researcher"],
  "max_steps": 50,
  "max_context_size": 16000
}`}</CodeBlock>
      </SubSection>

      {/* ---- Sessions ---- */}
      <SubSection title="Sessions">
        <p className="text-sm text-text-tertiary mb-3">
          Manage conversation sessions. Sessions store the full message history between a user
          and an agent. Requires <Ic>chat</Ic> or <Ic>admin</Ic> scope.
        </p>
        <CodeBlock>{`# List sessions (with optional filters)
curl "http://localhost:8080/api/v1/sessions?agent=sales-agent&limit=10" \\
  -H "Authorization: Bearer bb_your_token"

# Response
{
  "sessions": [
    {
      "id": "a1b2c3d4",
      "agent": "sales-agent",
      "created_at": "2025-03-19T10:00:00Z",
      "message_count": 12
    }
  ]
}

# Get session with messages
curl http://localhost:8080/api/v1/sessions/a1b2c3d4 \\
  -H "Authorization: Bearer bb_your_token"

# Delete session
curl -X DELETE http://localhost:8080/api/v1/sessions/a1b2c3d4 \\
  -H "Authorization: Bearer bb_your_token"`}</CodeBlock>
      </SubSection>

      {/* ---- Tasks ---- */}
      <SubSection title="Tasks">
        <p className="text-sm text-text-tertiary mb-3">
          Create and manage agent tasks. Tasks are units of work that agents process
          asynchronously -- they can be created by users, triggers, or other agents. Requires <Ic>tasks</Ic> or <Ic>admin</Ic> scope.
        </p>
        <CodeBlock>{`# Create a task
curl -X POST http://localhost:8080/api/v1/tasks \\
  -H "Authorization: Bearer bb_your_token" \\
  -H "Content-Type: application/json" \\
  -d '{
    "agent": "researcher",
    "title": "Market analysis Q1",
    "description": "Analyze Q1 2025 market trends for the SaaS sector"
  }'

# Response
{
  "id": "task_abc123",
  "agent": "researcher",
  "title": "Market analysis Q1",
  "status": "pending",
  "created_at": "2025-03-19T14:30:00Z"
}

# List tasks with filters
curl "http://localhost:8080/api/v1/tasks?status=pending&agent=researcher" \\
  -H "Authorization: Bearer bb_your_token"

# Get task details
curl http://localhost:8080/api/v1/tasks/task_abc123 \\
  -H "Authorization: Bearer bb_your_token"

# Cancel a task (pending or in_progress only)
curl -X DELETE http://localhost:8080/api/v1/tasks/task_abc123 \\
  -H "Authorization: Bearer bb_your_token"

# Provide input to a waiting task (status: needs_input)
curl -X POST http://localhost:8080/api/v1/tasks/task_abc123/input \\
  -H "Authorization: Bearer bb_your_token" \\
  -H "Content-Type: application/json" \\
  -d '{"input": "Focus on enterprise segment and include competitor analysis"}'`}</CodeBlock>

        <h4 className="text-sm font-semibold text-text-primary mt-4 mb-2">Task statuses</h4>
        <ParamTable params={[
          { name: 'pending', required: false, default: '--', desc: 'Task created, waiting to be picked up by the agent.' },
          { name: 'in_progress', required: false, default: '--', desc: 'Agent is actively working on the task.' },
          { name: 'needs_input', required: false, default: '--', desc: 'Agent paused and waiting for user input (e.g., confirmation).' },
          { name: 'completed', required: false, default: '--', desc: 'Task finished successfully.' },
          { name: 'failed', required: false, default: '--', desc: 'Task failed due to an error.' },
          { name: 'cancelled', required: false, default: '--', desc: 'Task was cancelled by a user or API call.' },
          { name: 'escalated', required: false, default: '--', desc: 'Agent escalated the task to a human operator.' },
        ]} />
      </SubSection>

      {/* ---- Config ---- */}
      <SubSection title="Config">
        <p className="text-sm text-text-tertiary mb-3">
          Manage engine configuration at runtime. Hot-reload applies changes without restarting
          the engine. Export/import enable GitOps workflows. Requires <Ic>config</Ic> or <Ic>admin</Ic> scope.
        </p>
        <CodeBlock>{`# Hot-reload configuration from the database
curl -X POST http://localhost:8080/api/v1/config/reload \\
  -H "Authorization: Bearer bb_your_token"

# Response
{"status":"ok","agents_loaded":4,"models_loaded":3}

# Export current config as YAML (secrets are excluded)
curl http://localhost:8080/api/v1/config/export \\
  -H "Authorization: Bearer bb_your_token" \\
  -o config-backup.yaml

# Import YAML config (merges with existing)
curl -X POST http://localhost:8080/api/v1/config/import \\
  -H "Authorization: Bearer bb_your_token" \\
  -H "Content-Type: application/x-yaml" \\
  --data-binary @new-config.yaml

# Response
{"status":"ok","agents_imported":2,"models_imported":1,"tools_imported":3}`}</CodeBlock>
      </SubSection>

      {/* ---- Health ---- */}
      <SubSection title="Health">
        <p className="text-sm text-text-tertiary mb-3">
          Check engine status. No authentication required -- useful for load balancer health checks.
        </p>
        <CodeBlock>{`curl http://localhost:8080/api/v1/health

# Response
{
  "status": "ok",
  "version": "1.0.0",
  "agents_count": 4,
  "uptime": "2h34m12s"
}`}</CodeBlock>
      </SubSection>

      {/* ---- BYOK ---- */}
      <SubSection title="BYOK Headers (per-request model override)">
        <p className="text-sm text-text-tertiary mb-3">
          Bring Your Own Key lets API consumers override the model for a single request.
          This must be enabled in Settings for each provider. Useful for multi-tenant
          deployments where each customer provides their own API key.
        </p>
        <CodeBlock>{`curl -N http://localhost:8080/api/v1/agents/my-agent/chat \\
  -H "Authorization: Bearer bb_your_token" \\
  -H "Content-Type: application/json" \\
  -H "X-Model-Provider: anthropic" \\
  -H "X-Model-API-Key: sk-ant-user-provided-key" \\
  -H "X-Model-Name: claude-sonnet-4-20250514" \\
  -d '{"message": "Hello"}'`}</CodeBlock>

        <Callout type="warning" title="Security consideration">
          BYOK headers are only accepted when the corresponding provider is explicitly enabled
          in Settings. By default, all providers are disabled for BYOK. The user-provided key
          is used for that single request only and is never stored.
        </Callout>
      </SubSection>

      <SectionDivider />
      <WhatNext items={[
        { label: 'Admin Dashboard: API Keys', id: 'admin-api-keys' },
        { label: 'Core Concepts: Tasks', id: 'concept-tasks' },
        { label: 'Example: Sales Agent', id: 'example-sales' },
      ]} />
    </div>
  );
}

/* ------------------------------------------------------------------ */
/*  Admin Dashboard content sections                                   */
/* ------------------------------------------------------------------ */

function AdminLoginContent() {
  return (
    <div>
      <PageTitle>Admin Dashboard: Login</PageTitle>
      <p className="text-sm text-text-tertiary mb-4">
        The Admin Dashboard is a web-based interface for managing all aspects of your ByteBrew Engine.
        It is protected by username/password authentication, and access credentials are configured
        through environment variables.
      </p>

      <SubSection title="Accessing the Dashboard">
        <BulletList items={[
          <>Navigate to <Ic>http://localhost:8443/admin</Ic> in your browser (default URL).</>,
          <>Enter the credentials configured via <Ic>ADMIN_USER</Ic> and <Ic>ADMIN_PASSWORD</Ic> environment variables.</>,
          <>On successful login, a JWT token is issued with a 24-hour expiration.</>,
          <>The token is stored in <Ic>localStorage</Ic> and sent automatically with all API requests.</>,
        ]} />
        <CodeBlock>{`# Set credentials in your docker-compose.yml or .env file
ADMIN_USER=admin
ADMIN_PASSWORD=your-secure-password

# The dashboard is served at:
# http://localhost:8443/admin`}</CodeBlock>
      </SubSection>

      <SubSection title="Security recommendations">
        <BulletList items={[
          <><strong className="text-text-secondary">Change default credentials</strong> -- never use &quot;admin/admin&quot; in production.</>,
          <><strong className="text-text-secondary">Use HTTPS</strong> -- put a reverse proxy (Caddy, nginx) in front of the engine with TLS.</>,
          <><strong className="text-text-secondary">Network isolation</strong> -- restrict dashboard access to internal networks or VPN.</>,
          <><strong className="text-text-secondary">Token expiration</strong> -- tokens expire after 24 hours. Re-login is required after expiration.</>,
        ]} />

        <Callout type="warning" title="No multi-user support yet">
          The current Admin Dashboard supports a single admin user. Multi-user support with
          role-based access control is planned for a future release. For team access,
          share the admin credentials securely or use API keys with scoped permissions.
        </Callout>
      </SubSection>

      <SubSection title="Troubleshooting">
        <BulletList items={[
          <><strong className="text-text-secondary">Login fails with correct credentials</strong> -- verify <Ic>ADMIN_USER</Ic> and <Ic>ADMIN_PASSWORD</Ic> are set and the engine was restarted after changing them.</>,
          <><strong className="text-text-secondary">Dashboard returns 401 after a while</strong> -- your JWT token expired. Reload the page to trigger a re-login.</>,
          <><strong className="text-text-secondary">Dashboard not loading</strong> -- check that port 8443 is exposed in Docker and not blocked by a firewall.</>,
        ]} />
      </SubSection>

      <SectionDivider />
      <WhatNext items={[
        { label: 'Agents', id: 'admin-agents' },
        { label: 'Models', id: 'admin-models' },
        { label: 'API Keys', id: 'admin-api-keys' },
      ]} />
    </div>
  );
}

function AdminAgentsContent() {
  return (
    <div>
      <PageTitle>Admin Dashboard: Agents</PageTitle>
      <p className="text-sm text-text-tertiary mb-4">
        The Agents page is your central hub for creating, configuring, and managing AI agents.
        Each agent is a self-contained entity with its own model, personality (system prompt),
        tools, and memory scope.
      </p>

      <SubSection title="Agent list view">
        <p className="text-sm text-text-tertiary mb-3">
          The main view shows a table of all configured agents with key information at a glance:
        </p>
        <BulletList items={[
          <><strong className="text-text-secondary">Name</strong> -- unique identifier (lowercase, alphanumeric, hyphens only).</>,
          <><strong className="text-text-secondary">Kit</strong> -- preset tool bundle (<Ic>none</Ic> or <Ic>developer</Ic>).</>,
          <><strong className="text-text-secondary">Tools count</strong> -- total number of tools available to the agent.</>,
          <><strong className="text-text-secondary">Knowledge</strong> -- whether a knowledge base is configured (RAG).</>,
        ]} />
        <p className="text-sm text-text-tertiary mt-3">
          Click any agent row to open a side panel with the full configuration. From there
          you can edit settings, view tools by security zone, or delete the agent.
        </p>
      </SubSection>

      <SubSection title="Creating an agent">
        <p className="text-sm text-text-tertiary mb-3">
          Click &quot;Create Agent&quot; to open the agent form. Here is a walkthrough of each field:
        </p>

        <ParamTable params={[
          { name: 'name', required: true, default: '--', desc: 'Unique identifier. Lowercase, alphanumeric + hyphens. Used in API endpoints and spawn references.' },
          { name: 'model', required: true, default: '--', desc: 'Dropdown populated from configured models. Determines the LLM backend.' },
          { name: 'system', required: true, default: '--', desc: 'System prompt that defines agent behavior. This is the most important field.' },
          { name: 'kit', required: false, default: 'none', desc: 'none = no preset tools. developer = adds read_file, edit_file, bash, and other dev tools.' },
          { name: 'lifecycle', required: false, default: 'persistent', desc: 'persistent = accumulates context across sessions. spawn = fresh context each time.' },
          { name: 'tool_execution', required: false, default: 'sequential', desc: 'sequential = one tool at a time. parallel = concurrent tool execution.' },
          { name: 'max_steps', required: false, default: '50', desc: 'Maximum reasoning iterations (1-500). Higher = more complex tasks, more tokens.' },
          { name: 'max_context_size', required: false, default: '16000', desc: 'Context window in tokens (1,000-200,000). Older messages are compressed when exceeded.' },
          { name: 'tools', required: false, default: '[]', desc: 'Select from available tools, grouped by security zone (Safe, Caution, Dangerous).' },
          { name: 'mcp_servers', required: false, default: '[]', desc: 'Multi-select from configured MCP servers.' },
          { name: 'can_spawn', required: false, default: '[]', desc: 'Which other agents this one can create at runtime.' },
          { name: 'confirm_before', required: false, default: '[]', desc: 'Tools that require user confirmation before execution.' },
        ]} />

        <Callout type="tip" title="Start simple, then iterate">
          Begin with a focused system prompt, 2-3 tools, and the default settings. Test the agent
          through the chat interface, then add more tools and tweak <Ic>max_steps</Ic> and
          <Ic>max_context_size</Ic> based on the agent&apos;s actual workload.
        </Callout>
      </SubSection>

      <SubSection title="YAML equivalent">
        <p className="text-sm text-text-tertiary mb-3">
          Everything configured through the form can also be expressed in YAML:
        </p>
        <CodeBlock>{`agents:
  my-agent:
    model: glm-5                    # Model dropdown
    system: |                       # System prompt
      You are a sales consultant for Acme Corp.
      Always be professional and helpful.
    kit: developer                  # Kit: none | developer
    lifecycle: persistent           # persistent | spawn
    tool_execution: parallel        # sequential | parallel
    max_steps: 100                  # 1-500
    max_context_size: 32000         # 1000-200000
    tools:                          # Grouped by security zone
      - web_search                  # Safe
      - edit_file                   # Caution
      - bash                        # Dangerous
    mcp_servers:
      - github-server
    can_spawn:
      - researcher
    confirm_before:
      - bash
      - create_order`}</CodeBlock>
      </SubSection>

      <SectionDivider />
      <WhatNext items={[
        { label: 'Models', id: 'admin-models' },
        { label: 'MCP Servers', id: 'admin-mcp' },
        { label: 'Core Concepts: Agents', id: 'concept-agents' },
      ]} />
    </div>
  );
}

function AdminModelsContent() {
  return (
    <div>
      <PageTitle>Admin Dashboard: Models</PageTitle>
      <p className="text-sm text-text-tertiary mb-4">
        The Models page lets you configure LLM providers and endpoints. Each model entry
        defines how the engine connects to an LLM backend -- you can have multiple models
        from different providers and assign each agent its own model.
      </p>

      <SubSection title="Supported providers">
        <ParamTable params={[
          { name: 'ollama', required: false, default: '--', desc: 'Local model inference via Ollama. Free, private, no API key needed. Requires Ollama installed on the host.' },
          { name: 'openai_compatible', required: false, default: '--', desc: 'Any API that follows OpenAI chat completions format. Works with OpenAI, DeepInfra, Together, Groq, vLLM, LiteLLM.' },
          { name: 'anthropic', required: false, default: '--', desc: 'Native Anthropic API. Supports Claude models with automatic message format conversion.' },
        ]} />
      </SubSection>

      <SubSection title="Adding a model">
        <p className="text-sm text-text-tertiary mb-3">
          Click &quot;Add Model&quot; and fill in the fields:
        </p>
        <BulletList items={[
          <><strong className="text-text-secondary">Display name</strong> -- a human-readable name used in the agent configuration dropdown.</>,
          <><strong className="text-text-secondary">Provider</strong> -- select from the supported providers above.</>,
          <><strong className="text-text-secondary">Model name</strong> -- the model identifier as expected by the provider API (e.g., <Ic>llama3.2</Ic>, <Ic>claude-sonnet-4-20250514</Ic>).</>,
          <><strong className="text-text-secondary">Base URL</strong> -- custom endpoint URL. Required for Ollama and third-party providers. Leave empty for default OpenAI/Anthropic endpoints.</>,
          <><strong className="text-text-secondary">API Key</strong> -- provider API key. Not needed for Ollama. Use the <Ic>{'${VAR}'}</Ic> syntax when configuring via YAML.</>,
        ]} />

        <Callout type="info" title="Model validation">
          After adding a model, the engine attempts a test connection to verify the endpoint
          is reachable and the API key is valid. If the connection fails, the model is saved
          but marked with a warning indicator in the list.
        </Callout>
      </SubSection>

      <SubSection title="Configuration examples">
        <CodeBlock>{`# Ollama (local, no API key needed)
models:
  llama-local:
    provider: ollama
    model: llama3.2
    base_url: "http://localhost:11434/v1"

# OpenAI-compatible (DeepInfra)
models:
  qwen-3-32b:
    provider: openai_compatible
    model: Qwen/Qwen3-32B
    base_url: "https://api.deepinfra.com/v1/openai"
    api_key: \${DEEPINFRA_API_KEY}

# Anthropic
models:
  claude-sonnet-4:
    provider: anthropic
    model: claude-sonnet-4-20250514
    api_key: \${ANTHROPIC_API_KEY}`}</CodeBlock>
      </SubSection>

      <SectionDivider />
      <WhatNext items={[
        { label: 'MCP Servers', id: 'admin-mcp' },
        { label: 'Configuration: Models', id: 'configuration' },
      ]} />
    </div>
  );
}

function AdminMcpContent() {
  return (
    <div>
      <PageTitle>Admin Dashboard: MCP Servers</PageTitle>
      <p className="text-sm text-text-tertiary mb-4">
        Model Context Protocol (MCP) is an open standard for connecting AI agents to external
        tools and data sources. The MCP Servers page lets you add, configure, and monitor
        MCP server connections.
      </p>

      <SubSection title="Transport types">
        <ParamTable params={[
          { name: 'stdio', required: false, default: '--', desc: 'The engine spawns a local process and communicates over stdin/stdout. Best for npm packages and local scripts.' },
          { name: 'http', required: false, default: '--', desc: 'The engine connects to a running HTTP server. Best for remote services and microservices.' },
          { name: 'sse', required: false, default: '--', desc: 'The engine connects via Server-Sent Events. Best for real-time data streams.' },
        ]} />
      </SubSection>

      <SubSection title="Adding from catalog">
        <p className="text-sm text-text-tertiary mb-3">
          The catalog contains pre-configured, well-known MCP servers. Adding one is a one-click operation:
        </p>
        <BulletList items={[
          <>Click &quot;Add from Catalog&quot; on the MCP Servers page.</>,
          <>Browse or search for the server you need (GitHub, filesystem, PostgreSQL, etc.).</>,
          <>Click &quot;Add&quot; -- the name, command, and args are pre-filled.</>,
          <>Fill in required environment variables (e.g., <Ic>GITHUB_TOKEN</Ic>) and save.</>,
          <>The engine spawns the process and discovers available tools automatically.</>,
        ]} />
      </SubSection>

      <SubSection title="Adding a custom server">
        <p className="text-sm text-text-tertiary mb-3">
          For servers not in the catalog, click &quot;Add Custom&quot; and fill in the form:
        </p>
        <BulletList items={[
          <><strong className="text-text-secondary">Name</strong> -- unique identifier for referencing from agent configs.</>,
          <><strong className="text-text-secondary">Type</strong> -- stdio, http, or sse.</>,
          <><strong className="text-text-secondary">Command / URL</strong> -- for stdio: the command to run. For http/sse: the server URL.</>,
          <><strong className="text-text-secondary">Args</strong> -- command-line arguments (stdio only).</>,
          <><strong className="text-text-secondary">Environment variables</strong> -- key-value pairs passed to the process (stdio only).</>,
        ]} />

        <CodeBlock>{`# Stdio: Engine spawns the process
mcp_servers:
  github:
    command: npx
    args: ["-y", "@modelcontextprotocol/server-github"]
    env:
      GITHUB_TOKEN: \${GITHUB_TOKEN}

  # Python MCP server
  custom-tools:
    command: python
    args: ["-m", "my_mcp_server"]
    env:
      DATABASE_URL: \${DATABASE_URL}

# HTTP: Engine connects to a running server
  analytics:
    type: http
    url: "http://analytics-service:3000/mcp"

# SSE: Engine connects via Server-Sent Events
  realtime:
    type: sse
    url: "http://localhost:4000/sse"`}</CodeBlock>
      </SubSection>

      <SubSection title="Monitoring and troubleshooting">
        <p className="text-sm text-text-tertiary mb-3">
          Each MCP server shows a status indicator and the count of discovered tools:
        </p>
        <BulletList items={[
          <><strong className="text-text-secondary">Connected (green)</strong> -- the server is running and tools are discovered.</>,
          <><strong className="text-text-secondary">Disconnected (red)</strong> -- the server process crashed or the HTTP endpoint is unreachable.</>,
          <><strong className="text-text-secondary">Tools count</strong> -- number of tools the server exposes. Click to see the full list with descriptions.</>,
        ]} />

        <Callout type="tip" title="Debugging connection issues">
          For stdio servers, check the engine logs for process spawn errors. Common causes:
          the command is not installed (<Ic>npx</Ic> not in PATH), missing environment variables,
          or the npm package failed to install. For HTTP servers, verify the URL is reachable
          from the engine container (<Ic>curl http://server:3000/mcp</Ic> from inside Docker).
        </Callout>
      </SubSection>

      <SectionDivider />
      <WhatNext items={[
        { label: 'Configuration: MCP', id: 'configuration' },
        { label: 'Core Concepts: Tools', id: 'concept-tools' },
        { label: 'Agents', id: 'admin-agents' },
      ]} />
    </div>
  );
}

function AdminTasksContent() {
  return (
    <div>
      <PageTitle>Admin Dashboard: Tasks</PageTitle>
      <p className="text-sm text-text-tertiary mb-4">
        Tasks are units of work that agents process asynchronously. They can be created manually
        through the dashboard, programmatically via the API, or automatically by triggers (cron/webhook).
        The Tasks page gives you visibility into everything your agents are working on.
      </p>

      <SubSection title="Task list and filtering">
        <p className="text-sm text-text-tertiary mb-3">
          The main view shows a paginated table of all tasks with powerful filters:
        </p>
        <BulletList items={[
          <><strong className="text-text-secondary">Status filter</strong> -- <Ic>pending</Ic>, <Ic>in_progress</Ic>, <Ic>completed</Ic>, <Ic>failed</Ic>, <Ic>cancelled</Ic>, <Ic>needs_input</Ic>, <Ic>escalated</Ic>.</>,
          <><strong className="text-text-secondary">Source filter</strong> -- <Ic>agent</Ic> (spawned by another agent), <Ic>cron</Ic>, <Ic>webhook</Ic>, <Ic>api</Ic>, <Ic>dashboard</Ic>.</>,
          <><strong className="text-text-secondary">Agent filter</strong> -- filter by which agent is assigned to the task.</>,
        ]} />
      </SubSection>

      <SubSection title="Creating a task">
        <p className="text-sm text-text-tertiary mb-3">
          Click &quot;Create Task&quot; and fill in the form:
        </p>
        <BulletList items={[
          <><strong className="text-text-secondary">Agent</strong> -- select which agent should handle the task.</>,
          <><strong className="text-text-secondary">Title</strong> -- short description of the task (shown in the list).</>,
          <><strong className="text-text-secondary">Description</strong> -- detailed instructions for the agent (this becomes the message).</>,
        ]} />
        <p className="text-sm text-text-tertiary mt-3">
          The agent starts working on the task immediately. You can track progress in the task
          detail view, which shows the agent&apos;s messages, tool calls, and results.
        </p>
      </SubSection>

      <SubSection title="Task actions">
        <BulletList items={[
          <><strong className="text-text-secondary">Cancel</strong> -- available for tasks in <Ic>pending</Ic> or <Ic>in_progress</Ic> status. The agent stops working immediately.</>,
          <><strong className="text-text-secondary">Provide input</strong> -- for tasks in <Ic>needs_input</Ic> status. The agent paused to ask a question or request confirmation. Type your response and the agent continues.</>,
          <><strong className="text-text-secondary">View details</strong> -- click any task to see the full conversation, tool calls, and results.</>,
        ]} />
      </SubSection>

      <SubSection title="API equivalent">
        <CodeBlock>{`# Create a task
curl -X POST http://localhost:8080/api/v1/tasks \\
  -H "Authorization: Bearer bb_token" \\
  -H "Content-Type: application/json" \\
  -d '{
    "agent": "researcher",
    "title": "Market analysis Q1",
    "description": "Analyze Q1 market trends for SaaS sector"
  }'

# Provide input to a waiting task
curl -X POST http://localhost:8080/api/v1/tasks/{id}/input \\
  -H "Authorization: Bearer bb_token" \\
  -H "Content-Type: application/json" \\
  -d '{"input": "Focus on enterprise segment"}'

# Cancel a task
curl -X DELETE http://localhost:8080/api/v1/tasks/{id} \\
  -H "Authorization: Bearer bb_token"`}</CodeBlock>
      </SubSection>

      <SectionDivider />
      <WhatNext items={[
        { label: 'Core Concepts: Tasks', id: 'concept-tasks' },
        { label: 'Triggers', id: 'admin-triggers' },
        { label: 'API Reference', id: 'api-reference' },
      ]} />
    </div>
  );
}

function AdminTriggersContent() {
  return (
    <div>
      <PageTitle>Admin Dashboard: Triggers</PageTitle>
      <p className="text-sm text-text-tertiary mb-4">
        Triggers enable agents to run autonomously without user interaction. Use cron triggers
        for scheduled tasks (daily reports, periodic checks) and webhook triggers for event-driven
        workflows (order created, payment received, deployment completed).
      </p>

      <SubSection title="Cron triggers">
        <p className="text-sm text-text-tertiary mb-3">
          Schedule agents to run at specific times using standard cron syntax.
          Each cron trigger creates a background task at the scheduled time.
        </p>

        <h4 className="text-sm font-semibold text-text-primary mt-4 mb-2">Common cron patterns</h4>
        <ParamTable params={[
          { name: '*/5 * * * *', required: false, default: '--', desc: 'Every 5 minutes' },
          { name: '0 */2 * * *', required: false, default: '--', desc: 'Every 2 hours' },
          { name: '0 9 * * 1-5', required: false, default: '--', desc: 'Weekdays at 9:00 AM' },
          { name: '0 9,17 * * *', required: false, default: '--', desc: 'Daily at 9:00 AM and 5:00 PM' },
          { name: '0 0 * * *', required: false, default: '--', desc: 'Every day at midnight' },
          { name: '0 0 * * 0', required: false, default: '--', desc: 'Every Sunday at midnight' },
          { name: '0 0 1 * *', required: false, default: '--', desc: 'First day of each month' },
        ]} />

        <CodeBlock>{`triggers:
  morning-report:
    cron: "0 9 * * 1-5"              # Weekdays at 9 AM
    agent: supervisor
    message: "Generate daily report"  # Message sent to the agent`}</CodeBlock>
      </SubSection>

      <SubSection title="Webhook triggers">
        <p className="text-sm text-text-tertiary mb-3">
          Expose HTTP endpoints that external services can call to trigger agents.
          The incoming request body is forwarded to the agent as the message.
        </p>

        <BulletList items={[
          <>The webhook URL follows the pattern <Ic>{'/api/v1/webhooks/<path>'}</Ic>.</>,
          <>Incoming POST body is forwarded to the agent as the task message.</>,
          <>Configure a <Ic>secret</Ic> for HMAC signature verification (recommended for production).</>,
          <>The webhook request is authenticated via the <Ic>X-Webhook-Secret</Ic> header.</>,
        ]} />

        <CodeBlock>{`triggers:
  order-webhook:
    type: webhook
    path: /webhooks/orders             # POST /api/v1/webhooks/orders
    agent: sales-agent
    secret: \${WEBHOOK_SECRET}          # Signature verification

# Trigger the webhook externally:
curl -X POST http://localhost:8080/api/v1/webhooks/orders \\
  -H "X-Webhook-Secret: your-secret" \\
  -H "Content-Type: application/json" \\
  -d '{"order_id": "12345", "event": "created"}'`}</CodeBlock>
      </SubSection>

      <SubSection title="Managing triggers">
        <BulletList items={[
          <><strong className="text-text-secondary">Enable/disable</strong> -- toggle triggers on and off without deleting them. Disabled triggers are retained in configuration but do not fire.</>,
          <><strong className="text-text-secondary">Edit</strong> -- change the schedule, agent, message, or secret at any time.</>,
          <><strong className="text-text-secondary">Delete</strong> -- permanently remove a trigger. Existing tasks created by the trigger are not affected.</>,
          <><strong className="text-text-secondary">History</strong> -- view recent trigger executions in the Audit Log, including task IDs and outcomes.</>,
        ]} />
      </SubSection>

      <SectionDivider />
      <WhatNext items={[
        { label: 'Core Concepts: Triggers', id: 'concept-triggers' },
        { label: 'Configuration: Triggers', id: 'configuration' },
        { label: 'Tasks', id: 'admin-tasks' },
      ]} />
    </div>
  );
}

function AdminApiKeysContent() {
  return (
    <div>
      <PageTitle>Admin Dashboard: API Keys</PageTitle>
      <p className="text-sm text-text-tertiary mb-4">
        API keys authenticate programmatic access to the ByteBrew Engine. Each key can be
        scoped to specific capabilities, allowing you to follow the principle of least privilege.
        Keys are created through the dashboard and can be revoked at any time.
      </p>

      <SubSection title="Creating an API key">
        <BulletList items={[
          <>Click &quot;Create API Key&quot; on the API Keys page.</>,
          <>Give it a descriptive name (e.g., &quot;chatbot-frontend&quot;, &quot;ci-cd-pipeline&quot;).</>,
          <>Select the scopes this key needs (see table below).</>,
          <>Click &quot;Create&quot; -- the key is shown once. Copy it immediately.</>,
        ]} />

        <Callout type="warning" title="Copy immediately">
          The full API key is shown only once at creation time. It is hashed before storage
          in the database and cannot be recovered. If you lose a key, revoke it and create a new one.
        </Callout>
      </SubSection>

      <SubSection title="Available scopes">
        <ParamTable params={[
          { name: 'chat', required: false, default: '--', desc: 'Send messages to agents (POST /agents/{name}/chat). The most common scope for client applications.' },
          { name: 'tasks', required: false, default: '--', desc: 'CRUD operations on /tasks. Create, list, cancel tasks and provide input.' },
          { name: 'agents:read', required: false, default: '--', desc: 'Read-only access to agent configurations (GET /agents).' },
          { name: 'config', required: false, default: '--', desc: 'Reload, export, and import configuration. Useful for CI/CD pipelines.' },
          { name: 'admin', required: false, default: '--', desc: 'Full access to all endpoints including API key management and settings.' },
        ]} />
      </SubSection>

      <SubSection title="Usage examples">
        <CodeBlock>{`# Use an API key in requests
curl http://localhost:8080/api/v1/agents \\
  -H "Authorization: Bearer bb_your_api_token"

# Example: key with chat + tasks scopes
# Can call: POST /agents/{name}/chat, GET/POST/DELETE /tasks
# Cannot call: /config/reload, API key management, settings

# Example: key with config scope only (CI/CD)
curl -X POST http://localhost:8080/api/v1/config/reload \\
  -H "Authorization: Bearer bb_cicd_deploy_token"`}</CodeBlock>
      </SubSection>

      <SubSection title="Revoking a key">
        <p className="text-sm text-text-tertiary mb-3">
          Click the &quot;Revoke&quot; button next to any key in the list. Revocation is immediate --
          any request using that key will receive a <Ic>401 Unauthorized</Ic> response.
          Revocation is logged in the Audit Log.
        </p>
      </SubSection>

      <SectionDivider />
      <WhatNext items={[
        { label: 'API Reference: Authentication', id: 'api-reference' },
        { label: 'Settings', id: 'admin-settings' },
        { label: 'Audit Log', id: 'admin-audit' },
      ]} />
    </div>
  );
}

function AdminSettingsContent() {
  return (
    <div>
      <PageTitle>Admin Dashboard: Settings</PageTitle>
      <p className="text-sm text-text-tertiary mb-4">
        The Settings page controls engine-wide preferences that affect all agents and API requests.
        Currently, it covers BYOK (Bring Your Own Key) configuration and logging levels.
      </p>

      <SubSection title="BYOK (Bring Your Own Key)">
        <p className="text-sm text-text-tertiary mb-3">
          BYOK allows API consumers to override the model for a single request by passing
          their own API key in request headers. This is useful for multi-tenant deployments
          where each customer uses their own LLM account.
        </p>
        <BulletList items={[
          <>BYOK is configured per-provider: you can enable it for OpenAI but disable for Anthropic.</>,
          <>When enabled, the consumer passes <Ic>X-Model-Provider</Ic>, <Ic>X-Model-API-Key</Ic>, and <Ic>X-Model-Name</Ic> headers.</>,
          <>The user-provided key is used for that single request only and is never stored or logged.</>,
          <>If the headers are not present, the agent uses its configured model as normal.</>,
        ]} />
        <CodeBlock>{`# BYOK headers in a request (when enabled for the provider)
curl -N http://localhost:8080/api/v1/agents/my-agent/chat \\
  -H "Authorization: Bearer bb_your_token" \\
  -H "X-Model-Provider: anthropic" \\
  -H "X-Model-API-Key: sk-ant-customer-key" \\
  -H "X-Model-Name: claude-sonnet-4-20250514" \\
  -H "Content-Type: application/json" \\
  -d '{"message": "Hello"}'`}</CodeBlock>
      </SubSection>

      <SubSection title="Logging level">
        <p className="text-sm text-text-tertiary mb-3">
          Change the engine&apos;s logging verbosity at runtime without restarting:
        </p>
        <ParamTable params={[
          { name: 'debug', required: false, default: '--', desc: 'Most verbose. Logs every LLM call, tool execution, and internal state change.' },
          { name: 'info', required: false, default: '--', desc: 'Default. Logs agent activity, task lifecycle, and MCP connections.' },
          { name: 'warn', required: false, default: '--', desc: 'Only warnings and errors. Good for production with stable agents.' },
          { name: 'error', required: false, default: '--', desc: 'Only errors. Minimal output, useful for high-traffic deployments.' },
        ]} />

        <Callout type="tip" title="Debugging agents">
          Set the logging level to <Ic>debug</Ic> temporarily when troubleshooting agent behavior.
          This shows the full LLM prompt, tool calls, and responses. Remember to set it back
          to <Ic>info</Ic> or <Ic>warn</Ic> for production -- debug logging generates significant output.
        </Callout>
      </SubSection>

      <SectionDivider />
      <WhatNext items={[
        { label: 'Config Management', id: 'admin-config' },
        { label: 'API Keys', id: 'admin-api-keys' },
      ]} />
    </div>
  );
}

function AdminConfigContent() {
  return (
    <div>
      <PageTitle>Admin Dashboard: Config Management</PageTitle>
      <p className="text-sm text-text-tertiary mb-4">
        The Config Management page provides three operations for managing engine configuration
        at runtime: hot reload, export, and import. These enable zero-downtime configuration
        changes and GitOps workflows.
      </p>

      <SubSection title="Hot Reload">
        <p className="text-sm text-text-tertiary mb-3">
          Apply configuration changes from the database without restarting the engine. This is
          triggered automatically when you save changes in the dashboard, but you can also
          trigger it manually or via API after a database import.
        </p>
        <BulletList items={[
          <>Agents are re-initialized with updated prompts, tools, and settings.</>,
          <>Active sessions are preserved -- only future messages use the new config.</>,
          <>MCP servers are reconnected if their configuration changed.</>,
          <>Failed reloads are rolled back -- the previous config remains active.</>,
        ]} />
        <CodeBlock>{`# Hot reload via API
curl -X POST http://localhost:8080/api/v1/config/reload \\
  -H "Authorization: Bearer bb_admin_token"

# Response
{"status":"ok","agents_loaded":4,"models_loaded":3}`}</CodeBlock>
      </SubSection>

      <SubSection title="Export">
        <p className="text-sm text-text-tertiary mb-3">
          Download the current configuration as a YAML file. Useful for backups, version control,
          and migrating between environments. Secrets (API keys) are excluded from the export.
        </p>
        <CodeBlock>{`# Export via API
curl http://localhost:8080/api/v1/config/export \\
  -H "Authorization: Bearer bb_admin_token" \\
  -o config-backup.yaml`}</CodeBlock>
        <Callout type="info" title="Secrets handling">
          Exported YAML replaces API keys with <Ic>{'${VAR_NAME}'}</Ic> placeholders. When importing
          into another environment, set the corresponding environment variables.
        </Callout>
      </SubSection>

      <SubSection title="Import">
        <p className="text-sm text-text-tertiary mb-3">
          Upload a YAML file to merge with or replace the current configuration. This is the
          recommended way to deploy configuration changes in CI/CD pipelines.
        </p>
        <CodeBlock>{`# Import via API
curl -X POST http://localhost:8080/api/v1/config/import \\
  -H "Authorization: Bearer bb_admin_token" \\
  -H "Content-Type: application/x-yaml" \\
  --data-binary @new-config.yaml

# Response
{"status":"ok","agents_imported":2,"models_imported":1,"tools_imported":3}`}</CodeBlock>
      </SubSection>

      <SubSection title="GitOps workflow">
        <p className="text-sm text-text-tertiary mb-3">
          A common pattern is to store your <Ic>agents.yaml</Ic> in Git and deploy changes via CI/CD:
        </p>
        <BulletList items={[
          <>Developer edits <Ic>agents.yaml</Ic> in a feature branch.</>,
          <>Pull request is reviewed and merged to main.</>,
          <>CI/CD pipeline runs <Ic>config/import</Ic> followed by <Ic>config/reload</Ic>.</>,
          <>Agents are updated with zero downtime.</>,
        ]} />
      </SubSection>

      <SectionDivider />
      <WhatNext items={[
        { label: 'Audit Log', id: 'admin-audit' },
        { label: 'API Reference: Config', id: 'api-reference' },
      ]} />
    </div>
  );
}

function AdminAuditContent() {
  return (
    <div>
      <PageTitle>Admin Dashboard: Audit Log</PageTitle>
      <p className="text-sm text-text-tertiary mb-4">
        The Audit Log provides a complete, immutable record of all administrative actions
        performed on the engine. Every configuration change, authentication event, and API key
        lifecycle event is captured with full context.
      </p>

      <SubSection title="What is logged">
        <BulletList items={[
          <><strong className="text-text-secondary">Configuration changes</strong> -- creating, updating, or deleting agents, models, tools, triggers, and MCP servers. Includes before/after state.</>,
          <><strong className="text-text-secondary">Authentication events</strong> -- admin login attempts (successful and failed).</>,
          <><strong className="text-text-secondary">API key lifecycle</strong> -- key creation (with scopes) and revocation.</>,
          <><strong className="text-text-secondary">Config operations</strong> -- hot reload, import, and export events.</>,
          <><strong className="text-text-secondary">Settings changes</strong> -- BYOK toggles, logging level changes.</>,
        ]} />
      </SubSection>

      <SubSection title="Filtering and search">
        <p className="text-sm text-text-tertiary mb-3">
          The audit log provides several filters to find specific events:
        </p>
        <BulletList items={[
          <><strong className="text-text-secondary">Actor type</strong> -- filter by who performed the action (admin user, API key, system).</>,
          <><strong className="text-text-secondary">Action</strong> -- create, update, delete, login, reload, import, export.</>,
          <><strong className="text-text-secondary">Resource</strong> -- agent, model, tool, trigger, mcp_server, api_key, config, settings.</>,
          <><strong className="text-text-secondary">Date range</strong> -- select a start and end date to narrow results.</>,
        ]} />
      </SubSection>

      <SubSection title="Audit entry structure">
        <p className="text-sm text-text-tertiary mb-3">
          Click any entry to expand the detail view with the full JSON payload:
        </p>
        <CodeBlock>{`{
  "id": "audit_abc123",
  "actor": "admin",
  "actor_type": "user",
  "action": "update",
  "resource_type": "agent",
  "resource_id": "sales-bot",
  "timestamp": "2025-03-19T14:30:00Z",
  "details": {
    "changes": [
      {
        "field": "max_steps",
        "old_value": 50,
        "new_value": 100
      },
      {
        "field": "tools",
        "old_value": ["web_search"],
        "new_value": ["web_search", "create_order"]
      }
    ]
  }
}`}</CodeBlock>
      </SubSection>

      <Callout type="info" title="Retention">
        Audit log entries are stored in PostgreSQL and retained indefinitely by default. For
        high-volume deployments, consider setting up a retention policy to archive or delete
        entries older than your compliance requirements.
      </Callout>

      <SectionDivider />
      <WhatNext items={[
        { label: 'API Keys', id: 'admin-api-keys' },
        { label: 'Settings', id: 'admin-settings' },
      ]} />
    </div>
  );
}

/* ------------------------------------------------------------------ */
/*  Core Concepts content sections                                     */
/* ------------------------------------------------------------------ */

function ConceptAgentsContent() {
  return (
    <div>
      <PageTitle>Agents & Lifecycle</PageTitle>
      <p className="text-sm text-text-tertiary mb-4">
        An agent in ByteBrew is an LLM-powered entity with a defined identity (system prompt),
        capabilities (tools), and memory scope (lifecycle). Agents are the fundamental building
        blocks of your AI-powered workflows.
      </p>

      <SubSection title="What is an agent?">
        <p className="text-sm text-text-tertiary mb-3">
          At its core, an agent is a loop: receive input, reason about it using an LLM,
          optionally call tools to gather information or take actions, and return a response.
          The system prompt defines who the agent is and how it behaves.
        </p>
        <BulletList items={[
          <><strong className="text-text-secondary">Identity</strong> -- the system prompt gives the agent a role, personality, and knowledge boundaries.</>,
          <><strong className="text-text-secondary">Capabilities</strong> -- tools, MCP servers, and knowledge bases determine what the agent can do.</>,
          <><strong className="text-text-secondary">Memory</strong> -- the lifecycle setting controls whether the agent remembers previous conversations.</>,
          <><strong className="text-text-secondary">Autonomy</strong> -- the agent decides which tools to call and in what order based on the user&apos;s request.</>,
        ]} />
      </SubSection>

      <SubSection title="Lifecycle: persistent vs spawn">
        <p className="text-sm text-text-tertiary mb-4">
          The <Ic>lifecycle</Ic> setting is one of the most important decisions you make
          when configuring an agent. It controls the agent&apos;s memory scope:
        </p>

        <ParamTable params={[
          { name: 'persistent', required: false, default: '--', desc: 'Accumulates context across sessions. Remembers previous conversations. Best for: customer-facing agents, personal assistants, support bots.' },
          { name: 'spawn', required: false, default: '--', desc: 'Fresh context per invocation. No memory between calls. Terminates after completing its task and returns a summary. Best for: sub-agents, one-off research tasks, data processing.' },
        ]} />

        <CodeBlock>{`agents:
  # Persistent: remembers customer history
  support-bot:
    model: glm-5
    lifecycle: persistent
    system: |
      You are a customer support agent. Remember
      previous interactions with each customer.

  # Spawn: fresh context, used for delegation
  researcher:
    model: qwen-3-32b
    lifecycle: spawn
    system: |
      Research the given topic thoroughly.
      Return a structured summary with sources.
    tools:
      - web_search
      - knowledge_search`}</CodeBlock>

        <Callout type="tip" title="When to use spawn">
          Use <Ic>spawn</Ic> for sub-agents in a multi-agent setup. When a supervisor spawns a
          researcher, the researcher gets a clean context focused solely on the research task.
          This keeps the sub-agent focused and prevents context pollution from unrelated conversations.
        </Callout>
      </SubSection>

      <SubSection title="System prompts">
        <p className="text-sm text-text-tertiary mb-3">
          The system prompt is the most important configuration for an agent. It defines the
          agent&apos;s personality, capabilities, constraints, and output format. You can set it
          inline or load it from a file:
        </p>
        <CodeBlock>{`agents:
  # Inline (good for short prompts)
  greeter:
    model: glm-5
    system: "You are a friendly greeter. Welcome users and ask how you can help."

  # Multi-line inline (good for medium prompts)
  analyst:
    model: glm-5
    system: |
      You are a data analyst. When given data, you:
      1. Identify key trends and patterns
      2. Calculate relevant statistics
      3. Provide actionable recommendations

  # External file (good for long, version-controlled prompts)
  enterprise-agent:
    model: glm-5
    system_file: "./prompts/enterprise-agent.txt"`}</CodeBlock>
      </SubSection>

      <SubSection title="Agent capabilities">
        <p className="text-sm text-text-tertiary mb-3">
          Each agent can be configured with a unique combination of capabilities:
        </p>
        <BulletList items={[
          <><strong className="text-text-secondary">Built-in tools</strong> -- <Ic>web_search</Ic>, <Ic>knowledge_search</Ic>, <Ic>manage_tasks</Ic>, <Ic>ask_user</Ic>.</>,
          <><strong className="text-text-secondary">Custom HTTP tools</strong> -- declarative API calls defined in YAML (see Tools docs).</>,
          <><strong className="text-text-secondary">MCP servers</strong> -- external tools via Model Context Protocol.</>,
          <><strong className="text-text-secondary">Knowledge base (RAG)</strong> -- auto-indexed document folder for grounded responses.</>,
          <><strong className="text-text-secondary">Sub-agent spawning</strong> -- ability to create and delegate to other agents.</>,
        ]} />
      </SubSection>

      <SectionDivider />
      <WhatNext items={[
        { label: 'Multi-Agent Orchestration', id: 'concept-multi-agent' },
        { label: 'Tools', id: 'concept-tools' },
        { label: 'Admin: Agents', id: 'admin-agents' },
      ]} />
    </div>
  );
}

function ConceptMultiAgentContent() {
  return (
    <div>
      <PageTitle>Multi-Agent Orchestration</PageTitle>
      <p className="text-sm text-text-tertiary mb-4">
        Multi-agent orchestration lets you build teams of specialized agents that collaborate
        on complex tasks. A supervisor agent coordinates the team, delegating subtasks to
        specialist agents that each have their own tools and expertise.
      </p>

      <SubSection title="How it works">
        <p className="text-sm text-text-tertiary mb-3">
          The orchestration model is simple but powerful:
        </p>
        <BulletList items={[
          <><Ic>can_spawn: [agent-name]</Ic> defines which agents a supervisor can create at runtime.</>,
          <>The engine auto-generates a <Ic>spawn_&lt;name&gt;</Ic> tool for each allowed target.</>,
          <>The LLM decides <strong className="text-text-secondary">when</strong> to spawn based on reasoning. The config limits <strong className="text-text-secondary">what</strong> is possible.</>,
          <>Spawned agents run with <Ic>lifecycle: spawn</Ic> (fresh context, focused on the subtask).</>,
          <>When the sub-agent completes, its summary is returned to the supervisor.</>,
          <>The supervisor integrates the result and continues its own reasoning.</>,
        ]} />
      </SubSection>

      <SubSection title="Spawn tree architecture">
        <p className="text-sm text-text-tertiary mb-3">
          In a multi-agent system, agents form a tree structure. The supervisor sits at the root
          and delegates to specialists. Specialists can even spawn their own sub-agents:
        </p>
        <CodeBlock>{`# Spawn tree visualization:
#
#   supervisor (persistent)
#   |-- sales-agent (spawn)
#   |   |-- inventory-checker (spawn)
#   |-- support-agent (spawn)
#   |-- researcher (spawn)
#
# Each spawn agent gets a fresh context focused solely on its task.
# Results flow back up the tree to the supervisor.`}</CodeBlock>
      </SubSection>

      <SubSection title="When to use multi-agent">
        <BulletList items={[
          <><strong className="text-text-secondary">Complex workflows</strong> -- a single agent cannot handle all aspects of a task (e.g., sales requires product lookup, inventory check, and order creation).</>,
          <><strong className="text-text-secondary">Specialized models</strong> -- use a powerful model for the supervisor (reasoning) and cheaper models for specialists (data retrieval).</>,
          <><strong className="text-text-secondary">Tool isolation</strong> -- a researcher should not have access to order creation tools, and vice versa.</>,
          <><strong className="text-text-secondary">Parallel processing</strong> -- spawn multiple agents simultaneously to work on independent subtasks.</>,
        ]} />
      </SubSection>

      <SubSection title="Full example">
        <p className="text-sm text-text-tertiary mb-3">
          A sales team with a supervisor that delegates to a sales consultant and a support agent:
        </p>
        <CodeBlock>{`agents:
  supervisor:
    model: glm-5                  # Powerful model for coordination
    lifecycle: persistent         # Remembers customer interactions
    can_spawn:
      - sales-agent               # Engine creates spawn_sales_agent tool
      - researcher                # Engine creates spawn_researcher tool
    system: |
      You lead a sales team. When a customer asks about products,
      delegate to the sales-agent. When they need research on
      a topic, delegate to the researcher.

      After receiving results from sub-agents, synthesize
      a final response for the customer.

  sales-agent:
    model: qwen-3-32b             # Cheaper model for data lookup
    lifecycle: spawn              # Fresh context per delegation
    tools:
      - search_products
      - check_inventory
      - create_order
    system: |
      You are a sales consultant. Find products matching
      the customer's needs, check availability, and
      create orders when the customer is ready.

  researcher:
    model: claude-sonnet-4
    lifecycle: spawn
    tools:
      - web_search
      - knowledge_search
    system: |
      Research the given topic thoroughly.
      Return a structured report with:
      - Key findings
      - Supporting data
      - Sources`}</CodeBlock>

        <Callout type="tip" title="Model selection strategy">
          Use your most capable (and expensive) model for the supervisor, since it handles
          the complex reasoning of when to delegate, how to synthesize results, and what to
          tell the user. Use faster, cheaper models for specialist agents that mostly do
          data retrieval and simple processing.
        </Callout>
      </SubSection>

      <SectionDivider />
      <WhatNext items={[
        { label: 'Agents & Lifecycle', id: 'concept-agents' },
        { label: 'Tools', id: 'concept-tools' },
        { label: 'Example: Sales Agent', id: 'example-sales' },
      ]} />
    </div>
  );
}

function ConceptToolsContent() {
  return (
    <div>
      <PageTitle>Tools (MCP + Declarative)</PageTitle>
      <p className="text-sm text-text-tertiary mb-4">
        Tools are the bridge between an agent&apos;s reasoning and the outside world. Without tools,
        an agent can only generate text. With tools, it can search the web, query databases,
        create orders, send notifications, and interact with any API.
      </p>

      <SubSection title="Types of tools">
        <ParamTable params={[
          { name: 'Built-in', required: false, default: '--', desc: 'Pre-built tools included with the engine: web_search, knowledge_search, manage_tasks, ask_user.' },
          { name: 'Declarative HTTP', required: false, default: '--', desc: 'Custom tools defined in YAML that make HTTP requests. No code required.' },
          { name: 'MCP', required: false, default: '--', desc: 'External tools provided by Model Context Protocol servers. Supports any MCP-compatible server.' },
          { name: 'Kit', required: false, default: '--', desc: 'Pre-packaged tool bundles. The developer kit adds read_file, edit_file, bash, and other dev tools.' },
        ]} />
      </SubSection>

      <SubSection title="Built-in tools">
        <ParamTable params={[
          { name: 'web_search', required: false, default: '--', desc: 'Search the internet for information. Returns relevant web page snippets. Security zone: Safe.' },
          { name: 'knowledge_search', required: false, default: '--', desc: 'Search the agent\'s knowledge base (RAG). Automatically injected when knowledge: path is set. Zone: Safe.' },
          { name: 'manage_tasks', required: false, default: '--', desc: 'Create, list, update, and complete tasks. Enables persistent task tracking across sessions. Zone: Safe.' },
          { name: 'ask_user', required: false, default: '--', desc: 'Pause execution and ask the user a question. Useful for clarification or confirmation. Zone: Safe.' },
        ]} />
      </SubSection>

      <SubSection title="Declarative HTTP tools">
        <p className="text-sm text-text-tertiary mb-3">
          Connect agents to any REST API without writing code. Define the endpoint, parameters,
          authentication, and the engine handles the HTTP request:
        </p>
        <CodeBlock>{`tools:
  # GET request with query parameters
  get_weather:
    type: http
    method: GET
    url: "https://api.weather.com/v1/current"
    description: "Get current weather for a city"
    params:
      location: "{{city}}"
      units: "metric"
    auth:
      type: bearer
      token: \${WEATHER_API_KEY}

  # POST request with JSON body
  create_ticket:
    type: http
    method: POST
    url: "\${HELPDESK_API}/tickets"
    description: "Create a support ticket"
    body:
      subject: "{{subject}}"
      description: "{{description}}"
      priority: "{{priority}}"
    confirmation_required: true     # Ask user before executing`}</CodeBlock>
      </SubSection>

      <SubSection title="MCP tools">
        <p className="text-sm text-text-tertiary mb-3">
          MCP (Model Context Protocol) is an open standard for connecting AI agents to external
          tools. Any MCP-compatible server works with ByteBrew:
        </p>
        <CodeBlock>{`mcp_servers:
  github:
    command: npx
    args: ["-y", "@modelcontextprotocol/server-github"]
    env:
      GITHUB_TOKEN: \${GITHUB_TOKEN}

  # Then reference it in agent config:
  agents:
    dev-agent:
      model: glm-5
      mcp_servers:
        - github            # All tools from the GitHub MCP server`}</CodeBlock>
      </SubSection>

      <SubSection title="Per-agent tool isolation">
        <p className="text-sm text-text-tertiary mb-3">
          Each agent sees only the tools listed in its configuration. This is a security and
          reliability feature:
        </p>
        <BulletList items={[
          <>A customer support agent should not have access to <Ic>bash</Ic> or <Ic>delete_file</Ic>.</>,
          <>A researcher should not be able to <Ic>create_order</Ic>.</>,
          <>Different agents can use different MCP servers with different credentials.</>,
        ]} />

        <Callout type="info" title="Tool names must be unique">
          Tool names are globally unique across your configuration. If you define a custom
          tool <Ic>search</Ic> and an MCP server also exposes a tool named <Ic>search</Ic>,
          the custom tool takes precedence and the MCP tool is shadowed.
        </Callout>
      </SubSection>

      <SectionDivider />
      <WhatNext items={[
        { label: 'Configuration: Tools', id: 'configuration' },
        { label: 'Admin: MCP Servers', id: 'admin-mcp' },
        { label: 'Tasks & Jobs', id: 'concept-tasks' },
      ]} />
    </div>
  );
}

function ConceptTasksContent() {
  return (
    <div>
      <PageTitle>Tasks & Job System</PageTitle>
      <p className="text-sm text-text-tertiary mb-4">
        The task system gives agents persistent memory for work items that survive context
        window limits and session boundaries. Tasks are also the mechanism for background
        execution -- triggers create tasks that agents process autonomously.
      </p>

      <SubSection title="Why tasks matter">
        <BulletList items={[
          <><strong className="text-text-secondary">Persistence</strong> -- tasks survive context window compression. Even if the agent forgets the conversation, it always knows what tasks are pending.</>,
          <><strong className="text-text-secondary">Background work</strong> -- cron and webhook triggers create tasks that agents work on without user interaction.</>,
          <><strong className="text-text-secondary">Cross-session tracking</strong> -- a user can create a task in one session and check its status in another.</>,
          <><strong className="text-text-secondary">Audit trail</strong> -- every task has a status history, making it easy to track what happened and when.</>,
        ]} />
      </SubSection>

      <SubSection title="Task lifecycle">
        <CodeBlock>{`# Task status flow:
#
#   pending ──> in_progress ──> completed
#                    |
#                    |──> needs_input ──> in_progress (after input)
#                    |
#                    |──> failed
#                    |
#                    |──> escalated
#
#   Any active status ──> cancelled (manual cancellation)`}</CodeBlock>

        <ParamTable params={[
          { name: 'pending', required: false, default: '--', desc: 'Task created, waiting to be picked up. Transitions to in_progress when the agent starts.' },
          { name: 'in_progress', required: false, default: '--', desc: 'Agent is actively working. Can transition to completed, failed, needs_input, or escalated.' },
          { name: 'needs_input', required: false, default: '--', desc: 'Agent paused and waiting for user input or confirmation.' },
          { name: 'completed', required: false, default: '--', desc: 'Task finished successfully. Terminal state.' },
          { name: 'failed', required: false, default: '--', desc: 'Task failed due to an error. Terminal state.' },
          { name: 'escalated', required: false, default: '--', desc: 'Agent determined the task needs human attention. Terminal state.' },
          { name: 'cancelled', required: false, default: '--', desc: 'Cancelled by user or API. Terminal state.' },
        ]} />
      </SubSection>

      <SubSection title="The manage_tasks tool">
        <p className="text-sm text-text-tertiary mb-3">
          Agents interact with tasks through the built-in <Ic>manage_tasks</Ic> tool. The LLM
          decides when and how to use it based on the conversation:
        </p>
        <CodeBlock>{`# Example conversation flow:
#
# User: "Track the quarterly report preparation"
# Agent: [calls manage_tasks: action=create, title="Quarterly report"]
#         "I've created a task to track that. I'll work on it."
#
# User: "What's on my plate?"
# Agent: [calls manage_tasks: action=list, status=pending]
#         "You have 3 pending tasks:
#          1. Quarterly report preparation
#          2. Customer feedback analysis
#          3. Team standup summary"
#
# Agent: [calls manage_tasks: action=update, id=task_abc, status=completed]
#         "Done! The quarterly report task is now complete."

# Enable for an agent:
agents:
  project-manager:
    model: glm-5
    tools:
      - manage_tasks        # Adds task tracking capability
    system: |
      You are a project manager. Track all work items as tasks.
      When a user mentions something to do, create a task for it.`}</CodeBlock>
      </SubSection>

      <SubSection title="Task sources">
        <p className="text-sm text-text-tertiary mb-3">
          Tasks can be created from multiple sources:
        </p>
        <ParamTable params={[
          { name: 'dashboard', required: false, default: '--', desc: 'Created manually through the Admin Dashboard task form.' },
          { name: 'api', required: false, default: '--', desc: 'Created programmatically via POST /api/v1/tasks.' },
          { name: 'agent', required: false, default: '--', desc: 'Created by an agent using the manage_tasks tool.' },
          { name: 'cron', required: false, default: '--', desc: 'Created automatically by a cron trigger at the scheduled time.' },
          { name: 'webhook', required: false, default: '--', desc: 'Created when an external service sends a POST to a webhook endpoint.' },
        ]} />
      </SubSection>

      <SectionDivider />
      <WhatNext items={[
        { label: 'Admin: Tasks', id: 'admin-tasks' },
        { label: 'Triggers', id: 'concept-triggers' },
        { label: 'API Reference: Tasks', id: 'api-reference' },
      ]} />
    </div>
  );
}

function ConceptRagContent() {
  return (
    <div>
      <PageTitle>Knowledge / RAG</PageTitle>
      <p className="text-sm text-text-tertiary mb-4">
        Retrieval-Augmented Generation (RAG) lets agents answer questions based on your documents.
        Instead of relying solely on the LLM&apos;s training data, the agent searches a knowledge base
        and includes relevant passages in its context before generating a response.
      </p>

      <SubSection title="How it works">
        <BulletList items={[
          <>Set <Ic>knowledge: &quot;./path/&quot;</Ic> in agent config to enable RAG for that agent.</>,
          <>The engine auto-indexes all documents in the folder at startup (Markdown, TXT, PDF, HTML).</>,
          <>A <Ic>knowledge_search</Ic> tool is injected automatically when a knowledge path is configured.</>,
          <>When the agent calls <Ic>knowledge_search</Ic>, the engine performs a vector similarity search and returns the most relevant passages.</>,
          <>The agent uses these passages to generate grounded, accurate responses.</>,
        ]} />

        <CodeBlock>{`# Document indexing flow:
#
#   ./docs/support/
#   |-- faq.md              --> chunked, embedded, indexed
#   |-- returns-policy.txt  --> chunked, embedded, indexed
#   |-- product-guide.pdf   --> extracted, chunked, embedded, indexed
#   |-- setup.html          --> parsed, chunked, embedded, indexed
#
#   Agent calls knowledge_search("return policy for electronics")
#   --> Engine finds the most relevant chunks from returns-policy.txt
#   --> Agent uses them to answer: "Our electronics return policy..."
`}</CodeBlock>
      </SubSection>

      <SubSection title="Configuration">
        <CodeBlock>{`agents:
  support-bot:
    model: glm-5
    knowledge: "./docs/support/"     # Path to knowledge base folder
    tools:
      - knowledge_search             # Injected automatically, but explicit is fine
    system: |
      Answer customer questions using the knowledge base.
      Always cite which document you found the information in.
      If you cannot find the answer, say so honestly -- do not
      make up information.`}</CodeBlock>

        <Callout type="info" title="Supported file formats">
          The engine indexes <Ic>.md</Ic>, <Ic>.txt</Ic>, <Ic>.pdf</Ic>, and <Ic>.html</Ic> files.
          Place files in the knowledge folder and restart the engine (or trigger a hot-reload) to index them.
          Sub-folders are included recursively.
        </Callout>
      </SubSection>

      <SubSection title="Per-agent isolation">
        <p className="text-sm text-text-tertiary mb-3">
          Each agent has its own isolated knowledge base. Agent A cannot search agent B&apos;s documents.
          This is important for multi-tenant deployments and role-based access:
        </p>
        <CodeBlock>{`agents:
  sales-bot:
    knowledge: "./docs/sales/"         # Only sales materials
    tools: [knowledge_search]

  hr-bot:
    knowledge: "./docs/hr-policies/"   # Only HR policies
    tools: [knowledge_search]

# sales-bot cannot see HR policies
# hr-bot cannot see sales materials`}</CodeBlock>
      </SubSection>

      <SubSection title="Best practices">
        <BulletList items={[
          <><strong className="text-text-secondary">Keep documents focused</strong> -- smaller, topic-specific documents work better than large monolithic ones.</>,
          <><strong className="text-text-secondary">Use clear headings</strong> -- Markdown headings help the chunking algorithm split documents at logical boundaries.</>,
          <><strong className="text-text-secondary">Update regularly</strong> -- keep knowledge bases current. Outdated information leads to incorrect agent responses.</>,
          <><strong className="text-text-secondary">Tell the agent to cite sources</strong> -- add instructions in the system prompt to reference which document the answer came from.</>,
          <><strong className="text-text-secondary">Set honest boundaries</strong> -- instruct the agent to say &quot;I don&apos;t know&quot; rather than hallucinate when the knowledge base does not contain the answer.</>,
        ]} />
      </SubSection>

      <SectionDivider />
      <WhatNext items={[
        { label: 'Agents & Lifecycle', id: 'concept-agents' },
        { label: 'Tools', id: 'concept-tools' },
        { label: 'Example: Support Agent', id: 'example-support' },
      ]} />
    </div>
  );
}

function ConceptTriggersContent() {
  return (
    <div>
      <PageTitle>Triggers (Cron, Webhooks)</PageTitle>
      <p className="text-sm text-text-tertiary mb-4">
        Triggers enable agents to operate autonomously without waiting for user messages.
        They are the foundation of proactive AI workflows -- agents that monitor, report,
        and react to events on their own.
      </p>

      <SubSection title="Cron triggers">
        <p className="text-sm text-text-tertiary mb-3">
          Schedule agents to run at specific times using standard 5-field cron expressions.
          At the scheduled time, the engine creates a background task with the configured message
          and assigns it to the specified agent.
        </p>
        <CodeBlock>{`triggers:
  # Run every weekday morning
  daily-digest:
    cron: "0 9 * * 1-5"
    agent: reporter
    message: "Compile the daily digest from all data sources."

  # Run every 10 minutes
  health-check:
    cron: "*/10 * * * *"
    agent: monitor
    message: "Check all monitored services and report any issues."`}</CodeBlock>

        <h4 className="text-sm font-semibold text-text-primary mt-4 mb-2">Common patterns</h4>
        <ParamTable params={[
          { name: '*/5 * * * *', required: false, default: '--', desc: 'Every 5 minutes -- health checks, monitoring' },
          { name: '0 */2 * * *', required: false, default: '--', desc: 'Every 2 hours -- periodic data sync' },
          { name: '0 9 * * 1-5', required: false, default: '--', desc: 'Weekdays at 9 AM -- daily reports' },
          { name: '0 0 * * 0', required: false, default: '--', desc: 'Sundays at midnight -- weekly summaries' },
          { name: '0 0 1 * *', required: false, default: '--', desc: 'Monthly -- billing reports, audits' },
        ]} />
      </SubSection>

      <SubSection title="Webhook triggers">
        <p className="text-sm text-text-tertiary mb-3">
          Expose HTTP endpoints that external services can call to activate agents. The webhook
          request body is forwarded to the agent as the task message, giving it full context
          about the event.
        </p>
        <CodeBlock>{`triggers:
  # Stripe payment events
  stripe-payment:
    type: webhook
    path: /webhooks/stripe
    agent: billing-agent
    secret: \${STRIPE_WEBHOOK_SECRET}

  # GitHub PR events
  github-pr:
    type: webhook
    path: /webhooks/github
    agent: code-reviewer
    secret: \${GITHUB_WEBHOOK_SECRET}`}</CodeBlock>

        <h4 className="text-sm font-semibold text-text-primary mt-4 mb-2">Calling a webhook</h4>
        <CodeBlock>{`# External service sends a POST request:
curl -X POST http://localhost:8080/api/v1/webhooks/stripe \\
  -H "X-Webhook-Secret: whsec_your_secret" \\
  -H "Content-Type: application/json" \\
  -d '{
    "type": "payment_intent.succeeded",
    "data": {
      "customer_id": "cus_123",
      "amount": 9900,
      "currency": "usd"
    }
  }'

# The agent receives the full JSON body as its task message
# and can act on it (update records, send notifications, etc.)`}</CodeBlock>
      </SubSection>

      <SubSection title="Use cases">
        <BulletList items={[
          <><strong className="text-text-secondary">Daily reports</strong> -- cron trigger at 9 AM generates and distributes a summary.</>,
          <><strong className="text-text-secondary">Alert handling</strong> -- PagerDuty/Datadog webhook triggers an agent to analyze and triage alerts.</>,
          <><strong className="text-text-secondary">Order processing</strong> -- e-commerce webhook triggers an agent when a new order is placed.</>,
          <><strong className="text-text-secondary">CI/CD notifications</strong> -- GitHub webhook triggers a code review agent on new pull requests.</>,
          <><strong className="text-text-secondary">Periodic health checks</strong> -- cron trigger every 5 minutes monitors service endpoints.</>,
        ]} />
      </SubSection>

      <SectionDivider />
      <WhatNext items={[
        { label: 'Admin: Triggers', id: 'admin-triggers' },
        { label: 'Tasks & Jobs', id: 'concept-tasks' },
        { label: 'Example: DevOps Monitor', id: 'example-devops' },
      ]} />
    </div>
  );
}

/* ------------------------------------------------------------------ */
/*  Examples content sections                                          */
/* ------------------------------------------------------------------ */

function ExampleSalesContent() {
  return (
    <div>
      <PageTitle>Example: Sales Agent</PageTitle>
      <p className="text-sm text-text-tertiary mb-4">
        A multi-agent sales team with a supervisor that coordinates product search,
        inventory checks, order creation, and customer support. This example demonstrates
        agent spawning, custom HTTP tools, MCP integration, and cron triggers.
      </p>

      <SubSection title="What this demonstrates">
        <BulletList items={[
          <><strong className="text-text-secondary">Multi-agent orchestration</strong> -- a supervisor delegates to specialized sales and support agents.</>,
          <><strong className="text-text-secondary">Custom HTTP tools</strong> -- product search, inventory check, and order creation via REST APIs.</>,
          <><strong className="text-text-secondary">MCP integration</strong> -- CRM data access via an MCP server.</>,
          <><strong className="text-text-secondary">Cron trigger</strong> -- automatic morning lead review on weekdays.</>,
          <><strong className="text-text-secondary">Mixed models</strong> -- powerful model for the supervisor, cheaper models for specialists.</>,
        ]} />
      </SubSection>

      <SubSection title="Prerequisites">
        <BulletList items={[
          <>A running ByteBrew Engine instance.</>,
          <>API keys for your chosen LLM providers (OpenAI and/or Anthropic).</>,
          <>A product catalog API (or a mock endpoint for testing).</>,
          <>A CRM API key for the MCP CRM server (optional).</>,
        ]} />
      </SubSection>

      <SubSection title="Full configuration">
        <CodeBlock>{SALES_AGENT_YAML}</CodeBlock>
      </SubSection>

      <SubSection title="How to test">
        <CodeBlock>{`# Start a conversation with the supervisor
curl -N http://localhost:8080/api/v1/agents/sales-supervisor/chat \\
  -H "Content-Type: application/json" \\
  -d '{"message": "I need a laptop for video editing under $1500"}'

# The supervisor will:
# 1. Analyze the request
# 2. Spawn the sales-agent to search products and check inventory
# 3. Return recommendations based on the results

# Follow up in the same session:
curl -N http://localhost:8080/api/v1/agents/sales-supervisor/chat \\
  -H "Content-Type: application/json" \\
  -d '{"message": "I will take the second option", "session_id": "<id-from-previous>"}'`}</CodeBlock>
      </SubSection>

      <SubSection title="Customization tips">
        <BulletList items={[
          <>Replace <Ic>{'${CATALOG_API}'}</Ic> and <Ic>{'${ORDER_API}'}</Ic> with your actual API endpoints.</>,
          <>Add a <Ic>knowledge</Ic> folder with product documentation for the support agent.</>,
          <>Add a <Ic>apply_discount</Ic> tool with <Ic>confirmation_required: true</Ic> for price overrides.</>,
          <>Adjust <Ic>max_steps</Ic> on the supervisor if complex multi-step orders time out.</>,
        ]} />
      </SubSection>

      <SectionDivider />
      <WhatNext items={[
        { label: 'Multi-Agent Orchestration', id: 'concept-multi-agent' },
        { label: 'Support Agent Example', id: 'example-support' },
        { label: 'Configuration Reference', id: 'configuration' },
      ]} />
    </div>
  );
}

function ExampleSupportContent() {
  return (
    <div>
      <PageTitle>Example: Support Agent</PageTitle>
      <p className="text-sm text-text-tertiary mb-4">
        A knowledge-based customer support agent that answers questions from a documentation
        knowledge base, looks up order status, and creates support tickets for unresolved issues.
        This example demonstrates RAG, confirmation-required tools, and escalation patterns.
      </p>

      <SubSection title="What this demonstrates">
        <BulletList items={[
          <><strong className="text-text-secondary">Knowledge base (RAG)</strong> -- the agent searches your support docs before answering.</>,
          <><strong className="text-text-secondary">Ticket creation with confirmation</strong> -- the agent asks before creating a support ticket.</>,
          <><strong className="text-text-secondary">Order status lookup</strong> -- real-time order tracking via HTTP tool.</>,
          <><strong className="text-text-secondary">Escalation behavior</strong> -- the system prompt instructs the agent when to hand off to humans.</>,
        ]} />
      </SubSection>

      <SubSection title="Prerequisites">
        <BulletList items={[
          <>A <Ic>./docs/support/</Ic> folder with your knowledge base documents (Markdown, TXT, PDF).</>,
          <>A helpdesk API for ticket creation (or a mock endpoint).</>,
          <>An order tracking API (or a mock endpoint).</>,
        ]} />
      </SubSection>

      <SubSection title="Full configuration">
        <CodeBlock>{SUPPORT_AGENT_YAML}</CodeBlock>
      </SubSection>

      <SubSection title="How to test">
        <CodeBlock>{`# Ask a question that should be answered from the knowledge base
curl -N http://localhost:8080/api/v1/agents/support-bot/chat \\
  -H "Content-Type: application/json" \\
  -d '{"message": "What is your return policy for electronics?"}'

# Ask about an order
curl -N http://localhost:8080/api/v1/agents/support-bot/chat \\
  -H "Content-Type: application/json" \\
  -d '{"message": "Where is my order #12345?"}'

# Trigger ticket creation (agent will ask for confirmation)
curl -N http://localhost:8080/api/v1/agents/support-bot/chat \\
  -H "Content-Type: application/json" \\
  -d '{"message": "My laptop screen is cracked, I need a replacement"}'`}</CodeBlock>
      </SubSection>

      <SubSection title="Customization tips">
        <BulletList items={[
          <>Populate <Ic>./docs/support/</Ic> with your actual FAQ, return policy, and product guides.</>,
          <>Add more tools for common support actions: refund processing, account lookup, shipping updates.</>,
          <>Adjust the system prompt to match your brand voice and escalation thresholds.</>,
          <>For high-volume support, set <Ic>max_context_size</Ic> to 8000 to reduce costs per conversation.</>,
        ]} />
      </SubSection>

      <SectionDivider />
      <WhatNext items={[
        { label: 'Knowledge / RAG', id: 'concept-rag' },
        { label: 'Sales Agent Example', id: 'example-sales' },
        { label: 'DevOps Monitor Example', id: 'example-devops' },
      ]} />
    </div>
  );
}

function ExampleDevopsContent() {
  return (
    <div>
      <PageTitle>Example: DevOps Monitor</PageTitle>
      <p className="text-sm text-text-tertiary mb-4">
        An alert handling agent that monitors infrastructure health, triages incoming PagerDuty
        alerts, and performs automated remediation. This example demonstrates webhook triggers,
        cron scheduling, and task management for operational workflows.
      </p>

      <SubSection title="What this demonstrates">
        <BulletList items={[
          <><strong className="text-text-secondary">Webhook trigger</strong> -- PagerDuty alerts are forwarded to the agent in real-time.</>,
          <><strong className="text-text-secondary">Cron trigger</strong> -- periodic health checks every 5 minutes.</>,
          <><strong className="text-text-secondary">Task management</strong> -- the agent tracks open incidents as tasks.</>,
          <><strong className="text-text-secondary">Escalation</strong> -- critical issues are flagged for human attention.</>,
        ]} />
      </SubSection>

      <SubSection title="Prerequisites">
        <BulletList items={[
          <>PagerDuty (or similar alerting platform) configured to send webhooks.</>,
          <>Service health check endpoints accessible from the engine.</>,
        ]} />
      </SubSection>

      <SubSection title="Full configuration">
        <CodeBlock>{DEVOPS_MONITOR_YAML}</CodeBlock>
      </SubSection>

      <SubSection title="How to test">
        <CodeBlock>{`# Simulate a PagerDuty alert
curl -X POST http://localhost:8080/api/v1/webhooks/pagerduty \\
  -H "X-Webhook-Secret: your-pagerduty-secret" \\
  -H "Content-Type: application/json" \\
  -d '{
    "event": {
      "type": "incident.triggered",
      "data": {
        "title": "High CPU usage on web-server-03",
        "severity": "critical",
        "service": "web-cluster"
      }
    }
  }'

# The agent will analyze the alert, determine severity,
# and either handle it automatically or escalate to a human.

# Check what the health-check cron has found:
curl http://localhost:8080/api/v1/tasks?agent=alert-handler&status=completed \\
  -H "Authorization: Bearer bb_your_token"`}</CodeBlock>
      </SubSection>

      <SubSection title="Customization tips">
        <BulletList items={[
          <>Add custom HTTP tools for automated remediation (restart services, scale infrastructure, clear caches).</>,
          <>Add a <Ic>knowledge</Ic> folder with runbooks so the agent can follow documented procedures.</>,
          <>Create a <Ic>can_spawn</Ic> relationship with a log-analyzer agent for deep-dive investigations.</>,
          <>Set <Ic>confirmation_required: true</Ic> on any tool that modifies production infrastructure.</>,
        ]} />
      </SubSection>

      <SectionDivider />
      <WhatNext items={[
        { label: 'Triggers', id: 'concept-triggers' },
        { label: 'Tasks & Jobs', id: 'concept-tasks' },
        { label: 'IoT Analyzer Example', id: 'example-iot' },
      ]} />
    </div>
  );
}

function ExampleIotContent() {
  return (
    <div>
      <PageTitle>Example: IoT Analyzer</PageTitle>
      <p className="text-sm text-text-tertiary mb-4">
        A telemetry monitoring system with a supervisor agent that coordinates anomaly detection
        across IoT sensors. This example demonstrates multi-agent orchestration, time-series
        data queries, Slack alerting with confirmation, and periodic cron triggers.
      </p>

      <SubSection title="What this demonstrates">
        <BulletList items={[
          <><strong className="text-text-secondary">Multi-agent spawning</strong> -- the supervisor delegates anomaly detection to a specialized sub-agent.</>,
          <><strong className="text-text-secondary">Time-series queries</strong> -- custom HTTP tool queries InfluxDB for sensor data.</>,
          <><strong className="text-text-secondary">Slack alerting with confirmation</strong> -- the agent asks before sending alerts to avoid noise.</>,
          <><strong className="text-text-secondary">Cron trigger</strong> -- automatic analysis every 10 minutes.</>,
        ]} />
      </SubSection>

      <SubSection title="Prerequisites">
        <BulletList items={[
          <>InfluxDB or similar time-series database with IoT sensor data.</>,
          <>Slack incoming webhook URL for alert delivery.</>,
          <>IoT sensors writing telemetry data to the time-series database.</>,
        ]} />
      </SubSection>

      <SubSection title="Full configuration">
        <CodeBlock>{IOT_ANALYZER_YAML}</CodeBlock>
      </SubSection>

      <SubSection title="How to test">
        <CodeBlock>{`# Manually trigger an analysis
curl -N http://localhost:8080/api/v1/agents/iot-supervisor/chat \\
  -H "Content-Type: application/json" \\
  -d '{"message": "Analyze temperature sensors in Building A for the last hour"}'

# The supervisor will:
# 1. Spawn the anomaly-detector with the specific query
# 2. The detector queries InfluxDB for the data window
# 3. Analyzes the data for statistical anomalies
# 4. Returns a structured report to the supervisor
# 5. Supervisor decides whether to alert operators (with confirmation)

# Check recent task history (cron creates these automatically):
curl http://localhost:8080/api/v1/tasks?agent=iot-supervisor \\
  -H "Authorization: Bearer bb_your_token"`}</CodeBlock>
      </SubSection>

      <SubSection title="Customization tips">
        <BulletList items={[
          <>Adjust the cron frequency based on how critical real-time monitoring is (1 min vs 10 min).</>,
          <>Add more sensor types and corresponding Flux queries for the anomaly detector.</>,
          <>Create a <Ic>knowledge</Ic> folder with equipment manuals so the agent can suggest specific remediation steps.</>,
          <>Replace the Slack webhook with PagerDuty or email for different alerting channels.</>,
          <>Add a <Ic>rule_engine</Ic> tool that lets the agent create automated threshold rules based on patterns it discovers.</>,
        ]} />
      </SubSection>

      <SectionDivider />
      <WhatNext items={[
        { label: 'Multi-Agent Orchestration', id: 'concept-multi-agent' },
        { label: 'DevOps Monitor Example', id: 'example-devops' },
        { label: 'Configuration Reference', id: 'configuration' },
      ]} />
    </div>
  );
}

/* ------------------------------------------------------------------ */
/*  YAML constants for examples                                        */
/* ------------------------------------------------------------------ */

const SALES_AGENT_YAML = `# Sales Agent — Full Configuration Example
agents:
  sales-supervisor:
    model: glm-5
    system: |
      You are a sales team supervisor. Route incoming customer
      queries to the appropriate sales or support agent.
      Prioritize high-intent buyers.
    can_spawn:
      - sales-agent
      - support-agent
    tools:
      - customer_lookup

  sales-agent:
    model: qwen-3-32b
    lifecycle: spawn
    system: |
      You are a sales consultant. Interview the buyer to
      understand their needs, recommend products, check
      inventory, and create orders when ready.
    tools:
      - product_search
      - check_inventory
      - create_order
      - apply_discount
    mcp_servers:
      - crm-api

  support-agent:
    model: claude-sonnet-4
    lifecycle: spawn
    system: |
      You are a customer support agent. Answer product
      questions using the knowledge base, create tickets
      for issues you cannot resolve.
    tools:
      - knowledge_search
      - create_ticket
      - order_status

tools:
  product_search:
    type: http
    method: GET
    url: "\${CATALOG_API}/products/search"
    params:
      query: "{{input}}"

  check_inventory:
    type: http
    method: GET
    url: "\${CATALOG_API}/inventory/{{product_id}}"

  create_order:
    type: http
    method: POST
    url: "\${ORDER_API}/orders"
    body:
      customer_id: "{{customer_id}}"
      items: "{{items}}"

mcp_servers:
  crm-api:
    command: npx
    args: ["-y", "@bytebrew/mcp-crm"]
    env:
      CRM_API_KEY: "\${CRM_API_KEY}"

models:
  glm-5:
    provider: openai
    api_key: "\${OPENAI_API_KEY}"
  qwen-3-32b:
    provider: openai
    api_key: "\${OPENAI_API_KEY}"
  claude-sonnet-4:
    provider: anthropic
    api_key: "\${ANTHROPIC_API_KEY}"

triggers:
  morning-leads:
    cron: "0 9 * * 1-5"
    agent: sales-supervisor
    message: "Check for new leads from overnight and prioritize follow-ups."`;

const SUPPORT_AGENT_YAML = `# Support Agent — Knowledge-based with ticket creation
agents:
  support-bot:
    model: glm-5
    system: |
      You are a customer support agent. Use the knowledge base
      to answer questions. Create tickets for unresolved issues.
      Escalate urgent matters to human support.
    tools:
      - knowledge_search
      - create_ticket
      - order_status
    knowledge: "./docs/support/"

tools:
  create_ticket:
    type: http
    method: POST
    url: "\${HELPDESK_API}/tickets"
    body:
      subject: "{{subject}}"
      description: "{{description}}"
      priority: "{{priority}}"
    confirmation_required: true

  order_status:
    type: http
    method: GET
    url: "\${ORDER_API}/orders/{{order_id}}/status"

models:
  glm-5:
    provider: openai
    api_key: "\${OPENAI_API_KEY}"`;

const DEVOPS_MONITOR_YAML = `# DevOps Monitor — Alert handling with webhooks and cron
agents:
  alert-handler:
    model: glm-5
    system: |
      You are a DevOps alert handler. Analyze incoming alerts,
      filter noise, identify real issues, suggest remediation.
      Escalate P1 incidents immediately.
    tools:
      - web_search
      - manage_tasks

triggers:
  pagerduty-webhook:
    type: webhook
    path: /webhooks/pagerduty
    agent: alert-handler
    secret: \${PAGERDUTY_SECRET}

  health-check:
    cron: "*/5 * * * *"
    agent: alert-handler
    message: "Check service health status for all monitored endpoints."

models:
  glm-5:
    provider: openai
    api_key: "\${OPENAI_API_KEY}"`;

const IOT_ANALYZER_YAML = `# IoT Analyzer — Telemetry monitoring with anomaly detection
agents:
  iot-supervisor:
    model: glm-5
    system: |
      You monitor IoT device telemetry streams. Detect anomalies,
      correlate events across sensors, and suggest automation rules.
      Alert operators when thresholds are breached.
    can_spawn:
      - anomaly-detector
    tools:
      - manage_tasks
      - send_alert

  anomaly-detector:
    model: qwen-3-32b
    lifecycle: spawn
    system: |
      Analyze the provided sensor data window. Identify statistical
      anomalies, trend deviations, and potential equipment failures.
      Return a structured report with severity levels.
    tools:
      - query_timeseries

tools:
  query_timeseries:
    type: http
    method: POST
    url: "\${INFLUX_API}/query"
    body:
      query: "{{flux_query}}"
    auth:
      type: bearer
      token: \${INFLUX_TOKEN}

  send_alert:
    type: http
    method: POST
    url: "\${SLACK_WEBHOOK_URL}"
    body:
      text: "{{message}}"
    confirmation_required: true

triggers:
  telemetry-check:
    cron: "*/10 * * * *"
    agent: iot-supervisor
    message: "Analyze the last 10 minutes of telemetry data for anomalies."

models:
  glm-5:
    provider: openai
    api_key: "\${OPENAI_API_KEY}"
  qwen-3-32b:
    provider: openai
    api_key: "\${OPENAI_API_KEY}"`;

/* ------------------------------------------------------------------ */
/*  Shared components                                                  */
/* ------------------------------------------------------------------ */

function PageTitle({ children }: { children: React.ReactNode }) {
  return (
    <h1 className="text-2xl font-bold text-text-primary mb-2">
      {children}
    </h1>
  );
}

function SubSection({ title, children }: { title: string; children: React.ReactNode }) {
  return (
    <div className="mb-8">
      <h3 className="text-base font-semibold text-text-primary mb-3 pb-2 border-b border-border">
        {title}
      </h3>
      {children}
    </div>
  );
}

function Callout({ type = 'info', title, children }: { type?: 'info' | 'warning' | 'tip'; title?: string; children: React.ReactNode }) {
  const styles = {
    info: 'border-blue-500/30 bg-blue-500/5 text-blue-400',
    warning: 'border-yellow-500/30 bg-yellow-500/5 text-yellow-400',
    tip: 'border-emerald-500/30 bg-emerald-500/5 text-emerald-400',
  };
  const icons = { info: '\u2139\uFE0F', warning: '\u26A0\uFE0F', tip: '\uD83D\uDCA1' };
  return (
    <div className={`my-4 rounded-[2px] border px-4 py-3 text-sm ${styles[type]}`}>
      {title && <div className="font-semibold mb-1">{icons[type]} {title}</div>}
      <div className="text-text-secondary">{children}</div>
    </div>
  );
}

/** Inline code span */
function Ic({ children }: { children: React.ReactNode }) {
  return (
    <code className="text-brand-accent text-xs bg-brand-accent/10 px-1.5 py-0.5 rounded font-mono">
      {children}
    </code>
  );
}

function BulletList({ items }: { items: React.ReactNode[] }) {
  return (
    <ul className="mb-4 space-y-1.5">
      {items.map((item, i) => (
        <li key={i} className="flex items-start gap-2 text-sm text-text-tertiary">
          <span className="mt-1.5 h-1.5 w-1.5 rounded-full bg-brand-accent/50 shrink-0" />
          <span>{item}</span>
        </li>
      ))}
    </ul>
  );
}

function QuickStartStep({
  n,
  title,
  children,
}: {
  n: number;
  title: string;
  children: React.ReactNode;
}) {
  return (
    <div className="mb-8 last:mb-0">
      <div className="flex items-center gap-3 mb-3">
        <span className="inline-flex items-center justify-center w-6 h-6 rounded-full bg-brand-accent/15 text-brand-accent text-xs font-bold shrink-0">
          {n}
        </span>
        <h4 className="font-semibold text-text-primary text-sm">{title}</h4>
      </div>
      <div className="ml-9">{children}</div>
    </div>
  );
}

function CodeBlock({ children }: { children: string }) {
  const [copied, setCopied] = useState(false);

  const handleCopy = () => {
    navigator.clipboard.writeText(children);
    setCopied(true);
    setTimeout(() => setCopied(false), 2000);
  };

  return (
    <div className="relative rounded-[2px] border border-border bg-surface p-4 overflow-x-auto">
      <pre className="font-mono text-sm text-text-secondary leading-relaxed whitespace-pre">
        {children}
      </pre>
      <button
        onClick={handleCopy}
        className="absolute top-3 right-3 rounded-[2px] border border-border px-2.5 py-1 text-xs text-text-tertiary hover:text-text-primary hover:border-border-hover transition-colors"
      >
        {copied ? 'Copied!' : 'Copy'}
      </button>
    </div>
  );
}

/** Parameter reference table */
function ParamTable({ params }: { params: { name: string; required?: boolean; default?: string; desc: string }[] }) {
  return (
    <div className="mb-4 overflow-x-auto">
      <table className="w-full text-sm border-collapse">
        <thead>
          <tr className="border-b border-border">
            <th className="text-left py-2 pr-3 text-text-tertiary font-medium text-xs uppercase tracking-wider">Parameter</th>
            <th className="text-left py-2 pr-3 text-text-tertiary font-medium text-xs uppercase tracking-wider">Default</th>
            <th className="text-left py-2 text-text-tertiary font-medium text-xs uppercase tracking-wider">Description</th>
          </tr>
        </thead>
        <tbody>
          {params.map((p, i) => (
            <tr key={i} className="border-b border-brand-shade3/8">
              <td className="py-2 pr-3 align-top whitespace-nowrap">
                <code className="text-brand-accent text-xs bg-brand-accent/10 px-1.5 py-0.5 rounded font-mono">{p.name}</code>
                {p.required && <span className="ml-1.5 text-red-400 text-xs">*</span>}
              </td>
              <td className="py-2 pr-3 align-top text-text-tertiary whitespace-nowrap text-xs">
                {p.default || '--'}
              </td>
              <td className="py-2 align-top text-text-tertiary">{p.desc}</td>
            </tr>
          ))}
        </tbody>
      </table>
    </div>
  );
}

/** Section divider line */
function SectionDivider() {
  return <hr className="my-8 border-border" />;
}

/** "What's next" links */
function WhatNext({ items }: { items: { label: string; id: string }[] }) {
  return (
    <div className="mt-2">
      <h4 className="text-sm font-semibold text-text-primary mb-2">What&apos;s next</h4>
      <ul className="space-y-1">
        {items.map((item, i) => (
          <li key={i} className="text-sm text-brand-accent hover:text-brand-accent/80 cursor-pointer">
            &rarr; {item.label}
          </li>
        ))}
      </ul>
    </div>
  );
}
