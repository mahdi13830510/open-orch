package db

import (
	"context"
	"encoding/json"
	"time"

	"github.com/open-orch/backend/internal/models"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

// ----------------------------- EnvironmentStore -----------------------------

type EnvironmentStore struct{ DB *DB }

func (s *EnvironmentStore) Create(ctx context.Context, e *models.Environment) error {
	if len(e.DesiredTopology) == 0 {
		e.DesiredTopology = json.RawMessage(`{}`)
	}
	const q = `
INSERT INTO environments (short_id, feature_id, state, generation, desired_topology,
                          docker_network, base_domain, ttl_seconds, last_event_at)
VALUES ($1,$2,$3,$4,$5,$6,$7,$8,NOW())
RETURNING id, created_at, updated_at`
	return s.DB.Pool.QueryRow(ctx, q,
		e.ShortID, e.FeatureID, e.State, e.Generation, e.DesiredTopology,
		e.DockerNetwork, e.BaseDomain, e.TTLSeconds,
	).Scan(&e.ID, &e.CreatedAt, &e.UpdatedAt)
}

func (s *EnvironmentStore) Get(ctx context.Context, id uuid.UUID) (*models.Environment, error) {
	row := s.DB.Pool.QueryRow(ctx, envSel+` WHERE id=$1`, id)
	return scanEnv(row)
}

func (s *EnvironmentStore) ByShortID(ctx context.Context, sid string) (*models.Environment, error) {
	row := s.DB.Pool.QueryRow(ctx, envSel+` WHERE short_id=$1`, sid)
	return scanEnv(row)
}

func (s *EnvironmentStore) ByFeature(ctx context.Context, featureID uuid.UUID) (*models.Environment, error) {
	row := s.DB.Pool.QueryRow(ctx, envSel+
		` WHERE feature_id=$1 AND state NOT IN ('destroyed') ORDER BY created_at DESC LIMIT 1`, featureID)
	return scanEnv(row)
}

func (s *EnvironmentStore) List(ctx context.Context) ([]models.Environment, error) {
	rows, err := s.DB.Pool.Query(ctx, envSel+` ORDER BY created_at DESC LIMIT 500`)
	if err != nil { return nil, err }
	defer rows.Close()
	var out []models.Environment
	for rows.Next() {
		e, err := scanEnv(rows)
		if err != nil { return nil, err }
		out = append(out, *e)
	}
	return out, rows.Err()
}

// ActiveForReconcile returns envs that should be reconciled this tick.
func (s *EnvironmentStore) ActiveForReconcile(ctx context.Context) ([]models.Environment, error) {
	const q = envSel + `
WHERE state IN ('pending','resolving','deploying','healthy','degraded','failed','destroying')
ORDER BY COALESCE(last_reconciled_at, to_timestamp(0)) ASC
LIMIT 100`
	rows, err := s.DB.Pool.Query(ctx, q)
	if err != nil { return nil, err }
	defer rows.Close()
	var out []models.Environment
	for rows.Next() {
		e, err := scanEnv(rows)
		if err != nil { return nil, err }
		out = append(out, *e)
	}
	return out, rows.Err()
}

func (s *EnvironmentStore) UpdateState(ctx context.Context, id uuid.UUID, state models.EnvState) error {
	_, err := s.DB.Pool.Exec(ctx,
		`UPDATE environments SET state=$1, updated_at=NOW() WHERE id=$2`, state, id)
	return err
}

func (s *EnvironmentStore) TouchEvent(ctx context.Context, id uuid.UUID) error {
	_, err := s.DB.Pool.Exec(ctx, `UPDATE environments SET last_event_at=NOW() WHERE id=$1`, id)
	return err
}

func (s *EnvironmentStore) TouchReconciled(ctx context.Context, id uuid.UUID) error {
	_, err := s.DB.Pool.Exec(ctx, `UPDATE environments SET last_reconciled_at=NOW() WHERE id=$1`, id)
	return err
}

// SetSuspended flips the suspended flag. When true the reconciler skips this
// environment entirely (containers keep running in their current state) — useful
// for pausing flaky branches without tearing the env down.
func (s *EnvironmentStore) SetSuspended(ctx context.Context, id uuid.UUID, suspended bool) error {
	_, err := s.DB.Pool.Exec(ctx,
		`UPDATE environments SET suspended=$1, updated_at=NOW() WHERE id=$2`, suspended, id)
	return err
}

func (s *EnvironmentStore) BumpGeneration(ctx context.Context, id uuid.UUID, topology json.RawMessage) (int64, error) {
	if len(topology) == 0 {
		topology = json.RawMessage(`{}`)
	}
	var gen int64
	err := s.DB.Pool.QueryRow(ctx,
		`UPDATE environments SET generation=generation+1, desired_topology=$2, updated_at=NOW()
		 WHERE id=$1 RETURNING generation`, id, topology).Scan(&gen)
	return gen, err
}

func (s *EnvironmentStore) MarkDestroyed(ctx context.Context, id uuid.UUID) error {
	_, err := s.DB.Pool.Exec(ctx,
		`UPDATE environments SET state='destroyed', destroyed_at=NOW(), updated_at=NOW() WHERE id=$1`, id)
	return err
}

func (s *EnvironmentStore) IdleSince(ctx context.Context, cutoff time.Time) ([]models.Environment, error) {
	rows, err := s.DB.Pool.Query(ctx, envSel+
		` WHERE last_event_at < $1 AND state NOT IN ('destroying','destroyed')`, cutoff)
	if err != nil { return nil, err }
	defer rows.Close()
	var out []models.Environment
	for rows.Next() {
		e, err := scanEnv(rows)
		if err != nil { return nil, err }
		out = append(out, *e)
	}
	return out, rows.Err()
}

const envSel = `SELECT id, short_id, feature_id, state, generation, desired_topology,
		docker_network, base_domain, ttl_seconds, last_event_at, last_reconciled_at,
		suspended, created_at, updated_at, destroyed_at FROM environments`

func scanEnv(r rowScanner) (*models.Environment, error) {
	var e models.Environment
	var topo []byte
	err := r.Scan(&e.ID, &e.ShortID, &e.FeatureID, &e.State, &e.Generation, &topo,
		&e.DockerNetwork, &e.BaseDomain, &e.TTLSeconds, &e.LastEventAt, &e.LastReconciledAt,
		&e.Suspended, &e.CreatedAt, &e.UpdatedAt, &e.DestroyedAt)
	if err != nil {
		if err == pgx.ErrNoRows { return nil, ErrNotFound }
		return nil, err
	}
	e.DesiredTopology = topo
	return &e, nil
}

// ----------------------------- DeploymentStore ------------------------------

type DeploymentStore struct{ DB *DB }

func (s *DeploymentStore) Insert(ctx context.Context, d *models.Deployment) error {
	ev, _ := json.Marshal(d.EnvVars)
	const q = `
INSERT INTO deployments (environment_id, repository_id, generation, branch, commit_sha,
                         image_ref, state, container_name, container_id, env_vars,
                         selection_strategy, buddy_run_id)
VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12)
ON CONFLICT (environment_id, repository_id, generation) DO UPDATE SET
   branch=EXCLUDED.branch, commit_sha=EXCLUDED.commit_sha, image_ref=EXCLUDED.image_ref,
   state=EXCLUDED.state, container_name=EXCLUDED.container_name,
   container_id=EXCLUDED.container_id, env_vars=EXCLUDED.env_vars,
   selection_strategy=EXCLUDED.selection_strategy, buddy_run_id=EXCLUDED.buddy_run_id,
   last_status_at=NOW()
RETURNING id, created_at`
	return s.DB.Pool.QueryRow(ctx, q,
		d.EnvironmentID, d.RepositoryID, d.Generation, d.Branch, d.CommitSHA,
		d.ImageRef, d.State, d.ContainerName, d.ContainerID, ev,
		d.SelectionStrategy, d.BuddyRunID,
	).Scan(&d.ID, &d.CreatedAt)
}

func (s *DeploymentStore) ByEnvGeneration(ctx context.Context, envID uuid.UUID, gen int64) ([]models.Deployment, error) {
	const q = `SELECT id,environment_id,repository_id,generation,branch,commit_sha,
			image_ref,state,container_name,container_id,env_vars,selection_strategy,
			buddy_run_id,last_status_at,created_at
		FROM deployments WHERE environment_id=$1 AND generation=$2`
	rows, err := s.DB.Pool.Query(ctx, q, envID, gen)
	if err != nil { return nil, err }
	defer rows.Close()
	var out []models.Deployment
	for rows.Next() {
		var d models.Deployment
		var ev []byte
		if err := rows.Scan(&d.ID, &d.EnvironmentID, &d.RepositoryID, &d.Generation,
			&d.Branch, &d.CommitSHA, &d.ImageRef, &d.State, &d.ContainerName,
			&d.ContainerID, &ev, &d.SelectionStrategy, &d.BuddyRunID, &d.LastStatusAt, &d.CreatedAt); err != nil {
			return nil, err
		}
		_ = json.Unmarshal(ev, &d.EnvVars)
		out = append(out, d)
	}
	return out, rows.Err()
}

func (s *DeploymentStore) UpdateRuntime(ctx context.Context, id uuid.UUID, state models.DeploymentState, containerID string) error {
	_, err := s.DB.Pool.Exec(ctx,
		`UPDATE deployments SET state=$1, container_id=$2, last_status_at=NOW() WHERE id=$3`,
		state, containerID, id)
	return err
}

// ------------------------------- DomainStore -------------------------------

type DomainStore struct{ DB *DB }

func (s *DomainStore) Upsert(ctx context.Context, d *models.Domain) error {
	const q = `
INSERT INTO domains (environment_id, repository_id, hostname, target_port, tls)
VALUES ($1,$2,$3,$4,$5)
ON CONFLICT (hostname) DO UPDATE SET environment_id=EXCLUDED.environment_id,
   repository_id=EXCLUDED.repository_id, target_port=EXCLUDED.target_port, tls=EXCLUDED.tls
RETURNING id`
	return s.DB.Pool.QueryRow(ctx, q,
		d.EnvironmentID, d.RepositoryID, d.Hostname, d.TargetPort, d.TLS).Scan(&d.ID)
}

func (s *DomainStore) DeleteForEnv(ctx context.Context, envID uuid.UUID) error {
	_, err := s.DB.Pool.Exec(ctx, `DELETE FROM domains WHERE environment_id=$1`, envID)
	return err
}

func (s *DomainStore) ByEnv(ctx context.Context, envID uuid.UUID) ([]models.Domain, error) {
	rows, err := s.DB.Pool.Query(ctx,
		`SELECT id,environment_id,repository_id,hostname,target_port,tls
		 FROM domains WHERE environment_id=$1`, envID)
	if err != nil { return nil, err }
	defer rows.Close()
	var out []models.Domain
	for rows.Next() {
		var d models.Domain
		if err := rows.Scan(&d.ID, &d.EnvironmentID, &d.RepositoryID, &d.Hostname, &d.TargetPort, &d.TLS); err != nil {
			return nil, err
		}
		out = append(out, d)
	}
	return out, rows.Err()
}
