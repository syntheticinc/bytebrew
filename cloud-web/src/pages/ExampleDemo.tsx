import { Link, useParams } from '@tanstack/react-router';
import { getExampleBySlug } from '../data/examples';
import { ExampleChat } from '../components/ExampleChat';
import { TerminalBlock } from '../components/TerminalBlock';

export function ExampleDemoPage() {
  const { slug } = useParams({ strict: false }) as { slug: string };
  const example = getExampleBySlug(slug);

  if (!example) {
    return (
      <div className="py-24 px-4 text-center">
        <h1 className="text-2xl font-bold text-brand-light">Example not found</h1>
        <p className="mt-4 text-brand-shade2">
          The example "{slug}" does not exist.
        </p>
        <Link
          to="/examples"
          className="mt-6 inline-block text-sm text-brand-accent hover:text-brand-accent-hover transition-colors"
        >
          Back to Examples
        </Link>
      </div>
    );
  }

  return (
    <div className="py-10 px-4">
      <div className="max-w-3xl mx-auto space-y-8">
        {/* Back link + title */}
        <div>
          <Link
            to="/examples"
            className="inline-flex items-center gap-1.5 text-sm text-brand-shade2 hover:text-brand-light transition-colors mb-4"
          >
            <svg
              className="h-4 w-4"
              fill="none"
              viewBox="0 0 24 24"
              strokeWidth={2}
              stroke="currentColor"
            >
              <path
                strokeLinecap="round"
                strokeLinejoin="round"
                d="M10.5 19.5L3 12m0 0l7.5-7.5M3 12h18"
              />
            </svg>
            Back to Examples
          </Link>

          <div className="flex items-center gap-3">
            <span className="text-4xl" role="img" aria-label={example.title}>
              {example.icon}
            </span>
            <div>
              <h1 className="text-3xl font-bold text-brand-light">{example.title}</h1>
              <p className="text-brand-shade2 mt-1">{example.subtitle}</p>
            </div>
          </div>
        </div>

        {/* What this demonstrates */}
        <div className="rounded-[2px] border border-brand-shade3/15 bg-brand-dark-alt p-5">
          <h2 className="text-sm font-medium text-brand-shade2 uppercase tracking-wider mb-3">
            What this demonstrates
          </h2>
          <p className="text-sm text-brand-light leading-relaxed mb-4">
            {example.description}
          </p>
          <div className="flex flex-wrap gap-2">
            {example.features.map((feature) => (
              <span
                key={feature}
                className="inline-flex items-center gap-1.5 rounded-full border border-brand-accent/20 bg-brand-accent/5 px-3 py-1 text-xs text-brand-accent"
              >
                <svg
                  className="h-3 w-3"
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
                {feature}
              </span>
            ))}
          </div>
        </div>

        {/* Chat */}
        <div>
          <h2 className="text-sm font-medium text-brand-shade2 uppercase tracking-wider mb-3">
            Live Demo
          </h2>
          <ExampleChat
            agentName={example.agentName}
            apiUrl={example.apiUrl}
            suggestions={example.suggestions}
          />
        </div>

        {/* Run it yourself */}
        <div className="rounded-[2px] border border-brand-shade3/15 bg-brand-dark-alt p-5">
          <h2 className="text-sm font-medium text-brand-shade2 uppercase tracking-wider mb-4">
            Run it yourself
          </h2>
          <TerminalBlock command={example.setupCommands.join(' && ')} />
          <div className="mt-4 flex justify-center">
            <a
              href={example.githubUrl}
              target="_blank"
              rel="noopener noreferrer"
              className="inline-flex items-center gap-2 rounded-[2px] border border-brand-shade3/20 px-4 py-2 text-sm text-brand-shade2 hover:text-brand-light hover:border-brand-shade3/40 transition-colors"
            >
              <svg className="h-4 w-4" viewBox="0 0 24 24" fill="currentColor">
                <path d="M12 0c-6.626 0-12 5.373-12 12 0 5.302 3.438 9.8 8.207 11.387.599.111.793-.261.793-.577v-2.234c-3.338.726-4.033-1.416-4.033-1.416-.546-1.387-1.333-1.756-1.333-1.756-1.089-.745.083-.729.083-.729 1.205.084 1.839 1.237 1.839 1.237 1.07 1.834 2.807 1.304 3.492.997.107-.775.418-1.305.762-1.604-2.665-.305-5.467-1.334-5.467-5.931 0-1.311.469-2.381 1.236-3.221-.124-.303-.535-1.524.117-3.176 0 0 1.008-.322 3.301 1.23.957-.266 1.983-.399 3.003-.404 1.02.005 2.047.138 3.006.404 2.291-1.552 3.297-1.23 3.297-1.23.653 1.653.242 2.874.118 3.176.77.84 1.235 1.911 1.235 3.221 0 4.609-2.807 5.624-5.479 5.921.43.372.823 1.102.823 2.222v3.293c0 .319.192.694.801.576 4.765-1.589 8.199-6.086 8.199-11.386 0-6.627-5.373-12-12-12z" />
              </svg>
              View on GitHub
            </a>
          </div>
        </div>
      </div>
    </div>
  );
}
