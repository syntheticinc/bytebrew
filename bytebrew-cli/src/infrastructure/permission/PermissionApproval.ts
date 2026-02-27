// PermissionApproval - handles user approval for permission requests
import { PermissionRequest } from '../../domain/permission/Permission.js';
import { getLogger } from '../../lib/logger.js';

export interface ApprovalResult {
  approved: boolean;
  remember: boolean;
}

export type ApprovalCallback = (request: PermissionRequest) => Promise<ApprovalResult>;

/** Behavior for 'ask' actions in headless mode */
export type HeadlessBehavior = 'block' | 'allow-once' | 'allow-remember';

/**
 * PermissionApproval - manages the approval flow for permission requests.
 *
 * In headless mode: behavior depends on headlessBehavior setting
 * In interactive mode: shows UI prompt via callback and waits for user response
 */
class PermissionApprovalClass {
  private mode: 'headless' | 'interactive' = 'headless';
  private headlessBehavior: HeadlessBehavior = 'block';
  private approvalCallback: ApprovalCallback | null = null;

  setHeadlessMode(behavior: HeadlessBehavior = 'block'): void {
    const logger = getLogger();
    logger.debug('PermissionApproval: set headless mode', { behavior });
    this.mode = 'headless';
    this.headlessBehavior = behavior;
    this.approvalCallback = null;
  }

  setInteractiveMode(callback: ApprovalCallback): void {
    const logger = getLogger();
    logger.debug('PermissionApproval: set interactive mode');
    this.mode = 'interactive';
    this.approvalCallback = callback;
  }

  isInteractive(): boolean {
    return this.mode === 'interactive';
  }

  getHeadlessBehavior(): HeadlessBehavior {
    return this.headlessBehavior;
  }

  async requestApproval(request: PermissionRequest): Promise<ApprovalResult> {
    const logger = getLogger();

    if (this.mode === 'headless') {
      switch (this.headlessBehavior) {
        case 'allow-once':
          logger.debug('PermissionApproval: allowing once (headless)', { type: request.type, value: request.value });
          return { approved: true, remember: false };

        case 'allow-remember':
          logger.debug('PermissionApproval: allowing and remembering (headless)', { type: request.type, value: request.value });
          return { approved: true, remember: true };

        case 'block':
        default:
          logger.debug('PermissionApproval: blocking (headless)', { type: request.type, value: request.value });
          return { approved: false, remember: false };
      }
    }

    if (!this.approvalCallback) {
      logger.warn('PermissionApproval: no callback set in interactive mode');
      return { approved: false, remember: false };
    }

    logger.debug('PermissionApproval: requesting approval', { type: request.type, value: request.value });
    return this.approvalCallback(request);
  }

  reset(): void {
    this.mode = 'headless';
    this.headlessBehavior = 'block';
    this.approvalCallback = null;
  }
}

// Singleton instance
export const PermissionApproval = new PermissionApprovalClass();
