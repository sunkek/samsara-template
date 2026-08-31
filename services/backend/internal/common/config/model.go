package config

import (
	"time"

	gf "github.com/gofiber/fiber/v3"
	"github.com/sunkek/samsara-components/fiber"
)

type Config struct {
	Health Health `envconfig:"HEALTH"`
	Log    Log    `envconfig:"LOG"`

	Fiber Fiber `envconfig:"FIBER"`
	// feat:if postgresql
	PostgreSQL PostgreSQL `envconfig:"POSTGRESQL"`
	// feat:end
	// feat:if rabbitmq
	RabbitMQ RabbitMQ `envconfig:"RABBITMQ"`
	// feat:end
	// feat:if redis
	Redis Redis `envconfig:"REDIS"`
	// feat:end
	S3 S3 `envconfig:"S3"`
	// feat:if postgresql
	JWT  JWT  `envconfig:"JWT"`
	Auth Auth `envconfig:"AUTH"`
	// feat:end
	// feat:if redis,rabbitmq
	Article Article `envconfig:"ARTICLE"`
	// feat:end
	// feat:if redis,rabbitmq
	Events Events `envconfig:"EVENTS"`
	// feat:end
}

// feat:if redis,rabbitmq
// Events configures the RabbitMQ topic exchange and routing/queue names used by
// the article domain's publisher and the articlestats consumer worker.
type Events struct {
	Exchange           string `envconfig:"EXCHANGE" default:"my_project.events"`
	ArticleCreatedKey  string `envconfig:"ARTICLE_CREATED_KEY" default:"article.created"`
	ArticleWorkerQueue string `envconfig:"ARTICLE_WORKER_QUEUE" default:"article.created.worker"`
}

// feat:end
// feat:if redis,rabbitmq
// Article configures the sample article domain. CacheTTL is the Redis
// cache-aside entry lifetime for article reads.
type Article struct {
	CacheTTL time.Duration `envconfig:"CACHE_TTL" default:"60s"`
}

// feat:end
// feat:if postgresql
// Auth tunes the auth domain's HTTP-edge protections. RateLimit* throttle the
// register/login/refresh endpoints per client IP to blunt credential
// brute-forcing.
type Auth struct {
	RateLimitMax    int           `envconfig:"RATE_LIMIT_MAX" default:"10"`
	RateLimitWindow time.Duration `envconfig:"RATE_LIMIT_WINDOW" default:"1m"`
}

// feat:end
// feat:if postgresql
// JWT configures the auth domain's token signing. Secret is required (no
// default) — the service fails fast at startup if it is empty.
type JWT struct {
	Secret     string        `envconfig:"SECRET" required:"true"`
	AccessTTL  time.Duration `envconfig:"ACCESS_TTL" default:"15m"`
	RefreshTTL time.Duration `envconfig:"REFRESH_TTL" default:"720h"`
}

// feat:end
type Health struct {
	Port     int           `envconfig:"PORT" default:"3333"`
	Interval time.Duration `envconfig:"INTERVAL" default:"1m"`
}

type Log struct {
	Level  LogLevel `envconfig:"LEVEL" default:"info"`
	Source bool     `envconfig:"SOURCE" default:"false"`
}

type Fiber struct {
	Host             string   `envconfig:"HOST" default:"0.0.0.0"`
	Port             int      `envconfig:"PORT" default:"80"`
	PathPrefix       string   `envconfig:"PATH_PREFIX" default:"/api/v1"`
	BodyLimitMB      int      `envconfig:"BODY_LIMIT_MB" default:"20"`
	CORSAllowOrigins []string `envconfig:"CORS_ALLOW_ORIGINS" default:"*"`
	CORSAllowMethods []string `envconfig:"CORS_ALLOW_METHODS" default:"*"`
	CORSAllowHeaders []string `envconfig:"CORS_ALLOW_HEADERS" default:"*"`
	// Timeouts default to non-zero values so the server is not exposed to
	// slowloris-style attacks out of the box. Raise WriteTimeout if you stream
	// large responses; set to 0 to disable a given timeout entirely.
	ReadTimeout           time.Duration   `envconfig:"READ_TIMEOUT" default:"15s"`
	WriteTimeout          time.Duration   `envconfig:"WRITE_TIMEOUT" default:"30s"`
	IdleTimeout           time.Duration   `envconfig:"IDLE_TIMEOUT" default:"120s"`
	ErrorHandler          gf.ErrorHandler `ignored:"true"`
	LoggerFormat          string          `envconfig:"LOGGER_FORMAT" default:"{\"time\":\"${time}\",\"ip\":\"${ip}\",\"x-forwarded-for\":\"${reqHeader:X-Forwarded-For}\",\"status\":${status},\"latency\":\"${latency}\",\"method\":\"${method}\",\"path\":\"${path}\",\"error\":\"${error}\"}\n"`
	EnableSecurityHeaders *bool           `envconfig:"ENABLE_SECURITY_HEADERS"`
	SwaggerFilePath       string          `envconfig:"SWAGGER_FILE_PATH"`

	// TrustProxy makes c.IP() read TrustProxyHeader instead of the socket peer,
	// but only when that peer is one of TrustedProxies. Turn it on exactly when
	// the service sits behind a reverse proxy you control — in this template's
	// stage/prod stacks it does, behind nginx, which is why every request would
	// otherwise carry nginx's address and share one rate-limit bucket.
	//
	// Leave it off when the service is exposed directly: a client can send its
	// own X-Forwarded-For, so trusting the header without pinning the peer hands
	// every caller a free identity.
	TrustProxy bool `envconfig:"TRUST_PROXY" default:"false"`
	// TrustedProxies are the immediate peers whose forwarded-for header is
	// believed, as IPs or CIDRs. Keep it as tight as the deployment allows.
	TrustedProxies []string `envconfig:"TRUSTED_PROXIES"`
	// TrustProxyHeader is the header consulted for the client address.
	//
	// Fiber reads the LEFT-MOST entry, which is only spoof-safe when a single
	// proxy OVERWRITES the header. The nginx config in this template sets
	// X-Forwarded-For to $remote_addr for that reason. If you put another proxy
	// in front, that chain appends instead, and the left-most entry becomes
	// attacker-controlled — resolve the client address yourself in that case.
	TrustProxyHeader string `envconfig:"TRUST_PROXY_HEADER" default:"X-Forwarded-For"`
}

