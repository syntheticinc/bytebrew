import { useEffect, useRef } from 'react';

interface DetailPanelProps {
  open: boolean;
  onClose: () => void;
  title: string;
  children: React.ReactNode;
  actions?: React.ReactNode;
  width?: string;
}

export default function DetailPanel({
  open,
  onClose,
  title,
  children,
  actions,
  width = 'w-[420px]',
}: DetailPanelProps) {
  const panelRef = useRef<HTMLDivElement>(null);

  useEffect(() => {
    function handleKeyDown(e: KeyboardEvent) {
      if (e.key === 'Escape') onClose();
    }

    if (open) {
      document.addEventListener('keydown', handleKeyDown);
    }
    return () => document.removeEventListener('keydown', handleKeyDown);
  }, [open, onClose]);

  if (!open) return null;

  return (
    <div className="fixed inset-0 z-40 flex justify-end">
      {/* Backdrop */}
      <div
        className="absolute inset-0 bg-black/50 transition-opacity"
        onClick={onClose}
      />

      {/* Panel */}
      <div
        ref={panelRef}
        className={`relative ${width} h-full bg-brand-dark-alt shadow-xl border-l border-brand-shade3/15 overflow-y-auto animate-slide-in-right`}
      >
        {/* Header */}
        <div className="sticky top-0 bg-brand-dark-alt border-b border-brand-shade3/15 px-6 py-4 flex items-center justify-between z-10">
          <h2 className="text-lg font-semibold text-brand-light truncate pr-4">{title}</h2>
          <button
            onClick={onClose}
            className="text-brand-shade3 hover:text-brand-light transition-colors p-1"
            aria-label="Close panel"
          >
            <svg className="w-5 h-5" fill="none" viewBox="0 0 24 24" stroke="currentColor" strokeWidth={2}>
              <path strokeLinecap="round" strokeLinejoin="round" d="M6 18L18 6M6 6l12 12" />
            </svg>
          </button>
        </div>

        {/* Body */}
        <div className="px-6 py-4">
          {children}
        </div>

        {/* Actions */}
        {actions && (
          <div className="sticky bottom-0 bg-brand-dark-alt border-t border-brand-shade3/15 px-6 py-4 flex gap-3">
            {actions}
          </div>
        )}
      </div>
    </div>
  );
}

interface DetailRowProps {
  label: string;
  children: React.ReactNode;
  className?: string;
}

export function DetailRow({ label, children, className = '' }: DetailRowProps) {
  return (
    <div className={`flex justify-between items-start py-2 border-b border-brand-shade3/10 last:border-0 ${className}`}>
      <span className="text-sm text-brand-shade3 shrink-0 mr-4">{label}</span>
      <span className="text-sm text-brand-light text-right">{children}</span>
    </div>
  );
}

interface DetailSectionProps {
  title: string;
  children: React.ReactNode;
  className?: string;
}

export function DetailSection({ title, children, className = '' }: DetailSectionProps) {
  return (
    <div className={`mb-5 ${className}`}>
      <h3 className="text-xs font-semibold text-brand-shade3 uppercase tracking-wider mb-2">{title}</h3>
      <div>{children}</div>
    </div>
  );
}
