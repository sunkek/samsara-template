<!--
Thanks for the PR. Keep it focused: one logical change per PR.
See CONTRIBUTING.md and AGENTS.md for conventions.
-->

## Summary

<!-- What changes, and why. One or two sentences. Link related issues. -->

## Changes

<!-- Bullet list of the concrete edits. -->

-

## Checklist

- [ ] `cd service/backend && gofmt -l . && go vet ./... && go test ./...` passes
- [ ] `cd service/frontend && npm run lint && npm run build` passes
- [ ] Swagger regenerated if API handlers changed (`make gen-api-docs`)
- [ ] Migration committed alongside the Go code that needs it (if any)
- [ ] Both CI files (`.github/workflows/ci.yml`, `.gitlab-ci.yml`) stay in sync
- [ ] Architecture docs updated if cross-cutting (`CLAUDE.md`, `docs/ARCHITECTURE.md`)

## Notes for reviewers

<!-- Anything non-obvious: tradeoffs, follow-ups, screenshots for UI changes. -->
