---
# feat:if postgresql,redis,rabbitmq
status: accepted — amended by ADR 0006
# feat:else
#~ status: accepted
# feat:end
---

# Optional infrastructure is a port with a no-op implementation, never a nil check

<!-- feat:if postgresql,redis,rabbitmq -->
Redis and RabbitMQ are optional in a fork, so the domains that use them declare
ports for what they need — `article.Cache`, `article.Events`, `auth.Revoker` — and
a build without that infrastructure injects a different implementation at the
composition root. Call sites are unchanged and unguarded.
<!-- feat:else -->
<!--~ Redis and RabbitMQ are optional in a fork, so the domains that use them declare -->
<!--~ ports for what they need — `auth.Revoker`, for one — and a build without that -->
<!--~ infrastructure injects a different implementation at the composition root. -->
<!--~ Call sites are unchanged and unguarded. -->
<!-- feat:end -->

The obvious alternatives were a nil field with `if d.cache != nil` at each use,
or build tags. Both spread the optionality across the domain logic, where it
would be re-derived at every call site and eventually forgotten at one. Keeping
it at the wiring boundary means dropping a feature changes exactly one line in
`cmd/main` and deletes one adapter package — which is why cutting a feature out
of this project was a deletion rather than a rewrite.

The consequence to keep in view: a no-op silently does nothing. Any capability
behind such a port must be one the domain can genuinely live without — caching,
event publication — and never one whose absence changes a result.

`auth.Revoker` is the port that fails that test, which is why it has no no-op:
its Redis-less stand-in, `auth/adapter/memory`, is a working process-local
denylist. See ADR 0004.

<!-- feat:if postgresql,redis,rabbitmq -->
ADR 0006 settled the shipping side of this. `article` is pruned along with the
infrastructure it demonstrates, so no call site is left for a no-op to keep
unguarded, and `NoopCache`/`NoopEvents` no longer ship. The wiring claim above —
optionality lives at the composition root, not in the domain logic — is what
survives, and `auth/adapter/memory` is the case that still exercises it.
<!-- feat:else -->
<!--~ No-op implementations no longer ship: the template's samples that existed -->
<!--~ only to demonstrate optional infrastructure are pruned along with it, so no -->
<!--~ call site is left for a no-op to keep unguarded. The wiring claim above — -->
<!--~ optionality lives at the composition root, not in the domain logic — is what -->
<!--~ survives, and `auth/adapter/memory` is the case that still exercises it. -->
<!-- feat:end -->
