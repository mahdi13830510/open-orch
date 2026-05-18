import { themes as prismThemes } from 'prism-react-renderer';
import type { Config } from '@docusaurus/types';
import type * as Preset from '@docusaurus/preset-classic';

const config: Config = {
  title: 'Open-Orch',
  tagline: 'Open-source control plane for ephemeral preview environments',
  favicon: 'img/favicon.ico',

  url: 'https://mahdi13830510.github.io',
  baseUrl: '/open-orch/',

  organizationName: 'mahdi13830510',
  projectName: 'open-orch',
  trailingSlash: false,

  onBrokenLinks: 'throw',
  onBrokenMarkdownLinks: 'warn',

  markdown: {
    mermaid: true,
  },

  themes: ['@docusaurus/theme-mermaid'],

  i18n: {
    defaultLocale: 'en',
    locales: ['en'],
  },

  presets: [
    [
      'classic',
      {
        docs: {
          sidebarPath: './sidebars.ts',
          editUrl: 'https://github.com/mahdi13830510/open-orch/edit/main/docs/',
          routeBasePath: '/',
        },
        blog: false,
        theme: {
          customCss: './src/css/custom.css',
        },
      } satisfies Preset.Options,
    ],
  ],

  themeConfig: {
    image: 'img/open-orch-social.png',

    mermaid: {
      theme: { light: 'neutral', dark: 'dark' },
      options: {
        maxTextSize: 50,
      },
    },

    navbar: {
      title: 'Open-Orch',
      logo: {
        alt: 'Open-Orch Logo',
        src: 'img/logo.svg',
      },
      items: [
        {
          type: 'docSidebar',
          sidebarId: 'docs',
          position: 'left',
          label: 'Docs',
        },
        {
          href: 'https://github.com/mahdi13830510/open-orch',
          label: 'GitHub',
          position: 'right',
        },
      ],
    },

    footer: {
      style: 'dark',
      links: [
        {
          title: 'Docs',
          items: [
            { label: 'Getting Started', to: '/' },
            { label: 'Architecture', to: '/architecture/overview' },
            { label: 'API Reference', to: '/api/reference' },
          ],
        },
        {
          title: 'Community',
          items: [
            { label: 'GitHub', href: 'https://github.com/mahdi13830510/open-orch' },
            { label: 'Issues', href: 'https://github.com/mahdi13830510/open-orch/issues' },
          ],
        },
      ],
      copyright: `Copyright © ${new Date().getFullYear()} Open-Orch Contributors. Built with Docusaurus.`,
    },

    prism: {
      theme: prismThemes.github,
      darkTheme: prismThemes.dracula,
      additionalLanguages: ['bash', 'go', 'typescript', 'yaml', 'sql'],
    },
  } satisfies Preset.ThemeConfig,
};

export default config;
