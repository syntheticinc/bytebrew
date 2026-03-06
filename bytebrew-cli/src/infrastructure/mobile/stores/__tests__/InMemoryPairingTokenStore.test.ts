import { describe, it, expect } from 'bun:test';
import { InMemoryPairingTokenStore } from '../InMemoryPairingTokenStore.js';
import { PairingToken } from '../../../../domain/entities/PairingToken.js';

function freshToken(token: string, shortCode: string): PairingToken {
  return PairingToken.create(token, shortCode);
}

function expiredToken(token: string, shortCode: string): PairingToken {
  return PairingToken.fromProps({
    token,
    shortCode,
    expiresAt: new Date(Date.now() - 1000),
    used: false,
    serverPublicKey: new Uint8Array(0),
    serverPrivateKey: new Uint8Array(0),
  });
}

describe('InMemoryPairingTokenStore', () => {
  describe('add + get', () => {
    it('stores and retrieves token by full token', () => {
      const store = new InMemoryPairingTokenStore();
      store.add(freshToken('full-tok', 'SC1'));

      const found = store.get('full-tok');
      expect(found).toBeDefined();
      expect(found!.shortCode).toBe('SC1');
    });

    it('returns undefined for unknown token', () => {
      const store = new InMemoryPairingTokenStore();
      expect(store.get('unknown')).toBeUndefined();
    });
  });

  describe('getByShortCode', () => {
    it('retrieves token by short code', () => {
      const store = new InMemoryPairingTokenStore();
      store.add(freshToken('full-tok', 'ABC'));

      const found = store.getByShortCode('ABC');
      expect(found).toBeDefined();
      expect(found!.token).toBe('full-tok');
    });

    it('returns undefined for unknown short code', () => {
      const store = new InMemoryPairingTokenStore();
      expect(store.getByShortCode('ZZZ')).toBeUndefined();
    });
  });

  describe('get() fallback to shortCode', () => {
    it('finds token by shortCode when full token not found', () => {
      const store = new InMemoryPairingTokenStore();
      store.add(freshToken('full-tok', 'SC1'));

      const found = store.get('SC1');
      expect(found).toBeDefined();
      expect(found!.token).toBe('full-tok');
    });
  });

  describe('useToken', () => {
    it('returns used token on first call', () => {
      const store = new InMemoryPairingTokenStore();
      store.add(freshToken('tok-1', 'SC1'));

      const used = store.useToken('tok-1');
      expect(used).toBeDefined();
      expect(used!.used).toBe(true);
      expect(used!.token).toBe('tok-1');
    });

    it('returns undefined on second call (already used)', () => {
      const store = new InMemoryPairingTokenStore();
      store.add(freshToken('tok-1', 'SC1'));

      store.useToken('tok-1');
      const second = store.useToken('tok-1');
      expect(second).toBeUndefined();
    });

    it('returns undefined for expired token', () => {
      const store = new InMemoryPairingTokenStore();
      store.add(expiredToken('tok-expired', 'SC-E'));

      const result = store.useToken('tok-expired');
      expect(result).toBeUndefined();
    });

    it('works with shortCode lookup', () => {
      const store = new InMemoryPairingTokenStore();
      store.add(freshToken('tok-1', 'SC1'));

      const used = store.useToken('SC1');
      expect(used).toBeDefined();
      expect(used!.token).toBe('tok-1');
    });

    it('returns undefined for unknown token', () => {
      const store = new InMemoryPairingTokenStore();
      expect(store.useToken('nope')).toBeUndefined();
    });
  });

  describe('removeExpired', () => {
    it('removes only expired tokens and returns count', () => {
      const store = new InMemoryPairingTokenStore();
      store.add(freshToken('tok-valid', 'V1'));
      store.add(expiredToken('tok-exp-1', 'E1'));
      store.add(expiredToken('tok-exp-2', 'E2'));

      const count = store.removeExpired();
      expect(count).toBe(2);

      expect(store.get('tok-valid')).toBeDefined();
      expect(store.get('tok-exp-1')).toBeUndefined();
      expect(store.get('tok-exp-2')).toBeUndefined();
    });

    it('cleans up shortCode index for expired tokens', () => {
      const store = new InMemoryPairingTokenStore();
      store.add(expiredToken('tok-exp', 'SC-E'));

      store.removeExpired();
      expect(store.getByShortCode('SC-E')).toBeUndefined();
    });

    it('returns 0 when no expired tokens', () => {
      const store = new InMemoryPairingTokenStore();
      store.add(freshToken('tok-1', 'SC1'));
      expect(store.removeExpired()).toBe(0);
    });
  });

  describe('remove', () => {
    it('removes token and its shortCode index', () => {
      const store = new InMemoryPairingTokenStore();
      store.add(freshToken('tok-1', 'SC1'));

      store.remove('tok-1');
      expect(store.get('tok-1')).toBeUndefined();
      expect(store.getByShortCode('SC1')).toBeUndefined();
    });
  });
});
