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

export function ExampleChat({ agentName: _agentName, apiUrl: _apiUrl, suggestions }: ExampleChatProps) {
  const { isAuthenticated, triggerAuthPopup } = useAuth();
  const [messages, setMessages] = useState<ChatMessage[]>([]);
  const [input, setInput] = useState('');
  const [isStreaming, setIsStreaming] = useState(false);
  const [messagesUsed, setMessagesUsed] = useState(0);
  const messagesEndRef = useRef<HTMLDivElement>(null);
  const inputRef = useRef<HTMLInputElement>(null);

  const messagesRemaining = MAX_MESSAGES_PER_HOUR - messagesUsed;

  useEffect(() => {
    messagesEndRef.current?.scrollIntoView({ behavior: 'smooth' });
  }, [messages]);

  const simulateStream = useCallback((userMessage: string) => {
    const assistantId = crypto.randomUUID();
    const mockResponse = `Thanks for your question about "${userMessage}". This is a demo response. When the backend is connected, you'll see real AI-generated answers streamed in real time.`;

    setMessages((prev) => [
      ...prev,
      { id: assistantId, role: 'assistant', content: '' },
    ]);
    setIsStreaming(true);

    let charIndex = 0;
    const interval = setInterval(() => {
      charIndex++;
      const partial = mockResponse.slice(0, charIndex);
      setMessages((prev) =>
        prev.map((m) => (m.id === assistantId ? { ...m, content: partial } : m)),
      );

      if (charIndex >= mockResponse.length) {
        clearInterval(interval);
        setIsStreaming(false);
      }
    }, 15);
  }, []);

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

      setMessages((prev) => [...prev, userMsg]);
      setMessagesUsed((prev) => prev + 1);
      setInput('');

      simulateStream(trimmed);
    },
    [isAuthenticated, isStreaming, messagesRemaining, triggerAuthPopup, simulateStream],
  );

  const handleSubmit = (e: React.FormEvent) => {
    e.preventDefault();
    handleSend(input);
  };

  const handleSuggestionClick = (suggestion: string) => {
    handleSend(suggestion);
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
                  onClick={() => handleSuggestionClick(suggestion)}
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
              className={`max-w-[80%] rounded-[10px] px-4 py-2.5 text-sm leading-relaxed ${
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
        <div ref={messagesEndRef} />
      </div>

      {/* Input area */}
      <div className="border-t border-brand-shade3/15 p-3">
        <form onSubmit={handleSubmit} className="flex gap-2">
          <input
            ref={inputRef}
            type="text"
            value={input}
            onChange={(e) => setInput(e.target.value)}
            placeholder="Type a message..."
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
