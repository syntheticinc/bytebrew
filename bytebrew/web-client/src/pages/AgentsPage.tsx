import { useState, useEffect } from 'react';
import { useNavigate, Link } from 'react-router-dom';
import { api } from '../api/client';
import { useAuth } from '../hooks/useAuth';
import type { AgentInfo } from '../types';

export function AgentsPage() {
  const navigate = useNavigate();
  const { logout } = useAuth();
  const [agents, setAgents] = useState<AgentInfo[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState('');

  useEffect(() => {
    api
      .listAgents()
      .then((data) => {
        setAgents(data.filter((a) => a.name));
        setLoading(false);
      })
      .catch((err: Error) => {
        setError(err.message);
        setLoading(false);
      });
  }, []);

  return (
    <div className="min-h-screen bg-brand-dark">
      {/* Header */}
      <header className="border-b border-brand-shade3/15 px-6 py-4">
        <div className="mx-auto flex max-w-5xl items-center justify-between">
          <Link to="/chat" className="text-sm font-bold text-brand-light">
            Byte<span className="text-brand-accent">Brew</span>
          </Link>
          <nav className="flex items-center gap-4">
            <Link to="/chat" className="text-xs text-brand-shade3 hover:text-brand-light">
              Chat
            </Link>
            <Link to="/tasks" className="text-xs text-brand-shade3 hover:text-brand-light">
              Tasks
            </Link>
            <button onClick={logout} className="text-xs text-brand-shade3 hover:text-brand-light">
              Logout
            </button>
          </nav>
        </div>
      </header>

      <div className="mx-auto max-w-5xl px-6 py-8">
        <h1 className="mb-6 text-xl font-bold text-brand-light">Agents</h1>

        {error && (
          <div className="mb-4 rounded-btn border border-red-500/30 bg-red-500/10 px-4 py-2 text-sm text-red-400">
            {error}
          </div>
        )}

        {loading ? (
          <div className="grid gap-4 sm:grid-cols-2 lg:grid-cols-3">
            {[1, 2, 3].map((i) => (
              <div key={i} className="h-32 animate-pulse rounded-card bg-brand-dark-alt" />
            ))}
          </div>
        ) : (
          <div className="grid gap-4 sm:grid-cols-2 lg:grid-cols-3">
            {agents.map((agent) => (
              <button
                key={agent.name}
                onClick={() => navigate(`/chat?agent=${encodeURIComponent(agent.name)}`)}
                className="group rounded-card border border-brand-shade3/15 bg-brand-dark-alt p-5 text-left transition-all hover:border-brand-accent/30 hover:shadow-lg"
              >
                <h3 className="font-bold text-brand-light group-hover:text-brand-accent">
                  {agent.name}
                </h3>
                {agent.description && (
                  <p className="mt-1 text-xs text-brand-shade3 line-clamp-2">
                    {agent.description}
                  </p>
                )}
                <div className="mt-3 flex flex-wrap gap-2">
                  <span className="inline-flex items-center rounded-full bg-brand-shade3/10 px-2 py-0.5 text-[10px] text-brand-shade2">
                    {agent.tools_count} tools
                  </span>
                  {agent.kit && (
                    <span className="inline-flex items-center rounded-full bg-brand-accent/10 px-2 py-0.5 text-[10px] text-brand-accent">
                      {agent.kit}
                    </span>
                  )}
                  {agent.has_knowledge && (
                    <span className="inline-flex items-center rounded-full bg-blue-500/10 px-2 py-0.5 text-[10px] text-blue-400">
                      knowledge
                    </span>
                  )}
                </div>
              </button>
            ))}
          </div>
        )}
      </div>
    </div>
  );
}
