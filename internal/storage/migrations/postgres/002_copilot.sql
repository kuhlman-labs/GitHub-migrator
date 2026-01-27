-- +goose Up
-- Add Copilot settings columns to the settings table

ALTER TABLE settings ADD COLUMN IF NOT EXISTS copilot_enabled BOOLEAN NOT NULL DEFAULT FALSE;
ALTER TABLE settings ADD COLUMN IF NOT EXISTS copilot_require_license BOOLEAN NOT NULL DEFAULT TRUE;
ALTER TABLE settings ADD COLUMN IF NOT EXISTS copilot_cli_path TEXT;
ALTER TABLE settings ADD COLUMN IF NOT EXISTS copilot_model TEXT;
ALTER TABLE settings ADD COLUMN IF NOT EXISTS copilot_session_timeout_min INTEGER NOT NULL DEFAULT 30;
ALTER TABLE settings ADD COLUMN IF NOT EXISTS copilot_streaming BOOLEAN NOT NULL DEFAULT TRUE;
ALTER TABLE settings ADD COLUMN IF NOT EXISTS copilot_log_level TEXT NOT NULL DEFAULT 'info';

-- Create table for Copilot chat sessions
CREATE TABLE IF NOT EXISTS copilot_sessions (
    id TEXT PRIMARY KEY,
    user_id TEXT NOT NULL,
    user_login TEXT NOT NULL,
    title TEXT,
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT CURRENT_TIMESTAMP,
    expires_at TIMESTAMP WITH TIME ZONE NOT NULL
);

CREATE INDEX IF NOT EXISTS idx_copilot_sessions_user_id ON copilot_sessions(user_id);
CREATE INDEX IF NOT EXISTS idx_copilot_sessions_expires_at ON copilot_sessions(expires_at);

-- Create table for Copilot chat messages
CREATE TABLE IF NOT EXISTS copilot_messages (
    id BIGSERIAL PRIMARY KEY,
    session_id TEXT NOT NULL REFERENCES copilot_sessions(id) ON DELETE CASCADE,
    role TEXT NOT NULL, -- 'user', 'assistant', 'system'
    content TEXT NOT NULL,
    tool_calls JSONB, -- JSON array of tool calls made by assistant
    tool_results JSONB, -- JSON array of tool results
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_copilot_messages_session_id ON copilot_messages(session_id);

-- +goose Down
DROP TABLE IF EXISTS copilot_messages;
DROP TABLE IF EXISTS copilot_sessions;
ALTER TABLE settings DROP COLUMN IF EXISTS copilot_enabled;
ALTER TABLE settings DROP COLUMN IF EXISTS copilot_require_license;
ALTER TABLE settings DROP COLUMN IF EXISTS copilot_cli_path;
ALTER TABLE settings DROP COLUMN IF EXISTS copilot_model;
ALTER TABLE settings DROP COLUMN IF EXISTS copilot_session_timeout_min;
ALTER TABLE settings DROP COLUMN IF EXISTS copilot_streaming;
ALTER TABLE settings DROP COLUMN IF EXISTS copilot_log_level;
