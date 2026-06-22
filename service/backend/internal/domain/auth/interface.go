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

// Revoker is the outbound port for the refresh-token denylist. Revoke marks a
// token id (jti) as revoked until ttl elapses (set ttl to the token's remaining
// lifetime so the entry self-expires). IsRevoked reports whether a jti has been
// revoked. The default implementation is an in-memory no-op; wire the Redis
// adapter in production so revocation survives restarts and spans replicas.
type Revoker interface {
	Revoke(ctx context.Context, jti string, ttl time.Duration) error
	IsRevoked(ctx context.Context, jti string) (bool, error)
}
