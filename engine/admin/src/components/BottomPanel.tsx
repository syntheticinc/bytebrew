import React, { useRef, useCallback, useEffect, useState } from 'react';
import { useLocation, matchPath } from 'react-router-dom';
import { useBottomPanel } from '../hooks/useBottomPanel';
import { useSSEChat } from '../hooks/useSSEChat';
import { dispatchAdminChanged } from '../hooks/useAdminRefresh';
import SchemaSelector from './SchemaSelector';
import TestFlowTab from './TestFlowTab';
import ContextUsageBar from './ContextUsageBar';
import { api } from '../api/client';

const CURSOR = <span className="inline-block w-1.5 h-3 bg-brand-accent ml-0.5 animate-pulse align-middle" />;

function renderMarkdown(text: string, showCursor = false): React.ReactNode {
  const lines = text.split('\n');
  const lastIdx = lines.length - 1;
  return (
    <div className="space-y-1">
      {lines.map((line, i) => {
        const isLast = i === lastIdx;
        const cursor = showCursor && isLast ? CURSOR : null;
        if (line.startsWith('### ')) {
          return <p key={i} className="font-semibold text-brand-light mt-2 first:mt-0">{renderInline(line.slice(4))}{cursor}</p>;
        }
        if (line.startsWith('## ')) {
          return <p key={i} className="font-semibold text-brand-light mt-2 first:mt-0">{renderInline(line.slice(3))}{cursor}</p>;
        }
        if (line.startsWith('- ') || line.startsWith('* ')) {
          return <p key={i} className="pl-3 before:content-['·'] before:mr-1.5 before:text-brand-shade3">{renderInline(line.slice(2))}{cursor}</p>;
        }
        if (line === '---' || line === '') {
          return <span key={i}>{cursor}</span>;
        }
        return <p key={i}>{renderInline(line)}{cursor}</p>;
      })}
    </div>
  );
}

function renderInline(text: string): React.ReactNode {
  const parts = text.split(/(\*\*[^*]+\*\*)/g);
  return parts.map((part, i) =>
    part.startsWith('**') && part.endsWith('**')
      ? <strong key={i} className="text-brand-light font-semibold">{part.slice(2, -2)}</strong>
      : part
  );
}

const BREW_PHRASES = ['Grinding beans...', 'Brewing...', 'Pulling a shot...', 'Steaming...', 'Almost ready...'];
let brewCounter = 0;

function BrewingSpinner() {
  const phrase = BREW_PHRASES[brewCounter++ % BREW_PHRASES.length];
  return (
    <div className="flex items-center gap-2 py-1">
      <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="1.5" className="w-3.5 h-3.5 text-brand-shade3">
        <path d="M17 8h1a4 4 0 010 8h-1" strokeLinecap="round" />
        <path d="M3 8h14v9a4 4 0 01-4 4H7a4 4 0 01-4-4V8z" />
        <path d="M7 2v3" strokeLinecap="round" style={{ animation: 'brewSteam 1.2s ease-in-out infinite' }} />
        <path d="M10 1v3" strokeLinecap="round" style={{ animation: 'brewSteam 1.2s ease-in-out infinite 0.3s' }} />
        <path d="M13 2v3" strokeLinecap="round" style={{ animation: 'brewSteam 1.2s ease-in-out infinite 0.6s' }} />
      </svg>
      <span className="text-xs font-mono text-brand-shade3 animate-pulse">{phrase}</span>
      <style>{`@keyframes brewSteam{0%{opacity:.3;transform:translateY(0)}50%{opacity:1;transform:translateY(-3px)}100%{opacity:.3;transform:translateY(0)}}`}</style>
    </div>
  );
}

/* ── Example prompts for empty state (m-01) ────────────────────────────────── */
const EXAMPLE_PROMPTS = [
  'Create a support agent with escalation',
  'List all agents in this schema',
  'Add memory capability to an agent',
  'Set up a webhook trigger',
];

