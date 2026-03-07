// AskUser tool - handles ask_user tool calls from the server proxy
// In headless mode: auto-selects first option or default
// In interactive mode: publishes event via EventBus and waits for user response
import { Tool, ToolResult } from './registry.js';
import type { IEventBus } from '../domain/ports/IEventBus.js';

// --- Questionnaire types ---

export interface QuestionOption {
  label: string;
  description?: string;
}

export interface Question {
  text: string;
  options?: QuestionOption[];
  default?: string;
}

export interface QuestionAnswer {
  question: string;
  answer: string;
}

/**
 * Callback type for interactive ask_user prompts.
 * Receives array of questions and returns array of answers.
 */
export type AskUserCallback = (questions: Question[]) => Promise<QuestionAnswer[]>;

/**
 * AskUserTool handles ask_user tool calls from the server.
 * The server sends ask_user as a proxied TOOL_CALL when
 * the Supervisor agent needs user input (e.g., task approval).
 *
 * The server now sends a `questions` JSON field containing an array of Question objects.
 */
export class AskUserTool implements Tool {
  readonly name = 'ask_user';
  private headlessMode: boolean;
  private interactiveCallback: AskUserCallback | null;

  constructor(headlessMode = false, interactiveCallback?: AskUserCallback) {
    this.headlessMode = headlessMode;
    this.interactiveCallback = interactiveCallback || null;
  }

  async execute(args: Record<string, string>): Promise<ToolResult> {
    const questions = this.parseQuestions(args);

    if (this.headlessMode) {
      return this.executeHeadless(questions);
    }

    if (this.interactiveCallback) {
      const answers = await this.interactiveCallback(questions);
      return {
        result: JSON.stringify(answers),
        summary: summarizeAnswers(answers),
      };
    }

    // Fallback: auto-answer (no interactive callback registered)
    return this.executeHeadless(questions);
  }

  private parseQuestions(args: Record<string, string>): Question[] {
    const questionsStr = args.questions;
    if (questionsStr) {
      try {
        const parsed = JSON.parse(questionsStr);
        if (Array.isArray(parsed) && parsed.length > 0) {
          return parsed;
        }
      } catch {
        // Fall through to legacy format
      }
    }

    // Legacy single-question format fallback
    const question = args.question || 'No question provided';
    const defaultAnswer = args.default_answer || undefined;
    return [{ text: question, default: defaultAnswer }];
  }

  private executeHeadless(questions: Question[]): ToolResult {
    const answers: QuestionAnswer[] = questions.map((q) => {
      const answer = q.options?.[0]?.label ?? q.default ?? 'approved';
      console.log(`\n[Ask User] ${q.text}`);
      console.log(`[Auto-answer: ${answer}]`);
      return { question: q.text, answer };
    });

    return {
      result: JSON.stringify(answers),
      summary: summarizeAnswers(answers),
    };
  }
}

/** Summarize answers for tool result summary. Truncates long answers to 25 chars. */
function summarizeAnswers(answers: QuestionAnswer[]): string {
  if (answers.length === 1) {
    const text = answers[0].answer;
    if (text.length <= 25) return text;
    return text.slice(0, 24) + '…';
  }
  return `${answers.length} answers`;
}

// --- Event-driven ask_user for interactive mode ---

// Stores the Promise resolve function for the current pending ask_user
let pendingResolve: ((answers: QuestionAnswer[]) => void) | null = null;

// EventBus reference set by the UI layer after container creation
let eventBusRef: IEventBus | null = null;

/**
 * Set the EventBus reference for ask_user event publishing.
 * Must be called after container creation (e.g., in ChatApp mount).
 */
export function setAskUserEventBus(eventBus: IEventBus): void {
  eventBusRef = eventBus;
}

/**
 * Create an interactive callback that publishes AskUserRequested events
 * via EventBus for instant UI notification (no polling).
 */
export function createInteractiveAskUserCallback(): AskUserCallback {
  return (questions: Question[]) => {
    return new Promise<QuestionAnswer[]>((resolve) => {
      pendingResolve = resolve;
      // Publish event for UI to react instantly
      if (eventBusRef) {
        eventBusRef.publish({
          type: 'AskUserRequested',
          questions,
        });
      }
    });
  };
}

/**
 * Resolve the pending ask_user prompt with user's answers.
 * Called by the UI when the user completes the questionnaire.
 */
export function resolveAskUser(answers: QuestionAnswer[]): void {
  if (pendingResolve) {
    pendingResolve(answers);
    pendingResolve = null;
    // Notify UI to clear questions immediately (important for mobile-originated answers
    // where handleComplete() is not called and ProcessingStopped may arrive late)
    if (eventBusRef) {
      eventBusRef.publish({ type: 'AskUserResolved' });
    }
  }
}
