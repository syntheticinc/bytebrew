import { xchacha20poly1305 } from "@noble/ciphers/chacha";
import { randomBytes } from "@noble/ciphers/webcrypto";
import nacl from "tweetnacl";

const X25519_KEY_SIZE = 32;
const NONCE_SIZE = 24; // XChaCha20-Poly1305 nonce
const RANDOM_NONCE_PREFIX = 16;
const COUNTER_BYTES = 8;
const TAG_SIZE = 16; // Poly1305 tag

export interface ICryptoService {
  generateKeyPair(): { publicKey: Uint8Array; privateKey: Uint8Array };
  computeSharedSecret(
    privateKey: Uint8Array,
    peerPublicKey: Uint8Array,
  ): Uint8Array;
  encrypt(
    plaintext: Uint8Array,
    sharedSecret: Uint8Array,
    counter: number,
  ): Uint8Array;
  decrypt(sealed: Uint8Array, sharedSecret: Uint8Array): Uint8Array;
}

/**
 * X25519 ECDH key exchange + XChaCha20-Poly1305 encryption.
 * Compatible with Go CryptoService in bytebrew-srv.
 *
 * Nonce layout (24 bytes): 16 random bytes + 8 bytes counter (little-endian).
 * Output format: nonce(24) || ciphertext+tag.
 */
export class CryptoService implements ICryptoService {
  generateKeyPair(): { publicKey: Uint8Array; privateKey: Uint8Array } {
    const keyPair = nacl.box.keyPair();
    return {
      publicKey: keyPair.publicKey,
      privateKey: keyPair.secretKey,
    };
  }

  computeSharedSecret(
    privateKey: Uint8Array,
    peerPublicKey: Uint8Array,
  ): Uint8Array {
    if (privateKey.length !== X25519_KEY_SIZE) {
      throw new Error(
        `private key must be ${X25519_KEY_SIZE} bytes, got ${privateKey.length}`,
      );
    }
    if (peerPublicKey.length !== X25519_KEY_SIZE) {
      throw new Error(
        `peer public key must be ${X25519_KEY_SIZE} bytes, got ${peerPublicKey.length}`,
      );
    }

    return nacl.scalarMult(privateKey, peerPublicKey);
  }

  encrypt(
    plaintext: Uint8Array,
    sharedSecret: Uint8Array,
    counter: number,
  ): Uint8Array {
    // Build nonce: 16 random bytes + 8 bytes counter (little-endian)
    const nonce = new Uint8Array(NONCE_SIZE);
    const randomPart = randomBytes(RANDOM_NONCE_PREFIX);
    nonce.set(randomPart, 0);

    const counterView = new DataView(nonce.buffer, nonce.byteOffset + RANDOM_NONCE_PREFIX, COUNTER_BYTES);
    counterView.setUint32(0, counter & 0xffffffff, true); // low 32 bits
    counterView.setUint32(4, Math.floor(counter / 0x100000000) & 0xffffffff, true); // high 32 bits

    const cipher = xchacha20poly1305(sharedSecret, nonce);
    const ciphertext = cipher.encrypt(plaintext);

    // Output: nonce(24) || ciphertext+tag
    const result = new Uint8Array(NONCE_SIZE + ciphertext.length);
    result.set(nonce, 0);
    result.set(ciphertext, NONCE_SIZE);

    return result;
  }

  decrypt(sealed: Uint8Array, sharedSecret: Uint8Array): Uint8Array {
    const minSize = NONCE_SIZE + TAG_SIZE;
    if (sealed.length < minSize) {
      throw new Error(
        `ciphertext too short: need at least ${minSize} bytes, got ${sealed.length}`,
      );
    }

    const nonce = sealed.slice(0, NONCE_SIZE);
    const ciphertext = sealed.slice(NONCE_SIZE);

    const cipher = xchacha20poly1305(sharedSecret, nonce);
    return cipher.decrypt(ciphertext);
  }
}
