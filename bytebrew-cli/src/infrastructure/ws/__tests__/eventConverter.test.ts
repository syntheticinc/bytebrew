import { describe, it, expect } from 'bun:test';
import { convertEventToStreamResponse, WsSessionEvent } from '../eventConverter.js';

describe('convertEventToStreamResponse', () => {
  it('converts StreamingProgress to ANSWER_CHUNK', () => {
    const event: WsSessionEvent = { type: 'StreamingProgress', content: 'Hello' };
    const result = convertEventToStreamResponse(event);
    expect(result).not.toBeNull();
    expect(result!.type).toBe('ANSWER_CHUNK');
    expect(result!.content).toBe('Hello');
    expect(result!.isFinal).toBe(false);
  });

  it('converts MessageCompleted to ANSWER', () => {
    const event: WsSessionEvent = { type: 'MessageCompleted', content: 'Done', agent_id: 'a1' };
    const result = convertEventToStreamResponse(event);
    expect(result).not.toBeNull();
    expect(result!.type).toBe('ANSWER');
    expect(result!.content).toBe('Done');
    expect(result!.isFinal).toBe(true);
    expect(result!.agentId).toBe('a1');
  });

  it('converts ReasoningChunk to REASONING', () => {
    const event: WsSessionEvent = { type: 'ReasoningChunk', content: 'Thinking...' };
    const result = convertEventToStreamResponse(event);
    expect(result).not.toBeNull();
    expect(result!.type).toBe('REASONING');
    expect(result!.reasoning?.thinking).toBe('Thinking...');
    expect(result!.reasoning?.isComplete).toBe(false);
  });

  it('converts ToolExecutionStarted to TOOL_CALL', () => {
    const event: WsSessionEvent = {
      type: 'ToolExecutionStarted',
      call_id: 'c1',
      tool_name: 'read_file',
      arguments: { path: '/test.txt' },
    };
    const result = convertEventToStreamResponse(event);
    expect(result).not.toBeNull();
    expect(result!.type).toBe('TOOL_CALL');
    expect(result!.toolCall?.callId).toBe('c1');
    expect(result!.toolCall?.toolName).toBe('read_file');
    expect(result!.toolCall?.arguments).toEqual({ path: '/test.txt' });
  });

  it('converts ToolExecutionCompleted to TOOL_RESULT', () => {
    const event: WsSessionEvent = {
      type: 'ToolExecutionCompleted',
      call_id: 'c1',
      result_summary: '50 lines',
      has_error: false,
    };
    const result = convertEventToStreamResponse(event);
    expect(result).not.toBeNull();
    expect(result!.type).toBe('TOOL_RESULT');
    expect(result!.toolResult?.callId).toBe('c1');
    expect(result!.toolResult?.result).toBe('50 lines');
    expect(result!.toolResult?.error).toBeUndefined();
  });

  it('converts ToolExecutionCompleted with error', () => {
    const event: WsSessionEvent = {
      type: 'ToolExecutionCompleted',
      call_id: 'c2',
      result_summary: 'File not found',
      has_error: true,
    };
    const result = convertEventToStreamResponse(event);
    expect(result!.toolResult?.error).toBe('File not found');
  });

  it('converts AskUserRequested to TOOL_CALL with ask_user', () => {
    const event: WsSessionEvent = {
      type: 'AskUserRequested',
      call_id: 'ask-1',
      question: 'Continue?',
      options: ['Yes', 'No'],
    };
    const result = convertEventToStreamResponse(event);
    expect(result).not.toBeNull();
    expect(result!.type).toBe('TOOL_CALL');
    expect(result!.toolCall?.toolName).toBe('ask_user');
    expect(result!.toolCall?.callId).toBe('ask-1');
    const args = result!.toolCall?.arguments;
    const questions = JSON.parse(args?.questions || '[]');
    expect(questions[0].text).toBe('Continue?');
    expect(questions[0].options).toHaveLength(2);
  });

  it('converts PlanUpdated to TOOL_CALL with manage_plan', () => {
    const event: WsSessionEvent = {
      type: 'PlanUpdated',
      plan_name: 'Migration',
      steps: [
        { title: 'Step 1', status: 'done' },
        { title: 'Step 2', status: 'pending' },
      ],
    };
    const result = convertEventToStreamResponse(event);
    expect(result!.type).toBe('TOOL_CALL');
    expect(result!.toolCall?.toolName).toBe('manage_plan');
    const steps = JSON.parse(result!.toolCall?.arguments?.steps || '[]');
    expect(steps).toHaveLength(2);
    expect(steps[0].description).toBe('Step 1');
  });

  it('converts ProcessingStarted to empty ANSWER_CHUNK', () => {
    const result = convertEventToStreamResponse({ type: 'ProcessingStarted' });
    expect(result!.type).toBe('ANSWER_CHUNK');
    expect(result!.content).toBe('');
    expect(result!.isFinal).toBe(false);
  });

  it('converts ProcessingStopped to final ANSWER_CHUNK', () => {
    const result = convertEventToStreamResponse({ type: 'ProcessingStopped' });
    expect(result!.type).toBe('ANSWER_CHUNK');
    expect(result!.isFinal).toBe(true);
  });

  it('converts Error to ERROR', () => {
    const event: WsSessionEvent = { type: 'Error', message: 'Bad request', code: '400' };
    const result = convertEventToStreamResponse(event);
    expect(result!.type).toBe('ERROR');
    expect(result!.error?.message).toBe('Bad request');
    expect(result!.error?.code).toBe('400');
  });

  it('returns null for unknown event type', () => {
    const result = convertEventToStreamResponse({ type: 'UnknownEvent' });
    expect(result).toBeNull();
  });
});
