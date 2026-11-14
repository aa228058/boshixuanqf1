-- +goose Up
-- campaigns / squads 两套业务在 R114 已整建制删除(引擎/服务/处理器/仓库/路由),
-- 表却一直留在库里占空间、也让人误以为功能还在。这里统一清掉。
DROP TABLE IF EXISTS campaign_targets CASCADE;
DROP TABLE IF EXISTS campaign_accounts CASCADE;
DROP TABLE IF EXISTS campaigns CASCADE;
DROP TABLE IF EXISTS squad_members CASCADE;
DROP TABLE IF EXISTS squads CASCADE;
DROP TABLE IF EXISTS _backup_account_groups CASCADE;

-- +goose Down
-- 不可逆：这些表对应的业务代码已不存在，恢复表结构没有意义。
SELECT 1;
