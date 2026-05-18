// Package api exposes the orchestrator's REST API. The API mirrors the
// control-plane intent: clients ask the orchestrator to create environments,
// reconcile them, restart them, or query state. They never talk to Docker
// or Traefik directly.
package api

import (
	"encoding/json"
	"errors"
	"net/http"
	"strings"
	"time"

	"github.com/open-orch/backend/internal/config"
	"github.com/open-orch/backend/internal/db"
	"github.com/open-orch/backend/internal/lifecycle"
	"github.com/open-orch/backend/internal/locks"
	"github.com/open-orch/backend/internal/models"
	"github.com/open-orch/backend/internal/normalizer"
	"github.com/open-orch/backend/internal/topology"
	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/rs/zerolog"
)

// Server holds dependencies for HTTP handlers.
type Server struct {
	Cfg          *config.Config
	Log          zerolog.Logger
	Repos        *db.RepositoryStore
	PRs          *db.PullRequestStore
	Features     *db.FeatureStore
	Envs         *db.EnvironmentStore
	Deploys      *db.DeploymentStore
	Domains      *db.DomainStore
	Runtime      *db.RuntimeResourceStore
	Events       *db.EventStore
	Integrations *db.IntegrationStore
	Resolver     *topology.Resolver
	Locks        *locks.Manager
}

