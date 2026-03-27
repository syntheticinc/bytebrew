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
  { type: 'sub_tool', tool: 'check_tickets', content: 'post-Feb cohort', delay: 1000 },
  { type: 'sub_result', content: '8.3 tickets/acct vs 2.1 avg', delay: 1400 },
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

const SPAWN_STEP_TYPES = new Set<DemoStep['type']>(['spawn', 'sub_tool', 'sub_result', 'spawn_done']);
const TYPEWRITER_TYPES = new Set<DemoStep['type']>(['input_typing', 'text', 'response']);
const USER_CHAR_MS = 55;
const AGENT_CHAR_MS = 12;
const IS_MOBILE = typeof window !== 'undefined' && window.innerWidth < 768;

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

function ToolCallBlock({ tool, args, result, done }: { tool: string; args?: string; result?: string; done?: boolean }) {
  return (
    <div
      className="rounded-[2px] border-l-2 px-3 py-1.5 text-xs font-mono flex items-center gap-0 overflow-hidden"
      style={{ borderColor: done ? BORDER_DONE : BORDER_TOOL, backgroundColor: SURFACE }}
    >
      <span className="flex items-center" style={{ color: MUTED }}>
        <StatusDot done={done} />
        {tool}
      </span>
      {args && <span style={{ color: 'rgba(135,134,127,0.4)' }}> ({args})</span>}
      {done && result && <span className="ml-1" style={{ color: 'rgba(135,134,127,0.6)' }}> — {result}</span>}
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
                color: isSelected ? '#fff' : isFaded ? 'rgba(135,134,127,0.5)' : '#CBC9BC',
                opacity: isFaded ? 0.6 : 1,
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

/* ------------------------------------------------------------------ */
/*  Config YAML (shown in Config tab)                                  */
/* ------------------------------------------------------------------ */

const CONFIG_YAML = `agents:
  analytics:
    model: glm-5
    system_prompt: |
      You are a senior business analyst.
      When asked about metrics, always check for
      anomalies before reporting. If you find
      something unusual, investigate the root cause
      by spawning a research agent.
    tools:
      - get_churn_data:
          description: "Query churn data by segment"
      - get_onboarding_nps:
          description: "NPS scores by onboarding cohort"
    can_spawn: [research-agent]

  research-agent:
    model: glm-5
    system_prompt: |
      You investigate anomalies found by the
      analytics agent. Cross-reference metrics
      with product and pricing changes.
    tools:
      - diff_pricing:
          description: "Compare pricing tiers"
      - check_tickets:
          description: "Support ticket volume"`;

const YAML_KEY = '#D7513E';       // accent — keys stand out
const YAML_STRING = '#8BC78B';    // green — string values
const YAML_BRACKET = '#CBC9BC';   // light — brackets
const YAML_DASH = '#87867F';      // muted — list dashes
const YAML_COLON = 'rgba(135,134,127,0.5)';
const YAML_COMMENT = 'rgba(135,134,127,0.4)';

function ConfigView() {
  return (
    <pre className="px-4 py-3 text-xs font-mono leading-relaxed overflow-auto h-[400px] sm:h-[450px]" style={{ color: '#CBC9BC', scrollbarWidth: 'thin', scrollbarColor: '#333 transparent' }}>
      {CONFIG_YAML.split('\n').map((line, i) => {
        // Empty line
        if (line.trim() === '') return <div key={i}>&nbsp;</div>;

        // Key: value
        const keyMatch = line.match(/^(\s*)([\w_]+)(:)(.*)/);
        if (keyMatch) {
          const [, indent, key, colon, rest] = keyMatch;
          const trimmed = rest.trim();
          const isSection = trimmed === '' || trimmed === '|';

          // Detect string value (quoted or unquoted after colon)
          let valueColor = YAML_BRACKET;
          if (trimmed.startsWith('"') || trimmed.startsWith("'")) valueColor = YAML_STRING;
          else if (trimmed.startsWith('[')) valueColor = YAML_BRACKET;
          else if (trimmed && !isSection) valueColor = YAML_STRING;

          return (
            <div key={i}>
              {indent}
              <span style={{ color: YAML_KEY }}>{key}</span>
              <span style={{ color: YAML_COLON }}>{colon}</span>
              {rest && <span style={{ color: isSection ? YAML_COLON : valueColor }}>{rest}</span>}
            </div>
          );
        }

        // List items (- value)
        const listMatch = line.match(/^(\s*)(- )(.*)/);
        if (listMatch) {
          const [, indent, dash, value] = listMatch;
          // Check if it's a key: value inside list item
          const subKeyMatch = value.match(/^([\w_]+)(:)(.*)/);
          if (subKeyMatch) {
            const [, sk, sc, sv] = subKeyMatch;
            return (
              <div key={i}>
                {indent}<span style={{ color: YAML_DASH }}>{dash}</span>
                <span style={{ color: YAML_KEY }}>{sk}</span>
                <span style={{ color: YAML_COLON }}>{sc}</span>
                <span style={{ color: YAML_STRING }}>{sv}</span>
              </div>
            );
          }
          return (
            <div key={i}>
              {indent}<span style={{ color: YAML_DASH }}>{dash}</span>
              <span style={{ color: YAML_STRING }}>{value}</span>
            </div>
          );
        }

        // Indented text (multiline string content)
        if (line.match(/^\s{6,}/)) {
          return <div key={i} style={{ color: YAML_STRING }}>{line}</div>;
        }

        // Comment
        if (line.trim().startsWith('#')) {
          return <div key={i} style={{ color: YAML_COMMENT }}>{line}</div>;
        }

        return <div key={i} style={{ color: '#87867F' }}>{line}</div>;
      })}
    </pre>
  );
}

/* ------------------------------------------------------------------ */
/*  Main Component                                                     */
/* ------------------------------------------------------------------ */

export function HeroDemo() {
  const [activeTab, setActiveTab] = useState<'config' | 'chat'>('config');
  const [userClickedConfig, setUserClickedConfig] = useState(false);
  const [visibleSteps, setVisibleSteps] = useState(0);
  const [isPaused, setIsPaused] = useState(false);
  const [typingIndex, setTypingIndex] = useState(-1);
  const [typedChars, setTypedChars] = useState(0);
  const chatRef = useRef<HTMLDivElement>(null);
  const [selectedButton, setSelectedButton] = useState<string | undefined>();
  const [inputText, setInputText] = useState('');

  // On mobile (<768px), skip sub-agent spawn steps to keep the demo compact
  const scenario = useMemo(
    () => (IS_MOBILE ? SCENARIO.filter((s) => !SPAWN_STEP_TYPES.has(s.type)) : SCENARIO),
    [],
  );

  // Auto-switch from Config to Chat after 4s (only if user didn't manually click Config)
  useEffect(() => {
    if (activeTab !== 'config' || userClickedConfig) return;
    const t = setTimeout(() => setActiveTab('chat'), 4000);
    return () => clearTimeout(t);
  }, [activeTab, userClickedConfig]);

  // When user leaves the area, reset manual override
  const handleMouseLeave = useCallback(() => {
    setIsPaused(false);
    setUserClickedConfig(false);
  }, []);

  const resetDemo = useCallback(() => {
    setActiveTab('config');
    setUserClickedConfig(false);
    setVisibleSteps(0);
    setTypingIndex(-1);
    setTypedChars(0);
    setSelectedButton(undefined);
    setInputText('');
    brewCounter = 0;
  }, []);

  useEffect(() => {
    if (isPaused) return;

    if (visibleSteps >= scenario.length) {
      const t = setTimeout(resetDemo, 3000);
      return () => clearTimeout(t);
    }

    const step = scenario[visibleSteps];

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
  }, [visibleSteps, isPaused, typingIndex, typedChars, resetDemo, scenario]);

  useEffect(() => {
    chatRef.current?.scrollTo({ top: chatRef.current.scrollHeight, behavior: 'smooth' });
  }, [visibleSteps, typedChars]);

  /* ---- helper: find preceding input_typing text for a given input_send index ---- */
  const getUserTextForSend = useCallback((sendIndex: number) => {
    for (let j = sendIndex - 1; j >= 0; j--) {
      if (scenario[j].type === 'input_typing') return scenario[j].content ?? '';
    }
    return '';
  }, [scenario]);

  const renderedSteps = useMemo(() => {
    const elements: React.ReactNode[] = [];

    for (let i = 0; i < visibleSteps; i++) {
      const step = scenario[i];
      const key = `step-${i}`;
      const text = step.content ?? '';

      switch (step.type) {
        case 'input_typing': break;
        case 'input_send': elements.push(<UserBubble key={key} text={getUserTextForSend(i)} />); break;
        case 'thinking': break;
        case 'tool_call': {
          const nextStep = i + 1 < visibleSteps ? scenario[i + 1] : null;
          if (nextStep?.type === 'tool_result') break; // result renders the done block
          elements.push(<ToolCallBlock key={key} tool={step.tool ?? ''} args={text} />);
          break;
        }
        case 'tool_result': {
          // Find preceding tool_call to get tool name and args
          let callArgs = '';
          for (let j = i - 1; j >= 0; j--) {
            if (scenario[j].type === 'tool_call' || scenario[j].type === 'sub_tool') {
              callArgs = scenario[j].content ?? '';
              break;
            }
          }
          elements.push(<ToolCallBlock key={key} tool={step.tool ?? ''} args={callArgs} result={text} done />);
          break;
        }
        case 'text': elements.push(<AgentText key={key} text={text} />); break;
        case 'spawn': elements.push(<SpawnBlock key={key} content={text} />); break;
        case 'sub_tool': {
          const nextStep = i + 1 < visibleSteps ? scenario[i + 1] : null;
          if (nextStep?.type === 'sub_result') break;
          elements.push(<div key={key} className="ml-4"><ToolCallBlock tool={step.tool ?? ''} args={text} /></div>);
          break;
        }
        case 'sub_result': {
          let callArgs = '';
          let toolName = step.tool ?? '';
          for (let j = i - 1; j >= 0; j--) {
            if (scenario[j].type === 'sub_tool') {
              callArgs = scenario[j].content ?? '';
              toolName = scenario[j].tool ?? toolName;
              break;
            }
          }
          elements.push(<div key={key} className="ml-4"><ToolCallBlock tool={toolName} args={callArgs} result={text} done /></div>);
          break;
        }
        case 'spawn_done': elements.push(<SpawnDoneBlock key={key} content={text} />); break;
        case 'response': elements.push(<AgentText key={key} text={text} isResponse />); break;
        case 'ask_buttons': elements.push(<AskButtons key={key} content={text} options={step.options ?? []} selected={selectedButton} />); break;
        case 'button_click': break;
      }
    }

    // Currently typing (agent only)
    if (typingIndex >= 0 && typingIndex === visibleSteps && typingIndex < scenario.length) {
      const step = scenario[typingIndex];
      if (step.type !== 'input_typing') {
        const partial = (step.content ?? '').slice(0, typedChars);
        elements.push(<AgentText key={`typing-${typingIndex}`} text={partial + '\u258C'} isResponse={step.type === 'response'} />);
      }
    }

    // Active thinking spinner
    if (visibleSteps < scenario.length && scenario[visibleSteps].type === 'thinking' && typingIndex < 0) {
      elements.push(<BrewingSpinner key={`brew-${visibleSteps}`} />);
    }

    return elements;
  }, [visibleSteps, typingIndex, typedChars, selectedButton, getUserTextForSend, scenario]);

  const inputDisplay = useMemo(() => {
    if (typingIndex >= 0 && typingIndex < scenario.length && scenario[typingIndex].type === 'input_typing') {
      return inputText + '\u258C';
    }
    return '';
  }, [typingIndex, inputText]);

  const isInputActive = inputDisplay.length > 0;

  return (
    <div
      className="relative mx-auto w-full max-w-[720px] shadow-lg dark:shadow-none"
      onMouseEnter={() => setIsPaused(true)}
      onMouseLeave={handleMouseLeave}
    >
      <div className="rounded-[2px] border overflow-hidden" style={{ borderColor: 'rgba(135,134,127,0.15)', backgroundColor: '#252525' }}>
        {/* Header with tabs */}
        <div className="flex items-center justify-between px-4 py-2.5 border-b" style={{ borderColor: 'rgba(135,134,127,0.08)' }}>
          <div className="flex items-center gap-3">
            <div className="flex gap-1.5">
              <span className="w-2.5 h-2.5 rounded-full" style={{ backgroundColor: 'rgba(135,134,127,0.3)' }} />
              <span className="w-2.5 h-2.5 rounded-full" style={{ backgroundColor: 'rgba(135,134,127,0.3)' }} />
              <span className="w-2.5 h-2.5 rounded-full" style={{ backgroundColor: 'rgba(135,134,127,0.3)' }} />
            </div>
            <span className="text-xs font-mono" style={{ color: MUTED }}>
              ByteBrew Agent <span style={{ color: 'rgba(135,134,127,0.5)' }}>&middot; analytics &middot; glm-5</span>
            </span>
          </div>
          <div className="flex gap-0 text-[11px] font-mono">
            {(['config', 'chat'] as const).map((tab) => {
              const isActive = activeTab === tab;
              const label = tab === 'config' ? 'agents.yaml' : 'Chat';
              return (
                <button
                  key={tab}
                  className="px-3 py-1.5 transition-colors cursor-pointer border-b-2"
                  style={{
                    color: isActive ? '#DFD8D0' : 'rgba(135,134,127,0.5)',
                    borderBottomColor: isActive ? '#D7513E' : 'transparent',
                    backgroundColor: 'transparent',
                  }}
                  onMouseEnter={(e) => { if (!isActive) e.currentTarget.style.color = '#CBC9BC'; }}
                  onMouseLeave={(e) => { if (!isActive) e.currentTarget.style.color = 'rgba(135,134,127,0.5)'; }}
                  onClick={() => {
                    setActiveTab(tab);
                    if (tab === 'config') setUserClickedConfig(true);
                  }}
                  tabIndex={-1}
                >
                  {label}
                </button>
              );
            })}
          </div>
        </div>

        {/* Content — Config or Chat */}
        {activeTab === 'config' ? (
          <ConfigView />
        ) : (
          <>
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
          </>
        )}
      </div>

      {isPaused && (
        <div className="absolute top-12 right-3 rounded-[2px] bg-black/60 px-2 py-0.5 text-[10px]" style={{ color: MUTED }}>
          Paused
        </div>
      )}
    </div>
  );
}
