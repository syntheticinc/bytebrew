import { useState } from 'react';
import { EnginePricingTable } from '../components/EnginePricingTable';

const FAQ_ITEMS = [
  {
    question: 'What happens if I stop paying for EE?',
    answer:
      'Your Engine continues working as Community Edition. All CE features remain fully functional. You only lose access to EE observability and compliance features. No data is deleted.',
  },
  {
    question: 'Is CE really free forever?',
    answer:
      'Yes. We publicly commit: CE features will never move behind a paywall. The Community Edition includes the full agent runtime, unlimited agents, unlimited sessions, and the complete Admin Dashboard.',
  },
  {
    question: 'Do you charge per agent or per session?',
    answer:
      'No. CE and EE have no limits on the number of agents, sessions, or API calls. You bring your own LLM API keys, so you control your model costs directly.',
  },
  {
    question: 'Can I self-host EE?',
    answer:
      'Yes. Both CE and EE are self-hosted. ByteBrew Engine runs as a single binary alongside PostgreSQL. There is no cloud dependency or phone-home requirement.',
  },
];

export function PricingPage() {
  return (
    <div>
      {/* Header */}
      <section className="py-16 px-4 text-center">
        <h1 className="text-4xl font-bold tracking-tight text-brand-light">Pricing</h1>
        <p className="mt-4 text-brand-shade2">
          Start free. Scale when you need observability and compliance.
        </p>
      </section>

      {/* Pricing Table */}
      <section className="px-4 pb-16">
        <EnginePricingTable />
      </section>

      {/* FAQ */}
      <section className="py-16 px-4 border-t border-brand-shade3/15">
        <div className="max-w-3xl mx-auto">
          <h2 className="text-2xl font-bold text-center text-brand-light mb-10">
            Frequently Asked Questions
          </h2>
          <div className="space-y-4">
            {FAQ_ITEMS.map((item) => (
              <FAQItem key={item.question} question={item.question} answer={item.answer} />
            ))}
          </div>
        </div>
      </section>
    </div>
  );
}

function FAQItem({ question, answer }: { question: string; answer: string }) {
  const [open, setOpen] = useState(false);

  return (
    <div className="rounded-[12px] border border-brand-shade3/15 bg-brand-dark-alt">
      <button
        onClick={() => setOpen(!open)}
        className="w-full flex items-center justify-between px-6 py-4 text-left"
      >
        <span className="text-sm font-medium text-brand-light">{question}</span>
        <svg
          className={`h-5 w-5 text-brand-shade2 shrink-0 transition-transform ${
            open ? 'rotate-180' : ''
          }`}
          fill="none"
          viewBox="0 0 24 24"
          strokeWidth={2}
          stroke="currentColor"
        >
          <path strokeLinecap="round" strokeLinejoin="round" d="M19.5 8.25l-7.5 7.5-7.5-7.5" />
        </svg>
      </button>
      {open && (
        <div className="px-6 pb-4">
          <p className="text-sm text-brand-shade2 leading-relaxed">{answer}</p>
        </div>
      )}
    </div>
  );
}
