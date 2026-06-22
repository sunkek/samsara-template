package model

import "time"

// User is the auth aggregate. PasswordHash never leaves the backend — it is
// excluded from JSON so it cannot leak through an API response.
type User struct {
	ID           string    `json:"id"`
	Email        string    `json:"email"`
	PasswordHash string    `json:"-"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
}

// RegisterInput / LoginInput are the use-case parameter objects. They live in
// model so the domain and the REST adapter share them without an import cycle.
type RegisterInput struct {
	Email    string
	Password string
}

type LoginInput struct {
	Email    string
	Password string
}

// Tokens is the access/refresh pair returned by login and refresh.
type Tokens struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
}

// TokenType distinguishes access tokens from refresh tokens so a refresh token
// cannot be replayed as an access token (and vice versa).
type TokenType string

const (
	AccessToken  TokenType = "access"
	RefreshToken TokenType = "refresh"
)

// Claims is the framework-free view of a verified token. The middleware stores
// it in the request context for downstream handlers.
type Claims struct {
	UserID string
	Email  string
	Type   TokenType
	// ID is the token's unique identifier (JWT jti), used to revoke a specific
	// token. ExpiresAt is when the token lapses; revocation entries only need to
	// live until then.
	ID        string
	ExpiresAt time.Time
}
