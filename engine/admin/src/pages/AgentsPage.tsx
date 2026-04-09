import { useState } from 'react';
import { useNavigate } from 'react-router-dom';
import { usePrototype } from '../hooks/usePrototype';
import { useApi } from '../hooks/useApi';
import { useAdminRefresh } from '../hooks/useAdminRefresh';
import { api } from '../api/client';
import type { AgentInfo } from '../types';

function AgentRow({ agent, onClick }: { agent: AgentInfo; onClick: () => void }) {
  const schemas = (agent as any).used_in_schemas ?? (agent as any).schemas ?? [];
  const lifecycle = (agent as any).lifecycle as string | undefined;
  return (
    <tr
      onClick={onClick}
      className="border-b border-brand-shade3/5 hover:bg-brand-dark-alt/50 cursor-pointer transition-colors"
    >
      <td className="px-4 py-3">
        <span className="text-brand-light font-medium font-mono">{agent.name}</span>
        {agent.is_system && (
          <span className="ml-2 text-[10px] text-brand-accent/70 bg-brand-accent/10 px-1.5 py-0.5 rounded border border-brand-accent/20">
            System
          </span>
        )}
      </td>
      <td className="px-4 py-3 text-brand-shade2 font-mono text-xs">{(agent as any).model ?? <span className="text-brand-shade3/50">—</span>}</td>
      <td className="px-4 py-3">
        {lifecycle ? (
          <span className={`text-xs px-2 py-0.5 rounded-full font-mono ${
            lifecycle === 'persistent'
              ? 'bg-green-500/10 text-green-400 border border-green-500/20'
              : 'bg-brand-dark text-brand-shade3 border border-brand-shade3/20'
          }`}>
            {lifecycle}
          </span>
        ) : <span className="text-xs text-brand-shade3/50">—</span>}
      </td>
      <td className="px-4 py-3 text-brand-shade2 text-xs">{agent.tools_count}</td>
      <td className="px-4 py-3">
        <span className="text-xs text-brand-shade3/50">—</span>
      </td>
      <td className="px-4 py-3">
        <div className="flex gap-1 flex-wrap">
          {schemas.map((s: string) => (
            <span key={s} className="text-[10px] text-blue-400 bg-blue-500/10 px-1.5 py-0.5 rounded border border-blue-500/20">
              {s}
            </span>
          ))}
          {schemas.length === 0 && <span className="text-xs text-brand-shade3/50">—</span>}
        </div>
      </td>
    </tr>
  );
}

const TABLE_HEADERS = ['Name', 'Model', 'Lifecycle', 'Tools', 'Capabilities', 'Used in Schemas'];

export default function AgentsPage() {
  const { isPrototype } = usePrototype();
  const navigate = useNavigate();
  const [search, setSearch] = useState('');
  const [systemExpanded, setSystemExpanded] = useState(false);

  const { data: apiAgents, refetch } = useApi(() => api.listAgents(), [isPrototype]);
  useAdminRefresh(refetch);
  const agents = apiAgents ?? [];
  const allFiltered = agents.filter(a => a.name.toLowerCase().includes(search.toLowerCase()));
  const userAgents = allFiltered.filter(a => !a.is_system);
  const systemAgents = allFiltered.filter(a => a.is_system);

  function handleAgentClick(agent: AgentInfo) {
    const schemas = (agent as any).used_in_schemas ?? (agent as any).schemas ?? [];
    if (schemas[0]) {
      navigate(`/builder/${schemas[0]}/${agent.name}`);
    } else {
      navigate(`/agents/${agent.name}`);
    }
  }

  return (
    <div className="p-6 max-w-6xl">
      <div className="flex items-center justify-between mb-6">
        <div>
          <h1 className="text-xl font-semibold text-brand-light">Agents</h1>
          <p className="text-sm text-brand-shade3 mt-1">Global agent configurations. Changes affect all schemas using the agent.</p>
        </div>
        <button className="px-4 py-2 bg-brand-accent text-white text-sm font-medium rounded-btn hover:bg-brand-accent/80 transition-colors">
          + New Agent
        </button>
      </div>

      {/* Search */}
      <div className="mb-4">
        <input
          type="text"
          placeholder="Search agents..."
          value={search}
          onChange={e => setSearch(e.target.value)}
          className="w-full max-w-sm bg-brand-dark-alt border border-brand-shade3/50 rounded-card text-sm text-brand-light px-3 py-2 focus:outline-none focus:border-brand-accent placeholder-brand-shade3"
        />
      </div>

      {/* User agents table */}
      <div className="bg-brand-dark-surface rounded-card border border-brand-shade3/10 overflow-hidden">
        <table className="w-full text-sm">
          <thead>
            <tr className="border-b border-brand-shade3/10">
              {TABLE_HEADERS.map(h => (
                <th key={h} className="text-left px-4 py-3 text-xs font-semibold text-brand-shade3 uppercase tracking-wider">{h}</th>
              ))}
            </tr>
          </thead>
          <tbody>
            {userAgents.map(agent => (
              <AgentRow key={agent.name} agent={agent} onClick={() => handleAgentClick(agent)} />
            ))}
          </tbody>
        </table>
        {userAgents.length === 0 && (
          <div className="px-4 py-8 text-center text-brand-shade3 text-sm">
            {search ? 'No agents match your search.' : 'No agents configured.'}
          </div>
        )}
      </div>

      {/* System agents — collapsible */}
      {systemAgents.length > 0 && (
        <div className="mt-4">
          <button
            onClick={() => setSystemExpanded(e => !e)}
            className="flex items-center gap-2 text-xs text-brand-shade3 hover:text-brand-shade2 transition-colors mb-2"
          >
            <svg
              width="12" height="12" viewBox="0 0 14 14" fill="none"
              className={`transition-transform ${systemExpanded ? 'rotate-180' : ''}`}
            >
              <path d="M3 5L7 9L11 5" stroke="currentColor" strokeWidth="1.5" strokeLinecap="round" strokeLinejoin="round" />
            </svg>
            <span className="uppercase tracking-wider font-semibold">System Agents</span>
            <span className="text-brand-shade3/50">({systemAgents.length})</span>
          </button>

          {systemExpanded && (
            <div className="bg-brand-dark-surface rounded-card border border-brand-accent/15 overflow-hidden">
              <table className="w-full text-sm">
                <thead>
                  <tr className="border-b border-brand-shade3/10">
                    {TABLE_HEADERS.map(h => (
                      <th key={h} className="text-left px-4 py-3 text-xs font-semibold text-brand-shade3 uppercase tracking-wider">{h}</th>
                    ))}
                  </tr>
                </thead>
                <tbody>
                  {systemAgents.map(agent => (
                    <AgentRow key={agent.name} agent={agent} onClick={() => handleAgentClick(agent)} />
                  ))}
                </tbody>
              </table>
            </div>
          )}
        </div>
      )}

      <p className="text-xs text-brand-shade3/50 mt-4">
        {userAgents.length} agent{userAgents.length !== 1 ? 's' : ''} total.
        {systemAgents.length > 0 && ` ${systemAgents.length} system agent${systemAgents.length !== 1 ? 's' : ''} hidden.`}
        {' '}Click to edit configuration.
      </p>
    </div>
  );
}
