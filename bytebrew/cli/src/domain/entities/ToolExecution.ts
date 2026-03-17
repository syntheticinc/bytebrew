// ToolExecution entity - represents a tool execution lifecycle
import { MessageId } from '../value-objects/MessageId.js';
import type { DiffLine } from '../message.js';

export type ToolExecutionStatus = 'pending' | 'executing' | 'completed' | 'failed';

export interface ToolExecutionProps {
  callId: string;
  toolName: string;
  arguments: Record<string, string>;
  messageId: MessageId;
  status: ToolExecutionStatus;
  agentId?: string;
  result?: string;
  error?: string;
  summary?: string;
  diffLines?: DiffLine[];
  startedAt?: Date;
  completedAt?: Date;
}

/**
 * ToolExecution entity - tracks the lifecycle of a tool call.
 * Immutable with behavior methods that return new instances.
 */
export class ToolExecution {
  private readonly _callId: string;
  private readonly _toolName: string;
  private readonly _arguments: Record<string, string>;
  private readonly _messageId: MessageId;
  private readonly _status: ToolExecutionStatus;
  private readonly _agentId?: string;
  private readonly _result?: string;
  private readonly _error?: string;
  private readonly _summary?: string;
  private readonly _diffLines?: DiffLine[];
  private readonly _startedAt?: Date;
  private readonly _completedAt?: Date;

  private constructor(props: ToolExecutionProps) {
    this._callId = props.callId;
    this._toolName = props.toolName;
    this._arguments = { ...props.arguments };
    this._messageId = props.messageId;
    this._status = props.status;
    this._agentId = props.agentId;
    this._result = props.result;
    this._error = props.error;
    this._summary = props.summary;
    this._diffLines = props.diffLines;
    this._startedAt = props.startedAt;
    this._completedAt = props.completedAt;
  }

  static create(
    callId: string,
    toolName: string,
    args: Record<string, string>,
    messageId: MessageId,
    agentId?: string
  ): ToolExecution {
    return new ToolExecution({
      callId,
      toolName,
      arguments: args,
      messageId,
      status: 'pending',
      agentId,
    });
  }

  // Getters
  get callId(): string {
    return this._callId;
  }

  get toolName(): string {
    return this._toolName;
  }

  get arguments(): Record<string, string> {
    return { ...this._arguments };
  }

  get messageId(): MessageId {
    return this._messageId;
  }

  get status(): ToolExecutionStatus {
    return this._status;
  }

  get result(): string | undefined {
    return this._result;
  }

  get error(): string | undefined {
    return this._error;
  }

  get summary(): string | undefined {
    return this._summary;
  }

  get diffLines(): DiffLine[] | undefined {
    return this._diffLines;
  }

  get startedAt(): Date | undefined {
    return this._startedAt;
  }

  get completedAt(): Date | undefined {
    return this._completedAt;
  }

  get agentId(): string | undefined {
    return this._agentId;
  }

  get isPending(): boolean {
    return this._status === 'pending';
  }

  get isExecuting(): boolean {
    return this._status === 'executing';
  }

  get isCompleted(): boolean {
    return this._status === 'completed';
  }

  get isFailed(): boolean {
    return this._status === 'failed';
  }

  get isFinished(): boolean {
    return this._status === 'completed' || this._status === 'failed';
  }

  get duration(): number | undefined {
    if (!this._startedAt || !this._completedAt) {
      return undefined;
    }
    return this._completedAt.getTime() - this._startedAt.getTime();
  }

  // Behavior methods
  startExecution(): ToolExecution {
    if (this._status !== 'pending') {
      return this;
    }
    return new ToolExecution({
      ...this.toProps(),
      status: 'executing',
      startedAt: new Date(),
    });
  }

  complete(result: string, summary?: string, diffLines?: DiffLine[]): ToolExecution {
    return new ToolExecution({
      ...this.toProps(),
      status: 'completed',
      result,
      summary,
      diffLines,
      completedAt: new Date(),
    });
  }

  fail(error: string): ToolExecution {
    return new ToolExecution({
      ...this.toProps(),
      status: 'failed',
      error,
      completedAt: new Date(),
    });
  }

  private toProps(): ToolExecutionProps {
    return {
      callId: this._callId,
      toolName: this._toolName,
      arguments: this._arguments,
      messageId: this._messageId,
      status: this._status,
      agentId: this._agentId,
      result: this._result,
      error: this._error,
      summary: this._summary,
      diffLines: this._diffLines,
      startedAt: this._startedAt,
      completedAt: this._completedAt,
    };
  }

  toSnapshot(): {
    callId: string;
    toolName: string;
    arguments: Record<string, string>;
    messageId: string;
    status: ToolExecutionStatus;
    agentId?: string;
    result?: string;
    error?: string;
    summary?: string;
  } {
    return {
      callId: this._callId,
      toolName: this._toolName,
      arguments: this._arguments,
      messageId: this._messageId.value,
      status: this._status,
      agentId: this._agentId,
      result: this._result,
      error: this._error,
      summary: this._summary,
    };
  }

  equals(other: ToolExecution): boolean {
    return this._callId === other._callId;
  }
}
