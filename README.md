<!-- TEMPLATE-BANNER:START -->
> **You are looking at `samsara-template`** — a runnable reference scaffold, not a product.
> Clone it and bring the stack up to study a production-shaped Go service end to end, or run
> `./bootstrap.sh` to fork-and-rename it into your own project. `bootstrap.sh` rewrites the
> placeholder identity below (`My Project` / `my_project` / `MY_PROJECT`) to the names you
> choose and removes this banner, leaving a clean README for the new project.
<!-- TEMPLATE-BANNER:END -->

# My Project

An opinionated full-stack reference service: a Go backend organized as ports & adapters on the [samsara](https://github.com/sunkek/samsara) component supervisor, paired with a React/Vite SPA. It runs as-is — clone it and bring up the stack to see a production-shaped Go service end to end (auth + a sample `note` domain, migrations, CI, Dockerized dev/stage/prod stacks, Swagger docs). To start a fresh named project from it, run `./bootstrap.sh` to fork-and-rename.

**Stack:** Go 1.26 · Fiber · PostgreSQL · RabbitMQ · Redis · JWT auth · pgx · golang-migrate · React 19 · Vite · TypeScript · Docker Compose · GitHub Actions + GitLab CI.

## Quick start

```bash
docker network create dev                        # once
make gen-env APP=my_project                     # fills env/dev + env/local (shared secrets)
cd service/backend && go mod tidy && cd ../..
cd service/frontend && npm install && cd ../..
make run                                          # start infra
make migrate-up                                   # apply migrations
make run-local                                    # backend (live reload) + frontend on host
```

Prefer everything in Docker? After `go mod tidy`, run `make up` — it starts the
full stack (infra + backend + frontend) for the chosen environment:

```bash
make up                      # dev: hot reload (air + vite), source mounted
make up ENVIRONMENT=stage    # built images, frontend served by nginx (needs env/stage)
make up ENVIRONMENT=prod     # built images, prod config        (needs env/prod)
```

`dev` shares its secrets with `local` (so `run-local` works too); generate
stage/prod separately for distinct secrets: `make gen-env GEN_ENVS=prod APP=…`.

- Backend API: http://localhost:8000/api/v1
- Swagger UI: http://localhost:8000/api/v1/docs
- Auth: `POST /api/v1/auth/register`, `/auth/login`, `/auth/refresh` (JWT). Login returns an access + refresh token pair.
- Sample domain: `note` — `POST /api/v1/notes`, `GET /api/v1/notes`, `GET /api/v1/notes/:id`. Protected: send `Authorization: Bearer <access_token>`.

## Layout

```
service/backend    # Go service (cmd/main, internal/domain, internal/common)
service/frontend   # React + Vite SPA
deploy             # docker-compose, postgres/rabbitmq/redis config, migrations
env                # per-environment env files (example/ is the template)
```

See `CLAUDE.md` for commands and architecture, `AGENTS.md` for coding/commit
conventions, and `docs/ARCHITECTURE.md` for the domain pattern.
