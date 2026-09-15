package config

import (
	"log/slog"
	"testing"
	"time"
)

// The config package is small, and all of it runs exactly once at startup —
// which is why it is worth testing. A rejected value here is a process that
// refuses to boot; a silently accepted one is a service running with settings
// nobody chose.

func TestLogLevelDecode(t *testing.T) {
	cases := []struct {
		in   string
		want slog.Level
	}{
		{"error", slog.LevelError},
		{"warn", slog.LevelWarn},
		{"info", slog.LevelInfo},
		{"debug", slog.LevelDebug},
		// Case and surrounding whitespace come from hand-edited env files, so
		// they are normalized rather than rejected.
		{"INFO", slog.LevelInfo},
		{"  Debug  ", slog.LevelDebug},
	}
	for _, tc := range cases {
		t.Run(tc.in, func(t *testing.T) {
			var got LogLevel
			if err := got.Decode(tc.in); err != nil {
				t.Fatalf("Decode(%q): %v", tc.in, err)
			}
			if slog.Level(got) != tc.want {
				t.Errorf("Decode(%q) = %v, want %v", tc.in, slog.Level(got), tc.want)
			}
		})
	}
}

// An unknown level must be an error, not a default. slog.LevelDebug is 0, so a
// permissive decoder would turn a typo into verbose production logs — which is
// a data-leak shape, not a cosmetic one.
func TestLogLevelDecodeRejectsUnknown(t *testing.T) {
	for _, in := range []string{"", "verbose", "trace", "INF0"} {
		t.Run(in, func(t *testing.T) {
			var got LogLevel
			if err := got.Decode(in); err == nil {
				t.Fatalf("Decode(%q) returned no error, got level %v", in, slog.Level(got))
			}
		})
	}
}

// requiredEnv sets the variables that have no default, so Read can complete.
// Read calls log.Fatalf when one is missing, which kills the test process
// rather than failing a case — so the "it is required" behaviour is asserted by
// the service failing to boot, not here.
//
// Setting a variable a given feature build does not define is harmless:
// envconfig ignores what no field claims.
func requiredEnv(t *testing.T) {
	t.Helper()
	t.Setenv("MY_PROJECT_API_JWT_SECRET", "test-secret")
}

func TestReadAppliesDefaults(t *testing.T) {
	requiredEnv(t)
	cfg := Read(false)

	if cfg.Health.Port != 3333 {
		t.Errorf("Health.Port = %d, want 3333", cfg.Health.Port)
	}
	if slog.Level(cfg.Log.Level) != slog.LevelInfo {
		t.Errorf("Log.Level = %v, want info", slog.Level(cfg.Log.Level))
	}
	if cfg.Fiber.PathPrefix != "/api/v1" {
		t.Errorf("Fiber.PathPrefix = %q, want /api/v1", cfg.Fiber.PathPrefix)
	}
	// The timeouts are the slowloris defaults documented in
	// infra/OPERATIONS.md. A zero here means the protection silently vanished.
	if cfg.Fiber.ReadTimeout != 15*time.Second {
		t.Errorf("Fiber.ReadTimeout = %v, want 15s", cfg.Fiber.ReadTimeout)
	}
	if cfg.Fiber.WriteTimeout != 30*time.Second {
		t.Errorf("Fiber.WriteTimeout = %v, want 30s", cfg.Fiber.WriteTimeout)
	}
	if cfg.Fiber.IdleTimeout != 120*time.Second {
		t.Errorf("Fiber.IdleTimeout = %v, want 120s", cfg.Fiber.IdleTimeout)
	}
}

func TestReadFromEnvironment(t *testing.T) {
	requiredEnv(t)
	t.Setenv("MY_PROJECT_API_HEALTH_PORT", "4444")
	t.Setenv("MY_PROJECT_API_LOG_LEVEL", "warn")
	t.Setenv("MY_PROJECT_API_FIBER_PORT", "8081")
	t.Setenv("MY_PROJECT_API_FIBER_READ_TIMEOUT", "5s")
	t.Setenv("MY_PROJECT_API_FIBER_CORS_ALLOW_ORIGINS", "https://a.example,https://b.example")

	cfg := Read(false)

	if cfg.Health.Port != 4444 {
		t.Errorf("Health.Port = %d, want 4444", cfg.Health.Port)
	}
	// Proves the custom decoder is reached through envconfig, not just when
	// called directly.
	if slog.Level(cfg.Log.Level) != slog.LevelWarn {
		t.Errorf("Log.Level = %v, want warn", slog.Level(cfg.Log.Level))
	}
	if cfg.Fiber.Port != 8081 {
		t.Errorf("Fiber.Port = %d, want 8081", cfg.Fiber.Port)
	}
	if cfg.Fiber.ReadTimeout != 5*time.Second {
		t.Errorf("Fiber.ReadTimeout = %v, want 5s", cfg.Fiber.ReadTimeout)
	}
	if len(cfg.Fiber.CORSAllowOrigins) != 2 || cfg.Fiber.CORSAllowOrigins[0] != "https://a.example" {
		t.Errorf("Fiber.CORSAllowOrigins = %v, want the two configured origins", cfg.Fiber.CORSAllowOrigins)
	}
}

// Init caches; Get serves the cached copy. The cache is what lets any package
// read config without threading it through every constructor.
func TestInitThenGet(t *testing.T) {
	requiredEnv(t)
	t.Setenv("MY_PROJECT_API_FIBER_PORT", "9091")
	t.Cleanup(func() { c = nil })

	initialized := Init(false)
	if initialized.Fiber.Port != 9091 {
		t.Fatalf("Init returned port %d, want 9091", initialized.Fiber.Port)
	}
	if got := Get(); got.Fiber.Port != 9091 {
		t.Errorf("Get returned port %d, want 9091", got.Fiber.Port)
	}
}