const MIN_HEIGHT = 150;
const COLLAPSED_HEIGHT = 40;
const MAX_HEIGHT_RATIO = 0.7;
const ASSISTANT_AGENT = 'builder-assistant';

export default function BottomPanel() {
  const { height, tab, collapsed, setHeight, setTab, setCollapsed, toggleCollapsed, selectedSchema } = useBottomPanel();
  const location = useLocation();
  const canvasMatch = matchPath({ path: '/builder/:schemaName', end: false }, location.pathname);
  const lockedSchema = canvasMatch?.params?.schemaName ? decodeURIComponent(canvasMatch.params.schemaName) : null;
  const dragRef = useRef<{ startY: number; startHeight: number } | null>(null);
  const [assistantInput, setAssistantInput] = useState('');
  const [expandedTools, setExpandedTools] = useState<Record<string, boolean>>({});
  const [maxContextTokens, setMaxContextTokens] = useState<number | null>(null);
  const messagesEndRef = useRef<HTMLDivElement>(null);

  // M-01: Pass schema context on ALL pages — lockedSchema (canvas) or selectedSchema (other pages)
  const effectiveSchema = lockedSchema ?? selectedSchema ?? undefined;

  const assistantPersistenceKey = effectiveSchema ? `bb_assistant_${effectiveSchema}` : undefined;
  const { messages, sendMessage, isStreaming, isRestoring, resetSession, tokenUsage } = useSSEChat({
    endpoint: `/api/v1/admin/assistant/chat`,
    agentName: ASSISTANT_AGENT,
    schemaContext: effectiveSchema,
    persistenceKey: assistantPersistenceKey,
    fetchMessages: (sid) => api.getSessionMessages(sid),
    onToolResult: (tool) => {
      if (tool.startsWith('admin_')) {
        dispatchAdminChanged(tool);
      }
    },
  });

  useEffect(() => {
    api.getAgent(ASSISTANT_AGENT)
      .then((d) => setMaxContextTokens(d.max_context_size || null))
      .catch(() => {});
  }, []);

  useEffect(() => {
    messagesEndRef.current?.scrollIntoView({ behavior: 'smooth' });
  }, [messages]);

  const maxHeight = Math.round(
    (typeof window !== 'undefined' ? window.innerHeight : 800) * MAX_HEIGHT_RATIO,
  );

  const onMouseDown = useCallback(
    (e: React.MouseEvent) => {
      e.preventDefault();
      dragRef.current = { startY: e.clientY, startHeight: height };

      const onMouseMove = (ev: MouseEvent) => {
        if (!dragRef.current) return;
        const delta = dragRef.current.startY - ev.clientY;
        const next = Math.min(maxHeight, Math.max(MIN_HEIGHT, dragRef.current.startHeight + delta));
        setHeight(next);
        if (collapsed) setCollapsed(false);
      };

      const onMouseUp = () => {
        dragRef.current = null;
        document.removeEventListener('mousemove', onMouseMove);
        document.removeEventListener('mouseup', onMouseUp);
      };

      document.addEventListener('mousemove', onMouseMove);
      document.addEventListener('mouseup', onMouseUp);
    },
    [height, maxHeight, collapsed, setHeight, setCollapsed],
  );

  const handleTabClick = (tabId: 'assistant' | 'testflow') => {
    if (collapsed) setCollapsed(false);
    setTab(tabId);
  };

  const handleSendAssistant = async (text?: string) => {
    const msg = (text ?? assistantInput).trim();
    if (!msg || isStreaming) return;
    setAssistantInput('');
    await sendMessage(msg);
  };

  const toggleTool = (key: string) => {
    setExpandedTools((prev) => ({ ...prev, [key]: !prev[key] }));
  };

  return (
    <div
      className="flex flex-col bg-brand-dark-surface border-t border-brand-shade3/10 font-mono select-none overflow-hidden flex-shrink-0"
      style={{ height: collapsed ? COLLAPSED_HEIGHT : height }}
    >
      {/* Drag handle */}
      <div
        className="flex items-center justify-center h-3 cursor-row-resize flex-shrink-0 hover:bg-brand-shade3/5 group"
        onMouseDown={onMouseDown}
      >
        <div className="h-[3px] w-10 bg-brand-shade3/25 rounded-full group-hover:bg-brand-shade3/40 transition-colors" />
      </div>

      {/* Tab bar + schema selector + collapse toggle */}
      <div className="flex items-center border-b border-brand-shade3/10 flex-shrink-0 px-2 gap-1">
        {/* Tabs */}
        <div className="flex items-center flex-1 overflow-x-auto">
          <button
            onClick={() => handleTabClick('assistant')}
            className={[
              'flex items-center gap-1.5 px-3 py-2 text-xs font-medium transition-colors whitespace-nowrap',
              tab === 'assistant'
                ? 'text-brand-light border-b-2 border-brand-accent'
                : 'text-brand-shade3 hover:text-brand-shade2',
            ].join(' ')}
          >
            <svg width="13" height="13" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="1.5" strokeLinecap="round" strokeLinejoin="round">
              <rect x="4" y="4" width="16" height="16" rx="2" />
              <rect x="9" y="9" width="6" height="6" rx="1" />
            </svg>
            AI Assistant
          </button>
          <button
            onClick={() => handleTabClick('testflow')}
            className={[
              'flex items-center gap-1.5 px-3 py-2 text-xs font-medium transition-colors whitespace-nowrap',
              tab === 'testflow'
                ? 'text-brand-light border-b-2 border-brand-accent'
                : 'text-brand-shade3 hover:text-brand-shade2',
            ].join(' ')}
          >
            <svg width="13" height="13" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="1.5" strokeLinecap="round" strokeLinejoin="round">
              <polygon points="5 3 19 12 5 21 5 3" />
            </svg>
            Test Flow
          </button>
        </div>

        {/* m-02: New Session button for AI Assistant */}
        {tab === 'assistant' && messages.length > 0 && (
          <button
            onClick={resetSession}
            className="flex items-center gap-1 px-2 py-1 text-[10px] text-brand-shade3 hover:text-brand-shade2 transition-colors"
            title="New session"
          >
            <svg width="11" height="11" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round">
              <path d="M23 4v6h-6" /><path d="M20.49 15a9 9 0 11-2.12-9.36L23 10" />
            </svg>
          </button>
        )}

        {/* Schema selector — locked when on canvas page */}
        {lockedSchema ? (
          <div
            className="flex items-center gap-1.5 px-2.5 py-1 rounded-btn text-xs font-medium text-brand-shade3 border border-brand-shade3/10"
            title="Schema locked to current canvas"
          >
            <svg width="12" height="12" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="1.5" strokeLinecap="round" strokeLinejoin="round">
              <path d="M22 19a2 2 0 01-2 2H4a2 2 0 01-2-2V5a2 2 0 012-2h5l2 3h9a2 2 0 012 2z" />
            </svg>
            <span className="max-w-[140px] truncate">{lockedSchema}</span>
            <svg width="10" height="10" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="1.5" strokeLinecap="round" strokeLinejoin="round" className="opacity-40">
              <rect x="3" y="11" width="18" height="11" rx="2" ry="2" />
              <path d="M7 11V7a5 5 0 0110 0v4" />
            </svg>
          </div>
        ) : (
          <SchemaSelector />
        )}

        {/* Collapse/expand */}
        <button
          onClick={toggleCollapsed}
          className="p-1.5 text-brand-shade3 hover:text-brand-shade2 transition-colors flex-shrink-0 ml-1"
          title={collapsed ? 'Expand panel' : 'Collapse panel'}
          aria-label={collapsed ? 'Expand panel' : 'Collapse panel'}
        >
          <svg
            width="14"
            height="14"
            viewBox="0 0 14 14"
            fill="none"
            className={`transition-transform ${collapsed ? 'rotate-180' : ''}`}
          >
            <path
              d="M3 5L7 9L11 5"
              stroke="currentColor"
              strokeWidth="1.5"
              strokeLinecap="round"
              strokeLinejoin="round"
            />
          </svg>
        </button>
      </div>

      {/* Content area */}
      {!collapsed && (
        <div className="flex-1 min-h-0 flex flex-col">
          {tab === 'assistant' && (
            <div className="flex-1 min-h-0 overflow-y-auto p-4 flex flex-col gap-2">
              {isRestoring && messages.length === 0 ? (
                <div className="flex items-center gap-2 text-xs text-brand-shade3 font-mono py-4 justify-center">
                  <svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" className="animate-spin">
                    <path d="M21 12a9 9 0 11-6.219-8.56" />
                  </svg>
                  Restoring session...
                </div>
              ) : messages.length === 0 ? (
                <div className="flex flex-col gap-2 text-xs text-brand-shade2 font-mono">
                  {/* cos-01: Removed redundant "— {schema}" — schema shown in selector */}
                  <div className="flex items-center gap-2 text-brand-shade3">
                    <svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="1.5" strokeLinecap="round" strokeLinejoin="round">
                      <rect x="4" y="4" width="16" height="16" rx="2" />
                      <rect x="9" y="9" width="6" height="6" rx="1" />
                    </svg>
                    <span>AI Assistant</span>
                  </div>
                  <p className="text-brand-shade3/80 mt-1">
                    Describe what you need. The assistant will configure agents, schemas, and flows for you.
                  </p>
                  {/* m-01: Example prompt chips */}
                  <div className="flex flex-wrap gap-1.5 mt-2">
                    {EXAMPLE_PROMPTS.map((prompt) => (
                      <button
                        key={prompt}
                        onClick={() => handleSendAssistant(prompt)}
                        className="px-2.5 py-1 bg-brand-dark border border-brand-shade3/15 rounded-full text-[11px] text-brand-shade3 hover:text-brand-light hover:border-brand-shade3/30 transition-colors"
                      >
                        {prompt}
                      </button>
                    ))}
                  </div>
                </div>
              ) : (
                <>
                  {messages.map((msg) => {
                    // Hide empty streaming assistant bubble — spinner renders separately
                    if (msg.role === 'assistant' && msg.streaming && msg.content === '' && (!msg.toolCalls || msg.toolCalls.length === 0)) return null;
                    return (
                      <div key={msg.id}>
                        {/* C-01: Tool calls rendered BEFORE text (chronological order: agent calls tools first, then responds) */}
                        {msg.role === 'assistant' && msg.toolCalls && msg.toolCalls.length > 0 && (
                          <div className="space-y-1 mt-1 ml-0">
                            {msg.toolCalls.map((tc, i) => {
                              const key = `${msg.id}-tc-${i}`;
                              const isExpanded = expandedTools[key] ?? false;
                              return (
                                <button
                                  key={i}
                                  onClick={() => toggleTool(key)}
                                  className="w-full text-left px-2 py-1 bg-brand-dark border border-brand-shade3/15 rounded text-[11px] font-mono hover:border-brand-shade3/30 transition-colors"
                                >
                                  <div className="flex items-center gap-1.5">
                                    <svg width="10" height="10" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" className="text-blue-400 flex-shrink-0">
                                      <circle cx="12" cy="12" r="3" />
                                      <path d="M19.4 15a1.65 1.65 0 00.33 1.82l.06.06a2 2 0 010 2.83 2 2 0 01-2.83 0l-.06-.06a1.65 1.65 0 00-1.82-.33 1.65 1.65 0 00-1 1.51V21a2 2 0 01-4 0v-.09A1.65 1.65 0 009 19.4a1.65 1.65 0 00-1.82.33l-.06.06a2 2 0 01-2.83-2.83l.06-.06A1.65 1.65 0 004.68 15a1.65 1.65 0 00-1.51-1H3a2 2 0 010-4h.09A1.65 1.65 0 004.6 9a1.65 1.65 0 00-.33-1.82l-.06-.06a2 2 0 012.83-2.83l.06.06A1.65 1.65 0 009 4.68a1.65 1.65 0 001-1.51V3a2 2 0 014 0v.09a1.65 1.65 0 001 1.51 1.65 1.65 0 001.82-.33l.06-.06a2 2 0 012.83 2.83l-.06.06A1.65 1.65 0 0019.4 9a1.65 1.65 0 001.51 1H21a2 2 0 010 4h-.09a1.65 1.65 0 00-1.51 1z" />
                                    </svg>
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
                        {/* Text content rendered AFTER tool calls (chronological order) */}
                        {msg.content && (
                          <div className={`flex ${msg.role === 'user' ? 'justify-end' : 'justify-start'} ${msg.role === 'assistant' && msg.toolCalls?.length ? 'mt-1' : ''}`}>
                            <div className={`max-w-[80%] px-3 py-2 rounded-card text-xs font-mono break-words ${
                              msg.role === 'user'
                                ? 'bg-brand-accent/20 text-brand-light'
                                : 'bg-brand-dark border border-brand-shade3/20 text-brand-shade2'
                            }`}>
                              {msg.role === 'assistant'
                                ? renderMarkdown(msg.content, !!msg.streaming && msg.content !== '')
                                : msg.content}
                            </div>
                          </div>
                        )}
                      </div>
                    );
                  })}
                </>
              )}
              {/* M-05: Show brewing spinner during streaming (both empty content and with tool calls) */}
              {isStreaming && (() => {
                const lastMsg = messages[messages.length - 1];
                if (!lastMsg || lastMsg.role !== 'assistant') return null;
                if (lastMsg.content === '' || (lastMsg.toolCalls && lastMsg.toolCalls.length > 0 && lastMsg.streaming)) {
                  return (
                    <div className="flex justify-start">
                      <BrewingSpinner />
                    </div>
                  );
                }
                return null;
              })()}
              <div ref={messagesEndRef} />
            </div>
          )}
          {tab === 'testflow' && (
            <div className="flex-1 min-h-0 overflow-y-auto">
              <TestFlowTab />
            </div>
          )}
        </div>
      )}

      {/* Context usage bar — assistant tab only */}
      {!collapsed && tab === 'assistant' && (
        <ContextUsageBar maxContextTokens={maxContextTokens} totalTokens={tokenUsage} />
      )}

      {/* Message input — assistant tab only (testflow has its own) */}
      {!collapsed && tab === 'assistant' && (
        <div className="flex items-center gap-2 px-3 py-2 border-t border-brand-shade3/10 flex-shrink-0">
          <input
            type="text"
            value={assistantInput}
            onChange={(e) => setAssistantInput(e.target.value)}
            onKeyDown={(e) => {
              if (e.key === 'Enter') handleSendAssistant();
            }}
            placeholder={isStreaming ? 'Assistant is working...' : 'Ask AI to configure agents...'}
            disabled={isStreaming}
            aria-label="Assistant message input"
            className="flex-1 bg-brand-dark-alt border border-brand-shade3/20 rounded-card text-brand-light text-xs px-2.5 py-1.5 outline-none font-mono focus:border-brand-accent transition-colors disabled:opacity-50"
          />
          <button
            type="button"
            onClick={() => handleSendAssistant()}
            disabled={isStreaming}
            aria-label="Send message"
            className="bg-brand-accent hover:bg-brand-accent-hover border-none rounded-card text-brand-light text-xs px-3 py-1.5 cursor-pointer font-medium font-mono transition-colors disabled:opacity-50"
          >
            Send
          </button>
        </div>
      )}
    </div>
  );
}
