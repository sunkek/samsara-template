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
	// feat:if template
	// The sample domains are Postgres-backed; a build without Postgres keeps
	// only the supervisor + HTTP skeleton. See scripts/apply_features.sh.
	// feat:end
	// feat:if postgresql
	"github.com/sunkek/samsara-template/backend/internal/domain/auth"
	authfiber "github.com/sunkek/samsara-template/backend/internal/domain/auth/adapter/fiber"
	// feat:if !redis
	//~ authmemory "github.com/sunkek/samsara-template/backend/internal/domain/auth/adapter/memory"
	// feat:end
	authpostgresql "github.com/sunkek/samsara-template/backend/internal/domain/auth/adapter/postgresql"
	// feat:if redis
	authredis "github.com/sunkek/samsara-template/backend/internal/domain/auth/adapter/redis"
	// feat:end
	"github.com/sunkek/samsara-template/backend/internal/domain/note"
	notefiber "github.com/sunkek/samsara-template/backend/internal/domain/note/adapter/fiber"
	notepostgresql "github.com/sunkek/samsara-template/backend/internal/domain/note/adapter/postgresql"
	// feat:if redis,rabbitmq
	"github.com/sunkek/samsara-template/backend/internal/domain/article"
	articlefiber "github.com/sunkek/samsara-template/backend/internal/domain/article/adapter/fiber"
	articlepostgresql "github.com/sunkek/samsara-template/backend/internal/domain/article/adapter/postgresql"
	articlerabbit "github.com/sunkek/samsara-template/backend/internal/domain/article/adapter/rabbitmq"
	articleredis "github.com/sunkek/samsara-template/backend/internal/domain/article/adapter/redis"
	"github.com/sunkek/samsara-template/backend/internal/domain/articlestats"
	articlestatsfiber "github.com/sunkek/samsara-template/backend/internal/domain/articlestats/adapter/fiber"
	articlestatspostgresql "github.com/sunkek/samsara-template/backend/internal/domain/articlestats/adapter/postgresql"
	articlestatsrabbit "github.com/sunkek/samsara-template/backend/internal/domain/articlestats/adapter/rabbitmq"
	// feat:end
	// feat:end

	"github.com/gofiber/contrib/v3/swaggo"
	gf "github.com/gofiber/fiber/v3"
	"github.com/gofiber/fiber/v3/middleware/adaptor"
	"github.com/gofiber/fiber/v3/middleware/static"
	"github.com/sunkek/samsara"
	"github.com/sunkek/samsara-components/fiber"
	// feat:if postgresql
	"github.com/sunkek/samsara-components/postgresql"
	// feat:end
	// feat:if rabbitmq
	"github.com/sunkek/samsara-components/rabbitmq"
	// feat:end
	// feat:if redis
	"github.com/sunkek/samsara-components/redis"
	// feat:end
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

	// feat:if postgresql
	postgresCmp := postgresql.New(postgresql.Config(cfg.PostgreSQL), postgresql.WithLogger(logger), postgresql.WithName("postgresql"))
	sup.Add(postgresCmp,
		samsara.WithTier(samsara.TierCritical),
		samsara.WithRestartPolicy(samsara.MaxRetries(5, 5*time.Second)),
	)

	// feat:end

	// feat:if rabbitmq
	rabbitCmp := rabbitmq.New(rabbitmq.Config(cfg.RabbitMQ), rabbitmq.WithLogger(logger), rabbitmq.WithName("rabbitmq"))
	sup.Add(rabbitCmp,
		samsara.WithTier(samsara.TierCritical),
		samsara.WithRestartPolicy(samsara.MaxRetries(5, 5*time.Second)),
	)
	// feat:if redis
	// Declare the events exchange up front; the component (re-)declares it on
	// every Start, so this is safe to call before the supervisor runs. The
	// exchange serves the article domain, which exists only in a build that has
	// both Redis and RabbitMQ.
	if err := rabbitCmp.DeclareExchange(cfg.Events.Exchange, rabbitmq.ExchangeTopic, true); err != nil {
		logger.Error("declare events exchange", "error", err)
		os.Exit(1)
	}
	// feat:end

	// feat:end

	// feat:if redis
	redisCmp := redis.New(redis.Config(cfg.Redis), redis.WithLogger(logger), redis.WithName("redis"))
	sup.Add(redisCmp,
		samsara.WithTier(samsara.TierCritical),
		samsara.WithRestartPolicy(samsara.MaxRetries(5, 5*time.Second)),
	)

	// feat:end

	fiberCmp := fiber.New(cfg.Fiber.ToSamsaraCfg(), fiber.WithLogger(logger), fiber.WithName("fiber"))
	// The HTTP server starts only once the infra it talks to is up.
	fiberDeps := make([]string, 0, 3)
	// feat:if postgresql
	fiberDeps = append(fiberDeps, postgresCmp.Name())
	// feat:end
	// feat:if redis
	fiberDeps = append(fiberDeps, redisCmp.Name())
	// feat:end
	// feat:if rabbitmq
	fiberDeps = append(fiberDeps, rabbitCmp.Name())
	// feat:end

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

	// feat:if postgresql
	// Domains. Build each as DB adapter → domain → REST adapter. The REST
	// adapter takes the domain's inbound Service interface, so wiring is
	// compile-time checked. Construct domains with no cross-domain deps first
	// and pass other domains' interfaces into the constructors that need them.

	// auth: owns users and JWT. Built first so its middleware can guard the
	// other domains' routes.
	authDB := authpostgresql.New(postgresCmp)
	// feat:if redis
	authRevoker := authredis.New(redisCmp)
	// feat:else
	//~ // Without Redis the denylist is process-local and resets on restart —
	//~ // fine for one replica, wire a shared store before scaling out.
	//~ authRevoker := authmemory.New()
	// feat:end
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

	// note: the protected vertical-slice sample — DB adapter, domain, REST
	// adapter, nothing else. This is the domain a fork copies.
	noteDB := notepostgresql.New(postgresCmp)
	noteDomain := note.New(noteDB)
	_ = notefiber.New(fiberCmp, noteDomain)

	// feat:if redis,rabbitmq
	// article: the same slice plus optional infrastructure — cache-aside reads
	// through Redis and article.created events through RabbitMQ.
	articleDB := articlepostgresql.New(postgresCmp)
	articleCache := articleredis.New(redisCmp, cfg.Article.CacheTTL)
	articleEvents := articlerabbit.New(rabbitCmp, cfg.Events.Exchange, cfg.Events.ArticleCreatedKey)
	articleDomain := article.New(articleDB, articleCache, articleEvents)
	_ = articlefiber.New(fiberCmp, articleDomain)

	// articlestats: a read model projected from article.created events
	// (CQRS-lite). The rabbitmq component owns the consume loop; we register a
	// handler and a queue bound to the events exchange. Subscribe is safe before
	// Start — the binding is (re-)applied when the broker connects.
	statsDB := articlestatspostgresql.New(postgresCmp)
	statsDomain := articlestats.New(statsDB)
	statsConsumer := articlestatsrabbit.NewConsumer(statsDomain)
	if err := rabbitCmp.SubscribeWithKey(cfg.Events.Exchange, cfg.Events.ArticleWorkerQueue, cfg.Events.ArticleCreatedKey, statsConsumer.Handle); err != nil {
		logger.Error("subscribe article.created worker", "error", err)
		os.Exit(1)
	}
	_ = articlestatsfiber.New(fiberCmp, statsDomain)
	// feat:end
	// feat:end

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
