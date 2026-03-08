/**
 * Diagnostic script: connects to bridge as server, receives encrypted messages
 * from device 0352b901, tries to decrypt with stored shared_secret.
 *
 * Usage: bun tests/e2e/diagnose-crypto.ts
 */
import { Database } from 'bun:sqlite';
import { homedir } from 'os';
import { join } from 'path';
import { xchacha20poly1305 } from '@noble/ciphers/chacha';

const DEVICE_ID = '0352b901-e863-41b2-9feb-5f8ccf48cee7';
const SERVER_ID = '10223b6a-01f7-4d53-aad5-b52dc956042d';
const BRIDGE_URL = 'wss://bridge.bytebrew.ai:443';
const BRIDGE_TOKEN = '5c1ffdc1280039d31b1ac71b731479ddda0e176e10bf137e9ccbc378ce770a2c';

// 1. Read shared_secret from SQLite
const dbPath = join(homedir(), '.bytebrew', 'bytebrew.db');
const db = new Database(dbPath, { readonly: true });
const device = db.query('SELECT shared_secret FROM paired_devices WHERE id = ?').get(DEVICE_ID) as { shared_secret: Uint8Array } | null;

if (!device) {
  console.error(`Device ${DEVICE_ID} not found in DB`);
  process.exit(1);
}

const sharedSecret = new Uint8Array(device.shared_secret);
console.log(`Shared secret loaded: ${sharedSecret.length} bytes, hex=${Buffer.from(sharedSecret).toString('hex').slice(0, 16)}...`);
db.close();

// 2. Connect to bridge as server
const registerUrl = `${BRIDGE_URL}/register?server_id=${SERVER_ID}&token=${BRIDGE_TOKEN}`;
console.log(`Connecting to bridge: ${registerUrl}`);

const ws = new WebSocket(registerUrl);

ws.addEventListener('open', () => {
  console.log('Connected to bridge as server');
});

ws.addEventListener('message', (event) => {
  try {
    const raw = typeof event.data === 'string' ? event.data : Buffer.from(event.data as ArrayBuffer).toString('utf-8');
    const envelope = JSON.parse(raw);

    console.log(`\n--- Message from bridge ---`);
    console.log(`  type: ${envelope.type}`);
    console.log(`  device_id: ${envelope.device_id}`);

    if (envelope.type === 'data' && envelope.device_id === DEVICE_ID) {
      const payload = envelope.payload;
      console.log(`  payload type: ${typeof payload}`);
      console.log(`  payload length: ${typeof payload === 'string' ? payload.length : JSON.stringify(payload).length}`);

      if (typeof payload === 'string') {
        // Try to decrypt
        try {
          const ciphertext = Buffer.from(payload, 'base64');
          console.log(`  ciphertext bytes: ${ciphertext.length}`);

          const NONCE_SIZE = 24;
          const nonce = ciphertext.slice(0, NONCE_SIZE);
          const encrypted = ciphertext.slice(NONCE_SIZE);

          console.log(`  nonce hex: ${Buffer.from(nonce).toString('hex')}`);

          const cipher = xchacha20poly1305(sharedSecret, new Uint8Array(nonce));
          const plaintext = cipher.decrypt(new Uint8Array(encrypted));
          const jsonStr = new TextDecoder().decode(plaintext);

          console.log(`  ✅ DECRYPT SUCCESS: ${jsonStr}`);

          // Send encrypted pong response
          const pongMsg = JSON.stringify({
            type: 'pong',
            request_id: JSON.parse(jsonStr).request_id,
            device_id: DEVICE_ID,
            payload: {},
          });
          const pongBytes = new TextEncoder().encode(pongMsg);

          // Encrypt pong
          const pongNonce = new Uint8Array(24);
          crypto.getRandomValues(pongNonce.subarray(0, 16));
          const pongCipher = xchacha20poly1305(sharedSecret, pongNonce);
          const pongEncrypted = pongCipher.encrypt(pongBytes);
          const pongResult = new Uint8Array(24 + pongEncrypted.length);
          pongResult.set(pongNonce, 0);
          pongResult.set(pongEncrypted, 24);
          const pongB64 = Buffer.from(pongResult).toString('base64');

          // Send via bridge
          const bridgeMsg = JSON.stringify({
            type: 'data',
            device_id: DEVICE_ID,
            payload: pongB64,
          });
          ws.send(bridgeMsg);
          console.log(`  ✅ PONG SENT (encrypted)`);
        } catch (decryptErr) {
          console.log(`  ❌ DECRYPT FAILED: ${decryptErr}`);
          console.log(`  → Key mismatch! Flutter has different shared_secret than CLI DB.`);
          console.log(`  → Solution: delete device from CLI DB and re-pair.`);
        }
      } else {
        console.log(`  Plaintext message: ${JSON.stringify(payload)}`);
      }
    } else if (envelope.type === 'device_connected') {
      console.log(`  Device connected: ${envelope.device_id}`);
    } else if (envelope.type === 'device_disconnected') {
      console.log(`  Device disconnected: ${envelope.device_id}`);
    }
  } catch (err) {
    console.log(`Parse error: ${err}`);
    console.log(`Raw: ${event.data}`);
  }
});

ws.addEventListener('error', (event) => {
  console.error('WS error:', event);
});

ws.addEventListener('close', (event) => {
  console.log(`WS closed: code=${event.code} reason=${event.reason}`);
});

// Keep alive
setInterval(() => {}, 1000);

console.log('Waiting for messages from Flutter device...');
console.log('Press Ctrl+C to stop.');
