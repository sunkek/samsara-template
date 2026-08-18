// Package memory implements the auth domain's Revoker outbound port with an
// in-process map. It is the stand-in used when the project is built without
// Redis: revocation works, but only within one process and only until restart.
// Wire the Redis adapter instead once you run more than one replica.
package memory

import (
	"context"
	"sync"
	"time"
)

// Adapter is a process-local refresh-token denylist. Entries carry the token's
// own expiry, so the map self-cleans on read and on a periodic sweep.
type Adapter struct {
	mu      sync.Mutex
	revoked map[string]time.Time // jti -> expiry
	now     func() time.Time
}

// New builds an empty denylist.
func New() *Adapter {
	return &Adapter{revoked: make(map[string]time.Time), now: time.Now}
}

func (a *Adapter) Revoke(_ context.Context, jti string, ttl time.Duration) error {
	a.mu.Lock()
	defer a.mu.Unlock()
	a.sweep()
	a.revoked[jti] = a.now().Add(ttl)
	return nil
}

func (a *Adapter) IsRevoked(_ context.Context, jti string) (bool, error) {
	a.mu.Lock()
	defer a.mu.Unlock()
	exp, ok := a.revoked[jti]
	if !ok {
		return false, nil
	}
	if !a.now().Before(exp) {
		delete(a.revoked, jti)
		return false, nil
	}
	return true, nil
}

// sweep drops expired entries so the map cannot grow without bound. Callers
// hold a.mu.
func (a *Adapter) sweep() {
	now := a.now()
	for jti, exp := range a.revoked {
		if !now.Before(exp) {
			delete(a.revoked, jti)
		}
	}
}
