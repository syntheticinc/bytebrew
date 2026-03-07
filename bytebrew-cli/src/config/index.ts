// Configuration management
import fs from 'fs';
import path from 'path';

export interface AppConfig {
  serverAddress: string;
  projectKey: string;
  userId: string;
  projectRoot: string;
  sessionId?: string; // Optional: reuse specific session
  debug: boolean;
  bridgeAddress?: string; // Bridge relay address (e.g. "bridge.bytebrew.ai:443")
  bridgeEnabled: boolean; // Enable Mobile via Bridge (default: false)
  bridgeAuthToken?: string; // Auth token for Bridge registration
  serverId?: string; // UUID of this CLI instance for Bridge registration
}

export interface ConfigValidationError {
  field: string;
  message: string;
}

/**
 * Validate server address format
 */
function isValidServerAddress(address: string): boolean {
  // Format: host:port or ip:port
  const regex = /^[a-zA-Z0-9.-]+:\d{1,5}$/;
  if (!regex.test(address)) return false;

  const port = parseInt(address.split(':')[1], 10);
  return port > 0 && port <= 65535;
}

/**
 * Validate configuration and return errors if any
 */
export function validateConfig(config: AppConfig): ConfigValidationError[] {
  const errors: ConfigValidationError[] = [];

  // Validate server address
  if (!config.serverAddress) {
    errors.push({ field: 'serverAddress', message: 'Server address is required' });
  } else if (!isValidServerAddress(config.serverAddress)) {
    errors.push({
      field: 'serverAddress',
      message: `Invalid server address format: ${config.serverAddress}. Expected format: host:port`,
    });
  }

  // Validate project key
  if (!config.projectKey || config.projectKey.trim() === '') {
    errors.push({ field: 'projectKey', message: 'Project key is required' });
  }

  // Validate user ID
  if (!config.userId || config.userId.trim() === '') {
    errors.push({ field: 'userId', message: 'User ID is required' });
  }

  // Validate project root
  if (!config.projectRoot) {
    errors.push({ field: 'projectRoot', message: 'Project root is required' });
  } else {
    const resolvedPath = path.resolve(config.projectRoot);
    if (!fs.existsSync(resolvedPath)) {
      errors.push({
        field: 'projectRoot',
        message: `Project root does not exist: ${resolvedPath}`,
      });
    } else {
      const stats = fs.statSync(resolvedPath);
      if (!stats.isDirectory()) {
        errors.push({
          field: 'projectRoot',
          message: `Project root is not a directory: ${resolvedPath}`,
        });
      }
    }
  }

  return errors;
}

/**
 * Load and validate configuration
 * Throws if validation fails
 */
export function loadConfig(overrides: Partial<AppConfig> = {}): AppConfig {
  const config: AppConfig = {
    serverAddress: overrides.serverAddress || process.env.BYTEBREW_SERVER || '',
    projectKey: overrides.projectKey || process.env.BYTEBREW_PROJECT || 'default',
    userId: overrides.userId || process.env.BYTEBREW_USER || `cli-user-${process.pid}`,
    projectRoot: overrides.projectRoot || process.cwd(),
    sessionId: overrides.sessionId, // Optional: pass through if provided
    debug: overrides.debug ?? process.env.BYTEBREW_DEBUG === 'true',
    bridgeAddress: overrides.bridgeAddress || process.env.BYTEBREW_BRIDGE || undefined,
    bridgeEnabled: overrides.bridgeEnabled ?? false,
    bridgeAuthToken: overrides.bridgeAuthToken || process.env.BYTEBREW_BRIDGE_AUTH_TOKEN || undefined,
    serverId: overrides.serverId,
  };

  // Normalize project root to absolute path
  config.projectRoot = path.resolve(config.projectRoot);

  return config;
}

/**
 * Load and validate configuration, throwing descriptive errors
 */
export function loadAndValidateConfig(overrides: Partial<AppConfig> = {}): AppConfig {
  const config = loadConfig(overrides);
  const errors = validateConfig(config);

  if (errors.length > 0) {
    const errorMessages = errors.map((e) => `  - ${e.field}: ${e.message}`).join('\n');
    throw new Error(`Configuration validation failed:\n${errorMessages}`);
  }

  return config;
}
