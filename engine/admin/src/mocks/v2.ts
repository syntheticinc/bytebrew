// V2 Mock Data for agent-first runtime prototype.
// Concept reference: docs/architecture/agent-first-runtime.md
// Mode guard: all consumption is behind isPrototype.

export type TriggerType = 'cron' | 'webhook' | 'chat';

export interface V2Trigger {
  id: string;
  type: TriggerType;
  title: string;
  agentId: string;
  schemaId: string;
  enabled: boolean;
  config: Record<string, unknown>;
  lastFiredAt?: string;
}

export interface V2Widget {
  id: string;
  name: string;
  triggerId: string; // → V2Trigger (must be type=chat)
  primaryColor: string;
  position: 'bottom-right' | 'bottom-left' | 'top-right' | 'top-left';
  size: 'compact' | 'standard' | 'large';
  welcomeMessage: string;
  placeholder: string;
  avatarUrl?: string;
  domainWhitelist: string[];
  enabled: boolean;
}

export interface V2Agent {
  id: string;
  name: string;
  model: string;
  description?: string;
  avatarInitials: string; // fallback avatar: 2 letters
  lifecycle: 'persistent' | 'spawn';
  toolsCount: number;
  knowledgeCount: number;
  flowsCount: number;
  activeSessions: number;
  state: 'idle' | 'active' | 'degraded';
}

export interface V2AgentRelation {
  id: string;
  sourceAgentId: string;
  targetAgentId: string;
  config?: Record<string, unknown>;
}

export interface V2Schema {
  id: string;
  name: string;
  description: string;
  entryAgentId: string;
  agentIds: string[];
  triggerIds: string[];
  sessionsToday: number;
  activeSessions: number;
  lastActivityAt: string;
  updatedAt: string;
}

export interface V2SessionMessage {
  step: number;
  agentId: string;
  kind: 'user_message' | 'assistant_message' | 'tool_call' | 'tool_result' | 'reasoning' | 'delegation' | 'delegation_return';
  content: string;
  toolName?: string;
  toolArgs?: string;
  toolResult?: string;
  targetAgentId?: string;
  sourceAgentId?: string;
  timestamp: string;
}

export interface V2Session {
  id: string;
  schemaId: string;
  triggerId: string;
  title: string;
  status: 'active' | 'completed' | 'failed';
  startedAt: string;
  participantAgentIds: string[];
  messages: V2SessionMessage[];
}

export interface V2FlowCheckpoint {
  id: string;
  name: string;
  goal: string;
  successCriteria: string;
  config?: Record<string, unknown>;
}

export interface V2Flow {
  id: string;
  agentId: string;
  name: string;
  description: string;
  triggerCondition: string;
  enabled: boolean;
  checkpoints: V2FlowCheckpoint[];
}

export interface V2OverviewEvent {
  timestamp: string;
  kind: 'trigger_fired' | 'delegation' | 'session_completed' | 'agent_error' | 'flow_entered';
  summary: string;
  schemaId?: string;
  sessionId?: string;
}

// ============================================================================
// AGENTS (cross-schema library)
// ============================================================================

