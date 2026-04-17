import { useCallback, useEffect, useMemo, useState } from 'react';
import { Link, useNavigate, useParams, useSearchParams } from 'react-router-dom';
import V2DelegationTree from '../../components/v2/V2DelegationTree';
import { api } from '../../api/client';
import { useApi } from '../../hooks/useApi';
import type { AgentDetail, Trigger } from '../../types';
import type { V2Agent, V2AgentRelation, V2Trigger } from '../../mocks/v2';

type TabKey = 'canvas' | 'triggers' | 'settings';

// ─── Type adapters ───────────────────────────────────────────────────────────
// V2DelegationTree expects V2Agent/V2AgentRelation/V2Trigger (prototype mock
// shapes). We adapt real API types to these shapes here at the boundary.

function agentDetailToV2Agent(a: AgentDetail): V2Agent {
  const initials = a.name
    .split(/[-_\s]/)
    .map((p) => p[0] ?? '')
    .join('')
    .slice(0, 2)
    .toUpperCase() || a.name.slice(0, 2).toUpperCase();
  return {
    id: a.name,
    name: a.name,
    model: a.model_id ?? '',
    description: a.description,
    avatarInitials: initials,
    lifecycle: a.lifecycle ?? 'persistent',
    toolsCount: a.tools_count ?? 0,
    knowledgeCount: 0,
    flowsCount: 0,
    activeSessions: 0,
    state: 'idle',
  };
}

function apiRelationToV2(r: { id: string; schema_id: string; source: string; target: string }): V2AgentRelation {
  return {
    id: r.id,
    sourceAgentId: r.source,
    targetAgentId: r.target,
  };
}

// triggerToV2 maps a real Trigger (from the API) to the V2Trigger shape
// that V2DelegationTree expects. V2 semantics: all triggers point at the
// schema's entry orchestrator. The server-side trigger type is a subset of
// V2TriggerType (cron | webhook | chat) — V2 UI exposes only these three.
function triggerToV2(t: Trigger, schemaId: string, entryAgentId: string): V2Trigger {
  return {
    id: t.id,
    type: t.type,
    title: t.title,
    agentId: entryAgentId,
    schemaId,
    enabled: t.enabled,
    config: (t.config ?? {}) as Record<string, unknown>,
    lastFiredAt: t.last_fired_at,
  };
}

// ─── AddAgentPanel ────────────────────────────────────────────────────────────

function AddAgentPanel({
  schemaAgentNames,
  parentAgentName,
  onAdd,
  onClose,
}: {
  schemaAgentNames: string[];
  parentAgentName?: string;
  onAdd: (agentName: string) => void;
  onClose: () => void;
}) {
  const { data: allAgents, loading } = useApi(() => api.listAgents());
  const available = (allAgents ?? []).filter((a) => !schemaAgentNames.includes(a.name));

  return (
    <div className="absolute top-4 right-4 z-20 w-[320px] bg-brand-dark-surface border border-brand-shade3/30 rounded-card shadow-2xl">
      <div className="px-4 py-3 border-b border-brand-shade3/15 flex items-center justify-between">
        <div>
          <div className="text-[12px] font-semibold text-brand-light">
            {parentAgentName ? 'Add delegate' : 'Add agent'}
          </div>
          {parentAgentName && (
            <div className="text-[10px] text-brand-shade3 mt-0.5">
              Under <span className="text-brand-accent">{parentAgentName}</span>
            </div>
          )}
        </div>
        <button onClick={onClose} className="text-brand-shade3 hover:text-brand-light text-[14px]">✕</button>
      </div>
      <div className="max-h-[340px] overflow-y-auto">
        {loading && (
          <div className="px-4 py-4 text-center text-[11px] text-brand-shade3">Loading agents…</div>
        )}
        {!loading && available.length === 0 && (
          <div className="px-4 py-4 text-center text-[11px] text-brand-shade3">
            All agents already added. Create a new one from Agents page.
          </div>
        )}
        {available.map((a) => {
          const initials = a.name
            .split(/[-_\s]/)
            .map((p) => p[0] ?? '')
            .join('')
            .slice(0, 2)
            .toUpperCase() || a.name.slice(0, 2).toUpperCase();
          return (
            <button
              key={a.name}
              onClick={() => onAdd(a.name)}
              className="w-full text-left flex items-center gap-3 px-4 py-2.5 hover:bg-brand-shade3/5 border-b border-brand-shade3/5 last:border-b-0 transition-colors"
            >
              <div className="shrink-0 w-8 h-8 rounded-full bg-gradient-to-br from-brand-shade3/30 to-brand-shade3/10 flex items-center justify-center text-[11px] font-semibold text-brand-light border border-brand-shade3/20">
                {initials}
              </div>
              <div className="min-w-0 flex-1">
                <div className="text-[12px] text-brand-light truncate">{a.name}</div>
                <div className="text-[10px] text-brand-shade3 truncate">{a.description ?? ''}</div>
              </div>
            </button>
          );
        })}
      </div>
      <div className="px-4 py-2 border-t border-brand-shade3/15 bg-brand-dark/50">
        <Link to="/agents" className="text-[11px] text-brand-accent hover:underline">
          + Create new agent
        </Link>
      </div>
    </div>
  );
}

