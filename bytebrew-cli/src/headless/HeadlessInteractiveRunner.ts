import { AppConfig } from '../config/index.js';
import { BaseHeadlessRunner, HeadlessRunnerOptions } from './BaseHeadlessRunner.js';

/**
 * Interactive headless runner - keeps connection open for multiple messages.
 * Reads from stdin, sends each line as a message, waits for response.
 */
export class HeadlessInteractiveRunner extends BaseHeadlessRunner {
  private responseResolver: (() => void) | null = null;

  constructor(config: AppConfig, options: HeadlessRunnerOptions = {}) {
    super(config, options);
  }

  async run(): Promise<void> {
    const container = this.initContainer();
    const { streamGateway, streamProcessor, eventBus } = container;

    // Subscribe to connection status
    this.unsubscribers.push(
      streamGateway.onStatusChange((status) => {
        this.logStatus(status);
        if (status === 'error' || status === 'disconnected') {
          console.error('[Disconnected]');
          process.exit(1);
        }
      })
    );

    // Subscribe to message events
    this.unsubscribers.push(
      eventBus.subscribe('MessageCompleted', () => {
        this.handleMessageCompleted();
      })
    );

    this.unsubscribers.push(
      eventBus.subscribe('ProcessingStopped', () => {
        if (this.responseResolver) {
          this.responseResolver();
          this.responseResolver = null;
        }
      })
    );

    // Setup tool display subscriptions
    this.setupToolEventSubscriptions();

    // Connect once
    await this.connect();
    console.log('[Connected - type messages, Ctrl+C to exit]');

    // Read stdin line by line
    const readline = await import('readline');
    const rl = readline.createInterface({
      input: process.stdin,
      output: process.stdout,
      prompt: '> ',
    });

    rl.prompt();

    rl.on('line', async (line) => {
      const trimmed = line.trim();
      if (!trimmed) {
        rl.prompt();
        return;
      }

      // Send message and wait for response
      streamProcessor.sendMessage(trimmed);

      await new Promise<void>((resolve) => {
        this.responseResolver = resolve;
      });

      rl.prompt();
    });

    rl.on('close', () => {
      this.cleanup();
      process.exit(0);
    });

    // Keep process alive
    await new Promise(() => {});
  }
}

/**
 * Run headless mode in interactive session (multiple messages)
 */
export async function runHeadlessInteractive(config: AppConfig): Promise<void> {
  const runner = new HeadlessInteractiveRunner(config);

  try {
    await runner.run();
  } catch (error) {
    const err = error as Error;
    console.error('Headless error:', err.message);
    if (config.debug && err.stack) {
      console.error(err.stack);
    }
    process.exit(1);
  }
}
