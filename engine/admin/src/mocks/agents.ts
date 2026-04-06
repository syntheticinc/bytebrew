import type { AgentDetail } from '../types';

export const MOCK_AGENTS: Record<string, AgentDetail> = {
  classifier: {
    name: 'classifier',
    description: 'Routes incoming requests to appropriate agent',
    tools_count: 3,
    has_knowledge: true,
    model_id: 1,
    system_prompt:
      'You are a classifier agent. Route incoming support requests to the appropriate specialist agent based on the query topic and complexity.',
    tools: ['search_knowledge', 'check_category', 'route_request'],
    can_spawn: [],
    lifecycle: 'persistent',
    tool_execution: 'sequential',
    max_steps: 10,
    max_context_size: 8192,
    max_turn_duration: 120,
    confirm_before: [],
    mcp_servers: [],
  },
  'support-agent': {
    name: 'support-agent',
    description: 'Handles customer support queries',
    tools_count: 8,
    has_knowledge: true,
    kit: 'support',
    model_id: 2,
    system_prompt:
      'You are a helpful support agent for ByteBrew. Your role is to assist users with billing, account, and subscription questions. Always be professional and empathetic. Use available tools to look up account information before responding.',
    tools: [
      'search_knowledge',
      'memory_recall',
      'memory_store',
      'get_account_info',
      'list_plans',
      'check_status',
      'send_email',
      'update_preference',
    ],
    can_spawn: ['escalation'],
    lifecycle: 'persistent',
    tool_execution: 'sequential',
    max_steps: 50,
    max_context_size: 16000,
    max_turn_duration: 120,
    confirm_before: ['cancel_subscription', 'process_refund'],
    mcp_servers: ['google-sheets'],
    escalation: {
      action: 'transfer_to_human',
      triggers: ['confidence < 0.4', 'user requests human'],
    },
  },
  escalation: {
    name: 'escalation',
    description: 'Handles complex escalated issues',
    tools_count: 5,
    has_knowledge: true,
    model_id: 3,
    system_prompt:
      'You handle escalated support cases that require deeper analysis. You have access to admin tools and can make account changes.',
    tools: [
      'search_knowledge',
      'get_account_info',
      'modify_account',
      'process_refund',
      'create_ticket',
    ],
    can_spawn: [],
    lifecycle: 'spawn',
    tool_execution: 'sequential',
    max_steps: 30,
    max_context_size: 16000,
    max_turn_duration: 120,
    confirm_before: ['process_refund'],
    mcp_servers: [],
  },
};

export const MOCK_MODELS = [
  { id: 1, name: 'claude-haiku-3', model_name: 'claude-3-haiku-20240307' },
  { id: 2, name: 'claude-sonnet-3.7', model_name: 'claude-3-5-sonnet-20241022' },
  { id: 3, name: 'claude-opus-4', model_name: 'claude-opus-4-20260414' },
];
