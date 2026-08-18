# Repository Guidelines

See `CLAUDE.md` for project overview, commands, and architecture.

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
- Never put a marker **inside** a backslash-continued Make recipe or shell
  command: the shell joins the lines, so `# feat:if …` comments out the rest of
  the command in the unrendered template. Gate the variable that feeds the
  recipe instead (see `LOCAL_INFRA` / `RUN_LOCAL_*` in the `Makefile`).
- Prefer plain (uncommented) content inside a positively-gated block; the `~`
  leader is for blocks that are *inactive* in the all-features build, so using
  it elsewhere hides that content from the template's own readers.
- Add a combination to the `feature-matrix` CI job when you add a feature.
- `scripts/features_test.sh` unit-tests the renderer; extend it when you change
  marker semantics.
<!-- feat:end -->

## Commit Format

`type(scope): short summary` — e.g. `feat(note): add tag filtering to list endpoint`

Keep commits atomic. Include migration files in the same commit as the Go code that requires them.
