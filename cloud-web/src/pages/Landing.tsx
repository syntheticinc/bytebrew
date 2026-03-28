import { useState, useEffect } from 'react';
import { Link } from '@tanstack/react-router';
import { TerminalBlock } from '../components/TerminalBlock';
import { EnginePricingTable } from '../components/EnginePricingTable';
import { HeroDemo } from '../components/HeroDemo';

export function LandingPage() {
  const [lightboxSrc, setLightboxSrc] = useState<string | null>(null);

  useEffect(() => {
    if (!lightboxSrc) return;
    const onKeyDown = (e: KeyboardEvent) => {
      if (e.key === 'Escape') setLightboxSrc(null);
    };
    document.addEventListener('keydown', onKeyDown);
    return () => document.removeEventListener('keydown', onKeyDown);
  }, [lightboxSrc]);

  return (
    <div>
      <HeroSection />
      <div className="py-6 px-4 border-t border-border bg-surface">
        <div className="max-w-4xl mx-auto flex flex-wrap items-center justify-center gap-x-3 gap-y-2">
          <span className="text-xs font-sans text-text-tertiary uppercase tracking-wider">Works with</span>
          <span className="text-text-tertiary">·</span>
          <span className="text-sm font-mono font-medium text-text-secondary">OpenAI</span>
          <span className="text-text-tertiary">·</span>
          <span className="text-sm font-mono font-medium text-text-secondary">Anthropic</span>
          <span className="text-text-tertiary">·</span>
          <span className="text-sm font-mono font-medium text-text-secondary">Google AI</span>
          <span className="text-text-tertiary">·</span>
          <span className="text-sm font-mono font-medium text-text-secondary">Ollama</span>
          <span className="text-text-tertiary">·</span>
          <span className="text-sm font-mono font-medium text-text-secondary">PostgreSQL</span>
          <span className="text-text-tertiary">·</span>
          <span className="text-sm font-mono font-medium text-text-secondary">Docker</span>
        </div>
      </div>
      <ProblemSection />
      <SolutionSection />
      <HowItWorksSection onImageClick={setLightboxSrc} />
      <CapabilitiesSection />
      <ProductShowcaseSection onImageClick={setLightboxSrc} />
      <ProductionReadySection />
      <UseCasesSection />
      <ComparisonSection />
      <InstallSection />
      <section className="py-16 px-4 border-t border-border bg-surface">
        <div className="max-w-3xl mx-auto text-center">
          <p className="text-lg font-sans font-medium text-text-primary">
            Powering AI agents in production.
          </p>
          <p className="text-sm font-sans text-text-secondary mt-2">
            Teams choose ByteBrew as their unfair advantage — a production-ready AI engine they deploy on their own infrastructure.
          </p>
          <div className="flex flex-wrap items-center justify-center gap-6 mt-6">
            <div className="rounded-[2px] border border-border bg-surface-alt px-4 py-2">
              <div className="text-xs font-sans text-text-tertiary">Self-dogfooding</div>
              <div className="text-sm font-sans font-medium text-text-primary">This site's AI docs are powered by ByteBrew</div>
            </div>
            <div className="rounded-[2px] border border-border bg-surface-alt px-4 py-2">
              <div className="text-xs font-sans text-text-tertiary">Live examples</div>
              <div className="text-sm font-sans font-medium text-text-primary">
                <a href="/examples" className="text-brand-accent hover:underline">Try working agents &rarr;</a>
              </div>
            </div>
            <div className="rounded-[2px] border border-border bg-surface-alt px-4 py-2">
              <div className="text-xs font-sans text-text-tertiary">Open examples</div>
              <div className="text-sm font-sans font-medium text-text-primary">
                <a href="https://github.com/syntheticinc/bytebrew-examples" target="_blank" rel="noopener noreferrer" className="text-brand-accent hover:underline">GitHub &rarr;</a>
              </div>
            </div>
          </div>
        </div>
      </section>
      <PricingSection />
      <FreeForeverBanner />
      {/* MCPDocsSection removed from landing — available on /docs and /download */}
      <FinalCTASection />

      {lightboxSrc && (
        <div
          className="fixed inset-0 z-50 flex items-center justify-center bg-black/80 backdrop-blur-sm cursor-pointer"
          onClick={() => setLightboxSrc(null)}
        >
          <img
            src={lightboxSrc}
            alt=""
            className="max-w-[90vw] max-h-[90vh] object-contain rounded-[2px] shadow-2xl"
          />
        </div>
      )}
    </div>
  );
}

/* ------------------------------------------------------------------ */
/*  Shared icons                                                       */
/* ------------------------------------------------------------------ */

function CheckIcon({ className = 'h-5 w-5 text-emerald-400' }: { className?: string }) {
  return (
    <svg
      className={`shrink-0 ${className}`}
      fill="none"
      viewBox="0 0 24 24"
      strokeWidth={2.5}
      stroke="currentColor"
    >
      <path
        strokeLinecap="round"
        strokeLinejoin="round"
        d="M4.5 12.75l6 6 9-13.5"
      />
    </svg>
  );
}

/* ------------------------------------------------------------------ */
/*  1. Hero                                                            */
/* ------------------------------------------------------------------ */

