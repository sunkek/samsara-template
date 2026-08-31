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
- `CONTEXT.md`: the project glossary, opinionated about which word to use where
  three different words in this repo mean "a unit of the system" (samsara
  component, Compose service, backend domain).
- `docs/adr/`: five architecture decision records covering feature rendering,
  domain-owned ports, no-op ports for optional infra, refresh-only token
  revocation, and hand-written SQL.
- `docs/FEATURES.md`: feature-marker syntax and the rules for editing a marked
  file, moved out of `AGENTS.md`. Removed from forks by `bootstrap.sh` along
  with the rest of the feature tooling.
- Tests at four previously uncovered seams: the auth middleware's public-prefix
  matching and Bearer handling, the per-IP rate limiter, `e.HTTPStatus`
  (exhaustive, now 100%), and the note HTTP handlers driven through the
  adapter's real route table.
- Frontend test runner: vitest + Testing Library on jsdom, wired into both CI
  files (`npm test`), with `test:watch` and `test:coverage` scripts.
- `POST-FORK.md`: `apply_features.sh` now names the documents whose prose this
  feature set cut, so a fork owner reviews those rather than re-reading
  everything. The baseline is the all-features render, so the template-only
  sections every fork loses are not reported as work.
- ADR 0006: three sample domains, one lesson each, and optional infra deleting
  its demo rather than degrading it.
- Dependabot config (`gomod`, `npm`, `github-actions`, weekly, grouped).
- CodeQL workflow (Go + JS/TS, gated on bootstrap).
- `make help` now lists `gen-env`, `gen-key-hex`, `gen-key-b64`, `ps`, `pull`,
  `restart-local` and `restart-local-v`, which it had never mentioned — including
  `gen-env`, which the first-run sequence depends on.
- `govulncheck` step in backend CI.
- Committed `services/frontend/package-lock.json` for reproducible installs.

### Fixed
- The per-IP auth rate limiter was one shared bucket in the stage and prod
  stacks. Every request arrives through nginx and no proxy trust was configured,
  so `c.IP()` returned nginx's address for all of them: ten login attempts from
  anyone locked the whole `/auth` group for everyone, and there was no
  per-attacker brute-force protection at all. `Fiber.TrustProxy`,
  `TrustedProxies` and `TrustProxyHeader` are now wired through to samsara and
  enabled in the stage/prod compose files. nginx now *overwrites*
  `X-Forwarded-For` rather than appending, because the backend reads the
  left-most entry and an appending chain would let a client forge it.
- Refresh-token rotation was not atomic: `IsRevoked` then `Revoke` are two
  round-trips, so two concurrent uses of a stolen token both passed the check
  and both minted a valid pair. The `Revoker` port gained `Claim` — Redis
  `SET NX`, or an insert under the memory adapter's lock — and a concurrency
  test that fails 16/16 against the old sequence.
- The `notestats` consumer ran projection writes on `context.Background()` with
  no deadline, and requeued failures instantly: a database outage became a hot
  nack/redeliver loop. Writes are now bounded and a failed delivery pauses
  before returning.
- A fork rendered without `frontend` shipped a README promising "paired with a
  React/Vite SPA" and telling the reader to run `./bootstrap.sh`, which
  bootstrap had just deleted. The intro is now gated per app shape.
- `bootstrap.sh` left template-maintenance material in the fork with nothing to
  point at: the template's own `CHANGELOG.md`, ADR 0001 (the feature renderer),
  and an issue template instructing reporters to run `bootstrap.sh`. All three
  are now removed or amended.
- A failed login returned 403. It is an authentication failure, so it is now
  401 via a new `e.Unauthorized`; `e.Forbidden` keeps its meaning of an
  authenticated caller denied access.
- A password over bcrypt's 72-byte limit returned 500 rather than 400.
- Malformed request bodies returned 500 on every handler — unauthenticated
  input driving the 5xx error rate. Now 400.
- `.gitlab-ci.yml` drifted from `.github/workflows/ci.yml` while three documents
  claimed they were in sync: node 22 vs 26, `npm install` vs `npm ci` despite a
  committed lockfile, and no `govulncheck`. Fixed, and the rule now states its
  one real exception (template-maintenance jobs are GitHub-only).
