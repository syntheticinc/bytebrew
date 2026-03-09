// Reconnection manager with exponential backoff
import { EventEmitter } from 'events';
import {
  MAX_RECONNECT_ATTEMPTS,
  INITIAL_RECONNECT_DELAY_MS,
  MAX_RECONNECT_DELAY_MS,
} from '../../domain/connection.js';

export interface ReconnectionManagerOptions {
  maxAttempts?: number;
  initialDelayMs?: number;
  maxDelayMs?: number;
  onReconnect: () => Promise<void>;
  onMaxAttemptsReached: () => void;
  onAttempt: (attempt: number, delayMs: number) => void;
}

export class ReconnectionManager extends EventEmitter {
  private options: Required<ReconnectionManagerOptions>;
  private attempts: number = 0;
  private isReconnecting: boolean = false;
  private timeoutHandle: NodeJS.Timeout | null = null;

  constructor(options: ReconnectionManagerOptions) {
    super();
    this.options = {
      maxAttempts: options.maxAttempts ?? MAX_RECONNECT_ATTEMPTS,
      initialDelayMs: options.initialDelayMs ?? INITIAL_RECONNECT_DELAY_MS,
      maxDelayMs: options.maxDelayMs ?? MAX_RECONNECT_DELAY_MS,
      onReconnect: options.onReconnect,
      onMaxAttemptsReached: options.onMaxAttemptsReached,
      onAttempt: options.onAttempt,
    };
  }

  /**
   * Calculate delay for current attempt using exponential backoff
   */
  private calculateDelay(): number {
    const delay = Math.min(
      this.options.initialDelayMs * Math.pow(2, this.attempts),
      this.options.maxDelayMs
    );
    // Add some jitter (±10%)
    const jitter = delay * 0.1 * (Math.random() * 2 - 1);
    return Math.floor(delay + jitter);
  }

  /**
   * Start reconnection process
   */
  async startReconnection(): Promise<void> {
    if (this.isReconnecting) return;

    this.isReconnecting = true;
    this.attempts = 0;

    await this.attemptReconnection();
  }

  /**
   * Attempt a single reconnection
   */
  private async attemptReconnection(): Promise<void> {
    if (this.attempts >= this.options.maxAttempts) {
      this.isReconnecting = false;
      this.options.onMaxAttemptsReached();
      return;
    }

    this.attempts++;
    const delay = this.calculateDelay();

    this.options.onAttempt(this.attempts, delay);

    this.timeoutHandle = setTimeout(async () => {
      try {
        await this.options.onReconnect();
        // Reconnection successful
        this.isReconnecting = false;
        this.attempts = 0;
        this.emit('reconnected');
      } catch (error) {
        // Reconnection failed, try again
        await this.attemptReconnection();
      }
    }, delay);
  }

  /**
   * Stop reconnection attempts
   */
  stop(): void {
    this.isReconnecting = false;
    if (this.timeoutHandle) {
      clearTimeout(this.timeoutHandle);
      this.timeoutHandle = null;
    }
  }

  /**
   * Reset the manager
   */
  reset(): void {
    this.stop();
    this.attempts = 0;
  }

  /**
   * Get current attempt count
   */
  getAttempts(): number {
    return this.attempts;
  }

  /**
   * Check if currently reconnecting
   */
  getIsReconnecting(): boolean {
    return this.isReconnecting;
  }
}
