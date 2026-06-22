package notestats

import (
	"context"

	notemodel "github.com/sunkek/samsara-template/backend/internal/domain/note/model"
)

// ApplyNoteCreated folds a note.created event into the projection.
func (d *Domain) ApplyNoteCreated(ctx context.Context, e notemodel.NoteCreatedEvent) error {
	return d.db.RecordNoteCreated(ctx, e)
}
