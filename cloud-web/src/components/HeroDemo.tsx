import { useState, useEffect, useRef, useCallback, useMemo } from 'react';

/* ------------------------------------------------------------------ */
/*  Types                                                              */
/* ------------------------------------------------------------------ */

interface DemoStep {
  type:
    | 'input_typing'
    | 'input_send'
    | 'thinking'
    | 'tool_call'
    | 'tool_result'
    | 'text'
    | 'spawn'
    | 'sub_tool'
    | 'sub_result'
    | 'spawn_done'
    | 'response'
    | 'ask_buttons'
    | 'button_click';
  content?: string;
  tool?: string;
  options?: string[];
  delay: number;
}

/* ------------------------------------------------------------------ */
/*  Scenario                                                           */
/* ------------------------------------------------------------------ */

const SCENARIO: DemoStep[] = [
  // === Turn 1: User asks a specific question ===
  { type: 'input_typing', content: 'Why did enterprise churn spike in March?', delay: 500 },
  { type: 'input_send', delay: 700 },
  { type: 'thinking', delay: 2200 },

  // Quick data pulls
  { type: 'tool_call', tool: 'get_churn_data', content: 'enterprise, Q1', delay: 500 },
  { type: 'tool_result', tool: 'get_churn_data', content: '14 accounts churned, +340% vs Q4', delay: 1400 },
  { type: 'tool_call', tool: 'get_onboarding_nps', content: 'cohort: post-Feb', delay: 400 },
  { type: 'tool_result', tool: 'get_onboarding_nps', content: 'NPS 31 (was 72), tickets 3x avg', delay: 1200 },

  // Agent pauses — notices a pattern
  { type: 'thinking', delay: 2400 },
  { type: 'text', content: 'Hmm. All 14 accounts joined after the Feb pricing change. Let me check what changed...', delay: 1200 },

  // Spawns sub-agent — with realistic timing
  { type: 'spawn', content: 'research-agent: diff pricing tiers Feb vs Jan', delay: 1800 },
  { type: 'thinking', delay: 1600 },
  { type: 'sub_tool', tool: 'diff_pricing', content: 'enterprise tier, Feb vs Jan', delay: 1200 },
  { type: 'sub_result', content: 'Removed: onboarding calls, CSM assignment', delay: 1800 },
  { type: 'thinking', delay: 1200 },
  { type: 'sub_tool', tool: 'check_support_tickets', content: 'cohort: post-Feb enterprise', delay: 1000 },
  { type: 'sub_result', content: 'Avg 8.3 tickets/account vs 2.1 baseline', delay: 1400 },
  { type: 'spawn_done', content: 'research-agent: 2 findings, high confidence', delay: 1000 },

  // Composing the answer
  { type: 'thinking', delay: 1800 },
  {
    type: 'response',
    content:
      '**Root cause:** Feb pricing removed onboarding calls. Enterprise buyers felt abandoned.\n\n' +
      '**Impact:** ~$420K churn risk. **Fix:** $15K/q. ROI 28:1.',
    delay: 2000,
  },

  // Options — user picks second
  {
    type: 'ask_buttons',
    content: 'Next step?',
    options: ['Draft rescue plan', 'Alert CS team', 'Deep dive'],
    delay: 2000,
  },
  { type: 'button_click', content: 'Alert CS team', delay: 1000 },

  // Action after button click
  { type: 'tool_call', tool: 'notify_cs_team', content: '#cs-enterprise, priority: high', delay: 400 },
  { type: 'tool_result', tool: 'notify_cs_team', content: 'Sent to 4 CSMs', delay: 1200 },
  { type: 'text', content: 'CS team alerted. Anything else?', delay: 1800 },

  // === Turn 2: User follows up ===
  { type: 'input_typing', content: 'Draft a rescue plan for those 14 accounts', delay: 500 },
  { type: 'input_send', delay: 600 },
  { type: 'thinking', delay: 1800 },

  { type: 'tool_call', tool: 'draft_rescue_plan', content: '14 accounts, white-glove', delay: 500 },
  { type: 'tool_result', tool: 'draft_rescue_plan', content: '14 personalized emails ready', delay: 1600 },

  { type: 'text', content: 'Rescue plan drafted. Ready for your review.', delay: 4000 },
];

