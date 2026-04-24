-- 20260417_01_multi_key_analytics_gemini.sql
-- Multi-key support for api_keys + Gemini provider catalog seed.

BEGIN;

ALTER TABLE api_keys
  DROP CONSTRAINT IF EXISTS api_keys_project_id_key;

CREATE INDEX IF NOT EXISTS api_keys_project_status_idx ON api_keys (project_id, status);
CREATE INDEX IF NOT EXISTS api_keys_project_created_at_idx ON api_keys (project_id, created_at DESC);

INSERT INTO provider_models (id, provider, status, input_cost, output_cost)
VALUES
  ('gemini-2.5-flash', 'gemini', 'up', 0.00025, 0.00070)
ON CONFLICT (id) DO UPDATE
SET provider = EXCLUDED.provider,
    status = EXCLUDED.status,
    input_cost = EXCLUDED.input_cost,
    output_cost = EXCLUDED.output_cost;

INSERT INTO routing_rules (id, model_id, primary_provider, fallback_provider, timeout_ms, retry_count)
VALUES
  (gen_random_uuid(), 'gemini-2.5-flash', 'gemini', 'mock', 8000, 1)
ON CONFLICT (model_id) DO UPDATE
SET primary_provider = EXCLUDED.primary_provider,
    fallback_provider = EXCLUDED.fallback_provider,
    timeout_ms = EXCLUDED.timeout_ms,
    retry_count = EXCLUDED.retry_count;

COMMIT;
