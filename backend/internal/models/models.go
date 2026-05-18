package models

import (
	"encoding/json"
	"time"

	"github.com/google/uuid"
)

// ---------------------------------------------------------------------------
// Lifecycle states for an Environment. These are the only legal values
// for `environments.state`. Transitions are enforced in lifecycle/fsm.go.
// ---------------------------------------------------------------------------
type EnvState string

const (
	EnvPending    EnvState = "pending"
	EnvResolving  EnvState = "resolving"
	EnvDeploying  EnvState = "deploying"
	EnvHealthy    EnvState = "healthy"
	EnvDegraded   EnvState = "degraded"
	EnvFailed     EnvState = "failed"
	EnvDestroying EnvState = "destroying"
	EnvDestroyed  EnvState = "destroyed"
)

// Per-service deployment states (a deployment is one service inside one env).
type DeploymentState string

const (
	DepPending  DeploymentState = "pending"
	DepBuilding DeploymentState = "building"
	DepRunning  DeploymentState = "running"
	DepFailed   DeploymentState = "failed"
	DepStopped  DeploymentState = "stopped"
)

type SelectionStrategy string

const (
	StrategyMatched           SelectionStrategy = "matched"
	StrategyFallbackMain      SelectionStrategy = "fallback_main"
	StrategyFallbackLatest    SelectionStrategy = "fallback_latest_stable"
	StrategyFallbackAutoCreate SelectionStrategy = "fallback_auto_create"
	StrategyFallbackPrevious  SelectionStrategy = "fallback_previous_compatible"
)

// ---------------------------------------------------------------------------
// Repository: a registered git repo with a backing service.
// ---------------------------------------------------------------------------
type Repository struct {
	ID            uuid.UUID       `json:"id"`
	Name          string          `json:"name"`
	FullName      string          `json:"full_name"`
	DefaultBranch string          `json:"default_branch"`
	ServiceKind   string          `json:"service_kind"`
	ExposePort    *int            `json:"expose_port,omitempty"`
	BuddyProject  string          `json:"buddy_project"`
	BuddyPipeline string          `json:"buddy_pipeline"`
	Metadata      json.RawMessage `json:"metadata,omitempty"`
	CreatedAt     time.Time       `json:"created_at"`
	UpdatedAt     time.Time       `json:"updated_at"`
}

// Feature: identity derived from a normalized branch (e.g. "billing-redesign").
type Feature struct {
	ID          uuid.UUID `json:"id"`
	Slug        string    `json:"slug"`
	DisplayName string    `json:"display_name,omitempty"`
	CreatedAt   time.Time `json:"created_at"`
	LastSeenAt  time.Time `json:"last_seen_at"`
}

// PullRequest mirrors GitHub state.
type PullRequest struct {
	ID           uuid.UUID `json:"id"`
	RepositoryID uuid.UUID `json:"repository_id"`
	Number       int       `json:"number"`
	Branch       string    `json:"branch"`
	HeadSHA      string    `json:"head_sha"`
	State        string    `json:"state"`
	Title        string    `json:"title,omitempty"`
	Author       string    `json:"author,omitempty"`
	Labels       []string  `json:"labels"`
}

// Topology snapshot stored on the environment. This is the *desired* state
// the reconciler aims to converge runtime toward.
type Topology struct {
	EnvironmentID uuid.UUID         `json:"environment_id"`
	FeatureSlug   string            `json:"feature_slug"`
	Services      []ServicePlan     `json:"services"`
	Generation    int64             `json:"generation"`
	GlobalEnv     map[string]string `json:"global_env"`
	Domains       []DomainPlan      `json:"domains"`
}

