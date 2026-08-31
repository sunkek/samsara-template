# Operations

## Docker stacks

`infra/docker-compose.yml` is the base stack; `docker-compose.{dev,stage,prod}.yml`
override it with the build target, source mounts, ports and healthchecks.
`make up [ENVIRONMENT=…]` merges base + override (`-f base -f <env>`) and brings
up the `app` profile. Container and volume names carry the `$ENVIRONMENT`
suffix; services find each other through stable network aliases, never through
container names.

- `dev` mounts source for hot reload.
<!-- feat:if backend -->
- `stage` / `prod` build the backend from `Dockerfile.prod` (scratch + `/health`).
<!-- feat:end -->
<!-- feat:if frontend -->
- `stage` / `prod` build the frontend as nginx serving `dist/` and proxying
  `/api` to the `backend` alias, so the SPA stays same-origin on a relative
  `/api/v1` base.
<!-- feat:end -->

## Env files and secrets

`make gen-env APP=<name>` materializes `env/<env>/*.env` from `env/example/`,
replacing placeholders with random secrets. `GEN_ENVS` (default `dev local`)
picks which environments a run materializes, and every environment in one run
**shares one secret pool** — that is deliberate, because `make run-local`
brings infra up from `env/dev` while loading the backend from `env/local`.
Generate stage and prod in separate invocations so they get distinct secrets:

```bash
make gen-env GEN_ENVS=prod APP=my_project
```

The same logical credential lives in two files (the Postgres app password is
`APP_PASSWORD` in `postgresql.env` and `_API_POSTGRESQL_PASS` in `api.env`), so
secrets are mapped by variable name *and* file. Get that wrong and the backend
fails to authenticate against its own infra (SQLSTATE 28P01).

## Host ports

Host-side ports live in `env/<env>/ports.env`, sourced by the Makefile before
every `docker compose` call and before the local `air` launch. Editing one
variable therefore shifts both the published host port and the port the backend
connects to. Change them here to coexist with other projects holding the
defaults; container-internal ports stay standard.

## Production hardening

The defaults favour local-dev convenience. Tighten these before exposing the
service publicly.
<!-- feat:if backend -->

In `env/<stage|prod>/api.env`:

- **CORS** — `MY_PROJECT_API_FIBER_CORS_ALLOW_ORIGINS` defaults to `*`. A
  wildcard origin on an authenticated API is unsafe; set explicit origins. The
  backend logs a startup warning while it is `*`.
- **Auth rate limiting** — the whole `/auth` group (register, login, refresh,
  logout) is throttled per-IP by an in-memory limiter
  (`internal/common/middleware`), which is per-process. With more than one
  backend replica, move the counter to a shared store (Redis is already wired).
- **`/metrics` is public** — it is in the auth middleware's public-prefix list in
  `cmd/main/main.go`, so Prometheus can scrape it without a token. That exposes
  request paths, volumes and error rates to anyone who can reach the port. Keep
  it off the public interface, or drop the prefix from the list and give your
  scraper a token.
- **Server timeouts** — `READ_TIMEOUT` / `WRITE_TIMEOUT` / `IDLE_TIMEOUT`
  default to non-zero for slowloris protection. Raise `WRITE_TIMEOUT` only to
  stream large responses.
<!-- feat:if postgresql -->
- **Postgres TLS** — `MY_PROJECT_API_POSTGRESQL_SSL_MODE` defaults to `disable`,
  safe only on a trusted internal network. Use `require` / `verify-full` when
  the DB is reached over an untrusted one.
<!-- feat:end -->
- **Swagger UI** — leave `MY_PROJECT_API_FIBER_SWAGGER_FILE_PATH` empty in prod
  to drop the public `/docs` UI and `swagger.json`; when set, those routes are
  unauthenticated by design.
<!-- feat:end -->
<!-- feat:if frontend -->
- **CSP** — `.docker/nginx.conf` ships a same-origin CSP plus baseline security
  headers. Widen it only for the third-party assets the page actually loads
  (fonts, analytics, embeds).
<!-- feat:end -->
