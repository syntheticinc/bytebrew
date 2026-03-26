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
    | 'user'
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
/*  Scenario                                                           */
/* ------------------------------------------------------------------ */

const SCENARIO: DemoStep[] = [
  { type: 'user', content: 'Analyze March sales and suggest a strategy', delay: 1500 },
  { type: 'thinking', content: 'Analyzing request... Need sales data first.', delay: 1000 },
  { type: 'tool_call', tool: 'query_database', content: 'SELECT revenue, deals FROM sales WHERE month = "march"', delay: 800 },
  { type: 'tool_result', tool: 'query_database', content: '847 records — $2.3M total revenue, -12% vs February', delay: 1200 },
  { type: 'tool_call', tool: 'analyze_trends', content: 'period: "Q1", metrics: ["revenue", "churn", "growth"]', delay: 800 },
  { type: 'tool_result', tool: 'analyze_trends', content: 'Churn +8% (SMB segment), Enterprise +23% growth', delay: 1200 },
  { type: 'text', content: 'Revenue dropped but enterprise is growing. Let me research competitors...', delay: 1000 },
  { type: 'spawn', content: 'Spawning research-agent: "Analyze competitor pricing Q1 2026"', delay: 1000 },
  { type: 'sub_tool', tool: 'web_search', content: '"SaaS competitor pricing changes March 2026"', delay: 800 },
  { type: 'sub_result', content: 'CompetitorX cut prices 15%, targeting SMB segment', delay: 1000 },
  { type: 'spawn_done', content: 'research-agent completed — competitor intel received', delay: 800 },
  { type: 'tool_call', tool: 'create_document', content: '"Q1 Sales Strategy & Recommendations"', delay: 800 },
  { type: 'tool_result', tool: 'create_document', content: 'Document created: Q1-Strategy.pdf', delay: 800 },
  {
    type: 'response',
    content:
      '**Sales Analysis — March 2026**\n\n' +
      '📊 Revenue: $2.3M (-12% MoM)\n' +
      '📈 Enterprise: +23% growth\n' +
      '⚠️ SMB churn: +8% — competitor price war\n\n' +
      '**Recommendations:**\n' +
      '1. Double down on enterprise segment\n' +
      '2. Launch SMB retention campaign\n' +
      '3. Evaluate competitive pricing tier\n\n' +
      'Document ready: Q1-Strategy.pdf',
    delay: 2000,
  },
  {
    type: 'ask_buttons',
    content: 'How should I proceed with the strategy?',
    options: ['Send to team', 'Export PDF', 'Schedule meeting'],
    delay: 2000,
  },
  { type: 'button_click', content: 'Send to team', delay: 1000 },
  {
    type: 'ask_form',
    content: 'Delivery details:',
    fields: [
      { label: 'To', value: 'team@company.com' },
      { label: 'Subject', value: 'Q1 Sales Strategy' },
      { label: 'To', value: 'High', type: 'radio', options: ['High', 'Normal'] },
    ],
    delay: 2500,
  },
  { type: 'form_submit', delay: 1000 },
  { type: 'tool_call', tool: 'send_email', content: 'to: team@company.com, subject: "Q1 Sales Strategy"', delay: 600 },
  { type: 'tool_result', tool: 'send_email', content: 'Email sent to 12 recipients', delay: 1500 },
  { type: 'text', content: 'Strategy sent to your team. Schedule a review meeting?', delay: 3000 },
];

const TYPEWRITER_TYPES = new Set<DemoStep['type']>(['user', 'text', 'response']);
const USER_CHAR_MS = 25;
const AGENT_CHAR_MS = 12;

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

