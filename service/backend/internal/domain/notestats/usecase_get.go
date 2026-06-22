package notestats

import (
	"context"

	"github.com/sunkek/samsara-template/backend/internal/domain/notestats/model"
)

// Get returns the current projection. An empty projection (no events yet) is a
// zero-value Stats, not an error.
func (d *Domain) Get(ctx context.Context) (model.Stats, error) {
	return d.db.Get(ctx)
}
