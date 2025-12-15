-- Add user_mappings table to map source users to destination users
-- This enables mannequin reclaim after migration

-- +goose Up
IF NOT EXISTS (SELECT * FROM sys.objects WHERE object_id = OBJECT_ID(N'[dbo].[user_mappings]') AND type in (N'U'))
BEGIN
    CREATE TABLE user_mappings (
        id BIGINT IDENTITY(1,1) PRIMARY KEY,
        source_login NVARCHAR(255) NOT NULL UNIQUE,
        source_email NVARCHAR(255),
        source_name NVARCHAR(255),
        destination_login NVARCHAR(255),
        destination_email NVARCHAR(255),
        mapping_status NVARCHAR(50) NOT NULL DEFAULT 'unmapped',
        mannequin_id NVARCHAR(255),
        mannequin_login NVARCHAR(255),
        reclaim_status NVARCHAR(50),
        reclaim_error NVARCHAR(MAX),
        created_at DATETIME2 NOT NULL DEFAULT GETUTCDATE(),
        updated_at DATETIME2 NOT NULL DEFAULT GETUTCDATE()
    );

    CREATE INDEX idx_user_mappings_status ON user_mappings(mapping_status);
    CREATE INDEX idx_user_mappings_source_email ON user_mappings(source_email);
    CREATE INDEX idx_user_mappings_destination_login ON user_mappings(destination_login);
END;

-- +goose Down
DROP TABLE IF EXISTS user_mappings;