// cors returns a middleware that sets CORS headers for the allowed origins.
func cors(origins string) func(http.Handler) http.Handler {
	allowed := map[string]bool{}
	for _, o := range strings.Split(origins, ",") {
		if t := strings.TrimSpace(o); t != "" {
			allowed[t] = true
		}
	}
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			origin := r.Header.Get("Origin")
			if origin != "" && (allowed["*"] || allowed[origin]) {
				w.Header().Set("Access-Control-Allow-Origin", origin)
				w.Header().Set("Access-Control-Allow-Methods", "GET,POST,PUT,PATCH,DELETE,OPTIONS")
				w.Header().Set("Access-Control-Allow-Headers", "Content-Type,Authorization")
				w.Header().Set("Vary", "Origin")
			}
			if r.Method == http.MethodOptions {
				w.WriteHeader(http.StatusNoContent)
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}

// Routes returns a chi.Router with all API routes mounted.
func (s *Server) Routes(webhook http.Handler) http.Handler {
	r := chi.NewRouter()
	r.Use(cors(s.Cfg.CORSOrigins))

	// Liveness / readiness.
	r.Get("/healthz", func(w http.ResponseWriter, _ *http.Request) {
		writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
	})

	// GitHub webhook (verified inside its own handler).
	r.Mount("/webhooks/github", webhook)

	// Repositories.
	r.Route("/repositories", func(r chi.Router) {
		r.Get("/", s.listRepos)
		r.Post("/", s.upsertRepo)
		r.Route("/{name}/dependencies", func(r chi.Router) {
			r.Get("/", s.listDeps)
			r.Post("/", s.addDep)
			r.Delete("/{depName}", s.removeDep)
		})
	})

	// Features.
	r.Route("/features", func(r chi.Router) {
		r.Get("/{slug}", s.getFeature)
	})

	// Environments.
	r.Route("/environments", func(r chi.Router) {
		r.Get("/", s.listEnvs)
		r.Post("/", s.createEnv)
		r.Get("/{id}", s.getEnv)
		r.Delete("/{id}", s.destroyEnv)
		r.Post("/{id}/suspend", s.suspendEnv)
		r.Post("/{id}/resume", s.resumeEnv)
	})

	// Deployments.
	r.Route("/deployments", func(r chi.Router) {
		r.Post("/reconcile", s.requestReconcile)
		r.Post("/restart", s.requestRestart)
	})

	// Integrations (third-party credentials).
	r.Route("/integrations", func(r chi.Router) {
		r.Get("/", s.listIntegrations)
		r.Post("/", s.upsertIntegration)
		r.Get("/{id}", s.getIntegration)
		r.Patch("/{id}", s.patchIntegration)
		r.Delete("/{id}", s.deleteIntegration)
		r.Post("/{id}/verify", s.verifyIntegration)
	})

	// Events (debugging).
	r.Route("/events", func(r chi.Router) {
		r.Get("/", s.listEvents)
	})

	return r
}

// ----------------- Repositories -----------------

type repoIn struct {
	Name          string          `json:"name"`
	FullName      string          `json:"full_name"`
	DefaultBranch string          `json:"default_branch"`
	ServiceKind   string          `json:"service_kind"`
	ExposePort    *int            `json:"expose_port,omitempty"`
	BuddyProject  string          `json:"buddy_project"`
	BuddyPipeline string          `json:"buddy_pipeline"`
	Metadata      json.RawMessage `json:"metadata,omitempty"`
}

func (s *Server) listRepos(w http.ResponseWriter, r *http.Request) {
	out, err := s.Repos.List(r.Context())
	if err != nil { writeErr(w, err); return }
	writeJSON(w, http.StatusOK, out)
}

func (s *Server) upsertRepo(w http.ResponseWriter, r *http.Request) {
	var in repoIn
	if err := json.NewDecoder(r.Body).Decode(&in); err != nil {
		writeJSON(w, http.StatusBadRequest, errResp("invalid body"))
		return
	}
	if in.Name == "" || in.FullName == "" {
		writeJSON(w, http.StatusBadRequest, errResp("name and full_name required"))
		return
	}
	if in.DefaultBranch == "" { in.DefaultBranch = "main" }
	if in.ServiceKind == "" { in.ServiceKind = "http" }
	rec := &models.Repository{
		Name: in.Name, FullName: in.FullName, DefaultBranch: in.DefaultBranch,
		ServiceKind: in.ServiceKind, ExposePort: in.ExposePort,
		BuddyProject: in.BuddyProject, BuddyPipeline: in.BuddyPipeline,
		Metadata: in.Metadata,
	}
	if err := s.Repos.Upsert(r.Context(), rec); err != nil { writeErr(w, err); return }
	writeJSON(w, http.StatusOK, rec)
}

type depIn struct {
	DependsOn string `json:"depends_on"` // name
	Required  bool   `json:"required"`
}

func (s *Server) addDep(w http.ResponseWriter, r *http.Request) {
	name := chi.URLParam(r, "name")
	var in depIn
	if err := json.NewDecoder(r.Body).Decode(&in); err != nil {
		writeJSON(w, http.StatusBadRequest, errResp("invalid body"))
		return
	}
	from, err := s.Repos.GetByName(r.Context(), name)
	if err != nil { writeJSON(w, http.StatusNotFound, errResp("repo not found")); return }
	to, err := s.Repos.GetByName(r.Context(), in.DependsOn)
	if err != nil { writeJSON(w, http.StatusNotFound, errResp("dep not found")); return }
	if from.ID == to.ID {
		writeJSON(w, http.StatusBadRequest, errResp("a repository cannot depend on itself"))
		return
	}
	if err := s.Repos.AddDependency(r.Context(), from.ID, to.ID, in.Required); err != nil {
		writeErr(w, err); return
	}
	writeJSON(w, http.StatusCreated, map[string]string{"status": "linked"})
}

func (s *Server) listDeps(w http.ResponseWriter, r *http.Request) {
	name := chi.URLParam(r, "name")
	repo, err := s.Repos.GetByName(r.Context(), name)
	if err != nil { writeJSON(w, http.StatusNotFound, errResp("repo not found")); return }
	deps, err := s.Repos.ListDependencies(r.Context(), repo.ID)
	if err != nil { writeErr(w, err); return }
	if deps == nil { deps = []db.RepositoryDependency{} }
	writeJSON(w, http.StatusOK, deps)
}

func (s *Server) removeDep(w http.ResponseWriter, r *http.Request) {
	name := chi.URLParam(r, "name")
	depName := chi.URLParam(r, "depName")
	repo, err := s.Repos.GetByName(r.Context(), name)
	if err != nil { writeJSON(w, http.StatusNotFound, errResp("repo not found")); return }
	dep, err := s.Repos.GetByName(r.Context(), depName)
	if err != nil { writeJSON(w, http.StatusNotFound, errResp("dep not found")); return }
	removed, err := s.Repos.RemoveDependency(r.Context(), repo.ID, dep.ID)
	if err != nil { writeErr(w, err); return }
	if !removed {
		writeJSON(w, http.StatusNotFound, errResp("no such dependency"))
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "unlinked"})
}

