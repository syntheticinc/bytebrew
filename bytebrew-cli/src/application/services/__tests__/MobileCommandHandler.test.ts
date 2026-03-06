import { describe, test, expect } from 'bun:test';
import { MobileCommandHandler } from '../MobileCommandHandler';
import type { QuestionAnswer } from '../../../tools/askUser';

function createHandler() {
  const sent: string[] = [];
  const cancelled: boolean[] = [];
  let processing = false;
  const resolved: QuestionAnswer[][] = [];

  const messageSender = {
    sendMessage: (content: string) => sent.push(content),
    cancel: () => cancelled.push(true),
    getIsProcessing: () => processing,
  };

  const askUserResolver = {
    resolve: (answers: QuestionAnswer[]) => resolved.push(answers),
  };

  const handler = new MobileCommandHandler(messageSender, askUserResolver);

  return {
    handler,
    sent,
    cancelled,
    resolved,
    setProcessing: (v: boolean) => { processing = v; },
  };
}

describe('MobileCommandHandler', () => {
  test('handleNewTask calls sendMessage with content', () => {
    const { handler, sent } = createHandler();

    handler.handleNewTask('dev-1', 'Analyze the project');

    expect(sent).toEqual(['Analyze the project']);
  });

  test('handleNewTask throws on empty deviceId', () => {
    const { handler } = createHandler();
    expect(() => handler.handleNewTask('', 'text')).toThrow('device_id is required');
  });

  test('handleNewTask throws on empty message', () => {
    const { handler } = createHandler();
    expect(() => handler.handleNewTask('dev-1', '')).toThrow('message is required');
  });

  test('handleAskUserReply calls resolver with answer', () => {
    const { handler, resolved } = createHandler();

    handler.handleAskUserReply('session-1', 'yes');

    expect(resolved).toHaveLength(1);
    expect(resolved[0]).toEqual([{ question: '', answer: 'yes' }]);
  });

  test('handleAskUserReply throws on empty sessionId', () => {
    const { handler } = createHandler();
    expect(() => handler.handleAskUserReply('', 'yes')).toThrow('session_id is required');
  });

  test('handleCancel calls cancel when processing', () => {
    const { handler, cancelled, setProcessing } = createHandler();

    setProcessing(true);
    handler.handleCancel('session-1');

    expect(cancelled).toHaveLength(1);
  });

  test('handleCancel is no-op when not processing', () => {
    const { handler, cancelled, setProcessing } = createHandler();

    setProcessing(false);
    handler.handleCancel('session-1');

    expect(cancelled).toHaveLength(0);
  });

  test('handleCancel throws on empty sessionId', () => {
    const { handler } = createHandler();
    expect(() => handler.handleCancel('')).toThrow('session_id is required');
  });
});
