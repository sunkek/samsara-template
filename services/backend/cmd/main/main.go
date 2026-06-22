package main

import (
	"context"
	"flag"
	"log/slog"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/sunkek/samsara-template/backend/internal/common/config"
	"github.com/sunkek/samsara-template/backend/internal/common/e"
	"github.com/sunkek/samsara-template/backend/internal/common/metrics"
	"github.com/sunkek/samsara-template/backend/internal/common/middleware"
	"github.com/sunkek/samsara-template/backend/internal/domain/auth"
	authfiber "github.com/sunkek/samsara-template/backend/internal/domain/auth/adapter/fiber"
	authpostgresql "github.com/sunkek/samsara-template/backend/internal/domain/auth/adapter/postgresql"
	authredis "github.com/sunkek/samsara-template/backend/internal/domain/auth/adapter/redis"
	"github.com/sunkek/samsara-template/backend/internal/domain/note"
	notefiber "github.com/sunkek/samsara-template/backend/internal/domain/note/adapter/fiber"
	notepostgresql "github.com/sunkek/samsara-template/backend/internal/domain/note/adapter/postgresql"
	noterabbit "github.com/sunkek/samsara-template/backend/internal/domain/note/adapter/rabbitmq"
	noteredis "github.com/sunkek/samsara-template/backend/internal/domain/note/adapter/redis"
	"github.com/sunkek/samsara-template/backend/internal/domain/notestats"
	notestatsfiber "github.com/sunkek/samsara-template/backend/internal/domain/notestats/adapter/fiber"
	notestatspostgresql "github.com/sunkek/samsara-template/backend/internal/domain/notestats/adapter/postgresql"
	notestatsrabbit "github.com/sunkek/samsara-template/backend/internal/domain/notestats/adapter/rabbitmq"

	"github.com/gofiber/contrib/v3/swaggo"
	gf "github.com/gofiber/fiber/v3"
	"github.com/gofiber/fiber/v3/middleware/adaptor"
	"github.com/gofiber/fiber/v3/middleware/static"
	"github.com/sunkek/samsara"
	"github.com/sunkek/samsara-components/fiber"
	"github.com/sunkek/samsara-components/postgresql"
	"github.com/sunkek/samsara-components/rabbitmq"
	"github.com/sunkek/samsara-components/redis"
)