function HeroSection() {
  return (
    <section className="py-20 px-4 bg-surface">
      <div className="max-w-5xl mx-auto text-center mb-12">
        <div className="flex items-center justify-center gap-3 mb-4">
          <span className="rounded-full border border-brand-accent/30 bg-brand-accent/10 px-3 py-1 text-xs font-sans font-medium text-brand-accent">
            Self-hosted
          </span>
          <span className="rounded-full border border-border px-3 py-1 text-xs font-sans font-medium text-text-secondary">
            Free forever
          </span>
        </div>
        <h1 className="text-5xl sm:text-6xl font-bold tracking-tight text-text-primary leading-tight">
          Ship AI agents in your product —{' '}
          <span className="text-brand-accent">not months of infrastructure</span>
        </h1>
        <p className="mt-6 text-lg text-text-secondary font-sans leading-relaxed max-w-2xl mx-auto">
          Self-hosted. One Docker container. Any LLM. Free forever.
        </p>
        <div className="mt-8 flex items-center justify-center gap-4 flex-wrap">
          <Link
            to="/download"
            className="rounded-[2px] bg-brand-accent px-7 py-3.5 text-sm font-medium text-white hover:bg-brand-accent-hover transition-colors"
          >
            Get Started Free &rarr;
          </Link>
          <Link
            to="/pricing"
            className="rounded-[2px] border border-border px-7 py-3.5 text-sm font-medium text-text-secondary hover:border-border-hover hover:text-text-primary transition-colors"
          >
            View Pricing &rarr;
          </Link>
        </div>
        <div className="mt-3 flex items-center justify-center gap-4 flex-wrap">
          <a href="/docs/getting-started/quick-start/" className="text-sm text-text-tertiary hover:text-text-secondary transition-colors">5-minute Quick Start &rarr;</a>
          <a href="/examples" className="inline-flex items-center gap-1 rounded-full border border-brand-accent/30 bg-brand-accent/5 px-4 py-1.5 text-sm font-sans font-medium text-brand-accent hover:bg-brand-accent/10 transition-colors">
            Try a live agent &rarr;
          </a>
        </div>
      </div>

      <div className="max-w-5xl mx-auto">
          <HeroDemo />
      </div>
    </section>
  );
}

/* ------------------------------------------------------------------ */
/*  2. Problem Statement                                               */
/* ------------------------------------------------------------------ */

const PROBLEMS = [
  {
    title: 'Without ByteBrew',
    description:
      '3-6 months building agent infrastructure. Your actual product waits.',
    icon: (
      <svg className="h-6 w-6 text-brand-accent" fill="none" viewBox="0 0 24 24" strokeWidth={1.5} stroke="currentColor">
        <path strokeLinecap="round" strokeLinejoin="round" d="M12 6v6h4.5m4.5 0a9 9 0 11-18 0 9 9 0 0118 0z" />
      </svg>
    ),
  },
  {
    title: 'With cloud AI platforms',
    description:
      '$500-2,000/month. Your data leaves. Locked to one provider.',
    icon: (
      <svg className="h-6 w-6 text-brand-accent" fill="none" viewBox="0 0 24 24" strokeWidth={1.5} stroke="currentColor">
        <path strokeLinecap="round" strokeLinejoin="round" d="M16.5 10.5V6.75a4.5 4.5 0 10-9 0v3.75m-.75 11.25h10.5a2.25 2.25 0 002.25-2.25v-6.75a2.25 2.25 0 00-2.25-2.25H6.75a2.25 2.25 0 00-2.25 2.25v6.75a2.25 2.25 0 002.25 2.25z" />
      </svg>
    ),
  },
  {
    title: 'With AI frameworks',
    description:
      'A Python library, not a product. You\'re still building infrastructure.',
    icon: (
      <svg className="h-6 w-6 text-brand-accent" fill="none" viewBox="0 0 24 24" strokeWidth={1.5} stroke="currentColor">
        <path strokeLinecap="round" strokeLinejoin="round" d="M14.25 9.75L16.5 12l-2.25 2.25m-4.5 0L7.5 12l2.25-2.25M6 20.25h12A2.25 2.25 0 0020.25 18V6A2.25 2.25 0 0018 3.75H6A2.25 2.25 0 003.75 6v12A2.25 2.25 0 006 20.25z" />
      </svg>
    ),
  },
];

function ProblemSection() {
  return (
    <section className="py-28 px-4 border-t border-border bg-surface-alt">
      <div className="max-w-5xl mx-auto">
        <h2 className="text-3xl font-bold tracking-tight text-center text-text-primary mb-4">
          You want AI in your product. Not a 3-month infrastructure project.
        </h2>
        <p className="text-center text-text-secondary font-sans mb-14 max-w-2xl mx-auto">
          Sound familiar?
        </p>
        <div className="grid grid-cols-1 md:grid-cols-3 gap-8">
          {PROBLEMS.map((p) => (
            <div
              key={p.title}
              className="rounded-[2px] border border-border bg-surface p-6 transition-all duration-200 hover:border-border-hover hover:-translate-y-0.5"
            >
              <div className="w-12 h-12 rounded-[2px] bg-brand-accent/10 flex items-center justify-center mb-4">
                {p.icon}
              </div>
              <h3 className="text-lg font-semibold text-text-primary">{p.title}</h3>
              <p className="mt-3 text-sm text-text-secondary font-sans leading-relaxed">
                {p.description}
              </p>
            </div>
          ))}
        </div>
      </div>
    </section>
  );
}

/* ------------------------------------------------------------------ */
/*  3. Solution                                                        */
/* ------------------------------------------------------------------ */

