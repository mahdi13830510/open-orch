// Package github handles GitHub App webhook ingestion. Every incoming event
// is verified, normalized, and persisted into PostgreSQL *before* any
// processing — event ingestion and processing are decoupled.
package github

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	"github.com/open-orch/backend/internal/db"
	"github.com/open-orch/backend/internal/models"
	"github.com/rs/zerolog"
)

// Handler is the HTTP webhook endpoint.
type Handler struct {
	Secret string
	Events *db.EventStore
	Log    zerolog.Logger
}

// PRPayload is a trimmed version of GitHub's pull_request event we care about.
// We keep it loose; the full payload is also persisted.
type PRPayload struct {
	Action      string `json:"action"`
	Number      int    `json:"number"`
	PullRequest struct {
		Number  int    `json:"number"`
		State   string `json:"state"`
		Merged  bool   `json:"merged"`
		Title   string `json:"title"`
		Head    struct {
			Ref string `json:"ref"`
			SHA string `json:"sha"`
		} `json:"head"`
		User struct {
			Login string `json:"login"`
		} `json:"user"`
		Labels []struct {
			Name string `json:"name"`
		} `json:"labels"`
	} `json:"pull_request"`
	Repository struct {
		Name     string `json:"name"`
		FullName string `json:"full_name"`
	} `json:"repository"`
}

// PushPayload: minimal slice of GitHub's push event.
type PushPayload struct {
	Ref     string `json:"ref"` // refs/heads/...
	After   string `json:"after"`
	Before  string `json:"before"`
	Repository struct {
		Name     string `json:"name"`
		FullName string `json:"full_name"`
	} `json:"repository"`
	Pusher struct{ Name string `json:"name"` } `json:"pusher"`
}

// ServeHTTP implements the webhook receiver.
func (h *Handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	body, err := io.ReadAll(io.LimitReader(r.Body, 5<<20))
	if err != nil {
		http.Error(w, "read", http.StatusBadRequest)
		return
	}
	defer r.Body.Close()

	if h.Secret != "" {
		sig := r.Header.Get("X-Hub-Signature-256")
		if !verifySignature(h.Secret, sig, body) {
			h.Log.Warn().Str("sig", sig).Msg("invalid signature")
			http.Error(w, "invalid signature", http.StatusUnauthorized)
			return
		}
	}

	evType := r.Header.Get("X-GitHub-Event")
	delivery := r.Header.Get("X-GitHub-Delivery")

	// We extract a quick "action" + repo for indexing without fully parsing.
	var meta struct {
		Action     string `json:"action"`
		Repository struct {
			FullName string `json:"full_name"`
		} `json:"repository"`
	}
	_ = json.Unmarshal(body, &meta)

	ev := &models.Event{
		Source:     "github",
		DeliveryID: delivery,
		EventType:  evType,
		Action:     meta.Action,
		Repository: meta.Repository.FullName,
		Payload:    body,
	}
	if err := h.Events.Ingest(r.Context(), ev); err != nil {
		h.Log.Error().Err(err).Msg("ingest event")
		http.Error(w, "ingest failed", http.StatusInternalServerError)
		return
	}
	h.Log.Info().
		Str("delivery", delivery).
		Str("event_type", evType).
		Str("action", meta.Action).
		Str("repo", meta.Repository.FullName).
		Msg("event ingested")

	w.WriteHeader(http.StatusAccepted)
	_, _ = fmt.Fprintf(w, `{"accepted":true,"event_id":"%s"}`, ev.ID)
}

func verifySignature(secret, sigHeader string, body []byte) bool {
	const prefix = "sha256="
	if len(sigHeader) < len(prefix) || sigHeader[:len(prefix)] != prefix {
		return false
	}
	want, err := hex.DecodeString(sigHeader[len(prefix):])
	if err != nil {
		return false
	}
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write(body)
	got := mac.Sum(nil)
	return hmac.Equal(want, got)
}

// Decode helpers for the processor.
func DecodePR(raw json.RawMessage) (*PRPayload, error) {
	var p PRPayload
	if err := json.Unmarshal(raw, &p); err != nil {
		return nil, err
	}
	return &p, nil
}
func DecodePush(raw json.RawMessage) (*PushPayload, error) {
	var p PushPayload
	if err := json.Unmarshal(raw, &p); err != nil {
		return nil, err
	}
	return &p, nil
}

// Context utilities -----------------------------------------------------------

func withTimeoutCancel(parent context.Context) (context.Context, context.CancelFunc) {
	return context.WithCancel(parent)
}
