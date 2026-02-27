import { useState } from 'react';
import { useNavigate } from '@tanstack/react-router';
import { PricingTable } from '../components/PricingTable';

export function LandingPage() {
  const navigate = useNavigate();

  const handleSelectPlan = () => {
    navigate({ to: '/register' });
  };

  return (
    <div>
      {/* Hero */}
      <section className="py-20 px-4 text-center">
        <div className="max-w-3xl mx-auto">
          <h1 className="text-5xl font-bold tracking-tight text-white">
            AI Agent for{' '}
            <span className="text-indigo-400">Software Engineers</span>
          </h1>
          <p className="mt-6 text-lg text-gray-400 leading-relaxed">
            Autonomous multi-agent system that plans, codes, reviews, and tests.
            Works with your codebase, your tools, your workflow.
          </p>
          <div className="mt-8 flex items-center justify-center gap-4">
            <button
              onClick={() => navigate({ to: '/register' })}
              className="rounded-lg bg-indigo-600 px-6 py-3 text-sm font-medium text-white hover:bg-indigo-500 transition-colors"
            >
              Start 14-Day Trial
            </button>
            <a
              href="#pricing"
              className="rounded-lg border border-gray-700 px-6 py-3 text-sm font-medium text-gray-300 hover:border-gray-500 hover:text-white transition-colors"
            >
              View Pricing
            </a>
          </div>
        </div>
      </section>

      {/* Features */}
      <section className="py-16 px-4 border-t border-gray-800">
        <div className="max-w-5xl mx-auto">
          <h2 className="text-2xl font-bold text-center text-white mb-12">
            How ByteBrew Works
          </h2>
          <div className="grid grid-cols-1 md:grid-cols-3 gap-8">
            <FeatureCard
              title="Multi-Agent Architecture"
              description="Supervisor plans, Coder implements, Reviewer checks, Tester validates. Four specialized agents working together."
            />
            <FeatureCard
              title="Full Tool Suite"
              description="File operations, code search, LSP diagnostics, shell execution. Everything an engineer needs, automated."
            />
            <FeatureCard
              title="BYOK Freedom"
              description="Use our proxy with smart routing, or bring your own API key. Anthropic, OpenAI, OpenRouter, Ollama."
            />
          </div>
        </div>
      </section>

      {/* Installation */}
      <InstallSection />

      {/* Pricing */}
      <section id="pricing" className="py-16 px-4 border-t border-gray-800">
        <div className="max-w-5xl mx-auto">
          <h2 className="text-2xl font-bold text-center text-white mb-4">
            Simple, Transparent Pricing
          </h2>
          <p className="text-center text-gray-400 mb-12">
            No free tier. Start with a 14-day trial, then choose your plan.
          </p>
          <PricingTable onSelectPlan={handleSelectPlan} />
        </div>
      </section>

      {/* CTA */}
      <section className="py-16 px-4 border-t border-gray-800">
        <div className="max-w-2xl mx-auto text-center">
          <h2 className="text-2xl font-bold text-white">Ready to try ByteBrew?</h2>
          <p className="mt-4 text-gray-400">
            14-day free trial with full access. No limitations.
          </p>
          <button
            onClick={() => navigate({ to: '/register' })}
            className="mt-6 rounded-lg bg-indigo-600 px-6 py-3 text-sm font-medium text-white hover:bg-indigo-500 transition-colors"
          >
            Start Trial
          </button>
        </div>
      </section>
    </div>
  );
}

const installCommands = {
  macOS: 'curl -fsSL https://raw.githubusercontent.com/Bearil/usm-epicsmasher/main/scripts/install.sh | sh',
  Linux: 'curl -fsSL https://raw.githubusercontent.com/Bearil/usm-epicsmasher/main/scripts/install.sh | sh',
  Windows:
    'irm https://raw.githubusercontent.com/Bearil/usm-epicsmasher/main/scripts/install.ps1 | iex',
} as const;

type Platform = keyof typeof installCommands;

const tabs: Platform[] = ['macOS', 'Linux', 'Windows'];

function InstallSection() {
  const [activeTab, setActiveTab] = useState<Platform>('macOS');
  const [copied, setCopied] = useState(false);

  const command = installCommands[activeTab];

  const handleCopy = () => {
    navigator.clipboard.writeText(command);
    setCopied(true);
    setTimeout(() => setCopied(false), 2000);
  };

  return (
    <section className="py-16 px-4 border-t border-gray-800">
      <div className="max-w-3xl mx-auto">
        <h2 className="text-2xl font-bold text-center text-white mb-2">
          Get Started in 30 Seconds
        </h2>
        <p className="text-center text-gray-400 mb-8">
          One command to install. No manual downloads.
        </p>

        {/* Tabs */}
        <div className="flex justify-center gap-1 mb-6">
          {tabs.map((tab) => (
            <button
              key={tab}
              onClick={() => setActiveTab(tab)}
              className={`px-4 py-2 text-sm font-medium transition-colors ${
                activeTab === tab
                  ? 'text-white border-b-2 border-indigo-500'
                  : 'text-gray-500 hover:text-gray-300'
              }`}
            >
              {tab}
            </button>
          ))}
        </div>

        {/* Terminal block */}
        <div className="relative bg-gray-950 border border-gray-800 rounded-xl p-5 overflow-x-auto">
          {/* Window dots */}
          <div className="flex gap-1.5 mb-4">
            <span className="w-3 h-3 rounded-full bg-red-500/80" />
            <span className="w-3 h-3 rounded-full bg-yellow-500/80" />
            <span className="w-3 h-3 rounded-full bg-green-500/80" />
          </div>

          <div className="flex items-start gap-2">
            <span className="text-gray-500 font-mono text-sm select-none shrink-0">$</span>
            <code className="font-mono text-sm text-green-400 break-all">{command}</code>
          </div>

          {/* Copy button */}
          <button
            onClick={handleCopy}
            className="absolute top-4 right-4 rounded-md border border-gray-700 px-3 py-1 text-xs text-gray-400 hover:text-white hover:border-gray-500 transition-colors"
          >
            {copied ? 'Copied!' : 'Copy'}
          </button>
        </div>

        <p className="mt-4 text-center text-sm text-gray-500">
          Then run: <code className="font-mono text-gray-400">bytebrew</code>
        </p>
      </div>
    </section>
  );
}

function FeatureCard({ title, description }: { title: string; description: string }) {
  return (
    <div className="rounded-xl border border-gray-800 bg-gray-900/50 p-6">
      <h3 className="text-base font-semibold text-white">{title}</h3>
      <p className="mt-2 text-sm text-gray-400 leading-relaxed">{description}</p>
    </div>
  );
}
