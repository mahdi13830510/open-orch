package db

import (
	"context"
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

// Integration is the persisted form of a third-party credential set.
// `Secret` is never returned from the API; only `HasSecret` is exposed
// so clients can tell whether one is configured.
type Integration struct {
	ID              uuid.UUID       `json:"id"`
	Kind            string          `json:"kind"`
	Name            string          `json:"name"`
	Config          json.RawMessage `json:"config"`
	HasSecret       bool            `json:"has_secret"`
	LastVerifiedAt  *time.Time      `json:"last_verified_at,omitempty"`
	LastError       string          `json:"last_error,omitempty"`
	CreatedAt       time.Time       `json:"created_at"`
	UpdatedAt       time.Time       `json:"updated_at"`
}

// IntegrationStore handles encrypted storage of third-party credentials.
// Secrets are encrypted with AES-256-GCM using a key derived from
// config.SecretKey; the DB only ever sees ciphertext + nonce.
type IntegrationStore struct {
	DB     *DB
	cipher cipher.AEAD // nil if SecretKey unset; calls requiring a secret will error
}

// NewIntegrationStore builds the store. If `secretKey` is empty, the store
// can still read and write integrations *that have no secret*, and will
// return ErrNoSecretKey for any operation requiring crypto.
func NewIntegrationStore(d *DB, secretKey string) (*IntegrationStore, error) {
	s := &IntegrationStore{DB: d}
	if secretKey == "" {
		return s, nil
	}
	key, err := decodeKey(secretKey)
	if err != nil {
		return nil, fmt.Errorf("ORCH_SECRET_KEY: %w", err)
	}
	block, err := aes.NewCipher(key)
	if err != nil { return nil, err }
	aead, err := cipher.NewGCM(block)
	if err != nil { return nil, err }
	s.cipher = aead
	return s, nil
}

// ErrNoSecretKey is returned when an integration with a secret value is
// requested but no master key was configured.
var ErrNoSecretKey = errors.New("ORCH_SECRET_KEY not configured")

// IntegrationUpsert is the write-side DTO. If Secret is nil, the existing
// stored secret (if any) is left alone. To clear a secret, pass an empty
// non-nil string pointer (i.e. *string pointing at "").
type IntegrationUpsert struct {
	Kind   string
	Name   string
	Config json.RawMessage
	Secret *string
}

func (s *IntegrationStore) Upsert(ctx context.Context, in IntegrationUpsert) (*Integration, error) {
	if in.Kind == "" || in.Name == "" {
		return nil, fmt.Errorf("kind and name required")
	}
	if !validIntegrationKind(in.Kind) {
		return nil, fmt.Errorf("invalid integration kind: %s", in.Kind)
	}
	if len(in.Config) == 0 { in.Config = json.RawMessage(`{}`) }

	var encVal, nonceVal []byte
	clearSecret := false
	if in.Secret != nil {
		if *in.Secret == "" {
			// caller explicitly wants the secret cleared
			clearSecret = true
		} else {
			if s.cipher == nil { return nil, ErrNoSecretKey }
			nonce := make([]byte, s.cipher.NonceSize())
			if _, err := io.ReadFull(rand.Reader, nonce); err != nil { return nil, err }
			encVal = s.cipher.Seal(nil, nonce, []byte(*in.Secret), nil)
			nonceVal = nonce
		}
	}

	// We use a CTE-free upsert that keeps the existing secret on update
	// unless the caller passed Secret != nil.
	const q = `
INSERT INTO integrations (kind, name, config, secret_enc, secret_nonce)
VALUES ($1, $2, $3, $4, $5)
ON CONFLICT (kind, name) DO UPDATE SET
  config       = EXCLUDED.config,
  secret_enc   = CASE WHEN $6 THEN NULL
                      WHEN $4 IS NOT NULL THEN EXCLUDED.secret_enc
                      ELSE integrations.secret_enc END,
  secret_nonce = CASE WHEN $6 THEN NULL
                      WHEN $5 IS NOT NULL THEN EXCLUDED.secret_nonce
                      ELSE integrations.secret_nonce END,
  updated_at   = NOW()
RETURNING id, kind, name, config, (secret_enc IS NOT NULL), last_verified_at, last_error, created_at, updated_at`

	row := s.DB.Pool.QueryRow(ctx, q,
		in.Kind, in.Name, in.Config, encVal, nonceVal, clearSecret,
	)
	return scanIntegration(row)
}

func (s *IntegrationStore) Get(ctx context.Context, id uuid.UUID) (*Integration, error) {
	const q = `SELECT id, kind, name, config, (secret_enc IS NOT NULL),
		last_verified_at, last_error, created_at, updated_at
		FROM integrations WHERE id=$1`
	return scanIntegration(s.DB.Pool.QueryRow(ctx, q, id))
}

// ByKindName looks up an integration by its natural key. Useful for the
// reconciler / processors that want "the GitHub App named primary".
func (s *IntegrationStore) ByKindName(ctx context.Context, kind, name string) (*Integration, error) {
	const q = `SELECT id, kind, name, config, (secret_enc IS NOT NULL),
		last_verified_at, last_error, created_at, updated_at
		FROM integrations WHERE kind=$1 AND name=$2`
	return scanIntegration(s.DB.Pool.QueryRow(ctx, q, kind, name))
}

// List returns all integrations, optionally filtered by kind ("" = all).
func (s *IntegrationStore) List(ctx context.Context, kind string) ([]Integration, error) {
	var rows pgx.Rows
	var err error
	if kind == "" {
		rows, err = s.DB.Pool.Query(ctx,
			`SELECT id, kind, name, config, (secret_enc IS NOT NULL),
			 last_verified_at, last_error, created_at, updated_at
			 FROM integrations ORDER BY kind, name`)
	} else {
		rows, err = s.DB.Pool.Query(ctx,
			`SELECT id, kind, name, config, (secret_enc IS NOT NULL),
			 last_verified_at, last_error, created_at, updated_at
			 FROM integrations WHERE kind=$1 ORDER BY name`, kind)
	}
	if err != nil { return nil, err }
	defer rows.Close()
	var out []Integration
	for rows.Next() {
		i, err := scanIntegration(rows)
		if err != nil { return nil, err }
		out = append(out, *i)
	}
	return out, rows.Err()
}

func (s *IntegrationStore) Delete(ctx context.Context, id uuid.UUID) (bool, error) {
	ct, err := s.DB.Pool.Exec(ctx, `DELETE FROM integrations WHERE id=$1`, id)
	if err != nil { return false, err }
	return ct.RowsAffected() > 0, nil
}

// Reveal decrypts and returns the stored secret value. Reserved for
// internal callers (the buddy client, the github webhook verifier) — never
// exposed by the REST API.
func (s *IntegrationStore) Reveal(ctx context.Context, id uuid.UUID) (string, error) {
	if s.cipher == nil { return "", ErrNoSecretKey }
	var enc, nonce []byte
	err := s.DB.Pool.QueryRow(ctx,
		`SELECT secret_enc, secret_nonce FROM integrations WHERE id=$1`, id,
	).Scan(&enc, &nonce)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) { return "", ErrNotFound }
		return "", err
	}
	if enc == nil { return "", nil }
	plain, err := s.cipher.Open(nil, nonce, enc, nil)
	if err != nil { return "", fmt.Errorf("decrypt: %w", err) }
	return string(plain), nil
}

