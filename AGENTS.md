# Repository Guidelines

See `CLAUDE.md` for project overview, commands, and architecture.

## Coding Style

- Language: Go (`go 1.26` module at `services/backend/go.mod`).
- Run `gofmt` before committing. No style linter beyond standard `go vet`.
- Exported identifiers: `CamelCase`. Internal helpers: `camelCase`. Package names: short lowercase.
- Config env var names: `UPPER_SNAKE_CASE`.
- One use case per file: `usecase_<verb>.go`.

## Testing

- Place tests next to code as `*_test.go`. Prefer table-driven tests.
- Run from `services/backend`: `go test ./...`
- Integration tests live in `internal/integration` behind the `integration` build tag, so they are excluded from the default `go test ./...`. Run them with `make test-integration` (needs dev infra up + migrations applied).
- CI runs on push/PR (`.github/workflows/ci.yml`, mirrored in `.gitlab-ci.yml`): backend `gofmt`/`go vet`/`go build`/`go test`, frontend `npm run lint`/`build`. Keep both CI files in sync.

<!-- feat:if template -->
## Feature markers (template only)

Files carry `feat:if <expr>` / `feat:else` / `feat:end` comments in their own
comment syntax; `scripts/apply_features.sh` renders them for a chosen feature
set (`backend`, `frontend`, `postgresql`, `redis`, `rabbitmq`). Rules when
editing:

- The checked-in template is the **all-features** build and must run as-is. Code
  that only exists when a feature is *off* lives behind a `~` comment leader
  (`//~`, `#~`), which the renderer uncomments.
- Markers are only recognized at the start of a line; mention them mid-sentence
  in prose freely.
- Whole files/dirs that belong to one feature are listed in the prune table in
  `scripts/apply_features.sh` instead of being marked up.
- Add a combination to the `feature-matrix` CI job when you add a feature.
- `scripts/features_test.sh` unit-tests the renderer; extend it when you change
  marker semantics.
<!-- feat:end -->

## Commit Format

`type(scope): short summary` — e.g. `feat(note): add tag filtering to list endpoint`

Keep commits atomic. Include migration files in the same commit as the Go code that requires them.
