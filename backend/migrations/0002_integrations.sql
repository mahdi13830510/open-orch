-- ---------------------------------------------------------------------------
-- 0002_integrations.sql
--
-- Adds a global, encrypted-at-rest store for third-party credentials
-- (GitHub Apps, Buddy.works tokens, Cloudflare DNS API tokens, container
-- registries, etc.). Distinct from `secrets`, which is *per-environment*.
--
-- Values are encrypted by the application using AES-256-GCM with the key
-- derived from ORCH_SECRET_KEY. The DB never sees the plaintext.
-- ---------------------------------------------------------------------------
CREATE TABLE IF NOT EXISTS integrations (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    kind            TEXT NOT NULL,             -- 'github_app' | 'buddy' | 'cloudflare' | 'registry' | 'webhook' | 'custom'
    name            TEXT NOT NULL,             -- human handle, unique within kind
    config          JSONB NOT NULL DEFAULT '{}',-- non-secret metadata (workspace slug, app id, base URL, etc.)
    secret_enc      BYTEA,                     -- nullable: some integrations have no secret part
    secret_nonce    BYTEA,                     -- 12-byte GCM nonce; NULL iff secret_enc IS NULL
    last_verified_at TIMESTAMPTZ,              -- set by a successful verification call
    last_error      TEXT,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE (kind, name)
);
CREATE INDEX IF NOT EXISTS idx_integrations_kind ON integrations(kind);
