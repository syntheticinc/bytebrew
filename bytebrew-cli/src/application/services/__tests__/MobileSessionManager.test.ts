import { describe, test, expect } from 'bun:test';
import { MobileSessionManager, type SessionInfo } from '../MobileSessionManager';

function session(id: string, status: 'active' | 'idle' = 'idle'): SessionInfo {
  return {
    sessionId: id,
    projectName: `project-${id}`,
    status,
    startedAt: new Date('2026-01-01'),
  };
}

describe('MobileSessionManager', () => {
  test('setCurrentSession + getCurrentSession', () => {
    const mgr = new MobileSessionManager();
    mgr.setCurrentSession(session('s1', 'active'));

    const current = mgr.getCurrentSession();
    expect(current).toBeDefined();
    expect(current!.sessionId).toBe('s1');
    expect(current!.status).toBe('active');
  });

  test('getCurrentSession returns undefined when empty', () => {
    const mgr = new MobileSessionManager();
    expect(mgr.getCurrentSession()).toBeUndefined();
  });

  test('getCurrentSession returns active session first', () => {
    const mgr = new MobileSessionManager();
    mgr.setCurrentSession(session('s1', 'idle'));
    mgr.setCurrentSession(session('s2', 'active'));

    const current = mgr.getCurrentSession();
    expect(current!.sessionId).toBe('s2');
  });

  test('getCurrentSession falls back to first session when none active', () => {
    const mgr = new MobileSessionManager();
    mgr.setCurrentSession(session('s1', 'idle'));
    mgr.setCurrentSession(session('s2', 'idle'));

    const current = mgr.getCurrentSession();
    expect(current).toBeDefined();
  });

  test('listSessions returns all sessions', () => {
    const mgr = new MobileSessionManager();
    mgr.setCurrentSession(session('s1'));
    mgr.setCurrentSession(session('s2'));
    mgr.setCurrentSession(session('s3'));

    const list = mgr.listSessions();
    expect(list).toHaveLength(3);

    const ids = list.map((s) => s.sessionId).sort();
    expect(ids).toEqual(['s1', 's2', 's3']);
  });

  test('updateStatus changes session status', () => {
    const mgr = new MobileSessionManager();
    mgr.setCurrentSession(session('s1', 'idle'));

    mgr.updateStatus('s1', 'active');

    const current = mgr.getCurrentSession();
    expect(current!.status).toBe('active');
  });

  test('updateStatus for unknown session is no-op', () => {
    const mgr = new MobileSessionManager();
    // Should not throw
    mgr.updateStatus('nonexistent', 'active');
    expect(mgr.listSessions()).toHaveLength(0);
  });

  test('clearSession removes the session', () => {
    const mgr = new MobileSessionManager();
    mgr.setCurrentSession(session('s1'));
    mgr.setCurrentSession(session('s2'));

    mgr.clearSession('s1');

    const list = mgr.listSessions();
    expect(list).toHaveLength(1);
    expect(list[0].sessionId).toBe('s2');
  });

  test('clearSession for unknown session is no-op', () => {
    const mgr = new MobileSessionManager();
    mgr.clearSession('nonexistent');
    expect(mgr.listSessions()).toHaveLength(0);
  });

  test('setCurrentSession returns a defensive copy', () => {
    const mgr = new MobileSessionManager();
    const original = session('s1', 'active');
    mgr.setCurrentSession(original);

    const retrieved = mgr.getCurrentSession()!;
    retrieved.status = 'error';

    // Original in store should be unchanged
    expect(mgr.getCurrentSession()!.status).toBe('active');
  });
});
