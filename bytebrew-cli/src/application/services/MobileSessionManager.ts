// MobileSessionManager - tracks CLI sessions for mobile clients
import { getLogger, type Logger } from '../../lib/logger.js';

export type SessionStatus = 'active' | 'idle' | 'completed' | 'error';

export interface SessionInfo {
  sessionId: string;
  projectName: string;
  status: SessionStatus;
  currentTask?: string;
  startedAt: Date;
}

/**
 * Tracks the current CLI session(s) for mobile clients.
 * In a typical CLI instance there is one session, but the interface
 * supports a list for forward-compatibility.
 */
export class MobileSessionManager {
  private readonly sessions = new Map<string, SessionInfo>();
  private readonly logger: Logger;

  constructor() {
    this.logger = getLogger().child({ component: 'MobileSessionManager' });
  }

  setCurrentSession(info: SessionInfo): void {
    this.sessions.set(info.sessionId, { ...info });
    this.logger.info('Session set', {
      sessionId: info.sessionId,
      status: info.status,
    });
  }

  getCurrentSession(): SessionInfo | undefined {
    // Return the first active session, or the most recently added one
    for (const session of this.sessions.values()) {
      if (session.status === 'active') {
        return { ...session };
      }
    }

    // Fallback: return any session if none are active
    const first = this.sessions.values().next();
    if (first.done) return undefined;
    return { ...first.value };
  }

  listSessions(): SessionInfo[] {
    return Array.from(this.sessions.values()).map((s) => ({ ...s }));
  }

  clearSession(sessionId: string): void {
    if (this.sessions.delete(sessionId)) {
      this.logger.info('Session cleared', { sessionId });
    }
  }

  updateStatus(sessionId: string, status: SessionStatus): void {
    const session = this.sessions.get(sessionId);
    if (!session) {
      this.logger.warn('Cannot update status: session not found', { sessionId });
      return;
    }

    session.status = status;
    this.logger.debug('Session status updated', { sessionId, status });
  }
}
