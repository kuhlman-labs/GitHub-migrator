-- Migration: Add enterprise_slug field to sources table
-- This enables pre-populating the enterprise slug for GitHub sources during discovery

-- Add enterprise_slug column to sources table
-- This is optional for GitHub sources and allows storing the default enterprise slug
ALTER TABLE sources ADD enterprise_slug NVARCHAR(255);

