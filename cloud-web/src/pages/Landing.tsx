import { Link } from '@tanstack/react-router';
import { TerminalBlock } from '../components/TerminalBlock';
import { HeroDemo } from '../components/HeroDemo';

export function LandingPage() {
  return (
    <div>
      <HeroSection />
      <LogoBar />
      <WhatWhatNotSection />
      <HowItWorksSection />
      <DemoSection />
      <UseCasesSection />
      <PricingSection />
      <CapabilitiesSection />
      <QuickStartSection />
      <CommunitySection />
      <FinalCTASection />
    </div>
  );
}

/* ------------------------------------------------------------------ */
/*  Shared icons                                                       */
/* ------------------------------------------------------------------ */

function CheckIcon({ className = 'h-5 w-5 text-emerald-400' }: { className?: string }) {
  return (
    <svg className={`shrink-0 ${className}`} fill="none" viewBox="0 0 24 24" strokeWidth={2.5} stroke="currentColor">
      <path strokeLinecap="round" strokeLinejoin="round" d="M4.5 12.75l6 6 9-13.5" />
    </svg>
  );
}

function XIcon({ className = 'h-5 w-5 text-red-400' }: { className?: string }) {
  return (
    <svg className={`shrink-0 ${className}`} fill="none" viewBox="0 0 24 24" strokeWidth={2.5} stroke="currentColor">
      <path strokeLinecap="round" strokeLinejoin="round" d="M6 18L18 6M6 6l12 12" />
    </svg>
  );
}

/* ------------------------------------------------------------------ */
/*  1. Hero                                                            */
/* ------------------------------------------------------------------ */

function HeroSection() {
  return (
    <section className="pt-16 pb-20 px-4 bg-surface">
      <div className="max-w-4xl mx-auto text-center">
        <p className="text-sm font-sans font-medium text-text-tertiary mb-4 tracking-wide">
          Not another AI chatbot.
        </p>
        <h1 className="text-4xl sm:text-5xl lg:text-6xl font-bold tracking-tight text-text-primary leading-[1.1]">
          ByteBrew — the open-source
          <br />
          <span className="text-brand-accent">agent brewery</span>
        </h1>
        <p className="mt-6 text-lg text-text-secondary font-sans leading-relaxed max-w-2xl mx-auto">
          Describe an operation — agents assemble themselves to run it.
          <br />
          <span className="text-text-tertiary">Memory. Tools. Multi-agent flows. Self-hosted. Any LLM.</span>
        </p>

        <div className="mt-8 flex items-center justify-center gap-3 flex-wrap">
          <Link
            to="/register"
            className="rounded-[2px] bg-brand-accent px-7 py-3 text-sm font-medium text-white hover:bg-brand-accent-hover transition-colors"
          >
            Try Free
          </Link>
          <a
            href="https://github.com/syntheticinc/bytebrew"
            target="_blank"
            rel="noopener noreferrer"
            className="rounded-[2px] border border-border px-7 py-3 text-sm font-medium text-text-secondary hover:border-border-hover hover:text-text-primary transition-colors flex items-center gap-2"
          >
            <svg className="h-4 w-4" viewBox="0 0 24 24" fill="currentColor">
              <path d="M12 0C5.37 0 0 5.37 0 12c0 5.31 3.435 9.795 8.205 11.385.6.105.825-.255.825-.57 0-.285-.015-1.23-.015-2.235-3.015.555-3.795-.735-4.035-1.41-.135-.345-.72-1.41-1.23-1.695-.42-.225-1.02-.78-.015-.795.945-.015 1.62.87 1.845 1.23 1.08 1.815 2.805 1.305 3.495.99.105-.78.42-1.305.765-1.605-2.67-.3-5.46-1.335-5.46-5.925 0-1.305.465-2.385 1.23-3.225-.12-.3-.54-1.53.12-3.18 0 0 1.005-.315 3.3 1.23.96-.27 1.98-.405 3-.405s2.04.135 3 .405c2.295-1.56 3.3-1.23 3.3-1.23.66 1.65.24 2.88.12 3.18.765.84 1.23 1.905 1.23 3.225 0 4.605-2.805 5.625-5.475 5.925.435.375.81 1.095.81 2.22 0 1.605-.015 2.895-.015 3.3 0 .315.225.69.825.57A12.02 12.02 0 0024 12c0-6.63-5.37-12-12-12z" />
            </svg>
            GitHub
          </a>
          <a
            href="/docs/getting-started/quick-start/"
            className="rounded-[2px] border border-border px-7 py-3 text-sm font-medium text-text-secondary hover:border-border-hover hover:text-text-primary transition-colors"
          >
            Self-host &rarr;
          </a>
        </div>
      </div>
    </section>
  );
}

