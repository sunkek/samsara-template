package auth

import (
	"context"
	"strings"

	"github.com/sunkek/mishap"
	"golang.org/x/crypto/bcrypt"

	"github.com/sunkek/samsara-template/backend/internal/common/e"
	"github.com/sunkek/samsara-template/backend/internal/domain/auth/model"
)

func (d *Domain) Login(ctx context.Context, in model.LoginInput) (model.Tokens, error) {
	email := strings.TrimSpace(strings.ToLower(in.Email))
	u, err := d.db.GetUserByEmail(ctx, email)
	if err != nil {
		// Return a uniform credentials error regardless of whether the user
		// exists, so the endpoint does not leak which emails are registered.
		return model.Tokens{}, mishap.New("invalid credentials", e.Forbidden)
	}
	if bcrypt.CompareHashAndPassword([]byte(u.PasswordHash), []byte(in.Password)) != nil {
		return model.Tokens{}, mishap.New("invalid credentials", e.Forbidden)
	}
	return d.tok.issue(u.ID, u.Email)
}
