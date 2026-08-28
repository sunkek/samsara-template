# CLAUDE.md

Guidance for Claude Code (claude.ai/code) working in this repository.

## What this is

<!-- feat:if backend,frontend -->
My Project is a backend + frontend scaffold: a Go service on the `samsara`
component supervisor, plus a React/Vite SPA.
<!-- feat:end -->
<!-- feat:if backend,!frontend -->
<!--~ My Project is a Go service built on the `samsara` component supervisor. -->
<!-- feat:end -->
<!-- feat:if frontend,!backend -->
<!--~ My Project is a React/Vite single-page app with its Docker, compose and CI wiring in place. -->
<!-- feat:end -->
<!-- feat:if !backend,!frontend -->
<!--~ My Project is an infrastructure-only scaffold: compose stacks, env generation and CI, with no application service yet. Add yours under `services/`. -->
<!-- feat:end -->
<!-- feat:if postgresql -->
It ships one sample domain (`note`) demonstrating the full vertical slice;
replace it with your own domains.
<!-- feat:end -->

## Where things are documented

<!-- feat:if backend -->
- **`docs/ARCHITECTURE.md`** — the domain pattern: layers, ports, dependency
  direction, sample domains, auth, config, error codes, correlated logging.
  Read it before adding or changing a domain, an adapter or a use case.
<!-- feat:end -->
- **`infra/OPERATIONS.md`** — compose stacks, env-file generation and secret
  sharing, host ports, production hardening. Read it before touching `infra/`,
  `env/`, or deploy config.
<!-- feat:if template -->
- **`docs/FEATURES.md`** — feature-marker syntax and the rules for editing a
  marked file. Read it before editing any file carrying `feat:if` comments.
<!-- feat:end -->
- **`CONTEXT.md`** — the glossary. Read it before naming a new type, package or
  concept, and use its words in code, comments and commits.
<!-- feat:if backend -->
- **`docs/adr/`** — why the shape is what it is: feature rendering, domain-owned
  ports, no-op ports for optional infra, token revocation, hand-written SQL.
  Read the relevant one before changing any of those.
<!-- feat:end -->
- **`AGENTS.md`** — coding style, testing, commit format.
- **`make help`** — every Make target, always current. Prefer it over asking.

## Conventions no lookup will tell you

<!-- feat:if backend -->
- Go module path is `github.com/sunkek/samsara-template/backend`; use it for all
  internal imports.
- One use case per file: `usecase_<verb>.go`, a method on `*Domain`.
- `make run-local` needs two host tools installed once:
  `go install github.com/air-verse/air@latest` and
  `go install github.com/swaggo/swag/v2/cmd/swag@latest`.
- Errors carry a `mishap` code from `internal/common/e`; the Fiber error handler
  in `cmd/main/main.go` maps codes to HTTP statuses. An unmapped code is a 500.
- Log with `logging.From(ctx)`, never `slog.Default()`, inside request handling —
  that is what keeps `request_id` and `user_id` on every line.
<!-- feat:end -->
- Host-side ports live in `env/<env>/ports.env`, not in the compose files.
- Both CI files (`.github/workflows/ci.yml`, `.gitlab-ci.yml`) must stay in sync.

## First run after cloning

```bash
docker network create dev                    # once
make gen-env APP=<your_app_name>             # fills env/dev + env/local
<!-- feat:if backend -->
cd services/backend && go mod tidy && cd ../..
<!-- feat:end -->
<!-- feat:if frontend -->
cd services/frontend && npm install && cd ../..
<!-- feat:end -->
<!-- feat:if postgresql|redis|rabbitmq -->
make run                                     # start infra
<!-- feat:end -->
<!-- feat:if postgresql -->
make migrate-up
<!-- feat:end -->
make run-local
```