export const v2Agents: V2Agent[] = [
  {
    id: 'agent-triage',
    name: 'Triage Orchestrator',
    model: 'claude-haiku-4-5',
    description: 'Classifies incoming requests and delegates to specialists.',
    avatarInitials: 'TR',
    lifecycle: 'persistent',
    toolsCount: 3,
    knowledgeCount: 1,
    flowsCount: 0,
    activeSessions: 3,
    state: 'active',
  },
  {
    id: 'agent-sales',
    name: 'Sales Specialist',
    model: 'claude-sonnet-4-6',
    description: 'Handles pricing, plans, conversion.',
    avatarInitials: 'SL',
    lifecycle: 'persistent',
    toolsCount: 6,
    knowledgeCount: 2,
    flowsCount: 1,
    activeSessions: 1,
    state: 'active',
  },
  {
    id: 'agent-tech',
    name: 'Tech Support',
    model: 'claude-sonnet-4-6',
    description: 'Debugging and technical assistance.',
    avatarInitials: 'TS',
    lifecycle: 'persistent',
    toolsCount: 8,
    knowledgeCount: 3,
    flowsCount: 1,
    activeSessions: 0,
    state: 'idle',
  },
  {
    id: 'agent-billing',
    name: 'Billing Agent',
    model: 'claude-haiku-4-5',
    description: 'Invoices, refunds, payment issues.',
    avatarInitials: 'BL',
    lifecycle: 'persistent',
    toolsCount: 4,
    knowledgeCount: 1,
    flowsCount: 0,
    activeSessions: 0,
    state: 'idle',
  },
  {
    id: 'agent-faq',
    name: 'FAQ Lookup',
    model: 'claude-haiku-4-5',
    description: 'Fast FAQ retrieval from knowledge base.',
    avatarInitials: 'FQ',
    lifecycle: 'spawn',
    toolsCount: 2,
    knowledgeCount: 4,
    flowsCount: 0,
    activeSessions: 0,
    state: 'idle',
  },
  {
    id: 'agent-escalation',
    name: 'Human Escalation',
    model: 'claude-sonnet-4-6',
    description: 'Routes to human operators via webhook.',
    avatarInitials: 'HE',
    lifecycle: 'spawn',
    toolsCount: 1,
    knowledgeCount: 0,
    flowsCount: 0,
    activeSessions: 0,
    state: 'idle',
  },
  {
    id: 'agent-sales-orch',
    name: 'Sales Qualification Orch',
    model: 'claude-opus-4-6',
    description: 'Qualifies leads via deep interview flow.',
    avatarInitials: 'SQ',
    lifecycle: 'persistent',
    toolsCount: 4,
    knowledgeCount: 1,
    flowsCount: 1,
    activeSessions: 1,
    state: 'active',
  },
  {
    id: 'agent-lead-researcher',
    name: 'Lead Researcher',
    model: 'claude-sonnet-4-6',
    description: 'Gathers public info about prospect.',
    avatarInitials: 'LR',
    lifecycle: 'persistent',
    toolsCount: 5,
    knowledgeCount: 0,
    flowsCount: 0,
    activeSessions: 0,
    state: 'idle',
  },
  {
    id: 'agent-closer',
    name: 'Closer',
    model: 'claude-opus-4-6',
    description: 'Final pitch and handoff to human AE.',
    avatarInitials: 'CL',
    lifecycle: 'persistent',
    toolsCount: 3,
    knowledgeCount: 1,
    flowsCount: 0,
    activeSessions: 0,
    state: 'idle',
  },
  {
    id: 'agent-health',
    name: 'Health Monitor',
    model: 'claude-haiku-4-5',
    description: 'Hourly system checks and alerting.',
    avatarInitials: 'HM',
    lifecycle: 'persistent',
    toolsCount: 5,
    knowledgeCount: 0,
    flowsCount: 0,
    activeSessions: 0,
    state: 'idle',
  },
  {
    id: 'agent-alerter',
    name: 'Alerter',
    model: 'claude-haiku-4-5',
    description: 'Dispatches alerts to Slack/PagerDuty.',
    avatarInitials: 'AL',
    lifecycle: 'spawn',
    toolsCount: 2,
    knowledgeCount: 0,
    flowsCount: 0,
    activeSessions: 0,
    state: 'idle',
  },
];

// ============================================================================
// RELATIONS (only delegation, source → target)
// ============================================================================

export const v2AgentRelations: V2AgentRelation[] = [
  // Support Schema
  { id: 'rel-1', sourceAgentId: 'agent-triage', targetAgentId: 'agent-sales' },
  { id: 'rel-2', sourceAgentId: 'agent-triage', targetAgentId: 'agent-tech' },
  { id: 'rel-3', sourceAgentId: 'agent-triage', targetAgentId: 'agent-billing' },
  { id: 'rel-4', sourceAgentId: 'agent-sales', targetAgentId: 'agent-faq' },
  { id: 'rel-5', sourceAgentId: 'agent-billing', targetAgentId: 'agent-escalation' },
  // Sales Schema
  { id: 'rel-6', sourceAgentId: 'agent-sales-orch', targetAgentId: 'agent-lead-researcher' },
  { id: 'rel-7', sourceAgentId: 'agent-sales-orch', targetAgentId: 'agent-closer' },
  // Health Schema
  { id: 'rel-8', sourceAgentId: 'agent-health', targetAgentId: 'agent-alerter' },
];

// ============================================================================
// SCHEMAS
// ============================================================================