function SolutionSection() {
  return (
    <section className="py-28 px-4 border-t border-border bg-surface">
      <div className="max-w-6xl mx-auto">
        <h2 className="text-3xl font-bold tracking-tight text-center text-text-primary mb-4">
          One Docker container. Full AI agent runtime. Your data stays with you.
        </h2>
        <p className="text-center text-text-secondary font-sans mb-14 max-w-2xl mx-auto">
          Your server talks to ByteBrew via REST API. Agents think, call tools, stream responses back.
        </p>

        <div className="flex items-center justify-center gap-4 mt-12 mb-12 py-6 flex-wrap">
          <div className="rounded-[2px] border border-border bg-surface-alt px-8 py-4 text-center">
            <div className="text-sm text-text-tertiary">Your Server</div>
            <div className="text-base font-semibold text-text-primary mt-0.5">Next.js, FastAPI, Go</div>
          </div>
          <div className="flex flex-col items-center gap-0.5">
            <span className="text-sm text-brand-accent font-medium">REST API →</span>
            <span className="text-sm text-text-tertiary">← SSE stream</span>
          </div>
          <div className="rounded-[2px] border-2 border-brand-accent shadow-md bg-surface-alt px-8 py-4 text-center">
            <div className="text-sm text-brand-accent font-medium">ByteBrew Engine</div>
            <div className="text-base font-semibold text-text-primary mt-0.5">Docker Container</div>
          </div>
          <div className="flex flex-col items-center gap-0.5">
            <span className="text-sm text-text-tertiary font-medium">→</span>
          </div>
          <div className="rounded-[2px] border border-border bg-surface-alt px-8 py-4 text-center">
            <div className="text-sm text-text-tertiary">Any LLM</div>
            <div className="text-base font-semibold text-text-primary mt-0.5">OpenAI, Gemini, Ollama</div>
          </div>
        </div>
        <p className="text-center text-xs font-sans text-text-tertiary -mt-8 mb-8">Your server handles authentication, then forwards requests to ByteBrew.</p>

        <div className="grid grid-cols-1 lg:grid-cols-2 gap-10 items-start">
          {/* Admin Dashboard workflow on the left */}
          <div className="rounded-[2px] border border-border bg-surface-elevated p-6">
            <div className="text-xs font-mono text-text-tertiary uppercase tracking-wider mb-4">Admin Dashboard</div>
            <div className="space-y-3">
              <div className="flex items-center gap-3">
                <div className="w-2 h-2 rounded-full bg-brand-accent"></div>
                <span className="text-sm font-sans text-text-primary">Create agents visually — name, model, system prompt</span>
              </div>
              <div className="flex items-center gap-3">
                <div className="w-2 h-2 rounded-full bg-brand-accent"></div>
                <span className="text-sm font-sans text-text-primary">Enable tools — MCP servers, built-in tools, HTTP APIs</span>
              </div>
              <div className="flex items-center gap-3">
                <div className="w-2 h-2 rounded-full bg-brand-accent"></div>
                <span className="text-sm font-sans text-text-primary">Set up multi-agent delegation — which agents can spawn which</span>
              </div>
              <div className="flex items-center gap-3">
                <div className="w-2 h-2 rounded-full bg-brand-accent"></div>
                <span className="text-sm font-sans text-text-primary">Configure triggers — cron schedules, webhook endpoints</span>
              </div>
              <div className="flex items-center gap-3">
                <div className="w-2 h-2 rounded-full bg-brand-accent"></div>
                <span className="text-sm font-sans text-text-primary">Manage models — add providers, test connectivity, set defaults</span>
              </div>
            </div>
            <div className="mt-4 pt-3 border-t border-border">
              <span className="text-xs font-sans text-text-tertiary">Prefer code? Export/import configuration as YAML anytime.</span>
            </div>
          </div>

          {/* Explanation on the right */}
          <div className="space-y-6">
            <ExplainerItem
              title="Agents that delegate"
              description="Supervisor breaks tasks into sub-tasks and assigns to specialist agents."
            />
            <ExplainerItem
              title="Any tool your app needs"
              description="Connect via MCP protocol or declare HTTP tools in the dashboard."
            />
            <ExplainerItem
              title="Agents that work on their own"
              description="Cron schedules and webhook triggers — no user interaction needed."
            />
            <ExplainerItem
              title="Any AI model, no lock-in"
              description="Mix different models across agents. Switch providers with one config change."
            />
          </div>
        </div>
      </div>
    </section>
  );
}

function ExplainerItem({ title, description }: { title: string; description: string }) {
  return (
    <div className="flex gap-4">
      <div className="mt-1">
        <CheckIcon className="h-5 w-5 text-brand-accent" />
      </div>
      <div>
        <h4 className="font-semibold text-text-primary">{title}</h4>
        <p className="mt-1 text-sm text-text-secondary font-sans leading-relaxed">{description}</p>
      </div>
    </div>
  );
}

/* ------------------------------------------------------------------ */
/*  4. How It Works (3 steps)                                          */
/* ------------------------------------------------------------------ */

