import { useState, useEffect } from 'react';
import { Link } from '@tanstack/react-router';
import { TerminalBlock } from '../components/TerminalBlock';
import { EnginePricingTable } from '../components/EnginePricingTable';

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
      <MCPDocsSection />
      <ProblemSection />
      <SolutionSection />
      <HowItWorksSection onImageClick={setLightboxSrc} />
      <CapabilitiesSection />
      <ProductShowcaseSection onImageClick={setLightboxSrc} />
      <UseCasesSection />
      <ComparisonSection />
      <InstallSection />
      <PricingSection />
      <FreeForeverBanner />
      <FinalCTASection />

      {lightboxSrc && (
        <div
          className="fixed inset-0 z-50 flex items-center justify-center bg-black/80 backdrop-blur-sm cursor-pointer"
          onClick={() => setLightboxSrc(null)}
        >
          <img
            src={lightboxSrc}
            alt=""
            className="max-w-[90vw] max-h-[90vh] object-contain rounded-lg shadow-2xl"
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
    <section className="py-24 px-4 text-center bg-brand-dark">
      <div className="max-w-4xl mx-auto">
        <h1 className="text-5xl sm:text-6xl font-bold tracking-tight text-brand-light leading-tight">
          Add an AI agent to{' '}
          <span className="text-brand-accent">your product</span>
        </h1>
        <p className="mt-6 text-xl text-brand-shade2 leading-relaxed max-w-2xl mx-auto">
          Ship autonomous AI agents in minutes, not months.
          <br />
          One Docker command. Any LLM. No vendor lock-in.
        </p>
        <div className="mt-10 flex items-center justify-center gap-4 flex-wrap">
          <Link
            to="/download"
            className="rounded-[10px] bg-brand-accent px-7 py-3.5 text-sm font-medium text-white hover:bg-brand-accent-hover transition-colors"
          >
            Get Started Free &rarr;
          </Link>
          <Link
            to="/pricing"
            className="rounded-[10px] border border-brand-shade3/20 px-7 py-3.5 text-sm font-medium text-brand-shade2 hover:border-brand-shade3/40 hover:text-brand-light transition-colors"
          >
            View Pricing &rarr;
          </Link>
        </div>

        {/* Terminal preview: YAML config + docker compose */}
        <div className="mt-16 max-w-2xl mx-auto text-left">
          <div className="relative rounded-[12px] border border-brand-shade3/15 bg-brand-dark-alt p-5 overflow-x-auto">
            <div className="flex gap-1.5 mb-4">
              <span className="w-3 h-3 rounded-full bg-red-500/80" />
              <span className="w-3 h-3 rounded-full bg-yellow-500/80" />
              <span className="w-3 h-3 rounded-full bg-green-500/80" />
            </div>
            <pre className="font-mono text-sm text-brand-shade2 leading-relaxed">
              <span className="text-brand-shade3"># agents.yaml</span>{'\n'}
              <span className="text-brand-accent">agents</span>:{'\n'}
              {'  '}<span className="text-brand-accent">sales-agent</span>:{'\n'}
              {'    '}<span className="text-brand-shade3">model</span>: <span className="text-emerald-400">glm-5</span>{'\n'}
              {'    '}<span className="text-brand-shade3">system</span>: <span className="text-emerald-400">"You are a sales consultant..."</span>{'\n'}
              {'    '}<span className="text-brand-shade3">tools</span>:{'\n'}
              {'      '}- <span className="text-emerald-400">product_search</span>{'\n'}
              {'      '}- <span className="text-emerald-400">check_inventory</span>{'\n'}
              {'      '}- <span className="text-emerald-400">create_order</span>{'\n'}
              {'    '}<span className="text-brand-shade3">mcp_servers</span>:{'\n'}
              {'      '}- <span className="text-emerald-400">crm-api</span>{'\n'}
              {'\n'}
              <span className="text-brand-shade3">$ docker compose up -d</span>{'\n'}
              <span className="text-emerald-400">Creating bytebrew-engine  ... done</span>{'\n'}
              <span className="text-emerald-400">Creating bytebrew-postgres ... done</span>{'\n'}
              <span className="text-brand-shade3"># Agent running on http://localhost:8443</span>
            </pre>
          </div>
        </div>
      </div>
    </section>
  );
}

/* ------------------------------------------------------------------ */
/*  2. Problem Statement                                               */
/* ------------------------------------------------------------------ */

