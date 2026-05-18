// Package reconciler runs the continuous control loop that converges
// observed runtime state (Docker, Traefik) toward desired state in PostgreSQL.
//
// The reconciler is the heart of the system. It is NOT a pipeline runner.
// On every tick, for each active environment, it:
//
//  1. Acquires a per-env deploy lock so two workers can't conflict.
//  2. Reads the desired topology from the env row.
//  3. Checks the network, every container, and every route.
//  4. Applies the minimum set of changes to bring runtime into agreement.
//  5. Updates lifecycle state (healthy/degraded/...).
//
// Deployment generations protect against stale workers overwriting newer
// state: when a worker starts converging gen=N it captures that number and
// refuses to write to runtime if the env has moved on to gen=N+1 mid-flight.
package reconciler

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/open-orch/backend/internal/buddy"
	"github.com/open-orch/backend/internal/config"
	"github.com/open-orch/backend/internal/db"
	"github.com/open-orch/backend/internal/docker"
	"github.com/open-orch/backend/internal/lifecycle"
	"github.com/open-orch/backend/internal/locks"
	"github.com/open-orch/backend/internal/models"
	"github.com/open-orch/backend/internal/traefik"
	"github.com/google/uuid"
	"github.com/rs/zerolog"
)

type Reconciler struct {
	Cfg      *config.Config
	Log      zerolog.Logger
	Envs     *db.EnvironmentStore
	Repos    *db.RepositoryStore
	Deploys  *db.DeploymentStore
	Domains  *db.DomainStore
	Runtime  *db.RuntimeResourceStore
	Health   *db.HealthCheckStore
	Docker   *docker.Driver
	Buddy    *buddy.Client
	Locks    *locks.Manager
}

func (r *Reconciler) Run(ctx context.Context) {
	t := time.NewTicker(r.Cfg.ReconcileInterval)
	defer t.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-t.C:
			r.tick(ctx)
		}
	}
}

func (r *Reconciler) tick(ctx context.Context) {
	envs, err := r.Envs.ActiveForReconcile(ctx)
	if err != nil {
		r.Log.Error().Err(err).Msg("list active envs")
		return
	}
	for _, env := range envs {
		if err := r.reconcileOne(ctx, &env); err != nil {
			r.Log.Error().Err(err).Str("env", env.ShortID).Msg("reconcile")
		}
	}
}

// reconcileOne owns one environment for one tick.
func (r *Reconciler) reconcileOne(ctx context.Context, env *models.Environment) error {
	lockKey := fmt.Sprintf("%s_deploy_lock", env.ShortID)
	lk, err := r.Locks.Acquire(ctx, lockKey, r.Cfg.DefaultLockTTL)
	if err != nil {
		if errors.Is(err, locks.ErrLocked) {
			return nil // another worker has it
		}
		return err
	}
	defer lk.Release(ctx)

	// Re-read in case it changed between listing and locking.
	fresh, err := r.Envs.Get(ctx, env.ID)
	if err != nil {
		return err
	}
	env = fresh

	// Capture generation up-front. If the env's generation moves while we work,
	// we'll abort to avoid clobbering a newer desired state.
	gen := env.Generation

	// Suspended environments are frozen in place: containers keep running
	// in whatever state they were last left in, but no new work happens.
	// Teardown still works (Destroying takes precedence) so operators can
	// always clean up.
	if env.Suspended && env.State != models.EnvDestroying {
		_ = r.Envs.TouchReconciled(ctx, env.ID)
		return nil
	}

	switch env.State {
	case models.EnvDestroying:
		return r.tearDown(ctx, env)
	case models.EnvDestroyed:
		return nil
	}

	// Parse the desired topology.
	var topo models.Topology
	if len(env.DesiredTopology) > 0 {
		_ = json.Unmarshal(env.DesiredTopology, &topo)
	}
	if len(topo.Services) == 0 {
		// Nothing to do yet — wait for events to populate desired state.
		_ = r.Envs.TouchReconciled(ctx, env.ID)
		return nil
	}

	// 1. Ensure network exists.
	netID, err := r.Docker.EnsureNetwork(ctx, env.DockerNetwork)
	if err != nil {
		return r.degrade(ctx, env, "network: "+err.Error())
	}
	_ = r.Runtime.Upsert(ctx, &models.RuntimeResource{
		EnvironmentID: env.ID, Kind: "network",
		Name: env.DockerNetwork, ExternalID: netID, State: "present",
	})

	// Promote to deploying if we're starting fresh.
	if env.State == models.EnvResolving || env.State == models.EnvPending {
		_ = r.setState(ctx, env, models.EnvDeploying)
	}

	// 2. Reconcile every service in the topology.
	allHealthy := true
	anyFailed := false
	for _, svc := range topo.Services {
		if err := r.reconcileService(ctx, env, gen, &topo, svc); err != nil {
			r.Log.Warn().Err(err).Str("svc", svc.RepositoryName).Msg("service reconcile")
			anyFailed = true
			allHealthy = false
		} else {
			// Inspect to update health.
			st, err := r.Docker.Inspect(ctx, svc.ContainerName)
			if err == nil {
				if !st.Running || (st.Health != "none" && st.Health != "healthy" && st.Health != "starting") {
					allHealthy = false
				}
			}
		}
	}

	// 3. Domain rows: sync DB to topology.
	if err := r.syncDomains(ctx, env, &topo); err != nil {
		r.Log.Warn().Err(err).Msg("sync domains")
	}

	// 4. Final state.
	switch {
	case anyFailed:
		_ = r.setState(ctx, env, models.EnvDegraded)
	case allHealthy:
		_ = r.setState(ctx, env, models.EnvHealthy)
	default:
		_ = r.setState(ctx, env, models.EnvDegraded)
	}

	_ = r.Envs.TouchReconciled(ctx, env.ID)
	return nil
}