export const v2Schemas: V2Schema[] = [
  {
    id: 'schema-support',
    name: 'Customer Support',
    description: 'Multi-channel customer support with triage and specialist delegation.',
    entryAgentId: 'agent-triage',
    agentIds: ['agent-triage', 'agent-sales', 'agent-tech', 'agent-billing', 'agent-faq', 'agent-escalation'],
    triggerIds: ['trg-support-chat-main', 'trg-support-webhook'],
    sessionsToday: 142,
    activeSessions: 3,
    lastActivityAt: '2026-04-15T12:35:00Z',
    updatedAt: '2026-04-14T09:15:00Z',
  },
  {
    id: 'schema-sales',
    name: 'Sales Qualification',
    description: 'Lead qualification flow with research and closing stages.',
    entryAgentId: 'agent-sales-orch',
    agentIds: ['agent-sales-orch', 'agent-lead-researcher', 'agent-closer'],
    triggerIds: ['trg-sales-webhook'],
    sessionsToday: 28,
    activeSessions: 1,
    lastActivityAt: '2026-04-15T12:30:00Z',
    updatedAt: '2026-04-13T14:00:00Z',
  },
  {
    id: 'schema-health',
    name: 'Daily Health Report',
    description: 'Hourly system health checks with automated alerting.',
    entryAgentId: 'agent-health',
    agentIds: ['agent-health', 'agent-alerter'],
    triggerIds: ['trg-health-cron'],
    sessionsToday: 24,
    activeSessions: 0,
    lastActivityAt: '2026-04-15T12:00:00Z',
    updatedAt: '2026-04-10T08:30:00Z',
  },
];

// ============================================================================
// TRIGGERS
// ============================================================================

export const v2Triggers: V2Trigger[] = [
  {
    id: 'trg-support-chat-main',
    type: 'chat',
    title: 'Main Chat',
    agentId: 'agent-triage',
    schemaId: 'schema-support',
    enabled: true,
    config: {},
    lastFiredAt: '2026-04-15T12:35:00Z',
  },
  {
    id: 'trg-support-webhook',
    type: 'webhook',
    title: 'Intake API',
    agentId: 'agent-triage',
    schemaId: 'schema-support',
    enabled: true,
    config: {
      webhookPath: '/api/v1/webhooks/support-intake',
    },
    lastFiredAt: '2026-04-15T12:28:00Z',
  },
  {
    id: 'trg-sales-webhook',
    type: 'webhook',
    title: 'Lead Form Submit',
    agentId: 'agent-sales-orch',
    schemaId: 'schema-sales',
    enabled: true,
    config: {
      webhookPath: '/api/v1/webhooks/lead-form',
    },
    lastFiredAt: '2026-04-15T12:30:00Z',
  },
  {
    id: 'trg-health-cron',
    type: 'cron',
    title: 'Hourly Health Check',
    agentId: 'agent-health',
    schemaId: 'schema-health',
    enabled: true,
    config: {
      schedule: '0 * * * *',
    },
    lastFiredAt: '2026-04-15T12:00:00Z',
  },
];

// ============================================================================
// WIDGETS (embeddable clients bound to chat-triggers)
// ============================================================================

export const v2Widgets: V2Widget[] = [
  {
    id: 'wgt-support-main',
    name: 'Support Widget (main site)',
    triggerId: 'trg-support-chat-main',
    primaryColor: '#6366f1',
    position: 'bottom-right',
    size: 'standard',
    welcomeMessage: 'Hi! How can I help?',
    placeholder: 'Type your question...',
    domainWhitelist: ['app.bytebrew.ai', 'bytebrew.ai'],
    enabled: true,
  },
  {
    id: 'wgt-support-docs',
    name: 'Docs Widget (compact)',
    triggerId: 'trg-support-chat-main',
    primaryColor: '#10b981',
    position: 'bottom-left',
    size: 'compact',
    welcomeMessage: 'Ask about our docs',
    placeholder: 'Search docs...',
    domainWhitelist: ['docs.bytebrew.ai'],
    enabled: true,
  },
];

// ============================================================================
// SCHEMA TEMPLATES (for empty-state / onboarding)
// ============================================================================

export interface V2SchemaTemplate {
  id: string;
  name: string;
  description: string;
  agentCount: number;
  triggerTypes: TriggerType[];
}

export const v2SchemaTemplates: V2SchemaTemplate[] = [
  {
    id: 'tpl-blank',
    name: 'Blank',
    description: 'Start from scratch. Create your entry orchestrator and build out from there.',
    agentCount: 0,
    triggerTypes: [],
  },
  {
    id: 'tpl-support',
    name: 'Customer Support',
    description: 'Triage orchestrator with delegation to Sales, Tech, Billing specialists.',
    agentCount: 5,
    triggerTypes: ['chat', 'webhook'],
  },
  {
    id: 'tpl-sales',
    name: 'Sales Qualification',
    description: 'Deep-interview flow: needs, authority, budget, technical fit.',
    agentCount: 3,
    triggerTypes: ['webhook'],
  },
  {
    id: 'tpl-research',
    name: 'Research Pipeline',
    description: 'Lead researcher + summarizer pattern.',
    agentCount: 2,
    triggerTypes: ['cron'],
  },
];

