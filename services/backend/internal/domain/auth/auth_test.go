package auth

import (
	"context"
	"testing"
	"time"

	"golang.org/x/crypto/bcrypt"

	"github.com/sunkek/mishap"

	"github.com/sunkek/samsara-template/backend/internal/common/e"
	"github.com/sunkek/samsara-template/backend/internal/domain/auth/model"
)

// stubDB is an in-memory auth.DB. byEmail/byID are looked up by the use cases;
// a nil entry makes the lookup return a not-found error.
type stubDB struct {
	byEmail   map[string]model.User
	byID      map[string]model.User
	insertErr error
}

func (s *stubDB) InsertUser(_ context.Context, u model.User) (model.User, error) {
	if s.insertErr != nil {
		return model.User{}, s.insertErr
	}
	return u, nil
}
func (s *stubDB) GetUserByEmail(_ context.Context, email string) (model.User, error) {
	if u, ok := s.byEmail[email]; ok {
		return u, nil
	}
	return model.User{}, mishap.New("user not found", e.NotFound)
}
func (s *stubDB) GetUserByID(_ context.Context, id string) (model.User, error) {
	if u, ok := s.byID[id]; ok {
		return u, nil
	}
	return model.User{}, mishap.New("user not found", e.NotFound)
}

// stubRevoker is an in-memory Revoker for tests.
type stubRevoker struct{ revoked map[string]bool }

func (s *stubRevoker) Revoke(_ context.Context, jti string, _ time.Duration) error {
	if s.revoked == nil {
		s.revoked = map[string]bool{}
	}
	s.revoked[jti] = true
	return nil
}
func (s *stubRevoker) IsRevoked(_ context.Context, jti string) (bool, error) {
	return s.revoked[jti], nil
}

func codeOf(err error) mishap.Code {
	if m, ok := mishap.As(err); ok {
		return m.Code()
	}
	return ""
}

func newDomain(db DB) *Domain {
	return New(db, &stubRevoker{}, "test-secret", 15*time.Minute, time.Hour)
}

// TestTokenRoundTrip exercises the unexported tokenManager directly: a freshly
// issued access token parses back to its claims, but only as the type it was
// minted for, and only before it expires.
func TestTokenRoundTrip(t *testing.T) {
	tm := tokenManager{secret: []byte("k"), accessTTL: time.Minute, refreshTTL: time.Hour}

	toks, err := tm.issue("user-1", "a@b.com")
	if err != nil {
		t.Fatalf("issue: %v", err)
	}

	claims, err := tm.parse(toks.AccessToken, model.AccessToken)
	if err != nil {
		t.Fatalf("parse access: %v", err)
	}
	if claims.UserID != "user-1" || claims.Email != "a@b.com" {
		t.Errorf("unexpected claims: %+v", claims)
	}

	// An access token must not be accepted where a refresh token is required.
	if _, err := tm.parse(toks.AccessToken, model.RefreshToken); codeOf(err) != e.JWT {
		t.Errorf("want e.JWT for wrong token type, got %v", err)
	}

	// A token signed with a different secret must fail verification.
	other := tokenManager{secret: []byte("other"), accessTTL: time.Minute}
	bad, _ := other.sign("user-1", "a@b.com", model.AccessToken, time.Minute)
	if _, err := tm.parse(bad, model.AccessToken); codeOf(err) != e.JWT {
		t.Errorf("want e.JWT for foreign signature, got %v", err)
	}

	// An expired token must fail.
	expired, _ := tm.sign("user-1", "a@b.com", model.AccessToken, -time.Minute)
	if _, err := tm.parse(expired, model.AccessToken); codeOf(err) != e.JWT {
		t.Errorf("want e.JWT for expired token, got %v", err)
	}
}