const TYPEWRITER_TYPES = new Set<DemoStep['type']>(['input_typing', 'text', 'response']);
const USER_CHAR_MS = 28;
const AGENT_CHAR_MS = 12;

/* ------------------------------------------------------------------ */
/*  Coffee spinner — rotating messages                                 */
/* ------------------------------------------------------------------ */

const BREW_PHRASES = [
  'Grinding beans...',
  'Brewing...',
  'Pulling a shot...',
  'Steaming...',
  'Almost ready...',
];

let brewCounter = 0;

function BrewingSpinner() {
  const phrase = BREW_PHRASES[brewCounter++ % BREW_PHRASES.length];
  return (
    <div className="flex items-center gap-2 py-2">
      <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="1.5" className="w-4 h-4" style={{ color: '#87867F' }}>
        <path d="M17 8h1a4 4 0 010 8h-1" strokeLinecap="round" />
        <path d="M3 8h14v9a4 4 0 01-4 4H7a4 4 0 01-4-4V8z" />
        <path d="M7 2v3" strokeLinecap="round" style={{ animation: 'heroSteam 1.2s ease-in-out infinite' }} />
        <path d="M10 1v3" strokeLinecap="round" style={{ animation: 'heroSteam 1.2s ease-in-out infinite 0.3s' }} />
        <path d="M13 2v3" strokeLinecap="round" style={{ animation: 'heroSteam 1.2s ease-in-out infinite 0.6s' }} />
      </svg>
      <span className="text-xs font-mono animate-pulse" style={{ color: '#87867F' }}>{phrase}</span>
      <style>{`
        @keyframes heroSteam {
          0% { opacity: 0.3; transform: translateY(0); }
          50% { opacity: 1; transform: translateY(-3px); }
          100% { opacity: 0.3; transform: translateY(0); }
        }
      `}</style>
    </div>
  );
}

/* ------------------------------------------------------------------ */
/*  Inline SVG icons (monochrome, no emoji)                            */
/* ------------------------------------------------------------------ */

function CheckIcon({ className = '' }: { className?: string }) {
  return (
    <svg className={`inline-block w-3 h-3 mr-1 ${className}`} viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2">
      <path d="M20 6L9 17l-5-5" strokeLinecap="round" strokeLinejoin="round" />
    </svg>
  );
}

function BranchIcon({ className = '' }: { className?: string }) {
  return (
    <svg className={`inline-block w-3 h-3 mr-1 ${className}`} viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2">
      <circle cx="18" cy="18" r="3" />
      <circle cx="6" cy="6" r="3" />
      <circle cx="6" cy="18" r="3" />
      <path d="M6 9v3a3 3 0 0 0 3 3h6" />
    </svg>
  );
}

/* ------------------------------------------------------------------ */
/*  Tiny markdown renderer                                             */
/* ------------------------------------------------------------------ */

function renderMarkdown(raw: string): React.ReactNode[] {
  const lines = raw.split('\n');
  const nodes: React.ReactNode[] = [];

  for (let i = 0; i < lines.length; i++) {
    const line = lines[i];
    if (line === '') { nodes.push(<br key={`br-${i}`} />); continue; }

    const listMatch = line.match(/^(\d+)\.\s+(.+)$/);
    if (listMatch) {
      nodes.push(
        <div key={`li-${i}`} className="pl-3">
          <span className="mr-1" style={{ color: '#87867F' }}>{listMatch[1]}.</span>
          {inlineBold(listMatch[2])}
        </div>,
      );
      continue;
    }
    nodes.push(<div key={`ln-${i}`}>{inlineBold(line)}</div>);
  }
  return nodes;
}

function inlineBold(text: string): React.ReactNode {
  const parts = text.split(/\*\*(.+?)\*\*/g);
  if (parts.length === 1) return text;
  return parts.map((p, i) => (i % 2 === 1 ? <strong key={i} className="text-brand-light">{p}</strong> : p));
}

/* ------------------------------------------------------------------ */
/*  Step renderers — monochrome, minimal                               */
/* ------------------------------------------------------------------ */

