// StreamingState value object - represents the streaming status of a message

export type StreamingStatus = 'pending' | 'streaming' | 'complete' | 'aborted';

export class StreamingState {
  private readonly _status: StreamingStatus;
  private readonly _startedAt: Date | null;
  private readonly _completedAt: Date | null;

  private constructor(
    status: StreamingStatus,
    startedAt: Date | null,
    completedAt: Date | null
  ) {
    this._status = status;
    this._startedAt = startedAt;
    this._completedAt = completedAt;
  }

  static pending(): StreamingState {
    return new StreamingState('pending', null, null);
  }

  static streaming(): StreamingState {
    return new StreamingState('streaming', new Date(), null);
  }

  static complete(): StreamingState {
    return new StreamingState('complete', null, new Date());
  }

  get status(): StreamingStatus {
    return this._status;
  }

  get isStreaming(): boolean {
    return this._status === 'streaming';
  }

  get isComplete(): boolean {
    return this._status === 'complete';
  }

  get isPending(): boolean {
    return this._status === 'pending';
  }

  get isAborted(): boolean {
    return this._status === 'aborted';
  }

  get startedAt(): Date | null {
    return this._startedAt;
  }

  get completedAt(): Date | null {
    return this._completedAt;
  }

  startStreaming(): StreamingState {
    if (this._status !== 'pending') {
      return this;
    }
    return new StreamingState('streaming', new Date(), null);
  }

  markComplete(): StreamingState {
    return new StreamingState('complete', this._startedAt, new Date());
  }

  abort(): StreamingState {
    return new StreamingState('aborted', this._startedAt, new Date());
  }

  equals(other: StreamingState): boolean {
    return this._status === other._status;
  }
}