func TestRegisterValidation(t *testing.T) {
	tests := []struct {
		name    string
		in      model.RegisterInput
		wantErr mishap.Code
	}{
		{"empty email", model.RegisterInput{Email: "  ", Password: "longenough"}, e.Validation},
		{"short password", model.RegisterInput{Email: "a@b.com", Password: "short"}, e.Validation},
		{"ok", model.RegisterInput{Email: "A@B.com", Password: "longenough"}, ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			u, err := newDomain(&stubDB{}).Register(context.Background(), tt.in)
			if tt.wantErr != "" {
				if codeOf(err) != tt.wantErr {
					t.Fatalf("want code %q, got %v", tt.wantErr, err)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if u.Email != "a@b.com" {
				t.Errorf("email = %q, want normalized lowercase", u.Email)
			}
			if u.PasswordHash == tt.in.Password || u.PasswordHash == "" {
				t.Error("password must be hashed, not stored in clear")
			}
		})
	}
}

func TestRefresh(t *testing.T) {
	db := &stubDB{byID: map[string]model.User{
		"u1": {ID: "u1", Email: "a@b.com"},
	}}
	d := newDomain(db)

	t.Run("access token rejected as refresh", func(t *testing.T) {
		toks, _ := d.tok.issue("u1", "a@b.com")
		if _, err := d.Refresh(context.Background(), toks.AccessToken); codeOf(err) != e.JWT {
			t.Fatalf("want e.JWT for access-as-refresh, got %v", err)
		}
	})
	t.Run("deleted user cannot mint tokens", func(t *testing.T) {
		// Valid refresh token, but the account is gone from the DB.
		ghost, _ := d.tok.issue("ghost", "ghost@b.com")
		if _, err := d.Refresh(context.Background(), ghost.RefreshToken); codeOf(err) != e.JWT {
			t.Fatalf("want e.JWT when user no longer exists, got %v", err)
		}
	})
	t.Run("success", func(t *testing.T) {
		toks, _ := d.tok.issue("u1", "a@b.com")
		got, err := d.Refresh(context.Background(), toks.RefreshToken)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if got.AccessToken == "" || got.RefreshToken == "" {
			t.Error("expected a non-empty token pair")
		}
	})
}

func TestLogoutRevokesRefresh(t *testing.T) {
	db := &stubDB{byID: map[string]model.User{"u1": {ID: "u1", Email: "a@b.com"}}}
	d := New(db, &stubRevoker{}, "test-secret", 15*time.Minute, time.Hour)

	toks, _ := d.tok.issue("u1", "a@b.com")
	if err := d.Logout(context.Background(), toks.RefreshToken); err != nil {
		t.Fatalf("logout: %v", err)
	}
	// A revoked refresh token must no longer mint new tokens.
	if _, err := d.Refresh(context.Background(), toks.RefreshToken); codeOf(err) != e.JWT {
		t.Fatalf("want e.JWT for revoked refresh, got %v", err)
	}
}

func TestRefreshRotatesToken(t *testing.T) {
	db := &stubDB{byID: map[string]model.User{"u1": {ID: "u1", Email: "a@b.com"}}}
	d := New(db, &stubRevoker{}, "test-secret", 15*time.Minute, time.Hour)

	toks, _ := d.tok.issue("u1", "a@b.com")
	if _, err := d.Refresh(context.Background(), toks.RefreshToken); err != nil {
		t.Fatalf("first refresh: %v", err)
	}
	// Reusing the same refresh token must fail: it was rotated (revoked) on use.
	if _, err := d.Refresh(context.Background(), toks.RefreshToken); codeOf(err) != e.JWT {
		t.Fatalf("want e.JWT for reused refresh, got %v", err)
	}
}

func TestLogin(t *testing.T) {
	hash, _ := bcrypt.GenerateFromPassword([]byte("correct-horse"), bcrypt.DefaultCost)
	db := &stubDB{byEmail: map[string]model.User{
		"a@b.com": {ID: "u1", Email: "a@b.com", PasswordHash: string(hash)},
	}}

	t.Run("unknown user", func(t *testing.T) {
		_, err := newDomain(db).Login(context.Background(), model.LoginInput{Email: "nobody@b.com", Password: "x"})
		if codeOf(err) != e.Forbidden {
			t.Fatalf("want e.Forbidden, got %v", err)
		}
	})
	t.Run("wrong password", func(t *testing.T) {
		_, err := newDomain(db).Login(context.Background(), model.LoginInput{Email: "a@b.com", Password: "nope"})
		if codeOf(err) != e.Forbidden {
			t.Fatalf("want e.Forbidden, got %v", err)
		}
	})
	t.Run("success", func(t *testing.T) {
		toks, err := newDomain(db).Login(context.Background(), model.LoginInput{Email: "A@B.com", Password: "correct-horse"})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if toks.AccessToken == "" || toks.RefreshToken == "" {
			t.Error("expected a non-empty token pair")
		}
	})
}