const MUTED = '#87867F';
const SURFACE = 'rgba(30,30,30,0.6)';
const BORDER_TOOL = 'rgba(135,134,127,0.25)';
const BORDER_DONE = 'rgba(135,134,127,0.15)';
const SPAWN_BORDER = 'rgba(147,197,253,0.3)';
const SPAWN_BG = 'rgba(30,40,60,0.5)';
const SPAWN_TEXT = '#93c5fd';

function UserBubble({ text }: { text: string }) {
  return (
    <div className="flex justify-end">
      <div className="max-w-[80%] rounded-[2px] px-3 py-2 text-sm text-white" style={{ backgroundColor: '#D7513E' }}>
        {text}
      </div>
    </div>
  );
}

function StatusDot({ done }: { done?: boolean }) {
  return (
    <span
      className="inline-block w-1.5 h-1.5 rounded-full mr-1.5 shrink-0"
      style={{
        backgroundColor: done ? '#4ade80' : '#87867F',
        boxShadow: done ? '0 0 4px rgba(74,222,128,0.4)' : 'none',
      }}
    />
  );
}

function ToolCallBlock({ tool, content, done }: { tool: string; content: string; done?: boolean }) {
  return (
    <div
      className="rounded-[2px] border-l-2 px-3 py-1.5 text-xs font-mono flex items-start gap-0"
      style={{ borderColor: done ? BORDER_DONE : BORDER_TOOL, backgroundColor: SURFACE }}
    >
      <span className="flex items-center" style={{ color: MUTED }}>
        <StatusDot done={done} />
        {tool}
      </span>
      {!done && <span style={{ color: 'rgba(135,134,127,0.5)' }}> ({content})</span>}
      {done && <span className="ml-1" style={{ color: '#CBC9BC' }}>{content}</span>}
    </div>
  );
}

function SpawnBlock({ content }: { content: string }) {
  return (
    <div
      className="rounded-[2px] border-l-2 px-3 py-1.5 text-xs"
      style={{ borderColor: SPAWN_BORDER, backgroundColor: SPAWN_BG, color: SPAWN_TEXT }}
    >
      <BranchIcon className="text-blue-300/60" />
      {content}
    </div>
  );
}

function SpawnDoneBlock({ content }: { content: string }) {
  return (
    <div
      className="rounded-[2px] border-l-2 px-3 py-1.5 text-xs"
      style={{ borderColor: SPAWN_BORDER, backgroundColor: SPAWN_BG, color: 'rgba(147,197,253,0.6)' }}
    >
      <CheckIcon className="text-blue-300/50" />
      {content}
    </div>
  );
}

function AgentText({ text, isResponse }: { text: string; isResponse?: boolean }) {
  return (
    <div className="text-sm" style={{ color: '#DFD8D0' }}>
      {isResponse ? renderMarkdown(text) : text}
    </div>
  );
}

function AskButtons({ content, options, selected }: { content: string; options: string[]; selected?: string }) {
  return (
    <div className="space-y-2">
      <div className="text-sm" style={{ color: '#DFD8D0' }}>{content}</div>
      <div className="flex flex-wrap gap-2">
        {options.map((opt) => {
          const isSelected = selected === opt;
          const isFaded = selected && !isSelected;
          return (
            <span
              key={opt}
              className="rounded-[2px] border px-3 py-1 text-xs transition-all duration-300"
              style={{
                borderColor: isSelected ? '#D7513E' : 'rgba(135,134,127,0.25)',
                backgroundColor: isSelected ? '#D7513E' : 'transparent',
                color: isSelected ? '#fff' : isFaded ? 'rgba(135,134,127,0.3)' : '#CBC9BC',
                opacity: isFaded ? 0.4 : 1,
              }}
            >
              {opt}
            </span>
          );
        })}
      </div>
    </div>
  );
}

/* ------------------------------------------------------------------ */
/*  Main Component                                                     */
/* ------------------------------------------------------------------ */