// @Title						My Project API
// @Version					0.1
// @Description				My Project backend API.
// @Contact.name				My Project
// @BasePath					/api/v1
// @securityDefinitions.apikey	BearerAuth
// @in							header
// @name						Authorization
func main() {
	local := flag.Bool("l", false, "load env/local/api.env for running outside Docker")
	flag.Parse()
	cfg := config.Init(*local)
	cfg.Fiber.ErrorHandler = func(ctx gf.Ctx, err error) error {
		// Status mapping lives in e.HTTPStatus so the metrics middleware can label
		// error responses with the same code the client receives.
		status := e.HTTPStatus(err)
		// Never echo internal wrapped messages (e.g. "revoke token: redis
		// connection refused") to clients. 5xx responses carry a generic
		// message; 4xx messages are user-facing by construction.
		msg := err.Error()
		if status >= 500 {
			msg = "internal server error"
		}
		return ctx.Status(status).JSON(gf.Map{"error": msg})
	}
	logger := slog.New(slog.NewJSONHandler(
		os.Stderr,
		&slog.HandlerOptions{
			Level:     slog.Level(cfg.Log.Level),
			AddSource: cfg.Log.Source,
		},
	))
	// Default logger for code paths without a request-scoped logger in context
	// (logging.From falls back to slog.Default()).
	slog.SetDefault(logger)

	// Warn loudly if CORS is left wide open. A wildcard origin on an
	// authenticated API lets any site issue credentialed cross-origin requests;
	// set explicit origins via MY_PROJECT_API_FIBER_CORS_ALLOW_ORIGINS in
	// stage/prod.
	for _, o := range cfg.Fiber.CORSAllowOrigins {
		if strings.TrimSpace(o) == "*" {
			logger.Warn("CORS allows all origins (*) — set explicit origins for production")
			break
		}
	}

	sup := samsara.NewSupervisor(
		samsara.WithSupervisorLogger(logger),
		samsara.WithMetricsObserver(metrics.NewObserver()),
		samsara.WithHealthInterval(cfg.Health.Interval),
		samsara.WithEventHooks(&samsara.EventHooks{
			OnUnhealthy: func(component string, err error) {
				logger.Error("component unhealthy", "component", component, "error", err)
			},
			OnRecovered: func(component string) {
				logger.Info("component recovered", "component", component)
			},
			OnFailed: func(component string, err error) {
				logger.Error("component permanently failed", "component", component, "error", err)
			},
		}),
	)

	hs := samsara.NewHealthServer(
		sup,
		samsara.WithHealthLogger(logger),
		samsara.WithHealthName("health"),
		samsara.WithHealthAddr(":"+strconv.Itoa(cfg.Health.Port)),
	)
	sup.Add(hs, samsara.WithTier(samsara.TierCritical))

	postgresCmp := postgresql.New(postgresql.Config(cfg.PostgreSQL), postgresql.WithLogger(logger), postgresql.WithName("postgresql"))
	sup.Add(postgresCmp,
		samsara.WithTier(samsara.TierCritical),
		samsara.WithRestartPolicy(samsara.MaxRetries(5, 5*time.Second)),
	)

	rabbitCmp := rabbitmq.New(rabbitmq.Config(cfg.RabbitMQ), rabbitmq.WithLogger(logger), rabbitmq.WithName("rabbitmq"))
	sup.Add(rabbitCmp,
		samsara.WithTier(samsara.TierCritical),
		samsara.WithRestartPolicy(samsara.MaxRetries(5, 5*time.Second)),
	)
	// Declare the events exchange up front; the component (re-)declares it on
	// every Start, so this is safe to call before the supervisor runs.
	if err := rabbitCmp.DeclareExchange(cfg.Events.Exchange, rabbitmq.ExchangeTopic, true); err != nil {
		logger.Error("declare events exchange", "error", err)
		os.Exit(1)
	}

	redisCmp := redis.New(redis.Config(cfg.Redis), redis.WithLogger(logger), redis.WithName("redis"))
	sup.Add(redisCmp,
		samsara.WithTier(samsara.TierCritical),
		samsara.WithRestartPolicy(samsara.MaxRetries(5, 5*time.Second)),
	)

	fiberCmp := fiber.New(cfg.Fiber.ToSamsaraCfg(), fiber.WithLogger(logger), fiber.WithName("fiber"))
	fiberDeps := []string{postgresCmp.Name(), redisCmp.Name(), rabbitCmp.Name()}

	// Correlate every request: assign/propagate X-Request-ID and seed a
	// request-scoped logger. Registered first so all routes are covered.
	fiberCmp.Use(middleware.RequestID(logger))
	// Record request count/latency per method+route.
	fiberCmp.Use(middleware.Metrics())
	// Expose Prometheus metrics. Public (scraped without a token); in production
	// bind it to an internal network/port rather than the public ingress.
	fiberCmp.Register(func(r gf.Router) {
		r.Get("/metrics", adaptor.HTTPHandler(metrics.Handler()))
	})

	if cfg.Fiber.SwaggerFilePath != "" {
		fiberCmp.Use(cfg.Fiber.PathPrefix+"/docs/swagger.json", static.New(cfg.Fiber.SwaggerFilePath))
		fiberCmp.Register(func(r gf.Router) {
			r.Get("/docs/*", swaggo.New(swaggo.Config{
				URL: cfg.Fiber.PathPrefix + "/docs/swagger.json",
			}))
			r.Get("/", func(ctx gf.Ctx) error {
				return ctx.Redirect().To(cfg.Fiber.PathPrefix + "/docs")
			})
		})
	}

	sup.Add(fiberCmp,
		samsara.WithTier(samsara.TierCritical),
		samsara.WithRestartPolicy(samsara.MaxRetries(5, 5*time.Second)),
		samsara.WithDependencies(fiberDeps...),
	)

	// Domains. Build each as DB adapter → domain → REST adapter. The REST
	// adapter takes the domain's inbound Service interface, so wiring is
	// compile-time checked. Construct domains with no cross-domain deps first
	// and pass other domains' interfaces into the constructors that need them.

	// auth: owns users and JWT. Built first so its middleware can guard the
	// other domains' routes.
	authDB := authpostgresql.New(postgresCmp)
	authRevoker := authredis.New(redisCmp)
	authDomain := auth.New(authDB, authRevoker, cfg.JWT.Secret, cfg.JWT.AccessTTL, cfg.JWT.RefreshTTL)
	// Throttle the credential endpoints per client IP to blunt brute-forcing.
	authLimiter := middleware.RateLimit(middleware.RateLimitConfig{
		Max:    cfg.Auth.RateLimitMax,
		Window: cfg.Auth.RateLimitWindow,
	})
	authREST := authfiber.New(fiberCmp, authDomain, authLimiter)

	// Require a valid access token on every route except the public ones
	// (auth endpoints, swagger). Verified claims land in ctx.Locals — read
	// them with authfiber.ClaimsFromContext. Health probes use the samsara
	// health server on its own port (see WithHealthAddr above); the fiber
	// component's built-in /health is not listed here because it registers
	// ahead of this middleware and stays public regardless — we just don't
	// rely on it.
	publicPrefixes := []string{
		cfg.Fiber.PathPrefix + "/auth",
		cfg.Fiber.PathPrefix + "/docs",
		cfg.Fiber.PathPrefix + "/metrics",
	}
	fiberCmp.Use(authREST.Middleware(publicPrefixes...))

	// note: protected sample domain. Reads are cache-aside via Redis; creates
	// publish a note.created event to RabbitMQ.
	noteDB := notepostgresql.New(postgresCmp)
	noteCache := noteredis.New(redisCmp, cfg.Note.CacheTTL)
	noteEvents := noterabbit.New(rabbitCmp, cfg.Events.Exchange, cfg.Events.NoteCreatedKey)
	noteDomain := note.New(noteDB, noteCache, noteEvents)
	_ = notefiber.New(fiberCmp, noteDomain)

	// notestats: a read model projected from note.created events (CQRS-lite).
	// The rabbitmq component owns the consume loop; we register a handler and a
	// queue bound to the events exchange. Subscribe is safe before Start — the
	// binding is (re-)applied when the broker connects.
	statsDB := notestatspostgresql.New(postgresCmp)
	statsDomain := notestats.New(statsDB)
	statsConsumer := notestatsrabbit.NewConsumer(statsDomain)
	if err := rabbitCmp.SubscribeWithKey(cfg.Events.Exchange, cfg.Events.NoteWorkerQueue, cfg.Events.NoteCreatedKey, statsConsumer.Handle); err != nil {
		logger.Error("subscribe note.created worker", "error", err)
		os.Exit(1)
	}
	_ = notestatsfiber.New(fiberCmp, statsDomain)

	app := samsara.NewApplication(
		samsara.WithSupervisor(sup),
		samsara.WithLogger(logger),
		samsara.WithShutdownTimeout(30*time.Second),
		samsara.WithMainFunc(func(ctx context.Context) error {
			<-ctx.Done()
			return nil
		}),
	)

	if err := app.Run(); err != nil {
		logger.Error("application exited with error", "error", err)
		os.Exit(1)
	}
}
