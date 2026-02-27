import * as fs from 'fs';
import * as path from 'path';
import { ByteBrewHome } from '../config/ByteBrewHome';

export class LicenseStorage {
  private filePath: string;

  constructor(filePath?: string) {
    this.filePath = filePath ?? ByteBrewHome.licenseFile();
  }

  save(jwt: string): void {
    const dir = path.dirname(this.filePath);
    fs.mkdirSync(dir, { recursive: true });

    fs.writeFileSync(this.filePath, jwt, 'utf-8');

    try {
      fs.chmodSync(this.filePath, 0o600);
    } catch {
      // chmod not supported on Windows — skip
    }
  }

  load(): string | null {
    try {
      const content = fs.readFileSync(this.filePath, 'utf-8');
      const trimmed = content.trim();
      return trimmed || null;
    } catch {
      return null;
    }
  }

  clear(): void {
    try {
      fs.unlinkSync(this.filePath);
    } catch {
      // File does not exist or already removed — ignore
    }
  }
}
