# Contributing

Thanks for considering a contribution. This template aims to stay small, opinionated, and easy to fork — please keep changes aligned with that goal.

## Ground rules

- Coding style, testing rules, and commit format live in [`AGENTS.md`](AGENTS.md). Read it first.
- Architecture and command reference live in [`CLAUDE.md`](CLAUDE.md) and [`docs/ARCHITECTURE.md`](docs/ARCHITECTURE.md).
- The `note` domain is the canonical example of the vertical-slice pattern. New domains should mirror its structure.

## Getting set up

1. `docker network create dev` (once)
2. `make gen-env APP=<your-app-name>`
3. `cd service/backend && go mod tidy`
4. `cd service/frontend && npm install`
5. `make run && make migrate-up && make run-local`

See [`README.md`](README.md) for the Docker-only path (`make up`).

## Before you open a PR

- `cd service/backend && gofmt -l . && go vet ./... && go test ./...` — all clean.
- `cd service/frontend && npm run lint && npm run build` — all clean.
- If you touched API handlers, regenerate Swagger: `make gen-api-docs`.
- If you added a migration, include it in the same commit as the Go code that needs it.
- Update `CLAUDE.md` / `docs/ARCHITECTURE.md` if you changed the architecture or added a cross-cutting convention.
- Keep both CI files (`.github/workflows/ci.yml` and `.gitlab-ci.yml`) in sync.

## Scope of changes welcome

- Bug fixes and security hardening — always welcome.
- Small DX improvements (Makefile targets, scripts, docs).
- New samsara components or adapter examples, if they generalize cleanly.

Changes that grow the template's surface area (new mandatory dependencies, framework swaps, opinionated business logic beyond the sample domain) are unlikely to be merged — fork instead.

## Reporting issues

Include: what you ran, what you expected, what happened, and the relevant logs (`make logs` or the failing CI job). For security issues, follow [`SECURITY.md`](SECURITY.md) — email the maintainer instead of opening a public issue.