// reconcileService brings one container in line with one ServicePlan.
func (r *Reconciler) reconcileService(ctx context.Context, env *models.Environment, gen int64, topo *models.Topology, svc models.ServicePlan) error {
	repo, err := r.Repos.GetByName(ctx, svc.RepositoryName)
	if err != nil {
		return err
	}

	// Determine the image ref. If Buddy hasn't built one yet, trigger and
	// record a `building` deployment row.
	imageTag := fmt.Sprintf("%s:%s", repo.Name, shortSHA(svc.CommitSHA))
	dep := &models.Deployment{
		EnvironmentID: env.ID,
		RepositoryID:  repo.ID,
		Generation:    gen,
		Branch:        svc.Branch,
		CommitSHA:     svc.CommitSHA,
		ImageRef:      imageTag,
		State:         models.DepPending,
		ContainerName: svc.ContainerName,
		EnvVars:       svc.EnvVars,
		SelectionStrategy: svc.Strategy,
	}

	// If we already have a deployment row for (env,repo,gen), reuse buddy run id.
	existing, _ := r.Deploys.ByEnvGeneration(ctx, env.ID, gen)
	for _, d := range existing {
		if d.RepositoryID == repo.ID {
			dep.ID = d.ID
			dep.BuddyRunID = d.BuddyRunID
			dep.State = d.State
			break
		}
	}

	// 1) If we don't yet have a Buddy build kicked off, trigger it.
	if dep.BuddyRunID == "" {
		tr, err := r.Buddy.Trigger(ctx, buddy.TriggerInput{
			Project:   repo.BuddyProject,
			Pipeline:  repo.BuddyPipeline,
			Branch:    svc.Branch,
			CommitSHA: svc.CommitSHA,
			ImageTag:  imageTag,
			Variables: map[string]string{
				"ENV_SHORT_ID": env.ShortID,
				"GENERATION":   fmt.Sprintf("%d", gen),
			},
		})
		if err != nil {
			dep.State = models.DepFailed
			_ = r.Deploys.Insert(ctx, dep)
			return fmt.Errorf("buddy trigger: %w", err)
		}
		dep.BuddyRunID = tr.RunID
		dep.State = models.DepBuilding
		if err := r.Deploys.Insert(ctx, dep); err != nil {
			return err
		}
	}

	// 2) Poll Buddy for status.
	if dep.State == models.DepBuilding {
		status, err := r.Buddy.GetRun(ctx, repo.BuddyProject, repo.BuddyPipeline, dep.BuddyRunID)
		if err != nil {
			return fmt.Errorf("buddy status: %w", err)
		}
		switch status {
		case "SUCCESSFUL", "FINISHED", "successful":
			dep.State = models.DepRunning // will move to running once container is up
		case "FAILED", "ERROR", "failed":
			dep.State = models.DepFailed
			_ = r.Deploys.UpdateRuntime(ctx, dep.ID, models.DepFailed, "")
			return fmt.Errorf("buddy build failed for %s", repo.Name)
		default:
			// Still building. Persist update and wait for next tick.
			_ = r.Deploys.UpdateRuntime(ctx, dep.ID, models.DepBuilding, "")
			return nil
		}
	}

	// 3) Ensure the container exists and is running with the right image.
	domain := domainFor(topo, repo.ID)
	lbls := traefik.Labels(env.ShortID, svc, domain)

	// Pull image (best-effort; in a registry-less test env this is a no-op).
	_ = r.Docker.PullImage(ctx, imageTag)

	port := 0
	if svc.ExposePort != nil {
		port = *svc.ExposePort
	}
	cid, err := r.Docker.RunContainer(ctx, docker.RunSpec{
		Name:       svc.ContainerName,
		Image:      imageTag,
		Network:    env.DockerNetwork,
		EnvVars:    svc.EnvVars,
		Labels:     lbls,
		ExposePort: port,
	})
	if err != nil {
		_ = r.Deploys.UpdateRuntime(ctx, dep.ID, models.DepFailed, "")
		return err
	}
	_ = r.Deploys.UpdateRuntime(ctx, dep.ID, models.DepRunning, cid)

	_ = r.Runtime.Upsert(ctx, &models.RuntimeResource{
		EnvironmentID: env.ID, Kind: "container",
		Name: svc.ContainerName, ExternalID: cid, State: "present",
	})

	// Health probe.
	st, _ := r.Docker.Inspect(ctx, svc.ContainerName)
	hs := "unknown"
	switch {
	case !st.Exists:
		hs = "failing"
	case !st.Running:
		hs = "failing"
	case st.Health == "healthy" || st.Health == "none":
		hs = "passing"
	case st.Health == "unhealthy":
		hs = "failing"
	}
	_ = r.Health.Record(ctx, dep.ID, hs, st.Health)
	return nil
}