/* ------------------------------------------------------------------ */
/*  Logo bar                                                           */
/* ------------------------------------------------------------------ */

function LogoBar() {
  return (
    <div className="py-6 px-4 border-t border-border bg-surface">
      <div className="max-w-4xl mx-auto flex flex-wrap items-center justify-center gap-x-8 gap-y-2">
        <span className="text-xs font-sans text-text-tertiary">Works with</span>
        <span className="text-sm font-semibold text-text-secondary">OpenAI</span>
        <span className="text-sm font-semibold text-text-secondary">Anthropic</span>
        <span className="text-sm font-semibold text-text-secondary">Google AI</span>
        <span className="text-sm font-semibold text-text-secondary">Ollama</span>
        <span className="text-sm font-semibold text-text-secondary">PostgreSQL</span>
        <span className="text-sm font-semibold text-text-secondary">Docker</span>
      </div>
    </div>
  );
}

/* ------------------------------------------------------------------ */
/*  2. What / What Not                                                 */
/* ------------------------------------------------------------------ */

const WHAT_IS = [
  'AI agents that reason, act, and coordinate',
  '2,000+ integrations via MCP',
  'Self-hosted, open source (BSL 1.1)',
  'Visual builder + Admin Dashboard',
  'Multi-agent flows with gates and loops',
  'Per-schema memory that persists across sessions',
];

const WHAT_IS_NOT = [
  'Not a chatbot wrapper',
  'Not a workflow builder (drag-and-drop boxes)',
  'Not "AI employee" hype',
  'Not cloud-only or vendor-locked',
];

function WhatWhatNotSection() {
  return (
    <section className="py-24 px-4 border-t border-border bg-surface-alt">
      <div className="max-w-5xl mx-auto">
        <h2 className="text-3xl font-bold tracking-tight text-center text-text-primary mb-14">
          What ByteBrew is — and what it's not
        </h2>
        <div className="grid grid-cols-1 md:grid-cols-2 gap-8">
          <div className="rounded-[2px] border border-emerald-500/20 bg-emerald-500/[0.03] p-6">
            <h3 className="text-sm font-semibold uppercase tracking-wider text-emerald-400 mb-4">ByteBrew is</h3>
            <ul className="space-y-3">
              {WHAT_IS.map((item) => (
                <li key={item} className="flex items-start gap-3">
                  <CheckIcon />
                  <span className="text-sm text-text-secondary font-sans">{item}</span>
                </li>
              ))}
            </ul>
          </div>
          <div className="rounded-[2px] border border-red-500/20 bg-red-500/[0.03] p-6">
            <h3 className="text-sm font-semibold uppercase tracking-wider text-red-400 mb-4">ByteBrew is not</h3>
            <ul className="space-y-3">
              {WHAT_IS_NOT.map((item) => (
                <li key={item} className="flex items-start gap-3">
                  <XIcon />
                  <span className="text-sm text-text-secondary font-sans">{item}</span>
                </li>
              ))}
            </ul>
          </div>
        </div>
      </div>
    </section>
  );
}

/* ------------------------------------------------------------------ */
/*  3. How It Works                                                    */
/* ------------------------------------------------------------------ */

