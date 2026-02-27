import { Message, ToolCallInfo } from '../../../domain/entities/Message.js';
import { MessageId } from '../../../domain/value-objects/MessageId.js';
import { IStreamGateway, StreamResponse, ConnectionStatus, SubResult } from '../../../domain/ports/IStreamGateway.js';
import { IMessageRepository } from '../../../domain/ports/IMessageRepository.js';
import { IToolExecutor, ToolExecutionResult } from '../../../domain/ports/IToolExecutor.js';
import { IEventBus, DomainEvent, DomainEventType, EventHandler } from '../../../domain/ports/IEventBus.js';

// Mock StreamGateway
export class MockStreamGateway implements IStreamGateway {
  private responseHandlers: ((response: StreamResponse) => void)[] = [];
  private errorHandlers: ((error: Error) => void)[] = [];
  private statusHandlers: ((status: ConnectionStatus) => void)[] = [];
  private _isConnected = true;
  public sentMessages: string[] = [];
  public sentToolResults: { callId: string; result: string; error?: Error; subResults?: SubResult[] }[] = [];

  async connect(): Promise<void> {
    this._isConnected = true;
  }

  disconnect(): void {
    this._isConnected = false;
  }

  sendMessage(message: string): void {
    this.sentMessages.push(message);
  }

  sendToolResult(callId: string, result: string, error?: Error, subResults?: SubResult[]): void {
    this.sentToolResults.push({ callId, result, error, subResults });
  }

  cancel(): void {}

  async reconnectStream(): Promise<void> {
    this._isConnected = true;
  }

  getStatus(): ConnectionStatus {
    return this._isConnected ? 'connected' : 'disconnected';
  }

  isConnected(): boolean {
    return this._isConnected;
  }

  getReconnectAttempts(): number {
    return 0;
  }

  onResponse(handler: (response: StreamResponse) => void): () => void {
    this.responseHandlers.push(handler);
    return () => {
      const idx = this.responseHandlers.indexOf(handler);
      if (idx >= 0) this.responseHandlers.splice(idx, 1);
    };
  }

  onError(handler: (error: Error) => void): () => void {
    this.errorHandlers.push(handler);
    return () => {
      const idx = this.errorHandlers.indexOf(handler);
      if (idx >= 0) this.errorHandlers.splice(idx, 1);
    };
  }

  onStatusChange(handler: (status: ConnectionStatus) => void): () => void {
    this.statusHandlers.push(handler);
    return () => {
      const idx = this.statusHandlers.indexOf(handler);
      if (idx >= 0) this.statusHandlers.splice(idx, 1);
    };
  }

  // Test helpers
  simulateResponse(response: StreamResponse): void {
    this.responseHandlers.forEach(h => h(response));
  }

  simulateError(error: Error): void {
    this.errorHandlers.forEach(h => h(error));
  }

  simulateStatusChange(status: ConnectionStatus): void {
    this.statusHandlers.forEach(h => h(status));
  }
}

// Mock MessageRepository
export class MockMessageRepository implements IMessageRepository {
  private messages: Map<string, Message> = new Map();
  private subscribers: ((messages: Message[]) => void)[] = [];

  save(message: Message): void {
    this.messages.set(message.id.value, message);
    this.notifySubscribers();
  }

  findById(id: MessageId): Message | undefined {
    return this.messages.get(id.value);
  }

  findByToolCallId(callId: string): Message | undefined {
    for (const msg of this.messages.values()) {
      if (msg.toolCall?.callId === callId) {
        return msg;
      }
    }
    return undefined;
  }

  findAll(): Message[] {
    return Array.from(this.messages.values());
  }

  findComplete(): Message[] {
    return this.findAll().filter(m => m.isComplete);
  }

  findRecent(limit: number): Message[] {
    return this.findAll().slice(-limit);
  }

  delete(id: MessageId): void {
    this.messages.delete(id.value);
    this.notifySubscribers();
  }

  clear(): void {
    this.messages.clear();
    this.notifySubscribers();
  }

  count(): number {
    return this.messages.size;
  }

  subscribe(listener: (messages: Message[]) => void): () => void {
    this.subscribers.push(listener);
    return () => {
      const idx = this.subscribers.indexOf(listener);
      if (idx >= 0) this.subscribers.splice(idx, 1);
    };
  }

  private notifySubscribers(): void {
    const messages = this.findAll();
    this.subscribers.forEach(s => s(messages));
  }
}

// Mock ToolExecutor
export class MockToolExecutor implements IToolExecutor {
  public executedCalls: ToolCallInfo[] = [];
  public executeResult: ToolExecutionResult = { result: 'mock result' };

  async execute(toolCall: ToolCallInfo): Promise<ToolExecutionResult> {
    this.executedCalls.push(toolCall);
    return this.executeResult;
  }

  hasTool(name: string): boolean {
    return true;
  }

  listTools(): string[] {
    return ['read_file', 'search_code'];
  }
}

// Mock EventBus
export class MockEventBus implements IEventBus {
  public publishedEvents: DomainEvent[] = [];
  private handlers: Map<DomainEventType | 'all', EventHandler[]> = new Map();

  publish<T extends DomainEvent>(event: T): void {
    this.publishedEvents.push(event);

    // Notify specific handlers
    const handlers = this.handlers.get(event.type as DomainEventType) || [];
    handlers.forEach(h => h(event));

    // Notify all handlers
    const allHandlers = this.handlers.get('all') || [];
    allHandlers.forEach(h => h(event));
  }

  subscribe<T extends DomainEventType>(
    eventType: T,
    handler: EventHandler<Extract<DomainEvent, { type: T }>>
  ): () => void {
    if (!this.handlers.has(eventType)) {
      this.handlers.set(eventType, []);
    }
    this.handlers.get(eventType)!.push(handler as EventHandler);
    return () => {
      const handlers = this.handlers.get(eventType);
      if (handlers) {
        const idx = handlers.indexOf(handler as EventHandler);
        if (idx >= 0) handlers.splice(idx, 1);
      }
    };
  }

  subscribeAll(handler: EventHandler): () => void {
    if (!this.handlers.has('all')) {
      this.handlers.set('all', []);
    }
    this.handlers.get('all')!.push(handler);
    return () => {
      const handlers = this.handlers.get('all');
      if (handlers) {
        const idx = handlers.indexOf(handler);
        if (idx >= 0) handlers.splice(idx, 1);
      }
    };
  }

  clear(): void {
    this.handlers.clear();
    this.publishedEvents = [];
  }

  // Test helper
  getEventsOfType(type: DomainEventType): DomainEvent[] {
    return this.publishedEvents.filter(e => e.type === type);
  }
}
