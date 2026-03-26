import { useState, useRef, useEffect, useCallback } from 'react';
import { useAuth } from '../lib/auth';
import { refreshAccessToken } from '../api/auth';

interface ToolCallInfo {
  id: string;
  tool: string;
  arguments?: string;
  status: 'calling' | 'completed' | 'error';
  result?: string;
}

interface AskUserQuestion {
  text: string;
  options?: { label: string }[];
  default?: string;
}

type MessageSegment =
  | { type: 'text'; content: string }
  | { type: 'tool'; toolCall: ToolCallInfo }
  | { type: 'ask_user'; callId: string; questions: AskUserQuestion[]; answered: boolean; answer?: string };

interface ChatMessage {
  id: string;
  role: 'user' | 'assistant';
  content: string;
  segments: MessageSegment[];
}

interface ExampleChatProps {
  agentName: string;
  apiUrl: string;
  suggestions: string[];
}

const MAX_MESSAGES_PER_HOUR = 15;
const STORAGE_KEY_ACCESS = 'bytebrew_access_token';
const STORAGE_KEY_REFRESH = 'bytebrew_refresh_token';

function ToolCallBlock({ tc, expanded, onToggle }: { tc: ToolCallInfo; expanded: boolean; onToggle: () => void }) {
  const hasLongResult = tc.result != null && tc.result.length > 120;

  return (
    <div className="bg-slate-800/50 border-l-2 border-orange-500/50 rounded px-3 py-1.5 text-sm my-1.5">
      <div className="flex items-center gap-1.5">
        <span className="text-xs">
          {tc.status === 'calling' ? '\u2699\uFE0F' : tc.status === 'error' ? '\u274C' : '\u2705'}
        </span>
        <span className="font-mono font-semibold text-orange-400 text-xs">{tc.tool}</span>
        {tc.arguments && (
          <span className="text-slate-400 font-mono text-xs truncate max-w-[200px]">
            ({tc.arguments.length > 60 ? tc.arguments.slice(0, 60) + '...' : tc.arguments})
          </span>
        )}
      </div>
      {tc.status === 'calling' && (
        <div className="text-slate-500 text-xs mt-0.5 animate-pulse">Running...</div>
      )}
      {tc.status === 'completed' && tc.result && (
        <div
          className={`text-xs mt-0.5 break-words ${hasLongResult ? 'cursor-pointer select-none' : ''}`}
          onClick={hasLongResult ? onToggle : undefined}
        >
          <span className="text-slate-500 mr-1">
            {hasLongResult ? (expanded ? '\u25BE' : '\u25B8') : '\u2192'}
          </span>
          {expanded ? (
            <pre className="text-slate-300 mt-1 overflow-x-auto whitespace-pre-wrap font-mono inline">{tc.result}</pre>
          ) : (
            <span className="text-slate-300">
              {tc.result.length > 120 ? tc.result.slice(0, 120) + '...' : tc.result}
            </span>
          )}
        </div>
      )}
      {tc.status === 'error' && tc.result && (
        <div
          className={`text-xs mt-0.5 break-words ${hasLongResult ? 'cursor-pointer select-none' : ''}`}
          onClick={hasLongResult ? onToggle : undefined}
        >
          <span className="mr-1">
            {hasLongResult ? (expanded ? '\u25BE' : '\u25B8') : '\u2192'}
          </span>
          {expanded ? (
            <pre className="text-red-400 mt-1 overflow-x-auto whitespace-pre-wrap font-mono inline">{tc.result}</pre>
          ) : (
            <span className="text-red-400">
              {tc.result.length > 120 ? tc.result.slice(0, 120) + '...' : tc.result}
            </span>
          )}
        </div>
      )}
    </div>
  );
}

function AskUserBlock({ segment, onAnswer }: {
  segment: Extract<MessageSegment, { type: 'ask_user' }>;
  onAnswer: (callId: string, answer: string) => void;
}) {
  return (
    <div className="bg-blue-900/20 border-l-2 border-blue-400/50 rounded px-3 py-2 my-2">
      {segment.questions.map((q, i) => (
        <div key={i} className="mb-2 last:mb-0">
          <p className="text-sm text-brand-light mb-1.5">{q.text}</p>
          {!segment.answered && q.options && (
            <div className="flex flex-wrap gap-1.5">
              {q.options.map((opt) => (
                <button
                  key={opt.label}
                  onClick={() => onAnswer(segment.callId, opt.label)}
                  className="rounded-[2px] border border-blue-400/30 px-2.5 py-1 text-xs text-blue-300 hover:bg-blue-400/10 hover:border-blue-400/50 transition-colors"
                >
                  {opt.label}
                </button>
              ))}
            </div>
          )}
          {segment.answered && segment.answer && (
            <div className="text-xs text-blue-300 mt-1">
              <span className="text-blue-400/60 mr-1">{'\u21B3'}</span>
              {segment.answer}
            </div>
          )}
        </div>
      ))}
    </div>
  );
}

