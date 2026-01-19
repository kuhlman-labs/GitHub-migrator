-- +goose Up
-- Add mannequin_org column to user_mappings table
IF NOT EXISTS (SELECT * FROM sys.columns WHERE object_id = OBJECT_ID('user_mappings') AND name = 'mannequin_org')
ALTER TABLE user_mappings ADD mannequin_org NVARCHAR(MAX);

-- Create index for mannequin_org
IF NOT EXISTS (SELECT * FROM sys.indexes WHERE name = 'idx_user_mappings_mannequin_org')
CREATE INDEX idx_user_mappings_mannequin_org ON user_mappings(mannequin_org);

-- User Mannequins table: tracks mannequin info per org (a user can have mannequins in multiple orgs)
IF NOT EXISTS (SELECT * FROM sys.tables WHERE name = 'user_mannequins')
CREATE TABLE user_mannequins (
    id BIGINT IDENTITY(1,1) PRIMARY KEY,
    source_login NVARCHAR(450) NOT NULL,
    mannequin_org NVARCHAR(450) NOT NULL,
    mannequin_id NVARCHAR(MAX) NOT NULL,
    mannequin_login NVARCHAR(MAX),
    reclaim_status NVARCHAR(MAX),
    reclaim_error NVARCHAR(MAX),
    created_at DATETIME2 NOT NULL DEFAULT GETUTCDATE(),
    updated_at DATETIME2 NOT NULL DEFAULT GETUTCDATE(),
    CONSTRAINT UQ_user_mannequins_source_org UNIQUE(source_login, mannequin_org)
);

IF NOT EXISTS (SELECT * FROM sys.indexes WHERE name = 'idx_user_mannequins_source')
CREATE INDEX idx_user_mannequins_source ON user_mannequins(source_login);

IF NOT EXISTS (SELECT * FROM sys.indexes WHERE name = 'idx_user_mannequins_org')
CREATE INDEX idx_user_mannequins_org ON user_mannequins(mannequin_org);

IF NOT EXISTS (SELECT * FROM sys.indexes WHERE name = 'idx_user_mannequins_status')
CREATE INDEX idx_user_mannequins_status ON user_mannequins(reclaim_status);

-- +goose Down
IF EXISTS (SELECT * FROM sys.indexes WHERE name = 'idx_user_mannequins_status')
DROP INDEX idx_user_mannequins_status ON user_mannequins;

IF EXISTS (SELECT * FROM sys.indexes WHERE name = 'idx_user_mannequins_org')
DROP INDEX idx_user_mannequins_org ON user_mannequins;

IF EXISTS (SELECT * FROM sys.indexes WHERE name = 'idx_user_mannequins_source')
DROP INDEX idx_user_mannequins_source ON user_mannequins;

IF EXISTS (SELECT * FROM sys.tables WHERE name = 'user_mannequins')
DROP TABLE user_mannequins;

IF EXISTS (SELECT * FROM sys.indexes WHERE name = 'idx_user_mappings_mannequin_org')
DROP INDEX idx_user_mappings_mannequin_org ON user_mappings;

IF EXISTS (SELECT * FROM sys.columns WHERE object_id = OBJECT_ID('user_mappings') AND name = 'mannequin_org')
ALTER TABLE user_mappings DROP COLUMN mannequin_org;
