import { useState, useRef, useCallback } from 'react';
import { api } from '../api/client';
import type { ChatMessage } from '../types';

export function useChat() {
  const [messages, setMessages] = useState<ChatMessage[]>([]);
  const [streaming, setStreaming] = useState(false);
  const controllerRef = useRef<AbortController | null>(null);
  const assistantContentRef = useRef('');

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

    controllerRef.current = api.chat(agent, text, (event) => {
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
          // MessageCompleted — replace with final content (not append)
          const msgContent = event.data.content as string;
          if (!msgContent) break; // ignore empty message events
          assistantContentRef.current = msgContent;
          setMessages((prev) => {
            const content = msgContent;
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

        case 'done':
          setStreaming(false);
          break;

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
    });
  }, []);

  const stop = useCallback(() => {
    controllerRef.current?.abort();
    setStreaming(false);
  }, []);

  const clear = useCallback(() => {
    setMessages([]);
  }, []);

  return { messages, streaming, send, stop, clear };
}
