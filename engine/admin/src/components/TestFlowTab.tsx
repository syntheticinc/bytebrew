import { useState, useRef, useEffect, useCallback } from 'react';
import { useNavigate } from 'react-router-dom';
import { useSSEChat, type SSEMessage } from '../hooks/useSSEChat';
import { useBottomPanel } from '../hooks/useBottomPanel';
import { usePrototype } from '../hooks/usePrototype';
import { api } from '../api/client';
import type { AgentDetail, SessionSummary } from '../types';
import HeadersEditor, { type HeaderEntry } from './HeadersEditor';
import ContextUsageBar from './ContextUsageBar';

// ─── Mock streaming for prototype mode ──────────────────────────────────────

const MOCK_TOOL_CALLS = [
  { tool: 'memory_recall', input: '{"query": "previous interactions"}', output: '{"memories": []}' },
  { tool: 'search_knowledge', input: '{"query": "product FAQ"}', output: '{"results": [{"title": "FAQ", "content": "..."}]}' },
];

const MOCK_RESPONSES = [
  'Based on the knowledge base, here is the answer to your question. The system supports multiple agent configurations with memory, knowledge, and escalation capabilities.',
  'I have processed your request. The agent flow executed successfully through the classifier and support pipeline.',
  'Your test message has been routed through the schema flow. All tools executed correctly and the response was generated.',
];

// ─── Friendly error mapping ─────────────────────────────────────────────────

type FriendlyResult = { text: string; agentLink?: string };

function friendlyError(raw: string, agentName?: string): FriendlyResult {
  if (raw.includes('resolve tool') && raw.includes('unknown builtin tool')) {
    const toolMatch = raw.match(/resolve tool (\S+):/);
    const toolName = toolMatch?.[1] ?? 'unknown';
    return { text: `Agent references tool "${toolName}" which is not available. Check agent configuration \u2192 Tools to fix this.`, agentLink: agentName };
  }
  if (raw.includes('model not found') || raw.includes('no model configured') || raw.includes('no model available')) {
    return { text: 'No model configured for this agent. Assign a model in agent settings.', agentLink: agentName };
  }
  if (raw.includes('connection refused') || raw.includes('ECONNREFUSED')) {
    return { text: 'Cannot connect to the model provider. Check model configuration and API keys.', agentLink: agentName };
  }
  return { text: raw };
}

// ─── Component ──────────────────────────────────────────────────────────────

