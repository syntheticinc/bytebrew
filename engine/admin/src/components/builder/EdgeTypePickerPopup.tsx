import { useEffect, useRef } from 'react';

// ---------------------------------------------------------------------------
// Types
// ---------------------------------------------------------------------------

export type EdgeType = 'flow' | 'transfer' | 'can_spawn';

interface EdgeOption {
  type: EdgeType;
  label: string;
  description: string;
  colorClass: string;
  stroke: string;
}

// ---------------------------------------------------------------------------
// Constants — colors match EdgeConfigPanel EDGE_TYPE_COLORS
// ---------------------------------------------------------------------------

const EDGE_OPTIONS: EdgeOption[] = [
  {
    type: 'flow',
    label: 'Flow',
    description: 'Sequential: A completes, then B runs',
    colorClass: 'text-green-400',
    stroke: '#4CAF50',
  },
  {
    type: 'transfer',
    label: 'Transfer',
    description: 'Hand off: A delegates control to B',
    colorClass: 'text-blue-400',
    stroke: '#3B82F6',
  },
  {
    type: 'can_spawn',
    label: 'Can Spawn',
    description: 'A can create B as a sub-agent',
    colorClass: 'text-red-400',
    stroke: '#D7513E',
  },
];

// ---------------------------------------------------------------------------
// Component
// ---------------------------------------------------------------------------

interface EdgeTypePickerPopupProps {
  position: { x: number; y: number };
  onSelect: (type: EdgeType) => void;
  onCancel: () => void;
}

export default function EdgeTypePickerPopup({ position, onSelect, onCancel }: EdgeTypePickerPopupProps) {
  const ref = useRef<HTMLDivElement>(null);

  useEffect(() => {
    function handlePointerDown(e: MouseEvent) {
      if (ref.current && !ref.current.contains(e.target as Node)) {
        onCancel();
      }
    }
    document.addEventListener('mousedown', handlePointerDown);
    return () => document.removeEventListener('mousedown', handlePointerDown);
  }, [onCancel]);

  // Clamp position so popup doesn't overflow viewport
  const x = Math.min(position.x + 8, window.innerWidth - 224);
  const y = Math.min(position.y - 8, window.innerHeight - 220);

  return (
    <div
      ref={ref}
      className="fixed z-50 bg-brand-dark-alt border border-brand-shade3/20 rounded-card shadow-xl w-52"
      style={{ left: x, top: y }}
    >
      <div className="px-3 py-2 border-b border-brand-shade3/15">
        <span className="text-[10px] text-brand-shade3 uppercase tracking-wider">Select Edge Type</span>
      </div>
      <div className="p-1">
        {EDGE_OPTIONS.map((opt) => (
          <button
            key={opt.type}
            onClick={() => onSelect(opt.type)}
            className="w-full flex items-start gap-2.5 px-2 py-2 rounded hover:bg-brand-shade3/10 transition-colors text-left"
          >
            {/* Mini edge preview */}
            <span className="mt-1 flex-shrink-0">
              <svg width="16" height="8" viewBox="0 0 16 8">
                <line x1="0" y1="4" x2="11" y2="4" stroke={opt.stroke} strokeWidth="1.5" />
                <polygon points="11,1.5 16,4 11,6.5" fill={opt.stroke} />
              </svg>
            </span>
            <div className="min-w-0">
              <div className={`text-xs font-medium ${opt.colorClass}`}>{opt.label}</div>
              <div className="text-[10px] text-brand-shade3 leading-snug mt-0.5">{opt.description}</div>
            </div>
          </button>
        ))}
      </div>
    </div>
  );
}
