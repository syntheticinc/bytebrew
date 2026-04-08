import { useRef, useCallback, useState, useEffect } from 'react';
import { useBottomPanel } from '../hooks/useBottomPanel';
import SchemaSelector from './SchemaSelector';
import TestFlowTab from './TestFlowTab';
import { api } from '../api/client';

const MIN_HEIGHT = 150;
const COLLAPSED_HEIGHT = 40;
const MAX_HEIGHT_RATIO = 0.7;

export default function BottomPanel() {
  const { height, tab, collapsed, setHeight, setTab, setCollapsed, toggleCollapsed, selectedSchema } = useBottomPanel();
  const dragRef = useRef<{ startY: number; startHeight: number } | null>(null);
  const [assistantInput, setAssistantInput] = useState('');
  const [messages, setMessages] = useState<Array<{ role: 'user' | 'assistant'; content: string }>>([]);
  const [loading, setLoading] = useState(false);
  const sessionId = useRef('assistant-' + Math.random().toString(36).slice(2));
  const messagesEndRef = useRef<HTMLDivElement>(null);

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
    if (!text || loading) return;
    setAssistantInput('');
    setMessages((prev) => [...prev, { role: 'user', content: text }]);
    setLoading(true);
    try {
      const result = await api.assistantChat(text, sessionId.current);
      setMessages((prev) => [...prev, { role: 'assistant', content: result.response }]);
    } catch {
      setMessages((prev) => [...prev, { role: 'assistant', content: 'Error: could not reach assistant.' }]);
    } finally {
      setLoading(false);
    }
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

        {/* Schema selector */}
        <SchemaSelector />

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
        <div className="flex-1 overflow-y-auto">
          {tab === 'assistant' && (
            <div className="flex flex-col h-full">
              <div className="flex-1 overflow-y-auto p-4 flex flex-col gap-2">
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
                    {messages.map((msg, i) => (
                      <div key={i} className={`flex ${msg.role === 'user' ? 'justify-end' : 'justify-start'}`}>
                        <div className={`max-w-[80%] px-3 py-2 rounded-card text-xs font-mono whitespace-pre-wrap ${
                          msg.role === 'user'
                            ? 'bg-brand-accent/20 text-brand-light'
                            : 'bg-brand-dark border border-brand-shade3/20 text-brand-shade2'
                        }`}>
                          {msg.content}
                        </div>
                      </div>
                    ))}
                    {loading && (
                      <div className="flex justify-start">
                        <div className="px-3 py-2 rounded-card text-xs font-mono text-brand-shade3 bg-brand-dark border border-brand-shade3/20">
                          <span className="animate-pulse">Thinking…</span>
                        </div>
                      </div>
                    )}
                  </>
                )}
                <div ref={messagesEndRef} />
              </div>
            </div>
          )}
          {tab === 'testflow' && (
            <TestFlowTab />
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
