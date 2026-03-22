import { useState, useRef, useEffect, useCallback } from 'react';
import { useAuth } from '../lib/auth';

interface ChatMessage {
  id: string;
  role: 'user' | 'assistant';
  content: string;
}

interface ExampleChatProps {
  agentName: string;
  apiUrl: string;
  suggestions: string[];
}

const MAX_MESSAGES_PER_HOUR = 15;
const STORAGE_KEY_ACCESS = 'bytebrew_access_token';

export function ExampleChat({ agentName, apiUrl, suggestions }: ExampleChatProps) {
  const { isAuthenticated, triggerAuthPopup } = useAuth();
  const [messages, setMessages] = useState<ChatMessage[]>([]);
  const [input, setInput] = useState('');
  const [isStreaming, setIsStreaming] = useState(false);
  const [messagesUsed, setMessagesUsed] = useState(0);
  const [sessionId, setSessionId] = useState<string | null>(null);
  const [error, setError] = useState<string | null>(null);
  const messagesEndRef = useRef<HTMLDivElement>(null);
  const abortRef = useRef<AbortController | null>(null);

  const messagesRemaining = MAX_MESSAGES_PER_HOUR - messagesUsed;

  useEffect(() => {
    messagesEndRef.current?.scrollIntoView({ behavior: 'smooth' });
  }, [messages]);

  const streamChat = useCallback(async (userMessage: string, currentSessionId: string | null) => {
    const assistantId = crypto.randomUUID();
    setMessages(prev => [...prev, { id: assistantId, role: 'assistant', content: '' }]);
    setIsStreaming(true);
    setError(null);

    const controller = new AbortController();
    abortRef.current = controller;

    try {
      const token = localStorage.getItem(STORAGE_KEY_ACCESS);
      const body: Record<string, string> = { message: userMessage };
      if (currentSessionId) body.session_id = currentSessionId;

      const response = await fetch(`${apiUrl}/v1/chat/${agentName}`, {
        method: 'POST',
        headers: {
          'Content-Type': 'application/json',
          ...(token ? { 'Authorization': `Bearer ${token}` } : {}),
        },
        body: JSON.stringify(body),
        signal: controller.signal,
      });

      if (response.status === 429) {
        setError('Rate limit exceeded. Try again later.');
        setMessages(prev => prev.filter(m => m.id !== assistantId));
        setIsStreaming(false);
        return;
      }

      if (response.status === 401) {
        setError('Authentication required.');
        setMessages(prev => prev.filter(m => m.id !== assistantId));
        setIsStreaming(false);
        return;
      }

      if (!response.ok) {
        const text = await response.text();
        setError(`Error: ${text || response.statusText}`);
        setMessages(prev => prev.filter(m => m.id !== assistantId));
        setIsStreaming(false);
        return;
      }

      const reader = response.body?.getReader();
      if (!reader) {
        setError('Streaming not supported');
        setIsStreaming(false);
        return;
      }

      const decoder = new TextDecoder();
      let buffer = '';
      let fullContent = '';

      while (true) {
        const { done, value } = await reader.read();
        if (done) break;

        buffer += decoder.decode(value, { stream: true });
        const lines = buffer.split('\n');
        buffer = lines.pop() || '';

        for (const line of lines) {
          if (line.startsWith('data: ')) {
            try {
              const data = JSON.parse(line.slice(6));

              if (data.content) {
                fullContent += data.content;
                setMessages(prev =>
                  prev.map(m => m.id === assistantId ? { ...m, content: fullContent } : m)
                );
              }

              if (data.session_id) {
                setSessionId(data.session_id);
              }

              if (data.error) {
                setError(data.error);
              }
            } catch {
              // skip non-JSON data lines
            }
          }

          if (line.startsWith('event: message_delta')) {
            // next data line has delta content
          }

          if (line.startsWith('event: message')) {
            // next data line has full message — we already built it from deltas
          }

          if (line.startsWith('event: error')) {
            // next data line has error
          }
        }
      }
    } catch (err: unknown) {
      if (err instanceof Error && err.name === 'AbortError') return;
      setError(`Connection error: ${err instanceof Error ? err.message : 'unknown'}`);
      setMessages(prev => prev.filter(m => m.id !== assistantId));
    } finally {
      setIsStreaming(false);
      abortRef.current = null;
    }
  }, [agentName, apiUrl]);

  const handleSend = useCallback(
    (text: string) => {
      const trimmed = text.trim();
      if (!trimmed || isStreaming) return;
      if (messagesRemaining <= 0) return;

      if (!isAuthenticated) {
        triggerAuthPopup(() => {
          handleSend(trimmed);
        }, 'Sign in to try the demo');
        return;
      }

      const userMsg: ChatMessage = {
        id: crypto.randomUUID(),
        role: 'user',
        content: trimmed,
      };

      setMessages(prev => [...prev, userMsg]);
      setMessagesUsed(prev => prev + 1);
      setInput('');

      streamChat(trimmed, sessionId);
    },
    [isAuthenticated, isStreaming, messagesRemaining, triggerAuthPopup, streamChat, sessionId],
  );

  const handleSubmit = (e: React.FormEvent) => {
    e.preventDefault();
    handleSend(input);
  };

  const showSuggestions = messages.length === 0 && !isStreaming;

  return (
    <div className="rounded-[12px] border border-brand-shade3/15 bg-brand-dark flex flex-col overflow-hidden" style={{ height: '480px' }}>
      {/* Messages area */}
      <div className="flex-1 overflow-y-auto p-4 space-y-4">
        {showSuggestions && (
          <div className="flex flex-col items-center justify-center h-full gap-4">
            <p className="text-sm text-brand-shade3">Try one of these conversation starters:</p>
            <div className="flex flex-wrap justify-center gap-2 max-w-lg">
              {suggestions.map((suggestion) => (
                <button
                  key={suggestion}
                  onClick={() => handleSend(suggestion)}
                  className="rounded-[10px] border border-brand-shade3/20 px-3 py-2 text-xs text-brand-shade2 hover:text-brand-light hover:border-brand-accent/40 hover:bg-brand-accent/5 transition-colors text-left"
                >
                  {suggestion}
                </button>
              ))}
            </div>
          </div>
        )}

        {messages.map((msg) => (
          <div
            key={msg.id}
            className={`flex ${msg.role === 'user' ? 'justify-end' : 'justify-start'}`}
          >
            <div
              className={`max-w-[80%] rounded-[10px] px-4 py-2.5 text-sm leading-relaxed whitespace-pre-wrap ${
                msg.role === 'user'
                  ? 'bg-brand-accent text-white'
                  : 'bg-brand-dark-alt text-brand-light border border-brand-shade3/15'
              }`}
            >
              {msg.content}
              {msg.role === 'assistant' && isStreaming && msg.id === messages[messages.length - 1]?.id && (
                <span className="inline-block w-1.5 h-4 bg-brand-accent ml-0.5 animate-pulse" />
              )}
            </div>
          </div>
        ))}

        {error && (
          <div className="text-center text-xs text-red-400 py-2">{error}</div>
        )}

        <div ref={messagesEndRef} />
      </div>

      {/* Input area */}
      <div className="border-t border-brand-shade3/15 p-3">
        <form onSubmit={handleSubmit} className="flex gap-2">
          <input
            type="text"
            value={input}
            onChange={(e) => setInput(e.target.value)}
            placeholder={messagesRemaining <= 0 ? 'Rate limit reached' : 'Type a message...'}
            disabled={isStreaming || messagesRemaining <= 0}
            className="flex-1 rounded-[10px] border border-brand-shade3/20 bg-brand-dark-alt px-4 py-2 text-sm text-brand-light placeholder:text-brand-shade3 focus:outline-none focus:border-brand-accent/50 disabled:opacity-50 transition-colors"
          />
          <button
            type="submit"
            disabled={!input.trim() || isStreaming || messagesRemaining <= 0}
            className="rounded-[10px] bg-brand-accent px-4 py-2 text-sm font-medium text-white hover:bg-brand-accent-hover disabled:opacity-40 disabled:cursor-not-allowed transition-colors"
          >
            Send
          </button>
        </form>
        <div className="mt-2 text-xs text-brand-shade3 text-center">
          {messagesRemaining}/{MAX_MESSAGES_PER_HOUR} messages remaining this hour
        </div>
      </div>
    </div>
  );
}