// MarkVerified records the result of an external verification call (e.g. a
// "GET /user" against Buddy to confirm the token works).
func (s *IntegrationStore) MarkVerified(ctx context.Context, id uuid.UUID, ok bool, detail string) error {
	if ok {
		_, err := s.DB.Pool.Exec(ctx,
			`UPDATE integrations SET last_verified_at=NOW(), last_error=NULL, updated_at=NOW() WHERE id=$1`, id)
		return err
	}
	_, err := s.DB.Pool.Exec(ctx,
		`UPDATE integrations SET last_error=$2, updated_at=NOW() WHERE id=$1`, id, detail)
	return err
}

// ----------- helpers -----------

func validIntegrationKind(k string) bool {
	switch k {
	case "github_app", "buddy", "cloudflare", "registry", "webhook", "custom":
		return true
	}
	return false
}

func scanIntegration(r rowScanner) (*Integration, error) {
	var x Integration
	var cfg []byte
	err := r.Scan(&x.ID, &x.Kind, &x.Name, &cfg, &x.HasSecret,
		&x.LastVerifiedAt, &x.LastError, &x.CreatedAt, &x.UpdatedAt)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) { return nil, ErrNotFound }
		return nil, fmt.Errorf("scan integration: %w", err)
	}
	x.Config = cfg
	return &x, nil
}

// decodeKey accepts a 32-byte AES-256 key as either:
//   - raw hex (64 chars)
//   - standard or url-safe base64
// Anything else is rejected.
func decodeKey(s string) ([]byte, error) {
	s = strings.TrimSpace(s)
	// hex first (unambiguous: only 0-9a-f)
	if len(s) == 64 {
		if b, err := hex.DecodeString(s); err == nil && len(b) == 32 {
			return b, nil
		}
	}
	// base64
	for _, dec := range []func(string) ([]byte, error){
		base64.StdEncoding.DecodeString,
		base64.RawStdEncoding.DecodeString,
		base64.URLEncoding.DecodeString,
		base64.RawURLEncoding.DecodeString,
	} {
		if b, err := dec(s); err == nil && len(b) == 32 {
			return b, nil
		}
	}
	return nil, fmt.Errorf("must be 32 bytes (hex or base64)")
}
