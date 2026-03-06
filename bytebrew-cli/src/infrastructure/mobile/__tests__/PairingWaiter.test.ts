import { describe, it, expect } from 'bun:test';
import { PairingWaiter } from '../PairingWaiter.js';

describe('PairingWaiter', () => {
  describe('wait + resolve', () => {
    it('resolves with deviceId and deviceName', async () => {
      const waiter = new PairingWaiter();

      const promise = waiter.wait('tok-1', 5000);
      waiter.resolve('tok-1', 'dev-123', 'iPhone 15');

      const result = await promise;
      expect(result.deviceId).toBe('dev-123');
      expect(result.deviceName).toBe('iPhone 15');
    });
  });

  describe('wait + timeout', () => {
    it('rejects after timeout', async () => {
      const waiter = new PairingWaiter();

      const promise = waiter.wait('tok-1', 50); // 50ms timeout

      await expect(promise).rejects.toThrow('pairing timeout for token: tok-1');
    });
  });

  describe('wait + cancel', () => {
    it('rejects immediately when cancelled', async () => {
      const waiter = new PairingWaiter();

      const promise = waiter.wait('tok-1', 5000);
      waiter.cancel('tok-1');

      await expect(promise).rejects.toThrow('pairing cancelled for token: tok-1');
    });
  });

  describe('double wait same token', () => {
    it('rejects duplicate wait', async () => {
      const waiter = new PairingWaiter();

      waiter.wait('tok-1', 5000);

      await expect(waiter.wait('tok-1', 5000)).rejects.toThrow(
        'duplicate wait for token: tok-1',
      );

      // Cleanup: resolve the first wait to avoid dangling timer
      waiter.resolve('tok-1', 'dev', 'name');
    });
  });

  describe('resolve without wait', () => {
    it('does not throw (silent no-op)', () => {
      const waiter = new PairingWaiter();
      expect(() => waiter.resolve('tok-none', 'dev', 'name')).not.toThrow();
    });
  });

  describe('cancel without wait', () => {
    it('does not throw (silent no-op)', () => {
      const waiter = new PairingWaiter();
      expect(() => waiter.cancel('tok-none')).not.toThrow();
    });
  });
});
