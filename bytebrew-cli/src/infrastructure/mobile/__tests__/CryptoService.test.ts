import { describe, it, expect } from 'bun:test';
import { CryptoService } from '../CryptoService.js';

describe('CryptoService', () => {
  const crypto = new CryptoService();

  describe('generateKeyPair', () => {
    it('returns 32-byte public and private keys', () => {
      const { publicKey, privateKey } = crypto.generateKeyPair();
      expect(publicKey.length).toBe(32);
      expect(privateKey.length).toBe(32);
    });

    it('generates different keys each time', () => {
      const a = crypto.generateKeyPair();
      const b = crypto.generateKeyPair();
      expect(a.publicKey).not.toEqual(b.publicKey);
      expect(a.privateKey).not.toEqual(b.privateKey);
    });
  });

  describe('computeSharedSecret', () => {
    it('is symmetric: A->B == B->A', () => {
      const alice = crypto.generateKeyPair();
      const bob = crypto.generateKeyPair();

      const secretAB = crypto.computeSharedSecret(alice.privateKey, bob.publicKey);
      const secretBA = crypto.computeSharedSecret(bob.privateKey, alice.publicKey);

      expect(secretAB).toEqual(secretBA);
    });

    it('returns 32-byte shared secret', () => {
      const alice = crypto.generateKeyPair();
      const bob = crypto.generateKeyPair();
      const secret = crypto.computeSharedSecret(alice.privateKey, bob.publicKey);
      expect(secret.length).toBe(32);
    });

    it('throws if private key is wrong size', () => {
      const bob = crypto.generateKeyPair();
      expect(() =>
        crypto.computeSharedSecret(new Uint8Array(16), bob.publicKey),
      ).toThrow('private key must be 32 bytes');
    });

    it('throws if public key is wrong size', () => {
      const alice = crypto.generateKeyPair();
      expect(() =>
        crypto.computeSharedSecret(alice.privateKey, new Uint8Array(16)),
      ).toThrow('peer public key must be 32 bytes');
    });
  });

  describe('encrypt + decrypt round-trip', () => {
    it('encrypts and decrypts to same plaintext', () => {
      const alice = crypto.generateKeyPair();
      const bob = crypto.generateKeyPair();
      const sharedSecret = crypto.computeSharedSecret(alice.privateKey, bob.publicKey);

      const plaintext = new TextEncoder().encode('Hello, World!');
      const sealed = crypto.encrypt(plaintext, sharedSecret, 1);
      const decrypted = crypto.decrypt(sealed, sharedSecret);

      expect(decrypted).toEqual(plaintext);
    });

    it('works with empty plaintext', () => {
      const alice = crypto.generateKeyPair();
      const bob = crypto.generateKeyPair();
      const sharedSecret = crypto.computeSharedSecret(alice.privateKey, bob.publicKey);

      const plaintext = new Uint8Array(0);
      const sealed = crypto.encrypt(plaintext, sharedSecret, 0);
      const decrypted = crypto.decrypt(sealed, sharedSecret);

      expect(decrypted).toEqual(plaintext);
    });

    it('works with different counter values', () => {
      const alice = crypto.generateKeyPair();
      const bob = crypto.generateKeyPair();
      const sharedSecret = crypto.computeSharedSecret(alice.privateKey, bob.publicKey);

      const plaintext = new TextEncoder().encode('test');

      for (const counter of [0, 1, 100, 999999]) {
        const sealed = crypto.encrypt(plaintext, sharedSecret, counter);
        const decrypted = crypto.decrypt(sealed, sharedSecret);
        expect(decrypted).toEqual(plaintext);
      }
    });
  });

  describe('decrypt with wrong key', () => {
    it('throws on decryption with wrong shared secret', () => {
      const alice = crypto.generateKeyPair();
      const bob = crypto.generateKeyPair();
      const eve = crypto.generateKeyPair();

      const realSecret = crypto.computeSharedSecret(alice.privateKey, bob.publicKey);
      const wrongSecret = crypto.computeSharedSecret(eve.privateKey, bob.publicKey);

      const plaintext = new TextEncoder().encode('secret data');
      const sealed = crypto.encrypt(plaintext, realSecret, 1);

      expect(() => crypto.decrypt(sealed, wrongSecret)).toThrow();
    });
  });

  describe('nonce layout', () => {
    it('output starts with 24-byte nonce, followed by ciphertext+tag', () => {
      const alice = crypto.generateKeyPair();
      const bob = crypto.generateKeyPair();
      const sharedSecret = crypto.computeSharedSecret(alice.privateKey, bob.publicKey);

      const plaintext = new TextEncoder().encode('hello');
      const sealed = crypto.encrypt(plaintext, sharedSecret, 42);

      // nonce(24) + ciphertext(5) + tag(16)
      expect(sealed.length).toBe(24 + 5 + 16);
    });

    it('counter is encoded in last 8 bytes of nonce (little-endian)', () => {
      const alice = crypto.generateKeyPair();
      const bob = crypto.generateKeyPair();
      const sharedSecret = crypto.computeSharedSecret(alice.privateKey, bob.publicKey);

      const sealed = crypto.encrypt(new Uint8Array([0]), sharedSecret, 42);

      // Extract nonce bytes 16..24 (counter part)
      const counterBytes = sealed.slice(16, 24);
      const view = new DataView(counterBytes.buffer, counterBytes.byteOffset, 8);
      const low = view.getUint32(0, true);
      const high = view.getUint32(4, true);

      expect(low).toBe(42);
      expect(high).toBe(0);
    });

    it('two encryptions of same plaintext produce different output (random nonce prefix)', () => {
      const alice = crypto.generateKeyPair();
      const bob = crypto.generateKeyPair();
      const sharedSecret = crypto.computeSharedSecret(alice.privateKey, bob.publicKey);

      const plaintext = new TextEncoder().encode('same');
      const sealed1 = crypto.encrypt(plaintext, sharedSecret, 1);
      const sealed2 = crypto.encrypt(plaintext, sharedSecret, 1);

      // Random prefix (first 16 bytes) should differ
      const prefix1 = sealed1.slice(0, 16);
      const prefix2 = sealed2.slice(0, 16);
      expect(prefix1).not.toEqual(prefix2);
    });
  });

  describe('decrypt validation', () => {
    it('throws if sealed data is too short', () => {
      const alice = crypto.generateKeyPair();
      const bob = crypto.generateKeyPair();
      const secret = crypto.computeSharedSecret(alice.privateKey, bob.publicKey);

      expect(() => crypto.decrypt(new Uint8Array(10), secret)).toThrow(
        'ciphertext too short',
      );
    });
  });
});
