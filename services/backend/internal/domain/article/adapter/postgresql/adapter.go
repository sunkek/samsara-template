package postgresql

import (
	"context"
	"errors"

	"github.com/jackc/pgx/v5"
	"github.com/sunkek/mishap"
	pgcmp "github.com/sunkek/samsara-components/postgresql"

	"github.com/sunkek/samsara-template/backend/internal/common/e"
	"github.com/sunkek/samsara-template/backend/internal/domain/article/model"
)

type Adapter struct {
	pg *pgcmp.Component
}

func New(pg *pgcmp.Component) *Adapter {
	return &Adapter{pg: pg}
}

func (a *Adapter) Insert(ctx context.Context, n model.Article) (model.Article, error) {
	const q = `
		INSERT INTO articles (id, title, body, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5)
		RETURNING id, title, body, created_at, updated_at`
	var out model.Article
	if err := a.pg.Get(ctx, &out, q, n.ID, n.Title, n.Body, n.CreatedAt, n.UpdatedAt); err != nil {
		return model.Article{}, mishap.Wrap(err, "insert article")
	}
	return out, nil
}

func (a *Adapter) List(ctx context.Context) ([]model.Article, error) {
	const q = `SELECT id, title, body, created_at, updated_at FROM articles ORDER BY created_at DESC`
	var out []model.Article
	if err := a.pg.Select(ctx, &out, q); err != nil {
		return nil, mishap.Wrap(err, "list articles")
	}
	return out, nil
}

func (a *Adapter) Get(ctx context.Context, id string) (model.Article, error) {
	const q = `SELECT id, title, body, created_at, updated_at FROM articles WHERE id = $1`
	var out model.Article
	if err := a.pg.Get(ctx, &out, q, id); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return model.Article{}, mishap.New("article not found", e.NotFound)
		}
		return model.Article{}, mishap.Wrap(err, "get article")
	}
	return out, nil
}
