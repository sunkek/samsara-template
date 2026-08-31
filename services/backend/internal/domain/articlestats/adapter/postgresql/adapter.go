// Package postgresql implements the articlestats projection store. The projection
// is a single row (id = 1) upserted on each article.created event.
package postgresql

import (
	"context"
	"errors"

	"github.com/jackc/pgx/v5"
	"github.com/sunkek/mishap"
	pgcmp "github.com/sunkek/samsara-components/postgresql"

	articlemodel "github.com/sunkek/samsara-template/backend/internal/domain/article/model"
	"github.com/sunkek/samsara-template/backend/internal/domain/articlestats/model"
)

type Adapter struct {
	pg *pgcmp.Component
}

func New(pg *pgcmp.Component) *Adapter {
	return &Adapter{pg: pg}
}

// RecordArticleCreated increments the counter and records the latest article. The
// upsert is idempotent against the singleton row.
func (a *Adapter) RecordArticleCreated(ctx context.Context, e articlemodel.ArticleCreatedEvent) error {
	const q = `
		INSERT INTO article_stats (id, total_count, last_article_id, last_title, updated_at)
		VALUES (1, 1, $1, $2, now())
		ON CONFLICT (id) DO UPDATE SET
			total_count  = article_stats.total_count + 1,
			last_article_id = EXCLUDED.last_article_id,
			last_title   = EXCLUDED.last_title,
			updated_at   = now()`
	if _, err := a.pg.Exec(ctx, q, e.ArticleID, e.Title); err != nil {
		return mishap.Wrap(err, "record article_stats")
	}
	return nil
}

func (a *Adapter) Get(ctx context.Context) (model.Stats, error) {
	const q = `SELECT total_count, COALESCE(last_article_id::text, ''), last_title, updated_at
		FROM article_stats WHERE id = 1`
	var out model.Stats
	if err := a.pg.Get(ctx, &out, q); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			// No events projected yet — return an empty (zero) projection.
			return model.Stats{}, nil
		}
		return model.Stats{}, mishap.Wrap(err, "get article_stats")
	}
	return out, nil
}
