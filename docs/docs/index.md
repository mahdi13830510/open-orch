---
layout: home

hero:
  name: "Open-Orch"
  text: "Preview environments,\nautomated."
  tagline: "Spin up a full-stack isolated environment for every branch. Powered by Docker, Traefik, and GitHub webhooks — zero config per PR."
  image:
    src: /logo.svg
    alt: Open-Orch
  actions:
    - theme: brand
      text: Get Started →
      link: /intro
    - theme: alt
      text: Architecture
      link: /architecture/overview
    - theme: alt
      text: GitHub
      link: https://github.com/mahdi13830510/open-orch

features:
  - icon: ⚡
    title: Instant Environments
    details: A GitHub push triggers a fully isolated Docker environment in seconds. Each feature branch gets its own network, containers, and HTTPS subdomain.

  - icon: 🔀
    title: Smart Branch Resolution
    details: When a service doesn't have a matching branch, Open-Orch falls back to main, latest stable tag, or auto-creates a PR — configurable per repo.

  - icon: 🔒
    title: Encrypted Credentials
    details: Third-party secrets (GitHub App keys, Buddy tokens, registry creds) are stored AES-256-GCM encrypted. Raw secrets never leave the server.

  - icon: 🔁
    title: Continuous Reconciliation
    details: A distributed reconciler runs every 15 seconds per environment, converging Docker state to the desired topology stored in PostgreSQL.

  - icon: 🌐
    title: Automatic TLS
    details: Traefik handles wildcard DNS-01 ACME via Cloudflare. Every preview environment gets a valid HTTPS subdomain with zero manual cert work.

  - icon: ⏱️
    title: TTL Lifecycle
    details: Environments expire automatically. Default 72-hour TTL, configurable per environment. Suspend/resume without teardown for flaky branches.

  - icon: 📊
    title: React Dashboard
    details: A full-featured UI for managing environments, repositories, integrations, and audit logs — with live topology visualization.

  - icon: 🏗️
    title: Distributed & Scalable
    details: Multiple orchestrator instances coordinate via PostgreSQL row locks and Redis leases. Scale reconciler workers independently from the API.
---

<div style="max-width:900px;margin:0 auto;padding:0 1.5rem;">

## How it works

```mermaid
sequenceDiagram
    participant Dev as 👩‍💻 Developer
    participant GH as GitHub
    participant OA as Open-Orch API
    participant Rec as Reconciler
    participant D as Docker

    Dev->>GH: git push feature/my-branch
    GH->>OA: POST /webhooks/github
    OA->>OA: Normalize branch → feature slug
    OA->>OA: Resolve topology (pick branches per service)
    OA-->>GH: 202 Accepted

    loop Every 15s
        Rec->>OA: Poll pending environments
        Rec->>D: EnsureNetwork + RunContainers
        D-->>Rec: Containers healthy
        Rec->>OA: State → healthy
    end

    Dev->>Dev: 🎉 Preview URL ready in ~30s
```

## Stack at a glance

| Layer | Technology |
|-------|-----------|
| Control plane | Go · chi · zerolog |
| State store | PostgreSQL · pgx |
| Distributed locks | Redis |
| Build executor | Buddy.works |
| Runtime | Docker Engine |
| Ingress + TLS | Traefik v3 · Let's Encrypt |
| Dashboard | React 19 · Vite · Tailwind |
| Docs | VitePress · Mermaid |

## Quick start

```bash
git clone https://github.com/mahdi13830510/open-orch.git
cd open-orch/backend

# Configure
cp deploy/.env.example deploy/.env
# Edit: BASE_DOMAIN, GITHUB_WEBHOOK_SECRET, BUDDY_TOKEN, ORCH_SECRET_KEY

# Launch
docker compose -f deploy/docker-compose.yml up -d --build

# Check
curl https://api.your-domain.com/healthz
# {"status":"ok"}
```

</div>
