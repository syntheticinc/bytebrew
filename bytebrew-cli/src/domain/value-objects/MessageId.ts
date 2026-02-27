// MessageId value object - immutable identifier for messages
import { v4 as uuidv4 } from 'uuid';

export class MessageId {
  private readonly _value: string;

  private constructor(value: string) {
    this._value = value;
  }

  static create(): MessageId {
    return new MessageId(uuidv4());
  }

  static from(value: string): MessageId {
    if (!value || value.trim().length === 0) {
      throw new Error('MessageId cannot be empty');
    }
    return new MessageId(value);
  }

  get value(): string {
    return this._value;
  }

  equals(other: MessageId): boolean {
    return this._value === other._value;
  }

  toString(): string {
    return this._value;
  }
}