function HowItWorksSection({ onImageClick }: { onImageClick: (src: string) => void }) {
  return (
    <section className="py-24 px-4 border-t border-border bg-surface-alt">
      <div className="max-w-5xl mx-auto">
        <h2 className="text-3xl font-bold tracking-tight text-center text-text-primary mb-4">
          From zero to AI agent in 5 minutes
        </h2>
        <p className="text-center text-text-secondary font-sans mb-14">
          No backend code required. Configure visually or in YAML.
        </p>

        <div className="space-y-16">
          {/* Step 1: Configure */}
          <div className="grid grid-cols-1 lg:grid-cols-2 gap-8 items-start">
            <div>
              <StepBadge n={1} />
              <h3 className="mt-3 text-xl font-semibold text-text-primary">Configure</h3>
              <p className="mt-2 text-text-secondary font-sans leading-relaxed">
                Open the Admin Dashboard and create your agent visually — pick a model, write a system prompt, enable tools. Or define everything in a YAML file if you prefer code.
              </p>
            </div>
            <div className="space-y-4">
              {/* Admin Dashboard */}
              <div className="rounded-[2px] border border-brand-accent/30 bg-brand-accent/5 p-4">
                <p className="text-xs font-semibold text-brand-accent uppercase tracking-wider mb-2">Admin Dashboard</p>
                <p className="text-sm text-text-secondary font-sans leading-relaxed mb-3">
                  Open the Admin Dashboard and create your agent visually. Set the model, system prompt, tools, and spawn rules — no YAML needed.
                </p>
                <div className="rounded-[2px] border border-border overflow-hidden">
                  <img src="/screenshots/admin-agents.png" alt="Admin Dashboard — Agents list with Create Agent button" className="w-full cursor-pointer hover:opacity-80 transition-opacity" onClick={() => onImageClick('/screenshots/admin-agents.png')} />
                </div>
              </div>

              {/* OR separator */}
              <div className="flex items-center gap-3">
                <div className="flex-1 h-px bg-border" />
                <span className="text-xs text-text-tertiary uppercase tracking-wider">or</span>
                <div className="flex-1 h-px bg-border" />
              </div>

              {/* YAML (secondary option) */}
              <div className="rounded-[2px] border border-border bg-surface p-4 opacity-60">
                <p className="text-xs font-semibold text-text-tertiary uppercase tracking-wider mb-2">Or use YAML</p>
                <pre className="font-mono text-sm text-text-secondary leading-relaxed overflow-x-auto">
                  <span className="text-text-tertiary"># agents.yaml</span>{'\n'}
                  <span className="text-brand-accent">agents</span>:{'\n'}
                  {'  '}<span className="text-brand-accent">my-agent</span>:{'\n'}
                  {'    '}<span className="text-text-tertiary">model</span>: <span className="text-emerald-400">glm-5</span>{'\n'}
                  {'    '}<span className="text-text-tertiary">system</span>: <span className="text-emerald-400">"You are a helpful assistant"</span>{'\n'}
                  {'    '}<span className="text-text-tertiary">tools</span>:{'\n'}
                  {'      '}- <span className="text-emerald-400">web_search</span>
                </pre>
              </div>
            </div>
          </div>

          {/* Step 2: Deploy */}
          <div className="grid grid-cols-1 lg:grid-cols-2 gap-8 items-center">
            <div>
              <StepBadge n={2} />
              <h3 className="mt-3 text-xl font-semibold text-text-primary">Deploy</h3>
              <p className="mt-2 text-text-secondary font-sans leading-relaxed">
                One command. Docker Compose pulls the engine and PostgreSQL, starts everything. Your agent is live.
              </p>
            </div>
            <TerminalBlock command="docker compose up -d">
              <div style={{ color: '#87867F' }}>Creating bytebrew-postgres ... done{'\n'}Creating bytebrew-engine  ... done</div>
              <div className="mt-3">
                <span style={{ color: '#87867F' }}>$ </span>
                <span style={{ color: '#4ade80' }}>curl localhost:8443/api/v1/health</span>
              </div>
              <div style={{ color: '#87867F' }}>{'{"status":"ok"}'}</div>
            </TerminalBlock>
          </div>

          {/* Step 3: Integrate */}
          <div className="grid grid-cols-1 lg:grid-cols-2 gap-8 items-start">
            <div>
              <StepBadge n={3} />
              <h3 className="mt-3 text-xl font-semibold text-text-primary">Integrate</h3>
              <p className="mt-2 text-text-secondary font-sans leading-relaxed">
                Send a message, get a streaming response. One HTTP call from any frontend.
              </p>
            </div>
            <TerminalBlock command={'curl -N localhost:8443/api/v1/agents/my-agent/chat \\\n  -d \'{"message":"Hello"}\''}>
              <div className="mt-1" style={{ color: '#87867F' }}>
                {'event: message_delta'}{'\n'}
                {'data: {"content":"Hello! How can"}'}{'\n'}
                {'\n'}
                {'event: message_delta'}{'\n'}
                {'data: {"content":" I help you?"}'}{'\n'}
                {'\n'}
                {'event: done'}{'\n'}
                {'data: {"session_id":"abc-123"}'}
              </div>
            </TerminalBlock>
          </div>
        </div>
      </div>
    </section>
  );
}

function StepBadge({ n }: { n: number }) {
  return (
    <span className="inline-flex items-center justify-center w-8 h-8 rounded-full bg-brand-accent/15 text-brand-accent text-sm font-bold">
      {n}
    </span>
  );
}

/* ------------------------------------------------------------------ */
/*  5. Key Capabilities (6 cards, 2x3)                                 */
/* ------------------------------------------------------------------ */

