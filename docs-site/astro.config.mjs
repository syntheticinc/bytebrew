// @ts-check
import { defineConfig } from 'astro/config';
import starlight from '@astrojs/starlight';
import starlightLlmsTxt from 'starlight-llms-txt';

export default defineConfig({
  site: 'https://bytebrew.ai',
  base: '/docs',
  integrations: [
    starlight({
      title: 'ByteBrew Engine',
      plugins: [
        starlightLlmsTxt({
          projectName: 'ByteBrew Engine',
          description: 'Self-hosted AI agent runtime. Multi-agent orchestration, tool calling (MCP), SSE streaming, BYOLLM, session memory. Deploy with Docker, integrate via REST API.',
        }),
      ],
      logo: {
        dark: './src/assets/logo.svg',
        light: './src/assets/logo-light.png',
        replacesTitle: true,
      },
      social: [
        { icon: 'github', label: 'GitHub', href: 'https://github.com/syntheticinc/bytebrew' },
      ],
      customCss: ['./src/styles/custom.css'],
      head: [
        {
          tag: 'script',
          content: `(function() {
            var bwTheme = localStorage.getItem('bytebrew-theme');
            if (bwTheme === 'light' || bwTheme === 'dark') {
              var slTheme = localStorage.getItem('starlight-theme');
              if (slTheme !== bwTheme) {
                localStorage.setItem('starlight-theme', bwTheme);
                document.documentElement.dataset.theme = bwTheme;
              }
            }
            var obs = new MutationObserver(function(mutations) {
              mutations.forEach(function(m) {
                if (m.attributeName === 'data-theme') {
                  var t = document.documentElement.dataset.theme;
                  if (t === 'dark' || t === 'light') {
                    localStorage.setItem('bytebrew-theme', t);
                  }
                }
              });
            });
            obs.observe(document.documentElement, { attributes: true });
          })();`,
        },
        {
          tag: 'script',
          content: `(function() {
            function fixLogo() {
              var logo = document.querySelector('a.site-title');
              if (logo) logo.href = 'https://bytebrew.ai';
            }
            fixLogo();
            document.addEventListener('DOMContentLoaded', fixLogo);
            if (document.readyState !== 'loading') fixLogo();
          })();`,
        },
        {
          tag: 'script',
          content: `(function() {
            function initLightbox() {
              var overlay = document.createElement('div');
              overlay.id = 'img-lightbox';
              overlay.style.cssText = 'display:none;position:fixed;inset:0;z-index:9999;background:rgba(0,0,0,0.85);backdrop-filter:blur(4px);cursor:pointer;justify-content:center;align-items:center;';
              var img = document.createElement('img');
              img.style.cssText = 'max-width:90vw;max-height:90vh;object-fit:contain;border-radius:8px;box-shadow:0 25px 50px rgba(0,0,0,0.5);';
              overlay.appendChild(img);
              document.body.appendChild(overlay);
              overlay.addEventListener('click', function() {
                overlay.style.display = 'none';
              });
              // Click anywhere (including on image) closes lightbox
              document.addEventListener('keydown', function(e) {
                if (e.key === 'Escape') overlay.style.display = 'none';
              });
              document.querySelectorAll('main img, .sl-markdown-content img').forEach(function(el) {
                if (el.width < 100 || el.closest('a')) return;
                el.style.cursor = 'pointer';
                el.addEventListener('click', function() {
                  img.src = el.src;
                  overlay.style.display = 'flex';
                });
              });
            }
            if (document.readyState === 'loading') {
              document.addEventListener('DOMContentLoaded', initLightbox);
            } else {
              initLightbox();
            }
          })();`,
        },
      ],
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
            { label: 'Model Registry', slug: 'deployment/model-registry' },
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
            { label: 'HR Assistant', slug: 'examples/hr-assistant' },
            { label: 'Support Agent', slug: 'examples/support-agent' },
            { label: 'Sales Agent', slug: 'examples/sales-agent' },
          ],
        },
      ],
    }),
  ],
});
