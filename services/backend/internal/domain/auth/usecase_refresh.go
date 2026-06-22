package auth

import (
	"context"
	"time"

	"github.com/sunkek/mishap"

	"github.com/sunkek/samsara-template/backend/internal/common/e"
	"github.com/sunkek/samsara-template/backend/internal/domain/auth/model"
)

// Refresh exchanges a valid refresh token for a fresh access/refresh pair.
// The user is re-read so a deleted account cannot keep minting tokens. The
// presented refresh token is rotated: it is added to the denylist so it cannot
// be replayed, and a previously revoked token is rejected.
func (d *Domain) Refresh(ctx context.Context, refreshToken string) (model.Tokens, error) {
	claims, err := d.tok.parse(refreshToken, model.RefreshToken)
	if err != nil {
		return model.Tokens{}, err
	}
	revoked, err := d.revoker.IsRevoked(ctx, claims.ID)
	if err != nil {
		return model.Tokens{}, mishap.Wrap(err, "check token revocation")
	}
	if revoked {
		return model.Tokens{}, mishap.New("refresh token revoked", e.JWT)
	}
	u, err := d.db.GetUserByID(ctx, claims.UserID)
	if err != nil {
		return model.Tokens{}, mishap.New("invalid refresh token", e.JWT)
	}
	// Rotate: revoke the presented token so it is single-use.
	if ttl := time.Until(claims.ExpiresAt); ttl > 0 {
		if err := d.revoker.Revoke(ctx, claims.ID, ttl); err != nil {
			return model.Tokens{}, mishap.Wrap(err, "revoke rotated token")
		}
	}
	return d.tok.issue(u.ID, u.Email)
}
