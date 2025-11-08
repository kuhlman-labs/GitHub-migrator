-- Add has_packages field to repositories table
ALTER TABLE repositories ADD COLUMN has_packages BOOLEAN DEFAULT FALSE;

-- Create index for efficient filtering by package presence
CREATE INDEX idx_repositories_has_packages ON repositories(has_packages);

