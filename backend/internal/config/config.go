package config

import (
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"
)

// Config holds all runtime configuration for the orchestrator.
// Everything comes from env vars so the binary itself is immutable.
type Config struct {
	// HTTP API
	ListenAddr string

	// Database
	PostgresDSN string

	// Redis (queues, locks, idempotency)
	RedisAddr     string
	RedisPassword string
	RedisDB       int

	// GitHub App
	GitHubAppID         int64
	GitHubAppPrivateKey string
	GitHubWebhookSecret string

	// Buddy
	BuddyBaseURL   string
	BuddyToken     string
	BuddyWorkspace string

	// Docker
	DockerHost string // unix:///var/run/docker.sock

	// Traefik / Routing
	BaseDomain string

	// Reconciler tuning
	ReconcileInterval time.Duration
	EventPollInterval time.Duration
	DefaultEnvTTL     time.Duration
	DefaultLockTTL    time.Duration
	WorkerID          string

	// Fallback strategy for missing branches across repos.
	// One of: latest_stable | main | auto_create | previous_compatible
	DefaultFallback string

	// SecretKey is the AES-256 key (32 bytes, base64 or hex) used to encrypt
	// integration credentials at rest. Required if any integrations exist.
	SecretKey string

	// CORS origins allowed by the API (comma-separated). Empty = deny all cross-origin.
	CORSOrigins string
}

func Load() (*Config, error) {
	c := &Config{
		ListenAddr:          env("ORCH_LISTEN_ADDR", ":8080"),
		PostgresDSN:         env("ORCH_POSTGRES_DSN", "postgres://orch:orch@localhost:5432/orch?sslmode=disable"),
		RedisAddr:           env("ORCH_REDIS_ADDR", "localhost:6379"),
		RedisPassword:       env("ORCH_REDIS_PASSWORD", ""),
		GitHubWebhookSecret: env("ORCH_GITHUB_WEBHOOK_SECRET", ""),
		GitHubAppPrivateKey: env("ORCH_GITHUB_PRIVATE_KEY", ""),
		BuddyBaseURL:        env("ORCH_BUDDY_BASE_URL", "https://api.buddy.works"),
		BuddyToken:          env("ORCH_BUDDY_TOKEN", ""),
		BuddyWorkspace:      env("ORCH_BUDDY_WORKSPACE", ""),
		DockerHost:          env("ORCH_DOCKER_HOST", "unix:///var/run/docker.sock"),
		BaseDomain:          env("ORCH_BASE_DOMAIN", "localhost"),
		WorkerID:            env("ORCH_WORKER_ID", hostnameOr("worker-1")),
		DefaultFallback:     env("ORCH_DEFAULT_FALLBACK", "main"),
		SecretKey:           env("ORCH_SECRET_KEY", ""),
		CORSOrigins:         env("ORCH_CORS_ORIGINS", "http://localhost:3000"),
	}

	c.RedisDB = envInt("ORCH_REDIS_DB", 0)
	c.GitHubAppID = int64(envInt("ORCH_GITHUB_APP_ID", 0))
	c.ReconcileInterval = envDuration("ORCH_RECONCILE_INTERVAL", 15*time.Second)
	c.EventPollInterval = envDuration("ORCH_EVENT_POLL_INTERVAL", 2*time.Second)
	c.DefaultEnvTTL = envDuration("ORCH_DEFAULT_ENV_TTL", 72*time.Hour)
	c.DefaultLockTTL = envDuration("ORCH_LOCK_TTL", 60*time.Second)

	if !validFallback(c.DefaultFallback) {
		return nil, fmt.Errorf("invalid ORCH_DEFAULT_FALLBACK: %s", c.DefaultFallback)
	}
	return c, nil
}

func validFallback(s string) bool {
	switch s {
	case "latest_stable", "main", "auto_create", "previous_compatible":
		return true
	}
	return false
}

func env(k, def string) string {
	if v, ok := os.LookupEnv(k); ok && v != "" {
		return v
	}
	return def
}
func envInt(k string, def int) int {
	if v, ok := os.LookupEnv(k); ok {
		if n, err := strconv.Atoi(v); err == nil {
			return n
		}
	}
	return def
}
func envDuration(k string, def time.Duration) time.Duration {
	if v, ok := os.LookupEnv(k); ok {
		if d, err := time.ParseDuration(v); err == nil {
			return d
		}
	}
	return def
}
func hostnameOr(def string) string {
	if h, err := os.Hostname(); err == nil && strings.TrimSpace(h) != "" {
		return h
	}
	return def
}
