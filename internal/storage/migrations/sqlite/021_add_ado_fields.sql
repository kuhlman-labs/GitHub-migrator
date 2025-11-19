-- Add Azure DevOps specific fields to repositories table
ALTER TABLE repositories ADD COLUMN ado_project TEXT;
ALTER TABLE repositories ADD COLUMN ado_is_git BOOLEAN DEFAULT 1;
ALTER TABLE repositories ADD COLUMN ado_has_boards BOOLEAN DEFAULT 0;
ALTER TABLE repositories ADD COLUMN ado_has_pipelines BOOLEAN DEFAULT 0;
ALTER TABLE repositories ADD COLUMN ado_has_ghas BOOLEAN DEFAULT 0;
ALTER TABLE repositories ADD COLUMN ado_pull_request_count INTEGER DEFAULT 0;
ALTER TABLE repositories ADD COLUMN ado_work_item_count INTEGER DEFAULT 0;
ALTER TABLE repositories ADD COLUMN ado_branch_policy_count INTEGER DEFAULT 0;

-- Create ado_projects table for tracking Azure DevOps projects
CREATE TABLE ado_projects (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    organization TEXT NOT NULL,
    name TEXT NOT NULL,
    description TEXT,
    repository_count INTEGER DEFAULT 0,
    state TEXT,
    visibility TEXT,
    discovered_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    UNIQUE(organization, name)
);

-- Create index on organization for faster queries
CREATE INDEX idx_ado_projects_organization ON ado_projects(organization);

