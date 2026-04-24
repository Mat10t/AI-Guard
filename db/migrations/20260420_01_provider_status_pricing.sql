BEGIN;

ALTER TABLE provider_models
  ADD COLUMN IF NOT EXISTS pricing_source TEXT NOT NULL DEFAULT 'seed';

ALTER TABLE provider_models
  ADD COLUMN IF NOT EXISTS pricing_updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW();

UPDATE provider_models
SET pricing_source = 'seed'
WHERE pricing_source IS NULL OR pricing_source = '';

UPDATE provider_models
SET pricing_updated_at = COALESCE(pricing_updated_at, created_at, NOW())
WHERE pricing_updated_at IS NULL;

COMMIT;
