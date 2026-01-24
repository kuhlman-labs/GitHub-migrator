-- +goose Up
-- Add Copilot settings columns to the settings table

ALTER TABLE settings ADD COLUMN copilot_enabled INTEGER NOT NULL DEFAULT 0;
ALTER TABLE settings ADD COLUMN copilot_require_license INTEGER NOT NULL DEFAULT 1;
ALTER TABLE settings ADD COLUMN copilot_cli_path TEXT;
ALTER TABLE settings ADD COLUMN copilot_model TEXT;
ALTER TABLE settings ADD COLUMN copilot_max_tokens INTEGER;
ALTER TABLE settings ADD COLUMN copilot_session_timeout_min INTEGER NOT NULL DEFAULT 30;

-- Create table for Copilot chat sessions
CREATE TABLE IF NOT EXISTS copilot_sessions (
    id TEXT PRIMARY KEY,
    user_id TEXT NOT NULL,
    user_login TEXT NOT NULL,
    title TEXT,
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    expires_at DATETIME NOT NULL
);

CREATE INDEX IF NOT EXISTS idx_copilot_sessions_user_id ON copilot_sessions(user_id);
CREATE INDEX IF NOT EXISTS idx_copilot_sessions_expires_at ON copilot_sessions(expires_at);

-- Create table for Copilot chat messages
CREATE TABLE IF NOT EXISTS copilot_messages (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    session_id TEXT NOT NULL REFERENCES copilot_sessions(id) ON DELETE CASCADE,
    role TEXT NOT NULL, -- 'user', 'assistant', 'system'
    content TEXT NOT NULL,
    tool_calls TEXT, -- JSON array of tool calls made by assistant
    tool_results TEXT, -- JSON array of tool results
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_copilot_messages_session_id ON copilot_messages(session_id);

-- +goose Down
DROP TABLE IF EXISTS copilot_messages;
DROP TABLE IF EXISTS copilot_sessions;
-- SQLite doesn't support DROP COLUMN, so we'd need to recreate the table
-- For simplicity in development, we leave the columns
