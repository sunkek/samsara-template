package article

import (
	"context"

	"github.com/sunkek/samsara-template/backend/internal/domain/article/model"
)

// Service is the inbound port: the set of use cases the REST adapter calls.
// *Domain implements it. The fiber adapter depends on this interface, so the
// dependency points adapter → domain (never the reverse).
type Service interface {
	Create(ctx context.Context, in model.CreateInput) (model.Article, error)
	List(ctx context.Context) ([]model.Article, error)
	Get(ctx context.Context, id string) (model.Article, error)
}

// DB is the outbound port: persistence the domain needs. The postgresql
// adapter implements it.
type DB interface {
	Insert(ctx context.Context, n model.Article) (model.Article, error)
	List(ctx context.Context) ([]model.Article, error)
	Get(ctx context.Context, id string) (model.Article, error)
}

// Cache is the outbound port for read caching (cache-aside). It is best-effort:
// the domain treats any cache error as a miss and falls back to the DB, so a
// cache outage never fails a request — but it logs the error, so a cache that
// is down is visible rather than silent. The bool reports a hit. Hit/miss
// metrics belong to the implementation, not to this port. The Redis adapter implements it.
type Cache interface {
	GetArticle(ctx context.Context, id string) (model.Article, bool, error)
	SetArticle(ctx context.Context, n model.Article) error
	GetList(ctx context.Context) ([]model.Article, bool, error)
	SetList(ctx context.Context, articles []model.Article) error
	InvalidateList(ctx context.Context) error
}

// Events is the outbound port for publishing domain events. It is best-effort:
// the domain ignores publish errors so a broker outage never fails a write. The RabbitMQ adapter implements it.
type Events interface {
	ArticleCreated(ctx context.Context, n model.Article) error
}
