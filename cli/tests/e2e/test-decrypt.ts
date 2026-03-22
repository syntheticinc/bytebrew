/**
 * Quick test: try to decrypt the real mobile payload using the SQLite shared secret.
 */
import { Database } from 'bun:sqlite';
import path from 'path';
import { CryptoService } from '../../src/infrastructure/mobile/CryptoService.js';

const DEVICE_ID = '428be1b8-c712-46d5-bb66-0c4f76a1c803';
const B64 = '0SUNP7MwJCVrxTVKno/7PAAAAAAAAAAAem5iqpvkkBDuWIkd/g6I8/OxA9w+ncO/r9osvp3kGtMNS6siUlOPiFHchqleghQt4YGa';

const dbPath = path.join(process.env.USERPROFILE ?? '~', '.bytebrew', 'bytebrew.db');
const db = new Database(dbPath, { readonly: true });
const row = db.query('SELECT shared_secret FROM paired_devices WHERE id = ?').get(DEVICE_ID) as { shared_secret: Buffer } | null;
db.close();

if (!row) {
  console.log('Device not found');
  process.exit(1);
}

const secret = row.shared_secret;
console.log('Secret type:', secret?.constructor?.name, 'length:', secret?.length);

const crypto = new CryptoService();
const ciphertext = Buffer.from(B64, 'base64');
console.log('Ciphertext length:', ciphertext.length);

try {
  const plaintext = crypto.decrypt(new Uint8Array(ciphertext), new Uint8Array(secret));
  console.log('DECRYPTED OK:', Buffer.from(plaintext).toString('utf8').slice(0, 500));
} catch (e) {
  console.error('DECRYPT FAILED:', e instanceof Error ? e.message : String(e));
}
