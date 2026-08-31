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

- [ ] `cd services/backend && gofmt -l . && go vet ./... && go test ./...` passes
- [ ] `cd services/frontend && npm run lint && npm test && npm run build` passes
- [ ] Swagger regenerated if API handlers changed (`make gen-api-docs`)
- [ ] Migration committed alongside the Go code that needs it (if any)
- [ ] Both CI files (`.github/workflows/ci.yml`, `.gitlab-ci.yml`) stay in sync
- [ ] Docs updated where the change lands (`docs/ARCHITECTURE.md`, `infra/OPERATIONS.md`)

## Notes for reviewers

<!-- Anything non-obvious: tradeoffs, follow-ups, screenshots for UI changes. -->