// ============================================================================
// SESSIONS (for debug mode)
// ============================================================================

export const v2Sessions: V2Session[] = [
  {
    id: 'sess-a7f2',
    schemaId: 'schema-support',
    triggerId: 'trg-support-chat-main',
    title: 'Customer asking about enterprise pricing',
    status: 'active',
    startedAt: '2026-04-15T12:34:05Z',
    participantAgentIds: ['agent-triage', 'agent-sales', 'agent-faq'],
    messages: [
      {
        step: 1,
        agentId: 'agent-triage',
        kind: 'user_message',
        content: 'Hi, I want to know about your enterprise pricing and SSO support.',
        timestamp: '2026-04-15T12:34:05Z',
      },
      {
        step: 2,
        agentId: 'agent-triage',
        kind: 'reasoning',
        content: 'This is a sales inquiry with a compliance component (SSO). I should delegate to Sales.',
        timestamp: '2026-04-15T12:34:08Z',
      },
      {
        step: 3,
        agentId: 'agent-triage',
        kind: 'delegation',
        content: 'Delegating to Sales Specialist: classify and respond about enterprise pricing + SSO.',
        targetAgentId: 'agent-sales',
        timestamp: '2026-04-15T12:34:10Z',
      },
      {
        step: 4,
        agentId: 'agent-sales',
        kind: 'reasoning',
        content: 'Need to check current pricing tier for SSO availability.',
        timestamp: '2026-04-15T12:34:12Z',
      },
      {
        step: 5,
        agentId: 'agent-sales',
        kind: 'delegation',
        content: 'Delegating to FAQ: "What plan includes SSO?"',
        targetAgentId: 'agent-faq',
        timestamp: '2026-04-15T12:34:13Z',
      },
      {
        step: 6,
        agentId: 'agent-faq',
        kind: 'tool_call',
        content: 'search_knowledge',
        toolName: 'search_knowledge',
        toolArgs: '{"query": "SSO plan availability"}',
        timestamp: '2026-04-15T12:34:14Z',
      },
      {
        step: 7,
        agentId: 'agent-faq',
        kind: 'tool_result',
        content: 'SSO is available on Enterprise plan ($499/mo) and Custom plans.',
        toolResult: '{"plan": "Enterprise", "price_usd": 499, "sso": true}',
        timestamp: '2026-04-15T12:34:16Z',
      },
      {
        step: 8,
        agentId: 'agent-faq',
        kind: 'delegation_return',
        content: 'Returning: SSO is available on Enterprise tier at $499/mo.',
        sourceAgentId: 'agent-sales',
        timestamp: '2026-04-15T12:34:17Z',
      },
      {
        step: 9,
        agentId: 'agent-sales',
        kind: 'delegation_return',
        content: 'Returning to Triage with pricing information package.',
        sourceAgentId: 'agent-triage',
        timestamp: '2026-04-15T12:34:19Z',
      },
      {
        step: 10,
        agentId: 'agent-triage',
        kind: 'assistant_message',
        content: 'Great question! Our Enterprise plan includes SSO and costs $499/month. It also covers advanced audit logging, SLA, and dedicated support. Want me to connect you with a rep?',
        timestamp: '2026-04-15T12:34:22Z',
      },
    ],
  },
  {
    id: 'sess-b3d1',
    schemaId: 'schema-support',
    triggerId: 'trg-support-chat-main',
    title: 'Billing dispute — refund request',
    status: 'active',
    startedAt: '2026-04-15T12:33:40Z',
    participantAgentIds: ['agent-triage', 'agent-billing'],
    messages: [
      {
        step: 1,
        agentId: 'agent-triage',
        kind: 'user_message',
        content: 'I was charged twice this month, need a refund.',
        timestamp: '2026-04-15T12:33:40Z',
      },
      {
        step: 2,
        agentId: 'agent-triage',
        kind: 'delegation',
        content: 'Delegating to Billing Agent.',
        targetAgentId: 'agent-billing',
        timestamp: '2026-04-15T12:33:42Z',
      },
      {
        step: 3,
        agentId: 'agent-billing',
        kind: 'tool_call',
        content: 'lookup_charges',
        toolName: 'lookup_charges',
        toolArgs: '{"user_id": "usr_123", "month": "2026-04"}',
        timestamp: '2026-04-15T12:33:45Z',
      },
      {
        step: 4,
        agentId: 'agent-billing',
        kind: 'tool_result',
        content: 'Found 2 charges: $49.00 on 2026-04-01, $49.00 on 2026-04-02 (duplicate).',
        toolResult: '{"charges": [{"amount": 49, "date": "2026-04-01"}, {"amount": 49, "date": "2026-04-02"}]}',
        timestamp: '2026-04-15T12:33:47Z',
      },
    ],
  },
  {
    id: 'sess-c9e4',
    schemaId: 'schema-sales',
    triggerId: 'trg-sales-webhook',
    title: 'Lead qualification: Acme Corp',
    status: 'active',
    startedAt: '2026-04-15T12:30:00Z',
    participantAgentIds: ['agent-sales-orch', 'agent-lead-researcher'],
    messages: [
      {
        step: 1,
        agentId: 'agent-sales-orch',
        kind: 'user_message',
        content: 'New lead: Acme Corp (500 employees, fintech).',
        timestamp: '2026-04-15T12:30:00Z',
      },
      {
        step: 2,
        agentId: 'agent-sales-orch',
        kind: 'delegation',
        content: 'Delegating research to Lead Researcher.',
        targetAgentId: 'agent-lead-researcher',
        timestamp: '2026-04-15T12:30:02Z',
      },
    ],
  },
];

