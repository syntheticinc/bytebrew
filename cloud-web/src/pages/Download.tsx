import { useState } from 'react';
import { TerminalBlock } from '../components/TerminalBlock';
import { SHOW_EE_PRICING } from '../lib/feature-flags';

type Tab = 'docker' | 'binary';

export function DownloadPage() {
  const [activeTab, setActiveTab] = useState<Tab>('docker');

  return (
    <div className="max-w-3xl mx-auto px-4 py-16">
      {/* Header */}
      <h1 className="text-3xl font-bold text-text-primary text-center">Install ByteBrew Engine</h1>
      <p className="mt-3 text-center text-text-secondary text-lg">
        One command. Full AI agent runtime. Free forever.
      </p>

      {/* Tab switcher */}
      <div className="mt-10 flex gap-2 border-b border-border">
        <button
          onClick={() => setActiveTab('docker')}
          className={`px-4 py-2.5 text-sm font-medium transition-colors relative ${
            activeTab === 'docker'
              ? 'text-text-primary border-b-2 border-brand-accent'
              : 'text-text-secondary hover:text-text-secondary'
          }`}
        >
          Docker Compose
          <span className="ml-2 inline-block text-[10px] font-semibold uppercase tracking-wide bg-green-600/20 text-green-400 border border-green-500/30 rounded-full px-2 py-0.5">
            Recommended
          </span>
        </button>
        <button
          onClick={() => setActiveTab('binary')}
          className={`px-4 py-2.5 text-sm font-medium transition-colors ${
            activeTab === 'binary'
              ? 'text-text-primary border-b-2 border-brand-accent'
              : 'text-text-secondary hover:text-text-secondary'
          }`}
        >
          Binary
        </button>
      </div>

      {/* Docker Compose tab */}
      {activeTab === 'docker' && (
        <div className="mt-8 space-y-8">
          {/* Step 1 */}
          <div>
            <div className="flex items-center gap-3 mb-3">
              <span className="flex items-center justify-center w-7 h-7 rounded-full bg-brand-accent/20 text-brand-accent text-sm font-bold shrink-0">
                1
              </span>
              <h3 className="text-sm font-medium text-text-primary">Download configuration</h3>
            </div>
            <TerminalBlock command="curl -fsSL https://bytebrew.ai/releases/docker-compose.yml -o docker-compose.yml" />
          </div>

          {/* Step 2 */}
          <div>
            <div className="flex items-center gap-3 mb-3">
              <span className="flex items-center justify-center w-7 h-7 rounded-full bg-brand-accent/20 text-brand-accent text-sm font-bold shrink-0">
                2
              </span>
              <h3 className="text-sm font-medium text-text-primary">Start Engine + PostgreSQL</h3>
            </div>
            <TerminalBlock command="docker compose up -d" />
          </div>

          {/* Info box */}
          <div className="rounded-[2px] border border-border bg-surface-alt p-5 text-sm text-text-secondary">
            <p className="font-medium text-text-primary mb-1">What&apos;s included</p>
            <p>ByteBrew Engine + PostgreSQL 16. Ready in 30 seconds.</p>
          </div>

          {/* Step 3 */}
          <div>
            <div className="flex items-center gap-3 mb-3">
              <span className="flex items-center justify-center w-7 h-7 rounded-full bg-brand-accent/20 text-brand-accent text-sm font-bold shrink-0">
                3
              </span>
              <h3 className="text-sm font-medium text-text-primary">Open admin dashboard</h3>
            </div>
            <p className="text-sm text-text-secondary">
              Open{' '}
              <code className="text-brand-accent bg-surface-alt px-1.5 py-0.5 rounded text-xs">
                http://localhost:8443/admin
              </code>{' '}
              to configure your first agent.
            </p>
          </div>

          {/* Upgrade */}
          <div className="pt-4 border-t border-border">
            <h3 className="text-sm font-medium text-text-primary mb-3">Upgrade to latest version</h3>
            <TerminalBlock command="docker compose pull && docker compose up -d" />
          </div>
        </div>
      )}

      {/* Binary tab */}
      {activeTab === 'binary' && (
        <div className="mt-8 space-y-8">
          {/* Prerequisites */}
          <div className="rounded-[2px] border border-yellow-500/20 bg-yellow-500/5 p-5 text-sm text-text-secondary">
            <p className="font-medium text-yellow-400 mb-1">Prerequisites</p>
            <p>Requires PostgreSQL 14+ (existing instance).</p>
          </div>

          {/* Linux/macOS */}
          <div>
            <h3 className="text-sm font-medium text-text-primary mb-3">Linux / macOS</h3>
            <TerminalBlock command="curl -fsSL https://bytebrew.ai/releases/install.sh | bash" />
          </div>

          {/* Windows */}
          <div>
            <h3 className="text-sm font-medium text-text-primary mb-3">Windows (PowerShell)</h3>
            <TerminalBlock
              command="irm https://bytebrew.ai/releases/install.ps1 | iex"
              prefix=">"
            />
          </div>

          {/* After install */}
          <div>
            <h3 className="text-sm font-medium text-text-primary mb-3">After install</h3>
            <div className="bg-surface border border-border rounded-[2px] p-5 font-mono text-sm text-text-secondary space-y-1 overflow-x-auto">
              <p>
                <span className="text-text-tertiary">1.</span>{' '}
                <span className="text-text-secondary">Set database:</span>{' '}
                <span className="text-green-400">export DATABASE_URL=&quot;postgresql://user:pass@host:5432/bytebrew&quot;</span>
              </p>
              <p>
                <span className="text-text-tertiary">2.</span>{' '}
                <span className="text-text-secondary">Set model key:</span>{' '}
                <span className="text-green-400">export LLM_API_KEY=sk-...</span>
              </p>
              <p>
                <span className="text-text-tertiary">3.</span>{' '}
                <span className="text-text-secondary">Start:</span>{' '}
                <span className="text-green-400">bytebrew-engine start</span>
              </p>
              <p>
                <span className="text-text-tertiary">4.</span>{' '}
                <span className="text-text-secondary">Open admin:</span>{' '}
                <span className="text-green-400">http://localhost:8443/admin</span>
              </p>
            </div>
          </div>

          {/* Note */}
          <p className="text-sm text-text-tertiary">
            Engine automatically creates all tables on first startup. No manual SQL needed.
          </p>
        </div>
      )}

      {/* System Requirements */}
      <div className="mt-16">
        <h2 className="text-lg font-semibold text-text-primary mb-4">System Requirements</h2>
        <div className="overflow-x-auto">
          <table className="w-full text-sm text-left">
            <thead>
              <tr className="border-b border-border">
                <th className="py-3 pr-4 font-medium text-text-secondary">Resource</th>
                <th className="py-3 pr-4 font-medium text-text-secondary">Docker Compose</th>
                <th className="py-3 font-medium text-text-secondary">Binary</th>
              </tr>
            </thead>
            <tbody className="text-text-secondary">
              <tr className="border-b border-border">
                <td className="py-3 pr-4 text-text-primary">Docker</td>
                <td className="py-3 pr-4">20.10+</td>
                <td className="py-3">Not needed</td>
              </tr>
              <tr className="border-b border-border">
                <td className="py-3 pr-4 text-text-primary">CPU</td>
                <td className="py-3 pr-4">1 core (2+ recommended)</td>
                <td className="py-3">Same</td>
              </tr>
              <tr className="border-b border-border">
                <td className="py-3 pr-4 text-text-primary">RAM</td>
                <td className="py-3 pr-4">1 GB (2 GB+ recommended)</td>
                <td className="py-3">512 MB + your PG</td>
              </tr>
              <tr className="border-b border-border">
                <td className="py-3 pr-4 text-text-primary">Disk</td>
                <td className="py-3 pr-4">500 MB + data</td>
                <td className="py-3">100 MB + your PG</td>
              </tr>
              <tr className="border-b border-border">
                <td className="py-3 pr-4 text-text-primary">PostgreSQL</td>
                <td className="py-3 pr-4">Included</td>
                <td className="py-3">14+ (bring your own)</td>
              </tr>
              <tr>
                <td className="py-3 pr-4 text-text-primary">Network</td>
                <td className="py-3 pr-4">Outbound to LLM provider</td>
                <td className="py-3">Same</td>
              </tr>
            </tbody>
          </table>
        </div>
      </div>

      {/* Bottom links */}
      <div className="mt-12 flex flex-wrap gap-4">
        <span className="text-sm text-text-tertiary cursor-not-allowed">
          Your First Agent in 5 Minutes
        </span>
        <span className="text-sm text-text-tertiary cursor-not-allowed">
          Configuration Reference
        </span>
        <span className="text-sm text-text-tertiary cursor-not-allowed">
          API Reference
        </span>
      </div>

      {/* EE upgrade section */}
      {SHOW_EE_PRICING && (
        <div className="mt-16 pt-8 border-t border-border">
          <h2 className="text-lg font-semibold text-text-primary mb-4">Upgrade to Enterprise Edition</h2>
          <div className="bg-surface border border-border rounded-[2px] p-5 font-mono text-sm text-text-secondary space-y-1 overflow-x-auto">
            <p>
              <span className="text-text-tertiary">1.</span>{' '}
              Subscribe at{' '}
              <span className="text-brand-accent">bytebrew.ai/pricing</span>
            </p>
            <p>
              <span className="text-text-tertiary">2.</span>{' '}
              Download <span className="text-green-400">license.jwt</span> from your dashboard
            </p>
            <p>
              <span className="text-text-tertiary">3.</span>{' '}
              Place in <span className="text-green-400">~/.bytebrew/license.jwt</span>
            </p>
            <p>
              <span className="text-text-tertiary">4.</span>{' '}
              Restart Engine — observability and compliance features are now active
            </p>
          </div>
        </div>
      )}
    </div>
  );
}
