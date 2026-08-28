# Optional infrastructure is a port with a no-op implementation, never a nil check

Redis and RabbitMQ are optional in a fork, so the domains that use them declare
ports for what they need — `note.Cache`, `note.Events` — and ship a no-op
implementation of each (`NoopCache`, `NoopEvents`). A build without that infra
injects the no-op. Call sites are unchanged and unguarded.

The obvious alternatives were a nil field with `if d.cache != nil` at each use,
or build tags. Both spread the optionality across the domain logic, where it
would be re-derived at every call site and eventually forgotten at one. Keeping
it at the wiring boundary means dropping a feature changes exactly one line in
`cmd/main` and deletes one adapter package — which is why `bootstrap.sh -F` can
be a deletion rather than a rewrite.

The consequence to keep in view: a no-op silently does nothing. Any capability
behind such a port must be one the domain can genuinely live without — caching,
event publication — and never one whose absence changes a result.

`auth.Revoker` is the port that fails that test, which is why it has no no-op:
its Redis-less stand-in, `auth/adapter/memory`, is a working process-local
denylist. See ADR 0004.
