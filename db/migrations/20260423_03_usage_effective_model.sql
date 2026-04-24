BEGIN;

ALTER TABLE usage_records
  ADD COLUMN IF NOT EXISTS requested_model TEXT;

ALTER TABLE usage_records
  ADD COLUMN IF NOT EXISTS effective_model TEXT;

UPDATE usage_records
SET requested_model = model
WHERE requested_model IS NULL OR requested_model = '';

UPDATE usage_records
SET effective_model = model
WHERE effective_model IS NULL OR effective_model = '';

COMMIT;
