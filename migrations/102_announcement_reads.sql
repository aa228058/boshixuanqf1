-- +goose Up
-- 公告已读状态。此前 POST /announcements/:id/read 是空实现（直接返回 204），
-- 前端标记已读后其实什么都没记，未读红点永远消不掉。
CREATE TABLE IF NOT EXISTS announcement_reads (
    tenant_id UUID NOT NULL REFERENCES tenants(tenant_id) ON DELETE CASCADE,
    announcement_id UUID NOT NULL REFERENCES announcements(announcement_id) ON DELETE CASCADE,
    read_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    PRIMARY KEY (tenant_id, announcement_id)
);

CREATE INDEX IF NOT EXISTS idx_announcement_reads_tenant ON announcement_reads(tenant_id);

-- +goose Down
DROP TABLE IF EXISTS announcement_reads;
