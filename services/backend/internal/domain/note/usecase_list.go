package note

import (
	"context"

	"github.com/sunkek/samsara-template/backend/internal/domain/note/model"
)

// List returns all notes.
func (d *Domain) List(ctx context.Context) ([]model.Note, error) {
	return d.db.List(ctx)
}
