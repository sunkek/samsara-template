package note

import (
	"context"

	"github.com/sunkek/samsara-template/backend/internal/common/logging"
	"github.com/sunkek/samsara-template/backend/internal/common/metrics"
	"github.com/sunkek/samsara-template/backend/internal/domain/note/model"
)

// Get returns a note by id, cache-aside: serve from cache on a hit, otherwise
// read the DB and populate the cache. Cache failures are logged but do not fail
// the request (best-effort).
func (d *Domain) Get(ctx context.Context, id string) (model.Note, error) {
	if n, ok, _ := d.cache.GetNote(ctx, id); ok {
		metrics.CacheHit()
		return n, nil
	}
	metrics.CacheMiss()
	n, err := d.db.Get(ctx, id)
	if err != nil {
		return model.Note{}, err
	}
	if err := d.cache.SetNote(ctx, n); err != nil {
		logging.From(ctx).Warn("note cache set failed", "note_id", n.ID, "error", err)
	}
	return n, nil
}
