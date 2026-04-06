import { useState } from 'react';
import { useNavigate } from 'react-router-dom';
import { usePrototype } from '../hooks/usePrototype';
import { CAPABILITY_META } from '../types';
import type { CapabilityType } from '../types';

// Mock data for prototype mode
const MOCK_AGENTS = [
  { name: 'classifier', model: 'claude-haiku-3', lifecycle: 'spawn' as const, tools_count: 3, capabilities: [] as CapabilityType[], schemas: ['Support Schema'], spawns: [] as string[] },
  { name: 'support-agent', model: 'claude-sonnet-3.7', lifecycle: 'persistent' as const, tools_count: 8, capabilities: ['memory', 'knowledge'] as CapabilityType[], schemas: ['Support Schema', 'Sales Schema'], spawns: ['escalation'] },
  { name: 'escalation', model: 'claude-opus-4', lifecycle: 'spawn' as const, tools_count: 5, capabilities: ['escalation'] as CapabilityType[], schemas: ['Support Schema'], spawns: [] as string[] },
  { name: 'dev-router', model: 'claude-sonnet-3.7', lifecycle: 'spawn' as const, tools_count: 4, capabilities: [] as CapabilityType[], schemas: ['Dev Schema'], spawns: ['code-agent', 'review-agent'] },
  { name: 'code-agent', model: 'claude-opus-4', lifecycle: 'persistent' as const, tools_count: 12, capabilities: ['memory', 'knowledge', 'guardrail'] as CapabilityType[], schemas: ['Dev Schema'], spawns: [] as string[] },
  { name: 'review-agent', model: 'claude-sonnet-3.7', lifecycle: 'spawn' as const, tools_count: 6, capabilities: ['guardrail'] as CapabilityType[], schemas: ['Dev Schema'], spawns: [] as string[] },
  { name: 'lead-scorer', model: 'claude-haiku-3', lifecycle: 'spawn' as const, tools_count: 3, capabilities: ['output_schema'] as CapabilityType[], schemas: ['Sales Schema'], spawns: [] as string[] },
  { name: 'outreach-agent', model: 'claude-sonnet-3.7', lifecycle: 'persistent' as const, tools_count: 7, capabilities: ['memory', 'policies'] as CapabilityType[], schemas: ['Sales Schema'], spawns: [] as string[] },
];

export default function AgentsPage() {
  const { isPrototype } = usePrototype();
  const navigate = useNavigate();
  const [search, setSearch] = useState('');

  const agents = isPrototype ? MOCK_AGENTS : [];
  const filtered = agents.filter(a => a.name.toLowerCase().includes(search.toLowerCase()));

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

      {/* Table */}
      <div className="bg-brand-dark-surface rounded-card border border-brand-shade3/10 overflow-hidden">
        <table className="w-full text-sm">
          <thead>
            <tr className="border-b border-brand-shade3/10">
              <th className="text-left px-4 py-3 text-xs font-semibold text-brand-shade3 uppercase tracking-wider">Name</th>
              <th className="text-left px-4 py-3 text-xs font-semibold text-brand-shade3 uppercase tracking-wider">Model</th>
              <th className="text-left px-4 py-3 text-xs font-semibold text-brand-shade3 uppercase tracking-wider">Lifecycle</th>
              <th className="text-left px-4 py-3 text-xs font-semibold text-brand-shade3 uppercase tracking-wider">Tools</th>
              <th className="text-left px-4 py-3 text-xs font-semibold text-brand-shade3 uppercase tracking-wider">Capabilities</th>
              <th className="text-left px-4 py-3 text-xs font-semibold text-brand-shade3 uppercase tracking-wider">Used in Schemas</th>
            </tr>
          </thead>
          <tbody>
            {filtered.map(agent => (
              <tr
                key={agent.name}
                onClick={() => navigate(`/builder/${agent.schemas[0]}/${agent.name}`)}
                className="border-b border-brand-shade3/5 hover:bg-brand-dark-alt/50 cursor-pointer transition-colors"
              >
                <td className="px-4 py-3">
                  <span className="text-brand-light font-medium font-mono">{agent.name}</span>
                  {agent.spawns.length > 0 && (
                    <span className="ml-2 text-[10px] text-brand-shade3 bg-brand-dark px-1.5 py-0.5 rounded">
                      {agent.spawns.length} spawns
                    </span>
                  )}
                </td>
                <td className="px-4 py-3 text-brand-shade2 font-mono text-xs">{agent.model}</td>
                <td className="px-4 py-3">
                  <span className={`text-xs px-2 py-0.5 rounded-full font-mono ${
                    agent.lifecycle === 'persistent'
                      ? 'bg-green-500/10 text-green-400 border border-green-500/20'
                      : 'bg-brand-dark text-brand-shade3 border border-brand-shade3/20'
                  }`}>
                    {agent.lifecycle}
                  </span>
                </td>
                <td className="px-4 py-3 text-brand-shade2 text-xs">{agent.tools_count}</td>
                <td className="px-4 py-3">
                  <div className="flex gap-1 flex-wrap">
                    {agent.capabilities.map(cap => (
                      <span key={cap} className="text-[10px] text-brand-shade3 bg-brand-dark px-1.5 py-0.5 rounded" title={CAPABILITY_META[cap].label}>
                        {CAPABILITY_META[cap].label}
                      </span>
                    ))}
                    {agent.capabilities.length === 0 && <span className="text-xs text-brand-shade3/50">—</span>}
                  </div>
                </td>
                <td className="px-4 py-3">
                  <div className="flex gap-1 flex-wrap">
                    {agent.schemas.map(s => (
                      <span key={s} className="text-[10px] text-blue-400 bg-blue-500/10 px-1.5 py-0.5 rounded border border-blue-500/20">
                        {s}
                      </span>
                    ))}
                  </div>
                </td>
              </tr>
            ))}
          </tbody>
        </table>
        {filtered.length === 0 && (
          <div className="px-4 py-8 text-center text-brand-shade3 text-sm">
            {search ? 'No agents match your search.' : 'No agents configured.'}
          </div>
        )}
      </div>

      <p className="text-xs text-brand-shade3/50 mt-4">
        {filtered.length} agent{filtered.length !== 1 ? 's' : ''} total. Click to edit configuration.
      </p>
    </div>
  );
}