// ----------------- Features -----------------

func (s *Server) getFeature(w http.ResponseWriter, r *http.Request) {
	slug := normalizer.Branch(chi.URLParam(r, "slug"))
	f, err := s.Features.BySlug(r.Context(), slug)
	if err != nil {
		writeJSON(w, http.StatusNotFound, errResp("feature not found"))
		return
	}
	env, _ := s.Envs.ByFeature(r.Context(), f.ID)
	writeJSON(w, http.StatusOK, map[string]any{
		"feature":     f,
		"environment": env,
	})
}

// ----------------- Environments -----------------

type envIn struct {
	Feature string `json:"feature"`
	TTL     string `json:"ttl,omitempty"` // duration
}

func (s *Server) createEnv(w http.ResponseWriter, r *http.Request) {
	var in envIn
	if err := json.NewDecoder(r.Body).Decode(&in); err != nil {
		writeJSON(w, http.StatusBadRequest, errResp("invalid body"))
		return
	}
	slug := normalizer.Branch(in.Feature)
	if slug == "" {
		writeJSON(w, http.StatusBadRequest, errResp("feature required"))
		return
	}
	feat, err := s.Features.GetOrCreate(r.Context(), slug)
	if err != nil { writeErr(w, err); return }

	existing, err := s.Envs.ByFeature(r.Context(), feat.ID)
	if err == nil {
		writeJSON(w, http.StatusOK, existing)
		return
	}
	if !errors.Is(err, db.ErrNotFound) {
		writeErr(w, err); return
	}

	ttl := int(s.Cfg.DefaultEnvTTL.Seconds())
	if in.TTL != "" {
		if d, err := time.ParseDuration(in.TTL); err == nil {
			ttl = int(d.Seconds())
		}
	}
	env := &models.Environment{
		ShortID:      "env_" + shortHash(slug),
		FeatureID:    feat.ID,
		State:        models.EnvPending,
		BaseDomain:   s.Cfg.BaseDomain,
		TTLSeconds:   &ttl,
	}
	env.DockerNetwork = env.ShortID
	if err := s.Envs.Create(r.Context(), env); err != nil { writeErr(w, err); return }

	// Resolve initial topology too — the reconciler will pick it up next tick.
	topo, err := s.Resolver.Resolve(r.Context(), env, feat, nil)
	if err == nil {
		raw, _ := json.Marshal(topo)
		_, _ = s.Envs.BumpGeneration(r.Context(), env.ID, raw)
		_ = s.Envs.UpdateState(r.Context(), env.ID, models.EnvResolving)
	}
	writeJSON(w, http.StatusCreated, env)
}

func (s *Server) listEnvs(w http.ResponseWriter, r *http.Request) {
	out, err := s.Envs.List(r.Context())
	if err != nil { writeErr(w, err); return }
	writeJSON(w, http.StatusOK, out)
}

func (s *Server) getEnv(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	env, err := s.lookupEnv(r, id)
	if err != nil {
		writeJSON(w, http.StatusNotFound, errResp("not found"))
		return
	}
	deps, _ := s.Deploys.ByEnvGeneration(r.Context(), env.ID, env.Generation)
	domains, _ := s.Domains.ByEnv(r.Context(), env.ID)
	runtime, _ := s.Runtime.ByEnv(r.Context(), env.ID)
	writeJSON(w, http.StatusOK, map[string]any{
		"environment": env,
		"deployments": deps,
		"domains":     domains,
		"runtime":     runtime,
	})
}

