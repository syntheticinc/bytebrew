/**
 * MobileCommandHandler routes commands from mobile devices to the CLI agent.
 * Port from Go: bytebrew-srv/internal/usecase/mobile_command/usecase.go
 *
 * Bridges the mobile UI with the same mechanisms used by the TUI:
 * - New tasks go through StreamProcessorService.sendMessage()
 * - Ask-user replies go through resolveAskUser()
 * - Cancel goes through StreamProcessorService.cancel()
 *
 * Consumer-side interfaces defined in this file (ISP).
 */

import { getLogger } from '../../lib/logger.js';
import type { QuestionAnswer } from '../../tools/askUser.js';

const logger = getLogger();

// --- Consumer-side interfaces ---

/** Sends messages and manages the agent stream (same as TUI keyboard input) */
interface MessageSender {
  sendMessage(content: string): void;
  cancel(): void;
  getIsProcessing(): boolean;
}

/** Resolves pending ask_user prompts from the server */
interface AskUserResolver {
  resolve(answers: QuestionAnswer[]): void;
}

// --- Service ---

export class MobileCommandHandler {
  private readonly messageSender: MessageSender;
  private readonly askUserResolver: AskUserResolver;

  constructor(
    messageSender: MessageSender,
    askUserResolver: AskUserResolver,
  ) {
    this.messageSender = messageSender;
    this.askUserResolver = askUserResolver;
  }

  /**
   * Handles a new task from mobile. Creates a message through the same
   * pipeline as TUI keyboard input (StreamProcessorService.sendMessage).
   */
  handleNewTask(deviceId: string, message: string): void {
    if (!deviceId) {
      throw new Error('device_id is required');
    }
    if (!message) {
      throw new Error('message is required');
    }

    logger.info('new task from mobile', { deviceId, messageLength: message.length });

    this.messageSender.sendMessage(message);
  }

  /**
   * Handles an ask_user reply from mobile. Resolves the pending
   * ask_user prompt exactly as the TUI AskUserPrompt does.
   */
  handleAskUserReply(sessionId: string, reply: string): void {
    if (!sessionId) {
      throw new Error('session_id is required');
    }
    if (!reply) {
      throw new Error('reply is required');
    }

    logger.info('ask_user reply from mobile', { sessionId });

    // Resolve with a single answer matching the ask_user protocol.
    // The question text is not available from mobile context, so we use
    // a placeholder. The server matches by position, not question text.
    this.askUserResolver.resolve([{ question: '', answer: reply }]);
  }

  /**
   * Cancels the current agent task. Uses the same cancel mechanism
   * as the TUI Escape key (StreamProcessorService.cancel).
   */
  handleCancel(sessionId: string): void {
    if (!sessionId) {
      throw new Error('session_id is required');
    }

    if (!this.messageSender.getIsProcessing()) {
      logger.info('cancel ignored — not processing', { sessionId });
      return;
    }

    logger.info('cancel from mobile', { sessionId });

    this.messageSender.cancel();
  }
}
