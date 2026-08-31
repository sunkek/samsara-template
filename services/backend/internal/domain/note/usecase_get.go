package note

import (
	"context"

	"github.com/sunkek/samsara-template/backend/internal/domain/note/model"
)

// Get returns a note by id.
func (d *Domain) Get(ctx context.Context, id string) (model.Note, error) {
	return d.db.Get(ctx, id)
}
