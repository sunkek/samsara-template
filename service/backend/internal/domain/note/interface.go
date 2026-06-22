package note

import (
	"context"

	"github.com/sunkek/samsara-template/backend/internal/domain/note/model"
)

// Service is the inbound port: the set of use cases the REST adapter calls.
// *Domain implements it. The fiber adapter depends on this interface, so the
// dependency points adapter → domain (never the reverse).
type Service interface {
	Create(ctx context.Context, in model.CreateInput) (model.Note, error)
	List(ctx context.Context) ([]model.Note, error)
	Get(ctx context.Context, id string) (model.Note, error)
}

// DB is the outbound port: persistence the domain needs. The postgresql
// adapter implements it.
type DB interface {
	Insert(ctx context.Context, n model.Note) (model.Note, error)
	List(ctx context.Context) ([]model.Note, error)
	Get(ctx context.Context, id string) (model.Note, error)
}

// Cache is the outbound port for read caching (cache-aside). It is best-effort:
// the domain treats any cache error as a miss and falls back to the DB, so a
// cache outage never fails a request. The bool reports a hit. The Redis adapter
// implements it; NoopCache disables caching without touching call sites.
type Cache interface {
	GetNote(ctx context.Context, id string) (model.Note, bool, error)
	SetNote(ctx context.Context, n model.Note) error
	GetList(ctx context.Context) ([]model.Note, bool, error)
	SetList(ctx context.Context, notes []model.Note) error
	InvalidateList(ctx context.Context) error
}

// Events is the outbound port for publishing domain events. It is best-effort:
// the domain ignores publish errors so a broker outage never fails a write. The
// RabbitMQ adapter implements it; NoopEvents disables publishing.
type Events interface {
	NoteCreated(ctx context.Context, n model.Note) error
}
