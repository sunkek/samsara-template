# Contributing

<!-- feat:if template -->
Thanks for considering a contribution. This template aims to stay small, opinionated, and easy to fork — please keep changes aligned with that goal.
<!-- feat:end -->
<!-- feat:if !template -->
<!--~ Thanks for contributing. This file is the short version of how work lands here; adjust it to match how your team actually operates. -->
<!-- feat:end -->

## Ground rules

- Coding style, testing rules, and commit format live in [`AGENTS.md`](AGENTS.md). Read it first.
<!-- feat:if backend -->
- Architecture and command reference live in [`CLAUDE.md`](CLAUDE.md) and [`docs/ARCHITECTURE.md`](docs/ARCHITECTURE.md).
<!-- feat:end -->
<!-- feat:if !backend -->
<!--~ - Architecture and command reference live in [`CLAUDE.md`](CLAUDE.md). -->
<!-- feat:end -->
<!-- feat:if postgresql -->
- The `note` domain is the canonical example of the vertical-slice pattern. New domains should mirror its structure.
<!-- feat:end -->

## Getting set up

- `docker network create dev` (once)
- `make gen-env APP=<your-app-name>`
<!-- feat:if backend -->
- `cd services/backend && go mod tidy`
<!-- feat:end -->
<!-- feat:if frontend -->
- `cd services/frontend && npm install`
<!-- feat:end -->
<!-- feat:if postgresql -->
- `make run && make migrate-up && make run-local`
<!-- feat:end -->
<!-- feat:if !postgresql -->
<!--~ - `make run-local` -->
<!-- feat:end -->

See [`README.md`](README.md) for the Docker-only path (`make up`).

## Before you open a PR

<!-- feat:if backend -->
- `cd services/backend && gofmt -l . && go vet ./... && go test ./...` — all clean.
<!-- feat:end -->
<!-- feat:if frontend -->
- `cd services/frontend && npm run lint && npm run build` — all clean.
<!-- feat:end -->
<!-- feat:if backend -->
- If you touched API handlers, regenerate Swagger: `make gen-api-docs`.
<!-- feat:end -->
<!-- feat:if postgresql -->
- If you added a migration, include it in the same commit as the Go code that needs it.
<!-- feat:end -->
<!-- feat:if backend -->
- Update `CLAUDE.md` / `docs/ARCHITECTURE.md` if you changed the architecture or added a cross-cutting convention.
<!-- feat:end -->
<!-- feat:if !backend -->
<!--~ - Update `CLAUDE.md` if you changed the architecture or added a cross-cutting convention. -->
<!-- feat:end -->
- Keep both CI files (`.github/workflows/ci.yml` and `.gitlab-ci.yml`) in sync.

<!-- feat:if template -->
## Scope of changes welcome

- Bug fixes and security hardening — always welcome.
- Small DX improvements (Makefile targets, scripts, docs).
- New samsara components or adapter examples, if they generalize cleanly.

Changes that grow the template's surface area (new mandatory dependencies, framework swaps, opinionated business logic beyond the sample domain) are unlikely to be merged — fork instead.
<!-- feat:end -->

## Reporting issues

Include: what you ran, what you expected, what happened, and the relevant logs (`make logs` or the failing CI job). For security issues, follow [`SECURITY.md`](SECURITY.md) — email the maintainer instead of opening a public issue.
