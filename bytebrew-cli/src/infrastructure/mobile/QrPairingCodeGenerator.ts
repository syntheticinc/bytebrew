import qrcode from 'qrcode-terminal';

interface QrPayload {
  bridge?: string;
  sid: string;
  spk: string;
  token: string;
}

/** Payload for local PairingService (Bridge mode, no gRPC server needed) */
export interface LocalPairingInfo {
  serverId: string;
  serverPublicKey: Uint8Array;
  token: string;
  shortCode: string;
}

export class QrPairingCodeGenerator {
  /**
   * Compose QR payload from local PairingService result (Bridge mode).
   * No LAN address — mobile connects exclusively via Bridge.
   */
  composeLocalPayload(params: {
    info: LocalPairingInfo;
    bridgeUrl: string;
  }): string {
    const { info, bridgeUrl } = params;

    const payload: QrPayload = {
      sid: info.serverId,
      spk: info.serverPublicKey.length > 0
        ? Buffer.from(info.serverPublicKey).toString('base64')
        : '',
      token: info.token,
      bridge: bridgeUrl,
    };

    return JSON.stringify(payload);
  }

  /** Display pairing info for local PairingService (Bridge mode). */
  displayLocalPairingInfo(params: {
    info: LocalPairingInfo;
    bridgeUrl: string;
  }): void {
    const payload = this.composeLocalPayload(params);

    qrcode.generate(payload, { small: true }, (qr: string) => {
      console.log(qr);
    });

    console.log(`Or enter code manually: ${params.info.shortCode}`);
    console.log(`Bridge: ${params.bridgeUrl}`);
    console.log('Scan with ByteBrew mobile app.');
  }
}
