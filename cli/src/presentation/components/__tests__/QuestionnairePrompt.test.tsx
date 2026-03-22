import { describe, it, expect, mock, afterEach } from 'bun:test';
import React from 'react';
import { render } from 'ink-testing-library';
import { QuestionnairePrompt } from '../QuestionnairePrompt.js';
import type { Question, QuestionAnswer } from '../../../tools/askUser.js';

const tick = () => new Promise(r => setTimeout(r, 10));

describe('QuestionnairePrompt', () => {
  let instance: ReturnType<typeof render> | null = null;

  afterEach(() => {
    instance?.unmount();
    instance = null;
  });

  // --- Rendering ---

  describe('rendering', () => {
    it('renders single question without options (freetext)', () => {
      instance = render(
        <QuestionnairePrompt
          questions={[{ text: 'What is your name?' }]}
          onComplete={() => {}}
        />
      );
      const frame = instance.lastFrame();
      expect(frame).toContain('What is your name?');
      expect(frame).toContain('Type your answer');
    });

    it('renders question counter for multiple questions', () => {
      instance = render(
        <QuestionnairePrompt
          questions={[
            { text: 'First question' },
            { text: 'Second question' },
          ]}
          onComplete={() => {}}
        />
      );
      const frame = instance.lastFrame();
      expect(frame).toContain('Question 1/2');
      expect(frame).toContain('First question');
    });

    it('does not show counter for single question', () => {
      instance = render(
        <QuestionnairePrompt
          questions={[{ text: 'Only question' }]}
          onComplete={() => {}}
        />
      );
      const frame = instance.lastFrame();
      expect(frame).not.toContain('1/1');
      expect(frame).toContain('Only question');
    });

    it('renders question with options', () => {
      instance = render(
        <QuestionnairePrompt
          questions={[{
            text: 'Pick a platform',
            options: [
              { label: 'iOS', description: 'Apple devices' },
              { label: 'Android', description: 'Google devices' },
            ],
          }]}
          onComplete={() => {}}
        />
      );
      const frame = instance.lastFrame();
      expect(frame).toContain('Pick a platform');
      expect(frame).toContain('[1] iOS');
      expect(frame).toContain('Apple devices');
      expect(frame).toContain('[2] Android');
      expect(frame).toContain('Google devices');
    });

    it('renders default value hint for freetext questions', () => {
      instance = render(
        <QuestionnairePrompt
          questions={[{ text: 'Confirm?', default: 'yes' }]}
          onComplete={() => {}}
        />
      );
      const frame = instance.lastFrame();
      expect(frame).toContain('Enter = yes');
    });

    it('shows cursor indicator', () => {
      instance = render(
        <QuestionnairePrompt
          questions={[{ text: 'Q?' }]}
          onComplete={() => {}}
        />
      );
      expect(instance.lastFrame()).toContain('\u258c');
    });
  });

  // --- Freetext input (no options) ---

  describe('freetext input', () => {
    it('displays typed characters', async () => {
      instance = render(
        <QuestionnairePrompt
          questions={[{ text: 'Name?' }]}
          onComplete={() => {}}
        />
      );
      instance.stdin.write('Alice');
      await tick();
      expect(instance.lastFrame()).toContain('Alice');
    });

    it('handles backspace', async () => {
      instance = render(
        <QuestionnairePrompt
          questions={[{ text: 'Name?' }]}
          onComplete={() => {}}
        />
      );
      instance.stdin.write('hello');
      await tick();
      instance.stdin.write('\u007f');
      await tick();
      expect(instance.lastFrame()).toContain('hell');
      expect(instance.lastFrame()).not.toContain('hello');
    });

    it('sends typed text on Enter', async () => {
      const onComplete = mock(() => {});
      instance = render(
        <QuestionnairePrompt
          questions={[{ text: 'Name?' }]}
          onComplete={onComplete}
        />
      );
      instance.stdin.write('Bob');
      await tick();
      instance.stdin.write('\r');
      await tick();

      expect(onComplete).toHaveBeenCalledTimes(1);
      const answers: QuestionAnswer[] = onComplete.mock.calls[0][0] as any;
      expect(answers).toHaveLength(1);
      expect(answers[0].question).toBe('Name?');
      expect(answers[0].answer).toBe('Bob');
    });

    it('sends default on empty Enter', async () => {
      const onComplete = mock(() => {});
      instance = render(
        <QuestionnairePrompt
          questions={[{ text: 'Confirm?', default: 'yes' }]}
          onComplete={onComplete}
        />
      );
      instance.stdin.write('\r');
      await tick();

      expect(onComplete).toHaveBeenCalledTimes(1);
      const answers: QuestionAnswer[] = onComplete.mock.calls[0][0] as any;
      expect(answers[0].answer).toBe('yes');
    });

    it('ignores empty Enter without default', async () => {
      const onComplete = mock(() => {});
      instance = render(
        <QuestionnairePrompt
          questions={[{ text: 'Name?' }]}
          onComplete={onComplete}
        />
      );
      instance.stdin.write('\r');
      await tick();
      expect(onComplete).not.toHaveBeenCalled();
    });
  });

  // --- Options selection ---

  describe('options selection', () => {
    const optionsQuestion: Question = {
      text: 'Pick DB',
      options: [
        { label: 'PostgreSQL' },
        { label: 'MySQL' },
        { label: 'SQLite' },
      ],
    };

    it('selects first option by default', () => {
      instance = render(
        <QuestionnairePrompt
          questions={[optionsQuestion]}
          onComplete={() => {}}
        />
      );
      const frame = instance.lastFrame();
      expect(frame).toContain('> [1] PostgreSQL');
    });

    it('navigates with arrow keys', async () => {
      instance = render(
        <QuestionnairePrompt
          questions={[optionsQuestion]}
          onComplete={() => {}}
        />
      );
      instance.stdin.write('\x1b[B'); // Arrow Down
      await tick();
      const frame = instance.lastFrame();
      expect(frame).toContain('> [2] MySQL');
    });

    it('confirms option with Enter', async () => {
      const onComplete = mock(() => {});
      instance = render(
        <QuestionnairePrompt
          questions={[optionsQuestion]}
          onComplete={onComplete}
        />
      );
      instance.stdin.write('\x1b[B'); // Arrow Down -> MySQL
      await tick();
      instance.stdin.write('\r'); // Enter
      await tick();

      expect(onComplete).toHaveBeenCalledTimes(1);
      const answers: QuestionAnswer[] = onComplete.mock.calls[0][0] as any;
      expect(answers[0].answer).toBe('MySQL');
    });

    it('quick selects with number key', async () => {
      const onComplete = mock(() => {});
      instance = render(
        <QuestionnairePrompt
          questions={[optionsQuestion]}
          onComplete={onComplete}
        />
      );
      instance.stdin.write('3'); // Quick select SQLite
      await tick();

      expect(onComplete).toHaveBeenCalledTimes(1);
      const answers: QuestionAnswer[] = onComplete.mock.calls[0][0] as any;
      expect(answers[0].answer).toBe('SQLite');
    });

    it('ignores number key out of range', async () => {
      const onComplete = mock(() => {});
      instance = render(
        <QuestionnairePrompt
          questions={[optionsQuestion]}
          onComplete={onComplete}
        />
      );
      instance.stdin.write('9'); // Out of range
      await tick();
      expect(onComplete).not.toHaveBeenCalled();
    });
  });

  // --- Tab toggle (options <-> freetext) ---

  describe('mode switching', () => {
    it('switches to freetext on Tab', async () => {
      instance = render(
        <QuestionnairePrompt
          questions={[{
            text: 'Pick one',
            options: [{ label: 'A' }, { label: 'B' }],
          }]}
          onComplete={() => {}}
        />
      );
      // Initially in options mode
      expect(instance.lastFrame()).toContain('[1] A');

      instance.stdin.write('\t'); // Tab
      await tick();

      // Now in freetext mode
      const frame = instance.lastFrame();
      expect(frame).toContain('Type custom answer');
    });

    it('submits custom answer in freetext mode', async () => {
      const onComplete = mock(() => {});
      instance = render(
        <QuestionnairePrompt
          questions={[{
            text: 'Pick one',
            options: [{ label: 'A' }, { label: 'B' }],
          }]}
          onComplete={onComplete}
        />
      );
      instance.stdin.write('\t'); // Tab to freetext
      await tick();
      instance.stdin.write('Custom');
      await tick();
      instance.stdin.write('\r'); // Enter
      await tick();

      expect(onComplete).toHaveBeenCalledTimes(1);
      const answers: QuestionAnswer[] = onComplete.mock.calls[0][0] as any;
      expect(answers[0].answer).toBe('Custom');
    });

    it('switches back to options on second Tab', async () => {
      instance = render(
        <QuestionnairePrompt
          questions={[{
            text: 'Pick one',
            options: [{ label: 'A' }, { label: 'B' }],
          }]}
          onComplete={() => {}}
        />
      );
      instance.stdin.write('\t'); // Tab to freetext
      await tick();
      expect(instance.lastFrame()).toContain('Type custom answer');

      instance.stdin.write('\t'); // Tab back to options
      await tick();
      expect(instance.lastFrame()).toContain('[1] A');
    });
  });

  // --- Multi-question flow ---

  describe('multi-question flow', () => {
    it('advances to next question after answering', async () => {
      const onComplete = mock(() => {});
      instance = render(
        <QuestionnairePrompt
          questions={[
            { text: 'First?', options: [{ label: 'A' }] },
            { text: 'Second?' },
          ]}
          onComplete={onComplete}
        />
      );

      // Answer first question
      expect(instance.lastFrame()).toContain('Question 1/2');
      instance.stdin.write('1'); // Quick select A
      await tick();

      // Should now show second question
      const frame = instance.lastFrame();
      expect(frame).toContain('Question 2/2');
      expect(frame).toContain('Second?');
      expect(onComplete).not.toHaveBeenCalled();
    });

    it('calls onComplete after last question', async () => {
      const onComplete = mock(() => {});
      instance = render(
        <QuestionnairePrompt
          questions={[
            { text: 'Q1?', options: [{ label: 'Yes' }] },
            { text: 'Q2?', default: 'ok' },
          ]}
          onComplete={onComplete}
        />
      );

      // Answer first
      instance.stdin.write('1');
      await tick();

      // Answer second (use default)
      instance.stdin.write('\r');
      await tick();

      expect(onComplete).toHaveBeenCalledTimes(1);
      const answers: QuestionAnswer[] = onComplete.mock.calls[0][0] as any;
      expect(answers).toHaveLength(2);
      expect(answers[0]).toEqual({ question: 'Q1?', answer: 'Yes' });
      expect(answers[1]).toEqual({ question: 'Q2?', answer: 'ok' });
    });

    it('resets selection state between questions', async () => {
      instance = render(
        <QuestionnairePrompt
          questions={[
            { text: 'Q1?', options: [{ label: 'A' }, { label: 'B' }] },
            { text: 'Q2?', options: [{ label: 'X' }, { label: 'Y' }] },
          ]}
          onComplete={() => {}}
        />
      );

      // Navigate to B in first question
      instance.stdin.write('\x1b[B'); // Down
      await tick();
      expect(instance.lastFrame()).toContain('> [2] B');

      // Answer first question
      instance.stdin.write('\r');
      await tick();

      // Second question should start with first option selected
      expect(instance.lastFrame()).toContain('> [1] X');
    });
  });

  // --- Escape ---

  describe('escape', () => {
    it('cancels with remaining questions marked as cancelled', async () => {
      const onComplete = mock(() => {});
      instance = render(
        <QuestionnairePrompt
          questions={[
            { text: 'Q1?' },
            { text: 'Q2?' },
            { text: 'Q3?' },
          ]}
          onComplete={onComplete}
        />
      );
      instance.stdin.write('\x1b'); // Escape
      await tick();

      expect(onComplete).toHaveBeenCalledTimes(1);
      const answers: QuestionAnswer[] = onComplete.mock.calls[0][0] as any;
      expect(answers).toHaveLength(3);
      expect(answers[0].answer).toBe('cancelled');
      expect(answers[1].answer).toBe('cancelled');
      expect(answers[2].answer).toBe('cancelled');
    });

    it('preserves answered questions on escape', async () => {
      const onComplete = mock(() => {});
      instance = render(
        <QuestionnairePrompt
          questions={[
            { text: 'Q1?', options: [{ label: 'Yes' }] },
            { text: 'Q2?' },
            { text: 'Q3?' },
          ]}
          onComplete={onComplete}
        />
      );

      // Answer first question
      instance.stdin.write('1');
      await tick();

      // Escape on second question
      instance.stdin.write('\x1b');
      await tick();

      expect(onComplete).toHaveBeenCalledTimes(1);
      const answers: QuestionAnswer[] = onComplete.mock.calls[0][0] as any;
      expect(answers).toHaveLength(3);
      expect(answers[0]).toEqual({ question: 'Q1?', answer: 'Yes' });
      expect(answers[1].answer).toBe('cancelled');
      expect(answers[2].answer).toBe('cancelled');
    });
  });

  // --- Double-submit guard ---

  describe('double-submit guard', () => {
    it('prevents double submit on rapid Enter presses', async () => {
      const onComplete = mock(() => {});
      instance = render(
        <QuestionnairePrompt
          questions={[{ text: 'Confirm?', default: 'yes' }]}
          onComplete={onComplete}
        />
      );
      instance.stdin.write('\r');
      instance.stdin.write('\r');
      instance.stdin.write('\r');
      await tick();

      expect(onComplete).toHaveBeenCalledTimes(1);
    });
  });
});
