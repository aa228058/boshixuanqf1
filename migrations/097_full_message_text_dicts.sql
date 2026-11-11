-- +goose Up
-- 097: full-message dictionaries with independent dictionary/item activation.
-- Keep legacy word values and APIs intact; widen the storage column for full messages.
ALTER TABLE text_dicts
    ADD COLUMN IF NOT EXISTS enabled BOOLEAN NOT NULL DEFAULT TRUE;

ALTER TABLE text_dict_entries
    ALTER COLUMN word TYPE TEXT,
    ADD COLUMN IF NOT EXISTS enabled BOOLEAN NOT NULL DEFAULT TRUE;

-- Preserve every existing message template verbatim as one enabled full-message
-- dictionary entry. The UUID suffix avoids collisions between duplicate names.
INSERT INTO text_dicts (tenant_id, name, enabled)
SELECT mt.tenant_id,
       LEFT('模板-' || mt.name || '-' || LEFT(mt.template_id::text, 8), 64),
       TRUE
FROM message_templates mt
WHERE NOT EXISTS (
    SELECT 1 FROM text_dicts d
    WHERE d.tenant_id = mt.tenant_id
      AND d.name = LEFT('模板-' || mt.name || '-' || LEFT(mt.template_id::text, 8), 64)
);

INSERT INTO text_dict_entries (dict_id, word, weight, enabled)
SELECT d.dict_id, mt.content, 1, TRUE
FROM message_templates mt
JOIN text_dicts d
  ON d.tenant_id = mt.tenant_id
 AND d.name = LEFT('模板-' || mt.name || '-' || LEFT(mt.template_id::text, 8), 64)
WHERE NOT EXISTS (
    SELECT 1 FROM text_dict_entries e
    WHERE e.dict_id = d.dict_id AND e.word = mt.content
);

CREATE INDEX IF NOT EXISTS idx_text_dicts_tenant_enabled
    ON text_dicts(tenant_id, enabled);
CREATE INDEX IF NOT EXISTS idx_text_dict_entries_enabled
    ON text_dict_entries(dict_id, enabled);

-- +goose Down
DROP INDEX IF EXISTS idx_text_dict_entries_enabled;
DROP INDEX IF EXISTS idx_text_dicts_tenant_enabled;
ALTER TABLE text_dict_entries DROP COLUMN IF EXISTS enabled;
ALTER TABLE text_dicts DROP COLUMN IF EXISTS enabled;
