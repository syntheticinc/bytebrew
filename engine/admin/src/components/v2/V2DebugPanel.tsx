import { useEffect, useMemo, useState } from 'react';
import type { V2Session, V2SessionMessage } from '../../mocks/v2';
import { getAgentById } from '../../mocks/v2';

interface V2DebugPanelProps {
  sessions: V2Session[];
  onStepChange: (session: V2Session, stepIdx: number) => void;
  onClose: () => void;
}

const SPEED_OPTIONS = [0.5, 1, 2, 5];

function formatTime(iso: string) {
  return new Date(iso).toLocaleTimeString('en-US', { hour12: false });
}

function kindBadge(kind: V2SessionMessage['kind']) {
  const map: Record<V2SessionMessage['kind'], { label: string; cls: string }> = {
    user_message: { label: 'user', cls: 'bg-blue-500/15 text-blue-300 border-blue-500/30' },
    assistant_message: { label: 'assistant', cls: 'bg-emerald-500/15 text-emerald-300 border-emerald-500/30' },
    tool_call: { label: 'tool →', cls: 'bg-amber-500/15 text-amber-300 border-amber-500/30' },
    tool_result: { label: 'tool ←', cls: 'bg-amber-500/10 text-amber-200 border-amber-500/20' },
    reasoning: { label: 'thinking', cls: 'bg-brand-shade3/15 text-brand-shade2 border-brand-shade3/30 italic' },
    delegation: { label: 'delegate →', cls: 'bg-purple-500/15 text-purple-300 border-purple-500/30' },
    delegation_return: { label: 'return ←', cls: 'bg-purple-500/10 text-purple-200 border-purple-500/20' },
  };
  return map[kind];
}

