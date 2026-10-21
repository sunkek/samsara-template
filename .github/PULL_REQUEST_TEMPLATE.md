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

<!-- feat:if backend -->
- [ ] `cd services/backend && gofmt -l . && go vet ./... && go test ./...` passes
<!-- feat:end -->
<!-- feat:if frontend -->
- [ ] `cd services/frontend && npm run lint && npm test && npm run build` passes
<!-- feat:end -->
<!-- feat:if backend -->
- [ ] Swagger regenerated if API handlers changed (`make gen-api-docs`)
<!-- feat:end -->
<!-- feat:if backend,postgresql -->
- [ ] Migration committed alongside the Go code that needs it (if any)
<!-- feat:end -->
- [ ] Both CI files (`.github/workflows/ci.yml`, `.gitlab-ci.yml`) stay in sync
<!-- feat:if backend -->
- [ ] Docs updated where the change lands (`docs/ARCHITECTURE.md`, `infra/OPERATIONS.md`)
<!-- feat:else -->
<!--~ - [ ] Docs updated where the change lands (`infra/OPERATIONS.md`, `README.md`) -->
<!-- feat:end -->

## Notes for reviewers

<!-- Anything non-obvious: tradeoffs, follow-ups, screenshots for UI changes. -->
