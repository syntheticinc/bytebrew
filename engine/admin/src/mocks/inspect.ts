import type { SessionTrace } from '../types';

export const MOCK_TRACE: SessionTrace = {
  session_id: 'sess_a3f2e8b1',
  agent_name: 'support-agent',
  status: 'completed',
  total_duration_ms: 4500,
  total_tokens: 2420,
  created_at: '2026-04-05T12:00:00Z',
  steps: [
    {
      id: 1,
      kind: 'reasoning',
      label:
        'User asks about changing subscription plan. Need to check current plan and interaction history.',
      duration_ms: 800,
      tokens: 450,
    },
    {
      id: 2,
      kind: 'tool_call',
      label: 'search_knowledge',
      input: JSON.stringify({ query: 'change subscription plan', top_k: 5 }, null, 2),
      output: JSON.stringify(
        [
          { chunk: 'To change your plan, go to Settings > Billing...', score: 0.91 },
          { chunk: 'Team plan includes 10 seats and priority support...', score: 0.87 },
        ],
        null,
        2,
      ),
      duration_ms: 1200,
      tokens: 320,
    },
    {
      id: 3,
      kind: 'memory_recall',
      label: 'Previous interaction context',
      output: JSON.stringify(
        { memory: 'Customer contacted 3 days ago about billing. Issue was resolved.' },
        null,
        2,
      ),
      duration_ms: 100,
      tokens: 150,
    },
    {
      id: 4,
      kind: 'knowledge_search',
      label: 'Search knowledge base',
      input: JSON.stringify({ query: 'refund policy' }, null, 2),
      output: JSON.stringify(
        { results: [{ content: 'Our refund policy allows...', score: 0.92 }] },
        null,
        2,
      ),
      duration_ms: 150,
      tokens: 50,
    },
    {
      id: 5,
      kind: 'tool_call',
      label: 'check_account',
      input: JSON.stringify({ user_id: 'u_4f3a' }, null, 2),
      output: JSON.stringify(
        { plan: 'pro', billing_date: '2025-04-15', seats: 1, status: 'active' },
        null,
        2,
      ),
      duration_ms: 800,
      tokens: 280,
    },
    {
      id: 6,
      kind: 'reasoning',
      label:
        'Customer has prior interaction history. Forming personalized response with account context.',
      duration_ms: 400,
      tokens: 220,
    },
    {
      id: 7,
      kind: 'guardrail_check',
      label: 'Output validation (JSON Schema)',
      input: JSON.stringify({ schema: { type: 'object', required: ['answer'] } }, null, 2),
      output: JSON.stringify({ pass: true }, null, 2),
      duration_ms: 50,
      tokens: 100,
    },
    {
      id: 8,
      kind: 'final_answer',
      label:
        'Hello! I see you have previously contacted us. Happy to help with upgrading to the Team plan. Your current Pro plan ($29/mo) can be upgraded to Team ($49/seat/mo) which includes collaboration features and priority support.',
      duration_ms: 1000,
      tokens: 850,
    },
  ],
};

export const MOCK_SESSIONS = [
  { id: 'sess_a3f2e8b1', status: 'completed' as const, created_at: '2026-04-05T12:00:00Z' },
  { id: 'sess_b9e1c4d7', status: 'completed' as const, created_at: '2026-04-05T11:30:00Z' },
  { id: 'sess_c4d7f2a8', status: 'failed' as const, created_at: '2026-04-05T10:15:00Z' },
];