function isJsonToolResult(text: string): boolean {
  const t = text.trimStart();
  return t.startsWith('{') &&
    (t.includes('"id"') || t.includes('"count"') || t.includes('"products"') ||
     t.includes('"employee_id"') || t.includes('"inventory"') || t.includes('"entries"'));
}

export function ExampleChat({ agentName, apiUrl, suggestions }: ExampleChatProps) {
  const { isAuthenticated, triggerAuthPopup } = useAuth();
  const [messages, setMessages] = useState<ChatMessage[]>([]);
  const [input, setInput] = useState('');
  const [isStreaming, setIsStreaming] = useState(false);
  const [messagesRemaining, setMessagesRemaining] = useState(MAX_MESSAGES_PER_HOUR);
  const [sessionId, setSessionId] = useState<string | null>(null);
  const [error, setError] = useState<string | null>(null);
  const [expandedToolIds, setExpandedToolIds] = useState<Set<string>>(new Set());
  const messagesEndRef = useRef<HTMLDivElement>(null);
  const abortRef = useRef<AbortController | null>(null);

  // Sync rate limit from server on mount
  useEffect(() => {
    const token = localStorage.getItem(STORAGE_KEY_ACCESS);
    if (!token) return;
    fetch(`${apiUrl}/v1/health`, {
      headers: { 'Authorization': `Bearer ${token}` },
    })
      .then(r => r.json())
      .then(data => {
        if (data?.rate_limit?.remaining != null) {
          setMessagesRemaining(data.rate_limit.remaining);
        }
      })
      .catch(() => { /* ignore */ });
  }, [apiUrl]);

  const toggleToolExpand = useCallback((toolId: string) => {
    setExpandedToolIds(prev => {
      const next = new Set(prev);
      if (next.has(toolId)) {
        next.delete(toolId);
      } else {
        next.add(toolId);
      }
      return next;
    });
  }, []);

  const respondToAskUser = useCallback(async (callId: string, answer: string) => {
    if (!sessionId) return;
    const token = localStorage.getItem(STORAGE_KEY_ACCESS);
    try {
      await fetch(`${apiUrl}/v1/sessions/${sessionId}/respond`, {
        method: 'POST',
        headers: {
          'Content-Type': 'application/json',
          ...(token ? { 'Authorization': `Bearer ${token}` } : {}),
        },
        body: JSON.stringify({ call_id: callId, response: answer }),
      });
      setMessages(prev => prev.map(m => ({
        ...m,
        segments: m.segments.map(seg =>
          seg.type === 'ask_user' && seg.callId === callId
            ? { ...seg, answered: true, answer }
            : seg
        ),
      })));
    } catch (err) {
      console.error('Failed to respond to ask_user:', err);
    }
  }, [sessionId, apiUrl]);

  useEffect(() => {
    messagesEndRef.current?.scrollIntoView({ behavior: 'smooth' });
  }, [messages]);

  const streamChat = useCallback(async (userMessage: string, currentSessionId: string | null) => {
    const assistantId = crypto.randomUUID();
    setMessages(prev => [...prev, { id: assistantId, role: 'assistant', content: '', segments: [] }]);
    setIsStreaming(true);
    setError(null);

    const controller = new AbortController();
    abortRef.current = controller;

    try {
      let token = localStorage.getItem(STORAGE_KEY_ACCESS);
      if (!token) {
        const refreshToken = localStorage.getItem(STORAGE_KEY_REFRESH);
        if (refreshToken) {
          try {
            token = await refreshAccessToken(refreshToken);
            localStorage.setItem(STORAGE_KEY_ACCESS, token);
          } catch { /* refresh failed */ }
        }
      }
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
        setMessagesRemaining(0);
        setError('Rate limit exceeded. Try again later.');
        setMessages(prev => prev.filter(m => m.id !== assistantId));
        setIsStreaming(false);
        return;
      }

      if (response.status === 401) {
        const refreshToken = localStorage.getItem(STORAGE_KEY_REFRESH);
        if (refreshToken && token) {
          try {
            const newToken = await refreshAccessToken(refreshToken);
            localStorage.setItem(STORAGE_KEY_ACCESS, newToken);
            setMessages(prev => prev.filter(m => m.id !== assistantId));
            setIsStreaming(false);
            streamChat(userMessage, currentSessionId);
            return;
          } catch { /* refresh failed */ }
        }
        setError('Authentication required. Please sign in again.');
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
      let currentEvent = '';
      // Track segments in order: text chunks accumulate into last text segment,
      // tool events insert tool segments breaking the text flow.
      const segments: MessageSegment[] = [];
      let currentText = '';

      const flushText = () => {
        if (currentText) {
          segments.push({ type: 'text', content: currentText });
          currentText = '';
        }
      };

      const updateMessage = () => {
        const content = segments
          .filter((s): s is { type: 'text'; content: string } => s.type === 'text')
          .map(s => s.content)
          .join('');
        setMessages(prev =>
          prev.map(m => m.id === assistantId ? { ...m, content, segments: [...segments] } : m)
        );
      };

      while (true) {
        const { done, value } = await reader.read();
        if (done) break;

        buffer += decoder.decode(value, { stream: true });
        const lines = buffer.split('\n');
        buffer = lines.pop() || '';

        for (const line of lines) {
          if (line.startsWith('event: ')) {
            currentEvent = line.slice(7).trim();
            continue;
          }

          if (line.startsWith('data: ')) {
            try {
              const data = JSON.parse(line.slice(6));

              if (currentEvent === 'tool_call') {
                flushText(); // Close current text segment before tool
                const argsStr = data.arguments
                  ? (typeof data.arguments === 'string' ? data.arguments : JSON.stringify(data.arguments))
                  : undefined;
                segments.push({
                  type: 'tool',
                  toolCall: {
                    id: data.call_id || crypto.randomUUID(),
                    tool: data.tool || 'unknown',
                    arguments: argsStr,
                    status: 'calling',
                  },
                });
                updateMessage();
              } else if (currentEvent === 'tool_result') {
                // Find matching tool segment and update
                const callId = data.call_id;
                const hasError = data.has_error === true;
                for (let i = segments.length - 1; i >= 0; i--) {
                  const seg = segments[i];
                  if (seg.type === 'tool' && seg.toolCall.id === callId) {
                    seg.toolCall.status = hasError ? 'error' : 'completed';
                    seg.toolCall.result = data.content || '';
                    break;
                  }
                }
                updateMessage();
              } else if (currentEvent === 'confirmation') {
                flushText();
                try {
                  const questions = JSON.parse(data.content) as AskUserQuestion[];
                  segments.push({
                    type: 'ask_user',
                    callId: data.call_id || '',
                    questions,
                    answered: false,
                  });
                } catch {
                  // If content isn't valid JSON questions, show as text
                  currentText += data.content || '';
                }
                updateMessage();
              } else if (data.content && currentEvent !== 'message') {
                const chunk = data.content as string;
                if (!isJsonToolResult(chunk)) {
                  currentText += chunk;
                  // Update with accumulated text (don't flush yet — more chunks coming)
                  const allSegments = [...segments, { type: 'text' as const, content: currentText }];
                  const content = allSegments
                    .filter((s): s is { type: 'text'; content: string } => s.type === 'text')
                    .map(s => s.content)
                    .join('');
                  setMessages(prev =>
                    prev.map(m => m.id === assistantId ? { ...m, content, segments: allSegments } : m)
                  );
                }
              }

              if (data.session_id) setSessionId(data.session_id);
              if (data.error) setError(data.error);
            } catch {
              // skip non-JSON data lines
            }
            currentEvent = '';
          }
        }
      }
      // Flush any remaining text
      flushText();
      updateMessage();
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
        segments: [{ type: 'text', content: trimmed }],
      };

      setMessages(prev => [...prev, userMsg]);
      setMessagesRemaining(prev => Math.max(0, prev - 1));
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
    <div className="rounded-[2px] border border-brand-shade3/15 bg-brand-dark flex flex-col overflow-hidden" style={{ height: '480px' }}>
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
                  className="rounded-[2px] border border-brand-shade3/20 px-3 py-2 text-xs text-brand-shade2 hover:text-brand-light hover:border-brand-accent/40 hover:bg-brand-accent/5 transition-colors text-left"
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
              className={`max-w-[80%] rounded-[2px] px-4 py-2.5 text-sm leading-relaxed ${
                msg.role === 'user'
                  ? 'bg-brand-accent text-white whitespace-pre-wrap'
                  : 'bg-brand-dark-alt text-brand-light border border-brand-shade3/15'
              }`}
            >
              {msg.role === 'assistant' ? (
                // Render segments in order: text and tool calls interleaved
                <>
                  {msg.segments.map((seg, i) =>
                    seg.type === 'tool' ? (
                      <ToolCallBlock
                        key={seg.toolCall.id}
                        tc={seg.toolCall}
                        expanded={expandedToolIds.has(seg.toolCall.id)}
                        onToggle={() => toggleToolExpand(seg.toolCall.id)}
                      />
                    ) : seg.type === 'ask_user' ? (
                      <AskUserBlock key={seg.callId} segment={seg} onAnswer={respondToAskUser} />
                    ) : (
                      seg.content ? (
                        <span key={i} className="whitespace-pre-wrap">{seg.content}</span>
                      ) : null
                    )
                  )}
                  {isStreaming && msg.id === messages[messages.length - 1]?.id && (
                    <span className="inline-block w-1.5 h-4 bg-brand-accent ml-0.5 animate-pulse" />
                  )}
                </>
              ) : (
                <span className="whitespace-pre-wrap">{msg.content}</span>
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
            className="flex-1 rounded-[2px] border border-brand-shade3/20 bg-brand-dark-alt px-4 py-2 text-sm text-brand-light placeholder:text-brand-shade3 focus:outline-none focus:border-brand-accent/50 disabled:opacity-50 transition-colors"
          />
          <button
            type="submit"
            disabled={!input.trim() || isStreaming || messagesRemaining <= 0}
            className="rounded-[2px] bg-brand-accent px-4 py-2 text-sm font-medium text-white hover:bg-brand-accent-hover disabled:opacity-40 disabled:cursor-not-allowed transition-colors"
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
