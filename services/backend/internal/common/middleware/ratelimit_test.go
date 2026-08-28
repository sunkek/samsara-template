package middleware

import (
	"net/http"
	"net/http/httptest"
	"strconv"
	"testing"
	"time"

	gf "github.com/gofiber/fiber/v3"

	"github.com/sunkek/samsara-template/backend/internal/common/e"
)

// limitedApp builds an app whose single route sits behind the limiter. The key
// is taken from a header so a test can act as several clients without faking
// network addresses.
func limitedApp(t *testing.T, cfg RateLimitConfig) *gf.App {
	t.Helper()
	if cfg.KeyFunc == nil {
		cfg.KeyFunc = func(c gf.Ctx) string { return c.Get("X-Client") }
	}
	app := gf.New(gf.Config{
		ErrorHandler: func(c gf.Ctx, err error) error { return c.SendStatus(e.HTTPStatus(err)) },
	})
	app.Use(RateLimit(cfg))
	app.Get("/", func(c gf.Ctx) error { return c.SendStatus(http.StatusOK) })
	return app
}

func request(t *testing.T, app *gf.App, client string) *http.Response {
	t.Helper()
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("X-Client", client)
	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("app.Test: %v", err)
	}
	return resp
}

// The Max-th request is still allowed; the one after it is refused with 429.
func TestRateLimitAllowsUpToMaxThenRefuses(t *testing.T) {
	app := limitedApp(t, RateLimitConfig{Max: 3, Window: time.Minute})

	for i := 1; i <= 3; i++ {
		if got := request(t, app, "alice").StatusCode; got != http.StatusOK {
			t.Fatalf("request %d = %d, want 200 (within the limit)", i, got)
		}
	}
	if got := request(t, app, "alice").StatusCode; got != http.StatusTooManyRequests {
		t.Errorf("request 4 = %d, want 429", got)
	}
}

// The limit is per client key: one client exhausting its budget must not
// refuse another.
func TestRateLimitIsPerClient(t *testing.T) {
	app := limitedApp(t, RateLimitConfig{Max: 1, Window: time.Minute})

	request(t, app, "alice")
	if got := request(t, app, "alice").StatusCode; got != http.StatusTooManyRequests {
		t.Fatalf("alice's second request = %d, want 429", got)
	}
	if got := request(t, app, "bob").StatusCode; got != http.StatusOK {
		t.Errorf("bob's first request = %d, want 200 — the limit is per client", got)
	}
}

// A refusal must tell the client when to come back.
func TestRateLimitSetsRetryAfter(t *testing.T) {
	app := limitedApp(t, RateLimitConfig{Max: 1, Window: 30 * time.Second})

	request(t, app, "alice")
	resp := request(t, app, "alice")

	raw := resp.Header.Get("Retry-After")
	if raw == "" {
		t.Fatal("no Retry-After header on a 429")
	}
	secs, err := strconv.Atoi(raw)
	if err != nil {
		t.Fatalf("Retry-After = %q, want an integer number of seconds", raw)
	}
	if secs < 1 || secs > 31 {
		t.Errorf("Retry-After = %d, want a value inside the 30s window", secs)
	}
}

// The window is fixed, not sliding: once it elapses the client's budget is
// whole again.
func TestRateLimitBudgetReturnsAfterTheWindow(t *testing.T) {
	app := limitedApp(t, RateLimitConfig{Max: 1, Window: 50 * time.Millisecond})

	request(t, app, "alice")
	if got := request(t, app, "alice").StatusCode; got != http.StatusTooManyRequests {
		t.Fatalf("second request inside the window = %d, want 429", got)
	}

	time.Sleep(60 * time.Millisecond)

	if got := request(t, app, "alice").StatusCode; got != http.StatusOK {
		t.Errorf("request after the window = %d, want 200", got)
	}
}

// Zero values must not mean "allow nothing" — an unconfigured limiter falls
// back to its documented defaults rather than refusing every request.
func TestRateLimitZeroConfigUsesDefaults(t *testing.T) {
	app := limitedApp(t, RateLimitConfig{})

	for i := 1; i <= 10; i++ {
		if got := request(t, app, "alice").StatusCode; got != http.StatusOK {
			t.Fatalf("request %d = %d, want 200 under the default max of 10", i, got)
		}
	}
	if got := request(t, app, "alice").StatusCode; got != http.StatusTooManyRequests {
		t.Errorf("request 11 = %d, want 429", got)
	}
}
