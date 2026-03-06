import { describe, it, expect } from 'bun:test';
import { PairingToken, PAIRING_TOKEN_EXPIRY_MS } from '../PairingToken.js';

describe('PairingToken', () => {
  describe('create()', () => {
    it('creates token with given values', () => {
      const token = PairingToken.create('full-token-123', 'ABC123');
      expect(token.token).toBe('full-token-123');
      expect(token.shortCode).toBe('ABC123');
      expect(token.used).toBe(false);
    });

    it('sets expiresAt to ~5 minutes from now', () => {
      const before = Date.now();
      const token = PairingToken.create('tok', 'SC');
      const after = Date.now();

      const expectedMin = before + PAIRING_TOKEN_EXPIRY_MS;
      const expectedMax = after + PAIRING_TOKEN_EXPIRY_MS;

      expect(token.expiresAt.getTime()).toBeGreaterThanOrEqual(expectedMin);
      expect(token.expiresAt.getTime()).toBeLessThanOrEqual(expectedMax);
    });

    it('sets empty keys by default', () => {
      const token = PairingToken.create('tok', 'SC');
      expect(token.serverPublicKey.length).toBe(0);
      expect(token.serverPrivateKey.length).toBe(0);
    });

    it('throws if token is empty', () => {
      expect(() => PairingToken.create('', 'SC')).toThrow('token is required');
    });

    it('throws if shortCode is empty', () => {
      expect(() => PairingToken.create('tok', '')).toThrow('short_code is required');
    });
  });

  describe('isExpired()', () => {
    it('returns false for fresh token', () => {
      const token = PairingToken.create('tok', 'SC');
      expect(token.isExpired()).toBe(false);
    });

    it('returns true for token with past expiry', () => {
      const token = PairingToken.fromProps({
        token: 'tok',
        shortCode: 'SC',
        expiresAt: new Date(Date.now() - 1000),
        used: false,
        serverPublicKey: new Uint8Array(0),
        serverPrivateKey: new Uint8Array(0),
      });
      expect(token.isExpired()).toBe(true);
    });
  });

  describe('isValid()', () => {
    it('returns true for fresh unused token', () => {
      const token = PairingToken.create('tok', 'SC');
      expect(token.isValid()).toBe(true);
    });

    it('returns false for used token', () => {
      const token = PairingToken.create('tok', 'SC').markUsed();
      expect(token.isValid()).toBe(false);
    });

    it('returns false for expired token', () => {
      const token = PairingToken.fromProps({
        token: 'tok',
        shortCode: 'SC',
        expiresAt: new Date(Date.now() - 1000),
        used: false,
        serverPublicKey: new Uint8Array(0),
        serverPrivateKey: new Uint8Array(0),
      });
      expect(token.isValid()).toBe(false);
    });
  });

  describe('markUsed()', () => {
    it('returns new token with used=true', () => {
      const original = PairingToken.create('tok', 'SC');
      const used = original.markUsed();

      expect(used.used).toBe(true);
      expect(used.token).toBe('tok');
      expect(used.shortCode).toBe('SC');
    });

    it('does not mutate original (immutable)', () => {
      const original = PairingToken.create('tok', 'SC');
      original.markUsed();
      expect(original.used).toBe(false);
    });
  });

  describe('withKeys()', () => {
    it('returns new token with server keypair', () => {
      const original = PairingToken.create('tok', 'SC');
      const pub = new Uint8Array([1, 2, 3]);
      const priv = new Uint8Array([4, 5, 6]);

      const updated = original.withKeys(pub, priv);

      expect(updated.serverPublicKey).toEqual(pub);
      expect(updated.serverPrivateKey).toEqual(priv);
      expect(updated.token).toBe('tok');
    });

    it('does not mutate original (immutable)', () => {
      const original = PairingToken.create('tok', 'SC');
      original.withKeys(new Uint8Array([1]), new Uint8Array([2]));
      expect(original.serverPublicKey.length).toBe(0);
      expect(original.serverPrivateKey.length).toBe(0);
    });
  });

  describe('fromProps()', () => {
    it('restores token from raw props', () => {
      const props = {
        token: 'full-tok',
        shortCode: 'XYZ',
        expiresAt: new Date('2026-06-01'),
        used: true,
        serverPublicKey: new Uint8Array([10]),
        serverPrivateKey: new Uint8Array([20]),
      };

      const token = PairingToken.fromProps(props);
      expect(token.token).toBe('full-tok');
      expect(token.shortCode).toBe('XYZ');
      expect(token.used).toBe(true);
      expect(token.serverPublicKey).toEqual(new Uint8Array([10]));
    });
  });
});
