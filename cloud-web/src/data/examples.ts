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
      'Watch the agent search policy documents, look up employee records, check leave balances, and submit requests — each step visible in real-time.',
    features: ['Knowledge Base (RAG)', 'Employee Lookup', 'Leave Management', 'Multi-step Workflows'],
    agentName: 'hr-assistant',
    apiUrl: import.meta.env.DEV ? 'http://localhost:3001/api' : '/examples/hr-assistant/api',
    suggestions: [
      "Check Alice Johnson's vacation balance",
      'Submit a sick day for Bob Martinez next Monday',
      "What's the PTO policy for employees with 5+ years?",
      'How many personal days does Emily Davis have left?',
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
      'See the agent identify issue type, run diagnostics on service health, analyze error logs, and create tickets — with parallel tool execution.',
    features: ['Parallel Diagnostics', 'Error Log Analysis', 'Knowledge Base', 'Ticket Management'],
    agentName: 'support-router',
    apiUrl: import.meta.env.DEV ? 'http://localhost:3002/api' : '/examples/support-agent/api',
    suggestions: [
      'Customer CUST-001 reports file uploads failing with timeouts',
      'CUST-003 was overcharged, check their billing and process a refund',
      'Search knowledge base for articles about password reset',
      'Create a high-priority ticket for CUST-002 about sync issues',
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
      'The agent searches products, checks inventory, validates discount rules, and creates quotes with confirmation gates — showing approval workflows.',
    features: ['Product Catalog', 'Inventory Checks', 'Business Rules', 'Quote Confirmation'],
    agentName: 'sales-agent',
    apiUrl: import.meta.env.DEV ? 'http://localhost:3003/api' : '/examples/sales-agent/api',
    suggestions: [
      'Find laptops under $1200 and check which are in stock',
      'Create a quote for 10 Keychron keyboards with maximum discount',
      'What monitors do you have? Check inventory for the Dell',
      'I need 3 webcams, find options and tell me about free shipping',
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
