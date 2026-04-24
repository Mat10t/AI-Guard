ALTER TABLE invitations
ADD COLUMN IF NOT EXISTS project_ids UUID[];

CREATE TABLE IF NOT EXISTS project_members (
  project_id UUID NOT NULL REFERENCES projects(id),
  user_id UUID NOT NULL REFERENCES users(id),
  assigned_by UUID NOT NULL REFERENCES users(id),
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  PRIMARY KEY (project_id, user_id)
);

CREATE INDEX IF NOT EXISTS project_members_user_idx ON project_members (user_id);
CREATE INDEX IF NOT EXISTS project_members_project_idx ON project_members (project_id);

INSERT INTO project_members (project_id, user_id, assigned_by)
SELECT p.id, u.id, p.created_by
FROM users u
JOIN projects p ON p.org_id = u.org_id
WHERE u.role IN ('PM', 'Dev')
  AND u.deleted_at IS NULL
  AND p.deleted_at IS NULL
ON CONFLICT (project_id, user_id) DO NOTHING;
