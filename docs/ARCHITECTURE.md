# Architecture

Go module path: `github.com/sunkek/samsara-template/backend` — use it for all
internal imports.

## Layers

```
cmd/main          # composition root: build config, register samsara components, wire domains
internal/common   # cross-cutting: config (envconfig), error codes (mishap), middleware, logging
internal/domain   # one package per bounded context
```

## Runtime: the samsara supervisor

`github.com/sunkek/samsara` is a component supervisor. Every piece of infra the
build carries, plus the Fiber HTTP server, registers as a component with a tier
(`Critical` / `Significant`) and a restart policy. Fiber declares its infra
dependencies via `WithDependencies`, so it starts after them. `main()` blocks on
`<-ctx.Done()`.

## A domain, end to end

<!-- feat:if postgresql -->
Using the sample `note` domain as the reference shape:

```
internal/domain/note/
  domain.go              # Domain struct; New(db, …) returns it. Implements the Service port.
  interface.go           # Service (inbound) + DB (outbound) ports
  usecase_create.go      # business logic, one verb per file
  usecase_list.go
  usecase_get.go
  model/note.go          # entity + input structs (no framework imports)
  adapter/fiber/         # REST adapter: takes Service, registers routes directly
  adapter/postgresql/    # DB adapter: SQL via samsara-components/postgresql
```
<!-- feat:end -->
<!-- feat:if !postgresql -->
<!--~ The shape of a domain package: -->
<!--~  -->
<!--~ ``` -->
<!--~ internal/domain/<name>/ -->
<!--~   domain.go              # Domain struct; New(db, …) returns it. Implements the Service port. -->
<!--~   interface.go           # Service (inbound) + DB (outbound) ports -->
<!--~   usecase_create.go      # business logic, one verb per file -->
<!--~   model/<name>.go        # entity + input structs (no framework imports) -->
<!--~   adapter/fiber/         # REST adapter: takes Service, registers routes directly -->
<!--~ ``` -->
<!-- feat:end -->

### Ports and dependency direction

Each domain declares two ports in `interface.go`:

- **`Service`** (inbound) — the use cases the REST adapter calls. `*Domain`
  implements it (`var _ Service = (*Domain)(nil)` asserts this at compile time).
- **`DB`** (outbound) — the persistence the domain needs. The postgresql
  adapter implements it.

Dependency direction: `adapter/fiber → Service ← Domain → DB ← adapter/postgresql`.
The REST adapter imports the domain; the domain imports neither adapter. Both
adapters depend on `model`. `cmd/main` depends on everything and wires it.
Adapters never import each other. Cross-domain calls go through interfaces
declared in `interface.go` and injected in `cmd/main/main.go`.

This replaces the old handler-injection pattern (`SetHandlerX` + nil guards):
routes are registered with a live handler the moment the adapter is built, so a
missing wire is a compile error, not a runtime nil.

## Request flow

```
HTTP → fiber adapter handler → Service (Domain method)
     → DB interface → postgresql adapter → Postgres
```

<!-- feat:if template -->
## Optional infrastructure

Redis and RabbitMQ are outbound ports, never hard dependencies: `note` declares
`Cache` and `Events`, and `auth` declares `Revoker`. Each has a stand-in — the
domain's `NoopCache`/`NoopEvents`, and `auth/adapter/memory` for the revoker —
so a build without that infra keeps the same call sites and loses only the
capability. That is what makes `bootstrap.sh -F` a deletion rather than a
rewrite: `cmd/main` picks a different adapter, and nothing else moves.

Postgres is the exception. The sample domains exist to demonstrate persistence,
so a build without it keeps the supervisor, the HTTP server, logging, metrics
and the error mapping, and you add your own first domain.
<!-- feat:end -->

<!-- feat:if postgresql -->
## Sample domains

**`note`** — the full vertical slice.
<!-- feat:if redis -->
Reads are **cache-aside** through a `Cache` port (`adapter/redis`): `Get`/`List`
serve from cache on a hit and populate it on a miss; `Create` warms the item and
invalidates the list. Caching is best-effort — a cache error is logged and falls
back to the DB, never failing the request — and the TTL is
`MY_PROJECT_API_NOTE_CACHE_TTL`. Pass `note.NoopCache{}` to disable it. Hit/miss
metrics are recorded by the Redis adapter rather than by the use cases, so a
build without a cache reports nothing instead of a permanent 0% hit rate.
<!-- feat:end -->
<!-- feat:if rabbitmq -->
`Create` also publishes a `note.created` event through an `Events` port
(`adapter/rabbitmq`) to a topic exchange, best-effort; `note.NoopEvents{}`
disables it.
<!-- feat:end -->

