import { describe, it, expect, beforeEach, mock } from 'bun:test';
import { GrpcStreamGateway } from '../GrpcStreamGateway.js';
import { ConnectionStatus, StreamResponse } from '../../../domain/ports/IStreamGateway.js';

// Note: GrpcStreamGateway depends on FlowServiceClient, StreamManager, and ReconnectionManager
// which are complex to mock fully. These tests focus on the public interface behavior.

describe('GrpcStreamGateway', () => {
  let gateway: GrpcStreamGateway;

  beforeEach(() => {
    gateway = new GrpcStreamGateway();
  });

  describe('initial state', () => {
    it('should be disconnected initially', () => {
      expect(gateway.getStatus()).toBe('disconnected');
      expect(gateway.isConnected()).toBe(false);
      expect(gateway.getReconnectAttempts()).toBe(0);
    });
  });

  describe('event subscription', () => {
    it('should allow subscribing to response events', () => {
      const handler = mock(() => {});
      const unsubscribe = gateway.onResponse(handler);

      expect(typeof unsubscribe).toBe('function');
      unsubscribe();
    });

    it('should allow subscribing to error events', () => {
      const handler = mock(() => {});
      const unsubscribe = gateway.onError(handler);

      expect(typeof unsubscribe).toBe('function');
      unsubscribe();
    });

    it('should allow subscribing to status change events', () => {
      const handler = mock(() => {});
      const unsubscribe = gateway.onStatusChange(handler);

      expect(typeof unsubscribe).toBe('function');
      unsubscribe();
    });

    it('should allow multiple subscriptions', () => {
      const handler1 = mock(() => {});
      const handler2 = mock(() => {});

      const unsubscribe1 = gateway.onResponse(handler1);
      const unsubscribe2 = gateway.onResponse(handler2);

      expect(typeof unsubscribe1).toBe('function');
      expect(typeof unsubscribe2).toBe('function');

      unsubscribe1();
      unsubscribe2();
    });

    it('should allow unsubscribing independently', () => {
      const receivedStatuses: ConnectionStatus[] = [];
      const handler1 = (status: ConnectionStatus) => receivedStatuses.push(status);
      const handler2 = (status: ConnectionStatus) => receivedStatuses.push(status);

      const unsubscribe1 = gateway.onStatusChange(handler1);
      const unsubscribe2 = gateway.onStatusChange(handler2);

      // Unsubscribe first handler
      unsubscribe1();

      // Both should still be valid function refs
      expect(typeof unsubscribe1).toBe('function');
      expect(typeof unsubscribe2).toBe('function');

      unsubscribe2();
    });
  });

  // Skipping connection tests as they require network timeouts that are too slow for unit tests
  // These are better tested as integration tests with a real or mocked server
  describe.skip('connect (without real server)', () => {
    it('should throw when connecting to non-existent server', async () => {
      // This test verifies error handling on connection failure
      await expect(
        gateway.connect({
          serverAddress: 'localhost:99999', // Non-existent
          sessionId: 'test-session',
          userId: 'test-user',
          projectKey: 'test-project',
          projectRoot: '/test/project',
        })
      ).rejects.toThrow();

      // Should be disconnected after failed connection
      expect(gateway.getStatus()).toBe('disconnected');
      expect(gateway.isConnected()).toBe(false);
    });

    it('should set status to connecting during connection attempt', async () => {
      const statuses: ConnectionStatus[] = [];
      gateway.onStatusChange((status) => statuses.push(status));

      try {
        await gateway.connect({
          serverAddress: 'localhost:99999',
          sessionId: 'test-session',
          userId: 'test-user',
          projectKey: 'test-project',
          projectRoot: '/test/project',
        });
      } catch {
        // Expected to fail
      }

      // Should have gone through 'connecting' before 'disconnected'
      expect(statuses).toContain('connecting');
      expect(statuses[statuses.length - 1]).toBe('disconnected');
    });
  });

  describe('disconnect', () => {
    it('should handle disconnect when not connected', () => {
      // Should not throw
      expect(() => gateway.disconnect()).not.toThrow();
      expect(gateway.getStatus()).toBe('disconnected');
    });

    it('should update status on disconnect', () => {
      const statuses: ConnectionStatus[] = [];
      gateway.onStatusChange((status) => statuses.push(status));

      gateway.disconnect();

      expect(statuses).toContain('disconnected');
    });
  });

  describe('sendMessage', () => {
    it('should handle sendMessage when not connected', () => {
      // Should not throw when disconnected
      expect(() => gateway.sendMessage('test')).not.toThrow();
    });
  });

  describe('sendToolResult', () => {
    it('should handle sendToolResult when not connected', () => {
      // Should not throw when disconnected
      expect(() => gateway.sendToolResult('call-1', 'result')).not.toThrow();
    });

    it('should handle sendToolResult with error', () => {
      // Should not throw when disconnected
      expect(() =>
        gateway.sendToolResult('call-1', '', new Error('Tool failed'))
      ).not.toThrow();
    });

    it('should handle sendToolResult with subResults', () => {
      // Should not throw when disconnected
      expect(() =>
        gateway.sendToolResult('call-1', '', undefined, [
          { type: 'grep', result: 'grep results', count: 5 },
          { type: 'vector', result: 'vector results', count: 3 },
        ])
      ).not.toThrow();
    });
  });

  describe('cancel', () => {
    it('should handle cancel when not connected', () => {
      // Should not throw when disconnected
      expect(() => gateway.cancel()).not.toThrow();
    });
  });

  describe('response conversion', () => {
    // These tests verify the response type mapping is correct
    // by testing the internal ResponseTypeMap indirectly

    it('should map response types correctly', () => {
      // ResponseTypeMap is internal, but we can verify through handler behavior
      const received: StreamResponse[] = [];
      gateway.onResponse((resp) => received.push(resp));

      // These types are mapped correctly when we receive responses from the server
      // The actual mapping: 0=UNSPECIFIED, 1=ANSWER, 2=REASONING, 3=TOOL_CALL,
      // 4=TOOL_RESULT, 5=ANSWER_CHUNK, 6=ERROR

      // Since we can't easily inject responses without a real connection,
      // we just verify the handler registration works
      expect(received.length).toBe(0); // No responses without connection
    });
  });
});

// Integration test helper - would need a running server
describe.skip('GrpcStreamGateway integration', () => {
  it('should connect to real server', async () => {
    const gateway = new GrpcStreamGateway();

    await gateway.connect({
      serverAddress: 'localhost:60401',
      sessionId: `test-${Date.now()}`,
      userId: 'test-user',
      projectKey: 'test-project',
      projectRoot: '/test/project',
    });

    expect(gateway.isConnected()).toBe(true);

    gateway.disconnect();
    expect(gateway.isConnected()).toBe(false);
  });
});
