package storage

import (
	"context"
	"database/sql"
	"embed"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"

	"github.com/brettkuhlman/github-migrator/internal/config"
	_ "github.com/lib/pq"
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

// rebindQuery converts SQLite-style ? placeholders to the appropriate syntax for the database type
// and transforms SQLite-specific functions to PostgreSQL equivalents
func (d *Database) rebindQuery(query string) string {
	if d.cfg.Type == "postgres" {
		// Transform SQLite functions to PostgreSQL
		query = d.transformSQLiteFunctionsToPostgres(query)

		// Convert ? placeholders to $1, $2, etc. for Postgres
		var result strings.Builder
		paramNum := 1
		for i := 0; i < len(query); i++ {
			if query[i] == '?' {
				result.WriteString(fmt.Sprintf("$%d", paramNum))
				paramNum++
			} else {
				result.WriteByte(query[i])
			}
		}
		return result.String()
	}

	return query
}

// transformSQLiteFunctionsToPostgres converts SQLite-specific functions to PostgreSQL equivalents
func (d *Database) transformSQLiteFunctionsToPostgres(query string) string {
	// INSTR(haystack, needle) -> POSITION(needle IN haystack)
	// Match INSTR with its two arguments and swap them
	instrRegex := regexp.MustCompile(`(?i)INSTR\(([^,]+),\s*('[^']+')\)`)
	query = instrRegex.ReplaceAllString(query, "POSITION($2 IN $1)")

	// SUBSTR(str, start, length) -> SUBSTRING(str, start, length)
	// PostgreSQL supports both SUBSTRING syntaxes, so simple replacement works
	query = strings.ReplaceAll(query, "SUBSTR(", "SUBSTRING(")
	query = strings.ReplaceAll(query, "substr(", "SUBSTRING(")

	return query
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
	var query string
	if d.cfg.Type == "postgres" {
		query = `
			CREATE TABLE IF NOT EXISTS schema_migrations (
				id SERIAL PRIMARY KEY,
				filename TEXT NOT NULL UNIQUE,
				applied_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
			)
		`
	} else {
		query = `
			CREATE TABLE IF NOT EXISTS schema_migrations (
				id INTEGER PRIMARY KEY AUTOINCREMENT,
				filename TEXT NOT NULL UNIQUE,
				applied_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
			)
		`
	}
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

	// Split migration content into individual statements
	// SQLite's Exec() only handles one statement at a time
	statements := splitSQLStatements(content)

	// Execute each statement
	for i, stmt := range statements {
		stmt = strings.TrimSpace(stmt)
		if stmt == "" {
			continue
		}

		// Transform SQLite syntax to PostgreSQL if needed
		if d.cfg.Type == "postgres" {
			stmt = d.transformSQLiteToPostgres(stmt)
		}

		if _, execErr := tx.Exec(stmt); execErr != nil {
			return fmt.Errorf("statement %d failed: %w\nStatement: %s", i+1, execErr, stmt)
		}
	}

	// Record migration
	insertQuery := d.rebindQuery("INSERT INTO schema_migrations (filename) VALUES (?)")
	_, err = tx.Exec(insertQuery, filename)
	if err != nil {
		return err
	}

	return tx.Commit()
}

// splitSQLStatements splits SQL content into individual statements
// It handles semicolon-separated statements and filters out comments
// Only processes the "Up" migration, stops at "Down" section
func splitSQLStatements(content string) []string {
	var statements []string
	var currentStmt strings.Builder

	lines := strings.Split(content, "\n")
	for _, line := range lines {
		trimmed := strings.TrimSpace(line)

		// Check for goose directives
		if strings.HasPrefix(trimmed, "-- +goose Up") {
			continue
		}
		if strings.HasPrefix(trimmed, "-- +goose Down") {
			// Stop processing - we only want the "Up" migration
			break
		}
		if strings.HasPrefix(trimmed, "-- +goose") {
			continue
		}

		// Skip standalone comment lines (but keep comments within statements)
		if strings.HasPrefix(trimmed, "--") && !strings.Contains(line, "CREATE") && !strings.Contains(line, "DROP") {
			// If we're building a statement, keep comment for context
			if currentStmt.Len() > 0 {
				currentStmt.WriteString(line)
				currentStmt.WriteString("\n")
			}
			continue
		}

		currentStmt.WriteString(line)
		currentStmt.WriteString("\n")

		// Check if line ends with semicolon (end of statement)
		if strings.HasSuffix(trimmed, ";") {
			stmt := currentStmt.String()
			if strings.TrimSpace(stmt) != "" {
				statements = append(statements, stmt)
			}
			currentStmt.Reset()
		}
	}

	// Add any remaining statement
	if currentStmt.Len() > 0 {
		stmt := currentStmt.String()
		if strings.TrimSpace(stmt) != "" {
			statements = append(statements, stmt)
		}
	}

	return statements
}

// transformSQLiteToPostgres converts SQLite-specific syntax to PostgreSQL syntax
func (d *Database) transformSQLiteToPostgres(stmt string) string {
	// Replace AUTOINCREMENT with SERIAL for simple cases
	// Handle both "INTEGER PRIMARY KEY AUTOINCREMENT" and standalone AUTOINCREMENT
	stmt = strings.ReplaceAll(stmt, "INTEGER PRIMARY KEY AUTOINCREMENT", "SERIAL PRIMARY KEY")
	stmt = strings.ReplaceAll(stmt, "AUTOINCREMENT", "")
	
	// Replace DATETIME with TIMESTAMP
	stmt = strings.ReplaceAll(stmt, "DATETIME", "TIMESTAMP")
	
	// Replace SQLite boolean defaults (0/1) with PostgreSQL boolean literals (FALSE/TRUE)
	// This needs to be done carefully to only replace in DEFAULT clauses
	stmt = strings.ReplaceAll(stmt, "DEFAULT 0", "DEFAULT FALSE")
	stmt = strings.ReplaceAll(stmt, "DEFAULT 1", "DEFAULT TRUE")
	
	return stmt
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

	rows, err := d.db.QueryContext(ctx, d.rebindQuery(query))
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
	err := d.db.QueryRowContext(ctx, d.rebindQuery(query), args...).Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("failed to count repositories: %w", err)
	}

	return count, nil
}
