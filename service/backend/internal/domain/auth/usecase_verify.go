package auth

import (
	"context"

	"github.com/sunkek/samsara-template/backend/internal/domain/auth/model"
)

// Verify validates an access token and returns its claims. Used by the auth
// middleware on every protected request.
func (d *Domain) Verify(_ context.Context, accessToken string) (model.Claims, error) {
	return d.tok.parse(accessToken, model.AccessToken)
}
