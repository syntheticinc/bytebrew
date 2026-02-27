// PermissionService - coordinates permission evaluation and approval
import {
  PermissionConfig,
  PermissionRequest,
  PermissionCheckResult,
} from '../../domain/permission/Permission.js';
import { evaluatePermission } from '../../domain/permission/PermissionEvaluator.js';
import { PermissionApproval } from '../../infrastructure/permission/PermissionApproval.js';
import {
  loadPermissionConfig,
  savePermissionConfig,
} from '../../infrastructure/permission/PermissionConfigLoader.js';
import { getLogger } from '../../lib/logger.js';

export class PermissionService {
  private projectRoot: string;
  private configPromise: Promise<PermissionConfig> | null = null;

  constructor(projectRoot: string) {
    this.projectRoot = projectRoot;
  }

  private async getConfig(): Promise<PermissionConfig> {
    if (!this.configPromise) {
      this.configPromise = loadPermissionConfig(this.projectRoot);
    }
    return this.configPromise;
  }

  /** Reset cached config (after config changes) */
  resetConfig(): void {
    this.configPromise = null;
  }

  /**
   * Check permission for a request.
   * 1. Evaluate against config rules (deny first-match → allow first-match → ask)
   * 2. If result is "allow" → allowed
   * 3. If result is "deny" → blocked
   * 4. If result is "ask":
   *    - Auto-approve for code agents
   *    - Otherwise prompt user via PermissionApproval
   *    - If user approves + remember → add rule to config
   */
  async check(request: PermissionRequest): Promise<PermissionCheckResult> {
    const logger = getLogger();
    const config = await this.getConfig();

    const evalResult = evaluatePermission(request, config);
    logger.debug('Permission evaluated', {
      type: request.type,
      value: request.value,
      action: evalResult.action,
      matchedPattern: evalResult.matchedPattern,
    });

    if (evalResult.action === 'allow') {
      return { allowed: true };
    }

    if (evalResult.action === 'deny') {
      const reason = evalResult.matchedPattern
        ? `Blocked by deny rule: ${evalResult.matchedPattern}`
        : `${request.type} operations are denied by policy`;
      return { allowed: false, reason };
    }

    // action === 'ask' → auto-approve for code agents
    if (evalResult.action === 'ask' && request.agentId?.startsWith('code-agent-')) {
      logger.info('Auto-approving for code agent', {
        agentId: request.agentId,
        type: request.type,
        value: request.value,
      });
      return { allowed: true };
    }

    // action === 'ask' → prompt user
    const approvalResult = await PermissionApproval.requestApproval(request);

    if (!approvalResult.approved) {
      // User rejected — optionally remember as deny rule
      if (approvalResult.remember) {
        try {
          const currentConfig = await this.getConfig();
          const rule = buildPermissionRule(request);
          if (rule) {
            logger.info('Adding permission rule to deny list', { rule });
            currentConfig.permissions.deny.push(rule);
            await savePermissionConfig(currentConfig, this.projectRoot);
            this.resetConfig();
          }
        } catch (err) {
          logger.warn('Failed to save deny rule', { error: (err as Error).message });
        }
      }
      return { allowed: false, reason: 'Rejected by user' };
    }

    // User approved — optionally remember as allow rule
    if (approvalResult.remember) {
      try {
        const currentConfig = await this.getConfig();
        const rule = buildPermissionRule(request);

        if (rule) {
          logger.info('Adding permission rule to allow list', { rule });
          currentConfig.permissions.allow.push(rule);
          await savePermissionConfig(currentConfig, this.projectRoot);
          this.resetConfig();
        }
      } catch (err) {
        logger.warn('Failed to save permission rule', { error: (err as Error).message });
      }
    }

    return { allowed: true };
  }
}

/**
 * Build a permission rule string from a request.
 * Returns rule in Claude Code format: "Bash(pattern)", "Read", etc.
 */
function buildPermissionRule(request: PermissionRequest): string | null {
  if (request.type === 'bash') {
    // Generate pattern from command
    const pattern = generateBashPattern(request.value);
    return `Bash(${pattern})`;
  }

  if (request.type === 'read') {
    return 'Read';
  }

  if (request.type === 'edit') {
    return 'Edit';
  }

  if (request.type === 'list') {
    return 'List';
  }

  return null;
}

/**
 * Generate a wildcard pattern from a bash command.
 * Used when user approves a command and wants to remember it.
 *
 * Examples:
 * - "make build" -> "make *"
 * - "docker compose up" -> "docker compose *"
 */
function generateBashPattern(command: string): string {
  const parts = command.trim().split(/\s+/);

  if (parts.length <= 2) {
    return `${parts[0]} *`;
  }

  return `${parts[0]} ${parts[1]} *`;
}
