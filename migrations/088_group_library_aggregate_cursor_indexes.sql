-- +goose Up
-- Stable keyset pagination and durable progress for group-library source scans.
CREATE TABLE IF NOT EXISTS group_library_aggregation_cursors (
    source VARCHAR(32) PRIMARY KEY CHECK (source IN ('scraped_links', 'groups', 'members_bio')),
    cursor_at TIMESTAMPTZ NOT NULL,
    cursor_id UUID NOT NULL,
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE INDEX IF NOT EXISTS idx_scraped_links_aggregate_cursor
    ON scraped_links (created_at ASC, link_id ASC);

CREATE INDEX IF NOT EXISTS idx_groups_aggregate_cursor
    ON groups ((GREATEST(created_at, updated_at)) ASC, group_id ASC)
    WHERE is_joined = TRUE AND username IS NOT NULL AND username <> '';

CREATE INDEX IF NOT EXISTS idx_members_bio_aggregate_cursor
    ON members (scraped_at ASC, member_id ASC);

-- +goose Down
DROP INDEX IF EXISTS idx_members_bio_aggregate_cursor;
DROP INDEX IF EXISTS idx_groups_aggregate_cursor;
DROP INDEX IF EXISTS idx_scraped_links_aggregate_cursor;
DROP TABLE IF EXISTS group_library_aggregation_cursors;
