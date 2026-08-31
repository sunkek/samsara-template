// Package note is the vertical-slice sample: model, use cases, an inbound
// Service port and outbound DB port, with a fiber adapter driving it and a
// postgresql adapter behind it. It carries no optional infrastructure on
// purpose — a fork copies this domain, renames it and starts writing. The
// article domain shows what caching and event publication add on top.
package note

// Domain holds the note use cases. It depends on the DB outbound port and
// nothing else. The REST/gRPC/GraphQL adapters depend on *Domain via the
// Service interface, so wiring is compile-time checked — no handler injection,
// no nil-handler guards.
type Domain struct {
	db DB
}

// New builds the note domain.
func New(db DB) *Domain {
	return &Domain{db: db}
}

// Compile-time assertion.
var _ Service = (*Domain)(nil)
