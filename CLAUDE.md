# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository. Coding style, testing rules, and commit format are in `AGENTS.md`.

## What this project is

My Project is a backend + frontend application scaffold. The backend is a Go service built on the `samsara` component supervisor (PostgreSQL, RabbitMQ, Redis, Fiber HTTP server); the frontend is a React/Vite SPA. The repo ships with one sample domain (`note`) demonstrating the full vertical slice; replace it with your own domains.

## Commands

### Local development
```bash
docker network create dev                        # once, creates the shared bridge network
make gen-env APP=my_project                     # first-time env setup: fills env/dev + env/local with shared secrets
make run-local                                    # infra containers + backend live reload + frontend
```

### Individual targets
```bash
make run              # infra containers only (docker compose up -d)
make up                          # full stack in Docker, dev (hot reload)
make up ENVIRONMENT=stage|prod   # full stack, built images (nginx-served frontend)
make stop             # stop and remove containers
make logs             # follow compose logs

cd service/backend && go mod tidy           # populate go.sum
cd service/backend && go build ./cmd/main   # build binary
cd service/backend && go test ./...         # run all tests
cd service/backend && go run ./cmd/main -l  # run locally (loads env/local/api.env)

make gen-api-docs     # regenerate Swagger (swag fmt + swag init)
```

### Database migrations
```bash
make migrate-new n=<name>   # create migration file pair
make migrate-up             # apply all pending migrations
make migrate-down           # roll back one migration
make migrate-force v=<num>  # force migration version
make psql                   # psql as app user
make psql-admin             # psql as postgres superuser
```

## Architecture

### Go module path
`github.com/sunkek/samsara-template/backend` — use this for all internal imports.

### Runtime framework: samsara
Uses `github.com/sunkek/samsara`, a component supervisor. PostgreSQL, RabbitMQ, Redis, and the Fiber HTTP server are each registered as components with a tier (`Critical`/`Significant`) and restart policy. Fiber declares its infra dependencies via `WithDependencies`; `main()` blocks on `<-ctx.Done()`.

### Domain layout
Each domain lives in `service/backend/internal/domain/<name>/`:

```
domain.go        # Domain struct + constructor; wires use-cases into REST adapter
interface.go     # DB and REST interface definitions
usecase_*.go     # one file per use case, methods on Domain
model/           # plain Go structs + shared input types (avoid import cycles)
adapter/
  fiber/         # HTTP adapter: registers routes, stores injected handler funcs
  postgresql/    # DB adapter: SQL queries via samsara-components/postgresql
```

Each domain declares two ports in `interface.go`: `Service` (inbound — the use cases, implemented by `*Domain`) and `DB` (outbound — persistence, implemented by the postgresql adapter). The fiber adapter takes the `Service` and registers routes against it directly, so wiring is compile-time checked (no `SetHandler*` injection, no nil-handler guards). The dependency points adapter → domain; adapters never import each other. Cross-domain calls go through interfaces declared in `interface.go` and injected in `cmd/main/main.go`.

**Adding a new domain:** copy the `note` domain as a starting point, then wire it in `cmd/main/main.go` as DB adapter → `domain.New(db, …deps)` → REST adapter (`fiber.New(fiberCmp, domain)`). Order matters — domains with no cross-domain dependencies must be constructed first.

**Sample `note` domain:** demonstrates the full vertical slice plus a Redis **cache-aside** read path via a `Cache` outbound port (`adapter/redis`): `Get`/`List` serve from cache on a hit and populate it on a miss; `Create` warms the item and invalidates the list. Caching is best-effort (errors fall back to the DB) and tunable via `MY_PROJECT_API_NOTE_CACHE_TTL`; pass `note.NoopCache{}` to disable it. `Create` also publishes a `note.created` event through an `Events` port (`adapter/rabbitmq`) to a RabbitMQ topic exchange (best-effort; `note.NoopEvents{}` disables).

**`notestats` read model:** a separate domain (CQRS-lite) that projects `note.created` events into a single-row `note_stats` table. The samsara rabbitmq component owns the consume loop; `notestats/adapter/rabbitmq` is the message handler, `adapter/postgresql` the projection store, and `adapter/fiber` exposes it at `GET /api/v1/stats`. Event/exchange/queue names are under `MY_PROJECT_API_EVENTS_*`. This is the end-to-end async demo: note create → publish → broker → consumer → projection → `/stats`.

