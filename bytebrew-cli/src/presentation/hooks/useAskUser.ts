// useAskUser hook - manages questionnaire dialog state and logic
import { useState, useEffect, useCallback } from 'react';
import { IEventBus } from '../../domain/ports/IEventBus.js';
import { setAskUserEventBus, resolveAskUser } from '../../tools/askUser.js';
import type { Question, QuestionAnswer } from '../../tools/askUser.js';

export interface UseAskUserOptions {
  eventBus: IEventBus;
}

export interface UseAskUserResult {
  questions: Question[] | null;
  handleComplete: (answers: QuestionAnswer[]) => void;
}

/**
 * Hook that manages ask_user questionnaire state.
 *
 * While active: QuestionnairePrompt renders the questions (Dynamic area).
 * On complete: questions disappear. The preamble text (LLM's streamed text before
 * the tool call) is already persisted by completeCurrentMessage in ToolExecutionHandler.
 */
export function useAskUser(options: UseAskUserOptions): UseAskUserResult {
  const { eventBus } = options;

  const [questions, setQuestions] = useState<Question[] | null>(null);

  useEffect(() => {
    setAskUserEventBus(eventBus);

    const unsubAskUser = eventBus.subscribe('AskUserRequested', (event) => {
      setQuestions(event.questions);
    });

    const unsubStopped = eventBus.subscribe('ProcessingStopped', () => {
      setQuestions(null);
    });

    const unsubResolved = eventBus.subscribe('AskUserResolved', () => {
      setQuestions(null);
    });

    return () => {
      unsubAskUser();
      unsubStopped();
      unsubResolved();
      setAskUserEventBus(null!);
    };
  }, [eventBus]);

  const handleComplete = useCallback((answers: QuestionAnswer[]) => {
    resolveAskUser(answers);
    setQuestions(null);
  }, []);

  return { questions, handleComplete };
}
