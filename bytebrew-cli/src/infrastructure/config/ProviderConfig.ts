// Provider and model configuration stored in ~/.bytebrew/provider.json
// Uses JSON format to avoid adding a YAML dependency.
import * as fs from 'fs';
import * as path from 'path';
import { ByteBrewHome } from './ByteBrewHome.js';

export type ProviderMode = 'proxy' | 'byok' | 'auto';

export interface ProviderSettings {
  mode: ProviderMode;
  cloudApiUrl?: string;
}

export interface ModelsConfig {
  overrides: Record<string, string>; // role -> model
}

interface ConfigFile {
  provider?: Partial<ProviderSettings>;
  models?: Partial<ModelsConfig>;
}

const DEFAULT_PROVIDER: ProviderSettings = { mode: 'auto' };
const DEFAULT_MODELS: ModelsConfig = { overrides: {} };

function configPath(): string {
  return path.join(ByteBrewHome.dir(), 'provider.json');
}

function readConfigFile(): ConfigFile {
  try {
    const raw = fs.readFileSync(configPath(), 'utf-8');
    return JSON.parse(raw) as ConfigFile;
  } catch {
    return {};
  }
}

function writeConfigFile(config: ConfigFile): void {
  ByteBrewHome.ensureDir();
  fs.writeFileSync(configPath(), JSON.stringify(config, null, 2), 'utf-8');
}

export function isValidProviderMode(value: string): value is ProviderMode {
  return value === 'proxy' || value === 'byok' || value === 'auto';
}

export function readProviderConfig(): ProviderSettings {
  const config = readConfigFile();
  return {
    mode: config.provider?.mode ?? DEFAULT_PROVIDER.mode,
    cloudApiUrl: config.provider?.cloudApiUrl,
  };
}

export function writeProviderConfig(settings: Partial<ProviderSettings>): void {
  const config = readConfigFile();
  config.provider = { ...config.provider, ...settings };
  writeConfigFile(config);
}

export function readModelsConfig(): ModelsConfig {
  const config = readConfigFile();
  return {
    overrides: config.models?.overrides ?? { ...DEFAULT_MODELS.overrides },
  };
}

export function writeModelOverride(role: string, model: string): void {
  const config = readConfigFile();
  if (!config.models) {
    config.models = { overrides: {} };
  }
  if (!config.models.overrides) {
    config.models.overrides = {};
  }
  config.models.overrides[role] = model;
  writeConfigFile(config);
}

export function resetModelOverrides(): void {
  const config = readConfigFile();
  if (config.models) {
    config.models.overrides = {};
  }
  writeConfigFile(config);
}
