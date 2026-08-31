// Package articlestats is the article read-model: an event-sourced projection
// updated by consuming article.created events. It is a separate domain from
// article on purpose — that separation of write and read models is the lesson
// this sample carries. Like article, it exists only in a build that has both
// Redis and RabbitMQ (see docs/adr/0006).
package articlestats

// Domain holds the read-model use cases. It depends only on the DB outbound
// port; the consumer and REST adapters depend on *Domain via Service.
type Domain struct {
	db DB
}

func New(db DB) *Domain {
	return &Domain{db: db}
}

var _ Service = (*Domain)(nil)
