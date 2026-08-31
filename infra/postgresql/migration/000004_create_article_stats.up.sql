-- article_stats is a single-row projection (id = 1) maintained by consuming
-- article.created events. It demonstrates an event-driven read model.
CREATE TABLE IF NOT EXISTS article_stats (
    id              SMALLINT PRIMARY KEY DEFAULT 1,
    total_count     BIGINT      NOT NULL DEFAULT 0,
    last_article_id UUID,
    last_title      TEXT        NOT NULL DEFAULT '',
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT now(),
    CONSTRAINT article_stats_singleton CHECK (id = 1)
);
