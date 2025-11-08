-- Add validation tracking fields for post-migration validation
-- These fields track whether the migrated repository matches the source repository

-- Add validation_status field to track validation results
ALTER TABLE repositories ADD COLUMN validation_status VARCHAR(50);

-- Add validation_details field to store detailed comparison results (JSON)
ALTER TABLE repositories ADD COLUMN validation_details TEXT;

-- Add destination_data field to store destination repository characteristics (JSON)
-- Only populated when validation finds mismatches
ALTER TABLE repositories ADD COLUMN destination_data TEXT;

