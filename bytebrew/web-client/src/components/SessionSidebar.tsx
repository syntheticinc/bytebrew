import { useState, useRef, useCallback, useEffect, type KeyboardEvent } from 'react';
import type { SessionResponse } from '../types';

function timeAgo(dateStr: string): string {
  const now = Date.now();
  const then = new Date(dateStr).getTime();
  const diffMs = now - then;
  const diffSec = Math.floor(diffMs / 1000);
  if (diffSec < 60) return 'just now';
  const diffMin = Math.floor(diffSec / 60);
  if (diffMin < 60) return `${diffMin}m ago`;
  const diffHr = Math.floor(diffMin / 60);
  if (diffHr < 24) return `${diffHr}h ago`;
  const diffDay = Math.floor(diffHr / 24);
  return `${diffDay}d ago`;
}

interface SessionSidebarProps {
  sessions: SessionResponse[];
  activeSessionId: string | null;
  onSelectSession: (id: string) => void;
  onNewSession: () => void;
  onRenameSession: (id: string, title: string) => void;
  onDeleteSession: (id: string) => void;
}

export function SessionSidebar({
  sessions,
  activeSessionId,
  onSelectSession,
  onNewSession,
  onRenameSession,
  onDeleteSession,
}: SessionSidebarProps) {
  const [editingId, setEditingId] = useState<string | null>(null);
  const [editValue, setEditValue] = useState('');
  const [menuOpenId, setMenuOpenId] = useState<string | null>(null);
  const [confirmDeleteId, setConfirmDeleteId] = useState<string | null>(null);
  const editInputRef = useRef<HTMLInputElement>(null);
  const menuRef = useRef<HTMLDivElement>(null);

  // Focus input when editing starts
  useEffect(() => {
    if (editingId && editInputRef.current) {
      editInputRef.current.focus();
      editInputRef.current.select();
    }
  }, [editingId]);

  // Close menu on outside click
  useEffect(() => {
    if (!menuOpenId) return;
    const handler = (e: MouseEvent) => {
      if (menuRef.current && !menuRef.current.contains(e.target as Node)) {
        setMenuOpenId(null);
      }
    };
    document.addEventListener('mousedown', handler);
    return () => document.removeEventListener('mousedown', handler);
  }, [menuOpenId]);

  const startEdit = useCallback((session: SessionResponse) => {
    setEditingId(session.id);
    setEditValue(session.title);
    setMenuOpenId(null);
  }, []);

  const commitEdit = useCallback(() => {
    if (!editingId) return;
    const trimmed = editValue.trim();
    if (trimmed && trimmed.length > 0) {
      onRenameSession(editingId, trimmed);
    }
    setEditingId(null);
  }, [editingId, editValue, onRenameSession]);

  const cancelEdit = useCallback(() => {
    setEditingId(null);
  }, []);

  const handleEditKeyDown = useCallback(
    (e: KeyboardEvent<HTMLInputElement>) => {
      if (e.key === 'Enter') {
        e.preventDefault();
        commitEdit();
      } else if (e.key === 'Escape') {
        e.preventDefault();
        cancelEdit();
      }
    },
    [commitEdit, cancelEdit],
  );

  const handleDelete = useCallback(
    (id: string) => {
      setConfirmDeleteId(null);
      setMenuOpenId(null);
      onDeleteSession(id);
    },
    [onDeleteSession],
  );

  return (
    <div className="flex flex-col">
      {/* Header */}
      <div className="flex items-center justify-between px-3 pt-3 pb-1">
        <span className="text-[10px] font-medium uppercase tracking-wider text-brand-shade3">
          Sessions
        </span>
        <button
          onClick={onNewSession}
          className="flex h-5 w-5 items-center justify-center rounded text-brand-shade3 transition-colors hover:bg-brand-shade3/15 hover:text-brand-light"
          title="New session"
        >
          <svg width="12" height="12" viewBox="0 0 12 12" fill="none">
            <path d="M6 1v10M1 6h10" stroke="currentColor" strokeWidth="1.5" strokeLinecap="round" />
          </svg>
        </button>
      </div>

      {/* Session list */}
      <div className="flex flex-col gap-0.5 px-2 py-1 overflow-y-auto">
        {sessions.length === 0 && (
          <p className="px-2 py-3 text-center text-xs text-brand-shade3/60">
            No sessions yet
          </p>
        )}
        {sessions.map((session) => {
          const isActive = session.id === activeSessionId;
          const isEditing = session.id === editingId;
          const isMenuOpen = session.id === menuOpenId;
          const isConfirmingDelete = session.id === confirmDeleteId;

          return (
            <div key={session.id} className="relative group">
              {/* Confirm delete overlay */}
              {isConfirmingDelete && (
                <div className="absolute inset-0 z-20 flex items-center justify-center rounded-btn bg-brand-dark/95 px-2">
                  <span className="mr-2 text-xs text-brand-shade2">Delete?</span>
                  <button
                    onClick={() => handleDelete(session.id)}
                    className="mr-1 rounded px-2 py-0.5 text-xs font-medium text-brand-accent hover:bg-brand-accent/15"
                  >
                    Yes
                  </button>
                  <button
                    onClick={() => setConfirmDeleteId(null)}
                    className="rounded px-2 py-0.5 text-xs text-brand-shade3 hover:bg-brand-shade3/15"
                  >
                    No
                  </button>
                </div>
              )}

              <button
                className={`
                  flex w-full flex-col rounded-btn px-3 py-2 text-left transition-colors
                  ${isActive
                    ? 'bg-brand-accent/15 text-brand-light'
                    : 'text-brand-shade2 hover:bg-brand-shade3/10 hover:text-brand-light'
                  }
                `}
                onClick={() => onSelectSession(session.id)}
                onDoubleClick={() => startEdit(session)}
              >
                {isEditing ? (
                  <input
                    ref={editInputRef}
                    className="w-full rounded bg-brand-dark-alt px-1 py-0.5 text-xs text-brand-light outline-none ring-1 ring-brand-accent/50"
                    value={editValue}
                    onChange={(e) => setEditValue(e.target.value)}
                    onKeyDown={handleEditKeyDown}
                    onBlur={commitEdit}
                    onClick={(e) => e.stopPropagation()}
                  />
                ) : (
                  <span className="truncate text-xs font-medium">
                    {session.title || 'New chat'}
                  </span>
                )}
                <span className="mt-0.5 truncate text-[10px] text-brand-shade3">
                  {session.agent_name} · {timeAgo(session.updated_at)}
                </span>
              </button>

              {/* Context menu trigger */}
              {!isEditing && (
                <button
                  className={`
                    absolute right-1.5 top-1.5 flex h-5 w-5 items-center justify-center rounded
                    text-brand-shade3 transition-all
                    ${isMenuOpen ? 'opacity-100' : 'opacity-0 group-hover:opacity-100'}
                    hover:bg-brand-shade3/15 hover:text-brand-light
                  `}
                  onClick={(e) => {
                    e.stopPropagation();
                    setMenuOpenId(isMenuOpen ? null : session.id);
                  }}
                >
                  <svg width="12" height="12" viewBox="0 0 12 12" fill="currentColor">
                    <circle cx="6" cy="2" r="1" />
                    <circle cx="6" cy="6" r="1" />
                    <circle cx="6" cy="10" r="1" />
                  </svg>
                </button>
              )}

              {/* Dropdown menu */}
              {isMenuOpen && (
                <div
                  ref={menuRef}
                  className="absolute right-0 top-8 z-30 w-28 rounded-btn border border-brand-shade3/15 bg-brand-dark-alt py-1 shadow-lg"
                >
                  <button
                    className="flex w-full items-center px-3 py-1.5 text-xs text-brand-shade2 hover:bg-brand-shade3/10 hover:text-brand-light"
                    onClick={(e) => {
                      e.stopPropagation();
                      startEdit(session);
                    }}
                  >
                    Rename
                  </button>
                  <button
                    className="flex w-full items-center px-3 py-1.5 text-xs text-brand-accent hover:bg-brand-accent/10"
                    onClick={(e) => {
                      e.stopPropagation();
                      setMenuOpenId(null);
                      setConfirmDeleteId(session.id);
                    }}
                  >
                    Delete
                  </button>
                </div>
              )}
            </div>
          );
        })}
      </div>
    </div>
  );
}
