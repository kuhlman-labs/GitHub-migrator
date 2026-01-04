-- +goose Up
-- Add destination_enterprise_slug to settings table
-- This allows configuring the destination GitHub Enterprise slug in the UI
-- Used for enterprise admin authorization checks

ALTER TABLE settings ADD COLUMN IF NOT EXISTS destination_enterprise_slug TEXT;

-- +goose Down
ALTER TABLE settings DROP COLUMN IF EXISTS destination_enterprise_slug;

