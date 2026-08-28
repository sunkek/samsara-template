# Features are rendered out of one all-features source, at bootstrap time

A fork needs only some of `backend`, `frontend`, `postgresql`, `redis`,
`rabbitmq`, and carrying the rest as dead adapters, unused compose services and
env vars for infra nobody runs is the thing that makes a scaffold rot. The
alternatives were a branch per combination (which must be merged forever) and a
generator template in Jinja/cookiecutter syntax (whose source is not a runnable
project). We chose in-file `feat:if` markers written in each file's own comment
syntax, rendered once by `scripts/apply_features.sh` when a fork is created.

The point of the choice is that the checked-in template is the all-features
build and runs as-is: markers are comments, so the source stays a working
project rather than becoming a template of one. The cost is that every edit to a
marked file happens in a language with its own rules (see `docs/FEATURES.md`),
and that a combination can break without anyone noticing — which is why the
`feature-matrix` CI job renders and builds every supported combination.

This is a build-time deletion, not a runtime switch. There is no feature flag
anywhere in this codebase.
