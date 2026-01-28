-- +goose Up
-- Add Copilot GH token column to settings table for CLI authentication
IF NOT EXISTS (SELECT * FROM sys.columns WHERE object_id = OBJECT_ID(N'settings') AND name = 'copilot_gh_token')
    ALTER TABLE settings ADD copilot_gh_token NVARCHAR(MAX);

-- Add model column to copilot_sessions table for per-session model tracking
IF NOT EXISTS (SELECT * FROM sys.columns WHERE object_id = OBJECT_ID(N'copilot_sessions') AND name = 'model')
    ALTER TABLE copilot_sessions ADD model NVARCHAR(MAX);

-- +goose Down
IF EXISTS (SELECT * FROM sys.columns WHERE object_id = OBJECT_ID(N'copilot_sessions') AND name = 'model')
    ALTER TABLE copilot_sessions DROP COLUMN model;
IF EXISTS (SELECT * FROM sys.columns WHERE object_id = OBJECT_ID(N'settings') AND name = 'copilot_gh_token')
    ALTER TABLE settings DROP COLUMN copilot_gh_token;
