package note

import (
	"context"

	"github.com/sunkek/samsara-template/backend/internal/domain/note/model"
)

// Service is the inbound port: the set of use cases the REST adapter calls.
// *Domain implements it. The fiber adapter depends on this interface, so the
// dependency points adapter → domain (never the reverse).
type Service interface {
	Create(ctx context.Context, in model.CreateInput) (model.Note, error)
	List(ctx context.Context) ([]model.Note, error)
	Get(ctx context.Context, id string) (model.Note, error)
}

// DB is the outbound port: persistence the domain needs. The postgresql
// adapter implements it.
type DB interface {
	Insert(ctx context.Context, n model.Note) (model.Note, error)
	List(ctx context.Context) ([]model.Note, error)
	Get(ctx context.Context, id string) (model.Note, error)
}
