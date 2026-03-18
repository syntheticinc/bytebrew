import type { AgentInfo } from '../types';

interface AgentSelectorProps {
  agents: AgentInfo[];
  selected: string | null;
  onSelect: (name: string) => void;
  loading?: boolean;
}

export function AgentSelector({ agents, selected, onSelect, loading }: AgentSelectorProps) {
  if (loading) {
    return (
      <div className="flex flex-col gap-2 p-3">
        {[1, 2, 3].map((i) => (
          <div key={i} className="h-10 animate-pulse rounded-btn bg-brand-shade3/10" />
        ))}
      </div>
    );
  }

  if (agents.length === 0) {
    return (
      <div className="p-4 text-center text-sm text-brand-shade3">
        No agents available
      </div>
    );
  }

  return (
    <div className="flex flex-col gap-1 p-2">
      {agents.map((agent) => (
        <button
          key={agent.name}
          className={`
            flex items-center gap-2 rounded-btn px-3 py-2 text-left text-sm transition-colors
            ${
              selected === agent.name
                ? 'bg-brand-accent/15 text-brand-accent'
                : 'text-brand-shade2 hover:bg-brand-shade3/10 hover:text-brand-light'
            }
          `}
          onClick={() => onSelect(agent.name)}
        >
          <span
            className={`h-2 w-2 rounded-full flex-shrink-0 ${
              selected === agent.name ? 'bg-brand-accent' : 'bg-brand-shade3/40'
            }`}
          />
          <span className="truncate font-medium">{agent.name}</span>
        </button>
      ))}
    </div>
  );
}
