package auth

import (
	"context"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/sunkek/mishap"
	"golang.org/x/crypto/bcrypt"

	"github.com/sunkek/samsara-template/backend/internal/common/e"
	"github.com/sunkek/samsara-template/backend/internal/domain/auth/model"
)

// minPasswordLen is a floor, not a policy — tune to your security needs.
const minPasswordLen = 8

func (d *Domain) Register(ctx context.Context, in model.RegisterInput) (model.User, error) {
	email := strings.TrimSpace(strings.ToLower(in.Email))
	if email == "" {
		return model.User{}, mishap.New("email is required", e.Validation)
	}
	if len(in.Password) < minPasswordLen {
		return model.User{}, mishap.New("password too short", e.Validation)
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(in.Password), bcrypt.DefaultCost)
	if err != nil {
		return model.User{}, mishap.Wrap(err, "hash password")
	}

	now := time.Now().UTC()
	u := model.User{
		ID:           uuid.NewString(),
		Email:        email,
		PasswordHash: string(hash),
		CreatedAt:    now,
		UpdatedAt:    now,
	}
	return d.db.InsertUser(ctx, u)
}
