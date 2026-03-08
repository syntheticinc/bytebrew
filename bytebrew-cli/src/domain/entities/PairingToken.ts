/**
 * PairingToken represents a temporary token for mobile device pairing.
 * Port from Go: bytebrew-srv/internal/domain/pairing_token.go
 *
 * Pure domain entity — no external dependencies.
 */

/** Default pairing token expiry: 5 minutes */
export const PAIRING_TOKEN_EXPIRY_MS = 15 * 60 * 1000;

export interface PairingTokenProps {
  readonly token: string;
  readonly shortCode: string;
  readonly expiresAt: Date;
  readonly used: boolean;
  readonly serverPublicKey: Uint8Array;
  readonly serverPrivateKey: Uint8Array;
}

export class PairingToken {
  readonly token: string;
  readonly shortCode: string;
  readonly expiresAt: Date;
  readonly used: boolean;
  readonly serverPublicKey: Uint8Array;
  readonly serverPrivateKey: Uint8Array;

  private constructor(props: PairingTokenProps) {
    this.token = props.token;
    this.shortCode = props.shortCode;
    this.expiresAt = props.expiresAt;
    this.used = props.used;
    this.serverPublicKey = props.serverPublicKey;
    this.serverPrivateKey = props.serverPrivateKey;
  }

  /**
   * Creates a new PairingToken with the given token and short code.
   * Expiry is set to 5 minutes from now.
   *
   * Crypto key generation is NOT done here (domain stays pure).
   * Use withKeys() to attach keys after generation in infrastructure layer.
   */
  static create(token: string, shortCode: string): PairingToken {
    if (!token) throw new Error('token is required');
    if (!shortCode) throw new Error('short_code is required');

    return new PairingToken({
      token,
      shortCode,
      expiresAt: new Date(Date.now() + PAIRING_TOKEN_EXPIRY_MS),
      used: false,
      serverPublicKey: new Uint8Array(0),
      serverPrivateKey: new Uint8Array(0),
    });
  }

  /**
   * Restores a PairingToken from persisted data (no validation).
   */
  static fromProps(props: PairingTokenProps): PairingToken {
    return new PairingToken(props);
  }

  /**
   * Returns a new PairingToken with server X25519 keys attached.
   */
  withKeys(serverPublicKey: Uint8Array, serverPrivateKey: Uint8Array): PairingToken {
    return new PairingToken({
      token: this.token,
      shortCode: this.shortCode,
      expiresAt: this.expiresAt,
      used: this.used,
      serverPublicKey,
      serverPrivateKey,
    });
  }

  /**
   * Returns true if the token has expired.
   */
  isExpired(): boolean {
    return Date.now() > this.expiresAt.getTime();
  }

  /**
   * Returns true if the token is not expired and not used.
   */
  isValid(): boolean {
    return !this.isExpired() && !this.used;
  }

  /**
   * Returns a new PairingToken marked as used.
   */
  markUsed(): PairingToken {
    return new PairingToken({
      token: this.token,
      shortCode: this.shortCode,
      expiresAt: this.expiresAt,
      used: true,
      serverPublicKey: this.serverPublicKey,
      serverPrivateKey: this.serverPrivateKey,
    });
  }
}
