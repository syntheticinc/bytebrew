import { useState, useEffect, useRef, useCallback, useMemo } from 'react';

/* ------------------------------------------------------------------ */
/*  Types                                                              */
/* ------------------------------------------------------------------ */

interface DemoField {
  label: string;
  value: string;
  type?: 'text' | 'radio';
  options?: string[];
}

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
    | 'button_click'
    | 'ask_form'
    | 'form_submit';
  content?: string;
  tool?: string;
  options?: string[];
  fields?: DemoField[];
  delay: number;
}

/* ------------------------------------------------------------------ */
/*  Scenario — deep anomaly insight                                    */
/* ------------------------------------------------------------------ */

const SCENARIO: DemoStep[] = [
  // User types in input bar
  { type: 'input_typing', content: 'Review Q1 and flag anything unusual', delay: 600 },
  { type: 'input_send', delay: 800 },
  // Brewing spinner shows while agent "thinks"
  { type: 'thinking', delay: 2800 },
  // Agent starts working
  { type: 'tool_call', tool: 'fetch_metrics', content: 'Q1-2026, segments: revenue, NPS, churn', delay: 1200 },
  { type: 'tool_result', tool: 'fetch_metrics', content: '4 categories loaded, 3 months', delay: 2200 },
  { type: 'tool_call', tool: 'anomaly_detection', content: 'sensitivity: high', delay: 1400 },
  { type: 'tool_result', tool: 'anomaly_detection', content: '3 anomalies detected', delay: 1600 },
  // Pause — agent "notices" something
  { type: 'thinking', delay: 2000 },
  { type: 'text', content: 'Hmm, there could be a risk here. Let me check something...', delay: 1800 },
  // Spawn sub-agent
  { type: 'spawn', content: 'analytics-agent: "Correlate NPS drop with product changes"', delay: 2200 },
  { type: 'sub_tool', tool: 'correlate_events', content: 'NPS anomaly + pricing_updates', delay: 1600 },
  { type: 'sub_result', content: 'Strong correlation: NPS drop → Feb pricing change', delay: 1400 },
  { type: 'spawn_done', content: 'analytics-agent completed', delay: 1000 },
  // The insight — concise and impactful
  {
    type: 'response',
    content:
      '**Hidden risk found.**\n\n' +
      'Revenue is up 18%, but NPS dropped 12 points in Enterprise — specifically accounts onboarded after the February pricing change.\n\n' +
      '**Root cause:** removed dedicated onboarding calls. Enterprise buyers feel abandoned.\n\n' +
      '**Predicted impact:** ~$420K churn over 2 quarters.\n' +
      '**Fix cost:** $15K/quarter. **ROI: 28:1.**',
    delay: 3000,
  },
  {
    type: 'ask_buttons',
    content: 'How to proceed?',
    options: ['Draft rescue campaign', 'Alert leadership', 'Deep dive'],
    delay: 2500,
  },
  { type: 'button_click', content: 'Draft rescue campaign', delay: 1200 },
  { type: 'tool_call', tool: 'draft_campaign', content: '23 affected accounts, template: white-glove', delay: 1400 },
  { type: 'tool_result', tool: 'draft_campaign', content: '23 personalized emails ready', delay: 2000 },
  { type: 'text', content: 'Campaign queued. Review before sending?', delay: 4000 },
];

const TYPEWRITER_TYPES = new Set<DemoStep['type']>(['input_typing', 'text', 'response']);
const USER_CHAR_MS = 30;
const AGENT_CHAR_MS = 14;

/* ------------------------------------------------------------------ */
/*  Tiny markdown renderer                                             */
/* ------------------------------------------------------------------ */

