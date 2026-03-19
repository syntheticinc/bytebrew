import { useState, useRef, useCallback, useEffect } from 'react';
import { api } from '../api/client';
import type { ChatMessage, ChatEvent } from '../types';

const STORAGE_KEY = 'bytebrew_chat_';
const SESSION_KEY = 'bytebrew_session_';

function loadMessages(agent: string): ChatMessage[] {
  try {
    const raw = sessionStorage.getItem(STORAGE_KEY + agent);
    if (!raw) return [];
    const parsed = JSON.parse(raw) as ChatMessage[];
    return parsed.map((m) => ({ ...m, timestamp: new Date(m.timestamp) }));
  } catch {
    return [];
  }
}

function saveMessages(agent: string, messages: ChatMessage[]) {
  if (!agent) return;
  try {
    sessionStorage.setItem(STORAGE_KEY + agent, JSON.stringify(messages));
  } catch {
    // ignore quota errors
  }
}

function getSessionId(agent: string): string {
  const existing = sessionStorage.getItem(SESSION_KEY + agent);
  if (existing) return existing;
  const id = 'web-' + crypto.randomUUID();
  sessionStorage.setItem(SESSION_KEY + agent, id);
  return id;
}

function clearSessionId(agent: string) {
  sessionStorage.removeItem(SESSION_KEY + agent);
}

export function useChat(currentAgent: string | null) {
  const [messages, setMessages] = useState<ChatMessage[]>(() =>
    currentAgent ? loadMessages(currentAgent) : [],
  );
  const [streaming, setStreaming] = useState(false);
  const controllerRef = useRef<AbortController | null>(null);
  const assistantContentRef = useRef('');
  const agentRef = useRef(currentAgent);

  // When agent changes, load saved messages
  useEffect(() => {
    if (currentAgent && currentAgent !== agentRef.current) {
      agentRef.current = currentAgent;
      setMessages(loadMessages(currentAgent));
      setStreaming(false);
      controllerRef.current?.abort();
    }
  }, [currentAgent]);

  // Persist messages on change
  useEffect(() => {
    if (agentRef.current) {
      saveMessages(agentRef.current, messages);
    }
  }, [messages]);

  const handleEvent = useCallback((event: ChatEvent) => {
    switch (event.type) {
      case 'thinking':
        setMessages((prev) => {
          const last = prev[prev.length - 1];
          if (last?.role === 'thinking') {
            return [
              ...prev.slice(0, -1),
              { ...last, content: last.content + (event.data.content as string) },
            ];
          }
          return [
            ...prev,
            {
              id: crypto.randomUUID(),
              role: 'thinking' as const,
              content: event.data.content as string,
              timestamp: new Date(),
            },
          ];
        });
        break;

      case 'message_delta':
        assistantContentRef.current += event.data.content as string;
        setMessages((prev) => {
          const content = assistantContentRef.current;
          const last = prev[prev.length - 1];
          if (last?.role === 'assistant') {
            return [...prev.slice(0, -1), { ...last, content }];
          }
          const filtered = prev.filter((m) => m.role !== 'thinking');
          return [
            ...filtered,
            {
              id: crypto.randomUUID(),
              role: 'assistant' as const,
              content,
              timestamp: new Date(),
            },
          ];
        });
        break;

      case 'message': {
        const msgContent = event.data.content as string;
        if (!msgContent) break;
        assistantContentRef.current = msgContent;
        setMessages((prev) => {
          const last = prev[prev.length - 1];
          if (last?.role === 'assistant') {
            return [...prev.slice(0, -1), { ...last, content: msgContent }];
          }
          const filtered = prev.filter((m) => m.role !== 'thinking');
          return [
            ...filtered,
            {
              id: crypto.randomUUID(),
              role: 'assistant' as const,
              content: msgContent,
              timestamp: new Date(),
            },
          ];
        });
        break;
      }

      case 'tool_call':
        setMessages((prev) => [
          ...prev,
          {
            id: crypto.randomUUID(),
            role: 'tool_call' as const,
            content: event.data.content as string,
            toolName: event.data.tool as string,
            timestamp: new Date(),
          },
        ]);
        break;

      case 'tool_result':
        setMessages((prev) => [
          ...prev,
          {
            id: crypto.randomUUID(),
            role: 'tool_result' as const,
            content: event.data.content as string,
            toolName: event.data.tool as string,
            timestamp: new Date(),
          },
        ]);
        break;

      case 'confirmation':
        setMessages((prev) => [
          ...prev,
          {
            id: crypto.randomUUID(),
            role: 'confirmation' as const,
            content: (event.data.prompt as string) ?? '',
            toolName: event.data.tool as string,
            confirmArgs: (event.data.args as Record<string, unknown>) ?? {},
            confirmPrompt: (event.data.prompt as string) ?? '',
            timestamp: new Date(),
          },
        ]);
        break;

      case 'done': {
        const status = event.data.status as string | undefined;
        setStreaming(false);

        if (status === 'max_steps') {
          setMessages((prev) => [
            ...prev,
            {
              id: crypto.randomUUID(),
              role: 'info' as const,
              content: 'Agent reached maximum steps. You can continue the conversation.',
              timestamp: new Date(),
            },
          ]);
        } else if (status === 'error') {
          setMessages((prev) => [
            ...prev,
            {
              id: crypto.randomUUID(),
              role: 'error' as const,
              content: (event.data.message as string) ?? 'An error occurred.',
              timestamp: new Date(),
            },
          ]);
        } else if (status === 'escalated') {
          setMessages((prev) => [
            ...prev,
            {
              id: crypto.randomUUID(),
              role: 'info' as const,
              content: (event.data.message as string) ?? 'Conversation escalated to a human operator.',
              timestamp: new Date(),
            },
          ]);
        } else if (status === 'timeout') {
          setMessages((prev) => [
            ...prev,
            {
              id: crypto.randomUUID(),
              role: 'info' as const,
              content: 'Agent timed out. You can try again or continue the conversation.',
              timestamp: new Date(),
            },
          ]);
        }
        break;
      }

      case 'error':
        setMessages((prev) => [
          ...prev,
          {
            id: crypto.randomUUID(),
            role: 'error' as const,
            content: event.data.message as string,
            timestamp: new Date(),
          },
        ]);
        setStreaming(false);
        break;
    }
  }, []);

  const send = useCallback((agent: string, text: string) => {
    const userMsg: ChatMessage = {
      id: crypto.randomUUID(),
      role: 'user',
      content: text,
      timestamp: new Date(),
    };
    setMessages((prev) => [...prev, userMsg]);
    setStreaming(true);
    assistantContentRef.current = '';

    const sessionId = getSessionId(agent);
    controllerRef.current = api.chat(agent, text, handleEvent, sessionId);
  }, [handleEvent]);

  const respond = useCallback((agent: string, text: string) => {
    setStreaming(true);
    assistantContentRef.current = '';

    const sessionId = getSessionId(agent);
    controllerRef.current = api.chat(agent, text, handleEvent, sessionId);
  }, [handleEvent]);

  const stop = useCallback(() => {
    controllerRef.current?.abort();
    setStreaming(false);
  }, []);

  const clear = useCallback(() => {
    setMessages([]);
    if (agentRef.current) {
      sessionStorage.removeItem(STORAGE_KEY + agentRef.current);
      clearSessionId(agentRef.current);
    }
  }, []);

  return { messages, streaming, send, respond, stop, clear };
}