// ============================================================================
// FLOWS (placeholder — V2+ feature)
// ============================================================================

export const v2Flows: V2Flow[] = [
  {
    id: 'flow-sales-qualification',
    agentId: 'agent-sales-orch',
    name: 'Deep Qualification Interview',
    description: 'Sequentially assesses fit, budget, authority, timeline.',
    triggerCondition: 'Auto-enter when new lead received via webhook.',
    enabled: true,
    checkpoints: [
      {
        id: 'cp-1',
        name: 'Needs Assessment',
        goal: 'Understand the customer problem and scale of impact.',
        successCriteria: 'Problem statement captured with quantified impact.',
      },
      {
        id: 'cp-2',
        name: 'Decision Authority',
        goal: 'Identify who makes the final purchase decision.',
        successCriteria: 'Decision-maker name and role confirmed.',
      },
      {
        id: 'cp-3',
        name: 'Budget & Timeline',
        goal: 'Determine budget range and expected implementation timeline.',
        successCriteria: 'Budget bracket + target go-live quarter confirmed.',
      },
      {
        id: 'cp-4',
        name: 'Technical Fit',
        goal: 'Verify technical compatibility (stack, integrations, compliance).',
        successCriteria: 'Technical requirements mapped to ByteBrew capabilities.',
      },
      {
        id: 'cp-5',
        name: 'Next Step Commitment',
        goal: 'Secure commitment to next concrete step (demo, POC, purchase).',
        successCriteria: 'Follow-up scheduled or explicit objection documented.',
      },
    ],
  },
  {
    id: 'flow-tech-debug',
    agentId: 'agent-tech',
    name: 'Structured Debug Session',
    description: 'OMC-style: assess uncertainty, isolate, verify, confirm.',
    triggerCondition: 'Manual entry by agent when confidence < 70%.',
    enabled: true,
    checkpoints: [
      {
        id: 'cp-1',
        name: 'Reproduce',
        goal: 'Get a reliable reproduction of the reported issue.',
        successCriteria: 'Steps documented that reliably trigger bug.',
      },
      {
        id: 'cp-2',
        name: 'Isolate',
        goal: 'Narrow the issue to a specific component or input.',
        successCriteria: 'Minimal failing case identified.',
      },
      {
        id: 'cp-3',
        name: 'Verify Hypothesis',
        goal: 'Test suspected root cause.',
        successCriteria: 'Evidence confirms or rejects each hypothesis.',
      },
      {
        id: 'cp-4',
        name: 'Propose Fix',
        goal: 'Recommend minimal intervention to resolve issue.',
        successCriteria: 'Fix proposal includes rollback path and tests.',
      },
    ],
  },
];

// ============================================================================
// OVERVIEW
// ============================================================================

export const v2OverviewStats = {
  activeSessions: 5,
  runsToday: 194,
  triggersFiredPerHour: 29,
  tokensToday: 1_243_000,
  avgLatencyMs: 2300,
  successRate: 0.96,
};

