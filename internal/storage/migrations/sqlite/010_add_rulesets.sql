-- Add rulesets field to repositories table
-- Rulesets are a newer version of branch protections that don't migrate with GEI APIs

ALTER TABLE repositories ADD COLUMN has_rulesets BOOLEAN DEFAULT FALSE;

-- Add index for filtering by rulesets
CREATE INDEX idx_repositories_has_rulesets ON repositories(has_rulesets);

