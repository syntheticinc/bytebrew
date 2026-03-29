import { useState, useRef, useEffect, useCallback } from 'react';
import { useAuth } from '../lib/auth';
import { refreshAccessToken } from '../api/auth';

/* ------------------------------------------------------------------ */
/*  Types                                                              */
/* ------------------------------------------------------------------ */

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

/* ------------------------------------------------------------------ */
/*  Constants                                                          */
/* ------------------------------------------------------------------ */

const MAX_MESSAGES_PER_HOUR = 15;
const RATE_LIMIT_WINDOW_MS = 60 * 60 * 1000;
const STORAGE_KEY_ACCESS = 'bytebrew_access_token';
const STORAGE_KEY_REFRESH = 'bytebrew_refresh_token';
const STORAGE_KEY_RATE = 'bytebrew_rate_limit';

const MUTED = '#87867F';
const SURFACE = 'rgba(30,30,30,0.6)';
const BORDER_TOOL = 'rgba(135,134,127,0.25)';
const BORDER_DONE = 'rgba(135,134,127,0.15)';
const ACCENT = '#D7513E';

/* ------------------------------------------------------------------ */
/*  Rate limiting helpers                                              */
/* ------------------------------------------------------------------ */

function getRateLimit(): { remaining: number; resetAt: number } {
  try {
    const raw = localStorage.getItem(STORAGE_KEY_RATE);
    if (raw) {
      const { remaining, resetAt } = JSON.parse(raw);
      if (Date.now() < resetAt) return { remaining, resetAt };
    }
  } catch { /* ignore */ }
  const resetAt = Date.now() + RATE_LIMIT_WINDOW_MS;
  const state = { remaining: MAX_MESSAGES_PER_HOUR, resetAt };
  localStorage.setItem(STORAGE_KEY_RATE, JSON.stringify(state));
  return state;
}

function decrementRateLimit(): number {
  const state = getRateLimit();
  state.remaining = Math.max(0, state.remaining - 1);
  localStorage.setItem(STORAGE_KEY_RATE, JSON.stringify(state));
  return state.remaining;
}

function setRateLimitRemaining(remaining: number): void {
  const state = getRateLimit();
  state.remaining = remaining;
  localStorage.setItem(STORAGE_KEY_RATE, JSON.stringify(state));
}

/* ------------------------------------------------------------------ */
/*  Tiny markdown renderer                                             */
/* ------------------------------------------------------------------ */

function renderMarkdown(raw: string): React.ReactNode[] {
  const lines = raw.split('\n');
  const nodes: React.ReactNode[] = [];

  for (let i = 0; i < lines.length; i++) {
    const line = lines[i];
    if (line === '') { nodes.push(<br key={`br-${i}`} />); continue; }

    const listMatch = line.match(/^(\d+)\.\s+(.+)$/);
    if (listMatch) {
      nodes.push(
        <div key={`li-${i}`} className="pl-3">
          <span className="mr-1" style={{ color: MUTED }}>{listMatch[1]}.</span>
          {inlineBold(listMatch[2])}
        </div>,
      );
      continue;
    }
    nodes.push(<div key={`ln-${i}`}>{inlineBold(line)}</div>);
  }
  return nodes;
}

function inlineBold(text: string): React.ReactNode {
  const parts = text.split(/\*\*(.+?)\*\*/g);
  if (parts.length === 1) return text;
  return parts.map((p, i) => (i % 2 === 1 ? <strong key={i} style={{ color: '#F7F8F1' }}>{p}</strong> : p));
}

/* ------------------------------------------------------------------ */
/*  Visual sub-components                                              */
/* ------------------------------------------------------------------ */

function StatusDot({ status }: { status: 'calling' | 'completed' | 'error' }) {
  const color = status === 'completed' ? '#4ade80' : status === 'error' ? '#ef4444' : MUTED;
  const glow = status === 'completed' ? '0 0 4px rgba(74,222,128,0.4)' : status === 'error' ? '0 0 4px rgba(239,68,68,0.4)' : 'none';
  return (
    <span
      className="inline-block w-1.5 h-1.5 rounded-full mr-1.5 shrink-0"
      style={{ backgroundColor: color, boxShadow: glow }}
    />
  );
}

