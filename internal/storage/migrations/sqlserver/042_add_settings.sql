-- Create settings table for dynamic configuration (hot reload)
-- Only database DSN and server port remain in .env (require restart)
IF NOT EXISTS (SELECT * FROM sys.tables WHERE name = 'settings')
BEGIN
    CREATE TABLE settings (
        id INT PRIMARY KEY CHECK (id = 1),
        -- Destination GitHub configuration
        destination_base_url NVARCHAR(500) NOT NULL DEFAULT 'https://api.github.com',
        destination_token NVARCHAR(500),
        destination_app_id BIGINT,
        destination_app_private_key NVARCHAR(MAX),
        destination_app_installation_id BIGINT,
        -- Migration settings
        migration_workers INT NOT NULL DEFAULT 5,
        migration_poll_interval_seconds INT NOT NULL DEFAULT 30,
        migration_dest_repo_exists_action NVARCHAR(50) NOT NULL DEFAULT 'fail',
        migration_visibility_public NVARCHAR(50) NOT NULL DEFAULT 'private',
        migration_visibility_internal NVARCHAR(50) NOT NULL DEFAULT 'private',
        -- Auth settings
        auth_enabled BIT NOT NULL DEFAULT 0,
        auth_session_secret NVARCHAR(500),
        auth_session_duration_hours INT NOT NULL DEFAULT 24,
        auth_callback_url NVARCHAR(500),
        auth_frontend_url NVARCHAR(500) NOT NULL DEFAULT 'http://localhost:3000',
        -- Timestamps
        created_at DATETIME2 DEFAULT GETUTCDATE(),
        updated_at DATETIME2 DEFAULT GETUTCDATE()
    );
END;
GO

-- Insert the single settings record if it doesn't exist
IF NOT EXISTS (SELECT 1 FROM settings WHERE id = 1)
BEGIN
    INSERT INTO settings (id) VALUES (1);
END;
GO