const CAPABILITIES = [
  {
    title: 'Multiple agents, one goal',
    description: 'Supervisor delegates to specialists — research, writing, QA — all coordinated automatically.',
  },
  {
    title: 'Connect any tool or API',
    description: '20+ built-in tools plus any external service via MCP or YAML declarations.',
  },
  {
    title: 'Agents that work while you sleep',
    description: 'Cron schedules and webhook triggers for automated lead scoring, reports, monitoring.',
  },
  {
    title: 'Your docs become agent memory',
    description: 'Upload PDFs or markdown. Agents search via vector similarity with per-agent isolation.',
  },
  {
    title: 'Plug into any app in minutes',
    description: 'Standard REST API with real-time SSE streaming. No SDK required.',
  },
  {
    title: 'Your data never leaves your servers',
    description: 'Self-hosted with BYOK, API key scopes, and audit logging.',
  },
];

function CapabilitiesSection() {
  return (
    <section className="py-28 px-4 border-t border-border bg-surface">
      <div className="max-w-6xl mx-auto">
        <h2 className="text-3xl font-bold tracking-tight text-center text-text-primary mb-4">
          Built-in, not bolted on
        </h2>
        <p className="text-center text-text-secondary font-sans mb-14 max-w-2xl mx-auto">
          Every feature your AI agent needs — included in one container.
        </p>
        <div className="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-3 gap-6">
          {CAPABILITIES.map((c) => (
            <div
              key={c.title}
              className="rounded-[2px] border border-border bg-surface-alt p-5 transition-all duration-200 hover:border-border-hover hover:-translate-y-0.5"
            >
              <h3 className="text-base font-semibold text-text-primary">{c.title}</h3>
              <p className="mt-2 text-sm text-text-secondary font-sans leading-relaxed">
                {c.description}
              </p>
            </div>
          ))}
        </div>
      </div>
    </section>
  );
}

/* ------------------------------------------------------------------ */
/*  5b. Production-Ready (trust signals + doc links)                   */
/* ------------------------------------------------------------------ */

function ProductionReadySection() {
  const items = [
    {
      title: 'Session persistence',
      description: 'PostgreSQL-backed. Crash mid-conversation? Resumes from last checkpoint.',
    },
    {
      title: 'Structured logging',
      description: 'Every step, tool call, and LLM request logged with trace IDs.',
    },
    {
      title: 'Production deployment guide',
      description: 'Caddy reverse proxy, systemd, SSL/TLS — full ops runbook included.',
      link: '/docs/deployment/production/',
    },
    {
      title: 'Prometheus metrics (EE)',
      description: 'Request latency, token usage, error rates. Alert before users notice.',
    },
  ];

  return (
    <section className="py-24 px-4 border-t border-border bg-surface">
      <div className="max-w-5xl mx-auto">
        <h2 className="text-3xl font-bold tracking-tight text-center text-text-primary">
          Built for production, not just demos
        </h2>
        <p className="text-center text-text-secondary font-sans mt-3 max-w-2xl mx-auto">
          Everything you need to run AI agents reliably in production — logging, persistence, security, and monitoring.
        </p>
        <div className="grid grid-cols-1 sm:grid-cols-2 gap-5 mt-12">
          {items.map((item) => (
            <div key={item.title} className="rounded-[2px] border border-border bg-surface-alt p-5">
              <h3 className="text-sm font-semibold text-text-primary">{item.title}</h3>
              <p className="mt-1.5 text-sm text-text-secondary font-sans leading-relaxed">{item.description}</p>
              {item.link && (
                <a href={item.link} className="mt-2 inline-block text-xs text-brand-accent hover:underline">
                  Read the guide →
                </a>
              )}
            </div>
          ))}
        </div>
      </div>
    </section>
  );
}

/* ------------------------------------------------------------------ */
/*  6. Use Cases (4 detailed cards with icons + outcomes)              */
/* ------------------------------------------------------------------ */

