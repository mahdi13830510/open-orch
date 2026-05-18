// Command orchestrator is the control-plane process for the ephemeral
// preview environment platform.
//
// It runs four concurrent components:
//
//   1. an HTTP server that exposes the REST API and the GitHub webhook
//   2. an event processor that drains the persisted webhook queue
//   3. a reconciler that continuously converges Docker/Traefik with the
//      desired state stored in PostgreSQL
//   4. a TTL cleaner that marks idle environments for destruction
//
// Each component can be scaled independently in production (run more
// reconciler workers, more event processors, etc.) — they coordinate via
// PostgreSQL row locks and Redis leases, not via in-process state.
package main

import (
	"context"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/open-orch/backend/internal/api"
	"github.com/open-orch/backend/internal/buddy"
	"github.com/open-orch/backend/internal/config"
	"github.com/open-orch/backend/internal/db"
	"github.com/open-orch/backend/internal/docker"
	"github.com/open-orch/backend/internal/events"
	gh "github.com/open-orch/backend/internal/github"
	"github.com/open-orch/backend/internal/locks"
	"github.com/open-orch/backend/internal/reconciler"
	"github.com/open-orch/backend/internal/topology"

	"github.com/redis/go-redis/v9"
	"github.com/rs/zerolog"
)

func main() {
	log := zerolog.New(os.Stdout).With().Timestamp().Logger()

	cfg, err := config.Load()
	if err != nil {
		log.Fatal().Err(err).Msg("config")
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// PostgreSQL.
	database, err := db.Open(ctx, cfg.PostgresDSN)
	if err != nil { log.Fatal().Err(err).Msg("postgres") }
	defer database.Close()

	// Redis.
	rdb := redis.NewClient(&redis.Options{
		Addr:     cfg.RedisAddr,
		Password: cfg.RedisPassword,
		DB:       cfg.RedisDB,
	})
	if err := rdb.Ping(ctx).Err(); err != nil {
		log.Fatal().Err(err).Msg("redis")
	}
	defer rdb.Close()

	// Docker.
	dock, err := docker.New(cfg.DockerHost)
	if err != nil { log.Fatal().Err(err).Msg("docker") }

	// Buddy client.
	bud := buddy.New(cfg.BuddyBaseURL, cfg.BuddyWorkspace, cfg.BuddyToken)

	// Stores.
	repos := &db.RepositoryStore{DB: database}
	prs := &db.PullRequestStore{DB: database}
	features := &db.FeatureStore{DB: database}
	envs := &db.EnvironmentStore{DB: database}
	deps := &db.DeploymentStore{DB: database}
	domains := &db.DomainStore{DB: database}
	runtime := &db.RuntimeResourceStore{DB: database}
	health := &db.HealthCheckStore{DB: database}
	eventStore := &db.EventStore{DB: database}

	integrations, err := db.NewIntegrationStore(database, cfg.SecretKey)
	if err != nil {
		log.Fatal().Err(err).Msg("integration store init")
	}
	if cfg.SecretKey == "" {
		log.Warn().Msg("ORCH_SECRET_KEY not set — integrations with secrets cannot be created")
	}

	lockMgr := locks.New(rdb)
	resolver := &topology.Resolver{
		Cfg: cfg, Repos: repos, PRs: prs, Features: features,
	}

	// Event processor.
	proc := &events.Processor{
		Cfg: cfg, Log: log.With().Str("comp", "events").Logger(),
		Events: eventStore, Repos: repos, PRs: prs,
		Features: features, Envs: envs, Resolver: resolver, Locks: lockMgr,
	}

	// Reconciler.
	rec := &reconciler.Reconciler{
		Cfg: cfg, Log: log.With().Str("comp", "reconciler").Logger(),
		Envs: envs, Repos: repos, Deploys: deps, Domains: domains,
		Runtime: runtime, Health: health,
		Docker: dock, Buddy: bud, Locks: lockMgr,
	}
	cleaner := &reconciler.Cleaner{
		Cfg: cfg, Log: log.With().Str("comp", "cleaner").Logger(), Envs: envs,
	}

	// GitHub webhook handler.
	webhook := &gh.Handler{
		Secret: cfg.GitHubWebhookSecret,
		Events: eventStore,
		Log:    log.With().Str("comp", "webhook").Logger(),
	}

	// API server.
	srv := &api.Server{
		Cfg: cfg, Log: log.With().Str("comp", "api").Logger(),
		Repos: repos, PRs: prs, Features: features,
		Envs: envs, Deploys: deps, Domains: domains, Runtime: runtime,
		Events: eventStore, Integrations: integrations,
		Resolver: resolver, Locks: lockMgr,
	}

	httpSrv := &http.Server{
		Addr:              cfg.ListenAddr,
		Handler:           srv.Routes(webhook),
		ReadHeaderTimeout: 10 * time.Second,
	}

	// Run components.
	go proc.Run(ctx)
	go rec.Run(ctx)
	go cleaner.Run(ctx)
	go func() {
		log.Info().Str("addr", cfg.ListenAddr).Msg("api listening")
		if err := httpSrv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Error().Err(err).Msg("http server")
		}
	}()

	// Shutdown.
	sig := make(chan os.Signal, 1)
	signal.Notify(sig, os.Interrupt, syscall.SIGTERM)
	<-sig
	log.Info().Msg("shutdown initiated")
	shCtx, shCancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer shCancel()
	_ = httpSrv.Shutdown(shCtx)
	cancel()
	log.Info().Msg("shutdown complete")
}
