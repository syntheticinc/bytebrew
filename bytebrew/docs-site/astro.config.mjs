// @ts-check
import { defineConfig } from 'astro/config';
import starlight from '@astrojs/starlight';

export default defineConfig({
  site: 'https://bytebrew.ai',
  base: '/docs',
  integrations: [
    starlight({
      title: 'ByteBrew Engine',
      logo: { src: './src/assets/logo.svg', replacesTitle: true },
      social: [
        { icon: 'github', label: 'GitHub', href: 'https://github.com/syntheticinc/bytebrew' },
      ],
      customCss: ['./src/styles/custom.css'],
      sidebar: [
        {
          label: 'Getting Started',
          items: [
            { label: 'Quick Start', slug: 'getting-started/quick-start' },
            { label: 'Configuration', slug: 'getting-started/configuration' },
            { label: 'API Reference', slug: 'getting-started/api-reference' },
          ],
        },
        {
          label: 'Admin Dashboard',
          items: [
            { label: 'Login', slug: 'admin/login' },
            { label: 'Agents', slug: 'admin/agents' },
            { label: 'Models', slug: 'admin/models' },
            { label: 'MCP Servers', slug: 'admin/mcp-servers' },
            { label: 'Tasks', slug: 'admin/tasks' },
            { label: 'Triggers', slug: 'admin/triggers' },
            { label: 'API Keys', slug: 'admin/api-keys' },
            { label: 'Settings', slug: 'admin/settings' },
            { label: 'Config Management', slug: 'admin/config-management' },
            { label: 'Audit Log', slug: 'admin/audit-log' },
          ],
        },
        {
          label: 'Core Concepts',
          items: [
            { label: 'Agents & Lifecycle', slug: 'concepts/agents' },
            { label: 'Multi-Agent Orchestration', slug: 'concepts/multi-agent' },
            { label: 'Tools', slug: 'concepts/tools' },
            { label: 'Tasks & Jobs', slug: 'concepts/tasks' },
            { label: 'Knowledge / RAG', slug: 'concepts/knowledge' },
            { label: 'Triggers', slug: 'concepts/triggers' },
          ],
        },
        {
          label: 'Deployment',
          items: [
            { label: 'Docker', slug: 'deployment/docker' },
            { label: 'Model Selection', slug: 'deployment/model-selection' },
            { label: 'Production', slug: 'deployment/production' },
          ],
        },
        {
          label: 'Integration',
          items: [
            { label: 'REST API Chat', slug: 'integration/rest-api' },
            { label: 'Multi-Agent Config', slug: 'integration/multi-agent' },
            { label: 'BYOK', slug: 'integration/byok' },
          ],
        },
        {
          label: 'Examples',
          items: [
            { label: 'Sales Agent', slug: 'examples/sales-agent' },
            { label: 'Support Agent', slug: 'examples/support-agent' },
            { label: 'DevOps Monitor', slug: 'examples/devops-monitor' },
            { label: 'IoT Analyzer', slug: 'examples/iot-analyzer' },
          ],
        },
      ],
    }),
  ],
});
