import { useState, useRef, useCallback, useEffect } from 'react';
import { parseSSELine, type ToolCall } from '../lib/sse';
import type { EventResponse } from '../types';

// ─── Types ──────────────────────────────────────────────────────────────────

export interface SSEMessage {
  id: string;
  role: 'user' | 'assistant';
  content: string;
  toolCalls?: ToolCall[];
  streaming?: boolean;
}

export interface UseSSEChatConfig {
  endpoint: string;
  agentName: string;
  schemaContext?: string;
  getHeaders?: () => Record<string, string>;
  onToolResult?: (tool: string, output: string) => void;
  /** When set, sessionId is persisted to localStorage under this key. */
  persistenceKey?: string;
  /** Injected fetch function for session event restore (keeps hook api-import-free). */
  fetchMessages?: (sessionId: string) => Promise<EventResponse[]>;
}

export interface UseSSEChatReturn {
  messages: SSEMessage[];
  sendMessage: (text: string) => Promise<void>;
  isStreaming: boolean;
  isRestoring: boolean;
  error: string | null;
  sessionId: string;
  tokenUsage: number | null;
  resetSession: () => void;
  stopStreaming: () => void;
  loadSession: (sessionId: string) => Promise<void>;
}

// ─── Helpers ─────────────────────────────────────────────────────────────────

/** Strip <think>...</think> blocks from streamed LLM output. */
function stripThinkTags(raw: string): string {
  let cleaned = raw.replace(/<think>[\s\S]*?<\/think>/g, '');
  cleaned = cleaned.replace(/<think>[\s\S]*$/, '');
  return cleaned.replace(/^\s+/, '');
}

/** Safe localStorage get — returns null on SecurityError (Safari ITP / iframe). */
function safeGetItem(key: string): string | null {
  try { return localStorage.getItem(key); } catch { return null; }
}

/** Safe localStorage set — no-op on SecurityError. */
function safeSetItem(key: string, value: string): void {
  try { localStorage.setItem(key, value); } catch { /* no-op */ }
}

/** Safe localStorage remove — no-op on SecurityError. */
function safeRemoveItem(key: string): void {
  try { localStorage.removeItem(key); } catch { /* no-op */ }
}

// ─── Events → SSEMessages mapper ─────────────────────────────────────────────

/** Convert EventResponse[] from backend into SSEMessage[] for rendering.
 *  Groups consecutive assistant events + tool calls into one SSEMessage,
 *  preserving chronological tool call order. */
function mapEventsToMessages(events: EventResponse[]): SSEMessage[] {
  const messages: SSEMessage[] = [];
  let currentAssistant: SSEMessage | null = null;

  const flushAssistant = () => {
    if (currentAssistant) {
      messages.push(currentAssistant);
      currentAssistant = null;
    }
  };

  for (const ev of events) {
    const payload = ev.payload ?? {};
    switch (ev.event_type) {
      case 'user_message':
        flushAssistant();
        messages.push({
          id: ev.id,
          role: 'user',
          content: (payload.content as string) ?? '',
          streaming: false,
        });
        break;

      case 'assistant_message':
        if (!currentAssistant) {
          currentAssistant = { id: ev.id, role: 'assistant', content: '', toolCalls: [], streaming: false };
        }
        currentAssistant.content += (payload.content as string) ?? '';
        break;

      case 'tool_call': {
        if (!currentAssistant) {
          currentAssistant = { id: ev.id, role: 'assistant', content: '', toolCalls: [], streaming: false };
        }
        const args = payload.arguments as Record<string, string> | undefined;
        currentAssistant.toolCalls = [...(currentAssistant.toolCalls ?? []), {
          tool: (payload.tool as string) ?? '',
          input: args ? JSON.stringify(args) : '',
        }];
        break;
      }

      case 'tool_result': {
        if (currentAssistant?.toolCalls) {
          const toolName = (payload.tool as string) ?? '';
          const output = (payload.content as string) ?? '';
          currentAssistant.toolCalls = currentAssistant.toolCalls.map((tc) =>
            tc.tool === toolName && !tc.output ? { ...tc, output, status: 'done' as const } : tc,
          );
        }
        break;
      }

      case 'reasoning':
        // Reasoning events are informational — skip for now in chat history
        break;

      case 'system':
        flushAssistant();
        messages.push({
          id: ev.id,
          role: 'assistant',
          content: (payload.content as string) ?? '',
          streaming: false,
        });
        break;
    }
  }

  flushAssistant();
  return messages;
}

