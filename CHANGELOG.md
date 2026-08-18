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
- Dependabot config (`gomod`, `npm`, `github-actions`, weekly, grouped).
- CodeQL workflow (Go + JS/TS, gated on bootstrap).
- `govulncheck` step in backend CI.
- Committed `services/frontend/package-lock.json` for reproducible installs.

### Changed
- Frontend CI now uses `npm ci` with npm cache (was `npm install`).
