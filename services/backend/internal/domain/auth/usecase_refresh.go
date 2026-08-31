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
	// Rotate first, and atomically: Claim both checks the denylist and adds to
	// it in one operation, so of two concurrent presentations of the same token
	// exactly one proceeds. Doing this before the user lookup also means a
	// replay cannot be timed against a slow database read.
	ttl := time.Until(claims.ExpiresAt)
	if ttl <= 0 {
		return model.Tokens{}, mishap.New("refresh token expired", e.JWT)
	}
	claimed, err := d.revoker.Claim(ctx, claims.ID, ttl)
	if err != nil {
		return model.Tokens{}, mishap.Wrap(err, "claim refresh token")
	}
	if !claimed {
		return model.Tokens{}, mishap.New("refresh token already used or revoked", e.JWT)
	}
	u, err := d.db.GetUserByID(ctx, claims.UserID)
	if err != nil {
		return model.Tokens{}, mishap.New("invalid refresh token", e.JWT)
	}
	return d.tok.issue(u.ID, u.Email)
}
