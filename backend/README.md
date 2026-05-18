# Open-Orch Backend

Control-plane for ephemeral preview environments. GitHub is the event source, Buddy.works is the executor, Docker is the runtime, Traefik is the ingress, PostgreSQL is the source of truth, Redis is the queue/lock substrate.

## Layout

```
cmd/orchestrator/        entrypoint, wiring, graceful shutdown
internal/api/            REST API (chi) + CORS middleware
internal/buddy/          Buddy.works pipeline trigger / status client
internal/config/         env-driven config
internal/db/             pgx-backed stores (repositories, prs, envs, events, ...)
internal/deps/           repository dependency graph + topo-sort
internal/docker/         docker SDK driver (networks, containers, labels)
internal/events/         decoupled event processor (FOR UPDATE SKIP LOCKED)
internal/github/         webhook signature verification + persistence
internal/lifecycle/      finite state machine for environment states
internal/locks/          redis distributed locks (token + lua release)
internal/models/         domain types
internal/normalizer/     branch -> DNS-safe feature slug
internal/reconciler/     continuous reconciliation loop + cleaner
internal/topology/       branch resolution + topology planning
internal/traefik/        dynamic label generation
migrations/              initial schema
deploy/                  docker-compose stack with traefik + cloudflare DNS-01
```

## Run locally

```bash
cp deploy/.env.example deploy/.env
# fill in BASE_DOMAIN, GITHUB_WEBHOOK_SECRET, BUDDY_TOKEN, etc.
docker compose -f deploy/docker-compose.yml up -d --build
```

## Environment variables

| Variable                     | Default                   | Description                          |
|------------------------------|---------------------------|--------------------------------------|
| `ORCH_LISTEN_ADDR`           | `:8080`                   | HTTP bind address                    |
| `ORCH_POSTGRES_DSN`          | local dev DSN             | PostgreSQL connection string         |
| `ORCH_REDIS_ADDR`            | `localhost:6379`          | Redis address                        |
| `ORCH_BASE_DOMAIN`           | `localhost`               | Wildcard domain for preview URLs     |
| `ORCH_GITHUB_APP_ID`         | —                         | GitHub App ID                        |
| `ORCH_GITHUB_PRIVATE_KEY`    | —                         | GitHub App PEM private key           |
| `ORCH_GITHUB_WEBHOOK_SECRET` | —                         | GitHub webhook HMAC secret           |
| `ORCH_BUDDY_BASE_URL`        | `https://api.buddy.works` | Buddy API base URL                   |
| `ORCH_BUDDY_TOKEN`           | —                         | Buddy personal access token          |
| `ORCH_BUDDY_WORKSPACE`       | —                         | Buddy workspace slug                 |
| `ORCH_SECRET_KEY`            | —                         | AES-256 key for storing secrets      |
| `ORCH_CORS_ORIGINS`          | `http://localhost:3000`   | Comma-separated allowed CORS origins |
| `ORCH_DEFAULT_FALLBACK`      | `main`                    | Branch fallback strategy             |
| `ORCH_RECONCILE_INTERVAL`    | `15s`                     | Reconciler tick                      |
| `ORCH_DEFAULT_ENV_TTL`       | `72h`                     | Default environment lifetime         |
