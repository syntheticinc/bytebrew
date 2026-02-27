// Integration test for ask_user event-driven flow.
// Tests the full chain: callback -> EventBus -> resolve
// WITHOUT server or React rendering -- pure data flow test.
import { describe, it, expect, beforeEach, afterEach } from 'bun:test';
import { SimpleEventBus } from '../../infrastructure/events/SimpleEventBus.js';
import { InMemoryMessageRepository } from '../../infrastructure/persistence/InMemoryMessageRepository.js';
import { Message } from '../../domain/entities/Message.js';
import {
  setAskUserEventBus,
  createInteractiveAskUserCallback,
  resolveAskUser,
} from '../askUser.js';
import type { AskUserRequestedEvent } from '../../domain/ports/IEventBus.js';
import type { Question, QuestionAnswer } from '../askUser.js';

describe('ask_user integration flow', () => {
  let eventBus: SimpleEventBus;
  let messageRepo: InMemoryMessageRepository;

  beforeEach(() => {
    eventBus = new SimpleEventBus();
    messageRepo = new InMemoryMessageRepository();
    setAskUserEventBus(eventBus);
  });

  afterEach(() => {
    setAskUserEventBus(null!);
    eventBus.clear();
  });

  // --- Event publishing ---

  describe('event publishing', () => {
    it('publishes AskUserRequested event with questions array', () => {
      const callback = createInteractiveAskUserCallback();

      const events: AskUserRequestedEvent[] = [];
      eventBus.subscribe('AskUserRequested', (e) => events.push(e));

      const questions: Question[] = [
        { text: 'Which platform?', options: [{ label: 'iOS' }, { label: 'Android' }] },
        { text: 'Approve?', default: 'approved' },
      ];

      callback(questions);

      expect(events).toHaveLength(1);
      expect(events[0].questions).toHaveLength(2);
      expect(events[0].questions[0].text).toBe('Which platform?');
      expect(events[0].questions[0].options).toHaveLength(2);
      expect(events[0].questions[1].text).toBe('Approve?');
      expect(events[0].questions[1].default).toBe('approved');
    });

    it('publishes event with single question (no options)', () => {
      const callback = createInteractiveAskUserCallback();

      const events: AskUserRequestedEvent[] = [];
      eventBus.subscribe('AskUserRequested', (e) => events.push(e));

      callback([{ text: 'What should I do next?' }]);

      expect(events[0].questions).toHaveLength(1);
      expect(events[0].questions[0].options).toBeUndefined();
    });
  });

  // --- Promise resolution ---

  describe('resolve', () => {
    it('resolveAskUser resolves the callback promise with answers array', async () => {
      const callback = createInteractiveAskUserCallback();

      const answerPromise = callback([{ text: 'Approve?' }]);

      const answers: QuestionAnswer[] = [{ question: 'Approve?', answer: 'approved' }];
      resolveAskUser(answers);

      const result = await answerPromise;
      expect(result).toHaveLength(1);
      expect(result[0].question).toBe('Approve?');
      expect(result[0].answer).toBe('approved');
    });

    it('resolves with multiple answers for multiple questions', async () => {
      const callback = createInteractiveAskUserCallback();

      const questions: Question[] = [
        { text: 'Platform?', options: [{ label: 'iOS' }, { label: 'Android' }] },
        { text: 'Priority?', options: [{ label: 'High' }, { label: 'Low' }] },
      ];

      const answerPromise = callback(questions);

      const answers: QuestionAnswer[] = [
        { question: 'Platform?', answer: 'iOS' },
        { question: 'Priority?', answer: 'High' },
      ];
      resolveAskUser(answers);

      const result = await answerPromise;
      expect(result).toHaveLength(2);
      expect(result[0].answer).toBe('iOS');
      expect(result[1].answer).toBe('High');
    });

    it('resolveAskUser is no-op when no pending callback', () => {
      // Should not throw
      resolveAskUser([{ question: 'orphan', answer: 'ignored' }]);
    });
  });

  // --- Full end-to-end flow ---

  describe('full flow', () => {
    it('questions -> event -> user answers -> resolve', async () => {
      let askUserShown = false;
      eventBus.subscribe('AskUserRequested', (event) => {
        const formatted = event.questions.map((q, i) => `${i + 1}. ${q.text}`).join('\n');
        const msg = Message.createAssistantWithContent(formatted);
        messageRepo.save(msg);
        eventBus.publish({ type: 'MessageCompleted', message: msg });
        askUserShown = true;
      });

      const callback = createInteractiveAskUserCallback();
      const questions: Question[] = [
        { text: 'Which DB?', options: [{ label: 'PostgreSQL' }, { label: 'MySQL' }] },
        { text: 'Confirm?', default: 'yes' },
      ];

      const answerPromise = callback(questions);

      expect(askUserShown).toBe(true);
      const messagesAfterQ = messageRepo.findComplete();
      expect(messagesAfterQ).toHaveLength(1);
      expect(messagesAfterQ[0].content.value).toContain('1. Which DB?');
      expect(messagesAfterQ[0].content.value).toContain('2. Confirm?');

      const answers: QuestionAnswer[] = [
        { question: 'Which DB?', answer: 'PostgreSQL' },
        { question: 'Confirm?', answer: 'yes' },
      ];

      // Simulate UI completing the questionnaire
      const answerFormatted = answers.map(a => `${a.question}: ${a.answer}`).join('\n');
      const userMsg = Message.createUser(answerFormatted);
      messageRepo.save(userMsg);
      eventBus.publish({ type: 'MessageCompleted', message: userMsg });
      resolveAskUser(answers);

      const result = await answerPromise;
      expect(result).toHaveLength(2);
      expect(result[0].answer).toBe('PostgreSQL');

      const allMessages = messageRepo.findComplete();
      expect(allMessages).toHaveLength(2);
      expect(allMessages[0].role).toBe('assistant');
      expect(allMessages[1].role).toBe('user');
      expect(allMessages[1].content.value).toContain('PostgreSQL');
    });

    it('event is delivered instantly (no polling delay)', () => {
      let eventReceivedAt = 0;
      let callbackCalledAt = 0;

      eventBus.subscribe('AskUserRequested', () => {
        eventReceivedAt = performance.now();
      });

      const callback = createInteractiveAskUserCallback();
      callbackCalledAt = performance.now();
      callback([{ text: 'Test?' }]);

      const delay = eventReceivedAt - callbackCalledAt;
      expect(delay).toBeLessThan(5);
    });
  });
});
