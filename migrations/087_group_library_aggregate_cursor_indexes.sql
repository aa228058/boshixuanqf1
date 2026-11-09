-- +goose Up
-- Stable keyset pagination for group-library incremental source scans.
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
