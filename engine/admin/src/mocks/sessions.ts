import type { SessionSummary, SessionStatus } from '../types';

// ─── Mock sessions (28 entries for pagination testing) ───────────────────────

const agents = ['support-agent', 'dev-assistant', 'sales-bot', 'onboarding-agent', 'escalation-handler'];
const statuses: SessionStatus[] = ['completed', 'completed', 'completed', 'running', 'failed', 'blocked', 'timeout'];

function makeSessions(count: number): SessionSummary[] {
  const sessions: SessionSummary[] = [];
  for (let i = 0; i < count; i++) {
    const agent = agents[i % agents.length]!;
    const status = statuses[i % statuses.length]!;
    const hours = Math.floor(i * 0.5);
    const date = new Date(2026, 3, 5, 14 - hours, (60 - i * 2 + 60) % 60);
    sessions.push({
      session_id: `sess_${(0xa3f2e8b1 + i * 0x1111).toString(16).slice(0, 8)}`,
      entry_agent: agent,
      status,
      duration_ms: 1500 + Math.floor(Math.random() * 8000),
      total_tokens: 500 + Math.floor(Math.random() * 5000),
      created_at: date.toISOString(),
    });
  }
  return sessions;
}

export const MOCK_SESSIONS_LIST: SessionSummary[] = makeSessions(28);
