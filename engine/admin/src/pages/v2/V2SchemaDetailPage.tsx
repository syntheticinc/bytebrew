import { useCallback, useEffect, useMemo, useRef, useState } from 'react';
import { Link, useNavigate, useParams, useSearchParams } from 'react-router-dom';
import V2DelegationTree from '../../components/v2/V2DelegationTree';
import V2DebugPanel from '../../components/v2/V2DebugPanel';
import {
  getSchemaById,
  getSchemaAgents,
  getSchemaRelations,
  getSchemaTriggers,
  getSchemaActiveSessions,
  v2Sessions,
  getAgentById,
  v2Agents,
  v2AgentRelations,
  type V2Session,
  type V2SessionMessage,
} from '../../mocks/v2';

type TabKey = 'canvas' | 'triggers' | 'flows' | 'settings';

function computeDebugHighlights(session: V2Session, stepIdx: number) {
  const msg: V2SessionMessage | undefined = session.messages[stepIdx];
  const agentIds = new Set<string>();
  const relationKeys = new Set<string>();
  if (!msg) return { agentIds, relationKeys };
  agentIds.add(msg.agentId);
  if (msg.kind === 'delegation' && msg.targetAgentId) {
    relationKeys.add(`${msg.agentId}->${msg.targetAgentId}`);
    agentIds.add(msg.targetAgentId);
  }
  if (msg.kind === 'delegation_return' && msg.sourceAgentId) {
    relationKeys.add(`${msg.sourceAgentId}->${msg.agentId}`);
    agentIds.add(msg.sourceAgentId);
  }
  return { agentIds, relationKeys };
}

function AddAgentPanel({
  schemaAgentIds,
  parentAgentName,
  onAdd,
  onClose,
}: {
  schemaAgentIds: string[];
  parentAgentName?: string;
  onAdd: (agentId: string) => void;
  onClose: () => void;
}) {
  const available = v2Agents.filter((a) => !schemaAgentIds.includes(a.id));
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
        {available.length === 0 && (
          <div className="px-4 py-4 text-center text-[11px] text-brand-shade3">
            All agents already added. Create a new one from Agents page.
          </div>
        )}
        {available.map((a) => (
          <button
            key={a.id}
            onClick={() => onAdd(a.id)}
            className="w-full text-left flex items-center gap-3 px-4 py-2.5 hover:bg-brand-shade3/5 border-b border-brand-shade3/5 last:border-b-0 transition-colors"
          >
            <div className="shrink-0 w-8 h-8 rounded-full bg-gradient-to-br from-brand-shade3/30 to-brand-shade3/10 flex items-center justify-center text-[11px] font-semibold text-brand-light border border-brand-shade3/20">
              {a.avatarInitials}
            </div>
            <div className="min-w-0 flex-1">
              <div className="text-[12px] text-brand-light truncate">{a.name}</div>
              <div className="text-[10px] text-brand-shade3 font-mono truncate">{a.model}</div>
            </div>
          </button>
        ))}
      </div>
      <div className="px-4 py-2 border-t border-brand-shade3/15 bg-brand-dark/50">
        <Link to="/agents" className="text-[11px] text-brand-accent hover:underline">
          + Create new agent
        </Link>
      </div>
    </div>
  );
}

function SessionsMenu({
  sessions,
  onPick,
}: {
  sessions: V2Session[];
  onPick: (sessionId: string) => void;
}) {
  const [open, setOpen] = useState(false);
  const ref = useRef<HTMLDivElement>(null);
  useEffect(() => {
    if (!open) return;
    const onClick = (e: MouseEvent) => {
      if (ref.current && !ref.current.contains(e.target as Node)) setOpen(false);
    };
    document.addEventListener('mousedown', onClick);
    return () => document.removeEventListener('mousedown', onClick);
  }, [open]);
  return (
    <div ref={ref} className="relative">
      <button
        onClick={() => setOpen((v) => !v)}
        disabled={sessions.length === 0}
        className="px-3 py-1.5 text-[11px] font-medium rounded-btn border bg-brand-dark-surface text-brand-shade2 border-brand-shade3/25 hover:border-brand-shade3/50 hover:text-brand-light disabled:opacity-40 disabled:cursor-not-allowed"
      >
        Sessions ({sessions.length}) ▾
      </button>
      {open && sessions.length > 0 && (
        <div className="absolute right-0 top-full mt-1 z-30 min-w-[320px] max-h-[360px] overflow-y-auto bg-brand-dark-alt border border-brand-shade3/25 rounded-card shadow-lg">
          {sessions.map((s) => (
            <button
              key={s.id}
              onClick={() => {
                onPick(s.id);
                setOpen(false);
              }}
              className="w-full text-left flex items-center gap-2 px-3 py-2 border-b border-brand-shade3/10 last:border-b-0 hover:bg-brand-shade3/5 transition-colors"
            >
              <span className={`w-1.5 h-1.5 rounded-full shrink-0 ${
                s.status === 'active' ? 'bg-emerald-400 animate-pulse' : 'bg-brand-shade3/50'
              }`} />
              <span className="font-mono text-[10px] text-brand-shade3 shrink-0">{s.id}</span>
              <span className="text-[11px] text-brand-light truncate flex-1">{s.title}</span>
              <span className="text-[10px] text-purple-300 shrink-0">Debug →</span>
            </button>
          ))}
        </div>
      )}
    </div>
  );
}