function ThinkingBubble({ text }: { text: string }) {
  return (
    <div className="text-sm italic" style={{ color: '#87867F' }}>
      {text}
      <span className="animate-pulse"> ...</span>
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
      \u2713 {content}
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

function AskForm({
  content,
  fields,
  submitted,
}: {
  content: string;
  fields: DemoField[];
  submitted?: boolean;
}) {
  if (submitted) {
    return (
      <div className="text-xs italic" style={{ color: '#87867F' }}>
        Form submitted
      </div>
    );
  }
  return (
    <div className="space-y-2">
      <div className="text-sm text-brand-light/90">{content}</div>
      <div
        className="rounded-[2px] border p-3 space-y-2"
        style={{
          borderColor: 'rgba(135,134,127,0.15)',
          backgroundColor: 'rgba(17,17,17,0.5)',
        }}
      >
        {fields.map((f, i) => (
          <div key={i} className="flex items-center gap-2 text-xs">
            <span className="text-brand-shade2 w-16 shrink-0">{f.label}:</span>
            {f.type === 'radio' && f.options ? (
              <div className="flex gap-3">
                {f.options.map((o) => (
                  <label key={o} className="flex items-center gap-1 text-brand-shade2">
                    <span
                      className="w-3 h-3 rounded-full border inline-block"
                      style={{
                        borderColor: 'rgba(135,134,127,0.4)',
                        backgroundColor: o === f.value ? '#D7513E' : 'transparent',
                      }}
                    />
                    {o}
                  </label>
                ))}
              </div>
            ) : (
              <span
                className="flex-1 rounded-[2px] border px-2 py-0.5 text-brand-light/80 font-mono"
                style={{
                  borderColor: 'rgba(135,134,127,0.2)',
                  backgroundColor: 'rgba(31,31,31,0.8)',
                }}
              >
                {f.value}
              </span>
            )}
          </div>
        ))}
        <div className="flex justify-end pt-1">
          <span
            className="rounded-[2px] px-3 py-1 text-xs text-white"
            style={{ backgroundColor: '#D7513E' }}
          >
            Submit
          </span>
        </div>
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
  const [formSubmitted, setFormSubmitted] = useState(false);

  /* ---- reset helper ---- */
  const resetDemo = useCallback(() => {
    setVisibleSteps(0);
    setTypingIndex(-1);
    setTypedChars(0);
    setSelectedButton(undefined);
    setFormSubmitted(false);
  }, []);

  /* ---- advance logic ---- */
  useEffect(() => {
    if (isPaused) return;

    if (visibleSteps >= SCENARIO.length) {
      const t = setTimeout(resetDemo, 3000);
      return () => clearTimeout(t);
    }

    const step = SCENARIO[visibleSteps];

    // If this step needs typewriter and we haven't started it yet
    if (TYPEWRITER_TYPES.has(step.type) && step.content) {
      if (typingIndex !== visibleSteps) {
        // Start typewriter for this step
        setTypingIndex(visibleSteps);
        setTypedChars(0);
        return;
      }

      // Typewriter in progress
      if (typedChars < step.content.length) {
        const charMs = step.type === 'user' ? USER_CHAR_MS : AGENT_CHAR_MS;
        const t = setTimeout(() => setTypedChars((c) => c + 1), charMs);
        return () => clearTimeout(t);
      }

      // Typewriter done — wait step.delay then advance
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

    // Handle form_submit
    if (step.type === 'form_submit') {
      setFormSubmitted(true);
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

  /* ---- render steps ---- */
  const renderedSteps = useMemo(() => {
    const elements: React.ReactNode[] = [];

    for (let i = 0; i < visibleSteps; i++) {
      const step = SCENARIO[i];
      const key = `step-${i}`;

      // For completed typewriter steps, show full content
      const displayText = step.content ?? '';

      switch (step.type) {
        case 'user':
          elements.push(<UserBubble key={key} text={displayText} />);
          break;
        case 'thinking':
          elements.push(<ThinkingBubble key={key} text={displayText} />);
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
          // Visual effect handled by updating selectedButton on ask_buttons
          break;
        case 'ask_form':
          elements.push(
            <AskForm
              key={key}
              content={displayText}
              fields={step.fields ?? []}
              submitted={formSubmitted}
            />,
          );
          break;
        case 'form_submit':
          // Visual effect handled by formSubmitted flag on ask_form
          break;
      }
    }

    // Currently-typing step
    if (typingIndex >= 0 && typingIndex === visibleSteps && typingIndex < SCENARIO.length) {
      const step = SCENARIO[typingIndex];
      const partial = (step.content ?? '').slice(0, typedChars);
      const key = `typing-${typingIndex}`;

      if (step.type === 'user') {
        elements.push(<UserBubble key={key} text={partial + '\u258C'} />);
      } else if (step.type === 'response') {
        elements.push(<AgentText key={key} text={partial + '\u258C'} isResponse />);
      } else {
        elements.push(<AgentText key={key} text={partial + '\u258C'} />);
      }
    }

    return elements;
  }, [visibleSteps, typingIndex, typedChars, selectedButton, formSubmitted]);

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
            <span className="text-brand-shade3">&middot; sales-assistant &middot; gpt-4o</span>
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

        {/* Footer */}
        <div
          className="flex items-center gap-2 px-4 py-2.5 border-t"
          style={{ borderColor: 'rgba(135,134,127,0.1)' }}
        >
          <div
            className="flex-1 rounded-[2px] border px-3 py-1.5 text-xs"
            style={{
              borderColor: 'rgba(135,134,127,0.15)',
              color: '#87867F',
              backgroundColor: 'rgba(17,17,17,0.4)',
            }}
          >
            Type a message...
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
