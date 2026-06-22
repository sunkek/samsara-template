package notestats

import (
	"context"

	notemodel "github.com/sunkek/samsara-template/backend/internal/domain/note/model"
	"github.com/sunkek/samsara-template/backend/internal/domain/notestats/model"
)

// Service is the inbound port. ApplyNoteCreated is driven by the RabbitMQ
// consumer adapter; Get is driven by the REST adapter. *Domain implements it.
type Service interface {
	ApplyNoteCreated(ctx context.Context, e notemodel.NoteCreatedEvent) error
	Get(ctx context.Context) (model.Stats, error)
}

// DB is the outbound port for the projection store. The postgresql adapter
// implements it.
type DB interface {
	RecordNoteCreated(ctx context.Context, e notemodel.NoteCreatedEvent) error
	Get(ctx context.Context) (model.Stats, error)
}
