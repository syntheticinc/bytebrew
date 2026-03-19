import { useState, useEffect } from 'react';
import { useNavigate } from 'react-router-dom';
import { api } from '../api/client';
import type { AgentDetail } from '../types';

interface AgentDetailModalProps {
  agentName: string;
  onClose: () => void;
}

const TOOL_COLORS = [
  'bg-blue-500/15 text-blue-400',
  'bg-emerald-500/15 text-emerald-400',
  'bg-purple-500/15 text-purple-400',
  'bg-amber-500/15 text-amber-400',
  'bg-cyan-500/15 text-cyan-400',
  'bg-rose-500/15 text-rose-400',
  'bg-indigo-500/15 text-indigo-400',
  'bg-teal-500/15 text-teal-400',
];

function toolColor(name: string): string {
  let hash = 0;
  for (let i = 0; i < name.length; i++) {
    hash = (hash * 31 + name.charCodeAt(i)) | 0;
  }
  return TOOL_COLORS[Math.abs(hash) % TOOL_COLORS.length] ?? TOOL_COLORS[0]!;
}

export function AgentDetailModal({ agentName, onClose }: AgentDetailModalProps) {
  const navigate = useNavigate();
  const [agent, setAgent] = useState<AgentDetail | null>(null);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState('');
  const [promptExpanded, setPromptExpanded] = useState(false);

  useEffect(() => {
    setLoading(true);
    setError('');
    api
      .getAgent(agentName)
      .then((data) => {
        setAgent(data);
        setLoading(false);
      })
      .catch((err: Error) => {
        setError(err.message);
        setLoading(false);
      });
  }, [agentName]);

  const handleChat = () => {
    navigate(`/chat?agent=${encodeURIComponent(agentName)}`);
  };

  return (
    <>
      {/* Backdrop */}
      <div className="fixed inset-0 z-40 bg-black/60" onClick={onClose} />

      {/* Modal */}
      <div className="fixed inset-0 z-50 flex items-center justify-center p-4">
        <div className="w-full max-w-xl animate-fade-in rounded-card border border-brand-shade3/15 bg-brand-dark-alt shadow-2xl max-h-[85vh] flex flex-col">
          {/* Header */}
          <div className="flex items-center justify-between border-b border-brand-shade3/15 px-6 py-4 flex-shrink-0">
            <h2 className="text-sm font-bold text-brand-light">{agentName}</h2>
            <button
              onClick={onClose}
              className="text-brand-shade3 transition-colors hover:text-brand-light"
            >
              <svg width="16" height="16" viewBox="0 0 16 16" fill="none">
                <path d="M4 4l8 8M12 4l-8 8" stroke="currentColor" strokeWidth="1.5" strokeLinecap="round" />
              </svg>
            </button>
          </div>

          {/* Content */}
          <div className="flex-1 overflow-y-auto px-6 py-5">
            {loading ? (
              <div className="text-sm text-brand-shade3">Loading...</div>
            ) : error ? (
              <div className="rounded-btn border border-red-500/30 bg-red-500/10 px-4 py-2 text-sm text-red-400">
                {error}
              </div>
            ) : agent ? (
              <div className="space-y-5">
                {/* Description */}
                {agent.description && (
                  <div>
                    <label className="mb-1 block text-[10px] font-medium uppercase tracking-wider text-brand-shade3">
                      Description
                    </label>
                    <p className="text-sm text-brand-shade2">{agent.description}</p>
                  </div>
                )}

                {/* Kit & Lifecycle */}
                <div className="grid grid-cols-2 gap-4">
                  {agent.kit && (
                    <div>
                      <label className="mb-1 block text-[10px] font-medium uppercase tracking-wider text-brand-shade3">
                        Kit
                      </label>
                      <span className="inline-flex items-center rounded-full bg-brand-accent/10 px-2.5 py-0.5 text-xs text-brand-accent">
                        {agent.kit}
                      </span>
                    </div>
                  )}
                  <div>
                    <label className="mb-1 block text-[10px] font-medium uppercase tracking-wider text-brand-shade3">
                      Lifecycle
                    </label>
                    <p className="text-sm text-brand-shade2">{agent.lifecycle || '-'}</p>
                  </div>
                </div>

                {/* Config row */}
                <div className="grid grid-cols-3 gap-4">
                  <div>
                    <label className="mb-1 block text-[10px] font-medium uppercase tracking-wider text-brand-shade3">
                      Max Steps
                    </label>
                    <p className="text-sm text-brand-shade2">{agent.max_steps || '-'}</p>
                  </div>
                  <div>
                    <label className="mb-1 block text-[10px] font-medium uppercase tracking-wider text-brand-shade3">
                      Tool Execution
                    </label>
                    <p className="text-sm text-brand-shade2">{agent.tool_execution || '-'}</p>
                  </div>
                  <div>
                    <label className="mb-1 block text-[10px] font-medium uppercase tracking-wider text-brand-shade3">
                      Max Context
                    </label>
                    <p className="text-sm text-brand-shade2">
                      {agent.max_context_size ? agent.max_context_size.toLocaleString() : '-'}
                    </p>
                  </div>
                </div>

                {/* Tools */}
                {agent.tools.length > 0 && (
                  <div>
                    <label className="mb-2 block text-[10px] font-medium uppercase tracking-wider text-brand-shade3">
                      Tools ({agent.tools.length})
                    </label>
                    <div className="flex flex-wrap gap-1.5">
                      {agent.tools.map((tool) => (
                        <span
                          key={tool}
                          className={`inline-flex items-center rounded-full px-2.5 py-0.5 text-[11px] font-medium ${toolColor(tool)}`}
                        >
                          {tool}
                        </span>
                      ))}
                    </div>
                  </div>
                )}

                {/* Can Spawn */}
                {agent.can_spawn.length > 0 && (
                  <div>
                    <label className="mb-2 block text-[10px] font-medium uppercase tracking-wider text-brand-shade3">
                      Can Spawn
                    </label>
                    <div className="flex flex-wrap gap-1.5">
                      {agent.can_spawn.map((name) => (
                        <span
                          key={name}
                          className="inline-flex items-center rounded-full bg-brand-shade3/10 px-2.5 py-0.5 text-[11px] font-medium text-brand-shade2"
                        >
                          {name}
                        </span>
                      ))}
                    </div>
                  </div>
                )}

                {/* System Prompt (collapsible) */}
                {agent.system_prompt && (
                  <div>
                    <button
                      onClick={() => setPromptExpanded(!promptExpanded)}
                      className="mb-2 flex items-center gap-1.5 text-[10px] font-medium uppercase tracking-wider text-brand-shade3 transition-colors hover:text-brand-light"
                    >
                      <svg
                        width="10"
                        height="10"
                        viewBox="0 0 10 10"
                        fill="none"
                        className={`transition-transform ${promptExpanded ? 'rotate-90' : ''}`}
                      >
                        <path d="M3 1l4 4-4 4" stroke="currentColor" strokeWidth="1.5" strokeLinecap="round" strokeLinejoin="round" />
                      </svg>
                      System Prompt
                    </button>
                    {promptExpanded && (
                      <pre className="max-h-64 overflow-auto rounded-btn border border-brand-shade3/15 bg-brand-dark p-3 font-mono text-xs leading-relaxed text-brand-shade1">
                        {agent.system_prompt}
                      </pre>
                    )}
                  </div>
                )}
              </div>
            ) : null}
          </div>

          {/* Footer */}
          <div className="flex items-center justify-end gap-3 border-t border-brand-shade3/15 px-6 py-4 flex-shrink-0">
            <button
              onClick={onClose}
              className="rounded-btn px-4 py-2 text-xs text-brand-shade3 transition-colors hover:text-brand-light"
            >
              Close
            </button>
            <button
              onClick={handleChat}
              className="rounded-btn bg-brand-accent px-4 py-2 text-xs font-medium text-white transition-colors hover:bg-brand-accent-hover"
            >
              Chat with this agent
            </button>
          </div>
        </div>
      </div>
    </>
  );
}
