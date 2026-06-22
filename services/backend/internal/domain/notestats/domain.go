// Package notestats is the note read-model: an event-sourced projection updated
// by consuming note.created events. It is intentionally a separate domain from
// note to demonstrate CQRS-style separation of write and read models.
package notestats

// Domain holds the read-model use cases. It depends only on the DB outbound
// port; the consumer and REST adapters depend on *Domain via Service.
type Domain struct {
	db DB
}

func New(db DB) *Domain {
	return &Domain{db: db}
}

var _ Service = (*Domain)(nil)
