import type { SidebarsConfig } from '@docusaurus/plugin-content-docs';

const sidebars: SidebarsConfig = {
  docs: [
    {
      type: 'doc',
      id: 'intro',
      label: 'Introduction',
    },
    {
      type: 'category',
      label: 'Architecture',
      items: [
        'architecture/overview',
        'architecture/data-flow',
        'architecture/environment-lifecycle',
      ],
    },
    {
      type: 'category',
      label: 'API Reference',
      items: ['api/reference'],
    },
    {
      type: 'category',
      label: 'Deployment',
      items: [
        'deployment/docker-compose',
        'deployment/configuration',
      ],
    },
  ],
};

export default sidebars;