export default function V2DebugPanel({ sessions, onStepChange, onClose }: V2DebugPanelProps) {
  const [sessionId, setSessionId] = useState<string>(sessions[0]?.id ?? '');
  const session = useMemo(
    () => sessions.find((s) => s.id === sessionId) ?? sessions[0],
    [sessions, sessionId],
  );
  const [step, setStep] = useState<number>(0);
  const [playing, setPlaying] = useState<boolean>(false);
  const [speed, setSpeed] = useState<number>(1);

  // Reset when session changes
  useEffect(() => {
    setStep(0);
    setPlaying(false);
  }, [sessionId]);

  // Drive parent highlights
  useEffect(() => {
    if (session) onStepChange(session, step);
  }, [session, step, onStepChange]);

  // Playback loop
  useEffect(() => {
    if (!playing || !session) return;
    const intervalMs = 1600 / speed;
    const t = setTimeout(() => {
      if (step < session.messages.length - 1) {
        setStep(step + 1);
      } else {
        setPlaying(false);
      }
    }, intervalMs);
    return () => clearTimeout(t);
  }, [playing, step, session, speed]);

  if (!session) return null;

  const total = session.messages.length;
  const currentMsg = session.messages[step];
  const agent = currentMsg ? getAgentById(currentMsg.agentId) : null;

  const prevDisabled = step === 0;
  const nextDisabled = step >= total - 1;

  return (
    <div className="fixed left-0 right-0 bottom-0 z-40 bg-brand-dark-surface border-t border-brand-shade3/25 shadow-[0_-12px_40px_rgba(0,0,0,0.4)]">
      {/* Header strip */}
      <div className="flex items-center gap-3 px-5 py-2.5 border-b border-brand-shade3/10 bg-brand-dark-alt/40">
        <span className="text-[10px] font-bold uppercase tracking-[0.2em] text-purple-300">
          Debug Mode
        </span>

        <select
          value={session.id}
          onChange={(e) => setSessionId(e.target.value)}
          className="bg-brand-dark border border-brand-shade3/30 rounded px-2 py-1 text-[11px] text-brand-light font-mono"
        >
          {sessions.map((s) => (
            <option key={s.id} value={s.id}>
              {s.id} — {s.title}
            </option>
          ))}
        </select>

        <span className={`text-[10px] px-2 py-0.5 rounded uppercase tracking-wider ${
          session.status === 'active'
            ? 'bg-emerald-500/15 text-emerald-300 border border-emerald-500/30'
            : 'bg-brand-shade3/15 text-brand-shade3 border border-brand-shade3/30'
        }`}>
          {session.status}
        </span>

        <div className="flex-1" />

        <span className="text-[11px] text-brand-shade3 font-mono">
          step {step + 1} / {total}
        </span>

        <button
          onClick={onClose}
          className="text-brand-shade3 hover:text-brand-light transition-colors px-2"
          aria-label="Close debug"
        >
          ✕
        </button>
      </div>

      {/* Main body: controls + current message */}
      <div className="grid grid-cols-[auto_1fr_320px] gap-4 px-5 py-3">
        {/* Controls */}
        <div className="flex flex-col gap-2 min-w-[200px]">
          <div className="flex items-center gap-1">
            <button
              onClick={() => setStep(0)}
              disabled={prevDisabled}
              className="w-8 h-8 flex items-center justify-center rounded bg-brand-dark border border-brand-shade3/20 text-brand-shade2 hover:text-brand-light hover:border-brand-shade3/40 disabled:opacity-30 disabled:cursor-not-allowed transition-colors"
              title="Jump to start"
            >
              ⏮
            </button>
            <button
              onClick={() => setStep(step - 1)}
              disabled={prevDisabled}
              className="w-8 h-8 flex items-center justify-center rounded bg-brand-dark border border-brand-shade3/20 text-brand-shade2 hover:text-brand-light hover:border-brand-shade3/40 disabled:opacity-30 disabled:cursor-not-allowed transition-colors"
              title="Step back"
            >
              ◀
            </button>
            <button
              onClick={() => setPlaying((p) => !p)}
              className="w-10 h-8 flex items-center justify-center rounded bg-purple-500/20 border border-purple-400/40 text-purple-200 hover:bg-purple-500/30 transition-colors"
              title={playing ? 'Pause' : 'Play'}
            >
              {playing ? '⏸' : '▶'}
            </button>
            <button
              onClick={() => setStep(step + 1)}
              disabled={nextDisabled}
              className="w-8 h-8 flex items-center justify-center rounded bg-brand-dark border border-brand-shade3/20 text-brand-shade2 hover:text-brand-light hover:border-brand-shade3/40 disabled:opacity-30 disabled:cursor-not-allowed transition-colors"
              title="Step forward"
            >
              ▶
            </button>
            <button
              onClick={() => setStep(total - 1)}
              disabled={nextDisabled}
              className="w-8 h-8 flex items-center justify-center rounded bg-brand-dark border border-brand-shade3/20 text-brand-shade2 hover:text-brand-light hover:border-brand-shade3/40 disabled:opacity-30 disabled:cursor-not-allowed transition-colors"
              title="Jump to end"
            >
              ⏭
            </button>
          </div>

          <div className="flex items-center gap-1">
            <span
              className="text-[10px] text-brand-shade3 mr-1"
              title="Applies only when auto-playing (▶). Ignored during manual stepping."
            >
              Playback speed:
            </span>
            {SPEED_OPTIONS.map((s) => (
              <button
                key={s}
                onClick={() => setSpeed(s)}
                disabled={!playing}
                className={`px-2 py-0.5 rounded text-[10px] font-mono transition-colors ${
                  speed === s
                    ? 'bg-brand-accent/20 text-brand-accent border border-brand-accent/40'
                    : 'text-brand-shade3 hover:text-brand-light border border-transparent disabled:opacity-40 disabled:cursor-not-allowed disabled:hover:text-brand-shade3'
                }`}
                title={playing ? `${s}× playback` : 'Press ▶ to enable playback controls'}
              >
                {s}×
              </button>
            ))}
          </div>

          {/* Scrubber */}
          <input
            type="range"
            min={0}
            max={total - 1}
            value={step}
            onChange={(e) => setStep(parseInt(e.target.value, 10))}
            className="w-full accent-purple-400 cursor-pointer"
          />

          <div className="flex items-center gap-2 mt-1">
            <button
              disabled
              className="text-[10px] text-brand-shade3/50 border border-brand-shade3/15 rounded px-2 py-1 cursor-not-allowed"
              title="Inject message into running session (not yet implemented)"
            >
              💬 Inject
            </button>
            <button
              disabled
              className="text-[10px] text-brand-shade3/50 border border-brand-shade3/15 rounded px-2 py-1 cursor-not-allowed"
              title="Stop running session (not yet implemented)"
            >
              ⏹ Stop
            </button>
          </div>
        </div>

        {/* Current message */}
        {currentMsg && (
          <div className="min-w-0 bg-brand-dark rounded p-3 border border-brand-shade3/15 overflow-hidden">
            <div className="flex items-center gap-2 mb-2 flex-wrap">
              <span
                className={`text-[9px] uppercase tracking-wider border rounded px-1.5 py-0.5 ${
                  kindBadge(currentMsg.kind).cls
                }`}
              >
                {kindBadge(currentMsg.kind).label}
              </span>
              {agent && (
                <>
                  <span className="text-[11px] font-semibold text-brand-light">{agent.name}</span>
                  <span className="text-[10px] text-brand-shade3 font-mono">{agent.model}</span>
                </>
              )}
              <span className="text-[10px] text-brand-shade3 font-mono ml-auto">
                {formatTime(currentMsg.timestamp)}
              </span>
            </div>
            <div className="text-[12px] text-brand-light leading-relaxed whitespace-pre-wrap break-words">
              {currentMsg.content}
            </div>
            {currentMsg.toolArgs && (
              <pre className="mt-2 text-[10px] text-amber-200/80 bg-amber-500/5 border border-amber-500/20 rounded px-2 py-1 overflow-x-auto font-mono">
                args: {currentMsg.toolArgs}
              </pre>
            )}
            {currentMsg.toolResult && (
              <pre className="mt-2 text-[10px] text-amber-200/70 bg-amber-500/5 border border-amber-500/15 rounded px-2 py-1 overflow-x-auto font-mono">
                result: {currentMsg.toolResult}
              </pre>
            )}
          </div>
        )}

        {/* Timeline (compact preview) */}
        <div className="min-w-0 overflow-y-auto max-h-[140px] bg-brand-dark rounded p-2 border border-brand-shade3/15 space-y-1">
          {session.messages.map((m, idx) => {
            const a = getAgentById(m.agentId);
            const b = kindBadge(m.kind);
            const active = idx === step;
            return (
              <button
                key={idx}
                onClick={() => setStep(idx)}
                className={`w-full text-left flex items-center gap-2 px-2 py-1 rounded text-[10px] transition-colors ${
                  active
                    ? 'bg-purple-500/20 text-purple-100'
                    : 'hover:bg-brand-shade3/10 text-brand-shade2'
                }`}
              >
                <span className="font-mono text-brand-shade3 w-5">{idx + 1}</span>
                <span className={`uppercase tracking-wider ${b.cls} border rounded px-1 py-0 text-[8px]`}>
                  {b.label}
                </span>
                <span className="truncate flex-1">{a?.name ?? m.agentId}</span>
              </button>
            );
          })}
        </div>
      </div>
    </div>
  );
}
