package note

import (
	"context"

	"github.com/sunkek/samsara-template/backend/internal/domain/note/model"
)

// Domain holds the note use cases. It depends on the DB outbound port for
// persistence and the Cache outbound port for read caching (cache-aside). The
// REST/gRPC/GraphQL adapters depend on *Domain via the Service interface, so
// wiring is compile-time checked — no handler injection, no nil-handler guards.
type Domain struct {
	db     DB
	cache  Cache
	events Events
}

// New builds the note domain. cache and events are required; pass NoopCache /
// NoopEvents to disable caching / event publishing (both are best-effort, so a
// no-op is a valid configuration).
func New(db DB, cache Cache, events Events) *Domain {
	return &Domain{db: db, cache: cache, events: events}
}

// NoopCache disables caching: every read is a miss and every write is dropped.
// Cache is best-effort, so a disabled cache is a valid configuration, not a
// foot-gun — the domain simply always reads through to the DB.
type NoopCache struct{}

func (NoopCache) GetNote(context.Context, string) (model.Note, bool, error) {
	return model.Note{}, false, nil
}
func (NoopCache) SetNote(context.Context, model.Note) error { return nil }
func (NoopCache) GetList(context.Context) ([]model.Note, bool, error) {
	return nil, false, nil
}
func (NoopCache) SetList(context.Context, []model.Note) error { return nil }
func (NoopCache) InvalidateList(context.Context) error        { return nil }

// NoopEvents disables event publishing: every publish is dropped. Events are
// best-effort, so this is a valid configuration (publishing turned off).
type NoopEvents struct{}

func (NoopEvents) NoteCreated(context.Context, model.Note) error { return nil }

// Compile-time assertions.
var (
	_ Service = (*Domain)(nil)
	_ Cache   = NoopCache{}
	_ Events  = NoopEvents{}
)