// ─── Hook ────────────────────────────────────────────────────────────────────

export function useSSEChat(config: UseSSEChatConfig): UseSSEChatReturn {
  const { endpoint, agentName, schemaContext, getHeaders, onToolResult, persistenceKey, fetchMessages } = config;

  const [messages, setMessages] = useState<SSEMessage[]>([]);
  const [isStreaming, setIsStreaming] = useState(false);
  const [isRestoring, setIsRestoring] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [sessionId, setSessionId] = useState(() =>
    persistenceKey ? (safeGetItem(persistenceKey) ?? '') : '',
  );
  const [tokenUsage, setTokenUsage] = useState<number | null>(null);

  const sessionIdRef = useRef(sessionId);
  const abortRef = useRef<AbortController | null>(null);
  const restoreAbortRef = useRef<AbortController | null>(null);

  // ── Restore session from backend on mount and persistenceKey change ──────
  useEffect(() => {
    if (!persistenceKey || !fetchMessages) return;

    // Abort any active SSE stream on key change
    abortRef.current?.abort();
    setIsStreaming(false);

    // Abort any previous restore fetch
    restoreAbortRef.current?.abort();

    const storedSid = safeGetItem(persistenceKey);
    if (!storedSid) {
      // No stored session — clear state, show empty
      sessionIdRef.current = '';
      setSessionId('');
      setMessages([]);
      return;
    }

    sessionIdRef.current = storedSid;
    setSessionId(storedSid);

    const controller = new AbortController();
    restoreAbortRef.current = controller;

    setIsRestoring(true);
    fetchMessages(storedSid)
      .then((raw) => {
        if (controller.signal.aborted) return;
        setMessages(mapEventsToMessages(raw));
      })
      .catch((err) => {
        if (controller.signal.aborted) return;
        // Non-abort error: session expired/deleted — clear key, start fresh
        if ((err as Error).name !== 'AbortError') {
          safeRemoveItem(persistenceKey);
          sessionIdRef.current = '';
          setSessionId('');
          setMessages([]);
        }
      })
      .finally(() => {
        if (!controller.signal.aborted) setIsRestoring(false);
      });

    return () => {
      controller.abort();
    };
  // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [persistenceKey]);

  const resetSession = useCallback(() => {
    sessionIdRef.current = '';
    setSessionId('');
    setMessages([]);
    abortRef.current?.abort();
    restoreAbortRef.current?.abort();
    setError(null);
    setTokenUsage(null);
    if (persistenceKey) safeRemoveItem(persistenceKey);
  }, [persistenceKey]);

  const stopStreaming = useCallback(() => {
    abortRef.current?.abort();
    setIsStreaming(false);
    setMessages((prev) =>
      prev.map((m) => (m.streaming ? { ...m, streaming: false } : m)),
    );
  }, []);

  const sendMessage = useCallback(async (text: string) => {
    if (!text.trim() || isStreaming) return;

    setIsStreaming(true);
    setError(null);
    abortRef.current = new AbortController();

    const userMsg: SSEMessage = {
      id: crypto.randomUUID(),
      role: 'user',
      content: text,
    };

    const assistantMsgId = crypto.randomUUID();
    const assistantMsg: SSEMessage = {
      id: assistantMsgId,
      role: 'assistant',
      content: '',
      toolCalls: [],
      streaming: true,
    };

    setMessages((prev) => [...prev, userMsg, assistantMsg]);

    const updateAssistant = (patch: Partial<SSEMessage>) => {
      setMessages((prev) =>
        prev.map((m) => (m.id === assistantMsgId ? { ...m, ...patch } : m)),
      );
    };

    try {
      const token = localStorage.getItem('jwt');
      const baseHeaders: Record<string, string> = {
        'Content-Type': 'application/json',
        ...(token ? { Authorization: `Bearer ${token}` } : {}),
      };
      const extraHeaders = getHeaders ? getHeaders() : {};
      const allHeaders = { ...baseHeaders, ...extraHeaders };

      const url = endpoint || `/api/v1/agents/${encodeURIComponent(agentName)}/chat`;
      const res = await fetch(url, {
        method: 'POST',
        headers: allHeaders,
        body: JSON.stringify({
          message: text,
          session_id: sessionIdRef.current || undefined,
          ...(schemaContext ? { schema_context: schemaContext } : {}),
        }),
        signal: abortRef.current.signal,
      });

      if (!res.ok || !res.body) {
        const errText = await res.text().catch(() => 'Request failed');
        sessionIdRef.current = '';
        setSessionId('');
        updateAssistant({ content: `Error: ${errText}`, streaming: false });
        setError(errText);
        return;
      }

      const reader = res.body.getReader();
      const decoder = new TextDecoder();
      let buffer = '';
      let currentEvent = '';
      let currentContent = '';
      let currentToolCalls: ToolCall[] = [];

      while (true) {
        const { done, value } = await reader.read();
        if (done) break;

        buffer += decoder.decode(value, { stream: true });
        const lines = buffer.split('\n');
        buffer = lines.pop() ?? '';

        for (const line of lines) {
          const { event, data } = parseSSELine(line);
          if (event !== undefined) {
            currentEvent = event;
            continue;
          }
          if (data === undefined) continue;

          let parsed: Record<string, unknown> = {};
          try {
            parsed = JSON.parse(data) as Record<string, unknown>;
          } catch {
            continue;
          }

          switch (currentEvent) {
            case 'message_delta': {
              const delta = (parsed.content as string) ?? '';
              currentContent += delta;
              updateAssistant({ content: stripThinkTags(currentContent) });
              break;
            }
            case 'message': {
              const full = (parsed.content as string) ?? '';
              if (full) currentContent = full;
              updateAssistant({ content: stripThinkTags(currentContent) });
              break;
            }
            case 'tool_call': {
              const tc: ToolCall = {
                tool: (parsed.tool as string) ?? '',
                input: (parsed.content as string) ?? '',
              };
              currentToolCalls = [...currentToolCalls, tc];
              updateAssistant({ toolCalls: currentToolCalls });
              break;
            }
            case 'tool_result': {
              const toolName = (parsed.tool as string) ?? '';
              const output = (parsed.content as string) ?? '';
              currentToolCalls = currentToolCalls.map((tc, idx) =>
                idx === currentToolCalls.length - 1 && tc.tool === toolName
                  ? { ...tc, output }
                  : tc,
              );
              updateAssistant({ toolCalls: currentToolCalls });
              onToolResult?.(toolName, output);
              break;
            }
            case 'done': {
              const sid = parsed.session_id as string;
              if (sid) {
                sessionIdRef.current = sid;
                setSessionId(sid);
                if (persistenceKey) safeSetItem(persistenceKey, sid);
              }
              const tokens = parsed.total_tokens as number | undefined;
              if (tokens && tokens > 0) {
                setTokenUsage(tokens);
              }
              updateAssistant({ streaming: false });
              break;
            }
            case 'error': {
              const errContent = (parsed.content as string) || (parsed.message as string) || 'Unknown error';
              sessionIdRef.current = '';
              setSessionId('');
              updateAssistant({ content: `Error: ${errContent}`, streaming: false });
              setError(errContent);
              break;
            }
          }
          currentEvent = '';
        }
      }

      updateAssistant({ streaming: false });
    } catch (err) {
      if ((err as Error).name !== 'AbortError') {
        sessionIdRef.current = '';
        setSessionId('');
        updateAssistant({ content: 'Connection error', streaming: false });
        setError('Connection error');
      }
    } finally {
      setIsStreaming(false);
    }
  // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [isStreaming, endpoint, agentName, getHeaders, persistenceKey]);

  const loadSession = useCallback(async (targetSessionId: string) => {
    abortRef.current?.abort();
    setIsStreaming(false);
    restoreAbortRef.current?.abort();

    sessionIdRef.current = targetSessionId;
    setSessionId(targetSessionId);
    if (persistenceKey) safeSetItem(persistenceKey, targetSessionId);

    if (!fetchMessages) {
      setMessages([]);
      return;
    }

    const controller = new AbortController();
    restoreAbortRef.current = controller;

    setIsRestoring(true);
    setMessages([]);

    try {
      const raw = await fetchMessages(targetSessionId);
      if (controller.signal.aborted) return;
      setMessages(mapEventsToMessages(raw));
    } catch (err) {
      if (controller.signal.aborted) return;
      if ((err as Error).name !== 'AbortError') {
        if (persistenceKey) safeRemoveItem(persistenceKey);
        sessionIdRef.current = '';
        setSessionId('');
        setMessages([]);
      }
    } finally {
      if (!controller.signal.aborted) setIsRestoring(false);
    }
  }, [persistenceKey, fetchMessages]);

  return { messages, sendMessage, isStreaming, isRestoring, error, sessionId, tokenUsage, resetSession, stopStreaming, loadSession };
}
