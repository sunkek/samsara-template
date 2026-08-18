<!-- TEMPLATE-BANNER:START -->
> **You are looking at `samsara-template`** — a runnable reference scaffold, not a product.
> Clone it and bring the stack up to study a production-shaped Go service end to end, or run
> `./bootstrap.sh` to fork-and-rename it into your own project. `bootstrap.sh` rewrites the
> placeholder identity below (`My Project` / `my_project` / `MY_PROJECT`) to the names you
> choose and removes this banner, leaving a clean README for the new project.
>
> **Pick what you keep.** `bootstrap.sh -F <features>` cuts the fork down to the
> parts you want — `backend`, `frontend`, `postgresql`, `redis`, `rabbitmq` — deleting
> the adapters, compose services, env vars, migrations and CI jobs of everything
> else. See *Choosing features* below.
<!-- TEMPLATE-BANNER:END -->

# My Project

<!-- feat:if backend -->
An opinionated full-stack reference service: a Go backend organized as ports & adapters on the [samsara](https://github.com/sunkek/samsara) component supervisor, paired with a React/Vite SPA. It runs as-is — clone it and bring up the stack to see a production-shaped Go service end to end (auth + a sample `note` domain, migrations, CI, Dockerized dev/stage/prod stacks, Swagger docs). To start a fresh named project from it, run `./bootstrap.sh` to fork-and-rename.
<!-- feat:end -->
<!-- feat:if frontend,!backend -->
<!--~ A React + Vite + TypeScript single-page app, with the Docker/nginx production image, compose stacks and CI wiring already in place. Run it with `make run-local`; deploy it with `make up ENVIRONMENT=prod`. -->
<!-- feat:end -->
<!-- feat:if !backend,!frontend -->
<!--~ An infrastructure-only scaffold: Docker Compose stacks, per-environment env-file generation and CI, with no application service yet. Add yours under `services/`. -->
<!-- feat:end -->

**Stack:**
<!-- feat:if backend -->
Go 1.26 · Fiber ·
<!-- feat:end -->
<!-- feat:if postgresql -->
PostgreSQL ·
<!-- feat:end -->
<!-- feat:if rabbitmq -->
RabbitMQ ·
<!-- feat:end -->
<!-- feat:if redis -->
Redis ·
<!-- feat:end -->
<!-- feat:if postgresql -->
JWT auth · pgx · golang-migrate ·
<!-- feat:end -->
<!-- feat:if frontend -->
React 19 · Vite · TypeScript ·
<!-- feat:end -->
Docker Compose · GitHub Actions + GitLab CI.

<!-- feat:if template -->
## Choosing features

Nothing here is mandatory. `bootstrap.sh` takes a feature list and physically
removes everything else from the fork — there are no dead adapters, no unused
compose services, and no env vars for infra you do not run:

```bash
./bootstrap.sh -d ../api  -F backend,postgresql          # JSON API on Postgres, nothing else
./bootstrap.sh -d ../spa  -F frontend                    # just the React SPA
./bootstrap.sh -d ../full                                # everything (the default)
```

| Feature | Dropping it means |
|---|---|
| `backend` | no Go service: `services/backend`, its CI jobs and its compose service go |
| `frontend` | no SPA: `services/frontend`, its CI jobs and its compose service go |
| `postgresql` | no persistence — and the sample domains (`auth`, `note`, `notestats`) go with it, leaving a supervisor + HTTP skeleton to build on |
| `redis` | `note` reads stop being cache-aside (`NoopCache`) and `auth` swaps its token denylist for the in-memory one (single-process only) |
| `rabbitmq` | `note` stops publishing events (`NoopEvents`) and the `notestats` read model is removed |

Under the hood this is `scripts/apply_features.sh`, which you can also run on a
checkout directly (`scripts/apply_features.sh -f backend,postgresql`). The
template itself is the all-features build: files carry `feat:if` marker comments
in their own comment syntax, so an unrendered checkout still runs as-is.
<!-- feat:end -->

## Quick start

<!-- feat:if backend -->
`make run-local` runs the backend with [air](https://github.com/air-verse/air)
(live reload), which regenerates Swagger docs with
[swag](https://github.com/swaggo/swag) on each build — install both once:

```bash
go install github.com/air-verse/air@latest
go install github.com/swaggo/swag/v2/cmd/swag@latest
```

<!-- feat:end -->
```bash
docker network create dev                        # once
make gen-env APP=<your_app_name>                # fills env/dev + env/local (shared secrets)
<!-- feat:if backend -->
cd services/backend && go mod tidy && cd ../..
<!-- feat:end -->
<!-- feat:if frontend -->
cd services/frontend && npm install && cd ../..
<!-- feat:end -->
<!-- feat:if postgresql|redis|rabbitmq -->
make run                                          # start infra
<!-- feat:end -->
<!-- feat:if postgresql -->
make migrate-up                                   # apply migrations
<!-- feat:end -->
make run-local                                    # dev servers on the host
```

Prefer everything in Docker? Run `make up` — it starts the full stack for the
chosen environment:

```bash
make up                      # dev: hot reload, source mounted
make up ENVIRONMENT=stage    # built images, frontend served by nginx (needs env/stage)
make up ENVIRONMENT=prod     # built images, prod config        (needs env/prod)
```

`dev` shares its secrets with `local` (so `run-local` works too); generate
stage/prod separately for distinct secrets: `make gen-env GEN_ENVS=prod APP=…`.

<!-- feat:if frontend -->
- Frontend: http://localhost:5173 (dev server) or the published nginx port under `stage`/`prod`.
<!-- feat:end -->

<!-- feat:if backend -->
- Backend API: http://localhost:8000/api/v1
- Swagger UI: http://localhost:8000/api/v1/docs
<!-- feat:if postgresql -->
- Auth: `POST /api/v1/auth/register`, `/auth/login`, `/auth/refresh` (JWT). Login returns an access + refresh token pair.
- Sample domain: `note` — `POST /api/v1/notes`, `GET /api/v1/notes`, `GET /api/v1/notes/:id`. Protected: send `Authorization: Bearer <access_token>`.
<!-- feat:end -->
<!-- feat:if rabbitmq -->
- Read model: `GET /api/v1/stats`, projected from `note.created` events.
<!-- feat:end -->
<!-- feat:end -->

## Layout

```
<!-- feat:if backend -->
services/backend    # Go service (cmd/main, internal/domain, internal/common)
<!-- feat:end -->
<!-- feat:if frontend -->
services/frontend   # React + Vite SPA (src/, .docker/ for the nginx image)
<!-- feat:end -->
infra               # docker-compose stacks (+ infra config and migrations)
env                 # per-environment env files (example/ is the template)
```

(Whatever you deselected at bootstrap time is simply absent.)

See `CLAUDE.md` for commands and architecture, and `AGENTS.md` for
coding/commit conventions.
<!-- feat:if backend -->
`docs/ARCHITECTURE.md` covers the domain pattern.
<!-- feat:end -->