const PROBLEMS = [
  {
    title: 'Months of Development',
    description:
      'Building agent infrastructure from scratch takes 3-6 months. Multi-agent orchestration, tool integration, state management, streaming APIs \u2014 all before your first user interaction.',
    icon: (
      <svg className="h-6 w-6 text-brand-accent" fill="none" viewBox="0 0 24 24" strokeWidth={1.5} stroke="currentColor">
        <path strokeLinecap="round" strokeLinejoin="round" d="M12 6v6h4.5m4.5 0a9 9 0 11-18 0 9 9 0 0118 0z" />
      </svg>
    ),
  },
  {
    title: 'Vendor Lock-in',
    description:
      'Cloud-only AI platforms lock you into per-token pricing. $600+/month for 1,000 dialogues. No self-hosting option. Your data on their servers.',
    icon: (
      <svg className="h-6 w-6 text-brand-accent" fill="none" viewBox="0 0 24 24" strokeWidth={1.5} stroke="currentColor">
        <path strokeLinecap="round" strokeLinejoin="round" d="M16.5 10.5V6.75a4.5 4.5 0 10-9 0v3.75m-.75 11.25h10.5a2.25 2.25 0 002.25-2.25v-6.75a2.25 2.25 0 00-2.25-2.25H6.75a2.25 2.25 0 00-2.25 2.25v6.75a2.25 2.25 0 002.25 2.25z" />
      </svg>
    ),
  },
  {
    title: 'Integration Complexity',
    description:
      'SDKs require dedicated engineering teams. Visual builders produce reactive chatbots, not autonomous daemon agents that work around the clock.',
    icon: (
      <svg className="h-6 w-6 text-brand-accent" fill="none" viewBox="0 0 24 24" strokeWidth={1.5} stroke="currentColor">
        <path strokeLinecap="round" strokeLinejoin="round" d="M14.25 9.75L16.5 12l-2.25 2.25m-4.5 0L7.5 12l2.25-2.25M6 20.25h12A2.25 2.25 0 0020.25 18V6A2.25 2.25 0 0018 3.75H6A2.25 2.25 0 003.75 6v12A2.25 2.25 0 006 20.25z" />
      </svg>
    ),
  },
];

