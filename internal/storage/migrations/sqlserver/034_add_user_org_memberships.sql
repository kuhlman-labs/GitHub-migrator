-- Add user_org_memberships table to track which users belong to which organizations
-- This enables organizing users by source organization for mannequin reclamation

-- +goose Up
IF NOT EXISTS (SELECT * FROM sys.objects WHERE object_id = OBJECT_ID(N'[dbo].[user_org_memberships]') AND type in (N'U'))
BEGIN
    CREATE TABLE user_org_memberships (
        id BIGINT IDENTITY(1,1) PRIMARY KEY,
        user_login NVARCHAR(255) NOT NULL,
        organization NVARCHAR(255) NOT NULL,
        role NVARCHAR(50) NOT NULL DEFAULT 'member',
        discovered_at DATETIME2 NOT NULL DEFAULT GETUTCDATE(),
        
        CONSTRAINT UQ_user_org_memberships UNIQUE (user_login, organization)
    );

    CREATE INDEX idx_user_org_memberships_login ON user_org_memberships(user_login);
    CREATE INDEX idx_user_org_memberships_org ON user_org_memberships(organization);
END;

-- +goose Down
DROP TABLE IF EXISTS user_org_memberships;