const USE_CASES = [
  {
    title: 'AI Support Assistant',
    description:
      'User asks \'why was I charged twice?\' Agent checks your billing API, finds the duplicate charge, initiates a refund, and replies with confirmation — no human touched this ticket. Connect your docs via RAG, add your APIs as tools, deploy.',
    outcome: 'Automate customer interactions at scale',
    icon: (
      <svg className="h-6 w-6 text-brand-accent" fill="none" viewBox="0 0 24 24" strokeWidth={1.5} stroke="currentColor">
        <path strokeLinecap="round" strokeLinejoin="round" d="M20.25 8.511c.884.284 1.5 1.128 1.5 2.097v4.286c0 1.136-.847 2.1-1.98 2.193-.34.027-.68.052-1.02.072v3.091l-3-3c-1.354 0-2.694-.055-4.02-.163a2.115 2.115 0 01-.825-.242m9.345-8.334a2.126 2.126 0 00-.476-.095 48.64 48.64 0 00-8.048 0c-1.131.094-1.976 1.057-1.976 2.192v4.286c0 .837.46 1.58 1.155 1.951m9.345-8.334V6.637c0-1.621-1.152-3.026-2.76-3.235A48.455 48.455 0 0011.25 3c-2.115 0-4.198.137-6.24.402-1.608.209-2.76 1.614-2.76 3.235v6.226c0 1.621 1.152 3.026 2.76 3.235.577.075 1.157.14 1.74.194V21l4.155-4.155" />
      </svg>
    ),
  },
  {
    title: 'Operations Autopilot',
    description:
      'Incident triage, log analysis, automated remediation — agents monitor and act around the clock.',
    outcome: 'Autonomous operations, not reactive scripts',
    icon: (
      <svg className="h-6 w-6 text-brand-accent" fill="none" viewBox="0 0 24 24" strokeWidth={1.5} stroke="currentColor">
        <path strokeLinecap="round" strokeLinejoin="round" d="M16.023 9.348h4.992v-.001M2.985 19.644v-4.992m0 0h4.992m-4.993 0l3.181 3.183a8.25 8.25 0 0013.803-3.7M4.031 9.865a8.25 8.25 0 0113.803-3.7l3.181 3.182m0-4.991v4.99" />
      </svg>
    ),
  },
  {
    title: 'Research & Analysis Agents',
    description:
      'Multi-step lead research, competitor analysis, report generation. Each step adapts dynamically.',
    outcome: 'AI reasoning replaces rigid pipelines',
    icon: (
      <svg className="h-6 w-6 text-brand-accent" fill="none" viewBox="0 0 24 24" strokeWidth={1.5} stroke="currentColor">
        <path strokeLinecap="round" strokeLinejoin="round" d="M3.75 6A2.25 2.25 0 016 3.75h2.25A2.25 2.25 0 0110.5 6v2.25a2.25 2.25 0 01-2.25 2.25H6a2.25 2.25 0 01-2.25-2.25V6zM3.75 15.75A2.25 2.25 0 016 13.5h2.25a2.25 2.25 0 012.25 2.25V18a2.25 2.25 0 01-2.25 2.25H6A2.25 2.25 0 013.75 18v-2.25zM13.5 6a2.25 2.25 0 012.25-2.25H18A2.25 2.25 0 0120.25 6v2.25A2.25 2.25 0 0118 10.5h-2.25a2.25 2.25 0 01-2.25-2.25V6zM13.5 15.75a2.25 2.25 0 012.25-2.25H18a2.25 2.25 0 012.25 2.25V18A2.25 2.25 0 0118 20.25h-2.25A2.25 2.25 0 0113.5 18v-2.25z" />
      </svg>
    ),
  },
  {
    title: 'Knowledge-Powered Assistants',
    description:
      'Upload docs and policies. Agents answer with accurate, sourced responses.',
    outcome: 'Your knowledge, amplified by AI',
    icon: (
      <svg className="h-6 w-6 text-brand-accent" fill="none" viewBox="0 0 24 24" strokeWidth={1.5} stroke="currentColor">
        <path strokeLinecap="round" strokeLinejoin="round" d="M4.26 10.147a60.436 60.436 0 00-.491 6.347A48.627 48.627 0 0112 20.904a48.627 48.627 0 018.232-4.41 60.46 60.46 0 00-.491-6.347m-15.482 0a50.57 50.57 0 00-2.658-.813A59.905 59.905 0 0112 3.493a59.902 59.902 0 0110.399 5.84c-.896.248-1.783.52-2.658.814m-15.482 0A50.697 50.697 0 0112 13.489a50.702 50.702 0 017.74-3.342M6.75 15a.75.75 0 100-1.5.75.75 0 000 1.5zm0 0v-3.675A55.378 55.378 0 0112 8.443m-7.007 11.55A5.981 5.981 0 006.75 15.75v-1.5" />
      </svg>
    ),
  },
];

/* ------------------------------------------------------------------ */
/*  Product Showcase — real screenshots of the web client               */
/* ------------------------------------------------------------------ */

function ProductShowcaseSection({ onImageClick }: { onImageClick: (src: string) => void }) {
  return (
    <section className="py-24 px-4 border-t border-border bg-surface-alt">
      <div className="max-w-6xl mx-auto">
        <h2 className="text-3xl font-bold tracking-tight text-center text-text-primary mb-4">
          See it in action
        </h2>
        <p className="text-center text-text-secondary font-sans mb-14 max-w-2xl mx-auto">
          ByteBrew comes with a ready-to-use web client and a full Admin Dashboard. Use them as-is, or build your own UI on top of the REST API.
        </p>

        {/* Two showcases in a grid */}
        <div className="grid grid-cols-1 lg:grid-cols-2 gap-8 mb-12">
          {/* Web Client */}
          <div>
            <h3 className="text-lg font-semibold text-text-primary mb-2">Web Client — Chat Interface</h3>
            <p className="text-sm text-text-secondary font-sans leading-relaxed mb-4">
              Chat interface with multi-agent sidebar, tool calls, and markdown. Included with Engine.
            </p>
            <div className="rounded-[2px] border border-border overflow-hidden shadow-2xl shadow-brand-accent/5">
              <img src="/screenshots/admin-agents.png" alt="ByteBrew Web Client — AI sales agent recommending laptops with web search tool calls and structured markdown response" className="w-full cursor-pointer hover:opacity-80 transition-opacity" onClick={() => onImageClick('/screenshots/admin-agents.png')} />
            </div>
          </div>

          {/* Admin Dashboard */}
          <div>
            <h3 className="text-lg font-semibold text-text-primary mb-2">Admin Dashboard — Visual Management</h3>
            <p className="text-sm text-text-secondary font-sans leading-relaxed mb-4">
              Configure agents, models, MCP servers, triggers, and API keys visually.
            </p>
            <div className="rounded-[2px] border border-border overflow-hidden shadow-2xl shadow-brand-accent/5">
              <img src="/screenshots/admin-agent-detail.png" alt="Admin Dashboard — Agent detail panel with model, system prompt, tools, and spawn rules configuration" className="w-full cursor-pointer hover:opacity-80 transition-opacity" onClick={() => onImageClick('/screenshots/admin-agent-detail.png')} />
            </div>
          </div>
        </div>

        {/* Feature highlights grid */}
        <div className="grid grid-cols-1 md:grid-cols-3 gap-6">
          <div className="rounded-[2px] border border-border bg-surface p-5 transition-all duration-200 hover:border-border-hover hover:-translate-y-0.5">
            <h3 className="text-sm font-semibold text-text-primary mb-2">Multi-Agent Sidebar</h3>
            <p className="text-xs text-text-secondary font-sans leading-relaxed">
              Switch between agents instantly with dedicated session history and inline tool calls.
            </p>
          </div>
          <div className="rounded-[2px] border border-border bg-surface p-5 transition-all duration-200 hover:border-border-hover hover:-translate-y-0.5">
            <h3 className="text-sm font-semibold text-text-primary mb-2">Rich Responses</h3>
            <p className="text-xs text-text-secondary font-sans leading-relaxed">
              Full markdown with code blocks, tables, and expandable tool call results.
            </p>
          </div>
          <div className="rounded-[2px] border border-border bg-surface p-5 transition-all duration-200 hover:border-border-hover hover:-translate-y-0.5">
            <h3 className="text-sm font-semibold text-text-primary mb-2">Visual Agent Editor</h3>
            <p className="text-xs text-text-secondary font-sans leading-relaxed">
              Configure agents, models, prompts, and spawn rules — no YAML needed.
            </p>
          </div>
        </div>
      </div>
    </section>
  );
}

