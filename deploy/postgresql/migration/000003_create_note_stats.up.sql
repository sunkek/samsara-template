-- note_stats is a single-row projection (id = 1) maintained by consuming
-- note.created events. It demonstrates an event-driven read model.
CREATE TABLE IF NOT EXISTS note_stats (
    id           SMALLINT PRIMARY KEY DEFAULT 1,
    total_count  BIGINT      NOT NULL DEFAULT 0,
    last_note_id UUID,
    last_title   TEXT        NOT NULL DEFAULT '',
    updated_at   TIMESTAMPTZ NOT NULL DEFAULT now(),
    CONSTRAINT note_stats_singleton CHECK (id = 1)
);
