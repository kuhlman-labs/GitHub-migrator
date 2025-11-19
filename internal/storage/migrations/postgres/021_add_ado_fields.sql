-- Add Azure DevOps specific fields to repositories table
ALTER TABLE repositories ADD COLUMN ado_project VARCHAR(255);
ALTER TABLE repositories ADD COLUMN ado_is_git BOOLEAN DEFAULT TRUE;
ALTER TABLE repositories ADD COLUMN ado_has_boards BOOLEAN DEFAULT FALSE;
ALTER TABLE repositories ADD COLUMN ado_has_pipelines BOOLEAN DEFAULT FALSE;
ALTER TABLE repositories ADD COLUMN ado_has_ghas BOOLEAN DEFAULT FALSE;
ALTER TABLE repositories ADD COLUMN ado_pull_request_count INTEGER DEFAULT 0;
ALTER TABLE repositories ADD COLUMN ado_work_item_count INTEGER DEFAULT 0;
ALTER TABLE repositories ADD COLUMN ado_branch_policy_count INTEGER DEFAULT 0;

-- Create ado_projects table for tracking Azure DevOps projects
CREATE TABLE ado_projects (
    id BIGSERIAL PRIMARY KEY,
    organization VARCHAR(255) NOT NULL,
    name VARCHAR(255) NOT NULL,
    description TEXT,
    repository_count INTEGER DEFAULT 0,
    state VARCHAR(50),
    visibility VARCHAR(50),
    discovered_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    CONSTRAINT unique_org_project UNIQUE (organization, name)
);

-- Create index on organization for faster queries
CREATE INDEX idx_ado_projects_organization ON ado_projects(organization);

