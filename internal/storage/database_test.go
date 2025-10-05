package storage

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/brettkuhlman/github-migrator/internal/config"
)

func TestNewDatabase(t *testing.T) {
	// Create temporary directory for test database
	tmpDir, err := os.MkdirTemp("", "db-test-*")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	cfg := config.DatabaseConfig{
		Type: "sqlite",
		DSN:  filepath.Join(tmpDir, "test.db"),
	}

	db, err := NewDatabase(cfg)
	if err != nil {
		t.Fatalf("NewDatabase() error = %v", err)
	}
	defer db.Close()

	if db.db == nil {
		t.Error("NewDatabase() db.db is nil")
	}

	// Verify connection works
	if err := db.db.Ping(); err != nil {
		t.Errorf("db.Ping() error = %v", err)
	}
}

func TestNewDatabase_InvalidDSN(t *testing.T) {
	cfg := config.DatabaseConfig{
		Type: "invalid-driver",
		DSN:  "/invalid/path/to/db.db",
	}

	_, err := NewDatabase(cfg)
	if err == nil {
		t.Error("NewDatabase() expected error for invalid driver, got nil")
	}
}

func TestNewDatabase_CreatesDirectory(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "db-test-*")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	// Use a subdirectory that doesn't exist yet
	dbPath := filepath.Join(tmpDir, "subdir", "test.db")

	cfg := config.DatabaseConfig{
		Type: "sqlite",
		DSN:  dbPath,
	}

	db, err := NewDatabase(cfg)
	if err != nil {
		t.Fatalf("NewDatabase() error = %v", err)
	}
	defer db.Close()

	// Verify directory was created
	dir := filepath.Dir(dbPath)
	if _, err := os.Stat(dir); os.IsNotExist(err) {
		t.Error("NewDatabase() did not create parent directory")
	}
}

func TestDatabase_Migrate(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "db-test-*")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	cfg := config.DatabaseConfig{
		Type: "sqlite",
		DSN:  filepath.Join(tmpDir, "test.db"),
	}

	db, err := NewDatabase(cfg)
	if err != nil {
		t.Fatalf("NewDatabase() error = %v", err)
	}
	defer db.Close()

	// Run migrations
	if err := db.Migrate(); err != nil {
		t.Fatalf("Migrate() error = %v", err)
	}

	// Verify schema_migrations table exists
	var count int
	err = db.db.QueryRow("SELECT COUNT(*) FROM schema_migrations").Scan(&count)
	if err != nil {
		t.Errorf("schema_migrations table not found: %v", err)
	}

	if count == 0 {
		t.Error("No migrations were recorded")
	}

	// Verify main tables exist
	tables := []string{"repositories", "migration_history", "migration_logs", "batches"}
	for _, table := range tables {
		var tableName string
		query := "SELECT name FROM sqlite_master WHERE type='table' AND name=?"
		err := db.db.QueryRow(query, table).Scan(&tableName)
		if err != nil {
			t.Errorf("Table %s does not exist: %v", table, err)
		}
		if tableName != table {
			t.Errorf("Expected table %s, got %s", table, tableName)
		}
	}
}

func TestDatabase_Migrate_Idempotent(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "db-test-*")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	cfg := config.DatabaseConfig{
		Type: "sqlite",
		DSN:  filepath.Join(tmpDir, "test.db"),
	}

	db, err := NewDatabase(cfg)
	if err != nil {
		t.Fatalf("NewDatabase() error = %v", err)
	}
	defer db.Close()

	// Run migrations first time
	if err := db.Migrate(); err != nil {
		t.Fatalf("First Migrate() error = %v", err)
	}

	// Get count of migrations
	var count1 int
	err = db.db.QueryRow("SELECT COUNT(*) FROM schema_migrations").Scan(&count1)
	if err != nil {
		t.Fatal(err)
	}

	// Run migrations second time (should be idempotent)
	if err := db.Migrate(); err != nil {
		t.Fatalf("Second Migrate() error = %v", err)
	}

	// Verify count hasn't changed
	var count2 int
	err = db.db.QueryRow("SELECT COUNT(*) FROM schema_migrations").Scan(&count2)
	if err != nil {
		t.Fatal(err)
	}

	if count1 != count2 {
		t.Errorf("Migration not idempotent: first run had %d migrations, second run has %d", count1, count2)
	}
}

func TestDatabase_DB(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "db-test-*")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	cfg := config.DatabaseConfig{
		Type: "sqlite",
		DSN:  filepath.Join(tmpDir, "test.db"),
	}

	db, err := NewDatabase(cfg)
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	sqlDB := db.DB()
	if sqlDB == nil {
		t.Error("DB() returned nil")
	}

	// Verify it's the same db
	if err := sqlDB.Ping(); err != nil {
		t.Errorf("DB().Ping() error = %v", err)
	}
}

func TestDatabase_Close(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "db-test-*")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	cfg := config.DatabaseConfig{
		Type: "sqlite",
		DSN:  filepath.Join(tmpDir, "test.db"),
	}

	db, err := NewDatabase(cfg)
	if err != nil {
		t.Fatal(err)
	}

	if err := db.Close(); err != nil {
		t.Errorf("Close() error = %v", err)
	}

	// Verify connection is closed
	if err := db.db.Ping(); err == nil {
		t.Error("Expected error after Close(), got nil")
	}
}
