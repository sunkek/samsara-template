// Package redis implements the auth domain's Revoker outbound port backed by
// the samsara Redis component. Revoked token ids are stored as keys that expire
// with the token, so the denylist self-cleans.
package redis

import (
	"context"
	"time"

	"github.com/sunkek/mishap"
	rediscmp "github.com/sunkek/samsara-components/redis"
)

// keyPrefix namespaces revocation entries so they cannot collide with other
// keys sharing the Redis database.
const keyPrefix = "auth:revoked:"

type Adapter struct {
	rdb rediscmp.Client
}

// New builds the adapter from any redis Client (the *Component satisfies it).
func New(rdb rediscmp.Client) *Adapter {
	return &Adapter{rdb: rdb}
}

// Claim is SET NX: the write succeeds only if the key is absent, so exactly one
// concurrent caller sees true. This is the whole reason refresh rotation is
// safe against a replay.
func (a *Adapter) Claim(ctx context.Context, jti string, ttl time.Duration) (bool, error) {
	won, err := a.rdb.SetNX(ctx, keyPrefix+jti, "1", ttl)
	if err != nil {
		return false, mishap.Wrap(err, "claim token")
	}
	return won, nil
}

func (a *Adapter) Revoke(ctx context.Context, jti string, ttl time.Duration) error {
	if err := a.rdb.Set(ctx, keyPrefix+jti, "1", ttl); err != nil {
		return mishap.Wrap(err, "revoke token")
	}
	return nil
}

func (a *Adapter) IsRevoked(ctx context.Context, jti string) (bool, error) {
	n, err := a.rdb.Exists(ctx, keyPrefix+jti)
	if err != nil {
		return false, mishap.Wrap(err, "check revoked token")
	}
	return n > 0, nil
}
