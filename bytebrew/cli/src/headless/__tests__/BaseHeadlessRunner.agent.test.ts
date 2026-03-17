import { describe, it, expect, beforeEach, afterEach, spyOn } from 'bun:test';
import { BaseHeadlessRunner } from '../BaseHeadlessRunner.js';
import { AppConfig } from '../../config/index.js';

/**
 * Testable subclass to expose protected methods for testing
 */
class TestableRunner extends BaseHeadlessRunner {
  // Expose protected method for testing
  public testPrintAgentPrefix(agentId: string): void {
    this.printAgentPrefix(agentId);
  }

  // Expose lastAgentId for testing
  public getLastAgentId(): string {
    return (this as any).lastAgentId;
  }

  // Required abstract implementation (noop for tests)
  async run(_question: string): Promise<void> {}
}

/**
 * Create minimal valid config for tests
 */
function createTestConfig(): AppConfig {
  return {
    projectRoot: process.cwd(),
    serverAddress: 'localhost:60401',
    projectKey: 'test',
    userId: 'test-user',
    debug: false,
  };
}

describe('BaseHeadlessRunner - agent support', () => {
  let runner: TestableRunner;
  let logSpy: ReturnType<typeof spyOn>;

  beforeEach(() => {
    const config = createTestConfig();
    runner = new TestableRunner(config);
    logSpy = spyOn(console, 'log').mockImplementation(() => {});
  });

  afterEach(() => {
    logSpy.mockRestore();
  });

  describe('printAgentPrefix', () => {
    it('no prefix when agentId matches last (supervisor → supervisor)', () => {
      // Initial lastAgentId is 'supervisor', so first supervisor call should not log
      runner.testPrintAgentPrefix('supervisor');

      expect(logSpy).not.toHaveBeenCalled();
      expect(runner.getLastAgentId()).toBe('supervisor');
    });

    it('prints "[Agent: Supervisor]" when switching to supervisor from code-agent', () => {
      // First set lastAgentId to code-agent
      runner.testPrintAgentPrefix('code-agent-abc123');
      logSpy.mockClear();

      // Now switch back to supervisor
      runner.testPrintAgentPrefix('supervisor');

      expect(logSpy).toHaveBeenCalledTimes(1);
      expect(logSpy).toHaveBeenCalledWith('\n[Agent: Supervisor]');
      expect(runner.getLastAgentId()).toBe('supervisor');
    });

    it('prints "[Agent: Code Agent abc123]" when switching to code-agent-abc123', () => {
      // Initial is supervisor, switch to code-agent
      runner.testPrintAgentPrefix('code-agent-abc123');

      expect(logSpy).toHaveBeenCalledTimes(1);
      expect(logSpy).toHaveBeenCalledWith('\n[Agent: Code Agent abc123]');
      expect(runner.getLastAgentId()).toBe('code-agent-abc123');
    });

    it('strips "code-agent-" prefix from name', () => {
      runner.testPrintAgentPrefix('code-agent-xyz789');

      expect(logSpy).toHaveBeenCalledWith('\n[Agent: Code Agent xyz789]');
    });

    it('prefix only once per agent change (same agentId does not log repeatedly)', () => {
      // First call to code-agent
      runner.testPrintAgentPrefix('code-agent-test');
      expect(logSpy).toHaveBeenCalledTimes(1);

      // Second call to same code-agent (should not log)
      runner.testPrintAgentPrefix('code-agent-test');
      expect(logSpy).toHaveBeenCalledTimes(1); // Still 1
    });

    it('alternating: A → B → A → B prints prefix each time', () => {
      // Start: supervisor (initial, no log)
      runner.testPrintAgentPrefix('supervisor');
      expect(logSpy).toHaveBeenCalledTimes(0);

      // supervisor → code-agent-A (prints)
      runner.testPrintAgentPrefix('code-agent-A');
      expect(logSpy).toHaveBeenCalledTimes(1);
      expect(logSpy).toHaveBeenCalledWith('\n[Agent: Code Agent A]');

      // code-agent-A → supervisor (prints)
      runner.testPrintAgentPrefix('supervisor');
      expect(logSpy).toHaveBeenCalledTimes(2);
      expect(logSpy).toHaveBeenCalledWith('\n[Agent: Supervisor]');

      // supervisor → code-agent-A (prints)
      runner.testPrintAgentPrefix('code-agent-A');
      expect(logSpy).toHaveBeenCalledTimes(3);
      expect(logSpy).toHaveBeenCalledWith('\n[Agent: Code Agent A]');
    });

    it('initial = supervisor, first supervisor message → no prefix', () => {
      // Initial lastAgentId is 'supervisor' by default
      expect(runner.getLastAgentId()).toBe('supervisor');

      // First call with supervisor (matches initial) → no log
      runner.testPrintAgentPrefix('supervisor');
      expect(logSpy).not.toHaveBeenCalled();
    });

    it('first code-agent message → prints prefix', () => {
      // Initial is supervisor, first code-agent should print
      runner.testPrintAgentPrefix('code-agent-first');

      expect(logSpy).toHaveBeenCalledTimes(1);
      expect(logSpy).toHaveBeenCalledWith('\n[Agent: Code Agent first]');
    });

    it('handles short code-agent IDs', () => {
      runner.testPrintAgentPrefix('code-agent-1');

      expect(logSpy).toHaveBeenCalledWith('\n[Agent: Code Agent 1]');
    });

    it('handles long code-agent IDs', () => {
      runner.testPrintAgentPrefix('code-agent-uuid-1234-5678-9abc-def0');

      expect(logSpy).toHaveBeenCalledWith('\n[Agent: Code Agent uuid-1234-5678-9abc-def0]');
    });

    it('supervisor label is capitalized correctly', () => {
      // Start with code-agent so supervisor will log
      runner.testPrintAgentPrefix('code-agent-temp');
      logSpy.mockClear();

      runner.testPrintAgentPrefix('supervisor');

      expect(logSpy).toHaveBeenCalledWith('\n[Agent: Supervisor]');
    });

    it('consecutive different code-agents print prefix each time', () => {
      runner.testPrintAgentPrefix('code-agent-A');
      expect(logSpy).toHaveBeenCalledTimes(1);

      runner.testPrintAgentPrefix('code-agent-B');
      expect(logSpy).toHaveBeenCalledTimes(2);
      expect(logSpy).toHaveBeenCalledWith('\n[Agent: Code Agent B]');

      runner.testPrintAgentPrefix('code-agent-C');
      expect(logSpy).toHaveBeenCalledTimes(3);
      expect(logSpy).toHaveBeenCalledWith('\n[Agent: Code Agent C]');
    });
  });
});
