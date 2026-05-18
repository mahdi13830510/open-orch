// Package lifecycle encodes the environment state machine.
// Only valid transitions are allowed; the FSM is enforced anywhere that
// changes environments.state.
package lifecycle

import (
	"fmt"

	"github.com/open-orch/backend/internal/models"
)

// transitions is the legal next-state set for each state.
var transitions = map[models.EnvState]map[models.EnvState]bool{
	models.EnvPending: {
		models.EnvResolving: true, models.EnvFailed: true, models.EnvDestroying: true,
	},
	models.EnvResolving: {
		models.EnvDeploying: true, models.EnvFailed: true, models.EnvDestroying: true,
	},
	models.EnvDeploying: {
		models.EnvHealthy: true, models.EnvDegraded: true,
		models.EnvFailed: true, models.EnvDestroying: true,
	},
	models.EnvHealthy: {
		models.EnvDegraded: true, models.EnvDeploying: true,
		models.EnvFailed: true, models.EnvDestroying: true,
	},
	models.EnvDegraded: {
		models.EnvHealthy: true, models.EnvDeploying: true,
		models.EnvFailed: true, models.EnvDestroying: true,
	},
	models.EnvFailed: {
		models.EnvDeploying: true, models.EnvResolving: true,
		models.EnvDestroying: true,
	},
	models.EnvDestroying: {
		models.EnvDestroyed: true, models.EnvFailed: true,
	},
	models.EnvDestroyed: {},
}

// CanTransition returns nil if from -> to is allowed.
func CanTransition(from, to models.EnvState) error {
	if from == to {
		return nil
	}
	if t, ok := transitions[from]; ok {
		if t[to] {
			return nil
		}
	}
	return fmt.Errorf("invalid transition: %s -> %s", from, to)
}
