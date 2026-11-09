-- +goose Up
-- Keep private Telegram invite hashes case-sensitive. Migration 082 lowercased
-- every legacy URL, which could make a later import of the same invite appear
-- to be a second link because runtime normalization preserves hash case.
WITH source_paths AS (
    SELECT link_id,
           regexp_replace(
               split_part(split_part(trim(original_link), '?', 1), '#', 1),
               '^((https?://)?(www\\.)?(t\\.me|telegram\\.me|telegram\\.dog)/)',
               '',
               'i'
           ) AS raw_path
    FROM system_group_links
),
canonical_paths AS (
    SELECT link_id,
           CASE
               WHEN lower(raw_path) LIKE 's/%' THEN substring(raw_path FROM 3)
               WHEN lower(raw_path) LIKE 'joinchat/%' THEN '+' || substring(raw_path FROM 10)
               ELSE raw_path
           END AS path
    FROM source_paths
)
UPDATE system_group_links AS links
SET normalized_link = CASE
    WHEN left(canonical_paths.path, 1) = '+'
        THEN 't.me/' || canonical_paths.path
    ELSE 't.me/' || lower(canonical_paths.path)
END
FROM canonical_paths
WHERE links.link_id = canonical_paths.link_id
  AND canonical_paths.path <> '';

-- ref_count is the number of distinct tenant/link contributions, with a
-- minimum baseline of one for legacy/system-discovered groups without a
-- tenant contribution row.
UPDATE system_groups AS groups
SET ref_count = GREATEST(
    1,
    (SELECT COUNT(*)::integer
     FROM system_group_contributions AS contributions
     WHERE contributions.sg_id = groups.sg_id)
);

-- +goose Down
-- The previous lower-cased private invite keys cannot be reconstructed
-- without potentially collapsing distinct, case-sensitive invite hashes.
-- Keep corrected canonical keys on rollback rather than corrupting identity.
