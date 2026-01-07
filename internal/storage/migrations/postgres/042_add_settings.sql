-- Create settings table for dynamic configuration (hot reload)
-- Only database DSN and server port remain in .env (require restart)
CREATE TABLE IF NOT EXISTS settings (
    id INTEGER PRIMARY KEY CHECK (id = 1),
    -- Destination GitHub configuration
    destination_base_url TEXT NOT NULL DEFAULT 'https://api.github.com',
    destination_token TEXT,
    destination_app_id BIGINT,
    destination_app_private_key TEXT,
    destination_app_installation_id BIGINT,
    -- Migration settings
    migration_workers INTEGER NOT NULL DEFAULT 5,
    migration_poll_interval_seconds INTEGER NOT NULL DEFAULT 30,
    migration_dest_repo_exists_action TEXT NOT NULL DEFAULT 'fail',
    migration_visibility_public TEXT NOT NULL DEFAULT 'private',
    migration_visibility_internal TEXT NOT NULL DEFAULT 'private',
    -- Auth settings
    auth_enabled BOOLEAN NOT NULL DEFAULT FALSE,
    auth_session_secret TEXT,
    auth_session_duration_hours INTEGER NOT NULL DEFAULT 24,
    auth_callback_url TEXT,
    auth_frontend_url TEXT NOT NULL DEFAULT 'http://localhost:3000',
    -- Timestamps
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

-- Insert the single settings record
INSERT INTO settings (id) VALUES (1) ON CONFLICT (id) DO NOTHING;

