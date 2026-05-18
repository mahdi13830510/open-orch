---
sidebar_position: 1
---

# Docker Compose Deployment

The canonical production deployment uses `backend/deploy/docker-compose.yml`, which runs:

```mermaid
graph TB
    subgraph open_orch_edge network
        OA[open-orch API :8080]
        PG[(PostgreSQL :5432)]
        RD[(Redis :6379)]
        TR[Traefik :80/:443]
    end

    GH[GitHub Webhooks] -->|HTTPS| TR
    TR -->|/webhooks/github| OA
    OA --> PG
    OA --> RD
    OA -->|/var/run/docker.sock| DK[Docker Engine]
    DK -->|containers with labels| TR
    TR -->|*.your-domain.com| Users
    CF[Cloudflare DNS] -.->|DNS-01 ACME| TR
```

## Prerequisites

- A Linux server with Docker Engine installed
- A domain with Cloudflare DNS
- A GitHub App (or webhook secret)
- A Buddy.works account with API token

## Steps

### 1. Clone and configure

```bash
git clone https://github.com/mahdi13830510/open-orch.git
cd open-orch/backend
cp deploy/.env.example deploy/.env
```

Edit `deploy/.env`:

```bash
POSTGRES_PASSWORD=change-me-strong
BASE_DOMAIN=preview.your-domain.com
ACME_EMAIL=ops@your-domain.com
CF_DNS_API_TOKEN=your-cloudflare-token

GITHUB_APP_ID=123456
GITHUB_PRIVATE_KEY=-----BEGIN RSA PRIVATE KEY-----...
GITHUB_WEBHOOK_SECRET=$(openssl rand -hex 32)

BUDDY_TOKEN=your-buddy-token
BUDDY_WORKSPACE=your-workspace

ORCH_SECRET_KEY=$(openssl rand -hex 32)

CORS_ORIGINS=https://dashboard.your-domain.com
```

### 2. Configure GitHub webhook

In your GitHub App or repo settings, set:
- **Payload URL**: `https://api.your-domain.com/webhooks/github`
- **Content type**: `application/json`
- **Secret**: the value of `GITHUB_WEBHOOK_SECRET`
- **Events**: `Push`, `Pull request`, `Create`, `Delete`

### 3. Start the stack

```bash
docker compose -f deploy/docker-compose.yml up -d --build
```

### 4. Verify

```bash
curl https://api.your-domain.com/healthz
# {"status":"ok"}
```

## Migrations

Migrations run automatically from `migrations/` on PostgreSQL startup (via `docker-entrypoint-initdb.d`). For updates, apply manually:

```bash
psql $ORCH_POSTGRES_DSN -f migrations/0002_integrations.sql
```

## Frontend deployment

Deploy the frontend as a static site. Example with Nginx or any static host:

```bash
cd frontend
VITE_API_URL=https://api.your-domain.com npm run build
# Serve dist/ from dashboard.your-domain.com
```
