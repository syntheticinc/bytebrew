// MessageContent value object - immutable content container

export class MessageContent {
  private readonly _value: string;

  private constructor(value: string) {
    this._value = value;
  }

  static empty(): MessageContent {
    return new MessageContent('');
  }

  static from(value: string): MessageContent {
    return new MessageContent(value);
  }

  get value(): string {
    return this._value;
  }

  get length(): number {
    return this._value.length;
  }

  get isEmpty(): boolean {
    return this._value.length === 0;
  }

  /**
   * Approximate token count (4 chars per token)
   */
  get approximateTokens(): number {
    return Math.ceil(this._value.length / 4);
  }

  append(chunk: string): MessageContent {
    return new MessageContent(this._value + chunk);
  }

  equals(other: MessageContent): boolean {
    return this._value === other._value;
  }

  toString(): string {
    return this._value;
  }

  /**
   * Truncate content with ellipsis
   */
  truncate(maxLength: number): MessageContent {
    if (this._value.length <= maxLength) {
      return this;
    }
    return new MessageContent(this._value.slice(0, maxLength) + '...');
  }
}
