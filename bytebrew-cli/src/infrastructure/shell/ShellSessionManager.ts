import { ShellSession } from './ShellSession.js';
import { BackgroundProcessManager } from './BackgroundProcessManager.js';
import { getLogger } from '../../lib/logger.js';

const POOL_SIZE = 3;

/**
 * Manages shell session pools and background processes.
 * Pool of up to POOL_SIZE sessions per agent per project root.
 */
export class ShellSessionManager {
  private pools: Map<string, ShellSession[]> = new Map();
  private backgroundManager: BackgroundProcessManager = new BackgroundProcessManager();

  /**
   * Get first available (non-busy) session from agent's pool.
   * Creates new session if pool has room (< POOL_SIZE).
   * Returns null if all POOL_SIZE sessions are busy (no queue).
   */
  getAvailableSession(projectRoot: string, agentId?: string): ShellSession | null {
    const key = agentId ? `${projectRoot}::${agentId}` : projectRoot;
    let pool = this.pools.get(key);
    if (!pool) {
      pool = [];
      this.pools.set(key, pool);
    }

    // Find existing non-busy session
    for (const session of pool) {
      if (!session.isExecuting()) {
        return session;
      }
    }

    // Create new if pool not full (lazy allocation)
    if (pool.length < POOL_SIZE) {
      const session = new ShellSession({ cwd: projectRoot });
      pool.push(session);
      return session;
    }

    // All sessions busy, no queue
    return null;
  }

  /**
   * Get the BackgroundProcessManager (singleton).
   */
  getBackgroundManager(): BackgroundProcessManager {
    return this.backgroundManager;
  }

  /**
   * Dispose all sessions and background processes.
   */
  async disposeAll(): Promise<void> {
    const logger = getLogger();
    logger.debug('Disposing all shell sessions and background processes');

    for (const [, pool] of this.pools) {
      for (const session of pool) {
        session.destroy();
      }
    }
    this.pools.clear();

    await this.backgroundManager.disposeAll();
  }
}
