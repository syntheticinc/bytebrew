import { useState, useRef, useEffect, useCallback } from 'react';
import { parseSSELine, type ToolCall } from '../../lib/sse';

// Simple markdown renderer: fenced code blocks, inline code, bold, bullet lists
function renderMarkdown(text: string): React.ReactNode {
  const parts = text.split(/(```[\s\S]*?```)/g);
  return parts.map((part, i) => {
    if (part.startsWith('```')) {
      const inner = part.replace(/^```\w*\n?/, '').replace(/\n?```$/, '');
      return (
        <pre key={i} className="bg-brand-dark border border-brand-shade3/20 rounded p-2 my-1 text-[10px] font-mono overflow-x-auto whitespace-pre">
          {inner}
        </pre>
      );
    }
    const lines = part.split('\n');
    return (
      <span key={i}>
        {lines.map((line, li) => {
          const isList = /^[\s]*[-*+] /.test(line);
          const content = isList ? line.replace(/^[\s]*[-*+] /, '') : line;
          const inlined = content.split(/(`[^`]+`|\*\*[^*]+\*\*)/g).map((seg, si) => {
            if (seg.startsWith('`') && seg.endsWith('`')) {
              return <code key={si} className="bg-brand-dark border border-brand-shade3/20 rounded px-1 text-[10px] font-mono text-status-active">{seg.slice(1, -1)}</code>;
            }
            if (seg.startsWith('**') && seg.endsWith('**')) {
              return <strong key={si} className="font-semibold text-brand-light">{seg.slice(2, -2)}</strong>;
            }
            return <span key={si}>{seg}</span>;
          });
          return (
            <span key={li} className={isList ? 'flex gap-1' : 'block'}>
              {isList && <span className="text-brand-shade3 flex-shrink-0">•</span>}
              <span>{inlined}</span>
              {!isList && li < lines.length - 1 && <br />}
            </span>
          );
        })}
      </span>
    );
  });
}

interface Message {
  id: string;
  role: 'user' | 'assistant';
  content: string;
  toolCalls?: ToolCall[];
  streaming?: boolean;
}

interface BuilderChatProps {
  agentName: string;
}

export default function BuilderChat({ agentName }: BuilderChatProps) {
  const [messages, setMessages] = useState<Message[]>([]);
  const [input, setInput] = useState('');
  const [streaming, setStreaming] = useState(false);
  const sessionIdRef = useRef<string>('');
  const abortRef = useRef<AbortController | null>(null);
  const bottomRef = useRef<HTMLDivElement>(null);
  const inputRef = useRef<HTMLTextAreaElement>(null);

  // Reset chat when agent changes
  useEffect(() => {
    setMessages([]);
    sessionIdRef.current = '';
    abortRef.current?.abort();
  }, [agentName]);

  useEffect(() => {
    bottomRef.current?.scrollIntoView({ behavior: 'smooth' });
  }, [messages]);

  const sendMessage = useCallback(async () => {
    const text = input.trim();
    if (!text || streaming) return;

    setInput('');
    setStreaming(true);
    abortRef.current = new AbortController();

    const userMsg: Message = {
      id: crypto.randomUUID(),
      role: 'user',
      content: text,
    };

    const assistantMsgId = crypto.randomUUID();
    const assistantMsg: Message = {
      id: assistantMsgId,
      role: 'assistant',
      content: '',
      toolCalls: [],
      streaming: true,
    };

    setMessages((prev) => [...prev, userMsg, assistantMsg]);

    try {
      const token = localStorage.getItem('jwt');
      const res = await fetch(`/api/v1/agents/${encodeURIComponent(agentName)}/chat`, {
        method: 'POST',
        headers: {
          'Content-Type': 'application/json',
          ...(token ? { Authorization: `Bearer ${token}` } : {}),
        },
        body: JSON.stringify({
          message: text,
          session_id: sessionIdRef.current || undefined,
        }),
        signal: abortRef.current.signal,
      });

      if (!res.ok || !res.body) {
        const errText = await res.text().catch(() => 'Request failed');
        sessionIdRef.current = '';
        setMessages((prev) =>
          prev.map((m) =>
            m.id === assistantMsgId
              ? { ...m, content: `Error: ${errText}`, streaming: false }
              : m,
          ),
        );
        return;
      }

      const reader = res.body.getReader();
      const decoder = new TextDecoder();
      let buffer = '';
      let currentEvent = '';
      let currentContent = '';
      let currentToolCalls: ToolCall[] = [];

      const updateAssistant = (patch: Partial<Message>) => {
        setMessages((prev) =>
          prev.map((m) => (m.id === assistantMsgId ? { ...m, ...patch } : m)),
        );
      };

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
              updateAssistant({ content: `Error: ${errContent}`, streaming: false });
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
        setMessages((prev) =>
          prev.map((m) =>
            m.id === assistantMsgId
              ? { ...m, content: 'Connection error', streaming: false }
              : m,
          ),
        );
      }
    } finally {
      setStreaming(false);
    }
  }, [agentName, input, streaming]);

  function handleKeyDown(e: React.KeyboardEvent<HTMLTextAreaElement>) {
    if (e.key === 'Enter' && !e.shiftKey) {
      e.preventDefault();
      sendMessage();
    }
  }

  function stopStreaming() {
    abortRef.current?.abort();
    setStreaming(false);
    setMessages((prev) =>
      prev.map((m) => (m.streaming ? { ...m, streaming: false } : m)),
    );
  }

  return (
    <div className="flex flex-col h-full">
      {/* Session indicator */}
      {sessionIdRef.current && (
        <div className="px-3 py-1.5 border-b border-brand-shade3/15 flex items-center justify-between">
          <span className="text-[10px] text-brand-shade3 font-mono truncate">
            session: {sessionIdRef.current.slice(0, 16)}…
          </span>
          <button
            onClick={() => { setMessages([]); sessionIdRef.current = ''; }}
            className="text-[10px] text-brand-shade3 hover:text-brand-light ml-2 flex-shrink-0"
          >
            new session
          </button>
        </div>
      )}

      {/* Messages */}
      <div className="flex-1 overflow-y-auto p-3 space-y-3 min-h-0">
        {messages.length === 0 && (
          <div className="text-center text-sm text-brand-shade3 mt-8 px-4">
            <p>Chat with <span className="text-brand-light font-medium">{agentName}</span></p>
            <p className="text-xs mt-1 text-brand-shade3/60">Test this agent's responses here. For platform configuration, use the AI Assistant (bottom-right button).</p>
          </div>
        )}

        {messages.map((msg) => (
          <div key={msg.id} className={msg.role === 'user' ? 'flex justify-end' : ''}>
            {msg.role === 'user' ? (
              <div className="max-w-[85%] px-3 py-2 bg-brand-accent/10 border border-brand-accent/20 rounded-card text-sm text-brand-light">
                {msg.content}
              </div>
            ) : (
              <div className="space-y-1.5">
                {/* Tool calls */}
                {msg.toolCalls && msg.toolCalls.length > 0 && (
                  <div className="space-y-1">
                    {msg.toolCalls.map((tc, i) => (
                      <div key={i} className="px-2 py-1.5 bg-brand-dark border border-brand-shade3/20 rounded text-[11px] font-mono">
                        <div className="text-blue-400 mb-0.5"><svg className="inline-block w-3 h-3 mr-1 -mt-0.5" viewBox="0 0 16 16" fill="currentColor"><path d="M8 4.754a3.246 3.246 0 1 0 0 6.492 3.246 3.246 0 0 0 0-6.492ZM5.754 8a2.246 2.246 0 1 1 4.492 0 2.246 2.246 0 0 1-4.492 0Z"/><path d="M9.796 1.343c-.527-1.79-3.065-1.79-3.592 0l-.094.319a.873.873 0 0 1-1.255.52l-.292-.16c-1.64-.892-3.433.902-2.54 2.541l.159.292a.873.873 0 0 1-.52 1.255l-.319.094c-1.79.527-1.79 3.065 0 3.592l.319.094a.873.873 0 0 1 .52 1.255l-.16.292c-.892 1.64.902 3.434 2.541 2.54l.292-.159a.873.873 0 0 1 1.255.52l.094.319c.527 1.79 3.065 1.79 3.592 0l.094-.319a.873.873 0 0 1 1.255-.52l.292.16c1.64.893 3.434-.902 2.54-2.541l-.159-.292a.873.873 0 0 1 .52-1.255l.319-.094c1.79-.527 1.79-3.065 0-3.592l-.319-.094a.873.873 0 0 1-.52-1.255l.16-.292c.893-1.64-.902-3.433-2.541-2.54l-.292.159a.873.873 0 0 1-1.255-.52l-.094-.319Zm-2.633.283c.246-.835 1.428-.835 1.674 0l.094.319a1.873 1.873 0 0 0 2.693 1.115l.291-.16c.764-.415 1.6.422 1.184 1.185l-.159.292a1.873 1.873 0 0 0 1.116 2.692l.318.094c.835.246.835 1.428 0 1.674l-.319.094a1.873 1.873 0 0 0-1.115 2.693l.16.291c.415.764-.422 1.6-1.185 1.184l-.292-.159a1.873 1.873 0 0 0-2.692 1.116l-.094.318c-.246.835-1.428.835-1.674 0l-.094-.319a1.873 1.873 0 0 0-2.693-1.115l-.291.16c-.764.415-1.6-.422-1.184-1.185l.159-.292A1.873 1.873 0 0 0 1.945 9.89l-.318-.094c-.835-.246-.835-1.428 0-1.674l.319-.094a1.873 1.873 0 0 0 1.115-2.693l-.16-.291c-.415-.764.422-1.6 1.185-1.184l.292.159a1.873 1.873 0 0 0 2.692-1.116l.094-.318Z"/></svg>{tc.tool}</div>
                        {tc.input && (
                          <div className="text-brand-shade3 truncate">{tc.input.slice(0, 80)}{tc.input.length > 80 ? '…' : ''}</div>
                        )}
                        {tc.output !== undefined && (
                          <div className="text-status-active mt-0.5 truncate">→ {tc.output.slice(0, 80)}{tc.output.length > 80 ? '…' : ''}</div>
                        )}
                      </div>
                    ))}
                  </div>
                )}

                {/* Message content */}
                {(msg.content || msg.streaming) && (
                  <div className="text-sm text-brand-light leading-relaxed">
                    {msg.streaming ? msg.content : renderMarkdown(msg.content)}
                    {msg.streaming && (
                      <span className="inline-block w-1.5 h-3.5 bg-brand-accent ml-0.5 animate-pulse" />
                    )}
                  </div>
                )}
              </div>
            )}
          </div>
        ))}
        <div ref={bottomRef} />
      </div>

      {/* Input */}
      <div className="border-t border-brand-shade3/15 p-3">
        <div className="flex gap-2 items-end">
          <textarea
            ref={inputRef}
            value={input}
            onChange={(e) => setInput(e.target.value)}
            onKeyDown={handleKeyDown}
            placeholder="Message..."
            rows={1}
            disabled={streaming}
            className="flex-1 px-3 py-2 bg-brand-dark border border-brand-shade3/30 rounded-card text-sm text-brand-light placeholder-brand-shade3 focus:outline-none focus:border-brand-accent focus:ring-1 focus:ring-brand-accent resize-none disabled:opacity-50 transition-colors"
            style={{ maxHeight: '80px', overflowY: 'auto' }}
          />
          {streaming ? (
            <button
              onClick={stopStreaming}
              className="px-3 py-2 bg-brand-dark border border-brand-shade3/30 rounded-card text-sm text-brand-shade2 hover:text-brand-light hover:border-brand-shade3 transition-colors flex-shrink-0"
              title="Stop"
            >
              <svg width="14" height="14" viewBox="0 0 24 24" fill="currentColor">
                <rect x="4" y="4" width="16" height="16" rx="2" />
              </svg>
            </button>
          ) : (
            <button
              onClick={sendMessage}
              disabled={!input.trim()}
              className="px-3 py-2 bg-brand-accent text-brand-light rounded-card text-sm hover:bg-brand-accent-hover disabled:opacity-40 transition-colors flex-shrink-0"
            >
              <svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round">
                <line x1="22" y1="2" x2="11" y2="13" />
                <polygon points="22 2 15 22 11 13 2 9 22 2" />
              </svg>
            </button>
          )}
        </div>
        <p className="text-[10px] text-brand-shade3/50 mt-1">Enter to send · Shift+Enter for newline</p>
      </div>
    </div>
  );
}
