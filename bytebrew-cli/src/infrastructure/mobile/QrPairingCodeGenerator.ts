import * as os from 'os';
import qrcode from 'qrcode-terminal';
import type { GeneratePairingTokenResponse } from '../grpc/mobile_client.js';

interface QrPayload {
  lan?: string;
  bridge?: string;
  sid: string;
  spk: string;
  token: string;
}

export class QrPairingCodeGenerator {
  /** Skip virtual/VPN interfaces that phones can't reach. */
  private static readonly VIRTUAL_IFACE_PATTERNS = [
    /vpn/i, /vethernet/i, /virtual/i, /hyper-v/i,
    /wsl/i, /docker/i, /br-/i, /vbox/i, /vmnet/i,
  ];

  /** Detect LAN IPv4 address using os.networkInterfaces(). */
  detectLanIp(): string | undefined {
    const interfaces = os.networkInterfaces();
    const candidates: { name: string; address: string }[] = [];

    for (const [name, entries] of Object.entries(interfaces)) {
      if (!entries) continue;
      if (QrPairingCodeGenerator.VIRTUAL_IFACE_PATTERNS.some(p => p.test(name))) continue;
      for (const entry of entries) {
        if (entry.family !== 'IPv4' || entry.internal) continue;
        if (entry.address.startsWith('169.254.')) continue;
        candidates.push({ name, address: entry.address });
      }
    }

    if (candidates.length === 0) return undefined;

    // Prefer 192.168.x.x (most common home/office LAN).
    const preferred = candidates.find(c => c.address.startsWith('192.168.'));
    return (preferred ?? candidates[0]).address;
  }

  /** Compose QR payload JSON from server response + bridge config + LAN IP. */
  composePayload(params: {
    response: GeneratePairingTokenResponse;
    bridgeUrl?: string;
  }): string {
    const { response, bridgeUrl } = params;
    const lanIp = this.detectLanIp();

    const payload: QrPayload = {
      sid: response.serverId,
      spk: response.serverPublicKey
        ? Buffer.from(response.serverPublicKey).toString('base64')
        : '',
      token: response.token,
    };

    if (lanIp) {
      payload.lan = `${lanIp}:${response.serverPort}`;
    }
    if (bridgeUrl) {
      payload.bridge = bridgeUrl;
    }

    return JSON.stringify(payload);
  }

  /** Display full pairing info: QR code + short code + instructions. */
  displayPairingInfo(params: {
    response: GeneratePairingTokenResponse;
    bridgeUrl?: string;
  }): void {
    const payload = this.composePayload(params);

    // Note: qrcode-terminal callback is synchronous despite callback API
    qrcode.generate(payload, { small: true }, (qr: string) => {
      console.log(qr);
    });

    console.log(`Or enter code manually: ${params.response.shortCode}`);
    console.log('Scan with ByteBrew mobile app.');
  }
}
