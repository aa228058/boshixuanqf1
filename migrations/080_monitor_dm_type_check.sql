-- +goose Up
-- 080: monitor_rules.type 补 'dm'(私聊触达)
-- 001 建表时的 CHECK(type IN ('forward','reply')) 会拒绝 dm 规则落库(23514),
-- 先删后建放宽约束。老库 079 已执行过, 新文件不回头改 079。
ALTER TABLE monitor_rules DROP CONSTRAINT IF EXISTS monitor_rules_type_check;
ALTER TABLE monitor_rules ADD CONSTRAINT monitor_rules_type_check CHECK (type IN ('forward', 'reply', 'dm'));

-- +goose Down
ALTER TABLE monitor_rules DROP CONSTRAINT IF EXISTS monitor_rules_type_check;
ALTER TABLE monitor_rules ADD CONSTRAINT monitor_rules_type_check CHECK (type IN ('forward', 'reply'));
