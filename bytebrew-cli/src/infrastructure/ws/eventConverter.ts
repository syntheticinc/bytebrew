// Converts flat mobile/WS event format to StreamResponse
import type { StreamResponse, StreamResponseType } from '../../domain/ports/IStreamGateway.js';
import type { ToolCallInfo } from '../../domain/entities/Message.js';

/**
 * Flat event format received from the WS server.
 * Same format used by EventBroadcaster for mobile clients.
 */
export interface WsSessionEvent {
  type: string;
  content?: string;
  agent_id?: string;
  call_id?: string;
  tool_name?: string;
  arguments?: Record<string, string>;
  result_summary?: string;
  has_error?: boolean;
  question?: string;
  options?: string[];
  message?: string;
  code?: string;
  plan_name?: string;
  steps?: Array<{ title: string; status: string }>;
  state?: string;
}

/**
 * Convert a flat WS session event to a StreamResponse.
 * Returns null for events that don't map to a response (e.g. ProcessingStarted).
 */
export function convertEventToStreamResponse(event: WsSessionEvent): StreamResponse | null {
  const agentId = event.agent_id || undefined;

  switch (event.type) {
    case 'StreamingProgress':
      return {
        type: 'ANSWER_CHUNK',
        content: event.content || '',
        isFinal: false,
        agentId,
      };

    case 'MessageCompleted':
      return {
        type: 'ANSWER',
        content: event.content || '',
        isFinal: true,
        agentId,
      };

    case 'ReasoningChunk':
      return {
        type: 'REASONING',
        content: event.content || '',
        isFinal: false,
        agentId,
        reasoning: {
          thinking: event.content || '',
          isComplete: false,
        },
      };

    case 'ToolExecutionStarted':
      return {
        type: 'TOOL_CALL',
        content: '',
        isFinal: false,
        agentId,
        toolCall: {
          callId: event.call_id || '',
          toolName: event.tool_name || '',
          arguments: event.arguments || {},
        },
      };

    case 'ToolExecutionCompleted':
      return {
        type: 'TOOL_RESULT',
        content: '',
        isFinal: false,
        agentId,
        toolResult: {
          callId: event.call_id || '',
          result: event.result_summary || '',
          error: event.has_error ? (event.result_summary || 'Tool error') : undefined,
          summary: event.result_summary || undefined,
        },
      };

    case 'AskUserRequested':
      return {
        type: 'TOOL_CALL',
        content: '',
        isFinal: false,
        agentId,
        toolCall: {
          callId: event.call_id || `ask-${Date.now()}`,
          toolName: 'ask_user',
          arguments: {
            questions: JSON.stringify([{
              text: event.question || 'Please respond',
              options: (event.options || []).map(o => ({ label: o })),
            }]),
          },
        },
      };

    case 'PlanUpdated':
      return {
        type: 'TOOL_CALL',
        content: '',
        isFinal: false,
        agentId,
        toolCall: {
          callId: `plan-${Date.now()}`,
          toolName: 'manage_plan',
          arguments: {
            goal: event.plan_name || '',
            steps: JSON.stringify(
              (event.steps || []).map((s, i) => ({
                index: i,
                description: s.title,
                status: s.status || 'pending',
              }))
            ),
          },
        },
      };

    case 'ProcessingStarted':
      return {
        type: 'ANSWER_CHUNK',
        content: '',
        isFinal: false,
        agentId,
      };

    case 'ProcessingStopped':
      return {
        type: 'ANSWER_CHUNK',
        content: '',
        isFinal: true,
        agentId,
      };

    case 'Error':
      return {
        type: 'ERROR',
        content: event.message || 'Unknown error',
        isFinal: false,
        agentId,
        error: {
          message: event.message || 'Unknown error',
          code: event.code,
        },
      };

    default:
      return null;
  }
}
