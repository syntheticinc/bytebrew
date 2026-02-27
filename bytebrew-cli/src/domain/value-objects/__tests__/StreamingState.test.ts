import { describe, it, expect } from 'bun:test';
import { StreamingState } from '../StreamingState.js';

describe('StreamingState', () => {
  it('pending creates pending state', () => {
    const state = StreamingState.pending();

    expect(state.status).toBe('pending');
    expect(state.isPending).toBe(true);
    expect(state.isStreaming).toBe(false);
    expect(state.isComplete).toBe(false);
    expect(state.isAborted).toBe(false);
    expect(state.startedAt).toBeNull();
    expect(state.completedAt).toBeNull();
  });

  it('streaming creates streaming state with startedAt', () => {
    const state = StreamingState.streaming();

    expect(state.status).toBe('streaming');
    expect(state.isStreaming).toBe(true);
    expect(state.isPending).toBe(false);
    expect(state.isComplete).toBe(false);
    expect(state.isAborted).toBe(false);
    expect(state.startedAt).toBeInstanceOf(Date);
    expect(state.completedAt).toBeNull();
  });

  it('complete creates complete state with completedAt', () => {
    const state = StreamingState.complete();

    expect(state.status).toBe('complete');
    expect(state.isComplete).toBe(true);
    expect(state.isPending).toBe(false);
    expect(state.isStreaming).toBe(false);
    expect(state.isAborted).toBe(false);
    expect(state.startedAt).toBeNull();
    expect(state.completedAt).toBeInstanceOf(Date);
  });

  it('startStreaming transitions from pending to streaming', () => {
    const pending = StreamingState.pending();
    const streaming = pending.startStreaming();

    expect(streaming.status).toBe('streaming');
    expect(streaming.isStreaming).toBe(true);
    expect(streaming.startedAt).toBeInstanceOf(Date);
    expect(streaming.completedAt).toBeNull();

    // Original unchanged (immutability)
    expect(pending.status).toBe('pending');
  });

  it('startStreaming from non-pending returns same instance', () => {
    const streaming = StreamingState.streaming();
    const streamingAgain = streaming.startStreaming();

    expect(streamingAgain).toBe(streaming);

    const complete = StreamingState.complete();
    const completeAgain = complete.startStreaming();

    expect(completeAgain).toBe(complete);
  });

  it('markComplete transitions to complete', () => {
    const pending = StreamingState.pending();
    const completed = pending.markComplete();

    expect(completed.status).toBe('complete');
    expect(completed.isComplete).toBe(true);
    expect(completed.completedAt).toBeInstanceOf(Date);

    // Original unchanged
    expect(pending.status).toBe('pending');

    // From streaming
    const streaming = StreamingState.streaming();
    const completedFromStreaming = streaming.markComplete();

    expect(completedFromStreaming.status).toBe('complete');
    expect(completedFromStreaming.startedAt).toBe(streaming.startedAt);
    expect(completedFromStreaming.completedAt).toBeInstanceOf(Date);
  });

  it('abort transitions to aborted', () => {
    const pending = StreamingState.pending();
    const aborted = pending.abort();

    expect(aborted.status).toBe('aborted');
    expect(aborted.isAborted).toBe(true);
    expect(aborted.completedAt).toBeInstanceOf(Date);

    // Original unchanged
    expect(pending.status).toBe('pending');

    // From streaming preserves startedAt
    const streaming = StreamingState.streaming();
    const abortedFromStreaming = streaming.abort();

    expect(abortedFromStreaming.status).toBe('aborted');
    expect(abortedFromStreaming.startedAt).toBe(streaming.startedAt);
    expect(abortedFromStreaming.completedAt).toBeInstanceOf(Date);
  });

  it('equals compares by status', () => {
    const pending1 = StreamingState.pending();
    const pending2 = StreamingState.pending();
    const streaming = StreamingState.streaming();
    const complete = StreamingState.complete();

    expect(pending1.equals(pending2)).toBe(true);
    expect(pending1.equals(streaming)).toBe(false);
    expect(pending1.equals(complete)).toBe(false);
    expect(streaming.equals(complete)).toBe(false);
  });
});