export default function V2SchemaDetailPage() {
  const { schemaId = '' } = useParams();
  const navigate = useNavigate();
  const [searchParams, setSearchParams] = useSearchParams();
  const validTabs: TabKey[] = ['canvas', 'triggers', 'flows', 'settings'];
  const rawTab = searchParams.get('tab') as TabKey | null;
  const [tab, setTab] = useState<TabKey>(
    rawTab && validTabs.includes(rawTab) ? rawTab : 'canvas',
  );
  const [debugOpen, setDebugOpen] = useState<boolean>(!!searchParams.get('session'));
  const [activeDebugSessionId, setActiveDebugSessionId] = useState<string | null>(
    searchParams.get('session'),
  );
  const [showAddAgent, setShowAddAgent] = useState(false);
  const [addChildParentId, setAddChildParentId] = useState<string | null>(null);
  const [mutationVersion, setMutationVersion] = useState(0); // bumped on prototype-mock mutations to force re-render
  const [debugHighlight, setDebugHighlight] = useState<{
    agentIds: Set<string>;
    relationKeys: Set<string>;
  }>({ agentIds: new Set(), relationKeys: new Set() });

  const schema = getSchemaById(schemaId);
  const agents = useMemo(() => getSchemaAgents(schemaId), [schemaId, mutationVersion]);
  const relations = useMemo(() => getSchemaRelations(schemaId), [schemaId, mutationVersion]);
  const triggers = useMemo(() => getSchemaTriggers(schemaId), [schemaId, mutationVersion]);
  const _activeSessions = useMemo(() => getSchemaActiveSessions(schemaId), [schemaId]);
  void _activeSessions;
  const allSchemaSessions = useMemo(
    () => v2Sessions.filter((s) => s.schemaId === schemaId),
    [schemaId],
  );

  const debugSessions = useMemo(() => {
    if (!activeDebugSessionId) return allSchemaSessions;
    const chosen = allSchemaSessions.find((s) => s.id === activeDebugSessionId);
    return chosen ? [chosen, ...allSchemaSessions.filter((s) => s.id !== activeDebugSessionId)] : allSchemaSessions;
  }, [allSchemaSessions, activeDebugSessionId]);

  const onAgentOpen = useCallback(
    (agentId: string) => navigate(`/agents/${agentId}`),
    [navigate],
  );

  const onAddChildRequest = useCallback((parentAgentId: string) => {
    setAddChildParentId(parentAgentId);
    setShowAddAgent(true);
  }, []);

  const handleAddAgent = useCallback(
    (agentId: string) => {
      if (!schema) return;
      // Mutate prototype mocks in place (safe: isPrototype guards real API).
      const parentId = addChildParentId ?? schema.entryAgentId;
      if (!schema.agentIds.includes(agentId)) {
        schema.agentIds.push(agentId);
      }
      const relExists = v2AgentRelations.some(
        (r) => r.sourceAgentId === parentId && r.targetAgentId === agentId,
      );
      if (!relExists) {
        v2AgentRelations.push({
          id: `rel-${parentId}-${agentId}-${Date.now()}`,
          sourceAgentId: parentId,
          targetAgentId: agentId,
        });
      }
      setMutationVersion((v) => v + 1);
      setShowAddAgent(false);
      setAddChildParentId(null);
    },
    [schema, addChildParentId],
  );

  const handleRemoveDelegation = useCallback(
    (agentId: string) => {
      if (!schema) return;
      if (agentId === schema.entryAgentId) return; // entry cannot be removed
      const currentRelations = v2AgentRelations.filter(
        (r) =>
          schema.agentIds.includes(r.sourceAgentId) &&
          schema.agentIds.includes(r.targetAgentId),
      );
      // Collect subtree (this agent + all descendants reachable via delegation).
      const subtreeIds = new Set<string>([agentId]);
      const queue = [agentId];
      while (queue.length > 0) {
        const current = queue.shift()!;
        for (const r of currentRelations) {
          if (r.sourceAgentId === current && !subtreeIds.has(r.targetAgentId)) {
            subtreeIds.add(r.targetAgentId);
            queue.push(r.targetAgentId);
          }
        }
      }
      // Mutate: drop subtree agents from schema.agentIds; drop relations where source or target is in subtree.
      schema.agentIds = schema.agentIds.filter((id) => !subtreeIds.has(id));
      for (let i = v2AgentRelations.length - 1; i >= 0; i--) {
        const r = v2AgentRelations[i];
        if (!r) continue;
        if (subtreeIds.has(r.sourceAgentId) || subtreeIds.has(r.targetAgentId)) {
          v2AgentRelations.splice(i, 1);
        }
      }
      setMutationVersion((v) => v + 1);
    },
    [schema],
  );

  const parentAgentName = useMemo(() => {
    if (!addChildParentId) return undefined;
    return getAgentById(addChildParentId)?.name;
  }, [addChildParentId]);

  const onStepChange = useCallback((session: V2Session, stepIdx: number) => {
    setDebugHighlight(computeDebugHighlights(session, stepIdx));
  }, []);

  const openDebugForSession = useCallback((sid: string) => {
    setActiveDebugSessionId(sid);
    setDebugOpen(true);
    setTab('canvas');
    const sp = new URLSearchParams(searchParams);
    sp.set('session', sid);
    setSearchParams(sp, { replace: true });
  }, [searchParams, setSearchParams]);

  if (!schema) {
    return (
      <div className="max-w-[800px] mx-auto text-center py-12">
        <p className="text-brand-shade3">Schema not found.</p>
        <Link to="/v2/schemas" className="text-brand-accent text-sm mt-4 inline-block">
          ← Back to schemas
        </Link>
      </div>
    );
  }

  const canvasEmpty = agents.length === 0;

  return (
    <div className="h-full flex flex-col">
      {/* Breadcrumb + title */}
      <div className="px-6 pt-4 pb-3 border-b border-brand-shade3/10">
        <Link to="/v2/schemas" className="text-[11px] text-brand-shade3 hover:text-brand-accent transition-colors">
          ← Schemas
        </Link>
        <div className="flex items-center gap-3 mt-2">
          <h1 className="text-xl font-semibold text-brand-light">{schema.name}</h1>
          {schema.activeSessions > 0 && (
            <span className="flex items-center gap-1.5 text-[11px] text-emerald-400 border border-emerald-500/30 bg-emerald-500/10 px-2 py-0.5 rounded">
              <span className="w-1.5 h-1.5 rounded-full bg-emerald-400 animate-pulse" />
              {schema.activeSessions} active
            </span>
          )}
          <div className="flex-1" />
          <SessionsMenu
            sessions={allSchemaSessions}
            onPick={openDebugForSession}
          />
          <button
            onClick={() => setDebugOpen((d) => !d)}
            disabled={!activeDebugSessionId}
            title={activeDebugSessionId ? '' : 'Pick a past session to inspect'}
            className={`px-3 py-1.5 text-[11px] font-medium rounded-btn border transition-colors disabled:opacity-40 disabled:cursor-not-allowed ${
              debugOpen
                ? 'bg-purple-500/20 text-purple-200 border-purple-500/40'
                : 'bg-brand-dark-surface text-brand-shade2 border-brand-shade3/25 hover:border-brand-shade3/50 hover:text-brand-light'
            }`}
          >
            {debugOpen ? '⏹ Close Debug' : 'Debug'}
          </button>
        </div>

        {/* Tabs */}
        <div className="flex items-center gap-1 mt-3">
          {([
            ['canvas', 'Canvas'],
            ['triggers', `Triggers (${triggers.length})`],
            ['flows', 'Flows'],
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
            {canvasEmpty ? (
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
                  triggers={triggers}
                  agents={agents}
                  relations={relations}
                  entryAgentId={schema.entryAgentId}
                  highlightAgentIds={debugHighlight.agentIds}
                  onAgentOpen={onAgentOpen}
                  onAddChild={onAddChildRequest}
                  onRemoveDelegation={handleRemoveDelegation}
                />

                {/* Canvas toolbar overlay */}
                <div className="absolute top-4 right-4 flex items-center gap-2 z-10">
                  <button
                    onClick={() => {
                      setAddChildParentId(schema.entryAgentId);
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
                    schemaAgentIds={schema.agentIds}
                    parentAgentName={parentAgentName}
                    onAdd={handleAddAgent}
                    onClose={() => {
                      setShowAddAgent(false);
                      setAddChildParentId(null);
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
            {triggers.map((t) => (
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
                    {t.type === 'webhook' && typeof t.config.webhookPath === 'string' && t.config.webhookPath}
                    {t.type === 'cron' && typeof t.config.schedule === 'string' && `schedule: ${t.config.schedule}`}
                    {t.type === 'chat' && 'POST /api/v1/triggers/' + t.id + '/chat'}
                  </div>
                </div>
                <div className="text-[10px] text-brand-shade3">
                  → {getAgentById(t.agentId)?.name ?? t.agentId}
                </div>
                <span className={`text-[10px] ${t.enabled ? 'text-emerald-400' : 'text-brand-shade3/50'}`}>
                  {t.enabled ? 'enabled' : 'disabled'}
                </span>
              </div>
            ))}
          </div>
        )}

        {tab === 'flows' && (
          <div className="p-6 max-w-[800px] mx-auto">
            <div className="bg-brand-dark-surface border border-dashed border-brand-shade3/25 rounded-card p-8 text-center">
              <h3 className="text-base font-semibold text-brand-light mb-2">
                Flows live inside agents
              </h3>
              <p className="text-[13px] text-brand-shade3 leading-relaxed max-w-md mx-auto">
                A flow is internal structured thinking for <span className="text-brand-light">one agent</span>,
                just like capabilities or tools. Open an agent to create or edit its flows.
              </p>
              <div className="flex items-center justify-center gap-2 mt-4">
                {agents.slice(0, 3).map((a) => (
                  <Link
                    key={a.id}
                    to={`/agents/${a.id}`}
                    className="px-3 py-1.5 text-[11px] text-brand-accent border border-brand-accent/40 rounded-btn hover:bg-brand-accent/10 transition-colors"
                  >
                    {a.name}
                  </Link>
                ))}
                {agents.length > 3 && (
                  <span className="text-[11px] text-brand-shade3">+{agents.length - 3} more</span>
                )}
              </div>
            </div>
          </div>
        )}

        {tab === 'settings' && (
          <div className="p-6 max-w-[600px] mx-auto space-y-4">
            <div>
              <label className="block text-[11px] uppercase tracking-wider text-brand-shade3 mb-1.5">Name</label>
              <input
                readOnly
                value={schema.name}
                className="w-full bg-brand-dark border border-brand-shade3/20 rounded-btn px-3 py-2 text-[13px] text-brand-light"
              />
            </div>
            <div>
              <label className="block text-[11px] uppercase tracking-wider text-brand-shade3 mb-1.5">
                Description
              </label>
              <textarea
                readOnly
                value={schema.description}
                rows={2}
                className="w-full bg-brand-dark border border-brand-shade3/20 rounded-btn px-3 py-2 text-[13px] text-brand-light"
              />
            </div>
            <div>
              <label className="block text-[11px] uppercase tracking-wider text-brand-shade3 mb-1.5">
                Entry Orchestrator
              </label>
              <div className="bg-brand-dark border border-brand-shade3/20 rounded-btn px-3 py-2 text-[13px] text-brand-light">
                {getAgentById(schema.entryAgentId)?.name}{' '}
                <span className="text-brand-shade3">— all triggers dispatch here</span>
              </div>
            </div>
          </div>
        )}
      </div>

      {debugOpen && debugSessions.length > 0 && (
        <V2DebugPanel
          sessions={debugSessions}
          onStepChange={onStepChange}
          onClose={() => {
            setDebugOpen(false);
            setActiveDebugSessionId(null);
            setDebugHighlight({ agentIds: new Set(), relationKeys: new Set() });
          }}
        />
      )}
    </div>
  );
}
