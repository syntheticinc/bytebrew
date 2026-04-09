import type { ContextMenuState, NodeMenuState, EdgeMenuState } from '../../hooks/useCanvasInteraction';

// ─── Canvas (pane) context menu ───────────────────────────────────────────────

interface CanvasContextMenuProps {
  menu: ContextMenuState;
  onClose: () => void;
  onAddAgent: (pos: { x: number; y: number }) => void;
  onAddTrigger: (pos: { x: number; y: number }) => void;
  onAutoLayout: () => void;
}

export function CanvasContextMenu({ menu, onClose, onAddAgent, onAddTrigger, onAutoLayout }: CanvasContextMenuProps) {
  return (
    <div
      className="fixed z-50 bg-brand-dark-alt border border-brand-shade3/20 rounded-card shadow-xl py-1 min-w-[160px] animate-modal-in"
      style={{ left: menu.x, top: menu.y }}
    >
      <button
        className="w-full px-4 py-2 text-left text-xs text-brand-shade2 hover:bg-brand-accent/10 hover:text-brand-light transition-colors flex items-center gap-2"
        onClick={() => {
          const pos = { x: menu.canvasX, y: menu.canvasY };
          onClose();
          onAddAgent(pos);
        }}
      >
        <svg width="12" height="12" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round">
          <circle cx="12" cy="12" r="10" /><line x1="12" y1="8" x2="12" y2="16" /><line x1="8" y1="12" x2="16" y2="12" />
        </svg>
        Add Agent
      </button>
      <button
        className="w-full px-4 py-2 text-left text-xs text-brand-shade2 hover:bg-brand-accent/10 hover:text-brand-light transition-colors flex items-center gap-2"
        onClick={() => {
          const pos = { x: menu.canvasX, y: menu.canvasY };
          onClose();
          onAddTrigger(pos);
        }}
      >
        <svg width="12" height="12" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round">
          <circle cx="12" cy="12" r="10" /><polyline points="12 6 12 12 16 14" />
        </svg>
        Add Trigger
      </button>
      <div className="border-t border-brand-shade3/10 my-1" />
      <button
        className="w-full px-4 py-2 text-left text-xs text-brand-shade2 hover:bg-brand-accent/10 hover:text-brand-light transition-colors flex items-center gap-2"
        onClick={() => {
          onClose();
          onAutoLayout();
        }}
      >
        <svg width="12" height="12" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round">
          <rect x="3" y="3" width="7" height="7" /><rect x="14" y="3" width="7" height="7" /><rect x="14" y="14" width="7" height="7" /><rect x="3" y="14" width="7" height="7" />
        </svg>
        Auto Layout
      </button>
    </div>
  );
}

// ─── Node context menu ────────────────────────────────────────────────────────

interface NodeContextMenuProps {
  menu: NodeMenuState;
  onClose: () => void;
  onDetails: (nodeId: string) => void;
  onDelete: (nodeId: string) => void;
  addToast: (message: string, type: 'success' | 'error' | 'info' | 'warning') => void;
}

export function NodeContextMenu({ menu, onClose, onDetails, onDelete, addToast }: NodeContextMenuProps) {
  return (
    <div
      className="fixed z-50 bg-brand-dark-alt border border-brand-shade3/20 rounded-card shadow-xl py-1 min-w-[180px] animate-modal-in"
      style={{ left: menu.x, top: menu.y }}
      onMouseLeave={onClose}
    >
      {menu.nodeType === 'triggerNode' ? (
        <button
          className="w-full px-4 py-2 text-left text-xs text-brand-shade2 hover:bg-brand-accent/10 hover:text-brand-light transition-colors"
          onClick={() => {
            onClose();
            addToast('Trigger connections are managed on the Triggers page', 'info');
          }}
        >
          Manage on Triggers page
        </button>
      ) : (
        <>
          <button
            className="w-full px-4 py-2 text-left text-xs text-brand-shade2 hover:bg-brand-accent/10 hover:text-brand-light transition-colors"
            onClick={() => {
              onClose();
              onDetails(menu.nodeId);
            }}
          >
            Details
          </button>
          <button
            className="w-full px-4 py-2 text-left text-xs text-red-400 hover:bg-red-500/10 hover:text-red-300 transition-colors"
            onClick={() => {
              onClose();
              onDelete(menu.nodeId);
            }}
          >
            Delete
          </button>
        </>
      )}
    </div>
  );
}

// ─── Edge context menu ────────────────────────────────────────────────────────

interface EdgeContextMenuProps {
  menu: EdgeMenuState;
  onDeleteEdge: () => void;
}

export function EdgeContextMenu({ menu, onDeleteEdge }: EdgeContextMenuProps) {
  return (
    <div
      className="fixed z-50 bg-brand-dark-alt border border-brand-shade3/20 rounded-card shadow-xl py-1 min-w-[180px] animate-modal-in"
      style={{ left: menu.x, top: menu.y }}
    >
      <div className="px-4 py-1.5 text-[10px] text-brand-shade3 uppercase tracking-wide border-b border-brand-shade3/10">
        {menu.source} &rarr; {menu.target}
      </div>
      <button
        className="w-full px-4 py-2 text-left text-xs text-red-400 hover:bg-red-500/10 hover:text-red-300 transition-colors flex items-center gap-2"
        onClick={onDeleteEdge}
      >
        <svg width="12" height="12" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round">
          <polyline points="3 6 5 6 21 6" /><path d="M19 6v14a2 2 0 01-2 2H7a2 2 0 01-2-2V6m3 0V4a2 2 0 012-2h4a2 2 0 012 2v2" />
        </svg>
        Delete Connection
      </button>
    </div>
  );
}
