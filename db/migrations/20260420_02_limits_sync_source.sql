BEGIN;

ALTER TABLE limits
  ADD COLUMN IF NOT EXISTS sync_source TEXT NOT NULL DEFAULT 'tokens';

UPDATE limits
SET sync_source = 'tokens'
WHERE sync_source IS NULL OR sync_source = '';

COMMIT;
