# Repository Guidelines

See `CLAUDE.md` for the project overview and the map of the other docs, and
`CONTEXT.md` for the glossary — naming follows it.

## Coding Style

<!-- feat:if backend -->
- Language: Go (`go 1.26` module at `services/backend/go.mod`).
- Run `gofmt` before committing. No style linter beyond standard `go vet`.
- Exported identifiers: `CamelCase`. Internal helpers: `camelCase`. Package names: short lowercase.
- One use case per file: `usecase_<verb>.go`.
<!-- feat:end -->
<!-- feat:if frontend -->
- Frontend: TypeScript + React function components. Run `npm run lint` (ESLint) before committing; `npm run build` type-checks via `tsc -b`.
- Components in `PascalCase`, hooks as `useThing`, files matching the component name.
<!-- feat:end -->
- Config env var names: `UPPER_SNAKE_CASE`.

## Testing

<!-- feat:if backend -->
- Place tests next to code as `*_test.go`. Prefer table-driven tests.
- Run from `services/backend`: `go test ./...`
<!-- feat:if postgresql -->
- Integration tests live in `internal/integration` behind the `integration` build tag, so they are excluded from the default `go test ./...`. Run them with `make test-integration` (needs dev infra up + migrations applied).
<!-- feat:end -->
<!-- feat:end -->
<!-- feat:if frontend,!backend -->
<!--~ - There is no test runner wired yet; add one (vitest is the natural fit for Vite) when the app grows past static markup. `npm run lint` and `npm run build` are what CI enforces today. -->
<!-- feat:end -->
- CI runs on push/PR (`.github/workflows/ci.yml`, mirrored in `.gitlab-ci.yml`). Keep both CI files in sync.
<!-- feat:if backend -->
  It runs backend `gofmt`/`go vet`/`go build`/`go test`.
<!-- feat:end -->
<!-- feat:if frontend -->
  It runs frontend `npm run lint` and `npm run build`.
<!-- feat:end -->

<!-- feat:if template -->
## Feature markers (template only)

Files carry `feat:if` / `feat:else` / `feat:end` comments that
`scripts/apply_features.sh` renders for a chosen feature set. Read
[`docs/FEATURES.md`](docs/FEATURES.md) before editing a marked file or adding a
feature — the marker rules have sharp edges (line-start only, per-line `~`
leaders, Make recipes).
<!-- feat:end -->

## Commit Format

`type(scope): short summary` — e.g. `feat(note): add tag filtering to list endpoint`

Keep commits atomic. Include migration files in the same commit as the Go code that requires them.