const STEPS = [
  {
    number: '01',
    title: 'Describe',
    subtitle: 'Tell ByteBrew what you need',
    description: 'Define your operation in natural language or configure agents via the visual builder. Set up tools, memory, and schemas.',
  },
  {
    number: '02',
    title: 'Brew',
    subtitle: 'Agents assemble themselves',
    description: 'ByteBrew creates the right agent configuration — ReAct reasoning, MCP tools, multi-agent flows with gates and conditions.',
  },
  {
    number: '03',
    title: 'Run',
    subtitle: 'The system works autonomously',
    description: 'Agents execute in production. Memory persists. Flows coordinate. You monitor via the Inspect dashboard and intervene when needed.',
  },
];

function HowItWorksSection() {
  return (
    <section className="py-24 px-4 border-t border-border bg-surface">
      <div className="max-w-5xl mx-auto">
        <h2 className="text-3xl font-bold tracking-tight text-center text-text-primary mb-4">
          How it works
        </h2>
        <p className="text-center text-text-secondary font-sans mb-14 max-w-2xl mx-auto">
          From description to production in minutes, not months.
        </p>
        <div className="grid grid-cols-1 md:grid-cols-3 gap-8">
          {STEPS.map((step) => (
            <div key={step.number} className="relative">
              <div className="text-5xl font-bold text-brand-accent/15 mb-2 font-mono">{step.number}</div>
              <h3 className="text-xl font-bold text-text-primary mb-1">{step.title}</h3>
              <p className="text-sm font-medium text-brand-accent mb-3">{step.subtitle}</p>
              <p className="text-sm text-text-secondary font-sans leading-relaxed">{step.description}</p>
            </div>
          ))}
        </div>
      </div>
    </section>
  );
}

/* ------------------------------------------------------------------ */
/*  4. Demo                                                            */
/* ------------------------------------------------------------------ */

function DemoSection() {
  return (
    <section className="py-20 px-4 border-t border-border bg-surface-alt">
      <div className="max-w-5xl mx-auto">
        <h2 className="text-3xl font-bold tracking-tight text-center text-text-primary mb-4">
          See it in action
        </h2>
        <p className="text-center text-text-secondary font-sans mb-10">
          A multi-agent system running in real-time — tool calls, reasoning, and streaming.
        </p>
        <HeroDemo />
        <p className="text-center text-xs font-sans text-text-tertiary mt-4">
          Live demo: an AI agent analyzing enterprise data with sub-agent delegation and real-time SSE streaming
        </p>
      </div>
    </section>
  );
}

/* ------------------------------------------------------------------ */
/*  5. Use Cases                                                       */
/* ------------------------------------------------------------------ */

const USE_CASES = [
  {
    title: 'Customer Support Agent',
    description: 'Answer questions, search knowledge bases, escalate to humans. Memory remembers each customer across sessions.',
    tags: ['Memory', 'Knowledge', 'Escalation'],
  },
  {
    title: 'AI-First Product in a Week',
    description: 'Embed an agent into your SaaS via REST API + Widget. Self-hosted, your data stays on your infrastructure.',
    tags: ['REST API', 'Widget', 'Self-hosted'],
  },
  {
    title: 'Autonomous Data Pipeline',
    description: 'Cron triggers kick off analysis agents. Sub-agents parallelize work. Gates validate output quality before delivery.',
    tags: ['Triggers', 'Flows', 'Gates'],
  },
];

function UseCasesSection() {
  return (
    <section className="py-24 px-4 border-t border-border bg-surface">
      <div className="max-w-5xl mx-auto">
        <h2 className="text-3xl font-bold tracking-tight text-center text-text-primary mb-4">
          Built for real operations
        </h2>
        <p className="text-center text-text-secondary font-sans mb-14 max-w-2xl mx-auto">
          Not demos. Production workloads running 24/7.
        </p>
        <div className="grid grid-cols-1 md:grid-cols-3 gap-6">
          {USE_CASES.map((uc) => (
            <div
              key={uc.title}
              className="rounded-[2px] border border-border bg-surface-alt p-6 hover:border-border-hover transition-colors"
            >
              <h3 className="text-base font-semibold text-text-primary mb-2">{uc.title}</h3>
              <p className="text-sm text-text-secondary font-sans leading-relaxed mb-4">{uc.description}</p>
              <div className="flex flex-wrap gap-1.5">
                {uc.tags.map((tag) => (
                  <span key={tag} className="text-[10px] font-medium px-2 py-0.5 rounded-[2px] bg-brand-accent/10 text-brand-accent">
                    {tag}
                  </span>
                ))}
              </div>
            </div>
          ))}
        </div>
      </div>
    </section>
  );
}