function ProblemSection() {
  return (
    <section className="py-20 px-4 border-t border-brand-shade3/15 bg-brand-dark-alt">
      <div className="max-w-5xl mx-auto">
        <h2 className="text-3xl font-bold text-center text-brand-light mb-4">
          Building AI agents is hard. It shouldn&apos;t be.
        </h2>
        <p className="text-center text-brand-shade3 mb-14 max-w-2xl mx-auto">
          Every team building AI agents hits the same walls.
        </p>
        <div className="grid grid-cols-1 md:grid-cols-3 gap-8">
          {PROBLEMS.map((p) => (
            <div
              key={p.title}
              className="rounded-[12px] border border-brand-shade3/15 bg-brand-dark p-6"
            >
              <div className="w-12 h-12 rounded-[10px] bg-brand-accent/10 flex items-center justify-center mb-4">
                {p.icon}
              </div>
              <h3 className="text-lg font-semibold text-brand-light">{p.title}</h3>
              <p className="mt-3 text-sm text-brand-shade3 leading-relaxed">
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
    <section className="py-20 px-4 border-t border-brand-shade3/15 bg-brand-dark">
      <div className="max-w-6xl mx-auto">
        <h2 className="text-3xl font-bold text-center text-brand-light mb-4">
          ByteBrew Engine: AI agent infrastructure that just works
        </h2>
        <p className="text-center text-brand-shade3 mb-14 max-w-2xl mx-auto">
          Configure visually or in YAML. Deploy with Docker. Scale when ready.
        </p>

        <div className="grid grid-cols-1 lg:grid-cols-2 gap-10 items-start">
          {/* YAML config on the left */}
          <div className="rounded-[12px] border border-brand-shade3/15 bg-brand-dark-alt p-5 overflow-x-auto">
            <div className="flex gap-1.5 mb-4">
              <span className="w-3 h-3 rounded-full bg-red-500/80" />
              <span className="w-3 h-3 rounded-full bg-yellow-500/80" />
              <span className="w-3 h-3 rounded-full bg-green-500/80" />
            </div>
            <pre className="font-mono text-sm text-brand-shade2 leading-relaxed">
              <span className="text-brand-shade3"># Full agent configuration</span>{'\n'}
              <span className="text-brand-accent">agents</span>:{'\n'}
              {'  '}<span className="text-brand-accent">supervisor</span>:{'\n'}
              {'    '}<span className="text-brand-shade3">model</span>: <span className="text-emerald-400">glm-5</span>{'\n'}
              {'    '}<span className="text-brand-shade3">system</span>: <span className="text-emerald-400">"Route customer requests"</span>{'\n'}
              {'    '}<span className="text-brand-shade3">can_spawn</span>:{'\n'}
              {'      '}- <span className="text-emerald-400">sales-agent</span>{'\n'}
              {'      '}- <span className="text-emerald-400">support-agent</span>{'\n'}
              {'\n'}
              {'  '}<span className="text-brand-accent">sales-agent</span>:{'\n'}
              {'    '}<span className="text-brand-shade3">model</span>: <span className="text-emerald-400">qwen-3-32b</span>{'\n'}
              {'    '}<span className="text-brand-shade3">tools</span>:{'\n'}
              {'      '}- <span className="text-emerald-400">product_search</span>{'\n'}
              {'      '}- <span className="text-emerald-400">create_order</span>{'\n'}
              {'    '}<span className="text-brand-shade3">mcp_servers</span>:{'\n'}
              {'      '}- <span className="text-emerald-400">crm-api</span>{'\n'}
              {'\n'}
              {'  '}<span className="text-brand-accent">support-agent</span>:{'\n'}
              {'    '}<span className="text-brand-shade3">model</span>: <span className="text-emerald-400">claude-sonnet-4</span>{'\n'}
              {'    '}<span className="text-brand-shade3">tools</span>:{'\n'}
              {'      '}- <span className="text-emerald-400">web_search</span>{'\n'}
              {'      '}- <span className="text-emerald-400">create_ticket</span>{'\n'}
              {'\n'}
              <span className="text-brand-accent">triggers</span>:{'\n'}
              {'  '}<span className="text-brand-shade3">daily-report</span>:{'\n'}
              {'    '}<span className="text-brand-shade3">cron</span>: <span className="text-emerald-400">"0 9 * * *"</span>{'\n'}
              {'    '}<span className="text-brand-shade3">agent</span>: <span className="text-emerald-400">supervisor</span>{'\n'}
              {'    '}<span className="text-brand-shade3">message</span>: <span className="text-emerald-400">"Generate daily report"</span>
            </pre>
          </div>

          {/* Explanation on the right */}
          <div className="space-y-6">
            <ExplainerItem
              title="Multi-agent hierarchy"
              description="Define supervisor agents that spawn sub-agents on demand. Each agent has its own model, tools, and system prompt."
            />
            <ExplainerItem
              title="Universal tool system"
              description="Connect any API through MCP servers or declarative YAML tools. No backend code required. Per-agent security isolation."
            />
            <ExplainerItem
              title="Built-in triggers"
              description="Cron schedules, webhooks, and background tasks. Your agents work 24/7 without polling or external orchestration."
            />
            <ExplainerItem
              title="Model-agnostic"
              description="Mix models across agents. GLM-5 for the supervisor, Claude Sonnet for support, local Qwen for internal tasks. Any OpenAI-compatible provider."
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
        <h4 className="font-semibold text-brand-light">{title}</h4>
        <p className="mt-1 text-sm text-brand-shade3 leading-relaxed">{description}</p>
      </div>
    </div>
  );
}

/* ------------------------------------------------------------------ */
/*  4. How It Works (3 steps)                                          */
/* ------------------------------------------------------------------ */

function HowItWorksSection({ onImageClick }: { onImageClick: (src: string) => void }) {
  return (
    <section className="py-20 px-4 border-t border-brand-shade3/15 bg-brand-dark-alt">
      <div className="max-w-5xl mx-auto">
        <h2 className="text-3xl font-bold text-center text-brand-light mb-4">
          Three steps to your first agent
        </h2>
        <p className="text-center text-brand-shade3 mb-14">
          From zero to production-ready agent in under 5 minutes.
        </p>

        <div className="space-y-16">
          {/* Step 1: Configure */}
          <div className="grid grid-cols-1 lg:grid-cols-2 gap-8 items-start">
            <div>
              <StepBadge n={1} />
              <h3 className="mt-3 text-xl font-semibold text-brand-light">Configure</h3>
              <p className="mt-2 text-brand-shade3 leading-relaxed">
                Create your agent visually through the Admin Dashboard, or define everything in YAML for version control and CI/CD.
              </p>
            </div>
            <div className="space-y-4">
              {/* Option A: Admin Dashboard */}
              <div className="rounded-[12px] border border-brand-accent/30 bg-brand-accent/5 p-4">
                <p className="text-xs font-semibold text-brand-accent uppercase tracking-wider mb-2">Option A: Admin Dashboard</p>
                <p className="text-sm text-brand-shade3 leading-relaxed mb-3">
                  Open the Admin Dashboard and create your agent visually. Set the model, system prompt, tools, and spawn rules — no YAML needed.
                </p>
                <div className="rounded-[8px] border border-brand-shade3/15 overflow-hidden">
                  <img src="/screenshots/admin-agents.png" alt="Admin Dashboard — Agents list with Create Agent button" className="w-full cursor-pointer hover:opacity-80 transition-opacity" onClick={() => onImageClick('/screenshots/admin-agents.png')} />
                </div>
              </div>

              {/* OR separator */}
              <div className="flex items-center gap-3">
                <div className="flex-1 h-px bg-brand-shade3/15" />
                <span className="text-xs font-medium text-brand-shade3 uppercase tracking-wider">or</span>
                <div className="flex-1 h-px bg-brand-shade3/15" />
              </div>

              {/* Option B: YAML */}
              <div className="rounded-[12px] border border-brand-shade3/15 bg-brand-dark p-4">
                <p className="text-xs font-semibold text-brand-shade3 uppercase tracking-wider mb-2">Option B: YAML (GitOps)</p>
                <pre className="font-mono text-sm text-brand-shade2 leading-relaxed overflow-x-auto">
                  <span className="text-brand-shade3"># agents.yaml</span>{'\n'}
                  <span className="text-brand-accent">agents</span>:{'\n'}
                  {'  '}<span className="text-brand-accent">my-agent</span>:{'\n'}
                  {'    '}<span className="text-brand-shade3">model</span>: <span className="text-emerald-400">glm-5</span>{'\n'}
                  {'    '}<span className="text-brand-shade3">system</span>: <span className="text-emerald-400">"You are a helpful assistant"</span>{'\n'}
                  {'    '}<span className="text-brand-shade3">tools</span>:{'\n'}
                  {'      '}- <span className="text-emerald-400">web_search</span>
                </pre>
              </div>
            </div>
          </div>

          {/* Step 2: Deploy */}
          <div className="grid grid-cols-1 lg:grid-cols-2 gap-8 items-center">
            <div>
              <StepBadge n={2} />
              <h3 className="mt-3 text-xl font-semibold text-brand-light">Deploy</h3>
              <p className="mt-2 text-brand-shade3 leading-relaxed">
                One command. Engine + PostgreSQL, ready in 30 seconds. Self-hosted on your infrastructure, your data never leaves your network.
              </p>
            </div>
            <div className="rounded-[12px] border border-brand-shade3/15 bg-brand-dark p-4 overflow-x-auto">
              <pre className="font-mono text-sm text-brand-shade2 leading-relaxed">
                <span className="text-brand-shade3">$</span> <span className="text-brand-light">docker compose up -d</span>{'\n'}
                <span className="text-emerald-400">Creating bytebrew-postgres ... done</span>{'\n'}
                <span className="text-emerald-400">Creating bytebrew-engine  ... done</span>{'\n'}
                {'\n'}
                <span className="text-brand-shade3">$</span> <span className="text-brand-light">curl localhost:8443/api/v1/health</span>{'\n'}
                <span className="text-emerald-400">{'{"status":"ok"}'}</span>
              </pre>
            </div>
          </div>

          {/* Step 3: Integrate */}
          <div className="grid grid-cols-1 lg:grid-cols-2 gap-8 items-center">
            <div>
              <StepBadge n={3} />
              <h3 className="mt-3 text-xl font-semibold text-brand-light">Integrate</h3>
              <p className="mt-2 text-brand-shade3 leading-relaxed">
                REST API + SSE streaming. Send a message, get a real-time agent response. OpenAI-compatible format your team already knows.
              </p>
            </div>
            <div className="rounded-[12px] border border-brand-shade3/15 bg-brand-dark p-4 overflow-x-auto">
              <pre className="font-mono text-sm text-brand-shade2 leading-relaxed">
                <span className="text-brand-shade3">$</span> <span className="text-brand-light">curl -N localhost:8443/api/v1/agents/my-agent/chat \</span>{'\n'}
                {'  '}<span className="text-brand-light">-d '{'"'}message":"Hello"{'"'}'</span>{'\n'}
                {'\n'}
                <span className="text-brand-shade3">{'event: message_delta'}</span>{'\n'}
                <span className="text-brand-shade3">{'data: {"content":"Hello! How can"}'}</span>{'\n'}
                {'\n'}
                <span className="text-brand-shade3">{'event: message_delta'}</span>{'\n'}
                <span className="text-brand-shade3">{'data: {"content":" I help you?"}'}</span>{'\n'}
                {'\n'}
                <span className="text-brand-shade3">{'event: done'}</span>{'\n'}
                <span className="text-brand-shade3">{'data: {"session_id":"abc-123"}'}</span>
              </pre>
            </div>
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
    title: 'Multi-Agent Orchestration',
    description: 'Supervisor agents spawn sub-agents on demand. Persistent and spawn lifecycle modes. Hierarchical delegation.',
  },
  {
    title: 'Universal Tool System',
    description: 'MCP servers + declarative YAML tools. Connect any API without backend code. Per-agent security isolation.',
  },
  {
    title: 'Background Automation',
    description: 'Cron schedules, webhook triggers, background task system. Long-running jobs that work while you sleep.',
  },
  {
    title: 'Knowledge / RAG',
    description: 'Document ingestion and semantic search. Per-agent knowledge isolation. Built-in vector storage.',
  },
  {
    title: 'REST API + SSE + WS',
    description: 'OpenAI-compatible format. Real-time streaming. WebSocket support for persistent connections.',
  },
  {
    title: 'Self-Hosted & Secure',
    description: 'Air-gap capable. BYOK (bring your own keys). Your models, your data, your infrastructure.',
  },
];

function CapabilitiesSection() {
  return (
    <section className="py-20 px-4 border-t border-brand-shade3/15 bg-brand-dark">
      <div className="max-w-6xl mx-auto">
        <h2 className="text-3xl font-bold text-center text-brand-light mb-4">
          Everything you need to ship AI agents
        </h2>
        <p className="text-center text-brand-shade3 mb-14 max-w-2xl mx-auto">
          Production-ready features built into the engine. No plugins, no add-ons.
        </p>
        <div className="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-3 gap-6">
          {CAPABILITIES.map((c) => (
            <div
              key={c.title}
              className="rounded-[12px] border border-brand-shade3/15 bg-brand-dark-alt p-5"
            >
              <h3 className="text-base font-semibold text-brand-light">{c.title}</h3>
              <p className="mt-2 text-sm text-brand-shade3 leading-relaxed">
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
/*  6. Use Cases (4 detailed cards with icons + outcomes)              */
/* ------------------------------------------------------------------ */

const USE_CASES = [
  {
    title: 'Customer-Facing Agents',
    description:
      'Sales consultants, support assistants, onboarding guides — AI agents that interact with your users 24/7. Any industry, any product.',
    outcome: 'Automate customer interactions at scale',
    icon: (
      <svg className="h-6 w-6 text-brand-accent" fill="none" viewBox="0 0 24 24" strokeWidth={1.5} stroke="currentColor">
        <path strokeLinecap="round" strokeLinejoin="round" d="M20.25 8.511c.884.284 1.5 1.128 1.5 2.097v4.286c0 1.136-.847 2.1-1.98 2.193-.34.027-.68.052-1.02.072v3.091l-3-3c-1.354 0-2.694-.055-4.02-.163a2.115 2.115 0 01-.825-.242m9.345-8.334a2.126 2.126 0 00-.476-.095 48.64 48.64 0 00-8.048 0c-1.131.094-1.976 1.057-1.976 2.192v4.286c0 .837.46 1.58 1.155 1.951m9.345-8.334V6.637c0-1.621-1.152-3.026-2.76-3.235A48.455 48.455 0 0011.25 3c-2.115 0-4.198.137-6.24.402-1.608.209-2.76 1.614-2.76 3.235v6.226c0 1.621 1.152 3.026 2.76 3.235.577.075 1.157.14 1.74.194V21l4.155-4.155" />
      </svg>
    ),
  },
  {
    title: 'Internal Automation',
    description:
      'Background agents that monitor, process, and report — without human intervention. Cron schedules, webhook triggers, proactive actions.',
    outcome: 'Autonomous operations, not reactive scripts',
    icon: (
      <svg className="h-6 w-6 text-brand-accent" fill="none" viewBox="0 0 24 24" strokeWidth={1.5} stroke="currentColor">
        <path strokeLinecap="round" strokeLinejoin="round" d="M16.023 9.348h4.992v-.001M2.985 19.644v-4.992m0 0h4.992m-4.993 0l3.181 3.183a8.25 8.25 0 0013.803-3.7M4.031 9.865a8.25 8.25 0 0113.803-3.7l3.181 3.182m0-4.991v4.99" />
      </svg>
    ),
  },
  {
    title: 'Multi-Step Workflows',
    description:
      'Complex chains where each step requires reasoning — not a predetermined DAG. Approvals, reviews, document processing, research tasks.',
    outcome: 'AI reasoning replaces rigid pipelines',
    icon: (
      <svg className="h-6 w-6 text-brand-accent" fill="none" viewBox="0 0 24 24" strokeWidth={1.5} stroke="currentColor">
        <path strokeLinecap="round" strokeLinejoin="round" d="M3.75 6A2.25 2.25 0 016 3.75h2.25A2.25 2.25 0 0110.5 6v2.25a2.25 2.25 0 01-2.25 2.25H6a2.25 2.25 0 01-2.25-2.25V6zM3.75 15.75A2.25 2.25 0 016 13.5h2.25a2.25 2.25 0 012.25 2.25V18a2.25 2.25 0 01-2.25 2.25H6A2.25 2.25 0 013.75 18v-2.25zM13.5 6a2.25 2.25 0 012.25-2.25H18A2.25 2.25 0 0120.25 6v2.25A2.25 2.25 0 0118 10.5h-2.25a2.25 2.25 0 01-2.25-2.25V6zM13.5 15.75a2.25 2.25 0 012.25-2.25H18a2.25 2.25 0 012.25 2.25V18A2.25 2.25 0 0118 20.25h-2.25A2.25 2.25 0 0113.5 18v-2.25z" />
      </svg>
    ),
  },
  {
    title: 'Domain Expert Agents',
    description:
      'Agents with deep knowledge bases — for any field. Medicine, finance, law, engineering, education. RAG-powered answers from your documents.',
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
    <section className="py-20 px-4 border-t border-brand-shade3/15 bg-brand-dark-alt">
      <div className="max-w-6xl mx-auto">
        <h2 className="text-3xl font-bold text-center text-brand-light mb-4">
          See it in action
        </h2>
        <p className="text-center text-brand-shade3 mb-14 max-w-2xl mx-auto">
          ByteBrew comes with a ready-to-use web client and a full Admin Dashboard. Use them as-is, or build your own UI on top of the REST API.
        </p>

        {/* Two showcases in a grid */}
        <div className="grid grid-cols-1 lg:grid-cols-2 gap-8 mb-12">
          {/* Web Client */}
          <div>
            <h3 className="text-lg font-semibold text-brand-light mb-2">Web Client — Chat Interface</h3>
            <p className="text-sm text-brand-shade3 leading-relaxed mb-4">
              Ready-to-use chat interface with multi-agent sidebar, tool calls, and rich markdown. Open source — fork and customize.
            </p>
            <div className="rounded-[12px] border border-brand-shade3/15 overflow-hidden shadow-2xl shadow-brand-accent/5">
              <img src="/screenshots/admin-agents.png" alt="ByteBrew Web Client — AI sales agent recommending laptops with web search tool calls and structured markdown response" className="w-full cursor-pointer hover:opacity-80 transition-opacity" onClick={() => onImageClick('/screenshots/admin-agents.png')} />
            </div>
          </div>

          {/* Admin Dashboard */}
          <div>
            <h3 className="text-lg font-semibold text-brand-light mb-2">Admin Dashboard — Visual Management</h3>
            <p className="text-sm text-brand-shade3 leading-relaxed mb-4">
              Configure agents, models, MCP servers, triggers, and API keys through a visual interface. No YAML required.
            </p>
            <div className="rounded-[12px] border border-brand-shade3/15 overflow-hidden shadow-2xl shadow-brand-accent/5">
              <img src="/screenshots/admin-agent-detail.png" alt="Admin Dashboard — Agent detail panel with model, system prompt, tools, and spawn rules configuration" className="w-full cursor-pointer hover:opacity-80 transition-opacity" onClick={() => onImageClick('/screenshots/admin-agent-detail.png')} />
            </div>
          </div>
        </div>

        {/* Feature highlights grid */}
        <div className="grid grid-cols-1 md:grid-cols-3 gap-6">
          <div className="rounded-[12px] border border-brand-shade3/15 bg-brand-dark p-5">
            <h3 className="text-sm font-semibold text-brand-light mb-2">Multi-Agent Sidebar</h3>
            <p className="text-xs text-brand-shade3 leading-relaxed">
              Switch between agents instantly. Each agent has its own session history, tools, and context. See tool calls and results inline.
            </p>
          </div>
          <div className="rounded-[12px] border border-brand-shade3/15 bg-brand-dark p-5">
            <h3 className="text-sm font-semibold text-brand-light mb-2">Rich Responses</h3>
            <p className="text-xs text-brand-shade3 leading-relaxed">
              Full markdown rendering — bold, code blocks, tables, lists, links. Tool call results expandable inline. Real-time SSE streaming.
            </p>
          </div>
          <div className="rounded-[12px] border border-brand-shade3/15 bg-brand-dark p-5">
            <h3 className="text-sm font-semibold text-brand-light mb-2">Visual Agent Editor</h3>
            <p className="text-xs text-brand-shade3 leading-relaxed">
              Create and configure agents through the Admin Dashboard. Set models, prompts, tools, security zones, and spawn rules — all without touching YAML.
            </p>
          </div>
        </div>
      </div>
    </section>
  );
}

function UseCasesSection() {
  return (
    <section className="py-20 px-4 border-t border-brand-shade3/15 bg-brand-dark-alt">
      <div className="max-w-6xl mx-auto">
        <h2 className="text-3xl font-bold text-center text-brand-light mb-4">
          Any product. Any industry. One Engine.
        </h2>
        <p className="text-center text-brand-shade3 mb-14 max-w-2xl mx-auto">
          ByteBrew is an infrastructure layer for AI agents — like PostgreSQL for data, but for autonomous reasoning.
        </p>
        <div className="grid grid-cols-1 sm:grid-cols-2 gap-6">
          {USE_CASES.map((uc) => (
            <div
              key={uc.title}
              className="rounded-[12px] border border-brand-shade3/15 bg-brand-dark p-6"
            >
              <div className="w-12 h-12 rounded-[10px] bg-brand-accent/10 flex items-center justify-center mb-4">
                {uc.icon}
              </div>
              <h3 className="text-lg font-semibold text-brand-light">{uc.title}</h3>
              <p className="mt-2 text-sm text-brand-shade3 leading-relaxed">
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
  { approach: 'Cloud-only AI platforms', problem: 'Per-token pricing, vendor lock-in, data on their servers', bytebrew: 'Self-hosted, BYOK, your data stays with you' },
  { approach: 'AI SDKs & frameworks', problem: 'Months of development, requires dedicated engineering team', bytebrew: 'YAML config, deploy in 30 seconds, no code required' },
  { approach: 'Visual workflow builders', problem: 'Reactive chatbots, no autonomous reasoning, limited logic', bytebrew: 'Autonomous daemon agents with multi-step reasoning' },
  { approach: 'Single-model APIs', problem: 'Locked to one provider, no orchestration, no tool system', bytebrew: 'Any model, multi-agent spawn tree, universal tool system' },
  { approach: 'Custom in-house solutions', problem: '3-6 months to build, ongoing maintenance burden', bytebrew: 'Production-ready engine, maintained and updated for you' },
];

function ComparisonSection() {
  return (
    <section className="py-20 px-4 border-t border-brand-shade3/15 bg-brand-dark">
      <div className="max-w-5xl mx-auto">
        <h2 className="text-3xl font-bold text-center text-brand-light mb-4">
          Why ByteBrew?
        </h2>
        <p className="text-center text-brand-shade3 mb-14">
          How ByteBrew Engine compares to traditional approaches.
        </p>

        <div className="overflow-x-auto rounded-[12px] border border-brand-shade3/15">
          <table className="w-full text-sm">
            <thead>
              <tr className="border-b border-brand-shade3/15 bg-brand-dark-alt">
                <th className="text-left py-3 px-4 font-medium text-brand-shade3">Traditional Approach</th>
                <th className="text-left py-3 px-4 font-medium text-brand-shade3">The Problem</th>
                <th className="text-left py-3 px-4 font-semibold text-brand-accent">ByteBrew Engine</th>
              </tr>
            </thead>
            <tbody>
              {APPROACH_COMPARISON.map((row, i) => (
                <tr
                  key={row.approach}
                  className={i < APPROACH_COMPARISON.length - 1 ? 'border-b border-brand-shade3/10' : ''}
                >
                  <td className="py-3 px-4 text-brand-shade2">{row.approach}</td>
                  <td className="py-3 px-4 text-brand-shade3">{row.problem}</td>
                  <td className="py-3 px-4 text-brand-light">{row.bytebrew}</td>
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
/*  8a. MCP Docs                                                       */
/* ------------------------------------------------------------------ */

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
  const [activeTab, setActiveTab] = useState<'claude' | 'codex' | 'other'>('claude');
  const tab = MCP_CODE[activeTab];

  return (
    <section className="py-20 px-4 border-t border-brand-shade3/15 bg-brand-dark">
      <div className="max-w-3xl mx-auto">
        <h2 className="text-3xl font-bold text-center text-brand-light mb-2">
          AI-Native Documentation
        </h2>
        <p className="text-center text-brand-shade3 mb-10 max-w-2xl mx-auto">
          Connect your AI coding assistant to ByteBrew docs. Get accurate answers about
          configuration, API, deployment, and more &mdash; powered by RAG over our documentation.
        </p>

        {/* Tabs */}
        <div className="flex items-center justify-center gap-2 mb-6">
          {MCP_TABS.map((t) => (
            <button
              key={t.key}
              onClick={() => setActiveTab(t.key)}
              className={`rounded-[8px] px-5 py-2 text-sm font-medium transition-colors ${
                activeTab === t.key
                  ? 'bg-brand-accent text-white'
                  : 'bg-brand-dark-alt text-brand-shade2 hover:text-brand-light'
              }`}
            >
              {t.label}
            </button>
          ))}
        </div>

        {/* Code block */}
        {tab.prefix ? (
          <TerminalBlock command={tab.code} />
        ) : (
          <div className="relative rounded-[12px] border border-brand-shade3/15 bg-brand-dark-alt p-5 overflow-x-auto">
            <pre className="font-mono text-sm text-brand-shade2 leading-relaxed whitespace-pre">
              {tab.code}
            </pre>
          </div>
        )}

        <p className="mt-6 text-center text-sm text-brand-shade3">
          After connecting, ask your AI assistant about ByteBrew &mdash; it will search our
          documentation automatically.
        </p>
      </div>
    </section>
  );
}

/* ------------------------------------------------------------------ */
/*  8b. Install / Quick Start                                          */
/* ------------------------------------------------------------------ */

function InstallSection() {
  return (
    <section className="py-20 px-4 border-t border-brand-shade3/15 bg-brand-dark-alt">
      <div className="max-w-3xl mx-auto">
        <h2 className="text-3xl font-bold text-center text-brand-light mb-2">
          Get Started in 30 Seconds
        </h2>
        <p className="text-center text-brand-shade3 mb-10">
          No PostgreSQL? No problem &mdash; it&apos;s included.
        </p>

        <TerminalBlock
          command="curl -fsSL https://bytebrew.ai/releases/docker-compose.yml -o docker-compose.yml && docker compose up -d"
        />

        <p className="mt-6 text-center text-sm text-brand-shade3">
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
    <section id="pricing" className="py-20 px-4 border-t border-brand-shade3/15 bg-brand-dark">
      <div className="max-w-5xl mx-auto">
        <h2 className="text-3xl font-bold text-center text-brand-light mb-4">
          Simple, Transparent Pricing
        </h2>
        <p className="text-center text-brand-shade3 mb-14">
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
    <section className="py-16 px-4 border-t border-brand-shade3/15 bg-brand-dark-alt">
      <div className="max-w-3xl mx-auto rounded-[12px] border border-brand-accent/30 bg-brand-accent/5 p-10 text-center">
        <h2 className="text-2xl font-bold text-brand-light">
          ByteBrew Community Edition is free forever.
        </h2>
        <p className="mt-4 text-brand-shade2 leading-relaxed">
          No agent limits. No session limits. No time limits.
          <br />
          CE features will never move behind a paywall.
        </p>
        <p className="mt-2 text-sm text-brand-shade3">
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
    <section className="py-20 px-4 border-t border-brand-shade3/15 bg-brand-dark">
      <div className="max-w-2xl mx-auto text-center">
        <h2 className="text-3xl font-bold text-brand-light">
          Start building with ByteBrew Engine
        </h2>
        <p className="mt-4 text-brand-shade3">
          One Docker command. Full AI agent runtime. Free forever.
        </p>
        <Link
          to="/download"
          className="mt-8 inline-block rounded-[10px] bg-brand-accent px-8 py-3.5 text-sm font-medium text-white hover:bg-brand-accent-hover transition-colors"
        >
          Get Started Free
        </Link>
      </div>
    </section>
  );
}