func (s *Server) destroyEnv(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	env, err := s.lookupEnv(r, id)
	if err != nil { writeJSON(w, http.StatusNotFound, errResp("not found")); return }
	if err := lifecycle.CanTransition(env.State, models.EnvDestroying); err != nil {
		writeJSON(w, http.StatusConflict, errResp(err.Error()))
		return
	}
	if err := s.Envs.UpdateState(r.Context(), env.ID, models.EnvDestroying); err != nil {
		writeErr(w, err); return
	}
	writeJSON(w, http.StatusAccepted, map[string]string{"status": "destroying"})
}

// suspendEnv freezes an environment in place. Containers keep running but the
// reconciler stops acting on it until /resume is called. Useful for pausing
// flaky branches without tearing down state.
func (s *Server) suspendEnv(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	env, err := s.lookupEnv(r, id)
	if err != nil { writeJSON(w, http.StatusNotFound, errResp("not found")); return }
	if env.State == models.EnvDestroying || env.State == models.EnvDestroyed {
		writeJSON(w, http.StatusConflict, errResp("cannot suspend a destroying/destroyed environment"))
		return
	}
	if err := s.Envs.SetSuspended(r.Context(), env.ID, true); err != nil {
		writeErr(w, err); return
	}
	writeJSON(w, http.StatusOK, map[string]any{"status": "suspended", "id": env.ID, "short_id": env.ShortID})
}

func (s *Server) resumeEnv(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	env, err := s.lookupEnv(r, id)
	if err != nil { writeJSON(w, http.StatusNotFound, errResp("not found")); return }
	if err := s.Envs.SetSuspended(r.Context(), env.ID, false); err != nil {
		writeErr(w, err); return
	}
	// Touch to nudge the reconciler.
	_ = s.Envs.TouchEvent(r.Context(), env.ID)
	writeJSON(w, http.StatusOK, map[string]any{"status": "resumed", "id": env.ID, "short_id": env.ShortID})
}

// ----------------- Deployments actions -----------------

type reconcileIn struct {
	EnvironmentID string `json:"environment_id"`
}

func (s *Server) requestReconcile(w http.ResponseWriter, r *http.Request) {
	var in reconcileIn
	if err := json.NewDecoder(r.Body).Decode(&in); err != nil {
		writeJSON(w, http.StatusBadRequest, errResp("invalid body"))
		return
	}
	env, err := s.lookupEnv(r, in.EnvironmentID)
	if err != nil { writeJSON(w, http.StatusNotFound, errResp("not found")); return }
	// Touch will cause reconciler to pick it up.
	_ = s.Envs.TouchEvent(r.Context(), env.ID)
	writeJSON(w, http.StatusAccepted, map[string]string{"status": "queued"})
}

type restartIn struct {
	EnvironmentID string `json:"environment_id"`
}

func (s *Server) requestRestart(w http.ResponseWriter, r *http.Request) {
	var in restartIn
	if err := json.NewDecoder(r.Body).Decode(&in); err != nil {
		writeJSON(w, http.StatusBadRequest, errResp("invalid body"))
		return
	}
	env, err := s.lookupEnv(r, in.EnvironmentID)
	if err != nil { writeJSON(w, http.StatusNotFound, errResp("not found")); return }
	feat, err := s.Features.BySlug(r.Context(), featureSlugForEnv(env))
	if err == nil {
		topo, err := s.Resolver.Resolve(r.Context(), env, feat, nil)
		if err == nil {
			raw, _ := json.Marshal(topo)
			_, _ = s.Envs.BumpGeneration(r.Context(), env.ID, raw)
		}
	}
	_ = s.Envs.UpdateState(r.Context(), env.ID, models.EnvDeploying)
	writeJSON(w, http.StatusAccepted, map[string]string{"status": "restarting"})
}

func featureSlugForEnv(env *models.Environment) string {
	var t models.Topology
	_ = json.Unmarshal(env.DesiredTopology, &t)
	return t.FeatureSlug
}

// ----------------- Integrations -----------------
//
// Integrations store third-party credentials (GitHub App keys, Buddy tokens,
// Cloudflare DNS tokens, container registry logins). Secrets are encrypted at
// rest by IntegrationStore using AES-256-GCM and are *never* returned by these
// endpoints — only `has_secret: true|false` is exposed.