func (f Fiber) ToSamsaraCfg() fiber.Config {
	return fiber.Config{
		Host:             f.Host,
		Port:             f.Port,
		PathPrefix:       f.PathPrefix,
		BodyLimitMB:      f.BodyLimitMB,
		CORSAllowOrigins: f.CORSAllowOrigins,
		CORSAllowMethods: f.CORSAllowMethods,
		CORSAllowHeaders: f.CORSAllowHeaders,
		ReadTimeout:      f.ReadTimeout,
		WriteTimeout:     f.WriteTimeout,
		IdleTimeout:      f.IdleTimeout,
		ErrorHandler:     f.ErrorHandler,
		TrustProxy:       f.TrustProxy,
		TrustProxyConfig: gf.TrustProxyConfig{Proxies: f.TrustedProxies},
		ProxyHeader:      f.TrustProxyHeader,
		// Skip malformed entries rather than treating them as a client address.
		EnableIPValidation:    true,
		LoggerFormat:          f.LoggerFormat,
		EnableSecurityHeaders: f.EnableSecurityHeaders,
	}
}

// feat:if postgresql
type PostgreSQL struct {
	Host           string        `envconfig:"HOST" default:"postgresql"`
	Port           int           `envconfig:"PORT" default:"5432"`
	Name           string        `envconfig:"NAME" default:"postgresql"`
	User           string        `envconfig:"USER" default:"postgresql"`
	Pass           string        `envconfig:"PASS"`
	SSLMode        string        `envconfig:"SSL_MODE" default:"disable"`
	URI            string        `envconfig:"URI"`
	ConnectTimeout time.Duration `envconfig:"CONNECT_TIMEOUT"`
	MaxConns       int32         `envconfig:"MAX_CONNS"`
	MinConns       int32         `envconfig:"MIN_CONNS"`
}

// feat:end
// feat:if rabbitmq
type RabbitMQ struct {
	Host           string        `envconfig:"HOST" default:"rabbitmq"`
	Port           int           `envconfig:"PORT" default:"5672"`
	VHost          string        `envconfig:"VHOST" default:"app"`
	User           string        `envconfig:"USER" default:"app"`
	Pass           string        `envconfig:"PASS"`
	URI            string        `envconfig:"URI"`
	ConnectTimeout time.Duration `envconfig:"CONNECT_TIMEOUT"`
	PublishTimeout time.Duration `envconfig:"PUBLISH_TIMEOUT"`
}

// feat:end
// feat:if redis
type Redis struct {
	Host           string        `envconfig:"HOST" default:"redis"`
	Port           int           `envconfig:"PORT" default:"6379"`
	DB             int           `envconfig:"DB" default:"0"`
	User           string        `envconfig:"USER" default:"redis"`
	Pass           string        `envconfig:"PASS"`
	ConnectTimeout time.Duration `envconfig:"CONNECT_TIMEOUT"`
	DialTimeout    time.Duration `envconfig:"DIAL_TIMEOUT"`
	ReadTimeout    time.Duration `envconfig:"READ_TIMEOUT"`
	WriteTimeout   time.Duration `envconfig:"WRITE_TIMEOUT"`
	PoolSize       int           `envconfig:"POOL_SIZE"`

	// TLS knobs (samsara-components/redis v0.2+). Disabled by default — safe on a
	// trusted internal network; enable when Redis is reached over an untrusted
	// link (mirrors the Postgres SSL_MODE hardening). Field order must match
	// redis.Config for the struct conversion in cmd/main.
	TLS                   bool   `envconfig:"TLS" default:"false"`
	TLSCAFile             string `envconfig:"TLS_CA_FILE"`
	TLSCertFile           string `envconfig:"TLS_CERT_FILE"`
	TLSKeyFile            string `envconfig:"TLS_KEY_FILE"`
	TLSServerName         string `envconfig:"TLS_SERVER_NAME"`
	TLSInsecureSkipVerify bool   `envconfig:"TLS_INSECURE_SKIP_VERIFY" default:"false"`
	TLSMinVersion         string `envconfig:"TLS_MIN_VERSION"`
}

// feat:end
// S3 is optional object-storage config. No S3 component is registered in
// main.go by default; wire one when you need uploads. Leave blank to ignore.
type S3 struct {
	Endpoint         string        `envconfig:"ENDPOINT"`
	Region           string        `envconfig:"REGION"`
	KeyID            string        `envconfig:"KEY_ID"`
	Secret           string        `envconfig:"SECRET"`
	ConnectTimeout   time.Duration `envconfig:"CONNECT_TIMEOUT"`
	PresignTTL       time.Duration `envconfig:"PRESIGNED_TTL"`
	PathStyleForcing bool          `envconfig:"PATH_STYLE_FORCING"`
}
