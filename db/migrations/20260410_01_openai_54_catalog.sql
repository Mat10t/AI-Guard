BEGIN;

-- 1) Upsert MVP catalog models (demo prices for 5.4 line).
INSERT INTO provider_models (id, provider, status, input_cost, output_cost)
VALUES
  ('gpt-5.4-mini', 'openai', 'up', 0.00020, 0.00080),
  ('gpt-5.4', 'openai', 'up', 0.00150, 0.00600),
  ('mock-fast', 'mock', 'up', 0.00000, 0.00000)
ON CONFLICT (id) DO UPDATE
SET
  provider = EXCLUDED.provider,
  status = EXCLUDED.status,
  input_cost = EXCLUDED.input_cost,
  output_cost = EXCLUDED.output_cost;

-- 2) Migrate legacy project/key limits from gpt-4o-mini to gpt-5.4-mini.
-- Keep token_limit as source of truth; recalculate budget and usd_per_token.
UPDATE limits
SET
  billing_model = 'gpt-5.4-mini',
  usd_per_token = ((0.00020 + 0.00080) / 2.0) / 1000.0,
  budget_limit_usd = ROUND((token_limit::numeric * ((((0.00020 + 0.00080) / 2.0) / 1000.0)::numeric)), 12),
  updated_at = NOW()
WHERE billing_model = 'gpt-4o-mini';

-- 3) Upsert routing profiles for 5.4 catalog.
INSERT INTO routing_rules (id, model_id, primary_provider, fallback_provider, timeout_ms, retry_count)
VALUES
  (gen_random_uuid(), 'gpt-5.4-mini', 'openai', 'mock', 8000, 1),
  (gen_random_uuid(), 'gpt-5.4', 'openai', 'mock', 8000, 1),
  (gen_random_uuid(), 'mock-fast', 'mock', 'mock', 3000, 0)
ON CONFLICT (model_id) DO UPDATE
SET
  primary_provider = EXCLUDED.primary_provider,
  fallback_provider = EXCLUDED.fallback_provider,
  timeout_ms = EXCLUDED.timeout_ms,
  retry_count = EXCLUDED.retry_count;

-- 4) Drop legacy model/routing profile.
DELETE FROM routing_rules WHERE model_id = 'gpt-4o-mini';
DELETE FROM provider_models WHERE id = 'gpt-4o-mini';

COMMIT;
