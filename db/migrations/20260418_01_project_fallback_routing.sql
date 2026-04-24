CREATE TABLE IF NOT EXISTS project_routing_settings (
  project_id UUID PRIMARY KEY REFERENCES projects(id),
  fallback_model_id TEXT REFERENCES provider_models(id),
  updated_by UUID NOT NULL REFERENCES users(id),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

ALTER TABLE technical_logs
ADD COLUMN IF NOT EXISTS fallback_model TEXT;
