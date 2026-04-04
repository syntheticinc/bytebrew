import { useState, useRef, useEffect, useCallback } from 'react';
import { useSSEChat } from '../../hooks/useSSEChat';

// ─── Types ──────────────────────────────────────────────────────────────────

interface BuilderAssistantProps {
  onClose: () => void;
  onConfigChanged: () => void;
  flowTestOpen?: boolean;
}

// ─── Constants ──────────────────────────────────────────────────────────────

const ASSISTANT_AGENT = 'builder-assistant';

// ─── Assistant Toggle Button ────────────────────────────────────────────────

export function AssistantToggleButton({ onClick }: { onClick: () => void }) {
  return (
    <button
      onClick={onClick}
      className="fixed bottom-6 right-6 z-60 w-12 h-12 bg-brand-accent text-brand-light rounded-full shadow-lg hover:bg-brand-accent-hover transition-all hover:scale-105 flex items-center justify-center"
      title="AI Assistant"
    >
      {/* Sparkles / AI icon */}
      <svg width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round">
        <path d="M12 2l2.4 7.4H22l-6.2 4.5 2.4 7.4L12 17l-6.2 4.3 2.4-7.4L2 9.4h7.6z" />
      </svg>
    </button>
  );
}

// ─── Main Component ─────────────────────────────────────────────────────────

