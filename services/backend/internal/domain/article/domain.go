// Package article is the optional-infrastructure sample: the same vertical
// slice as note, plus cache-aside reads through a Cache port and publication
// through an Events port. It exists only in a build that has both Redis and
// RabbitMQ — a fork without them has no use for a domain whose whole purpose is
// to demonstrate them (see docs/adr/0006).
package article

// Domain holds the article use cases. It depends on the DB outbound port for
// persistence, the Cache outbound port for read caching (cache-aside) and the
// Events outbound port for publication. The REST/gRPC/GraphQL adapters depend
// on *Domain via the Service interface, so wiring is compile-time checked — no
// handler injection, no nil-handler guards.
type Domain struct {
	db     DB
	cache  Cache
	events Events
}

// New builds the article domain. Cache and Events are best-effort: the use
// cases log their failures and carry on, so an outage degrades reads and
// publication rather than failing requests.
func New(db DB, cache Cache, events Events) *Domain {
	return &Domain{db: db, cache: cache, events: events}
}

// Compile-time assertion.
var _ Service = (*Domain)(nil)
