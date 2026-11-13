-- +goose Up
-- Migration 082 lowercased every legacy normalized Telegram link. Private
-- invite hashes are case-sensitive, so rebuild their canonical keys from the
-- original URL while keeping public usernames case-insensitive.
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

-- +goose Down
-- Do not lowercase private invite hashes again: doing so can collapse
-- distinct, case-sensitive Telegram invitations.