function UseCasesSection() {
  return (
    <section className="py-28 px-4 border-t border-border bg-surface-alt">
      <div className="max-w-6xl mx-auto">
        <h2 className="text-3xl font-bold tracking-tight text-center text-text-primary mb-4">
          What teams build with ByteBrew
        </h2>
        <p className="text-center text-text-secondary font-sans mb-14 max-w-2xl mx-auto">
          Real use cases, running in production.
        </p>
        <div className="grid grid-cols-1 sm:grid-cols-2 gap-6">
          {USE_CASES.map((uc) => (
            <div
              key={uc.title}
              className="rounded-[2px] border border-border bg-surface p-6 transition-all duration-200 hover:border-border-hover hover:-translate-y-0.5"
            >
              <div className="w-12 h-12 rounded-[2px] bg-brand-accent/10 flex items-center justify-center mb-4">
                {uc.icon}
              </div>
              <h3 className="text-lg font-semibold text-text-primary">{uc.title}</h3>
              <p className="mt-2 text-sm text-text-secondary font-sans leading-relaxed">
                {uc.description}
              </p>
              <p className="mt-4 text-xs font-medium text-brand-accent">
                {uc.outcome}
              </p>
            </div>
          ))}
        </div>
      </div>
    </section>
  );
}

/* ------------------------------------------------------------------ */
/*  7. Comparison Table                                                */
/* ------------------------------------------------------------------ */

const APPROACH_COMPARISON = [
  { approach: 'Cloud AI platforms', problem: 'Per-token pricing, data leaves your servers, locked to one provider', bytebrew: 'Self-hosted with your own API keys. Pay only your LLM provider directly — no markup' },
  { approach: 'Agent SDKs / frameworks', problem: 'You get a library, not a product. No API server, no admin UI, no scheduling', bytebrew: 'Complete runtime with REST API, admin dashboard, cron triggers, and session management' },
  { approach: 'Visual AI builders', problem: 'Simple chatbots only. No autonomous reasoning, no tool calling, no sub-agents', bytebrew: 'Multi-step reasoning agents that delegate tasks, call tools, and work autonomously' },
  { approach: 'Single-model APIs', problem: 'One provider, no orchestration, no memory, no background jobs', bytebrew: 'Mix any models across agents. Built-in RAG, sessions, and background automation' },
  { approach: 'Custom in-house solutions', problem: '3-6 months to build, ongoing maintenance, team distracted from core product', bytebrew: 'Production-ready in 5 minutes. We maintain the engine — you focus on your product' },
];

function ComparisonSection() {
  return (
    <section className="py-24 px-4 border-t border-border bg-surface">
      <div className="max-w-5xl mx-auto">
        <h2 className="text-3xl font-bold tracking-tight text-center text-text-primary mb-4">
          How ByteBrew compares
        </h2>
        <p className="text-center text-text-secondary font-sans mb-14">
          Every approach has trade-offs. Here&apos;s where ByteBrew fits.
        </p>

        <div className="overflow-x-auto rounded-[2px] border border-border">
          <table className="w-full text-base">
            <thead>
              <tr className="border-b border-border bg-surface-alt">
                <th className="text-left py-3 px-4 font-medium text-text-tertiary">Traditional Approach</th>
                <th className="text-left py-3 px-4 font-medium text-text-tertiary">The Problem</th>
                <th className="text-left py-3 px-4 font-bold text-brand-accent bg-brand-accent/5">ByteBrew Engine</th>
              </tr>
            </thead>
            <tbody>
              {APPROACH_COMPARISON.map((row, i) => (
                <tr
                  key={row.approach}
                  className={i < APPROACH_COMPARISON.length - 1 ? 'border-b border-border' : ''}
                >
                  <td className="py-3 px-4 text-text-secondary">{row.approach}</td>
                  <td className="py-3 px-4 text-text-tertiary">{row.problem}</td>
                  <td className="py-3 px-4 text-text-primary bg-brand-accent/5">{row.bytebrew}</td>
                </tr>
              ))}
            </tbody>
          </table>
        </div>
      </div>
    </section>
  );
}

