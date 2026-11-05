-- +goose Up
-- 群组主体、多个入口链接、租户贡献记录分离；保留旧 system_groups 数据与 dedup_key。
ALTER TABLE system_groups
    ADD COLUMN IF NOT EXISTS telegram_chat_id BIGINT,
    ADD COLUMN IF NOT EXISTS first_contributor_tenant_id UUID,
    ADD COLUMN IF NOT EXISTS first_contributor_tenant_name VARCHAR(255) NOT NULL DEFAULT '',
    ADD COLUMN IF NOT EXISTS resolution_status VARCHAR(24) NOT NULL DEFAULT 'pending',
    ADD COLUMN IF NOT EXISTS member_count_updated_at TIMESTAMPTZ;

CREATE INDEX IF NOT EXISTS idx_system_groups_telegram_chat_id
    ON system_groups(telegram_chat_id) WHERE telegram_chat_id IS NOT NULL;

CREATE TABLE IF NOT EXISTS system_group_links (
    link_id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    sg_id UUID NOT NULL REFERENCES system_groups(sg_id) ON DELETE CASCADE,
    original_link VARCHAR(512) NOT NULL,
    normalized_link VARCHAR(512) NOT NULL UNIQUE,
    link_type VARCHAR(20) NOT NULL DEFAULT 'unknown',
    invite_hash VARCHAR(255),
    validity_status VARCHAR(24) NOT NULL DEFAULT 'pending',
    source VARCHAR(32) NOT NULL DEFAULT 'import',
    last_verified_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE INDEX IF NOT EXISTS idx_system_group_links_sg_id ON system_group_links(sg_id);
CREATE INDEX IF NOT EXISTS idx_system_group_links_invite_hash ON system_group_links(invite_hash) WHERE invite_hash IS NOT NULL;

CREATE TABLE IF NOT EXISTS system_group_contributions (
    contribution_id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    sg_id UUID NOT NULL REFERENCES system_groups(sg_id) ON DELETE CASCADE,
    tenant_id UUID REFERENCES tenants(tenant_id) ON DELETE SET NULL,
    tenant_name_snapshot VARCHAR(255) NOT NULL DEFAULT '',
    submitted_link VARCHAR(512) NOT NULL DEFAULT '',
    source VARCHAR(32) NOT NULL DEFAULT 'import',
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE INDEX IF NOT EXISTS idx_system_group_contributions_sg_id
    ON system_group_contributions(sg_id, created_at ASC);
CREATE INDEX IF NOT EXISTS idx_system_group_contributions_tenant
    ON system_group_contributions(tenant_id, created_at DESC);

-- 将历史单链接迁移到链接表；历史记录没有可靠贡献者身份，不伪造贡献者。
INSERT INTO system_group_links (sg_id, original_link, normalized_link, link_type, validity_status, source)
SELECT sg_id, tme_link,
       lower(regexp_replace(regexp_replace(trim(tme_link), '^https?://(www\.)?', ''), '/+$', '')),
       CASE WHEN gtype='private' THEN 'private_invite' WHEN gtype='channel' THEN 'public' ELSE 'public' END,
       CASE WHEN status='invalid' THEN 'invalid' ELSE 'pending' END,
       COALESCE(NULLIF(source,''), 'legacy')
FROM system_groups
WHERE COALESCE(tme_link,'') <> ''
ON CONFLICT (normalized_link) DO NOTHING;

-- +goose Down
DROP TABLE IF EXISTS system_group_contributions;
DROP TABLE IF EXISTS system_group_links;
DROP INDEX IF EXISTS idx_system_groups_telegram_chat_id;
ALTER TABLE system_groups
    DROP COLUMN IF EXISTS telegram_chat_id,
    DROP COLUMN IF EXISTS first_contributor_tenant_id,
    DROP COLUMN IF EXISTS first_contributor_tenant_name,
    DROP COLUMN IF EXISTS resolution_status,
    DROP COLUMN IF EXISTS member_count_updated_at;
