-- Rename auth_require_identity_mapping_for_self_service to auth_enable_self_service
-- The new name is clearer: EnableSelfService = true means self-service is enabled

-- SQL Server uses sp_rename to rename columns
EXEC sp_rename 'settings.auth_require_identity_mapping_for_self_service', 'auth_enable_self_service', 'COLUMN';

