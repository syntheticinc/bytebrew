import { AppConfig } from '../config/index.js';
import { BaseHeadlessRunner, HeadlessRunnerOptions } from './BaseHeadlessRunner.js';

/**
 * Single-question headless runner.
 * Sends one question, waits for response, then exits.
 */
export class HeadlessRunner extends BaseHeadlessRunner {
  private isComplete = false;

  constructor(config: AppConfig, options: HeadlessRunnerOptions = {}) {
    super(config, options);
  }

  /**
   * Run a single question and wait for complete response
   */
  async ask(question: string): Promise<void> {
    return new Promise(async (resolve, reject) => {
      try {
        const container = this.initContainer();
        const { streamGateway, streamProcessor, eventBus } = container;

        // Subscribe to connection status
        this.unsubscribers.push(
          streamGateway.onStatusChange((status) => {
            this.logStatus(status);
            if (status === 'error' || status === 'disconnected') {
              if (!this.isComplete) {
                this.cleanup();
              }
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
            this.isComplete = true;
            this.cleanup();
            resolve();
          })
        );

        // Setup tool display subscriptions
        this.setupToolEventSubscriptions();

        // Connect and send
        await this.connect();

        if (this.config.debug) {
          console.error(`[Sending] ${question}`);
        }
        streamProcessor.sendMessage(question);

      } catch (error) {
        this.cleanup();
        reject(error);
      }
    });
  }
}

/**
 * Run headless mode with a single question
 * @param config - App configuration
 * @param question - Question to ask
 * @param options - Headless runner options (unknownCommandBehavior, etc.)
 */
export async function runHeadless(
  config: AppConfig,
  question: string,
  options: HeadlessRunnerOptions = {}
): Promise<void> {
  const runner = new HeadlessRunner(config, options);

  try {
    await runner.ask(question);
  } catch (error) {
    const err = error as Error;
    console.error('Headless error:', err.message);
    if (config.debug && err.stack) {
      console.error(err.stack);
    }
    process.exit(1);
  }
}
