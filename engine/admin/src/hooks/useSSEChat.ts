import { useState, useRef, useCallback } from 'react';
import { parseSSELine, type ToolCall } from '../lib/sse';

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
  getHeaders?: () => Record<string, string>;
}

export interface UseSSEChatReturn {
  messages: SSEMessage[];
  sendMessage: (text: string) => Promise<void>;
  isStreaming: boolean;
  error: string | null;
  resetSession: () => void;
  stopStreaming: () => void;
}

// ─── Hook ────────────────────────────────────────────────────────────────────

export function useSSEChat(config: UseSSEChatConfig): UseSSEChatReturn {
  const { endpoint, agentName, getHeaders } = config;

  const [messages, setMessages] = useState<SSEMessage[]>([]);
  const [isStreaming, setIsStreaming] = useState(false);
  const [error, setError] = useState<string | null>(null);

  const sessionIdRef = useRef<string>('');
  const abortRef = useRef<AbortController | null>(null);

  const resetSession = useCallback(() => {
    sessionIdRef.current = '';
    setMessages([]);
    abortRef.current?.abort();
    setError(null);
  }, []);

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
        }),
        signal: abortRef.current.signal,
      });

      if (!res.ok || !res.body) {
        const errText = await res.text().catch(() => 'Request failed');
        sessionIdRef.current = '';
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
              updateAssistant({ content: currentContent });
              break;
            }
            case 'message': {
              const full = (parsed.content as string) ?? '';
              if (full) currentContent = full;
              updateAssistant({ content: currentContent });
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
              break;
            }
            case 'done': {
              const sid = parsed.session_id as string;
              if (sid) sessionIdRef.current = sid;
              updateAssistant({ streaming: false });
              break;
            }
            case 'error': {
              const errContent = (parsed.content as string) || (parsed.message as string) || 'Unknown error';
              sessionIdRef.current = '';
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
        updateAssistant({ content: 'Connection error', streaming: false });
        setError('Connection error');
      }
    } finally {
      setIsStreaming(false);
    }
  // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [isStreaming, endpoint, agentName, getHeaders]);

  return { messages, sendMessage, isStreaming, error, resetSession, stopStreaming };
}
