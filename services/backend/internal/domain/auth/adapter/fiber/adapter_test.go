package fiber

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	gf "github.com/gofiber/fiber/v3"
	"github.com/sunkek/mishap"

	"github.com/sunkek/samsara-template/backend/internal/common/e"
	"github.com/sunkek/samsara-template/backend/internal/domain/auth/model"
)

// handlerStub is an auth.Service whose every method returns what the test
// configures, and records what it was called with. middleware_test.go has its
// own stub shaped for token verification; this one is shaped for the handlers.
type handlerStub struct {
	user   model.User
	tokens model.Tokens
	err    error

	gotRegister model.RegisterInput
	gotLogin    model.LoginInput
	gotRefresh  string
	gotLogout   string
	calls       int
}

func (s *handlerStub) Register(_ context.Context, in model.RegisterInput) (model.User, error) {
	s.calls++
	s.gotRegister = in
	return s.user, s.err
}

func (s *handlerStub) Login(_ context.Context, in model.LoginInput) (model.Tokens, error) {
	s.calls++
	s.gotLogin = in
	return s.tokens, s.err
}

func (s *handlerStub) Refresh(_ context.Context, token string) (model.Tokens, error) {
	s.calls++
	s.gotRefresh = token
	return s.tokens, s.err
}

func (s *handlerStub) Logout(_ context.Context, token string) error {
	s.calls++
	s.gotLogout = token
	return s.err
}

func (s *handlerStub) Verify(context.Context, string) (model.Claims, error) {
	return model.Claims{}, nil
}

// appWithAuthRoutes mounts the adapter's real route table, so a wrong verb or
// path in the adapter fails these tests rather than passing quietly.
func appWithAuthRoutes(t *testing.T, svc *handlerStub) *gf.App {
	t.Helper()
	app := gf.New(gf.Config{
		ErrorHandler: func(c gf.Ctx, err error) error { return c.SendStatus(e.HTTPStatus(err)) },
	})
	(&Adapter{svc: svc}).routes(app)
	return app
}

func post(t *testing.T, app *gf.App, path, body string) *http.Response {
	t.Helper()
	var r io.Reader
	if body != "" {
		r = strings.NewReader(body)
	}
	req := httptest.NewRequest(http.MethodPost, path, r)
	if body != "" {
		req.Header.Set("Content-Type", "application/json")
	}
	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("app.Test: %v", err)
	}
	t.Cleanup(func() { _ = resp.Body.Close() })
	return resp
}

func TestRegisterCreatesAndReturns201(t *testing.T) {
	svc := &handlerStub{user: model.User{ID: "u1", Email: "a@example.com", PasswordHash: "secret-hash"}}
	resp := post(t, appWithAuthRoutes(t, svc), "/auth/register", `{"email":"a@example.com","password":"pw"}`)

	if resp.StatusCode != http.StatusCreated {
		t.Fatalf("status = %d, want 201", resp.StatusCode)
	}
	if svc.gotRegister.Email != "a@example.com" || svc.gotRegister.Password != "pw" {
		t.Errorf("service got %+v, want the posted credentials", svc.gotRegister)
	}

	// PasswordHash carries `json:"-"`. Asserting on the wire bytes is the only
	// way to catch a future field that forgets it.
	body, _ := io.ReadAll(resp.Body)
	if strings.Contains(string(body), "secret-hash") {
		t.Errorf("response leaked the password hash: %s", body)
	}
	var got model.User
	if err := json.Unmarshal(body, &got); err != nil {
		t.Fatalf("decode body: %v", err)
	}
	if got.Email != "a@example.com" {
		t.Errorf("body email = %q, want %q", got.Email, "a@example.com")
	}
}

func TestLoginReturnsTokens(t *testing.T) {
	svc := &handlerStub{tokens: model.Tokens{AccessToken: "acc", RefreshToken: "ref"}}
	resp := post(t, appWithAuthRoutes(t, svc), "/auth/login", `{"email":"a@example.com","password":"pw"}`)

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status = %d, want 200", resp.StatusCode)
	}
	if svc.gotLogin.Email != "a@example.com" {
		t.Errorf("service got email %q, want %q", svc.gotLogin.Email, "a@example.com")
	}
	var got model.Tokens
	if err := json.NewDecoder(resp.Body).Decode(&got); err != nil {
		t.Fatalf("decode body: %v", err)
	}
	if got.AccessToken != "acc" || got.RefreshToken != "ref" {
		t.Errorf("body = %+v, want both tokens", got)
	}
}

func TestRefreshPassesTheTokenThrough(t *testing.T) {
	svc := &handlerStub{tokens: model.Tokens{AccessToken: "acc2", RefreshToken: "ref2"}}
	resp := post(t, appWithAuthRoutes(t, svc), "/auth/refresh", `{"refresh_token":"ref1"}`)

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status = %d, want 200", resp.StatusCode)
	}
	if svc.gotRefresh != "ref1" {
		t.Errorf("service got refresh token %q, want %q", svc.gotRefresh, "ref1")
	}
}

func TestLogoutReturns204(t *testing.T) {
	svc := &handlerStub{}
	resp := post(t, appWithAuthRoutes(t, svc), "/auth/logout", `{"refresh_token":"ref1"}`)

	if resp.StatusCode != http.StatusNoContent {
		t.Fatalf("status = %d, want 204", resp.StatusCode)
	}
	if svc.gotLogout != "ref1" {
		t.Errorf("service got %q, want %q", svc.gotLogout, "ref1")
	}
}

// The handlers return the domain's error unchanged and let the Fiber error
// handler map the code — so the status a caller sees is decided by e.HTTPStatus,
// not by each handler having its own opinion.
func TestServiceErrorsMapToStatus(t *testing.T) {
	cases := []struct {
		name string
		path string
		body string
		err  error
		want int
	}{
		{"register conflict", "/auth/register", `{"email":"a@example.com","password":"pw"}`, mishap.New("exists", e.Conflict), http.StatusConflict},
		{"login unauthorized", "/auth/login", `{"email":"a@example.com","password":"pw"}`, mishap.New("bad creds", e.Unauthorized), http.StatusUnauthorized},
		{"refresh jwt", "/auth/refresh", `{"refresh_token":"x"}`, mishap.New("bad token", e.JWT), http.StatusUnauthorized},
		{"logout not found", "/auth/logout", `{"refresh_token":"x"}`, mishap.New("unknown", e.NotFound), http.StatusNotFound},
		{"unmapped code is a 500", "/auth/login", `{"email":"a@example.com","password":"pw"}`, mishap.New("boom", e.Internal), http.StatusInternalServerError},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			svc := &handlerStub{err: tc.err}
			resp := post(t, appWithAuthRoutes(t, svc), tc.path, tc.body)
			if resp.StatusCode != tc.want {
				t.Errorf("status = %d, want %d", resp.StatusCode, tc.want)
			}
		})
	}
}

// A malformed body must fail as a 400 before the service is reached: the
// handlers wrap the bind error with e.Validation for exactly that.
func TestMalformedBodyIsRejectedBeforeTheService(t *testing.T) {
	for _, path := range []string{"/auth/register", "/auth/login", "/auth/refresh", "/auth/logout"} {
		t.Run(path, func(t *testing.T) {
			svc := &handlerStub{}
			resp := post(t, appWithAuthRoutes(t, svc), path, `{"email":`)
			if resp.StatusCode != http.StatusBadRequest {
				t.Errorf("status = %d, want 400", resp.StatusCode)
			}
			if svc.calls != 0 {
				t.Errorf("service was called %d times on a malformed body, want 0", svc.calls)
			}
		})
	}
}
