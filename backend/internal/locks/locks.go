// Package locks implements distributed leases per environment-operation
// using Redis. The orchestrator uses these to make sure two workers don't
// concurrently mutate the same environment.
package locks

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"time"

	"github.com/redis/go-redis/v9"
)

// ErrLocked is returned when the lock could not be acquired.
var ErrLocked = errors.New("lock not acquired")

// Manager wraps a Redis client. Locks are cooperative: holders set a unique
// token and only release if the token still matches (via a Lua compare-and-del).
type Manager struct {
	Client *redis.Client
}

func New(client *redis.Client) *Manager { return &Manager{Client: client} }

// Lock represents an acquired lease.
type Lock struct {
	Key   string
	Token string
	mgr   *Manager
}

// Acquire tries to grab the lock with the given TTL. Non-blocking.
func (m *Manager) Acquire(ctx context.Context, key string, ttl time.Duration) (*Lock, error) {
	tok, err := token()
	if err != nil {
		return nil, err
	}
	ok, err := m.Client.SetNX(ctx, key, tok, ttl).Result()
	if err != nil {
		return nil, err
	}
	if !ok {
		return nil, ErrLocked
	}
	return &Lock{Key: key, Token: tok, mgr: m}, nil
}

// AcquireWait retries until the deadline or ctx is done.
func (m *Manager) AcquireWait(ctx context.Context, key string, ttl, wait time.Duration) (*Lock, error) {
	deadline := time.Now().Add(wait)
	for {
		l, err := m.Acquire(ctx, key, ttl)
		if err == nil {
			return l, nil
		}
		if !errors.Is(err, ErrLocked) {
			return nil, err
		}
		if time.Now().After(deadline) {
			return nil, ErrLocked
		}
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		case <-time.After(200 * time.Millisecond):
		}
	}
}

// Refresh extends the TTL if we still own the lock.
func (l *Lock) Refresh(ctx context.Context, ttl time.Duration) error {
	const script = `
if redis.call("get", KEYS[1]) == ARGV[1] then
  return redis.call("pexpire", KEYS[1], ARGV[2])
else
  return 0
end`
	_, err := l.mgr.Client.Eval(ctx, script, []string{l.Key},
		l.Token, ttl.Milliseconds()).Result()
	return err
}

// Release drops the lock atomically if we still own it.
func (l *Lock) Release(ctx context.Context) error {
	const script = `
if redis.call("get", KEYS[1]) == ARGV[1] then
  return redis.call("del", KEYS[1])
else
  return 0
end`
	_, err := l.mgr.Client.Eval(ctx, script, []string{l.Key}, l.Token).Result()
	return err
}

func token() (string, error) {
	var b [16]byte
	if _, err := rand.Read(b[:]); err != nil {
		return "", err
	}
	return hex.EncodeToString(b[:]), nil
}
