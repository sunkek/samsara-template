// Package postgresql implements the notestats projection store. The projection
// is a single row (id = 1) upserted on each note.created event.
package postgresql

import (
	"context"
	"errors"

	"github.com/jackc/pgx/v5"
	"github.com/sunkek/mishap"
	pgcmp "github.com/sunkek/samsara-components/postgresql"

	notemodel "github.com/sunkek/samsara-template/backend/internal/domain/note/model"
	"github.com/sunkek/samsara-template/backend/internal/domain/notestats/model"
)

type Adapter struct {
	pg *pgcmp.Component
}

func New(pg *pgcmp.Component) *Adapter {
	return &Adapter{pg: pg}
}

// RecordNoteCreated increments the counter and records the latest note. The
// upsert is idempotent against the singleton row.
func (a *Adapter) RecordNoteCreated(ctx context.Context, e notemodel.NoteCreatedEvent) error {
	const q = `
		INSERT INTO note_stats (id, total_count, last_note_id, last_title, updated_at)
		VALUES (1, 1, $1, $2, now())
		ON CONFLICT (id) DO UPDATE SET
			total_count  = note_stats.total_count + 1,
			last_note_id = EXCLUDED.last_note_id,
			last_title   = EXCLUDED.last_title,
			updated_at   = now()`
	if _, err := a.pg.Exec(ctx, q, e.NoteID, e.Title); err != nil {
		return mishap.Wrap(err, "record note_stats")
	}
	return nil
}

func (a *Adapter) Get(ctx context.Context) (model.Stats, error) {
	const q = `SELECT total_count, COALESCE(last_note_id::text, ''), last_title, updated_at
		FROM note_stats WHERE id = 1`
	var out model.Stats
	if err := a.pg.Get(ctx, &out, q); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			// No events projected yet — return an empty (zero) projection.
			return model.Stats{}, nil
		}
		return model.Stats{}, mishap.Wrap(err, "get note_stats")
	}
	return out, nil
}
