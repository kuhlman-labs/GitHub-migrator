-- Add team_mappings table to map source teams to destination teams
-- This enables team permission migration planning

-- +goose Up
IF NOT EXISTS (SELECT * FROM sys.objects WHERE object_id = OBJECT_ID(N'[dbo].[team_mappings]') AND type in (N'U'))
BEGIN
    CREATE TABLE team_mappings (
        id BIGINT IDENTITY(1,1) PRIMARY KEY,
        source_org NVARCHAR(255) NOT NULL,
        source_team_slug NVARCHAR(255) NOT NULL,
        source_team_name NVARCHAR(255),
        destination_org NVARCHAR(255),
        destination_team_slug NVARCHAR(255),
        destination_team_name NVARCHAR(255),
        mapping_status NVARCHAR(50) NOT NULL DEFAULT 'unmapped',
        auto_created BIT NOT NULL DEFAULT 0,
        created_at DATETIME2 NOT NULL DEFAULT GETUTCDATE(),
        updated_at DATETIME2 NOT NULL DEFAULT GETUTCDATE(),
        
        CONSTRAINT UQ_team_mappings_source UNIQUE (source_org, source_team_slug)
    );

    CREATE INDEX idx_team_mappings_status ON team_mappings(mapping_status);
    CREATE INDEX idx_team_mappings_destination_org ON team_mappings(destination_org);
END;

-- +goose Down
DROP TABLE IF EXISTS team_mappings;