**Auth:** `internal/domain/auth` is a full sample auth domain (register/login/refresh/logout/JWT-verify). Its fiber adapter exposes `Middleware(publicPrefixes...)`, registered in `cmd/main` via `fiberCmp.Use(...)` to protect all routes except `/auth` and `/docs`. Read claims with `authfiber.ClaimsFromContext`. Requires `MY_PROJECT_API_JWT_SECRET`. Refresh tokens are revocable: `POST /auth/logout` denylists a refresh token, and `/auth/refresh` rotates (single-use) the presented token. Revocation is backed by the required `Revoker` port — the Redis adapter (`adapter/redis`) is the production wiring, injected positionally into `auth.New` in `cmd/main` (tests pass a stub). Access tokens stay short-lived and are not individually revoked. (Health probes hit the samsara health server on its own port; the fiber component's built-in `/health` registers ahead of the middleware and stays public regardless, but the project doesn't rely on it.)

### Config
Loaded via `github.com/kelseyhightower/envconfig` with prefix `MY_PROJECT_API`. Pass `-l` to load `env/local/api.env` for local development outside Docker. All env var names follow `MY_PROJECT_API_<SECTION>_<FIELD>` (e.g. `MY_PROJECT_API_POSTGRESQL_HOST`).

### Production hardening
The defaults favor local-dev convenience. Before exposing the service publicly, tighten these in `env/<stage|prod>/api.env`:

- **CORS** — `MY_PROJECT_API_FIBER_CORS_ALLOW_ORIGINS` defaults to `*`. A wildcard origin on an authenticated API is unsafe; set explicit origins (the backend logs a startup warning while it is `*`).
- **Auth rate limiting** — login/refresh are throttled per-IP by an in-memory limiter (`internal/common/middleware`). It is per-process; for multiple backend replicas move the counter to a shared store (Redis is already wired).
- **Server timeouts** — `READ_TIMEOUT`/`WRITE_TIMEOUT`/`IDLE_TIMEOUT` default to non-zero (slowloris protection). Raise `WRITE_TIMEOUT` only if you stream large responses.
- **Postgres TLS** — `MY_PROJECT_API_POSTGRESQL_SSL_MODE` defaults to `disable` (safe only on a trusted internal network). Set `require`/`verify-full` when the DB is reached over an untrusted network.
- **Swagger UI** — leave `MY_PROJECT_API_FIBER_SWAGGER_FILE_PATH` empty in prod to disable the public `/docs` UI and `swagger.json`; when set, those routes are unauthenticated by design.

### Observability
- **Correlated logging** — `internal/common/middleware.RequestID` assigns each request an `X-Request-ID` (honouring an inbound one), echoes it, and seeds a request-scoped `*slog.Logger` (bound to `request_id`) into the context via `internal/common/logging`. The auth middleware adds `user_id`. Handlers and domain code log with `logging.From(ctx)`, so every line for a request is correlated; off-request paths fall back to `slog.Default()`.

### Docker stack
`deploy/docker-compose.yml` is the base (infra + app-service skeleton); per-environment overrides `docker-compose.{dev,stage,prod}.yml` add the build target, source mounts, ports, and healthchecks. `make up [ENVIRONMENT=…]` merges base + override (`-f base -f <env>`) and brings up the `app` profile. `dev` runs air + the vite dev server with source mounted (hot reload); `stage`/`prod` build images — backend from `Dockerfile.prod` (scratch + `/health`), frontend built and served by nginx, which proxies `/api` to the `backend` alias so the SPA stays same-origin on a relative `/api/v1` base. Container and volume names are suffixed with `$ENVIRONMENT`; services find each other via stable network aliases (`postgresql`/`rabbitmq`/`redis`/`backend`). `make run` starts infra only; `make run-local` is dev infra + host backend/frontend.

### Host port mappings
Host-side ports for the dev/local infra (Postgres, RabbitMQ, Redis) live in `env/<env>/ports.env`. The Makefile sources that file before each `docker compose` call and before the local backend `air` launch, so changing one variable shifts both the published host port AND the port the backend connects to. Edit there to coexist with other projects already holding the defaults (`5432`/`5672`/`6379`/`15672`). Container-internal ports stay standard.

### Error handling
`github.com/sunkek/mishap`. Error codes in `internal/common/e/e.go`: `NotFound`, `Conflict`, `Forbidden`, `Internal`, `Validation`, `JWT`. Wrap with `mishap.Wrap(err, "message")`. The Fiber error handler in `cmd/main/main.go` maps these codes to HTTP statuses.

## First-run checklist after cloning
1. `docker network create dev`
2. `make gen-env APP=my_project` (fills env/dev + env/local with shared secrets)
3. `cd service/backend && go mod tidy`
4. `cd service/frontend && npm install`
5. `make migrate-up` (after infra is up)
6. `make run-local`
