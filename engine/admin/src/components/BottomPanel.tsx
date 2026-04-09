import React, { useRef, useCallback, useEffect, useState } from 'react';
import { useLocation, matchPath } from 'react-router-dom';
import { useBottomPanel } from '../hooks/useBottomPanel';
import { useSSEChat } from '../hooks/useSSEChat';
import { dispatchAdminChanged } from '../hooks/useAdminRefresh';
import SchemaSelector from './SchemaSelector';
import TestFlowTab from './TestFlowTab';

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
  const messagesEndRef = useRef<HTMLDivElement>(null);

  const { messages, sendMessage, isStreaming } = useSSEChat({
    endpoint: `/api/v1/admin/assistant/chat`,
    agentName: ASSISTANT_AGENT,
    schemaContext: lockedSchema ?? undefined,
    onToolResult: (tool) => {
      if (tool.startsWith('admin_')) {
        dispatchAdminChanged(tool);
      }
    },
  });

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

  const handleSendAssistant = async () => {
    const text = assistantInput.trim();
    if (!text || isStreaming) return;
    setAssistantInput('');
    await sendMessage(text);
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

        {/* Schema selector — locked when on canvas page */}
        {lockedSchema ? (
          <div className="flex items-center gap-1.5 px-2.5 py-1 rounded-btn text-xs font-medium text-brand-shade3 border border-brand-shade3/10">
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
              {messages.length === 0 ? (
                <div className="flex flex-col gap-2 text-xs text-brand-shade2 font-mono">
                  <div className="flex items-center gap-2 text-brand-shade3">
                    <svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="1.5" strokeLinecap="round" strokeLinejoin="round">
                      <rect x="4" y="4" width="16" height="16" rx="2" />
                      <rect x="9" y="9" width="6" height="6" rx="1" />
                    </svg>
                    <span>AI Assistant</span>
                    {selectedSchema && (
                      <span className="text-brand-shade3/60">— {selectedSchema}</span>
                    )}
                  </div>
                  <p className="text-brand-shade3/80 mt-1">
                    Describe what you need. The assistant will configure agents, schemas, and flows for you.
                  </p>
                </div>
              ) : (
                <>
                  {messages.map((msg) => {
                    // Hide empty streaming assistant bubble — spinner renders separately
                    if (msg.role === 'assistant' && msg.streaming && msg.content === '') return null;
                    return (
                      <div key={msg.id} className={`flex ${msg.role === 'user' ? 'justify-end' : 'justify-start'}`}>
                        <div className={`max-w-[80%] px-3 py-2 rounded-card text-xs font-mono ${
                          msg.role === 'user'
                            ? 'bg-brand-accent/20 text-brand-light'
                            : 'bg-brand-dark border border-brand-shade3/20 text-brand-shade2'
                        }`}>
                          {msg.role === 'assistant'
                            ? renderMarkdown(msg.content, !!msg.streaming && msg.content !== '')
                            : msg.content}
                        </div>
                      </div>
                    );
                  })}
                </>
              )}
              {isStreaming && messages[messages.length - 1]?.role === 'assistant' &&
               messages[messages.length - 1]?.content === '' && (
                <div className="flex justify-start">
                  <BrewingSpinner />
                </div>
              )}
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
            placeholder="Ask AI to configure agents..."
            aria-label="Assistant message input"
            className="flex-1 bg-brand-dark-alt border border-brand-shade3/20 rounded-card text-brand-light text-xs px-2.5 py-1.5 outline-none font-mono focus:border-brand-accent transition-colors"
          />
          <button
            type="button"
            onClick={handleSendAssistant}
            aria-label="Send message"
            className="bg-brand-accent hover:bg-brand-accent-hover border-none rounded-card text-brand-light text-xs px-3 py-1.5 cursor-pointer font-medium font-mono transition-colors"
          >
            Send
          </button>
        </div>
      )}
    </div>
  );
}
