// StreamProcessorService - coordinates stream processing with handlers
import { Message } from '../../domain/entities/Message.js';
import { IMessageRepository } from '../../domain/ports/IMessageRepository.js';
import { IStreamGateway, StreamResponse, ConnectionStatus } from '../../domain/ports/IStreamGateway.js';
import { IToolExecutor } from '../../domain/ports/IToolExecutor.js';
import { IEventBus } from '../../domain/ports/IEventBus.js';
import { MessageAccumulatorService } from './MessageAccumulatorService.js';
import { AgentStateManager } from '../../infrastructure/state/AgentStateManager.js';
import {
  StreamProcessorContext,
  ResponseTypeMap,
  completeCurrentMessage,
  stopProcessing,
} from './handlers/StreamProcessorContext.js';
import { handleAnswerChunk, handleAnswer } from './handlers/AnswerStreamHandler.js';
import { handleReasoning } from './handlers/ReasoningHandler.js';
import { handleToolCall, handleServerToolResult } from './handlers/ToolExecutionHandler.js';
import { handleStreamError, handleError } from './handlers/ErrorHandler.js';
import type { AgentLifecycleType } from '../../domain/ports/IEventBus.js';
import { formatLifecycleMessage } from '../../presentation/utils/formatLifecycleMessage.js';

export interface StreamProcessorOptions {
  streamGateway: IStreamGateway;
  messageRepository: IMessageRepository;
  toolExecutor: IToolExecutor;
  accumulator: MessageAccumulatorService;
  eventBus: IEventBus;
  agentStateManager?: AgentStateManager;
}

/**
 * Service that processes stream responses and coordinates between
 * the accumulator, repository, and event bus.
 *
 * Replaces the logic from useGrpcStream with a testable,
 * framework-agnostic implementation.
 */
export class StreamProcessorService {
  private readonly streamGateway: IStreamGateway;
  private readonly messageRepository: IMessageRepository;
  private readonly toolExecutor: IToolExecutor;
  private readonly accumulator: MessageAccumulatorService;
  private readonly eventBus: IEventBus;
  private readonly agentStateManager: AgentStateManager;

  // Processing state (per-agent message/reasoning IDs managed by AgentStateManager)
  private _isProcessing = false;

  // Track last agent in stream to insert separators when switching
  private lastAgentIdInStream = 'supervisor';

  // Unsubscribe functions
  private unsubscribeResponse: (() => void) | null = null;
  private unsubscribeError: (() => void) | null = null;
  private unsubscribeStatus: (() => void) | null = null;

  constructor(options: StreamProcessorOptions) {
    this.streamGateway = options.streamGateway;
    this.messageRepository = options.messageRepository;
    this.toolExecutor = options.toolExecutor;
    this.accumulator = options.accumulator;
    this.eventBus = options.eventBus;
    this.agentStateManager = options.agentStateManager || new AgentStateManager();
  }

  /**
   * Get the agent state manager for external access
   */
  getAgentStateManager(): AgentStateManager {
    return this.agentStateManager;
  }

  /**
   * Create context for handlers with current state.
   * If agentId is provided, uses per-agent state tracking.
   */
  private createContext(agentId?: string): StreamProcessorContext {
    // Per-agent state: if agentId provided, track message/reasoning IDs per agent
    const resolvedAgentId = agentId || 'supervisor';

    return {
      streamGateway: this.streamGateway,
      messageRepository: this.messageRepository,
      toolExecutor: this.toolExecutor,
      accumulator: this.accumulator,
      eventBus: this.eventBus,
      getCurrentMessageId: () => this.agentStateManager.getCurrentMessageId(resolvedAgentId),
      getCurrentReasoningId: () => this.agentStateManager.getCurrentReasoningId(resolvedAgentId),
      getIsProcessing: () => this._isProcessing,
      setCurrentMessageId: (id) => { this.agentStateManager.setCurrentMessageId(resolvedAgentId, id); },
      setCurrentReasoningId: (id) => { this.agentStateManager.setCurrentReasoningId(resolvedAgentId, id); },
      setIsProcessing: (value) => { this._isProcessing = value; },
      agentId: resolvedAgentId,
    };
  }

  /**
   * Initialize the processor and subscribe to stream events
   */
  initialize(): void {
    this.unsubscribeResponse = this.streamGateway.onResponse(
      this.handleResponse.bind(this)
    );
    this.unsubscribeError = this.streamGateway.onError(
      (error) => handleError(this.createContext(), error)
    );
    this.unsubscribeStatus = this.streamGateway.onStatusChange(
      this.handleStatusChange.bind(this)
    );
  }

