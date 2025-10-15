package storage

import (
	"context"
	"database/sql"
	"embed"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/brettkuhlman/github-migrator/internal/config"
	_ "github.com/mattn/go-sqlite3"
)

//go:embed migrations/*.sql
var migrationsFS embed.FS

type Database struct {
	db  *sql.DB
	cfg config.DatabaseConfig
}

func NewDatabase(cfg config.DatabaseConfig) (*Database, error) {
	// Ensure data directory exists for SQLite
	if cfg.Type == "sqlite" {
		dir := filepath.Dir(cfg.DSN)
		if err := os.MkdirAll(dir, 0750); err != nil {
			return nil, fmt.Errorf("failed to create data directory: %w", err)
		}
	}

	// Map config type to driver name
	driverName := cfg.Type
	if cfg.Type == "sqlite" {
		driverName = "sqlite3"
	}

	db, err := sql.Open(driverName, cfg.DSN)
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}

	if err := db.Ping(); err != nil {
		return nil, fmt.Errorf("failed to ping database: %w", err)
	}

	return &Database{
		db:  db,
		cfg: cfg,
	}, nil
}

func (d *Database) Close() error {
	return d.db.Close()
}

func (d *Database) DB() *sql.DB {
	return d.db
}

// Migrate runs all pending database migrations
func (d *Database) Migrate() error {
	slog.Info("Running database migrations...")

	// Create migrations tracking table
	if err := d.createMigrationsTable(); err != nil {
		return fmt.Errorf("failed to create migrations table: %w", err)
	}

	// Get list of applied migrations
	applied, err := d.getAppliedMigrations()
	if err != nil {
		return fmt.Errorf("failed to get applied migrations: %w", err)
	}

	// Read migration files from embedded filesystem
	entries, err := migrationsFS.ReadDir("migrations")
	if err != nil {
		return fmt.Errorf("failed to read migrations directory: %w", err)
	}

	// Sort migration files
	var migrationFiles []string
	for _, entry := range entries {
		if !entry.IsDir() && strings.HasSuffix(entry.Name(), ".sql") {
			migrationFiles = append(migrationFiles, entry.Name())
		}
	}
	sort.Strings(migrationFiles)

	// Apply pending migrations
	for _, filename := range migrationFiles {
		if applied[filename] {
			slog.Debug("Skipping already applied migration", "file", filename)
			continue
		}

		slog.Info("Applying migration", "file", filename)
		content, err := migrationsFS.ReadFile(filepath.Join("migrations", filename))
		if err != nil {
			return fmt.Errorf("failed to read migration file %s: %w", filename, err)
		}

		if err := d.applyMigration(filename, string(content)); err != nil {
			return fmt.Errorf("failed to apply migration %s: %w", filename, err)
		}

		slog.Info("Successfully applied migration", "file", filename)
	}

	slog.Info("Database migrations completed successfully")
	return nil
}

func (d *Database) createMigrationsTable() error {
	query := `
		CREATE TABLE IF NOT EXISTS schema_migrations (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			filename TEXT NOT NULL UNIQUE,
			applied_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
		)
	`
	_, err := d.db.Exec(query)
	return err
}

func (d *Database) getAppliedMigrations() (map[string]bool, error) {
	rows, err := d.db.Query("SELECT filename FROM schema_migrations")
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	applied := make(map[string]bool)
	for rows.Next() {
		var filename string
		if err := rows.Scan(&filename); err != nil {
			return nil, err
		}
		applied[filename] = true
	}

	return applied, rows.Err()
}

func (d *Database) applyMigration(filename, content string) error {
	tx, err := d.db.Begin()
	if err != nil {
		return err
	}
	defer func() {
		_ = tx.Rollback() // rollback is safe to call even after commit
	}()

	// Execute migration
	if _, execErr := tx.Exec(content); execErr != nil {
		return execErr
	}

	// Record migration
	_, err = tx.Exec("INSERT INTO schema_migrations (filename) VALUES (?)", filename)
	if err != nil {
		return err
	}

	return tx.Commit()
}

// GetDistinctOrganizations retrieves a list of unique organizations from repositories
func (d *Database) GetDistinctOrganizations(ctx context.Context) ([]string, error) {
	query := `
		SELECT DISTINCT 
			CASE 
				WHEN instr(full_name, '/') > 0 
				THEN substr(full_name, 1, instr(full_name, '/') - 1)
				ELSE full_name
			END as organization
		FROM repositories
		WHERE full_name LIKE '%/%'
		ORDER BY organization ASC
	`

	rows, err := d.db.QueryContext(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("failed to get distinct organizations: %w", err)
	}
	defer rows.Close()

	var orgs []string
	for rows.Next() {
		var org string
		if err := rows.Scan(&org); err != nil {
			return nil, fmt.Errorf("failed to scan organization: %w", err)
		}
		orgs = append(orgs, org)
	}

	return orgs, rows.Err()
}

// CountRepositoriesWithFilters counts repositories matching the given filters
func (d *Database) CountRepositoriesWithFilters(ctx context.Context, filters map[string]interface{}) (int, error) {
	query := "SELECT COUNT(*) FROM repositories WHERE 1=1"
	args := []interface{}{}

	// Apply the same filters as ListRepositories
	query, args = applyRepositoryFilters(query, args, filters)

	var count int
	err := d.db.QueryRowContext(ctx, query, args...).Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("failed to count repositories: %w", err)
	}

	return count, nil
}
