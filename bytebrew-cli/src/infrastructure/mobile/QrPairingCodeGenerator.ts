import qrcode from 'qrcode-terminal';

/**
 * Compact QR payload — short keys to minimize QR version and fit terminal width.
 */
interface QrPayload {
  s: string;  // server_id
  t: string;  // token
  b: string;  // bridge_url
  k: string;  // server_public_key (base64) — verified by mobile out-of-band
}

/** Payload for local PairingService (Bridge mode, no gRPC server needed) */
export interface LocalPairingInfo {
  serverId: string;
  serverPublicKey: Uint8Array;
  token: string;
  shortCode: string;
}

/**
 * Render QR code for terminal using qrcode-terminal with { small: true }.
 *
 * IMPORTANT: output must go to stdout directly (process.stdout.write),
 * NOT through Ink's rendering pipeline which would reflow the text.
 */
export function renderQrForTerminal(data: string): string {
  let result = '';
  qrcode.generate(data, { small: true }, (qr: string) => {
    result = qr;
  });
  return result;
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
      s: info.serverId,
      t: info.token,
      b: bridgeUrl,
      k: Buffer.from(info.serverPublicKey).toString('base64'),
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