function renderMarkdown(raw: string): React.ReactNode[] {
  const lines = raw.split('\n');
  const nodes: React.ReactNode[] = [];

  for (let i = 0; i < lines.length; i++) {
    const line = lines[i];

    if (line === '') {
      nodes.push(<br key={`br-${i}`} />);
      continue;
    }

    const listMatch = line.match(/^(\d+)\.\s+(.+)$/);
    if (listMatch) {
      nodes.push(
        <div key={`li-${i}`} className="pl-3">
          <span className="text-brand-shade2 mr-1">{listMatch[1]}.</span>
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
  return parts.map((p, i) => (i % 2 === 1 ? <strong key={i}>{p}</strong> : p));
}

/* ------------------------------------------------------------------ */
/*  Brewing spinner                                                    */
/* ------------------------------------------------------------------ */

function BrewingSpinner() {
  return (
    <div className="flex items-center gap-2 py-2">
      <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="1.5" className="w-4 h-4 text-brand-shade3">
        <path d="M17 8h1a4 4 0 010 8h-1" strokeLinecap="round" />
        <path d="M3 8h14v9a4 4 0 01-4 4H7a4 4 0 01-4-4V8z" />
        <path d="M7 2v3" strokeLinecap="round" style={{ animation: 'steam 1.2s ease-in-out infinite' }} />
        <path d="M10 1v3" strokeLinecap="round" style={{ animation: 'steam 1.2s ease-in-out infinite 0.3s' }} />
        <path d="M13 2v3" strokeLinecap="round" style={{ animation: 'steam 1.2s ease-in-out infinite 0.6s' }} />
      </svg>
      <span className="text-xs font-mono text-brand-shade3 animate-pulse">Brewing...</span>
      <style>{`
        @keyframes steam {
          0% { opacity: 0.3; transform: translateY(0); }
          50% { opacity: 1; transform: translateY(-3px); }
          100% { opacity: 0.3; transform: translateY(0); }
        }
      `}</style>
    </div>
  );
}

/* ------------------------------------------------------------------ */
/*  Step renderers                                                     */
/* ------------------------------------------------------------------ */

function UserBubble({ text }: { text: string }) {
  return (
    <div className="flex justify-end">
      <div
        className="max-w-[80%] rounded-[2px] px-3 py-2 text-sm text-white"
        style={{ backgroundColor: '#D7513E' }}
      >
        {text}
      </div>
    </div>
  );
}

function ToolCallBlock({ tool, content, done }: { tool: string; content: string; done?: boolean }) {
  return (
    <div
      className="rounded-[2px] border-l-2 px-3 py-2 text-xs font-mono"
      style={{
        borderColor: done ? 'rgba(34,197,94,0.5)' : 'rgba(249,115,22,0.5)',
        backgroundColor: 'rgba(30,41,59,0.5)',
      }}
    >
      <span className="text-brand-shade2">
        {done ? '\u2705 ' : '\u2699\uFE0F '}
        {tool}
      </span>
      {!done && <span className="text-orange-400/70">({content})</span>}
      {done && <span className="text-green-400/80 ml-1">{content}</span>}
      {!done && (
        <span className="ml-2 text-orange-300/50 animate-pulse text-[10px]">Running...</span>
      )}
    </div>
  );
}

function SpawnBlock({ content }: { content: string }) {
  return (
    <div
      className="rounded-[2px] border-l-2 px-3 py-2 text-xs"
      style={{
        borderColor: 'rgba(59,130,246,0.5)',
        backgroundColor: 'rgba(30,41,59,0.4)',
        color: '#93c5fd',
      }}
    >
      {'\uD83D\uDD00'} {content}
    </div>
  );
}

function SpawnDoneBlock({ content }: { content: string }) {
  return (
    <div
      className="rounded-[2px] border-l-2 px-3 py-2 text-xs"
      style={{
        borderColor: 'rgba(59,130,246,0.5)',
        backgroundColor: 'rgba(30,41,59,0.3)',
        color: '#93c5fd',
      }}
    >
      {'\u2713'} {content}
    </div>
  );
}

function AgentText({ text, isResponse }: { text: string; isResponse?: boolean }) {
  return (
    <div className="text-sm text-brand-light/90">
      {isResponse ? renderMarkdown(text) : text}
    </div>
  );
}

function AskButtons({
  content,
  options,
  selected,
}: {
  content: string;
  options: string[];
  selected?: string;
}) {
  return (
    <div className="space-y-2">
      <div className="text-sm text-brand-light/90">{content}</div>
      <div className="flex flex-wrap gap-2">
        {options.map((opt) => {
          const isSelected = selected === opt;
          const isFaded = selected && !isSelected;
          return (
            <span
              key={opt}
              className="rounded-[2px] border px-3 py-1 text-xs transition-all duration-300"
              style={{
                borderColor: isSelected ? '#D7513E' : 'rgba(135,134,127,0.3)',
                backgroundColor: isSelected ? '#D7513E' : 'transparent',
                color: isSelected ? '#fff' : isFaded ? 'rgba(135,134,127,0.4)' : '#CBC9BC',
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

  /* ---- reset helper ---- */
  const resetDemo = useCallback(() => {
    setVisibleSteps(0);
    setTypingIndex(-1);
    setTypedChars(0);
    setSelectedButton(undefined);
    setInputText('');
  }, []);

  /* ---- advance logic ---- */
  useEffect(() => {
    if (isPaused) return;

    if (visibleSteps >= SCENARIO.length) {
      const t = setTimeout(resetDemo, 3000);
      return () => clearTimeout(t);
    }

    const step = SCENARIO[visibleSteps];

    // input_typing — typewriter into input bar
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
      // Done typing — wait before "sending"
      const t = setTimeout(() => {
        setVisibleSteps((v) => v + 1);
        setTypingIndex(-1);
      }, step.delay);
      return () => clearTimeout(t);
    }

    // input_send — move text from input bar to chat body
    if (step.type === 'input_send') {
      setInputText('');
      const t = setTimeout(() => setVisibleSteps((v) => v + 1), step.delay);
      return () => clearTimeout(t);
    }

    // Typewriter for agent text/response
    if (TYPEWRITER_TYPES.has(step.type) && step.content && step.type !== 'input_typing') {
      if (typingIndex !== visibleSteps) {
        setTypingIndex(visibleSteps);
        setTypedChars(0);
        return;
      }
      if (typedChars < step.content.length) {
        const t = setTimeout(() => setTypedChars((c) => c + 1), AGENT_CHAR_MS);
        return () => clearTimeout(t);
      }
      const t = setTimeout(() => {
        setVisibleSteps((v) => v + 1);
        setTypingIndex(-1);
      }, step.delay);
      return () => clearTimeout(t);
    }

    // Handle button_click
    if (step.type === 'button_click') {
      setSelectedButton(step.content);
      const t = setTimeout(() => setVisibleSteps((v) => v + 1), step.delay);
      return () => clearTimeout(t);
    }

    // Default: wait delay then advance
    const t = setTimeout(() => setVisibleSteps((v) => v + 1), step.delay);
    return () => clearTimeout(t);
  }, [visibleSteps, isPaused, typingIndex, typedChars, resetDemo]);

  /* ---- auto-scroll ---- */
  useEffect(() => {
    chatRef.current?.scrollTo({ top: chatRef.current.scrollHeight, behavior: 'smooth' });
  }, [visibleSteps, typedChars]);

  /* ---- find user message text for chat body ---- */
  const userMessageText = useMemo(() => {
    const typingStep = SCENARIO.find((s) => s.type === 'input_typing');
    return typingStep?.content ?? '';
  }, []);

  /* ---- render steps ---- */
  const renderedSteps = useMemo(() => {
    const elements: React.ReactNode[] = [];

    for (let i = 0; i < visibleSteps; i++) {
      const step = SCENARIO[i];
      const key = `step-${i}`;
      const displayText = step.content ?? '';

      switch (step.type) {
        case 'input_typing':
          // Don't render in chat body — was in input bar
          break;
        case 'input_send':
          // Show user message in chat body now
          elements.push(<UserBubble key={key} text={userMessageText} />);
          break;
        case 'thinking':
          // Completed thinking — don't render (spinner disappears)
          break;
        case 'tool_call':
          elements.push(
            <ToolCallBlock key={key} tool={step.tool ?? ''} content={displayText} done={false} />,
          );
          break;
        case 'tool_result':
          elements.push(
            <ToolCallBlock key={key} tool={step.tool ?? ''} content={displayText} done />,
          );
          break;
        case 'text':
          elements.push(<AgentText key={key} text={displayText} />);
          break;
        case 'spawn':
          elements.push(<SpawnBlock key={key} content={displayText} />);
          break;
        case 'sub_tool':
          elements.push(
            <div key={key} className="ml-4">
              <ToolCallBlock tool={step.tool ?? ''} content={displayText} done={false} />
            </div>,
          );
          break;
        case 'sub_result':
          elements.push(
            <div key={key} className="ml-4">
              <ToolCallBlock tool="" content={displayText} done />
            </div>,
          );
          break;
        case 'spawn_done':
          elements.push(<SpawnDoneBlock key={key} content={displayText} />);
          break;
        case 'response':
          elements.push(<AgentText key={key} text={displayText} isResponse />);
          break;
        case 'ask_buttons':
          elements.push(
            <AskButtons
              key={key}
              content={displayText}
              options={step.options ?? []}
              selected={selectedButton}
            />,
          );
          break;
        case 'button_click':
          break;
      }
    }

    // Currently-typing step (agent text/response only — input_typing goes to input bar)
    if (typingIndex >= 0 && typingIndex === visibleSteps && typingIndex < SCENARIO.length) {
      const step = SCENARIO[typingIndex];
      if (step.type !== 'input_typing') {
        const partial = (step.content ?? '').slice(0, typedChars);
        const key = `typing-${typingIndex}`;
        if (step.type === 'response') {
          elements.push(<AgentText key={key} text={partial + '\u258C'} isResponse />);
        } else {
          elements.push(<AgentText key={key} text={partial + '\u258C'} />);
        }
      }
    }

    // Show brewing spinner for thinking step while it's active
    if (visibleSteps < SCENARIO.length && SCENARIO[visibleSteps].type === 'thinking' && typingIndex < 0) {
      elements.push(<BrewingSpinner key="thinking-active" />);
    }

    return elements;
  }, [visibleSteps, typingIndex, typedChars, selectedButton, userMessageText]);

  /* ---- input bar display ---- */
  const inputDisplay = useMemo(() => {
    // During input_typing — show typed text with cursor
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
      <div
        className="rounded-[2px] border overflow-hidden"
        style={{
          borderColor: 'rgba(135,134,127,0.15)',
          backgroundColor: '#1F1F1F',
        }}
      >
        {/* Header */}
        <div
          className="flex items-center gap-3 px-4 py-2.5 border-b"
          style={{ borderColor: 'rgba(135,134,127,0.1)' }}
        >
          <div className="flex gap-1.5">
            <span className="w-2.5 h-2.5 rounded-full bg-red-500/80" />
            <span className="w-2.5 h-2.5 rounded-full bg-yellow-500/80" />
            <span className="w-2.5 h-2.5 rounded-full bg-green-500/80" />
          </div>
          <span className="text-xs text-brand-shade2 font-mono">
            ByteBrew Agent{' '}
            <span className="text-brand-shade3">&middot; analytics-assistant &middot; gpt-4o</span>
          </span>
        </div>

        {/* Chat area */}
        <div
          ref={chatRef}
          className="px-4 py-3 space-y-3 overflow-y-auto h-[400px] sm:h-[450px]"
          style={{ scrollbarWidth: 'thin', scrollbarColor: '#333 transparent' }}
        >
          {renderedSteps}
        </div>

        {/* Footer — input bar */}
        <div
          className="flex items-center gap-2 px-4 py-2.5 border-t"
          style={{ borderColor: 'rgba(135,134,127,0.1)' }}
        >
          <div
            className="flex-1 rounded-[2px] border px-3 py-1.5 text-xs font-mono"
            style={{
              borderColor: isInputActive ? 'rgba(215,81,62,0.4)' : 'rgba(135,134,127,0.15)',
              color: isInputActive ? '#F7F8F1' : '#87867F',
              backgroundColor: 'rgba(17,17,17,0.4)',
            }}
          >
            {isInputActive ? inputDisplay : 'Type a message...'}
          </div>
          <button
            className="rounded-[2px] px-3 py-1.5 text-xs text-white shrink-0"
            style={{ backgroundColor: '#D7513E' }}
            tabIndex={-1}
          >
            Send
          </button>
        </div>
      </div>

      {/* Pause indicator */}
      {isPaused && (
        <div className="absolute top-12 right-3 rounded-[2px] bg-black/60 px-2 py-0.5 text-[10px] text-brand-shade3">
          Paused
        </div>
      )}
    </div>
  );
}
