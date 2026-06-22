package auth

import (
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"github.com/sunkek/mishap"

	"github.com/sunkek/samsara-template/backend/internal/common/e"
	"github.com/sunkek/samsara-template/backend/internal/domain/auth/model"
)

// tokenManager signs and parses HS256 JWTs. Kept unexported: token format is an
// implementation detail of the auth domain, not part of any port.
type tokenManager struct {
	secret     []byte
	accessTTL  time.Duration
	refreshTTL time.Duration
}

// jwtClaims is the on-the-wire claim set. UserID rides in the standard Subject
// field; Email and Type are private claims.
type jwtClaims struct {
	Email string          `json:"email"`
	Type  model.TokenType `json:"typ"`
	jwt.RegisteredClaims
}

// issue mints both tokens for a user in one call.
func (t tokenManager) issue(userID, email string) (model.Tokens, error) {
	access, err := t.sign(userID, email, model.AccessToken, t.accessTTL)
	if err != nil {
		return model.Tokens{}, err
	}
	refresh, err := t.sign(userID, email, model.RefreshToken, t.refreshTTL)
	if err != nil {
		return model.Tokens{}, err
	}
	return model.Tokens{AccessToken: access, RefreshToken: refresh}, nil
}

func (t tokenManager) sign(userID, email string, typ model.TokenType, ttl time.Duration) (string, error) {
	now := time.Now().UTC()
	claims := jwtClaims{
		Email: email,
		Type:  typ,
		RegisteredClaims: jwt.RegisteredClaims{
			ID:        uuid.NewString(),
			Subject:   userID,
			IssuedAt:  jwt.NewNumericDate(now),
			ExpiresAt: jwt.NewNumericDate(now.Add(ttl)),
		},
	}
	signed, err := jwt.NewWithClaims(jwt.SigningMethodHS256, claims).SignedString(t.secret)
	if err != nil {
		return "", mishap.Wrap(err, "sign token")
	}
	return signed, nil
}

// parse validates the signature and expiry, enforces the expected token type,
// and returns the framework-free claims. All failures map to e.JWT (→ 401).
func (t tokenManager) parse(tokenStr string, want model.TokenType) (model.Claims, error) {
	var claims jwtClaims
	_, err := jwt.ParseWithClaims(tokenStr, &claims, func(token *jwt.Token) (any, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, mishap.New("unexpected signing method", e.JWT)
		}
		return t.secret, nil
	})
	if err != nil {
		return model.Claims{}, mishap.New("invalid or expired token", e.JWT)
	}
	if claims.Type != want {
		return model.Claims{}, mishap.New("wrong token type", e.JWT)
	}
	out := model.Claims{
		UserID: claims.Subject,
		Email:  claims.Email,
		Type:   claims.Type,
		ID:     claims.ID,
	}
	if claims.ExpiresAt != nil {
		out.ExpiresAt = claims.ExpiresAt.Time
	}
	return out, nil
}