/* ------------------------------------------------------------------ */
/*  6. Pricing                                                         */
/* ------------------------------------------------------------------ */

const PRICING_TIERS = [
  {
    name: 'Free',
    price: '$0',
    period: 'forever',
    description: 'For individuals and experiments',
    features: ['1 schema', '10 agents', '1,000 API calls/mo', '100 MB storage', '1 widget', 'Community support'],
    cta: 'Get Started',
    ctaLink: '/register',
    highlighted: false,
  },
  {
    name: 'Pro',
    price: '$29',
    period: '/month',
    description: 'For teams shipping AI products',
    features: ['5 schemas', '50 agents', '50,000 API calls/mo', '5 GB storage', '3 widgets', 'Email support', 'BYOK (bring your own keys)'],
    cta: 'Start Free Trial',
    ctaLink: '/register',
    highlighted: true,
  },
  {
    name: 'Business',
    price: '$99',
    period: '/month',
    description: 'For production workloads at scale',
    features: ['Unlimited schemas', 'Unlimited agents', '500,000 API calls/mo', '50 GB storage', 'Unlimited widgets', 'Priority support', 'forward_headers'],
    cta: 'Start Free Trial',
    ctaLink: '/register',
    highlighted: false,
  },
  {
    name: 'Enterprise',
    price: 'Custom',
    period: '',
    description: 'SSO, audit, compliance, dedicated support',
    features: ['Everything in Business', 'SSO / SAML', 'Audit log', 'Prometheus metrics', 'Dedicated support', 'SLA guarantee'],
    cta: 'Contact Sales',
    ctaLink: 'mailto:sales@bytebrew.ai',
    highlighted: false,
  },
];

function PricingSection() {
  return (
    <section id="pricing" className="py-24 px-4 border-t border-border bg-surface-alt">
      <div className="max-w-6xl mx-auto">
        <h2 className="text-3xl font-bold tracking-tight text-center text-text-primary mb-4">
          Simple, transparent pricing
        </h2>
        <p className="text-center text-text-secondary font-sans mb-14">
          Self-hosted Community Edition is free forever. Cloud starts at $0.
        </p>
        <div className="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-4 gap-5">
          {PRICING_TIERS.map((tier) => (
            <div
              key={tier.name}
              className={`rounded-[2px] border p-6 flex flex-col ${
                tier.highlighted
                  ? 'border-brand-accent/50 bg-brand-accent/[0.03] ring-1 ring-brand-accent/20'
                  : 'border-border bg-surface'
              }`}
            >
              {tier.highlighted && (
                <span className="text-[10px] font-semibold uppercase tracking-wider text-brand-accent mb-2">Most popular</span>
              )}
              <h3 className="text-lg font-bold text-text-primary">{tier.name}</h3>
              <div className="mt-2 flex items-baseline gap-1">
                <span className="text-3xl font-bold text-text-primary">{tier.price}</span>
                {tier.period && <span className="text-sm text-text-tertiary">{tier.period}</span>}
              </div>
              <p className="text-xs text-text-tertiary mt-2 mb-5">{tier.description}</p>
              <ul className="space-y-2 flex-1 mb-6">
                {tier.features.map((f) => (
                  <li key={f} className="flex items-start gap-2">
                    <CheckIcon className="h-4 w-4 text-emerald-400 mt-0.5" />
                    <span className="text-xs text-text-secondary font-sans">{f}</span>
                  </li>
                ))}
              </ul>
              <Link
                to={tier.ctaLink}
                className={`block text-center rounded-[2px] px-4 py-2.5 text-sm font-medium transition-colors ${
                  tier.highlighted
                    ? 'bg-brand-accent text-white hover:bg-brand-accent-hover'
                    : 'border border-border text-text-secondary hover:border-border-hover hover:text-text-primary'
                }`}
              >
                {tier.cta}
              </Link>
            </div>
          ))}
        </div>

        {/* CE free forever callout */}
        <div className="mt-10 rounded-[2px] border border-border bg-surface p-6 text-center max-w-2xl mx-auto">
          <p className="text-sm font-semibold text-text-primary">
            ByteBrew Community Edition is free forever.
          </p>
          <p className="text-xs text-text-tertiary mt-1">
            Self-host on your own infrastructure. No limits on agents or sessions. No credit card required.
          </p>
        </div>
      </div>
    </section>
  );
}

