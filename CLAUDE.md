# CLAUDE.md

Guidance for Claude Code (claude.ai/code) working in this repository.

## What this is

My Project is built on the `samsara` component supervisor. Which pieces it has —
Go backend, React/Vite SPA, Postgres, Redis, RabbitMQ — you can read off
`services/` and `infra/` faster than this file could tell you, and unlike this
file they cannot go stale.

<!-- feat:if postgresql -->
The domains under `internal/domain/` are samples demonstrating the vertical
slice. Replace them with your own; they exist to be read and then deleted.
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
  ports, optional infra, token revocation, hand-written SQL, and the sample-domain
  split. Read the relevant one before changing any of those.
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
<!-- feat:if frontend,backend -->
- The dev-server proxy target lives in `services/frontend/config/devProxy.ts`, not
  in `vite.config.ts`, so it can be tested. Node-side config code belongs under
  `config/` (typed by `tsconfig.node.json`); `src/` is browser code.
<!-- feat:end -->
- Host-side ports live in `env/<env>/ports.env`, not in the compose files.
- Both CI files (`.github/workflows/ci.yml`, `.gitlab-ci.yml`) must stay in sync
  on what they check. The template-maintenance jobs (`feature-renderer`,
  `feature-matrix`, CodeQL) are the exception: they exist on GitHub only, because
  the template itself is hosted there.

## First run after cloning

`README.md` has the setup sequence and both ways to run the stack (host dev
servers via `make run-local`, or everything in Docker via `make up`). Follow it
rather than reconstructing the steps.
