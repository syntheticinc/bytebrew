import * as fs from 'fs';
import * as path from 'path';
import { ByteBrewHome } from './ByteBrewHome.js';

export interface ByteBrewConfigData {
  bridge_url?: string;
}

export class ByteBrewConfig {
  private readonly filePath: string;

  constructor() {
    this.filePath = path.join(ByteBrewHome.dir(), 'config.json');
  }

  load(): ByteBrewConfigData {
    try {
      const raw = fs.readFileSync(this.filePath, 'utf-8');
      return JSON.parse(raw) as ByteBrewConfigData;
    } catch {
      return {};
    }
  }

  save(config: ByteBrewConfigData): void {
    ByteBrewHome.ensureDir();
    fs.writeFileSync(this.filePath, JSON.stringify(config, null, 2), 'utf-8');
  }

  getBridgeUrl(): string | undefined {
    return this.load().bridge_url;
  }

  setBridgeUrl(url: string): void {
    const config = this.load();
    config.bridge_url = url;
    this.save(config);
  }

  clearBridgeUrl(): void {
    const config = this.load();
    delete config.bridge_url;
    this.save(config);
  }
}
