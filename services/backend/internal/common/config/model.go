package config

import (
	"time"

	gf "github.com/gofiber/fiber/v3"
	"github.com/sunkek/samsara-components/fiber"
)

type Config struct {
	Health Health `envconfig:"HEALTH"`
	Log    Log    `envconfig:"LOG"`

	Fiber      Fiber      `envconfig:"FIBER"`
	PostgreSQL PostgreSQL `envconfig:"POSTGRESQL"`
	RabbitMQ   RabbitMQ   `envconfig:"RABBITMQ"`
	Redis      Redis      `envconfig:"REDIS"`
	S3         S3         `envconfig:"S3"`
	JWT        JWT        `envconfig:"JWT"`
	Auth       Auth       `envconfig:"AUTH"`
	Note       Note       `envconfig:"NOTE"`
	Events     Events     `envconfig:"EVENTS"`
}

// Events configures the RabbitMQ topic exchange and routing/queue names used by
// the note domain's publisher and the consumer worker.
type Events struct {
	Exchange        string `envconfig:"EXCHANGE" default:"my_project.events"`
	NoteCreatedKey  string `envconfig:"NOTE_CREATED_KEY" default:"note.created"`
	NoteWorkerQueue string `envconfig:"NOTE_WORKER_QUEUE" default:"note.created.worker"`
}

// Note configures the sample note domain. CacheTTL is the Redis cache-aside
// entry lifetime for note reads.
type Note struct {
	CacheTTL time.Duration `envconfig:"CACHE_TTL" default:"60s"`
}

// Auth tunes the auth domain's HTTP-edge protections. RateLimit* throttle the
// register/login/refresh endpoints per client IP to blunt credential
// brute-forcing.
type Auth struct {
	RateLimitMax    int           `envconfig:"RATE_LIMIT_MAX" default:"10"`
	RateLimitWindow time.Duration `envconfig:"RATE_LIMIT_WINDOW" default:"1m"`
}

// JWT configures the auth domain's token signing. Secret is required (no
// default) — the service fails fast at startup if it is empty.
type JWT struct {
	Secret     string        `envconfig:"SECRET" required:"true"`
	AccessTTL  time.Duration `envconfig:"ACCESS_TTL" default:"15m"`
	RefreshTTL time.Duration `envconfig:"REFRESH_TTL" default:"720h"`
}

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
}

func (f Fiber) ToSamsaraCfg() fiber.Config {
	return fiber.Config{
		Host:                  f.Host,
		Port:                  f.Port,
		PathPrefix:            f.PathPrefix,
		BodyLimitMB:           f.BodyLimitMB,
		CORSAllowOrigins:      f.CORSAllowOrigins,
		CORSAllowMethods:      f.CORSAllowMethods,
		CORSAllowHeaders:      f.CORSAllowHeaders,
		ReadTimeout:           f.ReadTimeout,
		WriteTimeout:          f.WriteTimeout,
		IdleTimeout:           f.IdleTimeout,
		ErrorHandler:          f.ErrorHandler,
		LoggerFormat:          f.LoggerFormat,
		EnableSecurityHeaders: f.EnableSecurityHeaders,
	}
}

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
}

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