const BREW_PHRASES = ['Grinding beans...', 'Brewing...', 'Pulling a shot...', 'Steaming...', 'Almost ready...'];
let brewCounter = 0;

function BrewingSpinner() {
  const phrase = BREW_PHRASES[brewCounter++ % BREW_PHRASES.length];
  return (
    <div className="flex items-center gap-2 py-2">
      <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="1.5" className="w-4 h-4" style={{ color: MUTED }}>
        <path d="M17 8h1a4 4 0 010 8h-1" strokeLinecap="round" />
        <path d="M3 8h14v9a4 4 0 01-4 4H7a4 4 0 01-4-4V8z" />
        <path d="M7 2v3" strokeLinecap="round" style={{ animation: 'heroSteam 1.2s ease-in-out infinite' }} />
        <path d="M10 1v3" strokeLinecap="round" style={{ animation: 'heroSteam 1.2s ease-in-out infinite 0.3s' }} />
        <path d="M13 2v3" strokeLinecap="round" style={{ animation: 'heroSteam 1.2s ease-in-out infinite 0.6s' }} />
      </svg>
      <span className="text-xs font-mono animate-pulse" style={{ color: MUTED }}>{phrase}</span>
      <style>{`
        @keyframes heroSteam {
          0% { opacity: 0.3; transform: translateY(0); }
          50% { opacity: 1; transform: translateY(-3px); }
          100% { opacity: 0.3; transform: translateY(0); }
        }
      `}</style>
    </div>
  );
}

function ToolCallBlock({ tc, expanded, onToggle }: { tc: ToolCallInfo; expanded: boolean; onToggle: () => void }) {
  const isDone = tc.status === 'completed' || tc.status === 'error';
  const resultColor = tc.status === 'error' ? 'rgba(239,68,68,0.7)' : 'rgba(135,134,127,0.6)';

  return (
    <div
      className="rounded-[2px] border-l-2 px-3 py-1.5 text-xs font-mono my-1.5 cursor-pointer select-none"
      style={{ borderColor: isDone ? BORDER_DONE : BORDER_TOOL, backgroundColor: SURFACE }}
      onClick={onToggle}
    >
      {/* Collapsed: single truncated line */}
      {!expanded && (
        <div className="truncate" style={{ color: MUTED }}>
          <span className="inline-flex items-center">
            <StatusDot status={tc.status} />
            {tc.tool}
          </span>
          {tc.arguments && (
            <span style={{ color: 'rgba(135,134,127,0.4)' }}> ({tc.arguments})</span>
          )}
          {isDone && tc.result && (
            <span style={{ color: resultColor }}> &mdash; {tc.result}</span>
          )}
          {tc.status === 'calling' && (
            <span className="animate-pulse ml-1"> running...</span>
          )}
        </div>
      )}
      {/* Expanded: full details */}
      {expanded && (
        <>
          <div className="flex items-center" style={{ color: MUTED }}>
            <StatusDot status={tc.status} />
            <span>{tc.tool}</span>
            <span className="ml-1" style={{ color: resultColor }}>{'\u25BE'}</span>
          </div>
          {tc.arguments && (
            <div className="mt-1 whitespace-pre-wrap break-all" style={{ color: 'rgba(135,134,127,0.5)' }}>
              {tc.arguments}
            </div>
          )}
          {tc.result && (
            <pre className="mt-1 whitespace-pre-wrap break-words" style={{ color: resultColor }}>
              {tc.result}
            </pre>
          )}
        </>
      )}
    </div>
  );
}

