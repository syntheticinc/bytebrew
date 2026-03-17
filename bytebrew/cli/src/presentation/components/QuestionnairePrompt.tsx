// QuestionnairePrompt - renders a multi-question questionnaire with options and freetext input.
// Replaces AskUserPrompt with support for 1-5 questions, each with optional answer choices.
import React, { useState, useRef, useCallback } from 'react';
import { Box, Text, useInput } from 'ink';
import type { Question, QuestionAnswer } from '../../tools/askUser.js';

export interface QuestionnairePromptProps {
  questions: Question[];
  onComplete: (answers: QuestionAnswer[]) => void;
}

type InputMode = 'options' | 'freetext';

/**
 * Renders questions one at a time with optional selectable options.
 *
 * For questions with options:
 * - Arrow keys navigate options, number keys 1-5 for quick select
 * - Tab switches between options mode and freetext mode
 * - Enter confirms selected option or typed text
 *
 * For questions without options:
 * - Text input with optional default value
 * - Enter confirms (empty input uses default if available)
 *
 * Esc at any point: completes with current answers + "cancelled" for remaining.
 */
export const QuestionnairePrompt: React.FC<QuestionnairePromptProps> = ({
  questions,
  onComplete,
}) => {
  const [currentIndex, setCurrentIndex] = useState(0);
  const [answers, setAnswers] = useState<QuestionAnswer[]>([]);
  const [selectedOptionIndex, setSelectedOptionIndex] = useState(0);
  const [mode, setMode] = useState<InputMode>('options');
  const [freeText, setFreeText] = useState('');
  const submittedRef = useRef(false);

  const currentQuestion = questions[currentIndex];
  const hasOptions = !!currentQuestion?.options && currentQuestion.options.length > 0;
  const totalOptions = currentQuestion?.options?.length ?? 0;

  const advanceToNext = useCallback(
    (answer: string) => {
      const newAnswers = [...answers, { question: currentQuestion.text, answer }];

      if (currentIndex + 1 >= questions.length) {
        // Last question answered
        submittedRef.current = true;
        onComplete(newAnswers);
      } else {
        setAnswers(newAnswers);
        setCurrentIndex(currentIndex + 1);
        setSelectedOptionIndex(0);
        setMode('options');
        setFreeText('');
      }
    },
    [answers, currentIndex, currentQuestion, questions.length, onComplete]
  );

  const cancelAll = useCallback(() => {
    if (submittedRef.current) return;
    submittedRef.current = true;

    // Current answers + "cancelled" for remaining questions
    const remaining = questions.slice(currentIndex).map((q) => ({
      question: q.text,
      answer: 'cancelled',
    }));
    onComplete([...answers, ...remaining]);
  }, [answers, currentIndex, questions, onComplete]);

  useInput((input, key) => {
    if (submittedRef.current) return;

    // Escape cancels the entire questionnaire
    if (key.escape) {
      cancelAll();
      return;
    }

    // Questions without options: freetext only
    if (!hasOptions) {
      if (key.return) {
        const trimmed = freeText.trim();
        if (trimmed) {
          advanceToNext(trimmed);
        } else if (currentQuestion.default) {
          advanceToNext(currentQuestion.default);
        }
        // No default and no text -> ignore Enter
        return;
      }
      if (key.backspace || key.delete) {
        setFreeText((prev) => prev.slice(0, -1));
        return;
      }
      if (input && !key.ctrl && !key.meta) {
        setFreeText((prev) => prev + input);
      }
      return;
    }

    // Questions WITH options

    // Tab toggles between options and freetext mode
    if (key.tab) {
      setMode((prev) => (prev === 'options' ? 'freetext' : 'options'));
      return;
    }

    if (mode === 'freetext') {
      if (key.return) {
        const trimmed = freeText.trim();
        if (trimmed) {
          advanceToNext(trimmed);
        }
        return;
      }
      if (key.backspace || key.delete) {
        setFreeText((prev) => prev.slice(0, -1));
        return;
      }
      if (input && !key.ctrl && !key.meta) {
        setFreeText((prev) => prev + input);
      }
      return;
    }

    // Options mode
    // Number keys 1-5 for quick select
    const numKey = parseInt(input, 10);
    if (numKey >= 1 && numKey <= 5 && numKey <= totalOptions) {
      advanceToNext(currentQuestion.options![numKey - 1].label);
      return;
    }

    // Arrow navigation
    if (key.upArrow) {
      setSelectedOptionIndex((prev) => (prev > 0 ? prev - 1 : prev));
      return;
    }
    if (key.downArrow) {
      setSelectedOptionIndex((prev) =>
        prev < totalOptions - 1 ? prev + 1 : prev
      );
      return;
    }

    // Enter confirms currently selected option
    if (key.return) {
      const option = currentQuestion.options![selectedOptionIndex];
      advanceToNext(option.label);
      return;
    }
  });

  if (!currentQuestion) {
    return null;
  }

  const questionLabel = questions.length > 1
    ? `Question ${currentIndex + 1}/${questions.length}: ${currentQuestion.text}`
    : currentQuestion.text;

  // Render for questions without options (pure freetext)
  if (!hasOptions) {
    const placeholder = currentQuestion.default
      ? `Your answer (Enter = ${currentQuestion.default})`
      : 'Type your answer...';

    return (
      <Box flexDirection="column" borderStyle="round" borderColor="cyan" paddingX={1} marginY={1}>
        <Box marginBottom={1}>
          <Text color="cyan" bold>{questionLabel}</Text>
        </Box>

        <Box borderStyle="single" borderColor="cyan" paddingX={1}>
          <Text>
            {freeText || <Text dimColor>{placeholder}</Text>}
            <Text color="cyan">{'\u258c'}</Text>
          </Text>
        </Box>

        <Box marginTop={1}>
          <Text dimColor>Enter to confirm | Esc to cancel</Text>
        </Box>
      </Box>
    );
  }

  // Render for questions with options
  return (
    <Box flexDirection="column" borderStyle="round" borderColor="cyan" paddingX={1} marginY={1}>
      <Box marginBottom={1}>
        <Text color="cyan" bold>{questionLabel}</Text>
      </Box>

      {mode === 'options' ? (
        <>
          {currentQuestion.options!.map((opt, i) => {
            const isSelected = selectedOptionIndex === i;
            return (
              <Box key={i}>
                <Text color={isSelected ? 'cyan' : 'white'}>
                  {isSelected ? '> ' : '  '}
                  [{i + 1}] {opt.label}
                  {opt.description ? <Text dimColor> - {opt.description}</Text> : null}
                </Text>
              </Box>
            );
          })}
          <Box marginTop={1}>
            <Text dimColor>
              {'\u2191\u2193'} navigate | 1-{Math.min(totalOptions, 5)} quick select | Enter confirm | Tab custom answer | Esc cancel
            </Text>
          </Box>
        </>
      ) : (
        <>
          <Box borderStyle="single" borderColor="yellow" paddingX={1}>
            <Text>
              {freeText || <Text dimColor>Type custom answer...</Text>}
              <Text color="yellow">{'\u258c'}</Text>
            </Text>
          </Box>
          <Box marginTop={1}>
            <Text dimColor>Enter to confirm | Tab back to options | Esc cancel</Text>
          </Box>
        </>
      )}
    </Box>
  );
};
