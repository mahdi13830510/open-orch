// Package events contains the worker that drains the persisted GitHub event
// queue and turns events into state changes (PR upserts, environment creation
// or destruction, generation bumps).
//
// Ingestion is decoupled from processing: the webhook handler simply persists
// events in PostgreSQL. This worker reads them in order using FOR UPDATE
// SKIP LOCKED, applies the changes, and marks them processed.
package events

import (
	"context"
	"crypto/sha1"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/open-orch/backend/internal/config"
	"github.com/open-orch/backend/internal/db"
	gh "github.com/open-orch/backend/internal/github"
	"github.com/open-orch/backend/internal/locks"
	"github.com/open-orch/backend/internal/models"
	"github.com/open-orch/backend/internal/normalizer"
	"github.com/open-orch/backend/internal/topology"
	"github.com/google/uuid"
	"github.com/rs/zerolog"
)

type Processor struct {
	Cfg        *config.Config
	Log        zerolog.Logger
	Events     *db.EventStore
	Repos      *db.RepositoryStore
	PRs        *db.PullRequestStore
	Features   *db.FeatureStore
	Envs       *db.EnvironmentStore
	Resolver   *topology.Resolver
	Locks      *locks.Manager
}

// Run drains events forever until ctx is cancelled.
func (p *Processor) Run(ctx context.Context) {
	t := time.NewTicker(p.Cfg.EventPollInterval)
	defer t.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-t.C:
			p.tick(ctx)
		}
	}
}

func (p *Processor) tick(ctx context.Context) {
	batch, err := p.Events.NextBatch(ctx, 25)
	if err != nil {
		p.Log.Error().Err(err).Msg("fetch next batch")
		return
	}
	for _, ev := range batch {
		err := p.handle(ctx, &ev)
		if err != nil {
			p.Log.Error().Err(err).Str("event_id", ev.ID.String()).Msg("process event")
			_ = p.Events.MarkError(ctx, ev.ID, err.Error())
			continue
		}
		_ = p.Events.MarkProcessed(ctx, ev.ID)
	}
}

func (p *Processor) handle(ctx context.Context, ev *models.Event) error {
	switch ev.EventType {
	case "pull_request":
		return p.handlePR(ctx, ev)
	case "push":
		return p.handlePush(ctx, ev)
	case "ping":
		return nil // ack
	default:
		// Unknown — still mark processed so we don't loop.
		return nil
	}
}

// ---------------------------------------------------------------------------
// pull_request handler
// ---------------------------------------------------------------------------

func (p *Processor) handlePR(ctx context.Context, ev *models.Event) error {
	payload, err := gh.DecodePR(ev.Payload)
	if err != nil {
		return fmt.Errorf("decode PR payload: %w", err)
	}
	if payload.Repository.FullName == "" {
		return nil
	}

	repo, err := p.Repos.GetByFullName(ctx, payload.Repository.FullName)
	if err != nil {
		// Repository isn't registered with the orchestrator → ignore.
		p.Log.Warn().Str("repo", payload.Repository.FullName).Msg("unknown repo, skipping")
		return nil
	}

	labels := make([]string, 0, len(payload.PullRequest.Labels))
	for _, l := range payload.PullRequest.Labels {
		labels = append(labels, l.Name)
	}
	state := payload.PullRequest.State
	if payload.PullRequest.Merged {
		state = "merged"
	}

	pr := &models.PullRequest{
		RepositoryID: repo.ID,
		Number:       payload.PullRequest.Number,
		Branch:       payload.PullRequest.Head.Ref,
		HeadSHA:      payload.PullRequest.Head.SHA,
		State:        state,
		Title:        payload.PullRequest.Title,
		Author:       payload.PullRequest.User.Login,
		Labels:       labels,
	}
	if err := p.PRs.Upsert(ctx, pr); err != nil {
		return fmt.Errorf("upsert PR: %w", err)
	}

	slug := normalizer.Branch(pr.Branch)
	if slug == "" {
		return nil
	}

	feat, err := p.Features.GetOrCreate(ctx, slug)
	if err != nil {
		return fmt.Errorf("feature get/create: %w", err)
	}

	// React on action.
	switch payload.Action {
	case "opened", "reopened", "synchronize", "labeled", "unlabeled":
		// Ensure the environment exists, then bump its generation and let the
		// reconciler converge runtime to the new desired topology.
		return p.ensureAndBump(ctx, repo, feat, pr)
	case "closed":
		// closed (merged or just closed) → mark env destroying.
		return p.markDestroying(ctx, feat)
	}
	return nil
}

