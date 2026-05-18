package db

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/open-orch/backend/internal/models"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

// RepositoryStore handles registered services.
type RepositoryStore struct{ DB *DB }

func (s *RepositoryStore) GetByFullName(ctx context.Context, full string) (*models.Repository, error) {
	const q = `SELECT id,name,full_name,default_branch,service_kind,expose_port,
		buddy_project,buddy_pipeline,metadata,created_at,updated_at
		FROM repositories WHERE full_name=$1`
	row := s.DB.Pool.QueryRow(ctx, q, full)
	return scanRepo(row)
}

func (s *RepositoryStore) GetByName(ctx context.Context, name string) (*models.Repository, error) {
	const q = `SELECT id,name,full_name,default_branch,service_kind,expose_port,
		buddy_project,buddy_pipeline,metadata,created_at,updated_at
		FROM repositories WHERE name=$1`
	row := s.DB.Pool.QueryRow(ctx, q, name)
	return scanRepo(row)
}

func (s *RepositoryStore) List(ctx context.Context) ([]models.Repository, error) {
	const q = `SELECT id,name,full_name,default_branch,service_kind,expose_port,
		buddy_project,buddy_pipeline,metadata,created_at,updated_at
		FROM repositories ORDER BY name`
	rows, err := s.DB.Pool.Query(ctx, q)
	if err != nil { return nil, err }
	defer rows.Close()
	var out []models.Repository
	for rows.Next() {
		r, err := scanRepo(rows)
		if err != nil { return nil, err }
		out = append(out, *r)
	}
	return out, rows.Err()
}

func (s *RepositoryStore) Upsert(ctx context.Context, r *models.Repository) error {
	meta := r.Metadata
	if len(meta) == 0 {
		meta = json.RawMessage(`{}`)
	}
	const q = `
INSERT INTO repositories (name,full_name,default_branch,service_kind,expose_port,buddy_project,buddy_pipeline,metadata)
VALUES ($1,$2,$3,$4,$5,$6,$7,$8)
ON CONFLICT (full_name) DO UPDATE SET
  name=EXCLUDED.name, default_branch=EXCLUDED.default_branch,
  service_kind=EXCLUDED.service_kind, expose_port=EXCLUDED.expose_port,
  buddy_project=EXCLUDED.buddy_project, buddy_pipeline=EXCLUDED.buddy_pipeline,
  metadata=EXCLUDED.metadata, updated_at=NOW()
RETURNING id, created_at, updated_at`
	return s.DB.Pool.QueryRow(ctx, q,
		r.Name, r.FullName, r.DefaultBranch, r.ServiceKind, r.ExposePort,
		r.BuddyProject, r.BuddyPipeline, meta,
	).Scan(&r.ID, &r.CreatedAt, &r.UpdatedAt)
}

// AddDependency: repo depends_on depends.
func (s *RepositoryStore) AddDependency(ctx context.Context, repo, depends uuid.UUID, required bool) error {
	const q = `
INSERT INTO service_dependencies (repository_id, depends_on_id, required)
VALUES ($1,$2,$3) ON CONFLICT DO NOTHING`
	_, err := s.DB.Pool.Exec(ctx, q, repo, depends, required)
	return err
}

// RepositoryDependency is one resolved edge in the dep graph, returned by
// ListDependencies. We hydrate the dependent repo's name/full_name so clients
// don't have to round-trip per row.
type RepositoryDependency struct {
	RepositoryID    uuid.UUID `json:"repository_id"`
	DependsOnID     uuid.UUID `json:"depends_on_id"`
	DependsOnName   string    `json:"depends_on_name"`
	DependsOnFull   string    `json:"depends_on_full_name"`
	Required        bool      `json:"required"`
}

// ListDependencies returns all direct dependencies of `repo`.
func (s *RepositoryStore) ListDependencies(ctx context.Context, repo uuid.UUID) ([]RepositoryDependency, error) {
	const q = `
SELECT sd.repository_id, sd.depends_on_id, r.name, r.full_name, sd.required
FROM service_dependencies sd
JOIN repositories r ON r.id = sd.depends_on_id
WHERE sd.repository_id = $1
ORDER BY r.name`
	rows, err := s.DB.Pool.Query(ctx, q, repo)
	if err != nil { return nil, err }
	defer rows.Close()
	var out []RepositoryDependency
	for rows.Next() {
		var d RepositoryDependency
		if err := rows.Scan(&d.RepositoryID, &d.DependsOnID, &d.DependsOnName, &d.DependsOnFull, &d.Required); err != nil {
			return nil, err
		}
		out = append(out, d)
	}
	return out, rows.Err()
}

// RemoveDependency deletes the edge repo -> depends. Idempotent: returns nil
// (with deleted=false) if no row matched.
func (s *RepositoryStore) RemoveDependency(ctx context.Context, repo, depends uuid.UUID) (bool, error) {
	const q = `DELETE FROM service_dependencies WHERE repository_id=$1 AND depends_on_id=$2`
	ct, err := s.DB.Pool.Exec(ctx, q, repo, depends)
	if err != nil { return false, err }
	return ct.RowsAffected() > 0, nil
}

// Dependencies returns the full dependency graph as adjacency lists.
func (s *RepositoryStore) Dependencies(ctx context.Context) (map[uuid.UUID][]uuid.UUID, error) {
	const q = `SELECT repository_id, depends_on_id FROM service_dependencies`
	rows, err := s.DB.Pool.Query(ctx, q)
	if err != nil { return nil, err }
	defer rows.Close()
	out := map[uuid.UUID][]uuid.UUID{}
	for rows.Next() {
		var a, b uuid.UUID
		if err := rows.Scan(&a, &b); err != nil { return nil, err }
		out[a] = append(out[a], b)
	}
	return out, rows.Err()
}

type rowScanner interface{ Scan(...any) error }

func scanRepo(r rowScanner) (*models.Repository, error) {
	var x models.Repository
	var meta []byte
	err := r.Scan(&x.ID, &x.Name, &x.FullName, &x.DefaultBranch, &x.ServiceKind,
		&x.ExposePort, &x.BuddyProject, &x.BuddyPipeline, &meta, &x.CreatedAt, &x.UpdatedAt)
	if err != nil {
		if err == pgx.ErrNoRows { return nil, ErrNotFound }
		return nil, fmt.Errorf("scan repo: %w", err)
	}
	x.Metadata = meta
	return &x, nil
}

// ErrNotFound is returned when a row doesn't exist.
var ErrNotFound = fmt.Errorf("not found")