<!-- feat:if rabbitmq -->
**`notestats`** — a read model (CQRS-lite) projecting `note.created` events into
the single-row `note_stats` table. The samsara rabbitmq component owns the
consume loop; `notestats/adapter/rabbitmq` is the message handler,
`adapter/postgresql` the projection store, and `adapter/fiber` exposes
`GET /api/v1/stats`. Event, exchange and queue names live under
`MY_PROJECT_API_EVENTS_*`. This is the end-to-end async demo: note create,
publish, broker, consumer, projection, `/stats`.
<!-- feat:end -->

## Auth

`internal/domain/auth` is a full sample auth domain: register, login, refresh,
logout, and JWT verify. Its fiber adapter exposes `Middleware(publicPrefixes...)`,
registered in `cmd/main` via `fiberCmp.Use(...)` to guard every route except the
public prefixes (`/auth`, `/docs`, `/metrics`). Read verified claims with
`authfiber.ClaimsFromContext`. Tokens are HS256 signed with
`MY_PROJECT_API_JWT_SECRET`; passwords are bcrypt-hashed.

Refresh tokens are revocable: `POST /auth/logout` denylists one, and
`/auth/refresh` rotates (single-use) the token presented. The rotation is one
atomic `Claim` on the `Revoker` port — Redis `SET NX`, or an insert under the
memory adapter's lock — so two concurrent presentations of a stolen token yield
exactly one winner. Checking and then revoking in two steps would let both pass
the check before either write landed, which is the replay the denylist exists to
prevent. Revocation is backed
by the required `Revoker` port —
<!-- feat:if redis -->
the Redis adapter (`adapter/redis`) is the production wiring,
<!-- feat:end -->
<!-- feat:if !redis -->
<!--~ `adapter/memory` is the wiring (single-process only), -->
<!-- feat:end -->
injected positionally into `auth.New` in `cmd/main`; tests pass a stub. Access
tokens stay short-lived and are not individually revoked.

Health probes hit the samsara health server on its own port. The fiber
component's built-in `/health` registers ahead of the middleware and stays
public regardless, but nothing here relies on it.

## Adding a domain

1. Copy `internal/domain/note` to `internal/domain/<name>`.
2. Rename the package and types; adjust the `Service`/`DB` ports and the routes
   in `adapter/fiber`.
3. Add a migration: `make migrate-new n=create_<name>`.
4. Wire it in `cmd/main/main.go` as DB adapter → domain → REST adapter
   (construct dependency-free domains first).
<!-- feat:end -->

## Config

Loaded via `github.com/kelseyhightower/envconfig` under the prefix
`MY_PROJECT_API`, so every variable is `MY_PROJECT_API_<SECTION>_<FIELD>` (e.g.
`MY_PROJECT_API_POSTGRESQL_HOST`). Pass `-l` to load `env/local/api.env` when
running outside Docker.

## Error handling

`github.com/sunkek/mishap`. Codes live in `internal/common/e/e.go`: `NotFound`,
`Conflict`, `Forbidden`, `Internal`, `Validation`, `JWT`, `RateLimit`,
`Unauthorized`. `Unauthorized` (401) is a failed authentication — bad
credentials; `Forbidden` (403) is an authenticated caller denied access. Wrap
with
`mishap.Wrap(err, "message")`.

`e.HTTPStatus` is the single source of truth for the code-to-status mapping,
shared by the Fiber error handler in `cmd/main/main.go` and the metrics
middleware. A code with no case there falls through as 500, which silently turns
a client error into a server one — so add the case in the same commit as the
code, and extend the exhaustive table test in `e_test.go`.

## Correlated logging

`internal/common/middleware.RequestID` assigns each request an `X-Request-ID`
(honouring an inbound one), echoes it, and seeds a request-scoped `*slog.Logger`
bound to `request_id` into the context via `internal/common/logging`. The auth
middleware adds `user_id`. Log with `logging.From(ctx)` so every line for a
request is correlated; off-request paths fall back to `slog.Default()`.