type integrationIn struct {
	Kind   string          `json:"kind"`
	Name   string          `json:"name"`
	Config json.RawMessage `json:"config,omitempty"`
	Secret *string         `json:"secret,omitempty"` // omit to leave existing alone; "" clears
}

func (s *Server) listIntegrations(w http.ResponseWriter, r *http.Request) {
	if s.Integrations == nil {
		writeJSON(w, http.StatusServiceUnavailable, errResp("integrations subsystem not configured"))
		return
	}
	kind := r.URL.Query().Get("kind")
	out, err := s.Integrations.List(r.Context(), kind)
	if err != nil { writeErr(w, err); return }
	if out == nil { out = []db.Integration{} }
	writeJSON(w, http.StatusOK, out)
}

func (s *Server) upsertIntegration(w http.ResponseWriter, r *http.Request) {
	if s.Integrations == nil {
		writeJSON(w, http.StatusServiceUnavailable, errResp("integrations subsystem not configured"))
		return
	}
	var in integrationIn
	if err := json.NewDecoder(r.Body).Decode(&in); err != nil {
		writeJSON(w, http.StatusBadRequest, errResp("invalid body"))
		return
	}
	rec, err := s.Integrations.Upsert(r.Context(), db.IntegrationUpsert{
		Kind:   in.Kind,
		Name:   in.Name,
		Config: in.Config,
		Secret: in.Secret,
	})
	if err != nil {
		if errors.Is(err, db.ErrNoSecretKey) {
			writeJSON(w, http.StatusFailedDependency, errResp("ORCH_SECRET_KEY not configured; cannot store secrets"))
			return
		}
		// validation errors get 400, anything else 500
		msg := err.Error()
		if msg == "kind and name required" || hasPrefix(msg, "invalid integration kind") {
			writeJSON(w, http.StatusBadRequest, errResp(msg))
			return
		}
		writeErr(w, err); return
	}
	writeJSON(w, http.StatusOK, rec)
}

func (s *Server) getIntegration(w http.ResponseWriter, r *http.Request) {
	if s.Integrations == nil {
		writeJSON(w, http.StatusServiceUnavailable, errResp("integrations subsystem not configured"))
		return
	}
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil { writeJSON(w, http.StatusBadRequest, errResp("invalid id")); return }
	rec, err := s.Integrations.Get(r.Context(), id)
	if err != nil {
		if errors.Is(err, db.ErrNotFound) { writeJSON(w, http.StatusNotFound, errResp("not found")); return }
		writeErr(w, err); return
	}
	writeJSON(w, http.StatusOK, rec)
}

// patchIntegration is a partial update; the body has the same shape as upsert.
// The PATCH variant loads the existing row first so the caller can update only
// the fields they care about (most usefully: rotate the secret without
// re-sending config).
func (s *Server) patchIntegration(w http.ResponseWriter, r *http.Request) {
	if s.Integrations == nil {
		writeJSON(w, http.StatusServiceUnavailable, errResp("integrations subsystem not configured"))
		return
	}
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil { writeJSON(w, http.StatusBadRequest, errResp("invalid id")); return }
	existing, err := s.Integrations.Get(r.Context(), id)
	if err != nil {
		if errors.Is(err, db.ErrNotFound) { writeJSON(w, http.StatusNotFound, errResp("not found")); return }
		writeErr(w, err); return
	}
	var in integrationIn
	if err := json.NewDecoder(r.Body).Decode(&in); err != nil {
		writeJSON(w, http.StatusBadRequest, errResp("invalid body"))
		return
	}
	upd := db.IntegrationUpsert{
		Kind:   existing.Kind,         // immutable
		Name:   existing.Name,         // immutable
		Config: existing.Config,
		Secret: in.Secret,
	}
	if len(in.Config) > 0 { upd.Config = in.Config }
	rec, err := s.Integrations.Upsert(r.Context(), upd)
	if err != nil {
		if errors.Is(err, db.ErrNoSecretKey) {
			writeJSON(w, http.StatusFailedDependency, errResp("ORCH_SECRET_KEY not configured"))
			return
		}
		writeErr(w, err); return
	}
	writeJSON(w, http.StatusOK, rec)
}

