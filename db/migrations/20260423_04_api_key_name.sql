BEGIN;

ALTER TABLE api_keys
  ADD COLUMN IF NOT EXISTS name TEXT;

UPDATE api_keys
SET name = 'Key ' || key_prefix
WHERE name IS NULL OR btrim(name) = '';

ALTER TABLE api_keys
  ALTER COLUMN name SET NOT NULL;

COMMIT;