  /**
   * Cleanup subscriptions
   */
  dispose(): void {
    this.unsubscribeResponse?.();
    this.unsubscribeError?.();
    this.unsubscribeStatus?.();
  }

  /**
   * Send a user message.
   * If not connected (e.g. after cancel), auto-reconnects before sending.
   */
  sendMessage(content: string): void {
    if (!this.streamGateway.isConnected()) {
      this.reconnectAndSend(content);
      return;
    }
    this.executeSend(content);
  }

  /**
   * Reconnect the stream and then send the message.
   * Used when the connection was lost (e.g. after user cancel).
   */
  private reconnectAndSend(content: string): void {
    this.streamGateway.reconnectStream()
      .then(() => {
        if (this.streamGateway.isConnected()) {
          this.executeSend(content);
        }
      })
      .catch((err) => {
        this.eventBus.publish({
          type: 'ErrorOccurred',
          error: err instanceof Error ? err : new Error(String(err)),
          context: 'reconnectAndSend',
        });
      });
  }

  /**
   * Execute the actual send logic (called when connection is confirmed).
   */
  private executeSend(content: string): void {
    // Add user message to repository (immediately visible)
    const userMessage = Message.createUser(content);
    this.messageRepository.save(userMessage);

    this.eventBus.publish({
      type: 'MessageCompleted',
      message: userMessage,
    });

    if (this._isProcessing) {
      // Interrupt: complete partial messages so new server response starts fresh.
      // Server cancels current REACT turn and processes this message as a new turn.
      this.completePartialMessages();
      this.safeSendMessage(content);
      return;
    }

    // First message — full initialization
    this.agentStateManager.resetAll();
    this.lastAgentIdInStream = 'supervisor';
    this.accumulator.addInputTokens(content);

    this._isProcessing = true;
    this.eventBus.publish({ type: 'ProcessingStarted' });

    this.safeSendMessage(content);
  }

  /**
   * Send message to gateway, publishing ErrorOccurred on failure.
   */
  private safeSendMessage(content: string): void {
    try {
      this.streamGateway.sendMessage(content);
    } catch (err) {
      this.eventBus.publish({
        type: 'ErrorOccurred',
        error: err instanceof Error ? err : new Error(String(err)),
        context: 'sendMessage',
      });
    }
  }

  /**
   * Complete all partial messages across all agents.
   * Called before sending an interrupt so the new server response starts a fresh message
   * instead of appending to the interrupted one.
   */
  private completePartialMessages(): void {
    for (const agent of this.agentStateManager.getAllAgents()) {
      const ctx = this.createContext(agent.agentId);
      completeCurrentMessage(ctx);
      if (ctx.getCurrentReasoningId()) {
        ctx.setCurrentReasoningId(null);
      }
    }
  }

  /**
   * Cancel the current stream
   */
  cancel(): void {
    // Abort all per-agent accumulating messages
    for (const agent of this.agentStateManager.getAllAgents()) {
      if (agent.currentMessageId) {
        this.accumulator.abort(agent.currentMessageId);
      }
      if (agent.currentReasoningId) {
        this.accumulator.abort(agent.currentReasoningId);
      }
    }
    this.agentStateManager.resetAll();
    this.accumulator.resetTokenCounts();

    this.streamGateway.cancel();
    this._isProcessing = false;

    // Add cancellation message to chat history
    const cancelMsg = Message.createAssistantWithContent('[Cancelled by user]');
    this.messageRepository.save(cancelMsg);
    this.eventBus.publish({ type: 'MessageCompleted', message: cancelMsg });

    this.eventBus.publish({ type: 'ProcessingStopped' });
  }

  /**
   * Get current processing state
   */
  getIsProcessing(): boolean {
    return this._isProcessing;
  }

