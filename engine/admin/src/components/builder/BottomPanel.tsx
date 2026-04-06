import { useState, useRef, useCallback, type ReactNode } from 'react';

interface BottomPanelTab {
  id: string;
  label: string;
  icon: ReactNode;
  content: ReactNode;
}

interface BottomPanelProps {
  tabs: BottomPanelTab[];
  defaultTab?: string;
  defaultHeight?: number;
  minHeight?: number;
  maxHeight?: number;
  onSendMessage?: (message: string) => void;
  inputPlaceholder?: string;
}

export default function BottomPanel({
  tabs,
  defaultTab,
  defaultHeight = 300,
  minHeight = 48,
  maxHeight,
  onSendMessage,
  inputPlaceholder = 'Type a message...',
}: BottomPanelProps) {
  const [inputValue, setInputValue] = useState('');
  const [height, setHeight] = useState(defaultHeight);
  const [collapsed, setCollapsed] = useState(false);
  const [activeTab, setActiveTab] = useState(defaultTab ?? tabs[0]?.id ?? '');
  const dragRef = useRef<{ startY: number; startHeight: number } | null>(null);

  const resolvedMax = maxHeight ?? Math.round((typeof window !== 'undefined' ? window.innerHeight : 800) * 0.6);

  const onMouseDown = useCallback(
    (e: React.MouseEvent) => {
      e.preventDefault();
      dragRef.current = { startY: e.clientY, startHeight: height };

      const onMouseMove = (ev: MouseEvent) => {
        if (!dragRef.current) return;
        const delta = dragRef.current.startY - ev.clientY;
        const next = Math.min(
          resolvedMax,
          Math.max(minHeight, dragRef.current.startHeight + delta),
        );
        setHeight(next);
        if (next > minHeight) setCollapsed(false);
      };

      const onMouseUp = () => {
        dragRef.current = null;
        document.removeEventListener('mousemove', onMouseMove);
        document.removeEventListener('mouseup', onMouseUp);
      };

      document.addEventListener('mousemove', onMouseMove);
      document.addEventListener('mouseup', onMouseUp);
    },
    [height, minHeight, resolvedMax],
  );

  const toggleCollapse = () => setCollapsed((c) => !c);

  const activeContent = tabs.find((t) => t.id === activeTab)?.content;

  return (
    <div
      className="flex flex-col bg-brand-dark-surface border-t border-brand-shade3/10 font-mono select-none overflow-hidden"
      style={{ height: collapsed ? minHeight : height }}
    >
      {/* Drag handle */}
      <div
        className="flex items-center justify-center h-4 cursor-row-resize flex-shrink-0 hover:bg-brand-shade3/5"
        onMouseDown={onMouseDown}
      >
        <div className="h-1 w-12 bg-brand-shade3/30 rounded-full my-1" />
      </div>

      {/* Tab bar */}
      <div className="flex items-center border-b border-brand-shade3/10 flex-shrink-0 px-2">
        <div className="flex items-center flex-1 overflow-x-auto">
          {tabs.map((tab) => (
            <button
              key={tab.id}
              onClick={() => {
                setActiveTab(tab.id);
                if (collapsed) setCollapsed(false);
              }}
              className={[
                'flex items-center gap-1.5 px-4 py-2 text-sm font-medium transition-colors whitespace-nowrap',
                tab.id === activeTab
                  ? 'text-brand-light border-b-2 border-brand-accent'
                  : 'text-brand-shade3 hover:text-brand-shade2',
              ].join(' ')}
            >
              {tab.icon}
              {tab.label}
            </button>
          ))}
        </div>

        {/* Collapse/expand */}
        <button
          onClick={toggleCollapse}
          className="p-1.5 text-brand-shade3 hover:text-brand-shade2 transition-colors flex-shrink-0"
          title={collapsed ? 'Expand' : 'Collapse'}
          aria-label={collapsed ? 'Expand panel' : 'Collapse panel'}
        >
          <svg
            width="14"
            height="14"
            viewBox="0 0 14 14"
            fill="none"
            xmlns="http://www.w3.org/2000/svg"
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

      {/* Content */}
      {!collapsed && (
        <div className="flex-1 overflow-y-auto">
          {activeContent}
        </div>
      )}

      {/* Message input */}
      {!collapsed && (
        <div className="flex items-center gap-2 px-3 py-2 border-t border-brand-shade3/10 flex-shrink-0">
          <input
            type="text"
            value={inputValue}
            onChange={(e) => setInputValue(e.target.value)}
            onKeyDown={(e) => {
              if (e.key === 'Enter' && inputValue.trim()) {
                onSendMessage?.(inputValue.trim());
                setInputValue('');
              }
            }}
            placeholder={inputPlaceholder}
            aria-label="Message input"
            className="flex-1 bg-brand-dark-alt border border-brand-shade3/20 rounded-card text-brand-light text-xs px-2.5 py-1.5 outline-none font-mono focus:border-brand-accent transition-colors"
          />
          <button
            type="button"
            onClick={() => {
              if (inputValue.trim()) {
                onSendMessage?.(inputValue.trim());
                setInputValue('');
              }
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
