-- +goose Up
-- PR #14 follow-up: ref_count is a derived count of unique tenant + submitted-link
-- contributions. Older versions incremented it on repeated imports and did not
-- consistently repair counts after canonical identity merges.
UPDATE system_groups AS sg
SET ref_count = GREATEST(
    1,
    (
        SELECT COUNT(DISTINCT (c.tenant_id, c.submitted_link))
        FROM system_group_contributions AS c
        WHERE c.sg_id = sg.sg_id
    )
);

-- +goose Down
-- Intentionally no-op: the previous ref_count values were non-idempotent and
-- cannot be reconstructed safely after the corrected derived values are stored.
