-- =============================================================================
-- Orchestrator schema. PostgreSQL is the source of truth for desired state.
-- Runtime (Docker / Traefik) is reconciled toward this state continuously.
-- =============================================================================

CREATE EXTENSION IF NOT EXISTS "pgcrypto";

-- ---------------------------------------------------------------------------
-- repositories: services we know about, registered ahead of time.
-- ---------------------------------------------------------------------------
CREATE TABLE IF NOT EXISTS repositories (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name            TEXT NOT NULL UNIQUE,           -- "frontend", "backend"
    full_name       TEXT NOT NULL UNIQUE,           -- "askiza/frontend"
    default_branch  TEXT NOT NULL DEFAULT 'main',
    service_kind    TEXT NOT NULL,                  -- "http", "worker", "gateway"
    expose_port     INT,                            -- nullable for workers
    buddy_project   TEXT NOT NULL,                  -- buddy project slug
    buddy_pipeline  TEXT NOT NULL,                  -- buddy pipeline id/slug
    metadata        JSONB NOT NULL DEFAULT '{}',
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- ---------------------------------------------------------------------------
-- service_dependencies: directed edges of the dependency graph.
-- frontend --> backend, backend --> gateway, etc.
-- ---------------------------------------------------------------------------
CREATE TABLE IF NOT EXISTS service_dependencies (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    repository_id   UUID NOT NULL REFERENCES repositories(id) ON DELETE CASCADE,
    depends_on_id   UUID NOT NULL REFERENCES repositories(id) ON DELETE CASCADE,
    required        BOOLEAN NOT NULL DEFAULT TRUE,
    UNIQUE (repository_id, depends_on_id)
);

-- ---------------------------------------------------------------------------
-- pull_requests: replicated PR state per repo.
-- ---------------------------------------------------------------------------
CREATE TABLE IF NOT EXISTS pull_requests (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    repository_id   UUID NOT NULL REFERENCES repositories(id) ON DELETE CASCADE,
    number          INT  NOT NULL,
    branch          TEXT NOT NULL,
    head_sha        TEXT NOT NULL,
    state           TEXT NOT NULL,                  -- open, closed, merged
    title           TEXT,
    author          TEXT,
    labels          JSONB NOT NULL DEFAULT '[]',
    opened_at       TIMESTAMPTZ,
    closed_at       TIMESTAMPTZ,
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE (repository_id, number)
);
CREATE INDEX IF NOT EXISTS idx_pr_branch ON pull_requests(branch);

-- ---------------------------------------------------------------------------
-- features: a "feature identity" derived from a normalized branch name.
-- This is what binds multi-repo branches together into one environment.
-- ---------------------------------------------------------------------------
CREATE TABLE IF NOT EXISTS features (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    slug            TEXT NOT NULL UNIQUE,           -- "billing-redesign"
    display_name    TEXT,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    last_seen_at    TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- ---------------------------------------------------------------------------
-- environments: desired state for one preview environment (one feature).
-- ---------------------------------------------------------------------------
CREATE TABLE IF NOT EXISTS environments (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    short_id        TEXT NOT NULL UNIQUE,           -- "env_f83a12"
    feature_id      UUID NOT NULL REFERENCES features(id) ON DELETE CASCADE,
    state           TEXT NOT NULL,                  -- pending|resolving|deploying|healthy|degraded|failed|destroying|destroyed
    generation      BIGINT NOT NULL DEFAULT 0,      -- monotonically increasing
    desired_topology JSONB NOT NULL DEFAULT '{}',   -- resolved topology snapshot
    docker_network  TEXT,                           -- "env_f83a12"
    base_domain     TEXT NOT NULL,                  -- "askiza.com"
    ttl_seconds     INT,                            -- nullable: no TTL
    last_event_at   TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    last_reconciled_at TIMESTAMPTZ,
    suspended       BOOLEAN NOT NULL DEFAULT FALSE,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    destroyed_at    TIMESTAMPTZ
);
CREATE INDEX IF NOT EXISTS idx_env_feature ON environments(feature_id);
CREATE INDEX IF NOT EXISTS idx_env_state ON environments(state);

-- ---------------------------------------------------------------------------
-- deployments: one row per (environment, service) per generation.
-- Tracks the desired+observed deployment of one service inside one env.
-- ---------------------------------------------------------------------------
CREATE TABLE IF NOT EXISTS deployments (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    environment_id  UUID NOT NULL REFERENCES environments(id) ON DELETE CASCADE,
    repository_id   UUID NOT NULL REFERENCES repositories(id) ON DELETE CASCADE,
    generation      BIGINT NOT NULL,
    branch          TEXT NOT NULL,
    commit_sha      TEXT NOT NULL,
    image_ref       TEXT,                           -- "registry/svc:sha-xxx"
    state           TEXT NOT NULL,                  -- pending|building|running|failed|stopped
    container_name  TEXT,                           -- "frontend-env_f83a12"
    container_id    TEXT,                           -- docker container id
    env_vars        JSONB NOT NULL DEFAULT '{}',
    selection_strategy TEXT,                        -- matched|fallback_main|fallback_latest|...
    buddy_run_id    TEXT,
    last_status_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE (environment_id, repository_id, generation)
);
CREATE INDEX IF NOT EXISTS idx_dep_env_gen ON deployments(environment_id, generation);

-- ---------------------------------------------------------------------------
-- domains: routing entries managed by Traefik (one per exposed service).
-- ---------------------------------------------------------------------------
CREATE TABLE IF NOT EXISTS domains (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    environment_id  UUID NOT NULL REFERENCES environments(id) ON DELETE CASCADE,
    repository_id   UUID NOT NULL REFERENCES repositories(id) ON DELETE CASCADE,
    hostname        TEXT NOT NULL UNIQUE,           -- billing-redesign-pr120.askiza.com
    target_port     INT NOT NULL,
    tls             BOOLEAN NOT NULL DEFAULT TRUE,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- ---------------------------------------------------------------------------
-- runtime_resources: opaque tracking for created infra (networks, volumes...).
-- Reconciler uses this to know what was claimed so it can verify/repair.
-- ---------------------------------------------------------------------------
CREATE TABLE IF NOT EXISTS runtime_resources (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    environment_id  UUID NOT NULL REFERENCES environments(id) ON DELETE CASCADE,
    kind            TEXT NOT NULL,                  -- network|container|volume|route
    external_id     TEXT,                           -- docker id / traefik router
    name            TEXT NOT NULL,
    metadata        JSONB NOT NULL DEFAULT '{}',
    state           TEXT NOT NULL DEFAULT 'desired',-- desired|present|missing|stale
    last_seen_at    TIMESTAMPTZ,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE INDEX IF NOT EXISTS idx_rr_env ON runtime_resources(environment_id);

-- ---------------------------------------------------------------------------
-- events: every webhook delivered by GitHub is persisted *before* processing.
-- Decouples ingestion from processing.
-- ---------------------------------------------------------------------------
CREATE TABLE IF NOT EXISTS events (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    source          TEXT NOT NULL,                  -- "github"
    delivery_id     TEXT UNIQUE,                    -- X-GitHub-Delivery
    event_type      TEXT NOT NULL,                  -- "pull_request" / "push"
    action          TEXT,                           -- "opened" / "synchronize" / ...
    repository      TEXT,
    payload         JSONB NOT NULL,
    received_at     TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    processed_at    TIMESTAMPTZ,
    process_error   TEXT,
    attempt_count   INT NOT NULL DEFAULT 0
);
CREATE INDEX IF NOT EXISTS idx_events_unprocessed
    ON events(received_at) WHERE processed_at IS NULL;

-- ---------------------------------------------------------------------------
-- leases: distributed coordination markers persisted for visibility.
-- Real lock is held in Redis; this is a human-readable mirror.
-- ---------------------------------------------------------------------------
CREATE TABLE IF NOT EXISTS leases (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    key             TEXT NOT NULL UNIQUE,           -- env_f83a12_deploy_lock
    holder          TEXT NOT NULL,                  -- worker id
    acquired_at     TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    expires_at      TIMESTAMPTZ NOT NULL
);

-- ---------------------------------------------------------------------------
-- secrets: per-environment isolated secret bag (encrypted at rest in prod).
-- ---------------------------------------------------------------------------
CREATE TABLE IF NOT EXISTS secrets (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    environment_id  UUID NOT NULL REFERENCES environments(id) ON DELETE CASCADE,
    name            TEXT NOT NULL,
    value_enc       BYTEA NOT NULL,                 -- ciphertext
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    revoked_at      TIMESTAMPTZ,
    UNIQUE (environment_id, name)
);

-- ---------------------------------------------------------------------------
-- health_checks: rolling observation of container/service health.
-- ---------------------------------------------------------------------------
CREATE TABLE IF NOT EXISTS health_checks (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    deployment_id   UUID NOT NULL REFERENCES deployments(id) ON DELETE CASCADE,
    status          TEXT NOT NULL,                  -- passing|failing|unknown
    detail          TEXT,
    checked_at      TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE INDEX IF NOT EXISTS idx_hc_dep ON health_checks(deployment_id, checked_at DESC);
