-- +goose Up
-- 监控规则新增"私聊触达"动作(type=dm), 支持组合子动作:
--   私聊文本 dm_content / 联系人名片 dm_contact* / 语音振铃 dm_call + dm_call_seconds
ALTER TABLE monitor_rules ADD COLUMN IF NOT EXISTS dm_content TEXT NOT NULL DEFAULT '';
ALTER TABLE monitor_rules ADD COLUMN IF NOT EXISTS dm_call BOOLEAN NOT NULL DEFAULT FALSE;
ALTER TABLE monitor_rules ADD COLUMN IF NOT EXISTS dm_call_seconds INT NOT NULL DEFAULT 5;
ALTER TABLE monitor_rules ADD COLUMN IF NOT EXISTS dm_contact BOOLEAN NOT NULL DEFAULT FALSE;
ALTER TABLE monitor_rules ADD COLUMN IF NOT EXISTS dm_contact_phone TEXT NOT NULL DEFAULT '';
ALTER TABLE monitor_rules ADD COLUMN IF NOT EXISTS dm_contact_first_name TEXT NOT NULL DEFAULT '';
ALTER TABLE monitor_rules ADD COLUMN IF NOT EXISTS dm_contact_last_name TEXT NOT NULL DEFAULT '';

-- +goose Down
ALTER TABLE monitor_rules DROP COLUMN IF EXISTS dm_content;
ALTER TABLE monitor_rules DROP COLUMN IF EXISTS dm_call;
ALTER TABLE monitor_rules DROP COLUMN IF EXISTS dm_call_seconds;
ALTER TABLE monitor_rules DROP COLUMN IF EXISTS dm_contact;
ALTER TABLE monitor_rules DROP COLUMN IF EXISTS dm_contact_phone;
ALTER TABLE monitor_rules DROP COLUMN IF EXISTS dm_contact_first_name;
ALTER TABLE monitor_rules DROP COLUMN IF EXISTS dm_contact_last_name;
