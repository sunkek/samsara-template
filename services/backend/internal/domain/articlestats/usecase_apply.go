package articlestats

import (
	"context"

	articlemodel "github.com/sunkek/samsara-template/backend/internal/domain/article/model"
)

// ApplyArticleCreated folds an article.created event into the projection.
func (d *Domain) ApplyArticleCreated(ctx context.Context, e articlemodel.ArticleCreatedEvent) error {
	return d.db.RecordArticleCreated(ctx, e)
}