func (s *Server) deleteIntegration(w http.ResponseWriter, r *http.Request) {
	if s.Integrations == nil {
		writeJSON(w, http.StatusServiceUnavailable, errResp("integrations subsystem not configured"))
		return
	}
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil { writeJSON(w, http.StatusBadRequest, errResp("invalid id")); return }
	ok, err := s.Integrations.Delete(r.Context(), id)
	if err != nil { writeErr(w, err); return }
	if !ok { writeJSON(w, http.StatusNotFound, errResp("not found")); return }
	writeJSON(w, http.StatusOK, map[string]string{"status": "deleted"})
}

// verifyIntegration is a stub for client-driven verification. The orchestrator
// records the outcome the caller reports — this lets the UI run its own
// "test connection" check and persist the result without giving clients access
// to the raw secret. A future revision can move verification server-side.
func (s *Server) verifyIntegration(w http.ResponseWriter, r *http.Request) {
	if s.Integrations == nil {
		writeJSON(w, http.StatusServiceUnavailable, errResp("integrations subsystem not configured"))
		return
	}
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil { writeJSON(w, http.StatusBadRequest, errResp("invalid id")); return }
	var in struct {
		OK     bool   `json:"ok"`
		Detail string `json:"detail,omitempty"`
	}
	if err := json.NewDecoder(r.Body).Decode(&in); err != nil {
		writeJSON(w, http.StatusBadRequest, errResp("invalid body"))
		return
	}
	if err := s.Integrations.MarkVerified(r.Context(), id, in.OK, in.Detail); err != nil {
		writeErr(w, err); return
	}
	rec, _ := s.Integrations.Get(r.Context(), id)
	writeJSON(w, http.StatusOK, rec)
}

// ----------------- Events -----------------

func (s *Server) listEvents(w http.ResponseWriter, r *http.Request) {
	rows, err := s.Events.DB.Pool.Query(r.Context(),
		`SELECT id,source,delivery_id,event_type,action,repository,received_at,processed_at,attempt_count
		 FROM events ORDER BY received_at DESC LIMIT 100`)
	if err != nil { writeErr(w, err); return }
	defer rows.Close()
	var out []map[string]any
	for rows.Next() {
		var id uuid.UUID
		var src, et string
		var did, action, repo *string
		var rec time.Time
		var proc *time.Time
		var attempts int
		if err := rows.Scan(&id, &src, &did, &et, &action, &repo, &rec, &proc, &attempts); err != nil {
			writeErr(w, err); return
		}
		out = append(out, map[string]any{
			"id":          id,
			"source":      src,
			"delivery_id": ptrStr(did),
			"event_type":  et,
			"action":      ptrStr(action),
			"repository":  ptrStr(repo),
			"received_at": rec,
			"processed_at": proc,
			"attempt_count": attempts,
		})
	}
	writeJSON(w, http.StatusOK, out)
}

// ----------------- helpers -----------------

func (s *Server) lookupEnv(r *http.Request, id string) (*models.Environment, error) {
	if u, err := uuid.Parse(id); err == nil {
		return s.Envs.Get(r.Context(), u)
	}
	return s.Envs.ByShortID(r.Context(), id)
}

func writeJSON(w http.ResponseWriter, code int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	_ = json.NewEncoder(w).Encode(v)
}
func writeErr(w http.ResponseWriter, err error) {
	writeJSON(w, http.StatusInternalServerError, errResp(err.Error()))
}
func errResp(msg string) map[string]string { return map[string]string{"error": msg} }
func ptrStr(s *string) string { if s == nil { return "" }; return *s }
func hasPrefix(s, p string) bool { return len(s) >= len(p) && s[:len(p)] == p }

// shortHash mirrors events.shortHash but stays local to api to avoid an import cycle.
func shortHash(s string) string {
	id := uuid.NewString()
	if len(id) < 6 { return id }
	return id[:6]
}