/* ------------------------------------------------------------------ */
/*  8a. MCP Docs — removed from landing, available on /docs & /download */
/* ------------------------------------------------------------------ */

/*
const MCP_TABS = [
  { key: 'claude' as const, label: 'Claude Code' },
  { key: 'codex' as const, label: 'Codex' },
  { key: 'other' as const, label: 'Other' },
] as const;

const MCP_CODE: Record<'claude' | 'codex' | 'other', { code: string; prefix?: string }> = {
  claude: {
    code: 'claude mcp add bytebrew-docs --transport sse https://mcp.bytebrew.ai/sse',
    prefix: '$',
  },
  codex: {
    code: 'codex mcp add bytebrew-docs https://mcp.bytebrew.ai/sse',
    prefix: '$',
  },
  other: {
    code: 'MCP SSE endpoint: https://mcp.bytebrew.ai/sse',
    prefix: '#',
  },
};

function MCPDocsSection() {
  return null;
}
*/

/* ------------------------------------------------------------------ */
/*  8b. Install / Quick Start                                          */
/* ------------------------------------------------------------------ */

function InstallSection() {
  return (
    <section className="py-24 px-4 border-t border-border bg-surface-alt">
      <div className="max-w-3xl mx-auto">
        <h2 className="text-3xl font-bold tracking-tight text-center text-text-primary mb-2">
          Get Started in 30 Seconds
        </h2>
        <p className="text-center text-text-secondary font-sans mb-10">
          No PostgreSQL? No problem &mdash; it&apos;s included.
        </p>

        <TerminalBlock
          command="curl -fsSL https://bytebrew.ai/releases/docker-compose.yml -o docker-compose.yml && docker compose up -d"
        />
        <p className="text-center text-sm font-sans font-medium text-brand-accent mt-4">Community Edition is free forever — no limits, no credit card.</p>

        <div className="bg-surface-alt border border-border rounded-[2px] p-6 mt-6 max-w-3xl mx-auto">
          <p className="text-xs font-semibold text-text-tertiary uppercase tracking-wider mb-4">What happens next</p>
          <ol className="text-sm text-text-secondary space-y-2 list-decimal list-inside">
            <li>Open localhost:8443 — you'll see the Admin Dashboard</li>
            <li>Add your OpenAI / Gemini / Claude API key</li>
            <li>Create an agent — name, system prompt, model</li>
            <li>Send your first message via curl or the built-in Web Client</li>
            <li>Watch the agent think and respond in real time</li>
          </ol>
          <p className="mt-4 text-xs text-text-tertiary">
            Total time: under 5 minutes. No config files. No backend code.
          </p>
        </div>

        <p className="mt-6 text-center text-sm text-text-tertiary">
          Already have PostgreSQL?{' '}
          <Link
            to="/download"
            className="text-brand-accent hover:underline transition-colors"
          >
            See all install options &rarr;
          </Link>
        </p>
      </div>
    </section>
  );
}

/* ------------------------------------------------------------------ */
/*  9. Pricing                                                         */
/* ------------------------------------------------------------------ */

function PricingSection() {
  return (
    <section id="pricing" className="py-24 px-4 border-t border-border bg-surface">
      <div className="max-w-5xl mx-auto">
        <h2 className="text-3xl font-bold tracking-tight text-center text-text-primary mb-4">
          Simple, Transparent Pricing
        </h2>
        <p className="text-center text-text-secondary font-sans mb-14">
          Start free. Scale when you need observability and compliance.
        </p>
        <EnginePricingTable />
      </div>
    </section>
  );
}

/* ------------------------------------------------------------------ */
/*  10. Free Forever Banner                                            */
/* ------------------------------------------------------------------ */

function FreeForeverBanner() {
  return (
    <section className="py-24 px-4 border-t border-border bg-surface-alt">
      <div className="max-w-3xl mx-auto rounded-[2px] border border-brand-accent/30 bg-brand-accent/5 p-10 text-center">
        <h2 className="text-2xl font-bold tracking-tight text-text-primary">
          ByteBrew Community Edition is free forever.
        </h2>
        <p className="mt-4 text-text-secondary font-sans leading-relaxed">
          No agent limits. No session limits. No time limits. No credit card required.
          <br />
          Community Edition features will never move behind a paywall — we promise.
        </p>
        <p className="mt-2 text-sm text-text-tertiary">
          Build your product with confidence.
        </p>
      </div>
    </section>
  );
}

/* ------------------------------------------------------------------ */
/*  11. Final CTA                                                      */
/* ------------------------------------------------------------------ */

function FinalCTASection() {
  return (
    <section className="py-24 px-4 border-t border-border bg-surface">
      <div className="max-w-2xl mx-auto text-center">
        <h2 className="text-3xl font-bold text-text-primary">
          Add AI agents to your product today
        </h2>
        <p className="mt-4 text-text-secondary font-sans">
          One Docker command. Full AI agent runtime. Works with any LLM. Free forever.
        </p>
        <Link
          to="/download"
          className="mt-8 inline-block rounded-[2px] bg-brand-accent px-8 py-3.5 text-sm font-medium text-white hover:bg-brand-accent-hover transition-colors"
        >
          Get Started Free
        </Link>
      </div>
    </section>
  );
}