/* ------------------------------------------------------------------ */
/*  7. Engine Capabilities                                             */
/* ------------------------------------------------------------------ */

const CAPABILITIES = [
  {
    title: 'ReAct Reasoning',
    description: 'Agents think step-by-step: Reason → Act → Observe → Repeat. Not scripted flows — genuine reasoning.',
  },
  {
    title: 'Memory',
    description: 'Per-schema, cross-session persistence. Agents remember customers, context, and decisions across conversations.',
  },
  {
    title: 'Multi-Agent Flows',
    description: 'Agents coordinate via flow edges, transfer, and spawn. Build teams of specialized agents.',
  },
  {
    title: 'Gates & Conditions',
    description: 'Quality gates between agents: JSON Schema validation, LLM judges, human approval, join conditions.',
  },
  {
    title: 'Inspect Dashboard',
    description: 'Full session trace: every reasoning step, tool call, memory access, and decision — searchable and filterable.',
  },
  {
    title: 'Recovery & Resilience',
    description: 'Heartbeat monitoring, MCP timeout handling, dead letter queues, circuit breakers for external services.',
  },
  {
    title: '2,000+ MCP Tools',
    description: 'Connect to any external service via MCP. Curated catalog with one-click install. Stdio, SSE, Docker transport.',
  },
  {
    title: 'Knowledge / RAG',
    description: 'Upload PDF, DOCX, URLs. Agents search knowledge automatically. Per-schema isolation.',
  },
  {
    title: 'Visual Builder',
    description: 'Drag-and-drop canvas for agent schemas. Configure flows, gates, edges, triggers — all visual.',
  },
];

function CapabilitiesSection() {
  return (
    <section className="py-24 px-4 border-t border-border bg-surface">
      <div className="max-w-6xl mx-auto">
        <h2 className="text-3xl font-bold tracking-tight text-center text-text-primary mb-4">
          Everything your agents need
        </h2>
        <p className="text-center text-text-secondary font-sans mb-14 max-w-2xl mx-auto">
          Built-in, not bolted on. Every capability included in one container.
        </p>
        <div className="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-3 gap-5">
          {CAPABILITIES.map((c) => (
            <div
              key={c.title}
              className="rounded-[2px] border border-border bg-surface-alt p-5 hover:border-border-hover hover:-translate-y-0.5 transition-all duration-200"
            >
              <h3 className="text-base font-semibold text-text-primary">{c.title}</h3>
              <p className="mt-2 text-sm text-text-secondary font-sans leading-relaxed">{c.description}</p>
            </div>
          ))}
        </div>
      </div>
    </section>
  );
}

/* ------------------------------------------------------------------ */
/*  8. Quick Start                                                     */
/* ------------------------------------------------------------------ */

