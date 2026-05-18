package db

import (
	"context"
	"encoding/json"
	"time"

	"github.com/open-orch/backend/internal/models"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

// -------------------------------- EventStore --------------------------------

type EventStore struct{ DB *DB }

// Ingest persists the event before any processing. Idempotent on delivery_id.
func (s *EventStore) Ingest(ctx context.Context, e *models.Event) error {
	if len(e.Payload) == 0 { e.Payload = json.RawMessage("{}") }
	const q = `
INSERT INTO events (source, delivery_id, event_type, action, repository, payload)
VALUES ($1,$2,$3,$4,$5,$6)
ON CONFLICT (delivery_id) DO NOTHING
RETURNING id, received_at`
	row := s.DB.Pool.QueryRow(ctx, q, e.Source, nullable(e.DeliveryID), e.EventType, e.Action, e.Repository, e.Payload)
	err := row.Scan(&e.ID, &e.ReceivedAt)
	if err == pgx.ErrNoRows {
		// Already ingested; not an error. Look up existing.
		return s.DB.Pool.QueryRow(ctx,
			`SELECT id, received_at FROM events WHERE delivery_id=$1`, e.DeliveryID,
		).Scan(&e.ID, &e.ReceivedAt)
	}
	return err
}

// NextBatch grabs unprocessed events with a SKIP LOCKED claim so multiple
// workers can pull events concurrently without stepping on each other.
func (s *EventStore) NextBatch(ctx context.Context, limit int) ([]models.Event, error) {
	const q = `
WITH next AS (
  SELECT id FROM events
  WHERE processed_at IS NULL AND attempt_count < 10
  ORDER BY received_at ASC
  FOR UPDATE SKIP LOCKED
  LIMIT $1
)
UPDATE events e SET attempt_count = attempt_count + 1
FROM next WHERE e.id = next.id
RETURNING e.id, e.source, e.delivery_id, e.event_type, e.action, e.repository,
          e.payload, e.received_at, e.processed_at, e.process_error, e.attempt_count`
	rows, err := s.DB.Pool.Query(ctx, q, limit)
	if err != nil { return nil, err }
	defer rows.Close()
	var out []models.Event
	for rows.Next() {
		var ev models.Event
		var payload []byte
		var deliveryID, action, repo, perr *string
		if err := rows.Scan(&ev.ID, &ev.Source, &deliveryID, &ev.EventType, &action,
			&repo, &payload, &ev.ReceivedAt, &ev.ProcessedAt, &perr, &ev.AttemptCount); err != nil {
			return nil, err
		}
		if deliveryID != nil { ev.DeliveryID = *deliveryID }
		if action != nil { ev.Action = *action }
		if repo != nil { ev.Repository = *repo }
		if perr != nil { ev.ProcessError = *perr }
		ev.Payload = payload
		out = append(out, ev)
	}
	return out, rows.Err()
}

func (s *EventStore) MarkProcessed(ctx context.Context, id uuid.UUID) error {
	_, err := s.DB.Pool.Exec(ctx, `UPDATE events SET processed_at=NOW(), process_error=NULL WHERE id=$1`, id)
	return err
}
func (s *EventStore) MarkError(ctx context.Context, id uuid.UUID, errMsg string) error {
	_, err := s.DB.Pool.Exec(ctx, `UPDATE events SET process_error=$2 WHERE id=$1`, id, errMsg)
	return err
}

// ---------------------------- RuntimeResourceStore --------------------------

type RuntimeResourceStore struct{ DB *DB }

func (s *RuntimeResourceStore) Upsert(ctx context.Context, r *models.RuntimeResource) error {
	const q = `
INSERT INTO runtime_resources (environment_id, kind, external_id, name, state, last_seen_at)
VALUES ($1,$2,$3,$4,$5,NOW())
ON CONFLICT DO NOTHING
RETURNING id, last_seen_at`
	row := s.DB.Pool.QueryRow(ctx, q, r.EnvironmentID, r.Kind, r.ExternalID, r.Name, r.State)
	err := row.Scan(&r.ID, &r.LastSeenAt)
	if err == pgx.ErrNoRows {
		_, err = s.DB.Pool.Exec(ctx,
			`UPDATE runtime_resources SET external_id=$3, state=$4, last_seen_at=NOW()
			 WHERE environment_id=$1 AND name=$2`, r.EnvironmentID, r.Name, r.ExternalID, r.State)
	}
	return err
}

func (s *RuntimeResourceStore) ByEnv(ctx context.Context, envID uuid.UUID) ([]models.RuntimeResource, error) {
	rows, err := s.DB.Pool.Query(ctx,
		`SELECT id,environment_id,kind,external_id,name,state,last_seen_at
		 FROM runtime_resources WHERE environment_id=$1`, envID)
	if err != nil { return nil, err }
	defer rows.Close()
	var out []models.RuntimeResource
	for rows.Next() {
		var r models.RuntimeResource
		var extID *string
		if err := rows.Scan(&r.ID, &r.EnvironmentID, &r.Kind, &extID, &r.Name, &r.State, &r.LastSeenAt); err != nil {
			return nil, err
		}
		if extID != nil { r.ExternalID = *extID }
		out = append(out, r)
	}
	return out, rows.Err()
}

func (s *RuntimeResourceStore) DeleteForEnv(ctx context.Context, envID uuid.UUID) error {
	_, err := s.DB.Pool.Exec(ctx, `DELETE FROM runtime_resources WHERE environment_id=$1`, envID)
	return err
}

// ------------------------------- HealthChecks -------------------------------

type HealthCheckStore struct{ DB *DB }

func (s *HealthCheckStore) Record(ctx context.Context, depID uuid.UUID, status, detail string) error {
	_, err := s.DB.Pool.Exec(ctx,
		`INSERT INTO health_checks (deployment_id, status, detail) VALUES ($1,$2,$3)`,
		depID, status, detail)
	return err
}

func (s *HealthCheckStore) LatestForDeployment(ctx context.Context, depID uuid.UUID) (string, time.Time, error) {
	var status string
	var checked time.Time
	err := s.DB.Pool.QueryRow(ctx,
		`SELECT status, checked_at FROM health_checks WHERE deployment_id=$1
		 ORDER BY checked_at DESC LIMIT 1`, depID,
	).Scan(&status, &checked)
	if err == pgx.ErrNoRows { return "unknown", time.Time{}, nil }
	return status, checked, err
}

func nullable(s string) any {
	if s == "" { return nil }
	return s
}
