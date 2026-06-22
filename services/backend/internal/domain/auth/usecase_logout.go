package auth

import (
	"context"
	"time"

	"github.com/sunkek/samsara-template/backend/internal/domain/auth/model"
)

// Logout revokes a refresh token so it can no longer be exchanged for new
// tokens. The token's jti is added to the denylist until its natural expiry, so
// the entry self-cleans. Access tokens are short-lived and are not revoked
// here; they lapse on their own TTL.
func (d *Domain) Logout(ctx context.Context, refreshToken string) error {
	claims, err := d.tok.parse(refreshToken, model.RefreshToken)
	if err != nil {
		return err
	}
	ttl := time.Until(claims.ExpiresAt)
	if ttl <= 0 {
		// Already expired — nothing to revoke.
		return nil
	}
	return d.revoker.Revoke(ctx, claims.ID, ttl)
}