function QuickStartSection() {
  return (
    <section className="py-24 px-4 border-t border-border bg-surface-alt">
      <div className="max-w-3xl mx-auto text-center">
        <h2 className="text-3xl font-bold tracking-tight text-text-primary mb-4">
          One command to start
        </h2>
        <p className="text-text-secondary font-sans mb-8">
          Pull the Docker image and you're running. Full agent runtime with Admin Dashboard included.
        </p>
        <TerminalBlock command="docker run -d -p 8443:8443 bytebrew/engine">
          <p className="text-text-tertiary mt-2 text-sm font-mono">
            # Open http://localhost:8443/admin/<br />
            # Create agents, connect tools, start chatting.
          </p>
        </TerminalBlock>
        <div className="mt-8 flex items-center justify-center gap-4 flex-wrap">
          <a
            href="/docs/getting-started/quick-start/"
            className="text-sm text-brand-accent hover:underline transition-colors"
          >
            Full Quick Start Guide &rarr;
          </a>
          <a
            href="/docs/deploy/docker/"
            className="text-sm text-text-tertiary hover:text-text-secondary transition-colors"
          >
            Docker Compose &rarr;
          </a>
          <a
            href="/docs/deploy/kubernetes/"
            className="text-sm text-text-tertiary hover:text-text-secondary transition-colors"
          >
            Kubernetes (Helm) &rarr;
          </a>
        </div>
      </div>
    </section>
  );
}

/* ------------------------------------------------------------------ */
/*  9. Open Source + Community                                          */
/* ------------------------------------------------------------------ */

