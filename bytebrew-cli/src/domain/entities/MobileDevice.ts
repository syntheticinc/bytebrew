/**
 * MobileDevice represents a paired mobile device.
 * Port from Go: bytebrew-srv/internal/domain/mobile_device.go
 *
 * Pure domain entity — no external dependencies.
 */

export interface MobileDeviceProps {
  readonly id: string;
  readonly name: string;
  readonly deviceToken: string;
  readonly publicKey: Uint8Array;
  readonly sharedSecret: Uint8Array;
  readonly pairedAt: Date;
  readonly lastSeenAt: Date;
}

export class MobileDevice {
  readonly id: string;
  readonly name: string;
  readonly deviceToken: string;
  readonly publicKey: Uint8Array;
  readonly sharedSecret: Uint8Array;
  readonly pairedAt: Date;
  readonly lastSeenAt: Date;

  private constructor(props: MobileDeviceProps) {
    this.id = props.id;
    this.name = props.name;
    this.deviceToken = props.deviceToken;
    this.publicKey = props.publicKey;
    this.sharedSecret = props.sharedSecret;
    this.pairedAt = props.pairedAt;
    this.lastSeenAt = props.lastSeenAt;
  }

  /**
   * Creates a new MobileDevice with validation.
   * Sets pairedAt and lastSeenAt to current time.
   */
  static create(id: string, name: string, deviceToken: string): MobileDevice {
    if (!id) throw new Error('id is required');
    if (!name) throw new Error('name is required');
    if (!deviceToken) throw new Error('device_token is required');

    const now = new Date();
    return new MobileDevice({
      id,
      name,
      deviceToken,
      publicKey: new Uint8Array(0),
      sharedSecret: new Uint8Array(0),
      pairedAt: now,
      lastSeenAt: now,
    });
  }

  /**
   * Restores a MobileDevice from persisted data (no validation).
   */
  static fromProps(props: MobileDeviceProps): MobileDevice {
    return new MobileDevice(props);
  }

  /**
   * Returns a new MobileDevice with crypto keys set.
   */
  withKeys(publicKey: Uint8Array, sharedSecret: Uint8Array): MobileDevice {
    return new MobileDevice({
      id: this.id,
      name: this.name,
      deviceToken: this.deviceToken,
      publicKey,
      sharedSecret,
      pairedAt: this.pairedAt,
      lastSeenAt: this.lastSeenAt,
    });
  }

  /**
   * Returns a new MobileDevice with lastSeenAt updated to now.
   */
  withUpdatedLastSeen(): MobileDevice {
    return new MobileDevice({
      id: this.id,
      name: this.name,
      deviceToken: this.deviceToken,
      publicKey: this.publicKey,
      sharedSecret: this.sharedSecret,
      pairedAt: this.pairedAt,
      lastSeenAt: new Date(),
    });
  }

  /**
   * Validates that all required fields are present.
   */
  validate(): void {
    if (!this.id) throw new Error('id is required');
    if (!this.name) throw new Error('name is required');
    if (!this.deviceToken) throw new Error('device_token is required');
    if (this.pairedAt.getTime() === 0) throw new Error('pairedAt is required');
  }
}
