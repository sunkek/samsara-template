# Contributing

<!-- feat:if template -->
Thanks for considering a contribution. This template aims to stay small, opinionated, and easy to fork — please keep changes aligned with that goal.
<!-- feat:end -->
<!-- feat:if !template -->
<!--~ Thanks for contributing. This file is the short version of how work lands here; adjust it to match how your team actually operates. -->
<!-- feat:end -->

## Ground rules

- Coding style, testing rules, and commit format live in [`AGENTS.md`](AGENTS.md). Read it first.
- [`CLAUDE.md`](CLAUDE.md) maps the docs; `make help` lists every command.
- [`CONTEXT.md`](CONTEXT.md) is the glossary. Name things with its words, and extend it when you introduce a concept it does not cover.
- Compose stacks, env files and production hardening live in [`infra/OPERATIONS.md`](infra/OPERATIONS.md).
<!-- feat:if backend -->
- Architecture lives in [`docs/ARCHITECTURE.md`](docs/ARCHITECTURE.md), and the reasoning behind its shape in [`docs/adr/`](docs/adr/). The `note` domain is the canonical vertical slice; new domains mirror it.
<!-- feat:end -->

## Getting set up

Follow the quick start in [`README.md`](README.md) — it has the host-dev path
(`make run-local`) and the all-in-Docker path (`make up`), and it stays correct
for whichever pieces this project was built with.

## Before you open a PR

Everything CI enforces, run locally first. The CI definitions
(`.github/workflows/ci.yml` and `.gitlab-ci.yml`) are the authority on what that
is, and the two must stay in sync.

<!-- feat:if backend -->
- `cd services/backend && gofmt -l . && go vet ./... && go test ./...` — all clean.
- If you touched API handlers, regenerate Swagger: `make gen-api-docs`.
- Update `docs/ARCHITECTURE.md` if you changed the architecture, and `CLAUDE.md` only if you added a convention no lookup would reveal.
- Add an ADR under `docs/adr/` if the change is hard to reverse, surprising without context, and the result of a real trade-off. Otherwise skip it.
<!-- feat:end -->
<!-- feat:if frontend -->
- `cd services/frontend && npm run lint && npm test && npm run build` — all clean.
<!-- feat:end -->
<!-- feat:if postgresql -->
- If you added a migration, include it in the same commit as the Go code that needs it.
<!-- feat:end -->
<!-- feat:if !backend -->
<!--~ - Update `CLAUDE.md` if you added a convention no lookup would reveal. -->
<!-- feat:end -->

<!-- feat:if template -->
## Scope of changes welcome

- Bug fixes and security hardening — always welcome.
- Small DX improvements (Makefile targets, scripts, docs).
- New samsara components or adapter examples, if they generalize cleanly.

Changes that grow the template's surface area (new mandatory dependencies, framework swaps, opinionated business logic beyond the sample domain) are unlikely to be merged — fork instead.
<!-- feat:end -->

## Reporting issues

Include: what you ran, what you expected, what happened, and the relevant logs (`make logs` or the failing CI job). For security issues, follow [`SECURITY.md`](SECURITY.md) — email the maintainer instead of opening a public issue.
