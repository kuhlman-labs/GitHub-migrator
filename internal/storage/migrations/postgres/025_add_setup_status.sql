-- Create setup_status table to track server setup completion
CREATE TABLE IF NOT EXISTS setup_status (
    id INTEGER PRIMARY KEY CHECK (id = 1),
    setup_completed BOOLEAN NOT NULL DEFAULT FALSE,
    completed_at TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- Insert the single record for setup status
INSERT INTO setup_status (id, setup_completed) 
VALUES (1, FALSE)
ON CONFLICT (id) DO NOTHING;