export default function TestFlowTab() {
  const navigate = useNavigate();
  const { selectedSchema } = useBottomPanel();
  const { isPrototype } = usePrototype();

  const [allAgents, setAllAgents] = useState<string[]>([]);
  const [schemaAgents, setSchemaAgents] = useState<string[]>([]);
  const [selectedAgent, setSelectedAgent] = useState('');
  const [headers, setHeaders] = useState<HeaderEntry[]>([]);
  const [message, setMessage] = useState('');
  const [expandedItems, setExpandedItems] = useState<Record<string, boolean>>({});

  // Prototype mode state
  const [protoMessages, setProtoMessages] = useState<SSEMessage[]>([]);
  const [protoStreaming, setProtoStreaming] = useState(false);
  const [protoSessionId, setProtoSessionId] = useState('');

  // Session management state (production only)
  const [sessions, setSessions] = useState<SessionSummary[]>([]);
  const [agentDetail, setAgentDetail] = useState<AgentDetail | null>(null);
  const [sessionDropdownOpen, setSessionDropdownOpen] = useState(false);
  const sessionDropdownRef = useRef<HTMLDivElement>(null);

  const messagesEndRef = useRef<HTMLDivElement>(null);
  const headersRef = useRef(headers);
  const inputRef = useRef<HTMLTextAreaElement>(null);
  useEffect(() => { headersRef.current = headers; }, [headers]);

  // Build headers getter for SSE hook
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

  const testflowPersistenceKey = selectedAgent && selectedSchema
    ? `bb_testflow_${selectedSchema}_${selectedAgent}`
    : undefined;
  const sseChat = useSSEChat({
    endpoint: selectedAgent ? `/api/v1/agents/${encodeURIComponent(selectedAgent)}/chat` : '',
    agentName: selectedAgent,
    getHeaders,
    persistenceKey: testflowPersistenceKey,
    fetchMessages: (sid) => api.getSessionEvents(sid),
  });

  // Use either prototype or production messages
  const messages = isPrototype ? protoMessages : sseChat.messages;
  const isStreaming = isPrototype ? protoStreaming : sseChat.isStreaming;
  const sessionId = isPrototype ? protoSessionId : sseChat.sessionId;

  // Load agents (schema-scoped when a schema is selected)
  useEffect(() => {
    let cancelled = false;

    async function load() {
      try {
        const list = await api.listAgents();
        const names = list.map((a) => a.name);
        if (cancelled) return;
        setAllAgents(names);

        // If a schema is selected, fetch its agents to scope the dropdown
        let schemaNames: string[] = [];
        if (selectedSchema) {
          const schemas = await api.listSchemas();
          const match = schemas.find((s) => s.name === selectedSchema);
          if (match && !cancelled) {
            schemaNames = await api.listSchemaAgents(match.id);
          }
        }
        if (cancelled) return;
        setSchemaAgents(schemaNames);

        // Auto-select first schema agent, or first agent overall
        const preferred = schemaNames.length > 0 ? schemaNames : names;
        if (preferred.length > 0 && !selectedAgent) {
          setSelectedAgent(preferred[0]!);
        }
      } catch {
        // ignore
      }
    }

    load();
    return () => { cancelled = true; };
  }, [selectedSchema, selectedAgent]);

  // Fetch agent detail for context bar
  useEffect(() => {
    if (!selectedAgent) { setAgentDetail(null); return; }
    let cancelled = false;
    api.getAgent(selectedAgent)
      .then((d) => { if (!cancelled) setAgentDetail(d); })
      .catch(() => {});
    return () => { cancelled = true; };
  }, [selectedAgent]);

  // Fetch sessions for selected agent (production only)
  useEffect(() => {
    if (!selectedAgent || isPrototype) { setSessions([]); return; }
    let cancelled = false;
    api.listSessions({ agent_name: selectedAgent, per_page: 20 })
      .then((res) => { if (!cancelled) setSessions(res.sessions); })
      .catch(() => {});
    return () => { cancelled = true; };
  }, [selectedAgent, isPrototype]);

  // Refresh session list when a new session is created (sessionId changes)
  useEffect(() => {
    if (!selectedAgent || isPrototype || !sseChat.sessionId) return;
    if (sessions?.some((s) => s.session_id === sseChat.sessionId)) return;
    api.listSessions({ agent_name: selectedAgent, per_page: 20 })
      .then((res) => setSessions(res.sessions))
      .catch(() => {});
  // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [sseChat.sessionId]);

  // Close dropdown on outside click
  useEffect(() => {
    if (!sessionDropdownOpen) return;
    function handleClick(e: MouseEvent) {
      if (sessionDropdownRef.current && !sessionDropdownRef.current.contains(e.target as Node)) {
        setSessionDropdownOpen(false);
      }
    }
    document.addEventListener('mousedown', handleClick);
    return () => document.removeEventListener('mousedown', handleClick);
  }, [sessionDropdownOpen]);

  // Auto-scroll
  useEffect(() => {
    messagesEndRef.current?.scrollIntoView({ behavior: 'smooth' });
  }, [messages]);

  function toggleItem(key: string) {
    setExpandedItems((prev) => ({ ...prev, [key]: !prev[key] }));
  }

  // ── Prototype mock send ──────────────────────────────────────────────────

  function protoSend(text: string) {
    if (!text.trim() || protoStreaming) return;

    const userMsg: SSEMessage = { id: crypto.randomUUID(), role: 'user', content: text };
    const assistantId = crypto.randomUUID();
    const sid = protoSessionId || `session-${Date.now()}`;
    if (!protoSessionId) setProtoSessionId(sid);

    setProtoMessages((prev) => [
      ...prev,
      userMsg,
      { id: assistantId, role: 'assistant', content: '', toolCalls: [], streaming: true },
    ]);
    setProtoStreaming(true);

    // Simulate streaming with tool calls
    const toolCalls = MOCK_TOOL_CALLS.map((tc) => ({ ...tc }));
    const responseText = MOCK_RESPONSES[Math.floor(Math.random() * MOCK_RESPONSES.length)]!;

    // Step 1: show tool calls after 500ms
    setTimeout(() => {
      setProtoMessages((prev) =>
        prev.map((m) => m.id === assistantId ? { ...m, toolCalls } : m),
      );
    }, 500);

    // Step 2: stream response text
    let charIndex = 0;
    const interval = setInterval(() => {
      charIndex += 3;
      if (charIndex >= responseText.length) {
        clearInterval(interval);
        setProtoMessages((prev) =>
          prev.map((m) => m.id === assistantId ? { ...m, content: responseText, streaming: false } : m),
        );
        setProtoStreaming(false);
        return;
      }
      setProtoMessages((prev) =>
        prev.map((m) => m.id === assistantId ? { ...m, content: responseText.slice(0, charIndex) } : m),
      );
    }, 30);
  }

  // ── Send message ─────────────────────────────────────────────────────────

  async function handleSend() {
    const text = message.trim();
    if (!text || !selectedAgent || isStreaming) return;
    setMessage('');
    if (inputRef.current) inputRef.current.style.height = 'auto';

    if (isPrototype) {
      protoSend(text);
      return;
    }

    await sseChat.sendMessage(text);
  }

  function handleKeyDown(e: React.KeyboardEvent<HTMLTextAreaElement>) {
    if (e.key === 'Enter' && !e.shiftKey) {
      e.preventDefault();
      handleSend();
    }
  }

  function handleReset() {
    if (isPrototype) {
      setProtoMessages([]);
      setProtoSessionId('');
    } else {
      sseChat.resetSession();
    }
  }

  function handleStop() {
    if (isPrototype) {
      setProtoStreaming(false);
      setProtoMessages((prev) => prev.map((m) => m.streaming ? { ...m, streaming: false } : m));
    } else {
      sseChat.stopStreaming();
    }
  }

  // ── Session management (production only) ─────────────────────────────────

  async function handleSwitchSession(sid: string) {
    setSessionDropdownOpen(false);
    await sseChat.loadSession(sid);
  }

  async function handleDeleteSession(sid: string, e: React.MouseEvent) {
    e.stopPropagation();
    if (!confirm('Delete this session?')) return;
    try {
      await api.deleteSession(sid);
      setSessions((prev) => prev.filter((s) => s.session_id !== sid));
      if (sseChat.sessionId === sid) {
        sseChat.resetSession();
      }
    } catch {
      // ignore
    }
  }

  // ── Render ────────────────────────────────────────────────────────────────

  const lastMsg = messages.length > 0 ? messages[messages.length - 1] : null;
  const hasError = lastMsg?.role === 'assistant' && lastMsg.content?.startsWith('Error:');
  const showInspectLink = lastMsg?.role === 'assistant' && !lastMsg.streaming && !hasError && messages.length > 0;

  return (
    <div className="flex flex-col h-full">
      {/* Config section */}
      <div className="px-3 py-2 space-y-2 border-b border-brand-shade3/10 flex-shrink-0">
        {/* Agent selector */}
        <div className="flex items-center gap-2">
          <label className="text-[10px] text-brand-shade3 uppercase tracking-wide shrink-0">Agent:</label>
          <select
            value={selectedAgent}
            onChange={(e) => { setSelectedAgent(e.target.value); handleReset(); }}
            className="flex-1 px-2 py-1 bg-brand-dark border border-brand-shade3/30 rounded text-xs text-brand-light focus:outline-none focus:border-brand-accent transition-colors"
          >
            {allAgents.length === 0 && <option value="">No agents</option>}
            {selectedSchema && schemaAgents.length > 0 ? (
              <>
                <optgroup label={`Schema: ${selectedSchema}`}>
                  {schemaAgents.map((a) => (
                    <option key={a} value={a}>{a}</option>
                  ))}
                </optgroup>
                {allAgents.filter((a) => !schemaAgents.includes(a)).length > 0 && (
                  <optgroup label="Other agents">
                    {allAgents.filter((a) => !schemaAgents.includes(a)).map((a) => (
                      <option key={a} value={a}>{a}</option>
                    ))}
                  </optgroup>
                )}
              </>
            ) : (
              allAgents.map((a) => (
                <option key={a} value={a}>{a}</option>
              ))
            )}
          </select>
          {selectedSchema && (
            <span className="text-[10px] text-brand-shade3/60 shrink-0">Schema: {selectedSchema}</span>
          )}
          {messages.length > 0 && (
            <button
              onClick={handleReset}
              className="p-1 text-brand-shade3 hover:text-brand-light transition-colors shrink-0"
              title="New Session"
            >
              <svg width="12" height="12" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round">
                <polyline points="1 4 1 10 7 10" />
                <path d="M3.51 15a9 9 0 102.13-9.36L1 10" />
              </svg>
            </button>
          )}
        </div>

        {/* Session selector (production only) */}
        {!isPrototype && selectedAgent && (
          <div className="flex items-center gap-2">
            <label className="text-[10px] text-brand-shade3 uppercase tracking-wide shrink-0">Session:</label>
            <div ref={sessionDropdownRef} className="relative flex-1">
              <button
                onClick={() => setSessionDropdownOpen((p) => !p)}
                className="w-full px-2 py-1 bg-brand-dark border border-brand-shade3/30 rounded text-xs text-brand-light text-left flex items-center gap-1 hover:border-brand-shade3/50 transition-colors"
              >
                <span className="truncate flex-1">
                  {sseChat.sessionId ? sseChat.sessionId.slice(0, 12) + '...' : 'New Session'}
                </span>
                <svg width="8" height="8" viewBox="0 0 24 24" fill="currentColor" className={`text-brand-shade3 transition-transform ${sessionDropdownOpen ? 'rotate-180' : ''}`}>
                  <path d="M7 10l5 5 5-5H7z" />
                </svg>
              </button>
              {sessionDropdownOpen && (
                <div className="absolute top-full left-0 mt-1 w-full max-h-48 overflow-y-auto bg-brand-dark border border-brand-shade3/20 rounded shadow-lg z-50">
                  <button
                    onClick={() => { handleReset(); setSessionDropdownOpen(false); }}
                    className="w-full px-2 py-1.5 text-left text-xs text-brand-accent hover:bg-brand-shade3/10 flex items-center gap-1.5 transition-colors"
                  >
                    <svg width="10" height="10" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round">
                      <line x1="12" y1="5" x2="12" y2="19" /><line x1="5" y1="12" x2="19" y2="12" />
                    </svg>
                    New Session
                  </button>
                  {sessions.map((s) => (
                    <div
                      key={s.session_id}
                      onClick={() => handleSwitchSession(s.session_id)}
                      className={`w-full px-2 py-1.5 text-left text-xs flex items-center gap-1.5 hover:bg-brand-shade3/10 cursor-pointer transition-colors ${sseChat.sessionId === s.session_id ? 'text-brand-accent' : 'text-brand-light'}`}
                    >
                      <span className="truncate flex-1">
                        {s.session_id.slice(0, 10)}...
                        <span className="text-brand-shade3 ml-1">
                          {new Date(s.created_at).toLocaleString(undefined, { month: 'short', day: 'numeric', hour: '2-digit', minute: '2-digit' })}
                        </span>
                      </span>
                      <button
                        onClick={(e) => handleDeleteSession(s.session_id, e)}
                        className="p-0.5 text-brand-shade3 hover:text-red-400 transition-colors shrink-0"
                        title="Delete session"
                      >
                        <svg width="10" height="10" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round">
                          <polyline points="3 6 5 6 21 6" /><path d="M19 6l-1 14a2 2 0 01-2 2H8a2 2 0 01-2-2L5 6" /><path d="M10 11v6" /><path d="M14 11v6" />
                        </svg>
                      </button>
                    </div>
                  ))}
                  {sessions.length === 0 && (
                    <div className="px-2 py-1.5 text-[10px] text-brand-shade3">No sessions yet</div>
                  )}
                </div>
              )}
            </div>
          </div>
        )}

        {/* Headers editor */}
        <HeadersEditor headers={headers} onChange={setHeaders} />
      </div>

      {/* Messages area */}
      <div className="flex-1 overflow-y-auto min-h-0 px-3 py-2 space-y-2">
        {!isPrototype && sseChat.isRestoring && messages.length === 0 ? (
          <div className="flex items-center gap-2 text-[11px] text-brand-shade3 font-mono py-4 justify-center">
            <svg width="12" height="12" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" className="animate-spin">
              <path d="M21 12a9 9 0 11-6.219-8.56" />
            </svg>
            Restoring session...
          </div>
        ) : messages.length === 0 ? (
          <p className="text-[11px] text-brand-shade3/50 text-center mt-4">
            Send a message to test the agent flow.
          </p>
        ) : null}

        {messages.map((msg) => (
          <div key={msg.id} className={msg.role === 'user' ? 'flex justify-end' : ''}>
            {msg.role === 'user' ? (
              <div className="max-w-[85%] px-2.5 py-1.5 bg-brand-accent/10 border border-brand-accent/20 rounded-lg text-xs text-brand-light font-mono">
                {msg.content}
              </div>
            ) : (
              <div className="space-y-1">
                {/* Error */}
                {hasError && msg.id === lastMsg?.id && (() => {
                  const err = friendlyError(msg.content.replace(/^Error:\s*/, ''), selectedAgent);
                  return (
                    <div className="px-2 py-1.5 bg-red-900/20 border border-red-500/20 rounded text-[11px] text-red-400">
                      {err.text}
                      {err.agentLink && (
                        <button
                          onClick={() => navigate(`/agents/${encodeURIComponent(err.agentLink!)}`)}
                          className="ml-1.5 text-brand-accent hover:text-brand-accent-hover underline transition-colors"
                        >
                          Configure agent &rarr;
                        </button>
                      )}
                    </div>
                  );
                })()}

                {/* Reasoning (if present as a special content pattern) */}
                {msg.content?.includes('[thinking]') && (() => {
                  const thinkKey = `${msg.id}-think`;
                  const isExpanded = expandedItems[thinkKey] ?? false;
                  const thinkMatch = msg.content.match(/\[thinking\]([\s\S]*?)\[\/thinking\]/);
                  if (!thinkMatch) return null;
                  return (
                    <button
                      onClick={() => toggleItem(thinkKey)}
                      className="w-full text-left px-2 py-1 bg-amber-900/10 border border-amber-500/15 rounded text-[11px] font-mono hover:border-amber-500/30 transition-colors"
                    >
                      <div className="flex items-center gap-1.5 text-amber-400">
                        <span>Thinking...</span>
                        <svg
                          width="8" height="8" viewBox="0 0 24 24" fill="currentColor"
                          className={`transition-transform ml-auto ${isExpanded ? 'rotate-90' : ''}`}
                        >
                          <path d="M8 5l10 7-10 7V5z" />
                        </svg>
                      </div>
                      {isExpanded && (
                        <div className="mt-1 text-[10px] text-amber-400/70 whitespace-pre-wrap">
                          {thinkMatch[1]}
                        </div>
                      )}
                    </button>
                  );
                })()}

                {/* Content */}
                {msg.content && !msg.content.startsWith('Error:') && (
                  <div className="text-xs text-brand-light leading-relaxed whitespace-pre-wrap">
                    {msg.content.replace(/\[thinking\][\s\S]*?\[\/thinking\]/g, '').trim()}
                    {msg.streaming && (
                      <span className="inline-block w-1.5 h-3 bg-brand-accent ml-0.5 animate-pulse" />
                    )}
                  </div>
                )}

                {/* Tool calls rendered AFTER text */}
                {msg.toolCalls && msg.toolCalls.length > 0 && (
                  <div className="space-y-1">
                    {msg.toolCalls.map((tc, i) => {
                      const key = `${msg.id}-tc-${i}`;
                      const isExpanded = expandedItems[key] ?? false;
                      return (
                        <button
                          key={i}
                          onClick={() => toggleItem(key)}
                          className="w-full text-left px-2 py-1 bg-brand-dark border border-brand-shade3/15 rounded text-[11px] font-mono hover:border-brand-shade3/30 transition-colors"
                        >
                          <div className="flex items-center gap-1.5">
                            <span className="text-blue-400">
                              <svg width="10" height="10" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" className="inline">
                                <circle cx="12" cy="12" r="3" />
                                <path d="M19.4 15a1.65 1.65 0 00.33 1.82l.06.06a2 2 0 010 2.83 2 2 0 01-2.83 0l-.06-.06a1.65 1.65 0 00-1.82-.33 1.65 1.65 0 00-1 1.51V21a2 2 0 01-4 0v-.09A1.65 1.65 0 009 19.4a1.65 1.65 0 00-1.82.33l-.06.06a2 2 0 01-2.83-2.83l.06-.06A1.65 1.65 0 004.68 15a1.65 1.65 0 00-1.51-1H3a2 2 0 010-4h.09A1.65 1.65 0 004.6 9a1.65 1.65 0 00-.33-1.82l-.06-.06a2 2 0 012.83-2.83l.06.06A1.65 1.65 0 009 4.68a1.65 1.65 0 001-1.51V3a2 2 0 014 0v.09a1.65 1.65 0 001 1.51 1.65 1.65 0 001.82-.33l.06-.06a2 2 0 012.83 2.83l-.06.06A1.65 1.65 0 0019.4 9a1.65 1.65 0 001.51 1H21a2 2 0 010 4h-.09a1.65 1.65 0 00-1.51 1z" />
                              </svg>
                            </span>
                            <span className="text-blue-400 font-medium">{tc.tool}</span>
                            {tc.output !== undefined && (
                              <span className="text-emerald-400/60 ml-1">done</span>
                            )}
                            <svg
                              width="8" height="8" viewBox="0 0 24 24" fill="currentColor"
                              className={`text-brand-shade3 transition-transform ml-auto ${isExpanded ? 'rotate-90' : ''}`}
                            >
                              <path d="M8 5l10 7-10 7V5z" />
                            </svg>
                          </div>
                          {isExpanded && (
                            <div className="mt-1 space-y-1 text-[10px]">
                              {tc.input && (
                                <div className="text-brand-shade3 whitespace-pre-wrap break-all">
                                  <span className="text-brand-shade3/60">Input: </span>{tc.input}
                                </div>
                              )}
                              {tc.output !== undefined && (
                                <div className="text-emerald-400/80 whitespace-pre-wrap break-all">
                                  <span className="text-emerald-400/50">Output: </span>{tc.output}
                                </div>
                              )}
                            </div>
                          )}
                        </button>
                      );
                    })}
                  </div>
                )}

                {/* Waiting indicator */}
                {msg.streaming && !msg.content && (!msg.toolCalls || msg.toolCalls.length === 0) && (
                  <span className="text-brand-shade3/50 text-[11px]">Waiting for response...</span>
                )}
              </div>
            )}
          </div>
        ))}

        {/* Streaming indicator */}
        {isStreaming && (
          <div className="flex justify-end">
            <span className="text-[10px] text-brand-accent animate-pulse">streaming...</span>
          </div>
        )}

        {/* View in Inspect link */}
        {showInspectLink && (sessionId || protoSessionId) && (
          <div className="flex justify-start mt-1">
            <button
              onClick={() => {
                const sid = sessionId || protoSessionId;
                const schema = selectedSchema || 'default';
                navigate(`/builder/${encodeURIComponent(schema)}/${encodeURIComponent(selectedAgent)}/inspect/${encodeURIComponent(sid)}`);
              }}
              className="text-[11px] text-brand-accent hover:text-brand-accent-hover transition-colors inline-flex items-center gap-1"
            >
              View in Inspect
              <svg width="10" height="10" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round">
                <polyline points="9 18 15 12 9 6" />
              </svg>
            </button>
          </div>
        )}

        <div ref={messagesEndRef} />
      </div>

      {/* Context usage bar */}
      <ContextUsageBar maxContextTokens={agentDetail?.max_context_size ?? null} totalTokens={isPrototype ? null : sseChat.tokenUsage} contextTokens={isPrototype ? null : sseChat.contextTokens} />

      {/* Input area */}
      <div className="flex items-center gap-2 px-3 py-2 border-t border-brand-shade3/10 flex-shrink-0">
        <textarea
          ref={inputRef}
          value={message}
          onChange={(e) => {
            setMessage(e.target.value);
            e.target.style.height = 'auto';
            e.target.style.height = Math.min(e.target.scrollHeight, 120) + 'px';
          }}
          onKeyDown={handleKeyDown}
          placeholder="Send test message to entry agent..."
          rows={1}
          className="flex-1 px-2.5 py-1.5 bg-brand-dark-alt border border-brand-shade3/20 rounded-card text-xs text-brand-light placeholder-brand-shade3 font-mono focus:outline-none focus:border-brand-accent resize-none transition-colors"
          style={{ maxHeight: '120px', overflowY: 'auto' }}
        />
        {isStreaming ? (
          <button
            onClick={handleStop}
            className="px-2.5 py-1.5 bg-brand-dark border border-brand-shade3/30 rounded-card text-xs text-brand-shade2 hover:text-brand-light hover:border-brand-shade3 transition-colors flex-shrink-0 inline-flex items-center gap-1"
          >
            <svg width="10" height="10" viewBox="0 0 24 24" fill="currentColor">
              <rect x="4" y="4" width="16" height="16" rx="2" />
            </svg>
            Stop
          </button>
        ) : (
          <button
            onClick={handleSend}
            disabled={!message.trim() || !selectedAgent}
            className="px-2.5 py-1.5 bg-brand-accent text-brand-light rounded-card text-xs font-medium hover:bg-brand-accent-hover disabled:opacity-40 transition-colors flex-shrink-0 inline-flex items-center gap-1"
          >
            <svg width="10" height="10" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round">
              <polygon points="5 3 19 12 5 21 5 3" />
            </svg>
            Run
          </button>
        )}
      </div>
    </div>
  );
}
