import * as fs from 'fs';
import * as path from 'path';
import { ByteBrewHome } from './ByteBrewHome.js';

export interface ByteBrewConfigData {
  // Reserved for future config options
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
}
