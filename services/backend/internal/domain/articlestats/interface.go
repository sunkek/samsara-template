package articlestats

import (
	"context"

	articlemodel "github.com/sunkek/samsara-template/backend/internal/domain/article/model"
	"github.com/sunkek/samsara-template/backend/internal/domain/articlestats/model"
)

// Service is the inbound port. ApplyArticleCreated is driven by the RabbitMQ
// consumer adapter; Get is driven by the REST adapter. *Domain implements it.
type Service interface {
	ApplyArticleCreated(ctx context.Context, e articlemodel.ArticleCreatedEvent) error
	Get(ctx context.Context) (model.Stats, error)
}

// DB is the outbound port for the projection store. The postgresql adapter
// implements it.
type DB interface {
	RecordArticleCreated(ctx context.Context, e articlemodel.ArticleCreatedEvent) error
	Get(ctx context.Context) (model.Stats, error)
}
