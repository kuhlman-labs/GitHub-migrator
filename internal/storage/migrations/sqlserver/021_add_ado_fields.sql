-- Add Azure DevOps specific fields to repositories table
ALTER TABLE repositories ADD ado_project NVARCHAR(255) NULL;
ALTER TABLE repositories ADD ado_is_git BIT NOT NULL DEFAULT 1;
ALTER TABLE repositories ADD ado_has_boards BIT NOT NULL DEFAULT 0;
ALTER TABLE repositories ADD ado_has_pipelines BIT NOT NULL DEFAULT 0;
ALTER TABLE repositories ADD ado_has_ghas BIT NOT NULL DEFAULT 0;
ALTER TABLE repositories ADD ado_pull_request_count INT NOT NULL DEFAULT 0;
ALTER TABLE repositories ADD ado_work_item_count INT NOT NULL DEFAULT 0;
ALTER TABLE repositories ADD ado_branch_policy_count INT NOT NULL DEFAULT 0;
GO

-- Create ado_projects table for tracking Azure DevOps projects
CREATE TABLE ado_projects (
    id BIGINT PRIMARY KEY IDENTITY(1,1),
    organization NVARCHAR(255) NOT NULL,
    name NVARCHAR(255) NOT NULL,
    description NVARCHAR(MAX) NULL,
    repository_count INT NOT NULL DEFAULT 0,
    state NVARCHAR(50) NULL,
    visibility NVARCHAR(50) NULL,
    discovered_at DATETIME2 NOT NULL DEFAULT GETDATE(),
    updated_at DATETIME2 NOT NULL DEFAULT GETDATE(),
    CONSTRAINT unique_org_project UNIQUE (organization, name)
);
GO

-- Create index on organization for faster queries
CREATE INDEX idx_ado_projects_organization ON ado_projects(organization);
GO

