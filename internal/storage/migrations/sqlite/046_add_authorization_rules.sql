-- Add authorization rules columns to settings table
ALTER TABLE settings ADD COLUMN auth_migration_admin_teams TEXT;
ALTER TABLE settings ADD COLUMN auth_allow_org_admin_migrations BOOLEAN NOT NULL DEFAULT 0;
ALTER TABLE settings ADD COLUMN auth_allow_enterprise_admin_migrations BOOLEAN NOT NULL DEFAULT 0;
ALTER TABLE settings ADD COLUMN auth_require_identity_mapping_for_self_service BOOLEAN NOT NULL DEFAULT 0;

