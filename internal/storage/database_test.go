package storage

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/kuhlman-labs/github-migrator/internal/config"
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
	sqlDB, err := db.db.DB()
	if err != nil {
		t.Fatalf("db.DB() error = %v", err)
	}
	if err := sqlDB.Ping(); err != nil {
		t.Errorf("sqlDB.Ping() error = %v", err)
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
	var count int64
	err = db.db.Model(&SchemaMigration{}).Count(&count).Error
	if err != nil {
		t.Errorf("schema_migrations table not found: %v", err)
	}

	if count == 0 {
		t.Error("No migrations were recorded")
	}

	// Verify main tables exist using GORM's Migrator
	tables := []string{"repositories", "migration_history", "migration_logs", "batches"}
	for _, table := range tables {
		if !db.db.Migrator().HasTable(table) {
			t.Errorf("Table %s does not exist", table)
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
	var count1 int64
	err = db.db.Model(&SchemaMigration{}).Count(&count1).Error
	if err != nil {
		t.Fatal(err)
	}

	// Run migrations second time (should be idempotent)
	if err := db.Migrate(); err != nil {
		t.Fatalf("Second Migrate() error = %v", err)
	}

	// Verify count hasn't changed
	var count2 int64
	err = db.db.Model(&SchemaMigration{}).Count(&count2).Error
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

	gormDB := db.DB()
	if gormDB == nil {
		t.Error("DB() returned nil")
	}

	// Verify it's the same db
	sqlDB, err := gormDB.DB()
	if err != nil {
		t.Fatalf("gormDB.DB() error = %v", err)
	}
	if err := sqlDB.Ping(); err != nil {
		t.Errorf("sqlDB.Ping() error = %v", err)
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
	sqlDB, err := db.db.DB()
	if err != nil {
		t.Fatalf("db.db.DB() error = %v", err)
	}
	if err := sqlDB.Ping(); err == nil {
		t.Error("Expected error after Close(), got nil")
	}
}
