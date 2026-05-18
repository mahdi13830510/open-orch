# Open-Orch UI

React dashboard for the Open-Orch control plane. Manages environments, repositories, integrations, and audit logs.

## Prerequisites

- Node.js 20+
- Open-Orch backend running (see `../backend`)

## Run locally

```bash
cp .env.example .env
# set VITE_API_URL=http://localhost:8080
npm install
npm run dev
# opens on http://localhost:3000
```

## Build

```bash
npm run build
# output in dist/
```

## Environment variables

| Variable       | Default                 | Description              |
|----------------|-------------------------|--------------------------|
| `VITE_API_URL` | `http://localhost:8080` | Open-Orch backend API URL|
