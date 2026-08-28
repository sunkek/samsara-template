package note

import (
	"context"

	"github.com/sunkek/samsara-template/backend/internal/common/logging"
	"github.com/sunkek/samsara-template/backend/internal/domain/note/model"
)

// List returns all notes, cache-aside: serve the cached list on a hit,
// otherwise read the DB and populate the cache. Cache failures are logged but
// do not fail the request.
func (d *Domain) List(ctx context.Context) ([]model.Note, error) {
	notes, ok, err := d.cache.GetList(ctx)
	if err != nil {
		logging.From(ctx).Warn("note list cache get failed", "error", err)
	}
	if ok {
		return notes, nil
	}
	notes, err = d.db.List(ctx)
	if err != nil {
		return nil, err
	}
	if err := d.cache.SetList(ctx, notes); err != nil {
		logging.From(ctx).Warn("note list cache set failed", "error", err)
	}
	return notes, nil
}
