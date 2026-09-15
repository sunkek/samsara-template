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

Add a variable with `make env-add` rather than by hand — it writes the template
and every environment in one step, and re-encrypts whatever is committed:

```bash
make env-add FILE=api.env NAME=MY_PROJECT_API_FEATURE_X VALUE=false
make env-sync                    # after pulling someone else's new variable
```

Values that open production are encrypted in git with SOPS and age: ciphertext
under `env/sops/<env>/`, plaintext never committed, `make secrets-check` failing
if that is ever the other way round. A fresh clone plus one age key can then
bring an environment up.

```bash
make secrets-encrypt             # env/prod/ -> env/sops/prod/
make secrets-decrypt             # the other way, on another machine
```

**[`docs/SECRETS.md`](../docs/SECRETS.md) is the reference** — when a repository
wants SOPS at all, adding and removing an operator, per-environment recipients,
and why removing one means rotating. It is optional: a project that deploys
nothing can ignore it and keep using `gen-env`.

## Host ports

Host-side ports live in `env/<env>/ports.env`, sourced by the Makefile before
every `docker compose` call and before the local `air` launch. Editing one
variable therefore shifts both the published host port and the port the backend
connects to. Change them here to coexist with other projects holding the
defaults; container-internal ports stay standard.

## Deploying

This template ships no deploy pipeline, and that is deliberate — how code
reaches a host is the one thing every team already has an opinion about. What it
does ship is a stack that runs the same way everywhere, so a pipeline has little
to do:

```bash
# on the host, once
git clone <repo> && cd <repo>
make secrets-decrypt             # or place env/prod/ by hand
docker network create dev

# every release
git pull
make secrets-decrypt FORCE=1     # only if the encrypted env files changed
make migrate-up ENVIRONMENT=prod
make up ENVIRONMENT=prod
```

Order matters: **migrations run before the new image starts.** A container that
boots against a schema it expects to have been migrated will fail its
healthcheck, and `make up` will have replaced the running one by then.

Two properties worth preserving in whatever CI you wire this into:

- **Never send `env/<env>/` to the host from CI.** The host decrypts it, or it
  was placed there once by hand. A pipeline that carries plaintext secrets makes
  every CI variable and every job log part of the blast radius.
- **Keep deploy jobs uninterruptible.** A cancelled test costs nothing; a
  cancelled deploy leaves a half-migrated database.

Rolling back is `git checkout <previous tag> && make up ENVIRONMENT=prod`, which
covers code but **not** a migration that has already run — an `up` migration
that drops or rewrites data is not undone by starting the old image. Write
reversible migrations, and keep a dump from immediately before a destructive
one (below).

## Backup and restore

```bash
make pg-dump                                  # -> infra/postgresql/backup/<timestamp>.sql
make pg-dump DUMP_FILE=before-migration.sql
make pg-restore DUMP_FILE=infra/postgresql/backup/<file>.sql
```

Both act on `$ENVIRONMENT` (default `dev`), so name the environment explicitly
when it is not dev: `make pg-dump ENVIRONMENT=prod`.

`infra/postgresql/backup/` is a working directory on one host, not a backup
strategy. A dump that lives on the machine it was taken from is gone in exactly
the failure it exists for. Ship them somewhere else — object storage at a
different provider, on a schedule, with a restore you have actually run at least
once. An untested restore is a hope, not a backup.

Restore is destructive: `pg-restore` loads into the live database. Take a dump
first, even when restoring, so a bad file does not cost you both copies.

## Logs and health

```bash
make logs                        # follow every container in the stack
make ps                          # what is running
```

The backend serves `/health` on `MY_PROJECT_API_HEALTH_PORT` (default 3333,
separate from the API port so a health probe never traverses the API's
middleware) and Prometheus metrics on `/metrics` — see the hardening note below
on keeping the latter off a public interface.

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
- **Proxy trust must match the topology.** The limiter keys on `c.IP()`, which
  is the socket peer unless `MY_PROJECT_API_FIBER_TRUST_PROXY` is on. In the
  stage and prod stacks every request arrives through the frontend's nginx, so
  with trust off all clients share nginx's address: one bucket for everybody,
  and a single caller can lock the whole `/auth` group. The stage and prod
  compose files therefore set `TRUST_PROXY=true` with
  `TRUSTED_PROXIES=172.16.0.0/12`. Leave it **off** wherever the service is
  reachable directly — trusting the header without pinning the peer lets any
  caller pick its own identity. Narrow `TRUSTED_PROXIES` to your real subnet,
  and note that nginx must *overwrite* `X-Forwarded-For` (it does here): the
  backend reads the left-most entry, so an appending chain would let a client
  forge the value.
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
