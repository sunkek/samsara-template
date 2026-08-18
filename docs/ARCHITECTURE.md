# Architecture

## Layers

```
cmd/main          # composition root: build config, register samsara components, wire domains
internal/common   # cross-cutting: config (envconfig), error codes (mishap)
internal/domain   # one package per bounded context
```

## A domain, end to end

Using the sample `note` domain as the reference shape:

```
internal/domain/note/
  domain.go              # Domain struct; New(db) returns it. Implements the Service port.
  interface.go           # Service (inbound) + DB (outbound) ports
  usecase_create.go      # business logic, one verb per file
  usecase_list.go
  usecase_get.go
  model/note.go          # entity + input structs (no framework imports)
  adapter/fiber/         # REST adapter: takes Service, registers routes directly
  adapter/postgresql/    # DB adapter: SQL via samsara-components/postgresql
```

### Ports and dependency direction

Each domain declares two ports in `interface.go`:

- **`Service`** (inbound) — the use cases the REST adapter calls. `*Domain`
  implements it (`var _ Service = (*Domain)(nil)` asserts this at compile time).
- **`DB`** (outbound) — the persistence the domain needs. The postgresql
  adapter implements it.

Dependency direction: `adapter/fiber → Service ← Domain → DB ← adapter/postgresql`.
The REST adapter imports the domain; the domain imports neither adapter. Both
adapters depend on `model`. `cmd/main` depends on everything and wires it.
Adapters never import each other.

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

## Auth

`internal/domain/auth` is a full sample auth domain: register, login, refresh,
and JWT verify. Its fiber adapter also exposes `Middleware(publicPrefixes...)`,
registered in `cmd/main` via `fiberCmp.Use(...)` to guard every route except the
public prefixes (`/auth`, `/docs`, `/health`). Verified claims land in
`ctx.Locals`; read them with `authfiber.ClaimsFromContext`. Tokens are HS256,
signed with `JWT_SECRET`; passwords are bcrypt-hashed.

## Adding a domain

1. Copy `internal/domain/note` to `internal/domain/<name>`.
2. Rename the package and types; adjust the `Service`/`DB` ports and the routes
   in `adapter/fiber`.
3. Add a migration: `make migrate-new n=create_<name>`.
4. Wire it in `cmd/main/main.go` as DB adapter → domain → REST adapter
   (construct dependency-free domains first).
