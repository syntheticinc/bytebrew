import { Link } from '@tanstack/react-router';
import { EXAMPLES } from '../data/examples';

export function ExamplesPage() {
  return (
    <div>
      {/* Header */}
      <section className="py-16 px-4 text-center">
        <h1 className="text-4xl font-bold tracking-tight text-brand-light">
          Examples
        </h1>
        <p className="mt-4 text-brand-shade2 max-w-xl mx-auto">
          See ByteBrew Engine in action. Each demo is a fully working agent you can try live and run yourself.
        </p>
      </section>

      {/* Cards grid */}
      <section className="px-4 pb-20">
        <div className="max-w-5xl mx-auto grid gap-6 sm:grid-cols-2 lg:grid-cols-3">
          {EXAMPLES.map((example) => (
            <Link
              key={example.slug}
              to="/examples/$slug"
              params={{ slug: example.slug }}
              className="group rounded-[12px] border border-brand-shade3/15 bg-brand-dark-alt p-6 flex flex-col gap-4 hover:border-brand-accent/30 transition-colors"
            >
              {/* Icon + title */}
              <div className="flex items-center gap-3">
                <span className="text-3xl" role="img" aria-label={example.title}>
                  {example.icon}
                </span>
                <div>
                  <h2 className="text-lg font-bold text-brand-light group-hover:text-brand-accent transition-colors">
                    {example.title}
                  </h2>
                  <p className="text-sm text-brand-shade2 mt-0.5">{example.subtitle}</p>
                </div>
              </div>

              {/* Feature tags */}
              <div className="flex flex-wrap gap-1.5">
                {example.features.map((feature) => (
                  <span
                    key={feature}
                    className="rounded-full border border-brand-shade3/20 px-2.5 py-0.5 text-[11px] text-brand-shade2"
                  >
                    {feature}
                  </span>
                ))}
              </div>

              {/* CTA */}
              <div className="mt-auto pt-2">
                <span className="inline-flex items-center gap-1 text-sm font-medium text-brand-accent group-hover:gap-2 transition-all">
                  Try Demo
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
                      d="M13.5 4.5L21 12m0 0l-7.5 7.5M21 12H3"
                    />
                  </svg>
                </span>
              </div>
            </Link>
          ))}
        </div>
      </section>
    </div>
  );
}
