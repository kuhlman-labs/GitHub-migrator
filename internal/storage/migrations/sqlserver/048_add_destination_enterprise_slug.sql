-- Add destination_enterprise_slug to settings table
-- This allows configuring the destination GitHub Enterprise slug in the UI
-- Used for enterprise admin authorization checks

IF NOT EXISTS (SELECT * FROM sys.columns WHERE object_id = OBJECT_ID('settings') AND name = 'destination_enterprise_slug')
BEGIN
    ALTER TABLE settings ADD destination_enterprise_slug NVARCHAR(255);
END;

