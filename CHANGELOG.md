# Changelog

All notable changes to this project are documented here.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

### Added
- Feature selection: `bootstrap.sh -F <features>` (and `scripts/apply_features.sh`)
  cuts a fork down to any subset of `backend`, `frontend`, `postgresql`, `redis`,
  `rabbitmq`, deleting the adapters, compose services, env vars, migrations, docs
  and CI jobs of everything else.
- `auth/adapter/memory`: in-process refresh-token denylist, used when a fork is
  built without Redis.
- `feature-matrix` CI job: renders every supported feature combination and checks
  that it builds, is gofmt-clean, and produces valid compose files.
- `scripts/features_test.sh`: unit tests for the feature renderer, run in CI.
- `feature-matrix` CI now builds the rendered frontend (`npm install` + `npm run
  build`) and covers the frontend-only and no-features combinations.
- `infra/OPERATIONS.md`: compose stacks, env-file generation and secret sharing,
  host ports, and the production hardening checklist, moved out of `CLAUDE.md`.
- `docs/FEATURES.md`: feature-marker syntax and the rules for editing a marked
  file, moved out of `AGENTS.md`. Removed from forks by `bootstrap.sh` along
  with the rest of the feature tooling.
- Dependabot config (`gomod`, `npm`, `github-actions`, weekly, grouped).
- CodeQL workflow (Go + JS/TS, gated on bootstrap).
- `govulncheck` step in backend CI.
- Committed `services/frontend/package-lock.json` for reproducible installs.

### Fixed
- CI in a fork rendered without `backend` no longer skips every job: the
  "unbootstrapped template" guard keyed on `services/backend/go.sum`, a file
  such a fork never has, so GitHub CI, GitLab CI and CodeQL short-circuited
  permanently. The guard is now gated on `backend`, CodeQL's language matrix is
  gated per app service, and the CI definitions are pruned outright when neither
  app service is selected (an empty `jobs:` is not a valid workflow).

### Changed
- Restructured the agent-facing docs so each fact has one home: `CLAUDE.md`
  drops from 179 to 77 lines, replacing its command reference (a stale-able copy
  of `make help`) and its duplicated architecture prose with pointers that state
  when to read each doc. The domain pattern, config, error codes and logging now
  live only in `docs/ARCHITECTURE.md`.
- Feature-gated the last backend-specific leftovers so a fork without `backend`
  is self-consistent: the vite dev-server `/api` proxy and `VITE_API_BASE`,
  `env/example/api.env` and `docs/` (now pruned), the Makefile help text, `.PHONY`
  lists and comments, and the backend prose in `README.md`, `CLAUDE.md` and
  `AGENTS.md` — which now describe whichever build the fork actually is.
- `CONTRIBUTING.md` and `SECURITY.md` no longer address template contributors in
  a fork: the template-maintenance framing (contribution scope, samsara
  components, "track `main` and re-apply patches") is now gated on the
  template-only pseudo-feature, and the setup/PR checklists are gated per
  feature.
- The GitHub issue templates no longer ask forks to report bugs "in the
  template".
- Frontend CI now uses `npm ci` with npm cache (was `npm install`).
