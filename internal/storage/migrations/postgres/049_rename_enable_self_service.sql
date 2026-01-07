-- Rename auth_require_identity_mapping_for_self_service to auth_enable_self_service
-- The new name is clearer: EnableSelfService = true means self-service is enabled

-- Rename the column
ALTER TABLE settings RENAME COLUMN auth_require_identity_mapping_for_self_service TO auth_enable_self_service;