// ─── Main page ────────────────────────────────────────────────────────────────

export default function V2SchemaDetailPage() {
  const { schemaId = '' } = useParams<{ schemaId: string }>();
  const navigate = useNavigate();
  const [searchParams] = useSearchParams();
  const validTabs: TabKey[] = ['canvas', 'triggers', 'settings'];
  const rawTab = searchParams.get('tab') as TabKey | null;
  const [tab, setTab] = useState<TabKey>(
    rawTab && validTabs.includes(rawTab) ? rawTab : 'canvas',
  );
  const [showAddAgent, setShowAddAgent] = useState(false);
  const [addChildParentName, setAddChildParentName] = useState<string | null>(null);

  // ─── Data fetching ───────────────────────────────────────────────────────────
  const { data: schema, loading: schemaLoading } = useApi(
    () => api.getSchema(schemaId),
    [schemaId],
  );

  const { data: agentNames, loading: agentNamesLoading, refetch: refetchAgentNames } = useApi(
    () => api.listSchemaAgents(schemaId),
    [schemaId],
  );

  const { data: rawRelations, loading: relationsLoading, refetch: refetchRelations } = useApi(
    () => api.listAgentRelations(schemaId),
    [schemaId],
  );

  const { data: triggers, loading: triggersLoading } = useApi(
    () => api.listTriggers(schemaId),
    [schemaId],
  );

  // Load full agent details by name
  const [agents, setAgents] = useState<AgentDetail[]>([]);
  const [agentsLoading, setAgentsLoading] = useState(false);

  useEffect(() => {
    if (!agentNames || agentNames.length === 0) {
      setAgents([]);
      return;
    }
    let cancelled = false;
    setAgentsLoading(true);
    Promise.all(agentNames.map((name) => api.getAgent(name)))
      .then((result) => {
        if (!cancelled) setAgents(result);
      })
      .catch(() => {
        if (!cancelled) setAgents([]);
      })
      .finally(() => {
        if (!cancelled) setAgentsLoading(false);
      });
    return () => { cancelled = true; };
  }, [agentNames]);

  // Adapt to V2 types for V2DelegationTree
  const v2Agents = useMemo<V2Agent[]>(
    () => agents.map(agentDetailToV2Agent),
    [agents],
  );

  const v2Relations = useMemo<V2AgentRelation[]>(
    () => (rawRelations ?? []).map(apiRelationToV2),
    [rawRelations],
  );

  // Entry agent name: from schema or first agent in list
  const entryAgentId = schema?.entry_agent_name ?? agentNames?.[0] ?? '';

  const v2Triggers = useMemo<V2Trigger[]>(
    () => (triggers ?? []).map((t) => triggerToV2(t, schemaId, entryAgentId)),
    [triggers, schemaId, entryAgentId],
  );

  const isLoading = schemaLoading || agentNamesLoading || relationsLoading || agentsLoading || triggersLoading;

  // ─── Handlers ────────────────────────────────────────────────────────────────

  const onAgentOpen = useCallback(
    (agentId: string) => navigate(`/agents/${agentId}`),
    [navigate],
  );

  const onAddChildRequest = useCallback((parentAgentId: string) => {
    setAddChildParentName(parentAgentId);
    setShowAddAgent(true);
  }, []);

  const handleAddAgent = useCallback(
    async (agentName: string) => {
      const parent = addChildParentName ?? entryAgentId;
      if (!parent) return;
      try {
        await api.createAgentRelation(schemaId, parent, agentName);
        refetchRelations();
        refetchAgentNames();
      } catch {
        // silently ignore — user sees stale state
      }
      setShowAddAgent(false);
      setAddChildParentName(null);
    },
    [schemaId, addChildParentName, entryAgentId, refetchRelations, refetchAgentNames],
  );

  const handleRemoveDelegation = useCallback(
    async (agentId: string) => {
      if (agentId === entryAgentId) return;
      // Find all relations involving this agent as target and delete them
      const toDelete = (rawRelations ?? []).filter((r) => r.target === agentId);
      try {
        await Promise.all(toDelete.map((r) => api.deleteAgentRelation(schemaId, r.id)));
        refetchRelations();
        refetchAgentNames();
      } catch {
        // silently ignore
      }
    },
    [schemaId, entryAgentId, rawRelations, refetchRelations, refetchAgentNames],
  );

  // ─── Render guards ───────────────────────────────────────────────────────────

  if (!schemaLoading && schema === null) {
    return (
      <div className="max-w-[800px] mx-auto text-center py-12">
        <p className="text-brand-shade3">Schema not found.</p>
        <Link to="/schemas" className="text-brand-accent text-sm mt-4 inline-block">
          ← Back to schemas
        </Link>
      </div>
    );
  }

  const canvasEmpty = v2Agents.length === 0 && !isLoading;
  const schemaAgentNames = agentNames ?? [];

  return (
    <div className="h-full flex flex-col">
      {/* Breadcrumb + title */}
      <div className="px-6 pt-4 pb-3 border-b border-brand-shade3/10">
        <Link to="/schemas" className="text-[11px] text-brand-shade3 hover:text-brand-accent transition-colors">
          ← Schemas
        </Link>
        <div className="flex items-center gap-3 mt-2">
          <h1 className="text-xl font-semibold text-brand-light">
            {schema?.name ?? schemaId}
          </h1>
          <div className="flex-1" />
        </div>

        {/* Tabs */}
        <div className="flex items-center gap-1 mt-3">
          {([
            ['canvas', 'Canvas'],
            ['triggers', `Triggers (${triggers?.length ?? 0})`],
            ['settings', 'Settings'],
          ] as const).map(([key, label]) => (
            <button
              key={key}
              onClick={() => setTab(key as TabKey)}
              className={`px-3 py-1.5 text-[12px] rounded-btn transition-colors ${
                tab === key
                  ? 'bg-brand-dark-alt text-brand-light border border-brand-shade3/25'
                  : 'text-brand-shade3 hover:text-brand-light border border-transparent'
              }`}
            >
              {label}
            </button>
          ))}
        </div>
      </div>

      {/* Body */}
      <div className="flex-1 min-h-0 relative">
        {tab === 'canvas' && (
          <div className="absolute inset-0">
            {isLoading ? (
              <div className="h-full flex items-center justify-center">
                <span className="text-[13px] text-brand-shade3">Loading…</span>
              </div>
            ) : canvasEmpty ? (
              <div className="h-full flex items-center justify-center p-6">
                <div className="bg-brand-dark-surface border border-dashed border-brand-shade3/25 rounded-card p-8 max-w-md text-center">
                  <h3 className="text-[14px] font-semibold text-brand-light mb-2">Empty schema</h3>
                  <p className="text-[12px] text-brand-shade3 mb-4">
                    Add your entry orchestrator, then connect it to delegates. You can also import agents from
                    the global library.
                  </p>
                  <button
                    onClick={() => setShowAddAgent(true)}
                    className="px-4 py-2 text-[12px] text-white bg-brand-accent rounded-btn"
                  >
                    + Add first agent
                  </button>
                </div>
              </div>
            ) : (
              <>
                <V2DelegationTree
                  triggers={v2Triggers}
                  agents={v2Agents}
                  relations={v2Relations}
                  entryAgentId={entryAgentId}
                  onAgentOpen={onAgentOpen}
                  onAddChild={onAddChildRequest}
                  onRemoveDelegation={handleRemoveDelegation}
                />

                {/* Canvas toolbar overlay */}
                <div className="absolute top-4 right-4 flex items-center gap-2 z-10">
                  <button
                    onClick={() => {
                      setAddChildParentName(entryAgentId);
                      setShowAddAgent(true);
                    }}
                    className="px-3 py-1.5 text-[11px] font-medium bg-brand-dark-surface/95 backdrop-blur border border-brand-shade3/25 rounded-btn text-brand-light hover:border-brand-shade3/50 transition-colors"
                    title="Add delegate under entry orchestrator"
                  >
                    + Delegate
                  </button>
                  <div
                    className="text-[10px] text-brand-shade3/70 max-w-[180px] leading-tight"
                    title="Hover any agent card to see its + button"
                  >
                    Hover a card to add a delegate under that agent
                  </div>
                </div>

                {showAddAgent && (
                  <AddAgentPanel
                    schemaAgentNames={schemaAgentNames}
                    parentAgentName={addChildParentName ?? undefined}
                    onAdd={handleAddAgent}
                    onClose={() => {
                      setShowAddAgent(false);
                      setAddChildParentName(null);
                    }}
                  />
                )}
              </>
            )}
          </div>
        )}

        {tab === 'triggers' && (
          <div className="p-6 max-w-[1000px] mx-auto space-y-2">
            <div className="flex items-center justify-between mb-3">
              <p className="text-[12px] text-brand-shade3">
                Triggers attached to this schema. Each points to the entry orchestrator.
              </p>
              <button
                disabled
                className="px-3 py-1.5 text-[11px] bg-brand-accent/15 text-brand-accent border border-brand-accent/30 rounded-btn cursor-not-allowed"
              >
                + New Trigger
              </button>
            </div>
            {triggersLoading && (
              <div className="text-[12px] text-brand-shade3">Loading triggers…</div>
            )}
            {!triggersLoading && (triggers ?? []).length === 0 && (
              <div className="text-[12px] text-brand-shade3">No triggers configured for this schema.</div>
            )}
            {(triggers ?? []).map((t) => (
              <div
                key={t.id}
                className="bg-brand-dark-surface border border-brand-shade3/15 rounded-card px-4 py-3 flex items-center gap-4"
              >
                <span className="text-[10px] uppercase tracking-wider px-2 py-0.5 rounded bg-brand-dark-alt text-brand-shade2 border border-brand-shade3/20">
                  {t.type}
                </span>
                <div className="min-w-0 flex-1">
                  <div className="text-[13px] font-medium text-brand-light truncate">{t.title}</div>
                  <div className="text-[10px] text-brand-shade3 font-mono truncate">
                    {t.type === 'webhook' && t.config?.webhook_path && t.config.webhook_path}
                    {t.type === 'cron' && t.config?.schedule && `schedule: ${t.config.schedule}`}
                    {t.type === 'chat' && 'POST /api/v1/triggers/' + t.id + '/chat'}
                  </div>
                </div>
                <span className={`text-[10px] ${t.enabled ? 'text-emerald-400' : 'text-brand-shade3/50'}`}>
                  {t.enabled ? 'enabled' : 'disabled'}
                </span>
              </div>
            ))}
          </div>
        )}

        {tab === 'settings' && (
          <div className="p-6 max-w-[600px] mx-auto space-y-4">
            <div>
              <label className="block text-[11px] uppercase tracking-wider text-brand-shade3 mb-1.5">Name</label>
              <input
                readOnly
                value={schema?.name ?? ''}
                className="w-full bg-brand-dark border border-brand-shade3/20 rounded-btn px-3 py-2 text-[13px] text-brand-light"
              />
            </div>
            <div>
              <label className="block text-[11px] uppercase tracking-wider text-brand-shade3 mb-1.5">
                Description
              </label>
              <textarea
                readOnly
                value={schema?.description ?? ''}
                rows={2}
                className="w-full bg-brand-dark border border-brand-shade3/20 rounded-btn px-3 py-2 text-[13px] text-brand-light"
              />
            </div>
            {entryAgentId && (
              <div>
                <label className="block text-[11px] uppercase tracking-wider text-brand-shade3 mb-1.5">
                  Entry Orchestrator
                </label>
                <div className="bg-brand-dark border border-brand-shade3/20 rounded-btn px-3 py-2 text-[13px] text-brand-light">
                  {entryAgentId}{' '}
                  <span className="text-brand-shade3">— all triggers dispatch here</span>
                </div>
              </div>
            )}
          </div>
        )}
      </div>
    </div>
  );
}