export const v2OverviewEvents: V2OverviewEvent[] = [
  {
    timestamp: '2026-04-15T12:34:22Z',
    kind: 'session_completed',
    summary: 'Triage finalized response for enterprise pricing inquiry.',
    schemaId: 'schema-support',
    sessionId: 'sess-a7f2',
  },
  {
    timestamp: '2026-04-15T12:34:19Z',
    kind: 'delegation',
    summary: 'FAQ returned result to Sales Specialist.',
    schemaId: 'schema-support',
    sessionId: 'sess-a7f2',
  },
  {
    timestamp: '2026-04-15T12:34:10Z',
    kind: 'delegation',
    summary: 'Triage delegated to Sales Specialist (SSO pricing inquiry).',
    schemaId: 'schema-support',
    sessionId: 'sess-a7f2',
  },
  {
    timestamp: '2026-04-15T12:34:05Z',
    kind: 'trigger_fired',
    summary: 'Support Widget trigger fired → Triage.',
    schemaId: 'schema-support',
  },
  {
    timestamp: '2026-04-15T12:33:47Z',
    kind: 'delegation',
    summary: 'Billing Agent received duplicate-charge lookup result.',
    schemaId: 'schema-support',
    sessionId: 'sess-b3d1',
  },
  {
    timestamp: '2026-04-15T12:33:40Z',
    kind: 'trigger_fired',
    summary: 'Chat Endpoint fired → Triage (billing dispute).',
    schemaId: 'schema-support',
  },
  {
    timestamp: '2026-04-15T12:30:02Z',
    kind: 'delegation',
    summary: 'Sales Qualification Orch delegated to Lead Researcher.',
    schemaId: 'schema-sales',
    sessionId: 'sess-c9e4',
  },
  {
    timestamp: '2026-04-15T12:30:00Z',
    kind: 'flow_entered',
    summary: 'Sales Qualification Orch entered "Deep Qualification Interview".',
    schemaId: 'schema-sales',
    sessionId: 'sess-c9e4',
  },
  {
    timestamp: '2026-04-15T12:00:03Z',
    kind: 'session_completed',
    summary: 'Health Monitor completed hourly check (all green).',
    schemaId: 'schema-health',
  },
  {
    timestamp: '2026-04-15T12:00:00Z',
    kind: 'trigger_fired',
    summary: 'Hourly cron fired → Health Monitor.',
    schemaId: 'schema-health',
  },
];

// ============================================================================
// Selectors
// ============================================================================

export function getAgentById(id: string): V2Agent | undefined {
  return v2Agents.find((a) => a.id === id);
}

export function getSchemaById(id: string): V2Schema | undefined {
  return v2Schemas.find((s) => s.id === id);
}

export function getTriggerById(id: string): V2Trigger | undefined {
  return v2Triggers.find((t) => t.id === id);
}

export function getSessionById(id: string): V2Session | undefined {
  return v2Sessions.find((s) => s.id === id);
}

export function getFlowsForAgent(agentId: string): V2Flow[] {
  return v2Flows.filter((f) => f.agentId === agentId);
}

// getSchemaAgents returns the schema's members. V2: membership is derived
// from agent_relations (entry agent + reachable delegates) per
// docs/architecture/agent-first-runtime.md §2.1. The mock keeps an
// `agentIds` cache because the prototype mutates schemas optimistically;
// real production reads come from `GET /api/v1/schemas/{id}/agents` which
// derives via the SQL UNION on agent_relations.
export function getSchemaAgents(schemaId: string): V2Agent[] {
  const schema = getSchemaById(schemaId);
  if (!schema) return [];
  return schema.agentIds.map(getAgentById).filter((a): a is V2Agent => !!a);
}

export function getSchemaRelations(schemaId: string): V2AgentRelation[] {
  const schema = getSchemaById(schemaId);
  if (!schema) return [];
  const agentSet = new Set(schema.agentIds);
  return v2AgentRelations.filter(
    (r) => agentSet.has(r.sourceAgentId) && agentSet.has(r.targetAgentId),
  );
}

export function getSchemaTriggers(schemaId: string): V2Trigger[] {
  return v2Triggers.filter((t) => t.schemaId === schemaId);
}

export function getSchemaActiveSessions(schemaId: string): V2Session[] {
  return v2Sessions.filter((s) => s.schemaId === schemaId && s.status === 'active');
}
