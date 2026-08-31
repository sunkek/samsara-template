package note

import (
	"context"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/sunkek/mishap"

	"github.com/sunkek/samsara-template/backend/internal/common/e"
	"github.com/sunkek/samsara-template/backend/internal/common/metrics"
	"github.com/sunkek/samsara-template/backend/internal/domain/note/model"
)

func (d *Domain) Create(ctx context.Context, in model.CreateInput) (model.Note, error) {
	if strings.TrimSpace(in.Title) == "" {
		return model.Note{}, mishap.New("title is required", e.Validation)
	}
	now := time.Now().UTC()
	n := model.Note{
		ID:        uuid.NewString(),
		Title:     in.Title,
		Body:      in.Body,
		CreatedAt: now,
		UpdatedAt: now,
	}
	created, err := d.db.Insert(ctx, n)
	if err != nil {
		return model.Note{}, err
	}
	metrics.NoteCreated()
	return created, nil
}