function AskUserBlock({ segment, onAnswer }: {
  segment: Extract<MessageSegment, { type: 'ask_user' }>;
  onAnswer: (callId: string, answer: string) => void;
}) {
  return (
    <div className="space-y-2 my-2">
      {segment.questions.map((q, i) => (
        <div key={i}>
          <div className="text-sm" style={{ color: '#DFD8D0' }}>{q.text}</div>
          {!segment.answered && q.options && (
            <div className="flex flex-wrap gap-2 mt-1.5">
              {q.options.map((opt) => (
                <button
                  key={opt.label}
                  onClick={() => onAnswer(segment.callId, opt.label)}
                  className="rounded-[2px] border px-3 py-1 text-xs transition-all duration-300 cursor-pointer"
                  style={{
                    borderColor: 'rgba(135,134,127,0.25)',
                    color: '#CBC9BC',
                    backgroundColor: 'transparent',
                  }}
                  onMouseEnter={(e) => { e.currentTarget.style.borderColor = 'rgba(215,81,62,0.3)'; e.currentTarget.style.color = '#DFD8D0'; }}
                  onMouseLeave={(e) => { e.currentTarget.style.borderColor = 'rgba(135,134,127,0.25)'; e.currentTarget.style.color = '#CBC9BC'; }}
                >
                  {opt.label}
                </button>
              ))}
            </div>
          )}
          {segment.answered && q.options && (
            <div className="flex flex-wrap gap-2 mt-1.5">
              {q.options.map((opt) => {
                const isSelected = segment.answer === opt.label;
                return (
                  <span
                    key={opt.label}
                    className="rounded-[2px] border px-3 py-1 text-xs transition-all duration-300"
                    style={{
                      borderColor: isSelected ? ACCENT : 'rgba(135,134,127,0.25)',
                      backgroundColor: isSelected ? ACCENT : 'transparent',
                      color: isSelected ? '#fff' : 'rgba(135,134,127,0.5)',
                      opacity: isSelected ? 1 : 0.6,
                    }}
                  >
                    {opt.label}
                  </span>
                );
              })}
            </div>
          )}
        </div>
      ))}
    </div>
  );
}

/* ------------------------------------------------------------------ */
/*  Main Component                                                     */
/* ------------------------------------------------------------------ */

