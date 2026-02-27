// PermissionConfigLoader - loads and saves permission configuration
// Claude Code compatible format: allow/deny lists with "Bash(pattern)" syntax
import fs from 'fs/promises';
import path from 'path';
import { PermissionConfig } from '../../domain/permission/Permission.js';
import { getLogger } from '../../lib/logger.js';

const CONFIG_FILENAME = 'settings.local.json';
const BYTEBREW_DIR = '.bytebrew';

/** Default permission config (compact, Claude Code compatible) */
const DEFAULT_CONFIG: PermissionConfig = {
  permissions: {
    allow: [
      // Read-only tools
      'Read',
      'List',
      'Edit',
      // Navigation & info
      'Bash(cd *)',
      'Bash(ls)', 'Bash(ls *)',
      'Bash(dir)', 'Bash(dir *)',
      'Bash(pwd)',
      'Bash(which *)', 'Bash(where *)',
      'Bash(echo *)',
      // File viewing
      'Bash(cat *)', 'Bash(head *)', 'Bash(tail *)',
      'Bash(wc *)', 'Bash(find *)', 'Bash(grep *)', 'Bash(rg *)',
      // Package managers & build tools
      'Bash(npm *)', 'Bash(npx *)', 'Bash(yarn *)', 'Bash(pnpm *)', 'Bash(bun *)',
      'Bash(node *)', 'Bash(tsc *)', 'Bash(tsx *)',
      'Bash(go *)',
      'Bash(python *)', 'Bash(python3 *)', 'Bash(pip *)', 'Bash(pip3 *)',
      'Bash(cargo *)',
      // Testing
      'Bash(jest *)', 'Bash(vitest *)', 'Bash(mocha *)', 'Bash(pytest *)',
      'Bash(eslint *)', 'Bash(prettier *)',
      // Git read-only
      'Bash(git status)', 'Bash(git status *)',
      'Bash(git log)', 'Bash(git log *)',
      'Bash(git diff)', 'Bash(git diff *)',
      'Bash(git show *)',
      'Bash(git branch)', 'Bash(git branch -a)', 'Bash(git branch -v)',
      'Bash(git remote -v)',
      'Bash(git ls-files *)', 'Bash(git blame *)',
      // File operations (in project)
      'Bash(rm *)', 'Bash(mkdir *)', 'Bash(touch *)', 'Bash(mv *)', 'Bash(cp *)',
      // Environment
      'Bash(env)', 'Bash(printenv *)',
    ],
    deny: [
      // Destructive system operations
      'Bash(rm -rf /)', 'Bash(rm -rf /*)', 'Bash(rm -rf ~)', 'Bash(rm -rf ~/*)',
      'Bash(rm -rf C:/*)', 'Bash(rm -rf C:\\*)',
      'Bash(del /s /q C:/*)', 'Bash(del /s /q C:\\*)',
      // Privilege escalation
      'Bash(sudo *)', 'Bash(su *)',
      'Bash(chmod *)', 'Bash(chown *)',
      // Disk destruction
      'Bash(mkfs *)', 'Bash(dd if=*)', 'Bash(format *)', 'Bash(diskpart *)',
      // System control
      'Bash(shutdown *)', 'Bash(reboot *)',
      // Remote code execution
      'Bash(* | bash)', 'Bash(* | sh)', 'Bash(* | zsh)',
      'Bash(* | cmd)', 'Bash(* | powershell)',
      'Bash(curl * | *sh)', 'Bash(wget * | *sh)',
      // Git write operations
      'Bash(git push *)', 'Bash(git commit *)',
      'Bash(git reset *)', 'Bash(git checkout *)',
      'Bash(git rebase *)', 'Bash(git merge *)',
      'Bash(git cherry-pick *)', 'Bash(git revert *)',
      'Bash(git stash drop *)', 'Bash(git stash pop *)',
      'Bash(git clean *)',
      'Bash(git branch -d *)', 'Bash(git branch -D *)',
    ],
  },
};

/**
 * Load permission config from .bytebrew/settings.local.json
 * If not found, returns default config.
 */
export async function loadPermissionConfig(projectRoot: string): Promise<PermissionConfig> {
  const logger = getLogger();

  // Try project-level config
  const configPath = path.join(projectRoot, BYTEBREW_DIR, CONFIG_FILENAME);
  const config = await tryLoadConfig(configPath);

  if (config) {
    logger.debug('Loaded permission config', { path: configPath });
    return mergeWithDefaults(config);
  }

  // Return defaults
  logger.debug('Using default permission config');
  return deepCopyConfig(DEFAULT_CONFIG);
}

async function tryLoadConfig(configPath: string): Promise<Partial<PermissionConfig> | null> {
  const logger = getLogger();

  try {
    const content = await fs.readFile(configPath, 'utf-8');
    const json = JSON.parse(content);

    // Extract permissions field if it exists
    if (json.permissions) {
      return { permissions: json.permissions };
    }

    return null;
  } catch (error: any) {
    if (error.code !== 'ENOENT') {
      logger.warn('Failed to load permission config', { path: configPath, error: error.message });
    }
    return null;
  }
}

function mergeWithDefaults(partial: Partial<PermissionConfig>): PermissionConfig {
  const perms = partial.permissions;

  return {
    permissions: {
      allow: perms?.allow ?? DEFAULT_CONFIG.permissions.allow,
      deny: perms?.deny ?? DEFAULT_CONFIG.permissions.deny,
    },
  };
}

/**
 * Save permission config to .bytebrew/settings.local.json
 */
export async function savePermissionConfig(config: PermissionConfig, projectRoot: string): Promise<void> {
  const logger = getLogger();
  const bytebrewDir = path.join(projectRoot, BYTEBREW_DIR);
  const configPath = path.join(bytebrewDir, CONFIG_FILENAME);

  try {
    await fs.mkdir(bytebrewDir, { recursive: true });

    // Read existing config if present
    let existing: any = {};
    try {
      const content = await fs.readFile(configPath, 'utf-8');
      existing = JSON.parse(content);
    } catch {
      // File doesn't exist, start fresh
    }

    // Merge permissions into existing config
    existing.permissions = config.permissions;

    await fs.writeFile(configPath, JSON.stringify(existing, null, 2), 'utf-8');
    logger.debug('Saved permission config', { path: configPath });
  } catch (error: any) {
    logger.error('Failed to save permission config', { path: configPath, error: error.message });
    throw error;
  }
}

/**
 * Get the default config (useful for reference/testing)
 */
export function getDefaultPermissionConfig(): PermissionConfig {
  return deepCopyConfig(DEFAULT_CONFIG);
}

function deepCopyConfig(config: PermissionConfig): PermissionConfig {
  return {
    permissions: {
      allow: [...config.permissions.allow],
      deny: [...config.permissions.deny],
    },
  };
}