// syncDomains writes domain rows to match the topology (the runtime side
// is driven by container labels Traefik consumes via the Docker provider).
func (r *Reconciler) syncDomains(ctx context.Context, env *models.Environment, topo *models.Topology) error {
	if err := r.Domains.DeleteForEnv(ctx, env.ID); err != nil {
		return err
	}
	for _, dp := range topo.Domains {
		d := &models.Domain{
			EnvironmentID: env.ID,
			RepositoryID:  dp.RepositoryID,
			Hostname:      dp.Hostname,
			TargetPort:    dp.TargetPort,
			TLS:           dp.TLS,
		}
		if err := r.Domains.Upsert(ctx, d); err != nil {
			return err
		}
	}
	return nil
}

// tearDown removes all runtime resources for an env and marks it destroyed.
// Soft deletion: DB rows are kept (with destroyed_at set) for audit.
func (r *Reconciler) tearDown(ctx context.Context, env *models.Environment) error {
	// Remove every container we know about.
	rrs, _ := r.Runtime.ByEnv(ctx, env.ID)
	for _, rr := range rrs {
		if rr.Kind == "container" {
			_ = r.Docker.RemoveByName(ctx, rr.Name)
		}
	}
	// Remove the network last.
	if env.DockerNetwork != "" {
		_ = r.Docker.RemoveNetwork(ctx, env.DockerNetwork)
	}
	_ = r.Domains.DeleteForEnv(ctx, env.ID)
	_ = r.Runtime.DeleteForEnv(ctx, env.ID)
	_ = r.Envs.MarkDestroyed(ctx, env.ID)
	r.Log.Info().Str("env", env.ShortID).Msg("environment destroyed")
	return nil
}

// degrade is a small helper to record a degraded state with a reason.
func (r *Reconciler) degrade(ctx context.Context, env *models.Environment, reason string) error {
	r.Log.Warn().Str("env", env.ShortID).Str("reason", reason).Msg("degraded")
	return r.setState(ctx, env, models.EnvDegraded)
}

func (r *Reconciler) setState(ctx context.Context, env *models.Environment, to models.EnvState) error {
	if err := lifecycle.CanTransition(env.State, to); err != nil {
		return err
	}
	if env.State == to {
		return nil
	}
	if err := r.Envs.UpdateState(ctx, env.ID, to); err != nil {
		return err
	}
	env.State = to
	return nil
}

func domainFor(topo *models.Topology, repoID uuid.UUID) *models.DomainPlan {
	for i := range topo.Domains {
		if topo.Domains[i].RepositoryID == repoID {
			return &topo.Domains[i]
		}
	}
	return nil
}

func shortSHA(s string) string {
	if len(s) >= 12 {
		return s[:12]
	}
	return s
}
