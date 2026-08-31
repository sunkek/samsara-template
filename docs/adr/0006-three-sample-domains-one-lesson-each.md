---
status: accepted
---

# Three sample domains, one lesson each — and optional infra deletes its demo

The samples were carrying too many lessons at once. `note` demonstrated the
vertical slice, cache-aside reads, event publication and — through `notestats` —
a CQRS projection, all entangled. A reader converting it into real code had to
work out which parts were the shape and which were the demonstration, and
dropping `redis` or `rabbitmq` changed the shape of the only reference domain.

The samples split so each teaches one thing:

- **`note`** — the vertical slice and nothing else: model, `Service`, `DB`,
  fiber adapter. This is the domain a fork copies.
- **`article`** — optional infra: cache-aside reads through a `Cache` port and
  publication through an `Events` port.
- **`articlestats`** (today's `notestats`) — the projection: `article.created`
  consumed, folded into a read model, served over HTTP. Presented as the async
  showcase it is, not apologised for as "a sample read model".

Demo purpose is signalled in each package's doc comment and in `CONTEXT.md`,
never in the identifier: a fork's real domain should look like these, so the
names must be ones a reader is willing to copy.

## Optional infra deletes its demo rather than degrading it

Dropping `redis` or `rabbitmq` removes `article` and `articlestats` outright.
The alternative — keeping `article` wired to a no-op cache — leaves a fork
carrying a domain whose entire purpose is to demonstrate a capability that fork
does not have.

This revisits ADR 0003, whose claim was that optional infrastructure is a port
with a no-op implementation rather than a nil check. The wiring claim stands:
optionality belongs at the composition root, and a fork adding Redis later
declares a port and injects an adapter without touching a call site. What does
not stand is shipping `NoopCache`/`NoopEvents` to forks that have no use for
them — with `article` pruned, there is no call site left to keep unguarded.

## Why the template still renders every combination

The audience that breaks ties is a stranger evaluating samsara, so the
all-features build is the product and it must show the whole stack. But the
template must also stay a working project template, so every combination has to
render into a self-consistent fork — which is why the feature markers survive
this decision, and why CI now covers all 32 combinations rather than a curated
nine.
