import { describe, it, expect, beforeEach } from 'bun:test';
import { ToolExecution } from '../ToolExecution.js';
import { MessageId } from '../../value-objects/MessageId.js';

describe('ToolExecution', () => {
  const messageId = MessageId.create();
  const callId = 'test-call-123';
  const toolName = 'read_file';
  const args = { path: '/test/file.txt' };

  it('create returns pending execution', () => {
    const execution = ToolExecution.create(callId, toolName, args, messageId);

    expect(execution.status).toBe('pending');
    expect(execution.isPending).toBe(true);
    expect(execution.isExecuting).toBe(false);
    expect(execution.isCompleted).toBe(false);
    expect(execution.isFailed).toBe(false);
    expect(execution.isFinished).toBe(false);
    expect(execution.callId).toBe(callId);
    expect(execution.toolName).toBe(toolName);
    expect(execution.arguments).toEqual(args);
    expect(execution.messageId).toBe(messageId);
    expect(execution.result).toBeUndefined();
    expect(execution.error).toBeUndefined();
    expect(execution.startedAt).toBeUndefined();
    expect(execution.completedAt).toBeUndefined();
  });

  it('startExecution transitions to executing', () => {
    const execution = ToolExecution.create(callId, toolName, args, messageId);
    const started = execution.startExecution();

    expect(started.status).toBe('executing');
    expect(started.isExecuting).toBe(true);
    expect(started.isPending).toBe(false);
    expect(started.startedAt).toBeInstanceOf(Date);
    expect(started.completedAt).toBeUndefined();
    expect(started.isFinished).toBe(false);

    // Original unchanged (immutability)
    expect(execution.status).toBe('pending');
  });

  it('startExecution from non-pending returns same instance', () => {
    const execution = ToolExecution.create(callId, toolName, args, messageId);
    const started = execution.startExecution();
    const startedAgain = started.startExecution();

    expect(startedAgain).toBe(started);
    expect(startedAgain.status).toBe('executing');
  });

  it('complete transitions to completed with result', () => {
    const execution = ToolExecution.create(callId, toolName, args, messageId);
    const started = execution.startExecution();
    const completed = started.complete('file contents here');

    expect(completed.status).toBe('completed');
    expect(completed.isCompleted).toBe(true);
    expect(completed.isFinished).toBe(true);
    expect(completed.result).toBe('file contents here');
    expect(completed.error).toBeUndefined();
    expect(completed.completedAt).toBeInstanceOf(Date);

    // Original unchanged
    expect(started.status).toBe('executing');
    expect(execution.status).toBe('pending');
  });

  it('fail transitions to failed with error', () => {
    const execution = ToolExecution.create(callId, toolName, args, messageId);
    const started = execution.startExecution();
    const failed = started.fail('File not found');

    expect(failed.status).toBe('failed');
    expect(failed.isFailed).toBe(true);
    expect(failed.isFinished).toBe(true);
    expect(failed.error).toBe('File not found');
    expect(failed.result).toBeUndefined();
    expect(failed.completedAt).toBeInstanceOf(Date);

    // Original unchanged
    expect(started.status).toBe('executing');
  });

  it('duration returns ms between start and complete', () => {
    const execution = ToolExecution.create(callId, toolName, args, messageId);

    // Pending - no duration
    expect(execution.duration).toBeUndefined();

    const started = execution.startExecution();

    // Executing - no duration yet
    expect(started.duration).toBeUndefined();

    const completed = started.complete('result');

    // Completed - has duration
    expect(completed.duration).toBeTypeOf('number');
    expect(completed.duration).toBeGreaterThanOrEqual(0);
  });

  it('isFinished true for completed and failed', () => {
    const execution = ToolExecution.create(callId, toolName, args, messageId);

    expect(execution.isFinished).toBe(false);

    const started = execution.startExecution();
    expect(started.isFinished).toBe(false);

    const completed = started.complete('result');
    expect(completed.isFinished).toBe(true);

    const failed = started.fail('error');
    expect(failed.isFinished).toBe(true);
  });

  it('toSnapshot includes all fields', () => {
    const execution = ToolExecution.create(callId, toolName, args, messageId);
    const started = execution.startExecution();
    const completed = started.complete('result content');

    const snapshot = completed.toSnapshot();

    expect(snapshot.callId).toBe(callId);
    expect(snapshot.toolName).toBe(toolName);
    expect(snapshot.arguments).toEqual(args);
    expect(snapshot.messageId).toBe(messageId.value);
    expect(snapshot.status).toBe('completed');
    expect(snapshot.result).toBe('result content');
    expect(snapshot.error).toBeUndefined();
  });

  it('equals compares by callId', () => {
    const execution1 = ToolExecution.create(callId, toolName, args, messageId);
    const execution2 = ToolExecution.create(
      callId,
      'different_tool',
      {},
      MessageId.create()
    );
    const execution3 = ToolExecution.create(
      'different-call-id',
      toolName,
      args,
      messageId
    );

    expect(execution1.equals(execution2)).toBe(true); // Same callId
    expect(execution1.equals(execution3)).toBe(false); // Different callId
  });
});
