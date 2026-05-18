import { defineConfig } from 'vitepress';
import { withMermaid } from 'vitepress-plugin-mermaid';

export default withMermaid(
  defineConfig({
    title: 'Open-Orch',
    description: 'Open-source control plane for ephemeral preview environments',
    base: '/open-orch/',

    srcDir: 'docs',
    outDir: '.vitepress/dist',

    themeConfig: {
      logo: '/logo.svg',
      siteTitle: 'Open-Orch',

      nav: [
        { text: 'Docs', link: '/intro' },
        { text: 'API', link: '/api/reference' },
        { text: 'GitHub', link: 'https://github.com/mahdi13830510/open-orch' },
      ],

      sidebar: [
        { text: 'Introduction', link: '/intro' },
        {
          text: 'Architecture',
          items: [
            { text: 'Overview', link: '/architecture/overview' },
            { text: 'Data Flow', link: '/architecture/data-flow' },
            { text: 'Environment Lifecycle', link: '/architecture/environment-lifecycle' },
          ],
        },
        {
          text: 'API Reference',
          items: [
            { text: 'Endpoints', link: '/api/reference' },
          ],
        },
        {
          text: 'Deployment',
          items: [
            { text: 'Docker Compose', link: '/deployment/docker-compose' },
            { text: 'Configuration', link: '/deployment/configuration' },
          ],
        },
      ],

      socialLinks: [
        { icon: 'github', link: 'https://github.com/mahdi13830510/open-orch' },
      ],

      footer: {
        message: 'Released under the MIT License.',
        copyright: 'Copyright © 2026 Open-Orch Contributors',
      },

      search: {
        provider: 'local',
      },
    },

    mermaid: {
      theme: 'neutral',
    },
  }),
);
