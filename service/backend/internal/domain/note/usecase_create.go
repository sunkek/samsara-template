package note

import (
	"context"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/sunkek/mishap"

	"github.com/sunkek/samsara-template/backend/internal/common/e"
	"github.com/sunkek/samsara-template/backend/internal/common/logging"
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
	// Warm the item cache and invalidate the now-stale list (best-effort).
	if err := d.cache.SetNote(ctx, created); err != nil {
		logging.From(ctx).Warn("note cache warm failed", "note_id", created.ID, "error", err)
	}
	if err := d.cache.InvalidateList(ctx); err != nil {
		logging.From(ctx).Warn("note list cache invalidate failed", "error", err)
	}
	// Publish the domain event (best-effort: a broker outage must not fail the
	// write that already succeeded).
	if err := d.events.NoteCreated(ctx, created); err != nil {
		logging.From(ctx).Warn("note.created publish failed", "note_id", created.ID, "error", err)
	}
	return created, nil
}
