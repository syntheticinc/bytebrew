import { useState } from 'react';
import { Link } from 'react-router-dom';
import { v2Schemas, v2SchemaTemplates, getAgentById } from '../../mocks/v2';

function formatRelativeTime(iso: string) {
  const diff = Date.now() - new Date(iso).getTime();
  if (diff < 60_000) return 'just now';
  if (diff < 3_600_000) return `${Math.floor(diff / 60_000)}m ago`;
  if (diff < 86_400_000) return `${Math.floor(diff / 3_600_000)}h ago`;
  return `${Math.floor(diff / 86_400_000)}d ago`;
}

function TemplatePicker({ onClose }: { onClose: () => void }) {
  return (
    <div className="fixed inset-0 z-40 bg-black/60 flex items-center justify-center p-6">
      <div className="bg-brand-dark-surface border border-brand-shade3/25 rounded-card max-w-[720px] w-full max-h-[85vh] overflow-hidden shadow-2xl">
        <div className="px-5 py-4 border-b border-brand-shade3/15 flex items-center justify-between">
          <h2 className="text-[15px] font-semibold text-brand-light">Create Schema</h2>
          <button onClick={onClose} className="text-brand-shade3 hover:text-brand-light">✕</button>
        </div>
        <div className="px-5 py-3 text-[12px] text-brand-shade3">
          Start blank or pick a template. Templates scaffold entry orchestrator + typical delegates.
        </div>
        <div className="p-5 grid grid-cols-2 gap-3 overflow-y-auto">
          {v2SchemaTemplates.map((tpl) => (
            <button
              key={tpl.id}
              disabled
              className="text-left bg-brand-dark border border-brand-shade3/20 rounded-card p-4 hover:border-brand-accent/40 transition-colors cursor-not-allowed opacity-80"
              title="Prototype — not wired"
            >
              <div className="flex items-center gap-2 mb-2">
                <span className="text-[13px] font-semibold text-brand-light">{tpl.name}</span>
                {tpl.agentCount === 0 && (
                  <span className="text-[9px] uppercase tracking-wider px-1.5 py-0.5 rounded bg-brand-shade3/15 text-brand-shade3">
                    blank
                  </span>
                )}
              </div>
              <p className="text-[11px] text-brand-shade3 leading-relaxed">{tpl.description}</p>
              {tpl.agentCount > 0 && (
                <div className="mt-3 flex items-center gap-2 text-[10px] text-brand-shade3">
                  <span>{tpl.agentCount} agents</span>
                  <span>·</span>
                  <span>{tpl.triggerTypes.join(', ')}</span>
                </div>
              )}
            </button>
          ))}
        </div>
      </div>
    </div>
  );
}

export default function V2SchemasPage() {
  const [picking, setPicking] = useState(false);

  return (
    <div className="max-w-[1200px] mx-auto">
      <div className="flex items-center justify-between mb-6">
        <div>
          <h1 className="text-2xl font-semibold text-brand-light">Schemas</h1>
          <p className="text-sm text-brand-shade3 mt-1">
            Each schema has one entry orchestrator and its delegation tree.
          </p>
        </div>
        <button
          onClick={() => setPicking(true)}
          className="px-4 py-2 text-[12px] font-medium bg-brand-accent text-white rounded-btn hover:bg-brand-accent/90 transition-colors"
        >
          + New Schema
        </button>
      </div>

      {v2Schemas.length === 0 && (
        <div className="bg-brand-dark-surface border border-dashed border-brand-shade3/25 rounded-card p-10 text-center">
          <h3 className="text-base font-semibold text-brand-light mb-2">No schemas yet</h3>
          <p className="text-[13px] text-brand-shade3 max-w-md mx-auto mb-4">
            A schema binds triggers to an entry orchestrator and its delegates. Pick a template or start blank.
          </p>
          <button
            onClick={() => setPicking(true)}
            className="px-4 py-2 text-[12px] text-white bg-brand-accent rounded-btn"
          >
            Create first schema
          </button>
        </div>
      )}

      <div className="grid grid-cols-2 gap-4">
        {v2Schemas.map((s) => {
          const entry = getAgentById(s.entryAgentId);
          return (
            <Link
              key={s.id}
              to={`/v2/schemas/${s.id}`}
              className="block bg-brand-dark-surface border border-brand-shade3/15 rounded-card hover:border-brand-shade3/35 transition-all group"
            >
              <div className="px-5 py-4 border-b border-brand-shade3/10">
                <div className="flex items-start justify-between gap-3">
                  <div className="min-w-0 flex-1">
                    <div className="text-base font-semibold text-brand-light truncate">{s.name}</div>
                    <div className="text-[12px] text-brand-shade3 mt-1 line-clamp-2">{s.description}</div>
                  </div>
                  {s.activeSessions > 0 && (
                    <span className="flex items-center gap-1.5 text-[11px] text-emerald-400 shrink-0">
                      <span className="w-1.5 h-1.5 rounded-full bg-emerald-400 animate-pulse" />
                      {s.activeSessions} active
                    </span>
                  )}
                </div>
              </div>

              <div className="px-5 py-3 flex items-center gap-4">
                <div className="flex items-center gap-2 min-w-0">
                  <span className="shrink-0 w-7 h-7 rounded-full bg-gradient-to-br from-brand-shade3/30 to-brand-shade3/10 flex items-center justify-center text-[10px] font-semibold text-brand-light border border-brand-shade3/20">
                    {entry?.avatarInitials ?? '??'}
                  </span>
                  <div className="min-w-0">
                    <div className="text-[10px] uppercase tracking-wider text-brand-shade3">Entry</div>
                    <div className="text-[12px] font-medium text-brand-light truncate">
                      {entry?.name ?? 'unknown'}
                    </div>
                  </div>
                </div>
                <div className="flex-1" />
                <div className="flex items-center gap-4 text-[11px] text-brand-shade3">
                  <span>
                    <span className="text-brand-light font-medium">{s.agentIds.length}</span> agents
                  </span>
                  <span>
                    <span className="text-brand-light font-medium">{s.triggerIds.length}</span> triggers
                  </span>
                  <span>
                    <span className="text-brand-light font-medium">{s.sessionsToday}</span> today
                  </span>
                </div>
              </div>

              <div className="px-5 py-2 border-t border-brand-shade3/10 flex items-center justify-between">
                <span className="text-[10px] text-brand-shade3">
                  Last activity {formatRelativeTime(s.lastActivityAt)}
                </span>
                <span className="text-[11px] text-brand-shade3 group-hover:text-brand-accent transition-colors">
                  Open →
                </span>
              </div>
            </Link>
          );
        })}
      </div>

      {picking && <TemplatePicker onClose={() => setPicking(false)} />}
    </div>
  );
}
