package article

import (
	"context"

	"github.com/sunkek/samsara-template/backend/internal/common/logging"
	"github.com/sunkek/samsara-template/backend/internal/domain/article/model"
)

// List returns all articles, cache-aside: serve the cached list on a hit,
// otherwise read the DB and populate the cache. Cache failures are logged but
// do not fail the request.
func (d *Domain) List(ctx context.Context) ([]model.Article, error) {
	articles, ok, err := d.cache.GetList(ctx)
	if err != nil {
		logging.From(ctx).Warn("article list cache get failed", "error", err)
	}
	if ok {
		return articles, nil
	}
	articles, err = d.db.List(ctx)
	if err != nil {
		return nil, err
	}
	if err := d.cache.SetList(ctx, articles); err != nil {
		logging.From(ctx).Warn("article list cache set failed", "error", err)
	}
	return articles, nil
}
