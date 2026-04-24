CREATE EXTENSION IF NOT EXISTS pgcrypto;

CREATE TABLE IF NOT EXISTS organizations (
  id UUID PRIMARY KEY,
  name TEXT NOT NULL,
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS users (
  id UUID PRIMARY KEY,
  org_id UUID NOT NULL REFERENCES organizations(id),
  email TEXT NOT NULL,
  password_hash TEXT NOT NULL,
  role TEXT NOT NULL,
  deleted_at TIMESTAMPTZ,
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE UNIQUE INDEX IF NOT EXISTS users_email_active_unique_idx
ON users (email)
WHERE deleted_at IS NULL;

CREATE TABLE IF NOT EXISTS sessions (
  id UUID PRIMARY KEY,
  user_id UUID NOT NULL REFERENCES users(id),
  refresh_hash TEXT NOT NULL,
  expires_at TIMESTAMPTZ NOT NULL,
  revoked_at TIMESTAMPTZ
);

CREATE TABLE IF NOT EXISTS invitations (
  id UUID PRIMARY KEY,
  org_id UUID NOT NULL REFERENCES organizations(id),
  email TEXT NOT NULL,
  role TEXT NOT NULL,
  project_ids UUID[],
  token_hash TEXT NOT NULL,
  expires_at TIMESTAMPTZ NOT NULL,
  accepted_at TIMESTAMPTZ,
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

ALTER TABLE invitations
ADD COLUMN IF NOT EXISTS project_ids UUID[];

CREATE TABLE IF NOT EXISTS projects (
  id UUID PRIMARY KEY,
  org_id UUID NOT NULL REFERENCES organizations(id),
  name TEXT NOT NULL,
  created_by UUID NOT NULL REFERENCES users(id),
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  deleted_at TIMESTAMPTZ
);

CREATE TABLE IF NOT EXISTS project_members (
  project_id UUID NOT NULL REFERENCES projects(id),
  user_id UUID NOT NULL REFERENCES users(id),
  assigned_by UUID NOT NULL REFERENCES users(id),
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  PRIMARY KEY (project_id, user_id)
);

CREATE INDEX IF NOT EXISTS project_members_user_idx ON project_members (user_id);
CREATE INDEX IF NOT EXISTS project_members_project_idx ON project_members (project_id);

CREATE TABLE IF NOT EXISTS api_keys (
  id UUID PRIMARY KEY,
  project_id UUID NOT NULL REFERENCES projects(id),
  key_hash TEXT NOT NULL UNIQUE,
  key_prefix TEXT NOT NULL,
  key_value TEXT,
  name TEXT NOT NULL,
  status TEXT NOT NULL,
  created_by UUID NOT NULL REFERENCES users(id),
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  revoked_at TIMESTAMPTZ
);

ALTER TABLE api_keys
ADD COLUMN IF NOT EXISTS key_value TEXT;
ALTER TABLE api_keys
ADD COLUMN IF NOT EXISTS name TEXT;
UPDATE api_keys
SET name = 'Key ' || key_prefix
WHERE name IS NULL OR btrim(name) = '';
ALTER TABLE api_keys
ALTER COLUMN name SET NOT NULL;
ALTER TABLE api_keys
DROP CONSTRAINT IF EXISTS api_keys_project_id_key;

CREATE INDEX IF NOT EXISTS api_keys_project_status_idx ON api_keys (project_id, status);
CREATE INDEX IF NOT EXISTS api_keys_project_created_at_idx ON api_keys (project_id, created_at DESC);

CREATE TABLE IF NOT EXISTS limits (
  id UUID PRIMARY KEY,
  scope_type TEXT NOT NULL,
  scope_id UUID NOT NULL,
  token_limit BIGINT NOT NULL,
  period TEXT NOT NULL,
  budget_limit_usd NUMERIC(18,12),
  billing_model TEXT,
  usd_per_token NUMERIC(18,12),
  sync_source TEXT NOT NULL DEFAULT 'tokens',
  created_by UUID NOT NULL REFERENCES users(id),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  UNIQUE(scope_type, scope_id)
);

ALTER TABLE limits
ADD COLUMN IF NOT EXISTS budget_limit_usd NUMERIC(18,12);
ALTER TABLE limits
ADD COLUMN IF NOT EXISTS billing_model TEXT;
ALTER TABLE limits
ADD COLUMN IF NOT EXISTS usd_per_token NUMERIC(18,12);
ALTER TABLE limits
ADD COLUMN IF NOT EXISTS sync_source TEXT NOT NULL DEFAULT 'tokens';
UPDATE limits
SET sync_source = 'tokens'
WHERE sync_source IS NULL OR sync_source = '';

CREATE TABLE IF NOT EXISTS provider_models (
  id TEXT PRIMARY KEY,
  provider TEXT NOT NULL,
  status TEXT NOT NULL,
  input_cost NUMERIC(12,6) NOT NULL,
  output_cost NUMERIC(12,6) NOT NULL,
  pricing_source TEXT NOT NULL DEFAULT 'seed',
  pricing_updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

ALTER TABLE provider_models
ADD COLUMN IF NOT EXISTS pricing_source TEXT NOT NULL DEFAULT 'seed';
ALTER TABLE provider_models
ADD COLUMN IF NOT EXISTS pricing_updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW();

CREATE TABLE IF NOT EXISTS routing_rules (
  id UUID PRIMARY KEY,
  model_id TEXT NOT NULL REFERENCES provider_models(id),
  primary_provider TEXT NOT NULL,
  fallback_provider TEXT NOT NULL,
  timeout_ms INT NOT NULL,
  retry_count INT NOT NULL,
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  UNIQUE(model_id)
);

CREATE TABLE IF NOT EXISTS project_routing_settings (
  project_id UUID PRIMARY KEY REFERENCES projects(id),
  fallback_model_id TEXT REFERENCES provider_models(id),
  updated_by UUID NOT NULL REFERENCES users(id),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS technical_logs (
  id BIGSERIAL PRIMARY KEY,
  request_id TEXT NOT NULL,
  org_id UUID NOT NULL,
  project_id UUID NOT NULL,
  api_key_id UUID NOT NULL,
  model TEXT NOT NULL,
  status TEXT NOT NULL,
  error_code TEXT,
  retries INT NOT NULL DEFAULT 0,
  fallback_used BOOLEAN NOT NULL DEFAULT FALSE,
  fallback_model TEXT,
  input_tokens INT NOT NULL DEFAULT 0,
  output_tokens INT NOT NULL DEFAULT 0,
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

ALTER TABLE technical_logs
ADD COLUMN IF NOT EXISTS fallback_model TEXT;

CREATE TABLE IF NOT EXISTS audit_logs (
  id BIGSERIAL PRIMARY KEY,
  org_id UUID NOT NULL,
  project_id UUID,
  api_key_id UUID,
  actor_user_id UUID,
  action TEXT NOT NULL,
  object_type TEXT NOT NULL,
  object_id TEXT NOT NULL,
  details JSONB NOT NULL DEFAULT '{}'::jsonb,
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

ALTER TABLE audit_logs
ADD COLUMN IF NOT EXISTS project_id UUID;
ALTER TABLE audit_logs
ADD COLUMN IF NOT EXISTS api_key_id UUID;

CREATE INDEX IF NOT EXISTS audit_logs_org_created_idx ON audit_logs (org_id, created_at DESC);
CREATE INDEX IF NOT EXISTS audit_logs_org_project_created_idx ON audit_logs (org_id, project_id, created_at DESC);
CREATE INDEX IF NOT EXISTS audit_logs_org_apikey_created_idx ON audit_logs (org_id, api_key_id, created_at DESC);

CREATE TABLE IF NOT EXISTS usage_records (
  id BIGSERIAL PRIMARY KEY,
  org_id UUID NOT NULL,
  project_id UUID NOT NULL,
  api_key_id UUID NOT NULL,
  model TEXT NOT NULL,
  requested_model TEXT,
  effective_model TEXT,
  input_tokens INT NOT NULL,
  output_tokens INT NOT NULL,
  total_tokens INT NOT NULL,
  estimated_cost NUMERIC(12,6) NOT NULL,
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

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

INSERT INTO provider_models (id, provider, status, input_cost, output_cost, pricing_source, pricing_updated_at)
VALUES
  ('gpt-5.4-mini', 'openai', 'up', 0.00020, 0.00080, 'seed', NOW()),
  ('gpt-5.4', 'openai', 'up', 0.00150, 0.00600, 'seed', NOW()),
  ('gemini-2.5-flash', 'gemini', 'up', 0.00025, 0.00070, 'seed', NOW()),
  ('mock-fast', 'mock', 'up', 0.00000, 0.00000, 'seed', NOW())
ON CONFLICT (id) DO NOTHING;

INSERT INTO routing_rules (id, model_id, primary_provider, fallback_provider, timeout_ms, retry_count)
VALUES
  (gen_random_uuid(), 'gpt-5.4-mini', 'openai', 'mock', 8000, 1),
  (gen_random_uuid(), 'gpt-5.4', 'openai', 'mock', 8000, 1),
  (gen_random_uuid(), 'gemini-2.5-flash', 'gemini', 'mock', 8000, 1),
  (gen_random_uuid(), 'mock-fast', 'mock', 'mock', 3000, 0)
ON CONFLICT (model_id) DO NOTHING;

INSERT INTO project_members (project_id, user_id, assigned_by)
SELECT p.id, u.id, p.created_by
FROM users u
JOIN projects p ON p.org_id = u.org_id
WHERE u.role IN ('PM', 'Dev')
  AND u.deleted_at IS NULL
  AND p.deleted_at IS NULL
ON CONFLICT (project_id, user_id) DO NOTHING;