  /**
   * Handle incoming stream response - delegates to appropriate handler.
   * Routes to per-agent state using response.agentId.
   */
  private handleResponse(response: StreamResponse): void {
    const agentId = response.agentId || 'supervisor';
    const ctx = this.createContext(agentId);

    // Track agent activity and update lastMessageAt
    this.agentStateManager.getOrCreateAgent(agentId);
    this.agentStateManager.touchAgent(agentId);

    // Convert numeric type to string if needed
    const responseType = typeof response.type === 'number'
      ? ResponseTypeMap[response.type] || 'UNSPECIFIED'
      : response.type;

    // isFinal without content = stream end signal
    if (response.isFinal && !response.content && !response.toolCall && !response.reasoning) {
      completeCurrentMessage(ctx);
      if (agentId === 'supervisor') {
        stopProcessing(ctx);
      }
      return;
    }

    // Detect agent lifecycle events (sent as ANSWER_CHUNK from server)
    if (responseType === 'ANSWER_CHUNK') {
      const lifecycle = this.parseLifecycleEvent(response.content);
      if (lifecycle) {
        this.handleLifecycleEvent(lifecycle);
        return;
      }
    }

    // Insert agent separator when switching agents in multi-agent scenario.
    // Applies to content response types (ANSWER_CHUNK, TOOL_CALL, ANSWER, REASONING).
    // Lifecycle events are excluded (handled above with early return) since they
    // serve as visual separators themselves.
    if (agentId !== this.lastAgentIdInStream && this.agentStateManager.hasMultipleAgents()) {
      this.insertAgentSeparator(agentId);
      this.lastAgentIdInStream = agentId;
    }

    switch (responseType) {
      case 'ANSWER_CHUNK':
        handleAnswerChunk(ctx, response);
        break;

      case 'ANSWER':
        handleAnswer(ctx, response);
        break;

      case 'REASONING':
        handleReasoning(ctx, response);
        break;

      case 'TOOL_CALL':
        handleToolCall(ctx, response);
        break;

      case 'TOOL_RESULT':
        handleServerToolResult(ctx, response);
        break;

      case 'ERROR':
        handleStreamError(ctx, response);
        break;

      default:
        if (response.isFinal && agentId === 'supervisor') {
          completeCurrentMessage(ctx);
          stopProcessing(ctx);
        }
    }
  }

  /**
   * Handle agent lifecycle event: update state, create messages, notify UI.
   * Creates lifecycle message in supervisor stream and [Task] message in agent tab.
   */
  private handleLifecycleEvent(lifecycle: {
    lifecycleType: AgentLifecycleType;
    agentId: string;
    description: string;
  }): void {
    this.agentStateManager.getOrCreateAgent(lifecycle.agentId);
    this.agentStateManager.updateLifecycle(
      lifecycle.agentId, lifecycle.lifecycleType, lifecycle.description,
    );

    // Lifecycle message visible in supervisor's unified stream
    const content = formatLifecycleMessage(
      lifecycle.lifecycleType, lifecycle.agentId, lifecycle.description,
    );
    const msg = Message.createAssistantWithContent(content, 'supervisor');
    this.messageRepository.save(msg);
    this.eventBus.publish({ type: 'MessageCompleted', message: msg });

    // For spawned agents: task prompt visible in agent's own tab
    if (lifecycle.lifecycleType === 'agent_spawned' && lifecycle.description) {
      const taskContent = `[Task from Supervisor]\n${lifecycle.description}`;
      const taskMsg = Message.createAssistantWithContent(taskContent, lifecycle.agentId);
      this.messageRepository.save(taskMsg);
      this.eventBus.publish({ type: 'MessageCompleted', message: taskMsg });
    }

    // Notify UI hooks for immediate state refresh
    this.eventBus.publish({ type: 'AgentLifecycle', ...lifecycle });
  }

  /**
   * Parse agent lifecycle event from ANSWER_CHUNK content.
   * Returns null if content is not a lifecycle event.
   */
  private parseLifecycleEvent(content: string | undefined): {
    lifecycleType: AgentLifecycleType;
    agentId: string;
    description: string;
  } | null {
    if (!content) return null;

    // [^:]+ matches agentId (everything except colon), [\s\S]* captures description
    // (including potential newlines in content from server)
    const match = content.match(
      /^\[(agent_spawned|agent_completed|agent_failed|agent_restarted)\]\s+([^:]+):\s*([\s\S]*)$/
    );
    if (!match) return null;

    const lifecycleType = match[1] as AgentLifecycleType;
    const agentId = match[2]?.trim();
    if (!agentId) return null;

    return { lifecycleType, agentId, description: match[3] || '' };
  }

  /**
   * Handle connection status change
   */
  private handleStatusChange(_status: ConnectionStatus): void {
    // Status changes are handled by the presentation layer
    // through direct subscription to the gateway
  }

  /**
   * Insert visual separator when switching agents in multi-agent mode.
   * Creates a message like "─── Code Agent [abc]: Task description ───"
   */
  private insertAgentSeparator(agentId: string): void {
    const agent = this.agentStateManager.getOrCreateAgent(agentId);
    const shortId = agentId.replace('code-agent-', '');
    const label = agentId === 'supervisor'
      ? 'Supervisor'
      : agent.taskDescription
        ? `Code Agent [${shortId}]: ${agent.taskDescription}`
        : `Code Agent [${shortId}]`;

    const separator = `─── ${label} ───`;
    const separatorMsg = Message.createAssistantWithContent(separator, agentId);
    this.messageRepository.save(separatorMsg);
    this.eventBus.publish({ type: 'MessageCompleted', message: separatorMsg });
  }
}
