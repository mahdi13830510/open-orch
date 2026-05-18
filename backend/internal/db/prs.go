package db

import (
	"context"
	"encoding/json"
	"time"

	"github.com/open-orch/backend/internal/models"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

// ---------------------------- PullRequestStore ------------------------------

type PullRequestStore struct{ DB *DB }

func (s *PullRequestStore) Upsert(ctx context.Context, pr *models.PullRequest) error {
	labels, _ := json.Marshal(pr.Labels)
	const q = `
INSERT INTO pull_requests (repository_id, number, branch, head_sha, state, title, author, labels, opened_at, updated_at)
VALUES ($1,$2,$3,$4,$5,$6,$7,$8,NOW(),NOW())
ON CONFLICT (repository_id, number) DO UPDATE SET
  branch=EXCLUDED.branch, head_sha=EXCLUDED.head_sha, state=EXCLUDED.state,
  title=EXCLUDED.title, author=EXCLUDED.author, labels=EXCLUDED.labels, updated_at=NOW()
RETURNING id`
	return s.DB.Pool.QueryRow(ctx, q,
		pr.RepositoryID, pr.Number, pr.Branch, pr.HeadSHA, pr.State,
		pr.Title, pr.Author, labels,
	).Scan(&pr.ID)
}

func (s *PullRequestStore) MarkClosed(ctx context.Context, repoID uuid.UUID, number int) error {
	const q = `UPDATE pull_requests SET state='closed', closed_at=NOW(), updated_at=NOW()
		WHERE repository_id=$1 AND number=$2`
	_, err := s.DB.Pool.Exec(ctx, q, repoID, number)
	return err
}

// FindOpenByBranch returns the open PR (if any) matching a given branch
// inside a repository. Used to discover a "matching branch" across repos.
func (s *PullRequestStore) FindOpenByBranch(ctx context.Context, repoID uuid.UUID, branch string) (*models.PullRequest, error) {
	const q = `SELECT id, repository_id, number, branch, head_sha, state, title, author, labels
		FROM pull_requests WHERE repository_id=$1 AND branch=$2 AND state='open'
		ORDER BY updated_at DESC LIMIT 1`
	row := s.DB.Pool.QueryRow(ctx, q, repoID, branch)
	var pr models.PullRequest
	var labels []byte
	err := row.Scan(&pr.ID, &pr.RepositoryID, &pr.Number, &pr.Branch, &pr.HeadSHA,
		&pr.State, &pr.Title, &pr.Author, &labels)
	if err != nil {
		if err == pgx.ErrNoRows { return nil, ErrNotFound }
		return nil, err
	}
	_ = json.Unmarshal(labels, &pr.Labels)
	return &pr, nil
}

// ------------------------------- FeatureStore -------------------------------

type FeatureStore struct{ DB *DB }

func (s *FeatureStore) GetOrCreate(ctx context.Context, slug string) (*models.Feature, error) {
	const q = `
INSERT INTO features (slug, display_name, last_seen_at)
VALUES ($1, $1, NOW())
ON CONFLICT (slug) DO UPDATE SET last_seen_at=NOW()
RETURNING id, slug, display_name, created_at, last_seen_at`
	var f models.Feature
	err := s.DB.Pool.QueryRow(ctx, q, slug).
		Scan(&f.ID, &f.Slug, &f.DisplayName, &f.CreatedAt, &f.LastSeenAt)
	return &f, err
}

func (s *FeatureStore) BySlug(ctx context.Context, slug string) (*models.Feature, error) {
	const q = `SELECT id,slug,display_name,created_at,last_seen_at FROM features WHERE slug=$1`
	var f models.Feature
	err := s.DB.Pool.QueryRow(ctx, q, slug).
		Scan(&f.ID, &f.Slug, &f.DisplayName, &f.CreatedAt, &f.LastSeenAt)
	if err == pgx.ErrNoRows { return nil, ErrNotFound }
	return &f, err
}

func (s *FeatureStore) Touch(ctx context.Context, id uuid.UUID) error {
	_, err := s.DB.Pool.Exec(ctx, `UPDATE features SET last_seen_at=NOW() WHERE id=$1`, id)
	return err
}

func (s *FeatureStore) TouchedBefore(ctx context.Context, t time.Time) ([]models.Feature, error) {
	const q = `SELECT id,slug,display_name,created_at,last_seen_at FROM features WHERE last_seen_at<$1`
	rows, err := s.DB.Pool.Query(ctx, q, t)
	if err != nil { return nil, err }
	defer rows.Close()
	var out []models.Feature
	for rows.Next() {
		var f models.Feature
		if err := rows.Scan(&f.ID, &f.Slug, &f.DisplayName, &f.CreatedAt, &f.LastSeenAt); err != nil {
			return nil, err
		}
		out = append(out, f)
	}
	return out, rows.Err()
}
