-- Create setup_status table to track server setup completion
IF NOT EXISTS (SELECT * FROM sys.tables WHERE name = 'setup_status')
BEGIN
    CREATE TABLE setup_status (
        id INTEGER PRIMARY KEY CHECK (id = 1),
        setup_completed BIT NOT NULL DEFAULT 0,
        completed_at DATETIME2,
        updated_at DATETIME2 DEFAULT GETDATE()
    );
    
    -- Insert the single record for setup status
    INSERT INTO setup_status (id, setup_completed) VALUES (1, 0);
END;

