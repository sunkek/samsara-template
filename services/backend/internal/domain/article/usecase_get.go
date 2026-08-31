package article

import (
	"context"

	"github.com/sunkek/samsara-template/backend/internal/common/logging"
	"github.com/sunkek/samsara-template/backend/internal/domain/article/model"
)

// Get returns an article by id, cache-aside: serve from cache on a hit,
// otherwise read the DB and populate the cache. Cache failures are logged but do not fail
// the request (best-effort).
func (d *Domain) Get(ctx context.Context, id string) (model.Article, error) {
	n, ok, err := d.cache.GetArticle(ctx, id)
	if err != nil {
		logging.From(ctx).Warn("article cache get failed", "article_id", id, "error", err)
	}
	if ok {
		return n, nil
	}
	n, err = d.db.Get(ctx, id)
	if err != nil {
		return model.Article{}, err
	}
	if err := d.cache.SetArticle(ctx, n); err != nil {
		logging.From(ctx).Warn("article cache set failed", "article_id", n.ID, "error", err)
	}
	return n, nil
}