export function HeroDemo() {
  const [visibleSteps, setVisibleSteps] = useState(0);
  const [isPaused, setIsPaused] = useState(false);
  const [typingIndex, setTypingIndex] = useState(-1);
  const [typedChars, setTypedChars] = useState(0);
  const chatRef = useRef<HTMLDivElement>(null);
  const [selectedButton, setSelectedButton] = useState<string | undefined>();
  const [inputText, setInputText] = useState('');

  const resetDemo = useCallback(() => {
    setVisibleSteps(0);
    setTypingIndex(-1);
    setTypedChars(0);
    setSelectedButton(undefined);
    setInputText('');
    brewCounter = 0;
  }, []);

  useEffect(() => {
    if (isPaused) return;

    if (visibleSteps >= SCENARIO.length) {
      const t = setTimeout(resetDemo, 3000);
      return () => clearTimeout(t);
    }

    const step = SCENARIO[visibleSteps];

    if (step.type === 'input_typing' && step.content) {
      if (typingIndex !== visibleSteps) {
        setTypingIndex(visibleSteps);
        setTypedChars(0);
        setInputText('');
        return;
      }
      if (typedChars < step.content.length) {
        const t = setTimeout(() => {
          setTypedChars((c) => c + 1);
          setInputText(step.content!.slice(0, typedChars + 1));
        }, USER_CHAR_MS);
        return () => clearTimeout(t);
      }
      const t = setTimeout(() => { setVisibleSteps((v) => v + 1); setTypingIndex(-1); }, step.delay);
      return () => clearTimeout(t);
    }

    if (step.type === 'input_send') {
      setInputText('');
      const t = setTimeout(() => setVisibleSteps((v) => v + 1), step.delay);
      return () => clearTimeout(t);
    }

    if (TYPEWRITER_TYPES.has(step.type) && step.content && step.type !== 'input_typing') {
      if (typingIndex !== visibleSteps) { setTypingIndex(visibleSteps); setTypedChars(0); return; }
      if (typedChars < step.content.length) {
        const t = setTimeout(() => setTypedChars((c) => c + 1), AGENT_CHAR_MS);
        return () => clearTimeout(t);
      }
      const t = setTimeout(() => { setVisibleSteps((v) => v + 1); setTypingIndex(-1); }, step.delay);
      return () => clearTimeout(t);
    }

    if (step.type === 'button_click') {
      setSelectedButton(step.content);
      const t = setTimeout(() => setVisibleSteps((v) => v + 1), step.delay);
      return () => clearTimeout(t);
    }

    const t = setTimeout(() => setVisibleSteps((v) => v + 1), step.delay);
    return () => clearTimeout(t);
  }, [visibleSteps, isPaused, typingIndex, typedChars, resetDemo]);

  useEffect(() => {
    chatRef.current?.scrollTo({ top: chatRef.current.scrollHeight, behavior: 'smooth' });
  }, [visibleSteps, typedChars]);

  const userMessageText = useMemo(() => {
    const s = SCENARIO.find((s) => s.type === 'input_typing');
    return s?.content ?? '';
  }, []);

  const renderedSteps = useMemo(() => {
    const elements: React.ReactNode[] = [];

    for (let i = 0; i < visibleSteps; i++) {
      const step = SCENARIO[i];
      const key = `step-${i}`;
      const text = step.content ?? '';

      switch (step.type) {
        case 'input_typing': break;
        case 'input_send': elements.push(<UserBubble key={key} text={userMessageText} />); break;
        case 'thinking': break;
        case 'tool_call': {
          // Check if next completed step is the result — if so, skip (result renders the done block)
          const nextStep = i + 1 < visibleSteps ? SCENARIO[i + 1] : null;
          if (nextStep?.type === 'tool_result') break; // result will render the done version
          elements.push(<ToolCallBlock key={key} tool={step.tool ?? ''} content={text} />);
          break;
        }
        case 'tool_result': elements.push(<ToolCallBlock key={key} tool={step.tool ?? ''} content={text} done />); break;
        case 'text': elements.push(<AgentText key={key} text={text} />); break;
        case 'spawn': elements.push(<SpawnBlock key={key} content={text} />); break;
        case 'sub_tool': {
          const nextStep = i + 1 < visibleSteps ? SCENARIO[i + 1] : null;
          if (nextStep?.type === 'sub_result') break;
          elements.push(<div key={key} className="ml-4"><ToolCallBlock tool={step.tool ?? ''} content={text} /></div>);
          break;
        }
        case 'sub_result': elements.push(<div key={key} className="ml-4"><ToolCallBlock tool={step.tool ?? ''} content={text} done /></div>); break;
        case 'spawn_done': elements.push(<SpawnDoneBlock key={key} content={text} />); break;
        case 'response': elements.push(<AgentText key={key} text={text} isResponse />); break;
        case 'ask_buttons': elements.push(<AskButtons key={key} content={text} options={step.options ?? []} selected={selectedButton} />); break;
        case 'button_click': break;
      }
    }

    // Currently typing (agent only)
    if (typingIndex >= 0 && typingIndex === visibleSteps && typingIndex < SCENARIO.length) {
      const step = SCENARIO[typingIndex];
      if (step.type !== 'input_typing') {
        const partial = (step.content ?? '').slice(0, typedChars);
        elements.push(<AgentText key={`typing-${typingIndex}`} text={partial + '\u258C'} isResponse={step.type === 'response'} />);
      }
    }

    // Active thinking spinner
    if (visibleSteps < SCENARIO.length && SCENARIO[visibleSteps].type === 'thinking' && typingIndex < 0) {
      elements.push(<BrewingSpinner key={`brew-${visibleSteps}`} />);
    }

    return elements;
  }, [visibleSteps, typingIndex, typedChars, selectedButton, userMessageText]);

  const inputDisplay = useMemo(() => {
    if (typingIndex >= 0 && typingIndex < SCENARIO.length && SCENARIO[typingIndex].type === 'input_typing') {
      return inputText + '\u258C';
    }
    return '';
  }, [typingIndex, inputText]);

  const isInputActive = inputDisplay.length > 0;

  return (
    <div
      className="relative mx-auto w-full max-w-[720px]"
      onMouseEnter={() => setIsPaused(true)}
      onMouseLeave={() => setIsPaused(false)}
    >
      <div className="rounded-[2px] border overflow-hidden" style={{ borderColor: 'rgba(135,134,127,0.12)', backgroundColor: '#1A1A1A' }}>
        {/* Header */}
        <div className="flex items-center gap-3 px-4 py-2.5 border-b" style={{ borderColor: 'rgba(135,134,127,0.08)' }}>
          <div className="flex gap-1.5">
            <span className="w-2.5 h-2.5 rounded-full" style={{ backgroundColor: 'rgba(135,134,127,0.3)' }} />
            <span className="w-2.5 h-2.5 rounded-full" style={{ backgroundColor: 'rgba(135,134,127,0.3)' }} />
            <span className="w-2.5 h-2.5 rounded-full" style={{ backgroundColor: 'rgba(135,134,127,0.3)' }} />
          </div>
          <span className="text-xs font-mono" style={{ color: MUTED }}>
            ByteBrew Agent <span style={{ color: 'rgba(135,134,127,0.5)' }}>&middot; analytics &middot; claude-sonnet-4</span>
          </span>
        </div>

        {/* Chat */}
        <div ref={chatRef} className="px-4 py-3 space-y-3 overflow-y-auto h-[400px] sm:h-[450px]" style={{ scrollbarWidth: 'thin', scrollbarColor: '#333 transparent' }}>
          {renderedSteps}
        </div>

        {/* Input bar */}
        <div className="flex items-center gap-2 px-4 py-2.5 border-t" style={{ borderColor: 'rgba(135,134,127,0.08)' }}>
          <div
            className="flex-1 rounded-[2px] border px-3 py-1.5 text-xs font-mono"
            style={{
              borderColor: isInputActive ? 'rgba(215,81,62,0.3)' : 'rgba(135,134,127,0.12)',
              color: isInputActive ? '#DFD8D0' : '#87867F',
              backgroundColor: 'rgba(17,17,17,0.4)',
            }}
          >
            {isInputActive ? inputDisplay : 'Type a message...'}
          </div>
          <button className="rounded-[2px] px-3 py-1.5 text-xs text-white shrink-0" style={{ backgroundColor: '#D7513E' }} tabIndex={-1}>
            Send
          </button>
        </div>
      </div>

      {isPaused && (
        <div className="absolute top-12 right-3 rounded-[2px] bg-black/60 px-2 py-0.5 text-[10px]" style={{ color: MUTED }}>
          Paused
        </div>
      )}
    </div>
  );
}
