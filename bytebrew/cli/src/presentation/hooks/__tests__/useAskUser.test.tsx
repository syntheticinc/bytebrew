import { describe, it, expect, afterEach } from 'bun:test';
import React from 'react';
import { render } from 'ink-testing-library';
import { Text } from 'ink';
import { useAskUser } from '../useAskUser.js';
import { SimpleEventBus } from '../../../infrastructure/events/SimpleEventBus.js';
import { setAskUserEventBus } from '../../../tools/askUser.js';
import type { Question, QuestionAnswer } from '../../../tools/askUser.js';

const tick = () => new Promise(r => setTimeout(r, 30));

let lastHandleComplete: ((answers: QuestionAnswer[]) => void) | null = null;

function TestComponent({ eventBus }: { eventBus: SimpleEventBus }) {
  const { questions, handleComplete } = useAskUser({
    eventBus,
  });
  lastHandleComplete = handleComplete;

  return (
    <Text>
      {questions
        ? `Q:${questions.map(q => q.text).join(',')}|N:${questions.length}`
        : 'idle'}
    </Text>
  );
}

describe('useAskUser', () => {
  let instance: ReturnType<typeof render> | null = null;

  afterEach(() => {
    instance?.unmount();
    instance = null;
    lastHandleComplete = null;
    setAskUserEventBus(null!);
  });

  it('starts in idle state', () => {
    const eventBus = new SimpleEventBus();
    instance = render(<TestComponent eventBus={eventBus} />);
    expect(instance.lastFrame()).toContain('idle');
  });

  it('shows questions on AskUserRequested', async () => {
    const eventBus = new SimpleEventBus();
    instance = render(<TestComponent eventBus={eventBus} />);
    await tick();

    const questions: Question[] = [
      { text: 'Platform?' },
      { text: 'Approve?' },
    ];
    eventBus.publish({ type: 'AskUserRequested', questions });
    await tick();

    expect(instance.lastFrame()).toContain('Q:Platform?,Approve?');
    expect(instance.lastFrame()).toContain('N:2');
  });

  it('resets state on ProcessingStopped', async () => {
    const eventBus = new SimpleEventBus();
    instance = render(<TestComponent eventBus={eventBus} />);
    await tick();

    eventBus.publish({
      type: 'AskUserRequested',
      questions: [{ text: 'Q?' }],
    });
    await tick();
    expect(instance.lastFrame()).toContain('Q:Q?');

    eventBus.publish({ type: 'ProcessingStopped' } as any);
    await tick();
    expect(instance.lastFrame()).toContain('idle');
  });

  it('does NOT publish MessageCompleted (pure state, no side effects)', async () => {
    const eventBus = new SimpleEventBus();
    const events: string[] = [];
    eventBus.subscribe('MessageCompleted', () => events.push('MessageCompleted'));

    instance = render(<TestComponent eventBus={eventBus} />);
    await tick();

    eventBus.publish({
      type: 'AskUserRequested',
      questions: [{ text: 'Task?', default: 'approved' }],
    });
    await tick();

    lastHandleComplete!([{ question: 'Task?', answer: 'approved' }]);
    await tick();

    // No messages saved, no events -- useAskUser is pure state management
    expect(events).toEqual([]);
  });

  it('resets state after handleComplete', async () => {
    const eventBus = new SimpleEventBus();
    instance = render(<TestComponent eventBus={eventBus} />);
    await tick();

    eventBus.publish({
      type: 'AskUserRequested',
      questions: [{ text: 'Q?' }],
    });
    await tick();
    expect(instance.lastFrame()).toContain('Q:Q?');

    lastHandleComplete!([{ question: 'Q?', answer: 'yes' }]);
    await tick();
    expect(instance.lastFrame()).toContain('idle');
  });
});
