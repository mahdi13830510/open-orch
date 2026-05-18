# Contributing to Open-Orch

Thanks for contributing! This guide covers the development workflow for both the backend (Go) and frontend (React/TypeScript).

## Repository structure

```
backend/    Go control-plane API
frontend/   React dashboard
docs/       Docusaurus documentation site
```

## Prerequisites

- Go 1.22+
- Node.js 20+
- Docker + Docker Compose
- `gh` CLI (optional, for releases)

## Development setup

### Backend

```bash
cd backend

# Start dependencies
docker compose -f deploy/docker-compose.yml up postgres redis -d

# Copy and edit env
cp deploy/.env.example deploy/.env

# Run locally
go run ./cmd/orchestrator
```

### Frontend

```bash
cd frontend
cp .env.example .env
# Set VITE_API_URL=http://localhost:8080
npm install
npm run dev
```

### Docs

```bash
cd docs
npm install
npm run dev
```

## Making changes

1. Fork the repo and create a feature branch from `main`.
2. Follow the existing code style — `gofmt` for Go, Prettier for TypeScript.
3. Write tests for new backend behavior (especially DB stores and API handlers).
4. Update relevant docs in `docs/docs/` if behavior changes.
5. Open a pull request against `main`.

## Commit style

We use [Conventional Commits](https://www.conventionalcommits.org/):

```
feat(backend): add environment pause endpoint
fix(frontend): correct dep lookup to use repo name
docs: add data-flow diagram
chore: update dependencies
```

Prefixes: `feat` / `fix` / `docs` / `chore` / `refactor` / `test` / `ci`

## API contract

The backend API is the source of truth. If you change an endpoint:
1. Update `backend/internal/api/api.go`
2. Update `frontend/src/types.ts` and `frontend/src/services/api.ts`
3. Update `docs/docs/api/reference.md`

## Pull request checklist

- [ ] `go build ./...` passes
- [ ] `tsc --noEmit` passes
- [ ] No new `askiza` references introduced
- [ ] Docs updated if behavior changed
- [ ] Conventional commit messages

## Release process

Releases follow [Semantic Versioning](https://semver.org/). Maintainers tag releases via:

```bash
gh release create vX.Y.Z --generate-notes
```