export default function BuilderAssistant({ onClose, onConfigChanged, flowTestOpen }: BuilderAssistantProps) {
  const [input, setInput] = useState('');
  const [agentStatus, setAgentStatus] = useState<'loading' | 'not_found' | 'no_model' | 'ready'>('loading');
  const [minimized, setMinimized] = useState(false);

  // Resize state
  const [panelHeight, setPanelHeight] = useState(500);
  const isDraggingRef = useRef(false);
  const dragStartYRef = useRef(0);
  const dragStartHeightRef = useRef(0);

  const textareaRef = useRef<HTMLTextAreaElement>(null);
  const bottomRef = useRef<HTMLDivElement>(null);
  const onConfigChangedRef = useRef(onConfigChanged);
  const debounceTimerRef = useRef<ReturnType<typeof setTimeout> | null>(null);

  // Keep ref in sync so the debounce closure doesn't go stale
  useEffect(() => {
    onConfigChangedRef.current = onConfigChanged;
  }, [onConfigChanged]);

  const { messages, sendMessage, isStreaming, resetSession, stopStreaming } = useSSEChat({
    endpoint: `/api/v1/agents/${encodeURIComponent(ASSISTANT_AGENT)}/chat`,
    agentName: ASSISTANT_AGENT,
  });

  // ── Tool call expand state ───────────────────────────────────────────────
  const [expandedToolCalls, setExpandedToolCalls] = useState<Record<string, boolean>>({});

  function toggleToolCall(key: string) {
    setExpandedToolCalls((prev) => ({ ...prev, [key]: !prev[key] }));
  }

  // Check if builder-assistant agent exists and has a model
  useEffect(() => {
    let cancelled = false;
    const token = localStorage.getItem('jwt');

    fetch(`/api/v1/agents/${encodeURIComponent(ASSISTANT_AGENT)}`, {
      headers: token ? { Authorization: `Bearer ${token}` } : {},
    })
      .then(async (res) => {
        if (cancelled) return;
        if (!res.ok) {
          setAgentStatus('not_found');
          return;
        }
        try {
          const data = await res.json();
          if (data.model_id == null || data.model_id === 0) {
            setAgentStatus('no_model');
          } else {
            setAgentStatus('ready');
          }
        } catch {
          setAgentStatus('ready');
        }
      })
      .catch(() => {
        if (!cancelled) setAgentStatus('not_found');
      });

    return () => { cancelled = true; };
  }, []);

  // Auto-scroll
  useEffect(() => {
    bottomRef.current?.scrollIntoView({ behavior: 'smooth' });
  }, [messages]);

  // Debounced onConfigChanged after assistant response finishes
  useEffect(() => {
    if (!isStreaming && messages.length > 0) {
      if (debounceTimerRef.current) clearTimeout(debounceTimerRef.current);
      debounceTimerRef.current = setTimeout(() => {
        onConfigChangedRef.current();
      }, 500);
    }
    return () => {
      if (debounceTimerRef.current) clearTimeout(debounceTimerRef.current);
    };
  }, [isStreaming, messages.length]);

  // Textarea auto-grow
  useEffect(() => {
    const ta = textareaRef.current;
    if (!ta) return;
    ta.style.height = 'auto';
    const lineHeight = 20;
    const maxHeight = lineHeight * 4 + 16; // 4 rows + padding
    ta.style.height = Math.min(ta.scrollHeight, maxHeight) + 'px';
    ta.style.overflowY = ta.scrollHeight > maxHeight ? 'auto' : 'hidden';
  }, [input]);

  // ── Drag-to-resize ───────────────────────────────────────────────────────

  const handleDragStart = useCallback((e: React.MouseEvent) => {
    isDraggingRef.current = true;
    dragStartYRef.current = e.clientY;
    dragStartHeightRef.current = panelHeight;
    e.preventDefault();
  }, [panelHeight]);

  useEffect(() => {
    function onMouseMove(e: MouseEvent) {
      if (!isDraggingRef.current) return;
      const delta = dragStartYRef.current - e.clientY;
      const newHeight = Math.min(
        Math.max(300, dragStartHeightRef.current + delta),
        window.innerHeight * 0.8,
      );
      setPanelHeight(newHeight);
    }
    function onMouseUp() {
      isDraggingRef.current = false;
    }
    window.addEventListener('mousemove', onMouseMove);
    window.addEventListener('mouseup', onMouseUp);
    return () => {
      window.removeEventListener('mousemove', onMouseMove);
      window.removeEventListener('mouseup', onMouseUp);
    };
  }, []);

  // ── Send message ────────────────────────────────────────────────────────

  const handleSend = useCallback(async () => {
    const text = input.trim();
    if (!text || isStreaming) return;
    setInput('');
    await sendMessage(text);
  }, [input, isStreaming, sendMessage]);

  function handleKeyDown(e: React.KeyboardEvent<HTMLTextAreaElement>) {
    if (e.key === 'Enter' && !e.shiftKey) {
      e.preventDefault();
      handleSend();
    }
  }

  function clearChat() {
    resetSession();
    setInput('');
  }

  // ── Position: shift left when FlowTest is open ──────────────────────────
  const rightClass = flowTestOpen ? 'right-[calc(24rem+1rem)]' : 'right-6';

  // ── Render ──────────────────────────────────────────────────────────────

  return (
    <div
      className={`fixed bottom-6 ${rightClass} z-60 w-96 flex flex-col bg-brand-dark-alt border border-brand-shade3/20 rounded-lg shadow-2xl overflow-hidden transition-[right] duration-200`}
      style={{ height: minimized ? 'auto' : `${panelHeight}px` }}
    >
      {/* Drag handle */}
      {!minimized && (
        <div
          className="h-1.5 w-full cursor-ns-resize flex items-center justify-center flex-shrink-0 hover:bg-brand-shade3/20 transition-colors group"
          onMouseDown={handleDragStart}
        >
          <div className="w-8 h-0.5 bg-brand-shade3/30 rounded-full group-hover:bg-brand-shade3/60 transition-colors" />
        </div>
      )}

      {/* Header */}
      <div className="px-4 py-3 border-b border-brand-shade3/15 flex items-center justify-between flex-shrink-0">
        <div className="flex items-center gap-2">
          {/* Sparkles AI icon */}
          <svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round" className="text-brand-accent">
            <path d="M12 2l2.4 7.4H22l-6.2 4.5 2.4 7.4L12 17l-6.2 4.3 2.4-7.4L2 9.4h7.6z" />
          </svg>
          <h3 className="text-sm font-semibold text-brand-light">AI Assistant</h3>
        </div>
        <div className="flex items-center gap-1">
          {messages.length > 0 && !minimized && (
            <button
              onClick={clearChat}
              className="p-1 text-brand-shade3 hover:text-brand-light transition-colors"
              title="Clear chat"
            >
              <svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round">
                <polyline points="1 4 1 10 7 10" />
                <path d="M3.51 15a9 9 0 102.13-9.36L1 10" />
              </svg>
            </button>
          )}
          {/* Minimize / maximize */}
          <button
            onClick={() => setMinimized((v) => !v)}
            className="p-1 text-brand-shade3 hover:text-brand-light transition-colors"
            title={minimized ? 'Expand' : 'Minimize'}
          >
            {minimized ? (
              <svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2">
                <polyline points="15 3 21 3 21 9" />
                <polyline points="9 21 3 21 3 15" />
                <line x1="21" y1="3" x2="14" y2="10" />
                <line x1="3" y1="21" x2="10" y2="14" />
              </svg>
            ) : (
              <svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2">
                <polyline points="4 14 10 14 10 20" />
                <polyline points="20 10 14 10 14 4" />
                <line x1="10" y1="14" x2="3" y2="21" />
                <line x1="21" y1="3" x2="14" y2="10" />
              </svg>
            )}
          </button>
          <button
            onClick={onClose}
            className="p-1 text-brand-shade3 hover:text-brand-light transition-colors"
          >
            <svg className="w-4 h-4" fill="none" viewBox="0 0 24 24" stroke="currentColor" strokeWidth={2}>
              <path strokeLinecap="round" strokeLinejoin="round" d="M6 18L18 6M6 6l12 12" />
            </svg>
          </button>
        </div>
      </div>

      {/* Collapsed body — show nothing below header */}
      {minimized && null}

      {/* Agent not found placeholder */}
      {!minimized && agentStatus === 'not_found' && (
        <div className="flex-1 flex items-center justify-center p-6">
          <div className="text-center">
            <svg width="32" height="32" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="1.5" className="text-brand-shade3/40 mx-auto mb-3">
              <circle cx="12" cy="12" r="10" />
              <line x1="12" y1="8" x2="12" y2="12" />
              <line x1="12" y1="16" x2="12.01" y2="16" />
            </svg>
            <p className="text-sm text-brand-shade3 mb-1">Agent not found</p>
            <p className="text-xs text-brand-shade3/60 leading-relaxed">
              Configure a <span className="font-mono text-brand-shade2">builder-assistant</span> agent to enable AI-powered configuration assistance.
            </p>
          </div>
        </div>
      )}

      {/* Agent exists but has no model */}
      {!minimized && agentStatus === 'no_model' && (
        <div className="flex-1 flex items-center justify-center p-6">
          <div className="text-center">
            <svg width="32" height="32" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="1.5" className="text-amber-400/40 mx-auto mb-3">
              <path d="M10.29 3.86L1.82 18a2 2 0 001.71 3h16.94a2 2 0 001.71-3L13.71 3.86a2 2 0 00-3.42 0z" />
              <line x1="12" y1="9" x2="12" y2="13" />
              <line x1="12" y1="17" x2="12.01" y2="17" />
            </svg>
            <p className="text-sm text-brand-shade3 mb-1">Model not configured</p>
            <p className="text-xs text-brand-shade3/60 leading-relaxed">
              <span className="font-mono text-brand-shade2">builder-assistant</span> needs a model assigned. Go to the <span className="text-brand-shade2">Agents</span> page to assign one.
            </p>
          </div>
        </div>
      )}

      {/* Loading state */}
      {!minimized && agentStatus === 'loading' && (
        <div className="flex-1 flex items-center justify-center">
          <span className="text-xs text-brand-shade3">Checking assistant agent...</span>
        </div>
      )}

      {/* Chat area (only when agent is ready) */}
      {!minimized && agentStatus === 'ready' && (
        <>
          {/* Messages */}
          <div className="flex-1 overflow-y-auto p-3 space-y-3 min-h-0">
            {messages.length === 0 && (
              <div className="text-center mt-8 px-4">
                <p className="text-sm text-brand-shade3 mb-2">AI Configuration Assistant</p>
                <p className="text-xs text-brand-shade3/60 leading-relaxed">
                  Ask me to create agents, configure tools, set up MCP servers, or modify your platform configuration.
                </p>
              </div>
            )}

            {messages.map((msg) => (
              <div key={msg.id} className={msg.role === 'user' ? 'flex justify-end' : ''}>
                {msg.role === 'user' ? (
                  <div className="max-w-[85%] px-3 py-2 bg-brand-accent/10 border border-brand-accent/20 rounded-lg text-sm text-brand-light">
                    {msg.content}
                  </div>
                ) : (
                  <div className="space-y-1.5">
                    {/* Tool calls */}
                    {msg.toolCalls && msg.toolCalls.length > 0 && (
                      <div className="space-y-1">
                        {msg.toolCalls.map((tc, i) => {
                          const key = `${msg.id}-${i}`;
                          const expanded = expandedToolCalls[key] ?? false;
                          return (
                            <div key={i} className="px-2 py-1.5 bg-brand-dark border border-brand-shade3/20 rounded text-[11px] font-mono">
                              <div className="text-blue-400 mb-0.5">
                                <svg width="10" height="10" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" className="inline mr-1">
                                  <circle cx="12" cy="12" r="3" />
                                  <path d="M19.4 15a1.65 1.65 0 00.33 1.82l.06.06a2 2 0 010 2.83 2 2 0 01-2.83 0l-.06-.06a1.65 1.65 0 00-1.82-.33 1.65 1.65 0 00-1 1.51V21a2 2 0 01-4 0v-.09A1.65 1.65 0 009 19.4a1.65 1.65 0 00-1.82.33l-.06.06a2 2 0 01-2.83-2.83l.06-.06A1.65 1.65 0 004.68 15a1.65 1.65 0 00-1.51-1H3a2 2 0 010-4h.09A1.65 1.65 0 004.6 9a1.65 1.65 0 00-.33-1.82l-.06-.06a2 2 0 012.83-2.83l.06.06A1.65 1.65 0 009 4.68a1.65 1.65 0 001-1.51V3a2 2 0 014 0v.09a1.65 1.65 0 001 1.51 1.65 1.65 0 001.82-.33l.06-.06a2 2 0 012.83 2.83l-.06.06A1.65 1.65 0 0019.4 9a1.65 1.65 0 001.51 1H21a2 2 0 010 4h-.09a1.65 1.65 0 00-1.51 1z" />
                                </svg>
                                {tc.tool}
                              </div>
                              {tc.input && (
                                <button
                                  onClick={() => toggleToolCall(key)}
                                  className="text-brand-shade3 text-left w-full hover:text-brand-shade2 transition-colors"
                                >
                                  {expanded
                                    ? tc.input
                                    : `${tc.input.slice(0, 80)}${tc.input.length > 80 ? '...' : ''}`}
                                  {tc.input.length > 80 && (
                                    <span className="ml-1 text-brand-accent/70 text-[10px]">
                                      {expanded ? '[less]' : '[more]'}
                                    </span>
                                  )}
                                </button>
                              )}
                              {tc.output !== undefined && (
                                <button
                                  onClick={() => toggleToolCall(`${key}-out`)}
                                  className="text-emerald-400 mt-0.5 text-left w-full hover:text-emerald-300 transition-colors"
                                >
                                  {expandedToolCalls[`${key}-out`]
                                    ? tc.output
                                    : `${tc.output.slice(0, 80)}${tc.output.length > 80 ? '...' : ''}`}
                                  {tc.output.length > 80 && (
                                    <span className="ml-1 text-brand-accent/70 text-[10px]">
                                      {expandedToolCalls[`${key}-out`] ? '[less]' : '[more]'}
                                    </span>
                                  )}
                                </button>
                              )}
                            </div>
                          );
                        })}
                      </div>
                    )}

                    {/* Content */}
                    {(msg.content || msg.streaming) && (
                      <div className="text-sm text-brand-light leading-relaxed whitespace-pre-wrap">
                        {msg.content}
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
          <div className="border-t border-brand-shade3/15 p-3 flex-shrink-0">
            <div className="flex gap-2 items-end">
              <textarea
                ref={textareaRef}
                value={input}
                onChange={(e) => setInput(e.target.value)}
                onKeyDown={handleKeyDown}
                placeholder="Ask the assistant..."
                rows={1}
                disabled={isStreaming}
                className="flex-1 px-3 py-2 bg-brand-dark border border-brand-shade3/30 rounded-lg text-sm text-brand-light placeholder-brand-shade3 focus:outline-none focus:border-brand-accent focus:ring-1 focus:ring-brand-accent resize-none disabled:opacity-50 transition-colors"
                style={{ minHeight: '36px', overflowY: 'hidden' }}
              />
              {isStreaming ? (
                <button
                  onClick={stopStreaming}
                  className="px-3 py-2 bg-brand-dark border border-brand-shade3/30 rounded-lg text-sm text-brand-shade2 hover:text-brand-light hover:border-brand-shade3 transition-colors flex-shrink-0"
                  title="Stop"
                >
                  <svg width="14" height="14" viewBox="0 0 24 24" fill="currentColor">
                    <rect x="4" y="4" width="16" height="16" rx="2" />
                  </svg>
                </button>
              ) : (
                <button
                  onClick={handleSend}
                  disabled={!input.trim()}
                  className="px-3 py-2 bg-brand-accent text-brand-light rounded-lg text-sm hover:bg-brand-accent-hover disabled:opacity-40 transition-colors flex-shrink-0"
                >
                  <svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round">
                    <line x1="22" y1="2" x2="11" y2="13" />
                    <polygon points="22 2 15 22 11 13 2 9 22 2" />
                  </svg>
                </button>
              )}
            </div>
            <p className="text-[10px] text-brand-shade3/50 mt-1">Enter to send -- Shift+Enter for newline</p>
          </div>
        </>
      )}
    </div>
  );
}