function CommunitySection() {
  return (
    <section className="py-24 px-4 border-t border-border bg-surface">
      <div className="max-w-4xl mx-auto text-center">
        <h2 className="text-3xl font-bold tracking-tight text-text-primary mb-4">
          Open source. Community driven.
        </h2>
        <p className="text-text-secondary font-sans mb-12 max-w-2xl mx-auto">
          ByteBrew is BSL 1.1 licensed. Free to self-host, embed in your products, and modify.
          <br />
          Converts to Apache 2.0 after 4 years.
        </p>
        <div className="grid grid-cols-1 sm:grid-cols-3 gap-6">
          <a
            href="https://github.com/syntheticinc/bytebrew"
            target="_blank"
            rel="noopener noreferrer"
            className="rounded-[2px] border border-border bg-surface-alt p-6 hover:border-border-hover transition-colors group"
          >
            <svg className="h-8 w-8 text-text-tertiary group-hover:text-text-primary transition-colors mx-auto mb-3" viewBox="0 0 24 24" fill="currentColor">
              <path d="M12 0C5.37 0 0 5.37 0 12c0 5.31 3.435 9.795 8.205 11.385.6.105.825-.255.825-.57 0-.285-.015-1.23-.015-2.235-3.015.555-3.795-.735-4.035-1.41-.135-.345-.72-1.41-1.23-1.695-.42-.225-1.02-.78-.015-.795.945-.015 1.62.87 1.845 1.23 1.08 1.815 2.805 1.305 3.495.99.105-.78.42-1.305.765-1.605-2.67-.3-5.46-1.335-5.46-5.925 0-1.305.465-2.385 1.23-3.225-.12-.3-.54-1.53.12-3.18 0 0 1.005-.315 3.3 1.23.96-.27 1.98-.405 3-.405s2.04.135 3 .405c2.295-1.56 3.3-1.23 3.3-1.23.66 1.65.24 2.88.12 3.18.765.84 1.23 1.905 1.23 3.225 0 4.605-2.805 5.625-5.475 5.925.435.375.81 1.095.81 2.22 0 1.605-.015 2.895-.015 3.3 0 .315.225.69.825.57A12.02 12.02 0 0024 12c0-6.63-5.37-12-12-12z" />
            </svg>
            <h3 className="text-base font-semibold text-text-primary mb-1">GitHub</h3>
            <p className="text-xs text-text-tertiary">Star, fork, contribute</p>
          </a>
          <a
            href="https://discord.gg/bytebrew"
            target="_blank"
            rel="noopener noreferrer"
            className="rounded-[2px] border border-border bg-surface-alt p-6 hover:border-border-hover transition-colors group"
          >
            <svg className="h-8 w-8 text-text-tertiary group-hover:text-[#5865F2] transition-colors mx-auto mb-3" viewBox="0 0 24 24" fill="currentColor">
              <path d="M20.317 4.37a19.79 19.79 0 00-4.885-1.515.074.074 0 00-.079.037c-.21.375-.444.864-.608 1.25a18.27 18.27 0 00-5.487 0 12.64 12.64 0 00-.617-1.25.077.077 0 00-.079-.037A19.74 19.74 0 003.677 4.37a.07.07 0 00-.032.027C.533 9.046-.32 13.58.099 18.057a.082.082 0 00.031.057 19.9 19.9 0 005.993 3.03.078.078 0 00.084-.028 14.09 14.09 0 001.226-1.994.076.076 0 00-.041-.106 13.11 13.11 0 01-1.872-.892.077.077 0 01-.008-.128c.126-.094.252-.192.372-.291a.074.074 0 01.077-.01c3.928 1.793 8.18 1.793 12.062 0a.074.074 0 01.078.009c.12.099.246.198.373.292a.077.077 0 01-.006.127 12.3 12.3 0 01-1.873.892.077.077 0 00-.041.107c.36.698.772 1.362 1.225 1.993a.076.076 0 00.084.028 19.84 19.84 0 006.002-3.03.077.077 0 00.032-.054c.5-5.177-.838-9.674-3.549-13.66a.061.061 0 00-.031-.03zM8.02 15.33c-1.183 0-2.157-1.085-2.157-2.419 0-1.333.956-2.419 2.157-2.419 1.21 0 2.176 1.095 2.157 2.42 0 1.333-.956 2.418-2.157 2.418zm7.975 0c-1.183 0-2.157-1.085-2.157-2.419 0-1.333.955-2.419 2.157-2.419 1.21 0 2.176 1.095 2.157 2.42 0 1.333-.946 2.418-2.157 2.418z" />
            </svg>
            <h3 className="text-base font-semibold text-text-primary mb-1">Discord</h3>
            <p className="text-xs text-text-tertiary">Chat with the community</p>
          </a>
          <a
            href="/docs"
            className="rounded-[2px] border border-border bg-surface-alt p-6 hover:border-border-hover transition-colors group"
          >
            <svg className="h-8 w-8 text-text-tertiary group-hover:text-text-primary transition-colors mx-auto mb-3" fill="none" viewBox="0 0 24 24" strokeWidth={1.5} stroke="currentColor">
              <path strokeLinecap="round" strokeLinejoin="round" d="M12 6.042A8.967 8.967 0 006 3.75c-1.052 0-2.062.18-3 .512v14.25A8.987 8.987 0 016 18c2.305 0 4.408.867 6 2.292m0-14.25a8.966 8.966 0 016-2.292c1.052 0 2.062.18 3 .512v14.25A8.987 8.987 0 0018 18a8.967 8.967 0 00-6 2.292m0-14.25v14.25" />
            </svg>
            <h3 className="text-base font-semibold text-text-primary mb-1">Documentation</h3>
            <p className="text-xs text-text-tertiary">Guides, API reference, examples</p>
          </a>
        </div>
      </div>
    </section>
  );
}

/* ------------------------------------------------------------------ */
/*  Final CTA                                                          */
/* ------------------------------------------------------------------ */

function FinalCTASection() {
  return (
    <section className="py-24 px-4 border-t border-border bg-surface-alt">
      <div className="max-w-2xl mx-auto text-center">
        <h2 className="text-3xl font-bold text-text-primary">
          Start brewing agents today
        </h2>
        <p className="mt-4 text-text-secondary font-sans">
          One Docker command. Full AI agent runtime. Any LLM. Free forever.
        </p>
        <div className="mt-8 flex items-center justify-center gap-3 flex-wrap">
          <Link
            to="/register"
            className="rounded-[2px] bg-brand-accent px-8 py-3.5 text-sm font-medium text-white hover:bg-brand-accent-hover transition-colors"
          >
            Try Free
          </Link>
          <a
            href="https://github.com/syntheticinc/bytebrew"
            target="_blank"
            rel="noopener noreferrer"
            className="rounded-[2px] border border-border px-8 py-3.5 text-sm font-medium text-text-secondary hover:border-border-hover hover:text-text-primary transition-colors"
          >
            View on GitHub
          </a>
        </div>
      </div>
    </section>
  );
}
