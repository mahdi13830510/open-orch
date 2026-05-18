---
sidebar_position: 1
---

# API Reference

Base URL: `http://localhost:8080` (or `https://api.your-domain.com` in production)

All endpoints accept and return `application/json`. Errors always return `{ "error": "message" }`.

---

## Health

### `GET /healthz`

Liveness probe.

**Response `200`**
```json
{ "status": "ok" }
```

---

## Repositories

### `GET /repositories`

List all registered repositories.

**Response `200`** — array of Repository objects.

### `POST /repositories`

Register or update a repository (upserted on `full_name`).

**Body**
```json
{
  "name": "api-gateway",
  "full_name": "org/api-gateway",
  "default_branch": "main",
  "service_kind": "http",
  "expose_port": 8080,
  "buddy_project": "my-project",
  "buddy_pipeline": "build-and-push"
}
```

`service_kind`: `http` | `worker` | `static`

### `GET /repositories/{name}/dependencies`

List direct dependencies of a repository by name.

**Response `200`**
```json
[
  {
    "repository_id": "uuid",
    "depends_on_id": "uuid",
    "depends_on_name": "auth-service",
    "depends_on_full_name": "org/auth-service",
    "required": true
  }
]
```

### `POST /repositories/{name}/dependencies`

Add a dependency edge.

**Body**
```json
{ "depends_on": "auth-service", "required": true }
```

### `DELETE /repositories/{name}/dependencies/{depName}`

Remove a dependency edge.

---

## Environments

### `GET /environments`

List all non-destroyed environments.

### `POST /environments`

Create (or return existing) environment for a feature.

**Body**
```json
{
  "feature": "feat/billing-redesign",
  "ttl": "48h"
}
```

`feature` is normalized to a DNS-safe slug. If an environment already exists for that feature, it is returned.

**Response `201`** — Environment object.

### `GET /environments/{id}`

Get full environment details. `id` can be UUID or `short_id`.

**Response `200`**
```json
{
  "environment": { ... },
  "deployments": [ ... ],
  "domains": [ ... ],
  "runtime": [ ... ]
}
```

### `DELETE /environments/{id}`

Mark environment for destruction. Async — reconciler performs teardown.

**Response `202`** `{ "status": "destroying" }`

### `POST /environments/{id}/suspend`

Freeze the environment. Containers keep running but the reconciler stops acting on it.

**Response `200`** `{ "status": "suspended", "id": "...", "short_id": "..." }`

### `POST /environments/{id}/resume`

Resume a suspended environment. Nudges the reconciler.

**Response `200`** `{ "status": "resumed", "id": "...", "short_id": "..." }`

---

## Deployments

### `POST /deployments/reconcile`

Manually trigger a reconcile for an environment.

**Body** `{ "environment_id": "uuid-or-short-id" }`  
**Response `202`** `{ "status": "queued" }`

### `POST /deployments/restart`

Bump the topology generation and re-deploy all services.

**Body** `{ "environment_id": "uuid-or-short-id" }`  
**Response `202`** `{ "status": "restarting" }`

---

## Features

### `GET /features/{slug}`

Look up a feature and its associated environment.

**Response `200`**
```json
{
  "feature": { "id": "...", "slug": "feat-billing", "created_at": "..." },
  "environment": { ... }
}
```

---

## Integrations

Secrets are stored AES-256-GCM encrypted. The API never returns the raw secret — only `has_secret: true/false`.

Valid `kind` values: `github_app` | `buddy` | `cloudflare` | `registry` | `webhook` | `custom`

### `GET /integrations?kind=`

List integrations, optionally filtered by kind.

### `POST /integrations`

Create or update an integration (upserted on `kind` + `name`).

**Body**
```json
{
  "kind": "buddy",
  "name": "primary",
  "config": {},
  "secret": "my-token"
}
```

Omit `secret` to leave the existing secret unchanged.

### `GET /integrations/{id}`

Get a single integration.

### `PATCH /integrations/{id}`

Partial update. `kind` and `name` are immutable after creation.

### `DELETE /integrations/{id}`

Delete an integration.

### `POST /integrations/{id}/verify`

Record the result of a client-driven verification check.

**Body** `{ "ok": true, "detail": "optional message" }`

---

## Events

### `GET /events`

Return the last 100 processed webhook events (descending by `received_at`).

---

## Webhook

### `POST /webhooks/github`

GitHub webhook endpoint. Requires a valid `X-Hub-Signature-256` header matching `ORCH_GITHUB_WEBHOOK_SECRET`.

Supported event types: `push`, `pull_request`, `create`, `delete`.
