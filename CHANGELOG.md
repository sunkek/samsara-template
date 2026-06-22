# Changelog

All notable changes to this project are documented here.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

### Added
- Dependabot config (`gomod`, `npm`, `github-actions`, weekly, grouped).
- CodeQL workflow (Go + JS/TS, gated on bootstrap).
- `govulncheck` step in backend CI.
- Committed `service/frontend/package-lock.json` for reproducible installs.

### Changed
- Frontend CI now uses `npm ci` with npm cache (was `npm install`).
