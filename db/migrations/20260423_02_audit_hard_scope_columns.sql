ALTER TABLE audit_logs
ADD COLUMN IF NOT EXISTS project_id UUID;

ALTER TABLE audit_logs
ADD COLUMN IF NOT EXISTS api_key_id UUID;

CREATE INDEX IF NOT EXISTS audit_logs_org_created_idx
ON audit_logs (org_id, created_at DESC);

CREATE INDEX IF NOT EXISTS audit_logs_org_project_created_idx
ON audit_logs (org_id, project_id, created_at DESC);

CREATE INDEX IF NOT EXISTS audit_logs_org_apikey_created_idx
ON audit_logs (org_id, api_key_id, created_at DESC);
