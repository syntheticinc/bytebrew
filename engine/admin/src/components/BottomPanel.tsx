import { useRef, useCallback, useState } from 'react';
import { useBottomPanel } from '../hooks/useBottomPanel';
import SchemaSelector from './SchemaSelector';

const MIN_HEIGHT = 150;
const COLLAPSED_HEIGHT = 40;
const MAX_HEIGHT_RATIO = 0.7;

export default function BottomPanel() {
  const { height, tab, collapsed, setHeight, setTab, setCollapsed, toggleCollapsed, selectedSchema } = useBottomPanel();
  const dragRef = useRef<{ startY: number; startHeight: number } | null>(null);
  const [assistantInput, setAssistantInput] = useState('');
  const [testInput, setTestInput] = useState('');

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
    if (collapsed) {
      setCollapsed(false);
      setTab(tabId);
    } else {
      setTab(tabId);
    }
  };

  const handleSendAssistant = () => {
    if (!assistantInput.trim()) return;
    // Placeholder — real assistant integration in Phase 5
    setAssistantInput('');
  };

  const handleSendTest = () => {
    if (!testInput.trim()) return;
    // Placeholder — real test flow in Phase 2 WP-13
    setTestInput('');
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
              <div className="flex-1 p-4">
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
              </div>
            </div>
          )}
          {tab === 'testflow' && (
            <div className="flex flex-col h-full">
              <div className="flex-1 p-4">
                <div className="flex flex-col gap-2 text-xs text-brand-shade2 font-mono">
                  <div className="flex items-center gap-2 text-brand-shade3">
                    <svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="1.5" strokeLinecap="round" strokeLinejoin="round">
                      <polygon points="5 3 19 12 5 21 5 3" />
                    </svg>
                    <span>Test Flow</span>
                    {selectedSchema && (
                      <span className="text-brand-shade3/60">— {selectedSchema}</span>
                    )}
                  </div>
                  <p className="text-brand-shade3/80 mt-1">
                    Test Flow will be available in a future update. You'll be able to send test messages to entry agents and observe the flow execution.
                  </p>
                </div>
              </div>
            </div>
          )}
        </div>
      )}

      {/* Message input */}
      {!collapsed && (
        <div className="flex items-center gap-2 px-3 py-2 border-t border-brand-shade3/10 flex-shrink-0">
          <input
            type="text"
            value={tab === 'assistant' ? assistantInput : testInput}
            onChange={(e) => {
              if (tab === 'assistant') setAssistantInput(e.target.value);
              else setTestInput(e.target.value);
            }}
            onKeyDown={(e) => {
              if (e.key === 'Enter') {
                if (tab === 'assistant') handleSendAssistant();
                else handleSendTest();
              }
            }}
            placeholder={
              tab === 'assistant'
                ? 'Ask AI to configure agents...'
                : 'Send test message to entry agent...'
            }
            aria-label={tab === 'assistant' ? 'Assistant message input' : 'Test flow message input'}
            className="flex-1 bg-brand-dark-alt border border-brand-shade3/20 rounded-card text-brand-light text-xs px-2.5 py-1.5 outline-none font-mono focus:border-brand-accent transition-colors"
          />
          <button
            type="button"
            onClick={() => {
              if (tab === 'assistant') handleSendAssistant();
              else handleSendTest();
            }}
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
