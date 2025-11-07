-- +goose Up
-- Add validation tracking fields for post-migration validation
-- These fields track whether the migrated repository matches the source repository

-- Add validation_status field to track validation results
IF NOT EXISTS (SELECT * FROM sys.columns WHERE object_id = OBJECT_ID('repositories') AND name = 'validation_status')
BEGIN
    ALTER TABLE repositories ADD validation_status VARCHAR(50);
END
GO

-- Add validation_details field to store detailed comparison results (JSON)
IF NOT EXISTS (SELECT * FROM sys.columns WHERE object_id = OBJECT_ID('repositories') AND name = 'validation_details')
BEGIN
    ALTER TABLE repositories ADD validation_details NVARCHAR(MAX);
END
GO

-- Add destination_data field to store destination repository characteristics (JSON)
-- Only populated when validation finds mismatches
IF NOT EXISTS (SELECT * FROM sys.columns WHERE object_id = OBJECT_ID('repositories') AND name = 'destination_data')
BEGIN
    ALTER TABLE repositories ADD destination_data NVARCHAR(MAX);
END
GO



GO

-- +goose Down
-- Add rollback logic here
GO