// ServicePlan: how one repo participates in this topology.
type ServicePlan struct {
	RepositoryID    uuid.UUID         `json:"repository_id"`
	RepositoryName  string            `json:"repository_name"`
	Branch          string            `json:"branch"`
	CommitSHA       string            `json:"commit_sha"`
	ImageRef        string            `json:"image_ref,omitempty"`
	ContainerName   string            `json:"container_name"`
	ExposePort      *int              `json:"expose_port,omitempty"`
	EnvVars         map[string]string `json:"env_vars"`
	Strategy        SelectionStrategy `json:"strategy"`
	HealthcheckPath string            `json:"healthcheck_path,omitempty"`
}

type DomainPlan struct {
	Hostname       string    `json:"hostname"`
	RepositoryName string    `json:"repository_name"`
	RepositoryID   uuid.UUID `json:"repository_id"`
	TargetPort     int       `json:"target_port"`
	TLS            bool      `json:"tls"`
}

// Environment: a preview environment row.
type Environment struct {
	ID               uuid.UUID       `json:"id"`
	ShortID          string          `json:"short_id"`
	FeatureID        uuid.UUID       `json:"feature_id"`
	State            EnvState        `json:"state"`
	Generation       int64           `json:"generation"`
	DesiredTopology  json.RawMessage `json:"desired_topology"`
	DockerNetwork    string          `json:"docker_network"`
	BaseDomain       string          `json:"base_domain"`
	TTLSeconds       *int            `json:"ttl_seconds,omitempty"`
	LastEventAt      time.Time       `json:"last_event_at"`
	LastReconciledAt *time.Time      `json:"last_reconciled_at,omitempty"`
	Suspended        bool            `json:"suspended"`
	CreatedAt        time.Time       `json:"created_at"`
	UpdatedAt        time.Time       `json:"updated_at"`
	DestroyedAt      *time.Time      `json:"destroyed_at,omitempty"`
}

// Deployment: one service inside one env at one generation.
type Deployment struct {
	ID                uuid.UUID         `json:"id"`
	EnvironmentID     uuid.UUID         `json:"environment_id"`
	RepositoryID      uuid.UUID         `json:"repository_id"`
	Generation        int64             `json:"generation"`
	Branch            string            `json:"branch"`
	CommitSHA         string            `json:"commit_sha"`
	ImageRef          string            `json:"image_ref,omitempty"`
	State             DeploymentState   `json:"state"`
	ContainerName     string            `json:"container_name"`
	ContainerID       string            `json:"container_id,omitempty"`
	EnvVars           map[string]string `json:"env_vars"`
	SelectionStrategy SelectionStrategy `json:"selection_strategy"`
	BuddyRunID        string            `json:"buddy_run_id,omitempty"`
	LastStatusAt      time.Time         `json:"last_status_at"`
	CreatedAt         time.Time         `json:"created_at"`
}

type Event struct {
	ID           uuid.UUID       `json:"id"`
	Source       string          `json:"source"`
	DeliveryID   string          `json:"delivery_id,omitempty"`
	EventType    string          `json:"event_type"`
	Action       string          `json:"action,omitempty"`
	Repository   string          `json:"repository,omitempty"`
	Payload      json.RawMessage `json:"payload"`
	ReceivedAt   time.Time       `json:"received_at"`
	ProcessedAt  *time.Time      `json:"processed_at,omitempty"`
	ProcessError string          `json:"process_error,omitempty"`
	AttemptCount int             `json:"attempt_count"`
}

type RuntimeResource struct {
	ID            uuid.UUID `json:"id"`
	EnvironmentID uuid.UUID `json:"environment_id"`
	Kind          string    `json:"kind"`
	ExternalID    string    `json:"external_id,omitempty"`
	Name          string    `json:"name"`
	State         string    `json:"state"`
	LastSeenAt    *time.Time `json:"last_seen_at,omitempty"`
}

type Domain struct {
	ID            uuid.UUID `json:"id"`
	EnvironmentID uuid.UUID `json:"environment_id"`
	RepositoryID  uuid.UUID `json:"repository_id"`
	Hostname      string    `json:"hostname"`
	TargetPort    int       `json:"target_port"`
	TLS           bool      `json:"tls"`
}
