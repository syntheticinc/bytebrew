export interface ExampleConfig {
  slug: string;
  title: string;
  icon: string;
  subtitle: string;
  description: string;
  features: string[];
  agentName: string;
  apiUrl: string;
  suggestions: string[];
  githubUrl: string;
  setupCommands: string[];
}

export const EXAMPLES: ExampleConfig[] = [
  {
    slug: 'hr-assistant',
    title: 'HR Assistant',
    icon: '\u{1F3E2}',
    subtitle: 'AI-powered HR chatbot with company knowledge base',
    description:
      'Search company policies, check leave balance, submit requests, and escalate complex cases to HR specialists.',
    features: ['Knowledge Base (RAG)', 'Structured Q&A', 'Escalation', 'Task Management'],
    agentName: 'hr-assistant',
    apiUrl: '/examples/hr-assistant/api',
    suggestions: [
      "What's the PTO policy for employees with 2+ years?",
      'I want to request time off next week',
      'What are our health insurance options?',
      'How does the remote work policy work?',
    ],
    githubUrl: 'https://github.com/syntheticinc/bytebrew-examples/tree/main/hr-assistant',
    setupCommands: [
      'git clone https://github.com/syntheticinc/bytebrew-examples',
      'cd bytebrew-examples/hr-assistant',
      'docker compose up -d',
    ],
  },
  {
    slug: 'support-agent',
    title: 'Support Agent',
    icon: '\u{1F6E0}\u{FE0F}',
    subtitle: 'Multi-agent support system with parallel diagnostics',
    description:
      'Route issues to billing or technical specialists. Run parallel diagnostics for fast resolution.',
    features: ['Multi-Agent Spawn', 'Parallel Execution', 'Ticket System', '8 MCP Tools'],
    agentName: 'support-router',
    apiUrl: '/examples/support-agent/api',
    suggestions: [
      'My API is returning 500 errors since this morning',
      'I was double-charged on my last invoice',
      'How do I upgrade from Starter to Pro plan?',
      'The dashboard is loading very slowly',
    ],
    githubUrl: 'https://github.com/syntheticinc/bytebrew-examples/tree/main/support-agent',
    setupCommands: [
      'git clone https://github.com/syntheticinc/bytebrew-examples',
      'cd bytebrew-examples/support-agent',
      'docker compose up -d',
    ],
  },
  {
    slug: 'sales-agent',
    title: 'Sales Agent',
    icon: '\u{1F6D2}',
    subtitle: 'Sales assistant with approval workflows and configurable rules',
    description:
      'Search products, create quotes with confirmation, apply discounts with business rule validation.',
    features: ['Confirmation Gates', 'Business Rules', 'BYOK Models', 'Quote System'],
    agentName: 'sales-agent',
    apiUrl: '/examples/sales-agent/api',
    suggestions: [
      'I need 5 laptops for my team, budget $1200 each',
      'What monitors do you have in stock?',
      'Can I get a bulk discount on 10 keyboards?',
      'Show me your best-selling headsets',
    ],
    githubUrl: 'https://github.com/syntheticinc/bytebrew-examples/tree/main/sales-agent',
    setupCommands: [
      'git clone https://github.com/syntheticinc/bytebrew-examples',
      'cd bytebrew-examples/sales-agent',
      'docker compose up -d',
    ],
  },
];

export function getExampleBySlug(slug: string): ExampleConfig | undefined {
  return EXAMPLES.find((e) => e.slug === slug);
}
