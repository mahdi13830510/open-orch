# Configuration Reference

All configuration is supplied via environment variables. The binary reads them at startup — no config file.

## Backend environment variables

| Variable | Default | Required | Description |
|----------|---------|----------|-------------|
| `ORCH_LISTEN_ADDR` | `:8080` | | HTTP bind address |
| `ORCH_POSTGRES_DSN` | local dev | ✅ | PostgreSQL connection string |
| `ORCH_REDIS_ADDR` | `localhost:6379` | ✅ | Redis address |
| `ORCH_REDIS_PASSWORD` | | | Redis password |
| `ORCH_REDIS_DB` | `0` | | Redis database index |
| `ORCH_BASE_DOMAIN` | `localhost` | ✅ | Root domain for preview URLs (e.g. `preview.example.com`) |
| `ORCH_GITHUB_APP_ID` | | ✅ | GitHub App numeric ID |
| `ORCH_GITHUB_PRIVATE_KEY` | | ✅ | GitHub App RSA private key (PEM) |
| `ORCH_GITHUB_WEBHOOK_SECRET` | | ✅ | GitHub webhook HMAC secret |
| `ORCH_BUDDY_BASE_URL` | `https://api.buddy.works` | | Buddy API base URL |
| `ORCH_BUDDY_TOKEN` | | ✅ | Buddy personal access token |
| `ORCH_BUDDY_WORKSPACE` | | ✅ | Buddy workspace slug |
| `ORCH_DOCKER_HOST` | `unix:///var/run/docker.sock` | | Docker socket or TCP address |
| `ORCH_SECRET_KEY` | | ✅ | AES-256 key for integration secrets (32-byte hex or base64) |
| `ORCH_CORS_ORIGINS` | `http://localhost:3000` | | Comma-separated allowed CORS origins |
| `ORCH_DEFAULT_FALLBACK` | `main` | | Branch fallback: `main` \| `latest_stable` \| `auto_create` \| `previous_compatible` |
| `ORCH_RECONCILE_INTERVAL` | `15s` | | Reconciler tick interval |
| `ORCH_EVENT_POLL_INTERVAL` | `2s` | | Event processor poll interval |
| `ORCH_DEFAULT_ENV_TTL` | `72h` | | Default environment lifetime |
| `ORCH_LOCK_TTL` | `60s` | | Redis distributed lock TTL |
| `ORCH_WORKER_ID` | hostname | | Unique ID for this worker instance |

## Generate a secret key

```bash
# Hex (recommended)
openssl rand -hex 32

# Base64
openssl rand -base64 32
```

## Frontend environment variables

| Variable | Default | Description |
|----------|---------|-------------|
| `VITE_API_URL` | `http://localhost:8080` | Open-Orch backend API URL (no trailing slash) |

## Docker Compose variables (deploy/.env)

These are substituted into `docker-compose.yml` at runtime:

| Variable | Description |
|----------|-------------|
| `POSTGRES_PASSWORD` | PostgreSQL superuser password |
| `BASE_DOMAIN` | Root domain (maps to `ORCH_BASE_DOMAIN`) |
| `ACME_EMAIL` | Email for Let's Encrypt ACME |
| `CF_DNS_API_TOKEN` | Cloudflare DNS API token for DNS-01 challenge |
| `GITHUB_APP_ID` | GitHub App ID |
| `GITHUB_PRIVATE_KEY` | GitHub App private key |
| `GITHUB_WEBHOOK_SECRET` | Webhook HMAC secret |
| `BUDDY_TOKEN` | Buddy API token |
| `BUDDY_WORKSPACE` | Buddy workspace |
| `ORCH_SECRET_KEY` | AES-256 encryption key |
| `CORS_ORIGINS` | Frontend URL(s) for CORS |
