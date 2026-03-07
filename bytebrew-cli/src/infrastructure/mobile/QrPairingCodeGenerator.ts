import qrcode from 'qrcode-terminal';

interface QrPayload {
  server_id: string;
  server_public_key?: string;
  token: string;
  bridge_url: string;
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
      server_id: info.serverId,
      token: info.token,
      bridge_url: bridgeUrl,
    };

    if (info.serverPublicKey.length > 0) {
      payload.server_public_key = Buffer.from(info.serverPublicKey).toString('base64');
    }

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
