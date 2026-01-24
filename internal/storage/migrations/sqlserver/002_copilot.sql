-- +goose Up
-- Add Copilot settings columns to the settings table

IF NOT EXISTS (SELECT * FROM sys.columns WHERE object_id = OBJECT_ID(N'settings') AND name = 'copilot_enabled')
    ALTER TABLE settings ADD copilot_enabled BIT NOT NULL DEFAULT 0;

IF NOT EXISTS (SELECT * FROM sys.columns WHERE object_id = OBJECT_ID(N'settings') AND name = 'copilot_require_license')
    ALTER TABLE settings ADD copilot_require_license BIT NOT NULL DEFAULT 1;

IF NOT EXISTS (SELECT * FROM sys.columns WHERE object_id = OBJECT_ID(N'settings') AND name = 'copilot_cli_path')
    ALTER TABLE settings ADD copilot_cli_path NVARCHAR(MAX);

IF NOT EXISTS (SELECT * FROM sys.columns WHERE object_id = OBJECT_ID(N'settings') AND name = 'copilot_model')
    ALTER TABLE settings ADD copilot_model NVARCHAR(MAX);

IF NOT EXISTS (SELECT * FROM sys.columns WHERE object_id = OBJECT_ID(N'settings') AND name = 'copilot_max_tokens')
    ALTER TABLE settings ADD copilot_max_tokens INT;

IF NOT EXISTS (SELECT * FROM sys.columns WHERE object_id = OBJECT_ID(N'settings') AND name = 'copilot_session_timeout_min')
    ALTER TABLE settings ADD copilot_session_timeout_min INT NOT NULL DEFAULT 30;

-- Create table for Copilot chat sessions
IF NOT EXISTS (SELECT * FROM sys.tables WHERE name = 'copilot_sessions')
CREATE TABLE copilot_sessions (
    id NVARCHAR(450) PRIMARY KEY,
    user_id NVARCHAR(450) NOT NULL,
    user_login NVARCHAR(450) NOT NULL,
    title NVARCHAR(MAX),
    created_at DATETIME2 NOT NULL DEFAULT GETUTCDATE(),
    updated_at DATETIME2 NOT NULL DEFAULT GETUTCDATE(),
    expires_at DATETIME2 NOT NULL
);

IF NOT EXISTS (SELECT * FROM sys.indexes WHERE name = 'idx_copilot_sessions_user_id')
    CREATE INDEX idx_copilot_sessions_user_id ON copilot_sessions(user_id);

IF NOT EXISTS (SELECT * FROM sys.indexes WHERE name = 'idx_copilot_sessions_expires_at')
    CREATE INDEX idx_copilot_sessions_expires_at ON copilot_sessions(expires_at);

-- Create table for Copilot chat messages
IF NOT EXISTS (SELECT * FROM sys.tables WHERE name = 'copilot_messages')
CREATE TABLE copilot_messages (
    id BIGINT IDENTITY(1,1) PRIMARY KEY,
    session_id NVARCHAR(450) NOT NULL REFERENCES copilot_sessions(id) ON DELETE CASCADE,
    role NVARCHAR(50) NOT NULL, -- 'user', 'assistant', 'system'
    content NVARCHAR(MAX) NOT NULL,
    tool_calls NVARCHAR(MAX), -- JSON array of tool calls made by assistant
    tool_results NVARCHAR(MAX), -- JSON array of tool results
    created_at DATETIME2 NOT NULL DEFAULT GETUTCDATE()
);

IF NOT EXISTS (SELECT * FROM sys.indexes WHERE name = 'idx_copilot_messages_session_id')
    CREATE INDEX idx_copilot_messages_session_id ON copilot_messages(session_id);

-- +goose Down
DROP TABLE IF EXISTS copilot_messages;
DROP TABLE IF EXISTS copilot_sessions;
IF EXISTS (SELECT * FROM sys.columns WHERE object_id = OBJECT_ID(N'settings') AND name = 'copilot_enabled')
    ALTER TABLE settings DROP COLUMN copilot_enabled;
IF EXISTS (SELECT * FROM sys.columns WHERE object_id = OBJECT_ID(N'settings') AND name = 'copilot_require_license')
    ALTER TABLE settings DROP COLUMN copilot_require_license;
IF EXISTS (SELECT * FROM sys.columns WHERE object_id = OBJECT_ID(N'settings') AND name = 'copilot_cli_path')
    ALTER TABLE settings DROP COLUMN copilot_cli_path;
IF EXISTS (SELECT * FROM sys.columns WHERE object_id = OBJECT_ID(N'settings') AND name = 'copilot_model')
    ALTER TABLE settings DROP COLUMN copilot_model;
IF EXISTS (SELECT * FROM sys.columns WHERE object_id = OBJECT_ID(N'settings') AND name = 'copilot_max_tokens')
    ALTER TABLE settings DROP COLUMN copilot_max_tokens;
IF EXISTS (SELECT * FROM sys.columns WHERE object_id = OBJECT_ID(N'settings') AND name = 'copilot_session_timeout_min')
    ALTER TABLE settings DROP COLUMN copilot_session_timeout_min;
