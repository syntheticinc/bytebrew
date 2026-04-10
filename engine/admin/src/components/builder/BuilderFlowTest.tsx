import { useState, useRef, useEffect, useCallback } from 'react';
import { useSSEChat } from '../../hooks/useSSEChat';

// ─── Types ──────────────────────────────────────────────────────────────────

interface BuilderFlowTestProps {
  agents: Array<{ name: string; model_id: string }>;
  onClose: () => void;
}

interface Header {
  id: string;
  key: string;
  value: string;
}

// ─── Component ──────────────────────────────────────────────────────────────

export default function BuilderFlowTest({ agents, onClose }: BuilderFlowTestProps) {
  const [selectedAgent, setSelectedAgent] = useState(agents[0]?.name ?? '');
  const [headers, setHeaders] = useState<Header[]>([]);
  const [headersExpanded, setHeadersExpanded] = useState(false);
  const [message, setMessage] = useState('');

  // Tool call expand state
  const [expandedToolCalls, setExpandedToolCalls] = useState<Record<string, boolean>>({});

  const messagesEndRef = useRef<HTMLDivElement>(null);

  // Build custom headers getter for SSE hook
  const headersRef = useRef(headers);
  useEffect(() => { headersRef.current = headers; }, [headers]);

  const getHeaders = useCallback((): Record<string, string> => {
    const result: Record<string, string> = {};
    const blocked = ['authorization', 'host', 'cookie', 'origin', 'referer', 'content-type', 'content-length'];
    for (const h of headersRef.current) {
      const k = h.key.trim();
      const v = h.value.trim();
      if (k && v && !blocked.includes(k.toLowerCase())) result[k] = v;
    }
    return result;
  }, []);

  const { messages, sendMessage, isStreaming, resetSession, stopStreaming } = useSSEChat({
    endpoint: `/api/v1/agents/${encodeURIComponent(selectedAgent)}/chat`,
    agentName: selectedAgent,
    getHeaders,
  });

  // Auto-scroll on new content
  useEffect(() => {
    messagesEndRef.current?.scrollIntoView({ behavior: 'smooth' });
  }, [messages]);

  function toggleToolCall(key: string) {
    setExpandedToolCalls((prev) => ({ ...prev, [key]: !prev[key] }));
  }

  // ── Headers management ──────────────────────────────────────────────────

  function addHeader() {
    setHeaders((prev) => [...prev, { id: crypto.randomUUID(), key: '', value: '' }]);
    if (!headersExpanded) setHeadersExpanded(true);
  }

  function updateHeader(id: string, field: 'key' | 'value', val: string) {
    setHeaders((prev) => prev.map((h) => (h.id === id ? { ...h, [field]: val } : h)));
  }

  function removeHeader(id: string) {
    setHeaders((prev) => prev.filter((h) => h.id !== id));
  }

  // ── Run test ────────────────────────────────────────────────────────────

  const runTest = useCallback(async () => {
    const text = message.trim();
    if (!text || !selectedAgent || isStreaming) return;
    setMessage('');
    await sendMessage(text);
  }, [message, selectedAgent, isStreaming, sendMessage]);

  function handleKeyDown(e: React.KeyboardEvent<HTMLTextAreaElement>) {
    if (e.key === 'Enter' && !e.shiftKey) {
      e.preventDefault();
      runTest();
    }
  }

  // ── Render ──────────────────────────────────────────────────────────────

  const lastMsg = messages.length > 0 ? messages[messages.length - 1] : null;
  const hasError = lastMsg?.role === 'assistant' && lastMsg.content?.startsWith('Error:');

  return (
    <div className="w-96 border-l border-brand-shade3/15 bg-brand-dark-alt flex flex-col h-full flex-shrink-0">
      {/* Header */}
      <div className="px-4 py-3 border-b border-brand-shade3/15 flex items-center justify-between flex-shrink-0">
        <div className="flex items-center gap-2">
          <svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round" className="text-brand-accent">
            <polygon points="5 3 19 12 5 21 5 3" />
          </svg>
          <h3 className="text-sm font-semibold text-brand-light">Flow Test</h3>
        </div>
        <div className="flex items-center gap-1">
          {messages.length > 0 && (
            <button
              onClick={resetSession}
              className="p-1 text-brand-shade3 hover:text-brand-light transition-colors"
              title="New Session"
            >
              <svg width="13" height="13" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round">
                <polyline points="1 4 1 10 7 10" />
                <path d="M3.51 15a9 9 0 102.13-9.36L1 10" />
              </svg>
            </button>
          )}
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

      {/* Config */}
      <div className="p-4 space-y-3 border-b border-brand-shade3/10 flex-shrink-0">
        {/* Agent selector */}
        <div>
          <label className="block text-xs font-medium text-brand-shade3 mb-1 uppercase tracking-wide">Orchestrator Agent</label>
          <select
            value={selectedAgent}
            onChange={(e) => { setSelectedAgent(e.target.value); resetSession(); }}
            className="w-full px-2.5 py-1.5 bg-brand-dark border border-brand-shade3/30 rounded-card text-sm text-brand-light focus:outline-none focus:border-brand-accent transition-colors"
          >
            {agents.length === 0 && (
              <option value="">No agents available</option>
            )}
            {agents.map((a) => (
              <option key={a.name} value={a.name}>{a.name}</option>
            ))}
          </select>
        </div>

        {/* Headers */}
        <div>
          <button
            onClick={() => setHeadersExpanded(!headersExpanded)}
            className="flex items-center gap-1.5 text-xs font-medium text-brand-shade3 uppercase tracking-wide hover:text-brand-light transition-colors w-full"
          >
            <svg
              width="10" height="10" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2"
              className={`transition-transform ${headersExpanded ? 'rotate-90' : ''}`}
            >
              <polyline points="9 18 15 12 9 6" />
            </svg>
            Headers
            {headers.length > 0 && (
              <span className="text-[10px] text-brand-shade3/60 font-normal normal-case">({headers.length})</span>
            )}
          </button>

          {headersExpanded && (
            <div className="mt-2 space-y-2">
              {headers.map((h) => (
                <div key={h.id} className="flex gap-1.5 items-start">
                  <input
                    type="text"
                    value={h.key}
                    onChange={(e) => updateHeader(h.id, 'key', e.target.value)}
                    placeholder="Header"
                    className="flex-1 px-2 py-1 bg-brand-dark border border-brand-shade3/30 rounded text-xs text-brand-light placeholder-brand-shade3/50 focus:outline-none focus:border-brand-accent transition-colors"
                  />
                  <input
                    type="text"
                    value={h.value}
                    onChange={(e) => updateHeader(h.id, 'value', e.target.value)}
                    placeholder="Value"
                    className="flex-1 px-2 py-1 bg-brand-dark border border-brand-shade3/30 rounded text-xs text-brand-light placeholder-brand-shade3/50 focus:outline-none focus:border-brand-accent transition-colors"
                  />
                  <button
                    onClick={() => removeHeader(h.id)}
                    className="p-1 text-brand-shade3 hover:text-red-400 transition-colors flex-shrink-0"
                  >
                    <svg width="12" height="12" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2">
                      <line x1="18" y1="6" x2="6" y2="18" />
                      <line x1="6" y1="6" x2="18" y2="18" />
                    </svg>
                  </button>
                </div>
              ))}
              <button
                onClick={addHeader}
                className="text-[11px] text-brand-shade3 hover:text-brand-light transition-colors"
              >
                + Add Header
              </button>
            </div>
          )}
        </div>
      </div>

      {/* Conversation history */}
      <div className="flex-1 overflow-y-auto min-h-0 p-3 space-y-3">
        {messages.length === 0 && (
          <p className="text-xs text-brand-shade3/50 text-center mt-4">
            Send a message to start the flow test.
          </p>
        )}

        {messages.map((msg) => (
          <div key={msg.id} className={msg.role === 'user' ? 'flex justify-end' : ''}>
            {msg.role === 'user' ? (
              <div className="max-w-[85%] px-3 py-2 bg-brand-accent/10 border border-brand-accent/20 rounded-lg text-sm text-brand-light font-mono">
                {msg.content}
              </div>
            ) : (
              <div className="space-y-1.5">
                {/* Error */}
                {hasError && msg.id === lastMsg?.id && (
                  <div className="px-2 py-1.5 bg-red-900/20 border border-red-500/20 rounded text-xs text-red-400">
                    {msg.content.replace(/^Error:\s*/, '')}
                  </div>
                )}

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
                                : `${tc.input.slice(0, 120)}${tc.input.length > 120 ? '...' : ''}`}
                              {tc.input.length > 120 && (
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
                                : `${tc.output.slice(0, 120)}${tc.output.length > 120 ? '...' : ''}`}
                              {tc.output.length > 120 && (
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

                {/* Content (skip if error — already rendered above) */}
                {msg.content && !msg.content.startsWith('Error:') && (
                  <div className="text-sm text-brand-light leading-relaxed whitespace-pre-wrap">
                    {msg.content}
                    {msg.streaming && (
                      <span className="inline-block w-1.5 h-3.5 bg-brand-accent ml-0.5 animate-pulse" />
                    )}
                  </div>
                )}

                {/* Waiting indicator */}
                {msg.streaming && !msg.content && (!msg.toolCalls || msg.toolCalls.length === 0) && (
                  <span className="text-brand-shade3/50 text-xs">Waiting for response...</span>
                )}
              </div>
            )}
          </div>
        ))}

        {isStreaming && (
          <div className="flex justify-end">
            <span className="text-[10px] text-brand-accent animate-pulse">streaming...</span>
          </div>
        )}

        <div ref={messagesEndRef} />
      </div>

      {/* Input area */}
      <div className="border-t border-brand-shade3/15 p-3 flex-shrink-0 space-y-2">
        <div className="flex gap-2 items-end">
          <textarea
            value={message}
            onChange={(e) => setMessage(e.target.value)}
            onKeyDown={handleKeyDown}
            placeholder="Type a test message..."
            rows={2}
            className="flex-1 px-2.5 py-1.5 bg-brand-dark border border-brand-shade3/30 rounded-card text-sm text-brand-light placeholder-brand-shade3 font-mono focus:outline-none focus:border-brand-accent focus:ring-1 focus:ring-brand-accent resize-none transition-colors"
          />
          {isStreaming ? (
            <button
              onClick={stopStreaming}
              className="px-3 py-2 bg-brand-dark border border-brand-shade3/30 rounded-btn text-sm text-brand-shade2 hover:text-brand-light hover:border-brand-shade3 transition-colors flex-shrink-0 inline-flex items-center gap-1"
            >
              <svg width="12" height="12" viewBox="0 0 24 24" fill="currentColor">
                <rect x="4" y="4" width="16" height="16" rx="2" />
              </svg>
              Stop
            </button>
          ) : (
            <button
              onClick={runTest}
              disabled={!message.trim() || !selectedAgent}
              className="px-3 py-2 bg-brand-accent text-brand-light rounded-btn text-sm font-medium hover:bg-brand-accent-hover disabled:opacity-40 transition-colors flex-shrink-0 inline-flex items-center gap-1"
            >
              <svg width="12" height="12" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round">
                <polygon points="5 3 19 12 5 21 5 3" />
              </svg>
              Run
            </button>
          )}
        </div>
        <p className="text-[10px] text-brand-shade3/50">Enter to send -- Shift+Enter for newline</p>
      </div>
    </div>
  );
}