export function ExampleChat({ agentName, apiUrl, suggestions }: ExampleChatProps) {
  const { isAuthenticated, triggerAuthPopup } = useAuth();
  const [messages, setMessages] = useState<ChatMessage[]>([]);
  const [input, setInput] = useState('');
  const [isStreaming, setIsStreaming] = useState(false);
  const [messagesRemaining, setMessagesRemaining] = useState(() => getRateLimit().remaining);
  const [sessionId, setSessionId] = useState<string | null>(null);
  const [error, setError] = useState<string | null>(null);
  const [expandedToolIds, setExpandedToolIds] = useState<Set<string>>(new Set());
  const messagesEndRef = useRef<HTMLDivElement>(null);
  const abortRef = useRef<AbortController | null>(null);

  useEffect(() => {
    const token = localStorage.getItem(STORAGE_KEY_ACCESS);
    if (!token) return;
    fetch(`${apiUrl}/v1/health`, {
      headers: { 'Authorization': `Bearer ${token}` },
    })
      .then(r => r.json())
      .then(data => {
        if (data?.rate_limit?.remaining != null) {
          setRateLimitRemaining(data.rate_limit.remaining);
          setMessagesRemaining(data.rate_limit.remaining);
        }
      })
      .catch(() => { /* ignore */ });
  }, [apiUrl]);

  const toggleToolExpand = useCallback((toolId: string) => {
    setExpandedToolIds(prev => {
      const next = new Set(prev);
      if (next.has(toolId)) next.delete(toolId);
      else next.add(toolId);
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

      const rateLimitRemaining = response.headers.get('X-RateLimit-Remaining');
      if (rateLimitRemaining != null) {
        const remaining = parseInt(rateLimitRemaining, 10);
        if (!isNaN(remaining)) {
          setRateLimitRemaining(remaining);
          setMessagesRemaining(remaining);
        }
      }

      if (response.status === 429) {
        setRateLimitRemaining(0);
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
      const segments: MessageSegment[] = [];
      let currentText = '';

      const render = () => {
        const allSegments = [...segments, ...(currentText ? [{ type: 'text' as const, content: currentText }] : [])];
        const content = allSegments
          .filter((s): s is { type: 'text'; content: string } => s.type === 'text')
          .map(s => s.content)
          .join('');
        setMessages(prev =>
          prev.map(m => m.id === assistantId ? { ...m, content, segments: allSegments } : m)
        );
      };

      while (true) {
        const { done, value } = await reader.read();
        if (done) break;

        const raw = decoder.decode(value, { stream: true });
        buffer += raw;
        const lines = buffer.split('\n');
        buffer = lines.pop() || '';

        for (const line of lines) {
          if (line.startsWith('event: ')) {
            currentEvent = line.slice(7).trim();
            continue;
          }
          if (!line.startsWith('data: ')) continue;

          try {
            const data = JSON.parse(line.slice(6));

            if (currentEvent === 'message_delta') {
              currentText += (data.content as string) || '';
            } else if (currentEvent === 'tool_call') {
              if (currentText) { segments.push({ type: 'text', content: currentText }); currentText = ''; }
              const argsStr = data.arguments
                ? (typeof data.arguments === 'string' ? data.arguments as string : JSON.stringify(data.arguments))
                : undefined;
              segments.push({
                type: 'tool',
                toolCall: {
                  id: (data.call_id as string) || crypto.randomUUID(),
                  tool: (data.tool as string) || 'unknown',
                  arguments: argsStr,
                  status: 'calling',
                },
              });
              render(); // show tool call immediately
            } else if (currentEvent === 'tool_result') {
              const callId = data.call_id as string;
              for (let i = segments.length - 1; i >= 0; i--) {
                const seg = segments[i];
                if (seg.type === 'tool' && seg.toolCall.id === callId) {
                  seg.toolCall.status = data.has_error ? 'error' : 'completed';
                  seg.toolCall.result = (data.content as string) || '';
                  break;
                }
              }
              render(); // show tool result immediately
            } else if (currentEvent === 'confirmation') {
              if (currentText) { segments.push({ type: 'text', content: currentText }); currentText = ''; }
              try {
                const questions = JSON.parse(data.content as string) as AskUserQuestion[];
                segments.push({ type: 'ask_user', callId: (data.call_id as string) || '', questions, answered: false });
              } catch { currentText += (data.content as string) || ''; }
            }

            if (data.session_id) setSessionId(data.session_id);
            if (data.error) setError(data.error);
          } catch { /* skip */ }
          currentEvent = '';
        }

        render();
      }

      if (currentText) { segments.push({ type: 'text', content: currentText }); currentText = ''; }
      render();
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
      setMessagesRemaining(decrementRateLimit());
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
  const lastMsg = messages[messages.length - 1];
  const isLastAssistantStreaming = isStreaming && lastMsg?.role === 'assistant';

  return (
    <div
      className="rounded-[2px] border overflow-hidden flex flex-col shadow-lg"
      style={{ borderColor: 'rgba(135,134,127,0.15)', backgroundColor: '#252525', height: '480px' }}
    >
      {/* Window chrome header */}
      <div className="flex items-center gap-3 px-4 py-2.5 border-b" style={{ borderColor: 'rgba(135,134,127,0.08)' }}>
        <div className="flex gap-1.5">
          <span className="w-2.5 h-2.5 rounded-full" style={{ backgroundColor: 'rgba(135,134,127,0.3)' }} />
          <span className="w-2.5 h-2.5 rounded-full" style={{ backgroundColor: 'rgba(135,134,127,0.3)' }} />
          <span className="w-2.5 h-2.5 rounded-full" style={{ backgroundColor: 'rgba(135,134,127,0.3)' }} />
        </div>
        <span className="text-xs font-mono" style={{ color: MUTED }}>
          ByteBrew Agent <span style={{ color: 'rgba(135,134,127,0.5)' }}>&middot; {agentName}</span>
        </span>
      </div>

      {/* Messages area */}
      <div
        className="flex-1 overflow-y-auto px-4 py-3 space-y-3"
        style={{ scrollbarWidth: 'thin', scrollbarColor: '#333 transparent' }}
      >
        {showSuggestions && (
          <div className="flex flex-col items-center justify-center h-full gap-4">
            <p className="text-sm font-mono" style={{ color: MUTED }}>Try one of these:</p>
            <div className="flex flex-wrap justify-center gap-2 max-w-lg">
              {suggestions.map((suggestion) => (
                <button
                  key={suggestion}
                  onClick={() => handleSend(suggestion)}
                  className="rounded-[2px] border px-3 py-1.5 text-xs font-mono transition-all duration-200 cursor-pointer text-left"
                  style={{
                    borderColor: 'rgba(135,134,127,0.25)',
                    color: '#CBC9BC',
                    backgroundColor: 'transparent',
                  }}
                  onMouseEnter={(e) => { e.currentTarget.style.borderColor = 'rgba(215,81,62,0.3)'; e.currentTarget.style.color = '#DFD8D0'; }}
                  onMouseLeave={(e) => { e.currentTarget.style.borderColor = 'rgba(135,134,127,0.25)'; e.currentTarget.style.color = '#CBC9BC'; }}
                >
                  {suggestion}
                </button>
              ))}
            </div>
          </div>
        )}

        {messages.map((msg) => (
          msg.role === 'user' ? (
            <div key={msg.id} className="flex justify-end">
              <div
                className="max-w-[80%] rounded-[2px] px-3 py-2 text-sm text-white whitespace-pre-wrap"
                style={{ backgroundColor: ACCENT }}
              >
                {msg.content}
              </div>
            </div>
          ) : (
            <div key={msg.id} className="space-y-1.5">
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
                ) : seg.content ? (
                  <div key={i} className="text-sm" style={{ color: '#DFD8D0' }}>
                    {renderMarkdown(seg.content)}
                  </div>
                ) : null
              )}
              {isLastAssistantStreaming && msg.id === lastMsg.id && (
                <BrewingSpinner />
              )}
            </div>
          )
        ))}

        {error && (
          <div className="text-center text-xs font-mono py-2" style={{ color: 'rgba(239,68,68,0.7)' }}>{error}</div>
        )}

        <div ref={messagesEndRef} />
      </div>

      {/* Input bar */}
      <div className="flex items-center gap-2 px-4 py-2.5 border-t" style={{ borderColor: 'rgba(135,134,127,0.08)' }}>
        <form onSubmit={handleSubmit} className="flex items-center gap-2 flex-1">
          <input
            type="text"
            value={input}
            onChange={(e) => setInput(e.target.value)}
            placeholder={messagesRemaining <= 0 ? 'Rate limit reached' : 'Type a message...'}
            disabled={isStreaming || messagesRemaining <= 0}
            className="flex-1 rounded-[2px] border px-3 py-1.5 text-xs font-mono focus:outline-none disabled:opacity-50 transition-colors"
            style={{
              borderColor: input ? 'rgba(215,81,62,0.3)' : 'rgba(135,134,127,0.12)',
              color: '#DFD8D0',
              backgroundColor: 'rgba(17,17,17,0.4)',
            }}
          />
          <button
            type="submit"
            disabled={!input.trim() || isStreaming || messagesRemaining <= 0}
            className="rounded-[2px] px-3 py-1.5 text-xs text-white shrink-0 disabled:opacity-40 disabled:cursor-not-allowed transition-colors"
            style={{ backgroundColor: ACCENT }}
          >
            Send
          </button>
        </form>
        <span className="text-[10px] font-mono shrink-0" style={{ color: 'rgba(135,134,127,0.4)' }}>
          {messagesRemaining}/{MAX_MESSAGES_PER_HOUR}
        </span>
      </div>
    </div>
  );
}
