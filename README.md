# Open-Orch

Open-source control plane for ephemeral preview environments.

## Structure

```
backend/    Go control-plane API (chi, pgx, Redis, Docker SDK)
frontend/   React dashboard (Vite, Tailwind, TypeScript)
```

## Quick start

```bash
# Backend
cd backend
cp deploy/.env.example deploy/.env
# fill in BASE_DOMAIN, GITHUB_WEBHOOK_SECRET, BUDDY_TOKEN, etc.
docker compose -f deploy/docker-compose.yml up -d --build

# Frontend
cd frontend
cp .env.example .env
# set VITE_API_URL=http://localhost:8080
npm install
npm run dev
```

## Ports

| Service     | Default         |
|-------------|-----------------|
| API         | :8080           |
| Frontend    | :3000           |
| Traefik     | :80 / :443      |
