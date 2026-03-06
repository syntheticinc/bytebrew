/**
 * In-memory pairing token store.
 * Port from Go: bytebrew-srv/internal/infrastructure/mobile/pairing_token_store.go
 *
 * Uses dual-index pattern: tokens by full token + shortCode-to-token index.
 * Supports lookup by either full token or short code.
 */

import type { PairingToken } from '../../../domain/entities/PairingToken.js';

export interface IPairingTokenStore {
  add(token: PairingToken): void;
  get(tokenOrCode: string): PairingToken | undefined;
  getByShortCode(code: string): PairingToken | undefined;
  useToken(tokenOrCode: string): PairingToken | undefined;
  remove(token: string): void;
  removeExpired(): number;
}

export class InMemoryPairingTokenStore implements IPairingTokenStore {
  private readonly tokens = new Map<string, PairingToken>();
  private readonly shortCodeIndex = new Map<string, string>(); // shortCode → full token

  add(token: PairingToken): void {
    this.tokens.set(token.token, token);

    if (token.shortCode) {
      this.shortCodeIndex.set(token.shortCode, token.token);
    }
  }

  get(tokenOrCode: string): PairingToken | undefined {
    return this.findToken(tokenOrCode);
  }

  getByShortCode(code: string): PairingToken | undefined {
    const fullToken = this.shortCodeIndex.get(code);
    if (!fullToken) {
      return undefined;
    }
    return this.tokens.get(fullToken);
  }

  /**
   * Atomically finds, validates, and marks a pairing token as used.
   * Returns undefined if token not found, expired, or already used.
   */
  useToken(tokenOrCode: string): PairingToken | undefined {
    const token = this.findToken(tokenOrCode);
    if (!token) {
      return undefined;
    }

    if (!token.isValid()) {
      return undefined;
    }

    const used = token.markUsed();
    this.tokens.set(used.token, used);
    return used;
  }

  remove(token: string): void {
    const existing = this.tokens.get(token);
    if (!existing) {
      return;
    }

    if (existing.shortCode) {
      this.shortCodeIndex.delete(existing.shortCode);
    }
    this.tokens.delete(token);
  }

  /**
   * Removes all expired tokens. Returns the count of removed tokens.
   */
  removeExpired(): number {
    let removed = 0;

    this.tokens.forEach((token, key) => {
      if (token.isExpired()) {
        if (token.shortCode) {
          this.shortCodeIndex.delete(token.shortCode);
        }
        this.tokens.delete(key);
        removed++;
      }
    });

    return removed;
  }

  /**
   * Looks up a token by full token first, then by short code.
   */
  private findToken(tokenOrCode: string): PairingToken | undefined {
    const byToken = this.tokens.get(tokenOrCode);
    if (byToken) {
      return byToken;
    }

    const fullToken = this.shortCodeIndex.get(tokenOrCode);
    if (!fullToken) {
      return undefined;
    }

    return this.tokens.get(fullToken);
  }
}