// ---------------------------------------------------------------------------
// push handler
// ---------------------------------------------------------------------------

func (p *Processor) handlePush(ctx context.Context, ev *models.Event) error {
	payload, err := gh.DecodePush(ev.Payload)
	if err != nil {
		return err
	}
	if payload.Repository.FullName == "" {
		return nil
	}
	branch := strings.TrimPrefix(payload.Ref, "refs/heads/")
	if branch == "" {
		return nil
	}
	repo, err := p.Repos.GetByFullName(ctx, payload.Repository.FullName)
	if err != nil {
		return nil
	}
	slug := normalizer.Branch(branch)
	if slug == "" {
		return nil
	}
	// Only bump existing environments — push without a PR is informational.
	feat, err := p.Features.BySlug(ctx, slug)
	if err != nil {
		return nil
	}
	env, err := p.Envs.ByFeature(ctx, feat.ID)
	if err != nil {
		return nil
	}
	// Touch & bump generation; the resolver will pick up the new HEAD via PRs.
	_ = p.Envs.TouchEvent(ctx, env.ID)
	pr, _ := p.PRs.FindOpenByBranch(ctx, repo.ID, branch)
	_ = repo
	return p.bumpGeneration(ctx, env, feat, pr)
}

// ---------------------------------------------------------------------------
// helpers
// ---------------------------------------------------------------------------

func (p *Processor) ensureAndBump(ctx context.Context, originRepo *models.Repository, feat *models.Feature, pr *models.PullRequest) error {
	// Acquire a lock on this feature so concurrent events don't race.
	lockKey := fmt.Sprintf("feature:%s:plan_lock", feat.Slug)
	lk, err := p.Locks.AcquireWait(ctx, lockKey, p.Cfg.DefaultLockTTL, 5*time.Second)
	if err != nil {
		if errors.Is(err, locks.ErrLocked) {
			// Another worker will requeue; treat as transient.
			return fmt.Errorf("feature locked (retry): %s", feat.Slug)
		}
		return err
	}
	defer lk.Release(ctx)

	env, err := p.Envs.ByFeature(ctx, feat.ID)
	if err != nil {
		// Create a fresh environment.
		env = &models.Environment{
			ShortID:    "env_" + shortHash(feat.Slug),
			FeatureID:  feat.ID,
			State:      models.EnvPending,
			Generation: 0,
			BaseDomain: p.Cfg.BaseDomain,
		}
		env.DockerNetwork = env.ShortID
		ttl := int(p.Cfg.DefaultEnvTTL.Seconds())
		env.TTLSeconds = &ttl
		if err := p.Envs.Create(ctx, env); err != nil {
			return err
		}
	}
	_ = p.Envs.TouchEvent(ctx, env.ID)
	return p.bumpGeneration(ctx, env, feat, pr)
}

func (p *Processor) bumpGeneration(ctx context.Context, env *models.Environment, feat *models.Feature, pr *models.PullRequest) error {
	// Resolve a topology for the new generation.
	topo, err := p.Resolver.Resolve(ctx, env, feat, pr)
	if err != nil {
		_ = p.Envs.UpdateState(ctx, env.ID, models.EnvFailed)
		return fmt.Errorf("resolve topology: %w", err)
	}
	raw, _ := json.Marshal(topo)
	gen, err := p.Envs.BumpGeneration(ctx, env.ID, raw)
	if err != nil {
		return err
	}
	p.Log.Info().
		Str("env", env.ShortID).
		Int64("gen", gen).
		Str("feature", feat.Slug).
		Int("services", len(topo.Services)).
		Msg("topology resolved, generation bumped")
	// Move state to resolving; reconciler will pick up from here.
	_ = p.Envs.UpdateState(ctx, env.ID, models.EnvResolving)
	return nil
}

func (p *Processor) markDestroying(ctx context.Context, feat *models.Feature) error {
	env, err := p.Envs.ByFeature(ctx, feat.ID)
	if err != nil {
		return nil
	}
	// Only mark; the reconciler does the actual teardown.
	return p.Envs.UpdateState(ctx, env.ID, models.EnvDestroying)
}

// shortHash returns a short, stable id derived from input. 6 hex chars is
// enough entropy for env_xxxxxx given the small set of concurrent envs.
func shortHash(s string) string {
	h := sha1.Sum([]byte(s + ":" + uuid.NewString()[:8]))
	return hex.EncodeToString(h[:])[:6]
}
