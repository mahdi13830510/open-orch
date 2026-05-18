// Package reconciler also contains the cleanup worker: TTL enforcement plus
// automatic suspension/destruction of stale environments.
package reconciler

import (
	"context"
	"time"

	"github.com/open-orch/backend/internal/config"
	"github.com/open-orch/backend/internal/db"
	"github.com/open-orch/backend/internal/models"
	"github.com/rs/zerolog"
)

type Cleaner struct {
	Cfg  *config.Config
	Log  zerolog.Logger
	Envs *db.EnvironmentStore
}

func (c *Cleaner) Run(ctx context.Context) {
	t := time.NewTicker(time.Minute)
	defer t.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-t.C:
			c.tick(ctx)
		}
	}
}

func (c *Cleaner) tick(ctx context.Context) {
	cutoff := time.Now().Add(-c.Cfg.DefaultEnvTTL)
	idle, err := c.Envs.IdleSince(ctx, cutoff)
	if err != nil {
		c.Log.Error().Err(err).Msg("idle scan")
		return
	}
	for _, e := range idle {
		// Move to destroying; the reconciler will perform actual teardown.
		c.Log.Info().Str("env", e.ShortID).Msg("TTL expired, marking destroying")
		_ = c.Envs.UpdateState(ctx, e.ID, models.EnvDestroying)
	}
}
