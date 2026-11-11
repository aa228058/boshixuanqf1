-- +goose Up
-- 097: full-message dictionaries with independent dictionary/item activation.
-- Keep legacy word values and APIs intact; widen the storage column for full messages.
ALTER TABLE text_dicts
    ADD COLUMN IF NOT EXISTS enabled BOOLEAN NOT NULL DEFAULT TRUE;

ALTER TABLE text_dict_entries
    ALTER COLUMN word TYPE TEXT,
    ADD COLUMN IF NOT EXISTS enabled BOOLEAN NOT NULL DEFAULT TRUE;

-- Older deployments may have message_templates created outside the tracked
-- migration history. Migrate it when present, but do not make a fresh install
-- fail when that legacy table was never provisioned.
-- Keep the UUID suffix at the end so long template names cannot collide after
-- the 64-character dictionary-name limit is applied.
-- +goose StatementBegin
DO $migration$
BEGIN
    IF to_regclass('message_templates') IS NOT NULL THEN
        EXECUTE $sql$
            INSERT INTO text_dicts (tenant_id, name, enabled)
            SELECT mt.tenant_id,
                   LEFT('模板-' || mt.name, 54) || '-' || LEFT(mt.template_id::text, 8),
                   TRUE
            FROM message_templates mt
            WHERE NOT EXISTS (
                SELECT 1 FROM text_dicts d
                WHERE d.tenant_id = mt.tenant_id
                  AND d.name = LEFT('模板-' || mt.name, 54) || '-' || LEFT(mt.template_id::text, 8)
            )
        $sql$;

        EXECUTE $sql$
            INSERT INTO text_dict_entries (dict_id, word, weight, enabled)
            SELECT d.dict_id, mt.content, 1, TRUE
            FROM message_templates mt
            JOIN text_dicts d
              ON d.tenant_id = mt.tenant_id
             AND d.name = LEFT('模板-' || mt.name, 54) || '-' || LEFT(mt.template_id::text, 8)
            WHERE NOT EXISTS (
                SELECT 1 FROM text_dict_entries e
                WHERE e.dict_id = d.dict_id AND e.word = mt.content
            )
        $sql$;
    ELSE
        RAISE NOTICE 'message_templates table not found; skipping legacy template migration';
    END IF;
END
$migration$;
-- +goose StatementEnd

CREATE INDEX IF NOT EXISTS idx_text_dicts_tenant_enabled
    ON text_dicts(tenant_id, enabled);
CREATE INDEX IF NOT EXISTS idx_text_dict_entries_enabled
    ON text_dict_entries(dict_id, enabled);

-- +goose Down
DROP INDEX IF EXISTS idx_text_dict_entries_enabled;
DROP INDEX IF EXISTS idx_text_dicts_tenant_enabled;
ALTER TABLE text_dict_entries DROP COLUMN IF EXISTS enabled;
ALTER TABLE text_dicts DROP COLUMN IF EXISTS enabled;
