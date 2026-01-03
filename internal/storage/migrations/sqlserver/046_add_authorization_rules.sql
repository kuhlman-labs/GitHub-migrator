-- Add authorization rules columns to settings table
ALTER TABLE settings ADD auth_migration_admin_teams NVARCHAR(MAX);
ALTER TABLE settings ADD auth_allow_org_admin_migrations BIT NOT NULL DEFAULT 0;
ALTER TABLE settings ADD auth_allow_enterprise_admin_migrations BIT NOT NULL DEFAULT 0;
ALTER TABLE settings ADD auth_require_identity_mapping_for_self_service BIT NOT NULL DEFAULT 0;

