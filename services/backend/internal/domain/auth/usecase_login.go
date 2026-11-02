package auth

import (
	"context"
	"strings"
	"sync"

	"github.com/sunkek/mishap"
	"golang.org/x/crypto/bcrypt"

	"github.com/sunkek/samsara-template/backend/internal/common/e"
	"github.com/sunkek/samsara-template/backend/internal/domain/auth/model"
)

// dummyHash is compared against when the email is unknown, so that path costs
// the same bcrypt work as a wrong password. Without it, an unknown email
// returns in microseconds and a known one in tens of milliseconds, and the
// response time alone tells a caller which emails are registered. It is made
// lazily, at the cost Register uses, so the two paths stay matched if that
// cost changes.
var dummyHash = sync.OnceValue(func() []byte {
	h, err := bcrypt.GenerateFromPassword([]byte("login-timing-dummy"), bcrypt.DefaultCost)
	if err != nil {
		panic(err) // only fails on a password over 72 bytes, which this is not
	}
	return h
})

func (d *Domain) Login(ctx context.Context, in model.LoginInput) (model.Tokens, error) {
	email := strings.TrimSpace(strings.ToLower(in.Email))
	u, err := d.db.GetUserByEmail(ctx, email)
	if err != nil {
		// Return a uniform credentials error regardless of whether the user
		// exists, and spend the same bcrypt work as a wrong password, so
		// neither the response nor its timing reveals which emails are
		// registered.
		_ = bcrypt.CompareHashAndPassword(dummyHash(), []byte(in.Password))
		return model.Tokens{}, mishap.New("invalid credentials", e.Unauthorized)
	}
	if bcrypt.CompareHashAndPassword([]byte(u.PasswordHash), []byte(in.Password)) != nil {
		return model.Tokens{}, mishap.New("invalid credentials", e.Unauthorized)
	}
	return d.tok.issue(u.ID, u.Email)
}
