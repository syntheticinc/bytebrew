import { describe, it, expect } from 'bun:test';
import { formatLifecycleMessage } from '../formatLifecycleMessage.js';

describe('formatLifecycleMessage', () => {
  it('formats agent_spawned for supervisor', () => {
    expect(formatLifecycleMessage('agent_spawned', 'supervisor', 'Analyze project'))
      .toBe('+ Supervisor spawned: "Analyze project"');
  });

  it('formats agent_spawned for code agent', () => {
    expect(formatLifecycleMessage('agent_spawned', 'code-agent-abc123', 'Create hello.go'))
      .toBe('+ Code Agent [abc123] spawned: "Create hello.go"');
  });

  it('formats agent_completed with Completed: prefix', () => {
    expect(formatLifecycleMessage('agent_completed', 'code-agent-xyz', 'Completed: Task done\nExtra details'))
      .toBe('✓ Code Agent [xyz] completed: "Task done"');
  });

  it('formats agent_completed without prefix', () => {
    expect(formatLifecycleMessage('agent_completed', 'code-agent-xyz', 'Task done'))
      .toBe('✓ Code Agent [xyz] completed: "Task done"');
  });

  it('formats agent_failed with Failed: prefix', () => {
    expect(formatLifecycleMessage('agent_failed', 'code-agent-err', 'Failed: Out of memory\nStack trace'))
      .toBe('✗ Code Agent [err] failed: "Out of memory"');
  });

  it('formats agent_failed without prefix', () => {
    expect(formatLifecycleMessage('agent_failed', 'code-agent-err', 'Out of memory'))
      .toBe('✗ Code Agent [err] failed: "Out of memory"');
  });

  it('formats agent_restarted', () => {
    expect(formatLifecycleMessage('agent_restarted', 'code-agent-r1', 'Retry attempt'))
      .toBe('↻ Code Agent [r1] restarted: "Retry attempt"');
  });

  it('formats unknown lifecycle type with fallback', () => {
    expect(formatLifecycleMessage('agent_paused', 'supervisor', 'Waiting'))
      .toBe('[agent_paused] Supervisor: Waiting');
  });
});
