package fiber

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	gf "github.com/gofiber/fiber/v3"
	"github.com/sunkek/mishap"

	"github.com/sunkek/samsara-template/backend/internal/common/e"
	"github.com/sunkek/samsara-template/backend/internal/domain/auth/model"
)

// stubService is an auth.Service that accepts exactly one access token.
type stubService struct {
	validToken string
	claims     model.Claims
	verifyCall int
}

func (s *stubService) Register(context.Context, model.RegisterInput) (model.User, error) {
	return model.User{}, nil
}
func (s *stubService) Login(context.Context, model.LoginInput) (model.Tokens, error) {
	return model.Tokens{}, nil
}
func (s *stubService) Refresh(context.Context, string) (model.Tokens, error) {
	return model.Tokens{}, nil
}
func (s *stubService) Logout(context.Context, string) error { return nil }
func (s *stubService) Verify(_ context.Context, token string) (model.Claims, error) {
	s.verifyCall++
	if token != s.validToken {
		return model.Claims{}, mishap.New("invalid token", e.JWT)
	}
	return s.claims, nil
}

// appWithMiddleware builds a Fiber app guarded by the auth middleware, with one
// catch-all route that reports whether the request got through.
func appWithMiddleware(t *testing.T, svc *stubService, publicPrefixes ...string) *gf.App {
	t.Helper()
	app := gf.New(gf.Config{
		ErrorHandler: func(c gf.Ctx, err error) error {
			return c.SendStatus(e.HTTPStatus(err))
		},
	})
	app.Use((&Adapter{svc: svc}).Middleware(publicPrefixes...))
	app.All("/*", func(c gf.Ctx) error { return c.SendStatus(http.StatusOK) })
	return app
}

func do(t *testing.T, app *gf.App, path, authHeader string) *http.Response {
	t.Helper()
	req := httptest.NewRequest(http.MethodGet, path, nil)
	if authHeader != "" {
		req.Header.Set("Authorization", authHeader)
	}
	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("app.Test(%s): %v", path, err)
	}
	return resp
}

// A public prefix must unguard that path and its children, and nothing else.
// The sibling case is the one that matters: "/api/v1/authz" shares the string
// prefix "/api/v1/auth" but is a different route, and must stay protected.
func TestMiddlewarePublicPrefixMatchesOnPathBoundary(t *testing.T) {
	cases := []struct {
		name string
		path string
		want int
	}{
		{"the prefix itself", "/api/v1/auth", http.StatusOK},
		{"a child of the prefix", "/api/v1/auth/login", http.StatusOK},
		{"a sibling sharing the string prefix", "/api/v1/authz", http.StatusUnauthorized},
		{"a sibling with a longer suffix", "/api/v1/authorize", http.StatusUnauthorized},
		{"an unrelated route", "/api/v1/notes", http.StatusUnauthorized},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			app := appWithMiddleware(t, &stubService{}, "/api/v1/auth")
			if got := do(t, app, tc.path, "").StatusCode; got != tc.want {
				t.Errorf("GET %s = %d, want %d", tc.path, got, tc.want)
			}
		})
	}
}

// On a protected route the middleware must demand a Bearer token, reject
// anything else without consulting the service, and pass a valid one through.
func TestMiddlewareRequiresBearerToken(t *testing.T) {
	cases := []struct {
		name       string
		header     string
		want       int
		wantVerify int
	}{
		{"no header", "", http.StatusUnauthorized, 0},
		{"empty bearer", "Bearer ", http.StatusUnauthorized, 0},
		{"wrong scheme", "Basic dXNlcjpwYXNz", http.StatusUnauthorized, 0},
		{"lowercase scheme", "bearer good-token", http.StatusUnauthorized, 0},
		{"rejected token", "Bearer bad-token", http.StatusUnauthorized, 1},
		{"accepted token", "Bearer good-token", http.StatusOK, 1},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			svc := &stubService{validToken: "good-token"}
			app := appWithMiddleware(t, svc, "/api/v1/auth")

			if got := do(t, app, "/api/v1/notes", tc.header).StatusCode; got != tc.want {
				t.Errorf("status = %d, want %d", got, tc.want)
			}
			if svc.verifyCall != tc.wantVerify {
				t.Errorf("Verify called %d times, want %d", svc.verifyCall, tc.wantVerify)
			}
		})
	}
}

// A handler behind the middleware must be able to read the verified claims, and
// a handler on a public route must be told there are none.
func TestClaimsReachTheHandlerOnlyWhenAuthenticated(t *testing.T) {
	svc := &stubService{
		validToken: "good-token",
		claims:     model.Claims{UserID: "u-1", Email: "user@example.com"},
	}
	app := gf.New(gf.Config{
		ErrorHandler: func(c gf.Ctx, err error) error { return c.SendStatus(e.HTTPStatus(err)) },
	})
	app.Use((&Adapter{svc: svc}).Middleware("/api/v1/auth"))
	app.All("/*", func(c gf.Ctx) error {
		claims, ok := ClaimsFromContext(c)
		if !ok {
			return c.SendString("anonymous")
		}
		return c.SendString(claims.UserID)
	})

	t.Run("protected route sees the user", func(t *testing.T) {
		resp := do(t, app, "/api/v1/notes", "Bearer good-token")
		if body := readBody(t, resp); body != "u-1" {
			t.Errorf("handler saw %q, want the authenticated user id", body)
		}
	})

	t.Run("public route sees nobody", func(t *testing.T) {
		resp := do(t, app, "/api/v1/auth/login", "")
		if body := readBody(t, resp); body != "anonymous" {
			t.Errorf("handler saw %q, want no claims on a public route", body)
		}
	})
}

func readBody(t *testing.T, resp *http.Response) string {
	t.Helper()
	defer resp.Body.Close()
	b, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatalf("read body: %v", err)
	}
	return string(b)
}
