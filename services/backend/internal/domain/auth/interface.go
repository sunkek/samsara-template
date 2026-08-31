package auth

import (
	"context"
	"time"

	"github.com/sunkek/samsara-template/backend/internal/domain/auth/model"
)

// Service is the inbound port consumed by the REST adapter and the auth
// middleware. *Domain implements it.
type Service interface {
	Register(ctx context.Context, in model.RegisterInput) (model.User, error)
	Login(ctx context.Context, in model.LoginInput) (model.Tokens, error)
	Refresh(ctx context.Context, refreshToken string) (model.Tokens, error)
	Verify(ctx context.Context, accessToken string) (model.Claims, error)
	Logout(ctx context.Context, refreshToken string) error
}

// DB is the outbound port for user persistence.
type DB interface {
	InsertUser(ctx context.Context, u model.User) (model.User, error)
	GetUserByEmail(ctx context.Context, email string) (model.User, error)
	GetUserByID(ctx context.Context, id string) (model.User, error)
}

// Revoker is the outbound port for the refresh-token denylist. A jti is revoked
// until ttl elapses — pass the token's remaining lifetime so the entry
// self-expires. adapter/memory is a process-local implementation; wire the Redis
// adapter in production so revocation survives restarts and spans replicas.
type Revoker interface {
	// Claim atomically revokes jti and reports whether this caller was the one
	// that did it. A second caller presenting the same token gets false.
	//
	// Rotation depends on the atomicity: checking and then revoking in two steps
	// lets two concurrent uses of a stolen refresh token both pass the check and
	// both mint a valid pair, which is exactly the replay the denylist exists to
	// stop.
	Claim(ctx context.Context, jti string, ttl time.Duration) (bool, error)
	// Revoke marks jti revoked whether or not it already was. Used by logout,
	// where revoking an already-revoked token is success, not a conflict.
	Revoke(ctx context.Context, jti string, ttl time.Duration) error
	IsRevoked(ctx context.Context, jti string) (bool, error)
}