- `features.awk` set `seenelse` but never read it, so a second `feat:else` in
  one block silently flipped the branch back on instead of erroring.
- `apply_features.sh` selected files to render with a narrower pattern than the
  renderer itself matches, so a file whose first marker was written differently
  was skipped whole and shipped its markers raw.
- `bootstrap.sh`'s interactive destination defaulted to `.` — the in-place path
  that rewrites the template checkout itself. It now defaults to `../<app>`.
- `internal/common/crypto` documented a different project entirely
  (`SHIPPER_API_SECRETS_MASTER_KEY`, "per-domain Secrets ports").
- `/metrics` is in the auth middleware's public-prefix list and was documented
  nowhere; `docs/ARCHITECTURE.md` and `infra/OPERATIONS.md` now say so.
- ADR 0006 described `article`/`articlestats` packages that do not exist yet
  with status `accepted`; it is now marked not-yet-implemented.
- The `feature-matrix` "no markers left" check matched `feat:if` anywhere in a
  line, so prose documenting the marker syntax (`docs/FEATURES.md`, ADR 0001)
  would have failed CI. It now matches the renderer's own rule — line start
  only — and also rejects surviving inert `~` leaders.
- Removed `services/backend/CONTEXT.md`, a stray duplicate of the root glossary
  committed by mistake.
- `docs/ARCHITECTURE.md` omitted the `RateLimit` error code and attributed the
  code-to-status mapping to `cmd/main`; it is `e.HTTPStatus`, shared with the
  metrics middleware.
- Cache read failures in the `note` domain were discarded (`_`), so a Redis
  outage was silent on the read path while write-path failures logged. Reads now
  log the error and still fall back to the database.
- Cache hit/miss metrics moved from the `note` use cases into the Redis adapter.
  They were counted in the domain, so a build wired with `NoopCache` recorded a
  miss on every read and reported a permanent 0% hit rate for a deployment that
  has no cache at all.
- `auth.Revoker`'s doc comment described the default implementation as "an
  in-memory no-op". There is no no-op revoker: `auth/adapter/memory` is a real
  process-local denylist, and a no-op there would make logout a lie.
- CI in a fork rendered without `backend` no longer skips every job: the
  "unbootstrapped template" guard keyed on `services/backend/go.sum`, a file
  such a fork never has, so GitHub CI, GitLab CI and CodeQL short-circuited
  permanently. The guard is now gated on `backend`, CodeQL's language matrix is
  gated per app service, and the CI definitions are pruned outright when neither
  app service is selected (an empty `jobs:` is not a valid workflow).

### Changed
- Split the sample domains so each teaches one thing (ADR 0006): `note` is now
  the vertical slice and nothing else — no cache, no events — and is the domain
  a fork copies; the new `article` domain carries the cache-aside reads and the
  `article.created` publication; `notestats` becomes `articlestats`, projecting
  `article.created` into `article_stats`. `article` and `articlestats` exist to
  demonstrate Redis and RabbitMQ, so a build without either is rendered without
  them, and `note.NoopCache` / `note.NoopEvents` no longer ship: with the
  demonstrating domain pruned, no call site is left for a no-op to keep
  unguarded. `MY_PROJECT_API_NOTE_CACHE_TTL` becomes
  `MY_PROJECT_API_ARTICLE_CACHE_TTL`, and `EVENTS_NOTE_*` becomes
  `EVENTS_ARTICLE_*`. New migrations `000003_create_articles` and
  `000004_create_article_stats` replace `000003_create_note_stats`.
- `feature-matrix` CI now covers all 32 feature combinations rather than a
  curated nine. A curated subset leaves the rest to break silently, and a fork
  only finds out after it has been created.
- Consolidated feature-gated prose: `CONTRIBUTING.md` 32 marker lines to 16 and
  `CLAUDE.md` 20 to 12, by pointing at the one authoritative place instead of
  restating per-feature variants. `CLAUDE.md` no longer describes which pieces
  the build has — that is readable off `services/` and `infra/`, and cannot go
  stale there.
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
