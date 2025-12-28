package storage

import (
	"context"
	"embed"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/kuhlman-labs/github-migrator/internal/config"
	"github.com/kuhlman-labs/github-migrator/internal/models"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

//go:embed migrations/*/*.sql
var migrationsFS embed.FS

// Database type constants
const (
	DBTypeSQLite     = "sqlite"
	DBTypePostgres   = "postgres"
	DBTypePostgreSQL = "postgresql"
	DBTypeSQLServer  = "sqlserver"
	DBTypeMSSQL      = "mssql"
)

type Database struct {
	db      *gorm.DB
	cfg     config.DatabaseConfig
	dialect DialectDialer
}

func NewDatabase(cfg config.DatabaseConfig) (*Database, error) {
	// Ensure data directory exists for SQLite
	if cfg.Type == DBTypeSQLite {
		dir := filepath.Dir(cfg.DSN)
		if err := os.MkdirAll(dir, 0750); err != nil {
			return nil, fmt.Errorf("failed to create data directory: %w", err)
		}
	}

	// Create dialect dialer
	dialect, err := NewDialectDialer(cfg)
	if err != nil {
		return nil, fmt.Errorf("failed to create dialect: %w", err)
	}

	// Configure GORM logger to use slog
	gormLogger := logger.New(
		&slogWriter{},
		logger.Config{
			SlowThreshold:             200 * time.Millisecond, // Log slow queries (>200ms)
			LogLevel:                  logger.Warn,
			IgnoreRecordNotFoundError: true,
			Colorful:                  false,
		},
	)

	// Open database with GORM
	db, err := gorm.Open(dialect.Dialect(), &gorm.Config{
		Logger: gormLogger,
		NowFunc: func() time.Time {
			return time.Now().UTC()
		},
	})
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}

	// Configure connection pooling
	if err := dialect.ConfigureConnection(db); err != nil {
		return nil, fmt.Errorf("failed to configure connection: %w", err)
	}

	// Test connection
	sqlDB, err := db.DB()
	if err != nil {
		return nil, fmt.Errorf("failed to get underlying sql.DB: %w", err)
	}
	if err := sqlDB.Ping(); err != nil {
		return nil, fmt.Errorf("failed to ping database: %w", err)
	}

	return &Database{
		db:      db,
		cfg:     cfg,
		dialect: dialect,
	}, nil
}

func (d *Database) Close() error {
	sqlDB, err := d.db.DB()
	if err != nil {
		return err
	}
	return sqlDB.Close()
}

func (d *Database) DB() *gorm.DB {
	return d.db
}

// slogWriter adapts slog for GORM's logger interface
type slogWriter struct{}

func (w *slogWriter) Printf(format string, args ...any) {
	slog.Info(fmt.Sprintf(format, args...))
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

	// Determine which dialect folder to use
	dialectFolder := d.getDialectFolder()
	migrationPath := filepath.Join("migrations", dialectFolder)

	slog.Info("Loading migrations from dialect folder", "path", migrationPath, "database_type", d.cfg.Type)

	// Read migration files from embedded filesystem for the specific dialect
	entries, err := migrationsFS.ReadDir(migrationPath)
	if err != nil {
		return fmt.Errorf("failed to read migrations directory %s: %w", migrationPath, err)
	}

	// Sort migration files
	var migrationFiles []string
	for _, entry := range entries {
		if !entry.IsDir() && strings.HasSuffix(entry.Name(), ".sql") {
			migrationFiles = append(migrationFiles, entry.Name())
		}
	}
	sort.Strings(migrationFiles)

	slog.Info("Found migrations", "count", len(migrationFiles), "folder", dialectFolder)

	// Apply pending migrations
	for _, filename := range migrationFiles {
		if applied[filename] {
			slog.Debug("Skipping already applied migration", "file", filename)
			continue
		}

		slog.Info("Applying migration", "file", filename)
		content, err := migrationsFS.ReadFile(filepath.Join(migrationPath, filename))
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

// getDialectFolder returns the migration folder name based on the database type
func (d *Database) getDialectFolder() string {
	switch d.cfg.Type {
	case DBTypePostgres, DBTypePostgreSQL:
		return DBTypePostgres
	case DBTypeSQLServer, DBTypeMSSQL:
		return "sqlserver"
	case DBTypeSQLite, "sqlite3":
		return DBTypeSQLite
	default:
		// Default to sqlite for backward compatibility
		return DBTypeSQLite
	}
}

// SchemaMigration tracks applied migrations
type SchemaMigration struct {
	ID        int64     `gorm:"primaryKey;autoIncrement"`
	Filename  string    `gorm:"uniqueIndex;not null"`
	AppliedAt time.Time `gorm:"not null;autoCreateTime"`
}

// TableName specifies the table name for SchemaMigration
func (SchemaMigration) TableName() string {
	return "schema_migrations"
}

func (d *Database) createMigrationsTable() error {
	// Use GORM AutoMigrate for schema_migrations table
	err := d.db.AutoMigrate(&SchemaMigration{})
	return err
}

func (d *Database) getAppliedMigrations() (map[string]bool, error) {
	var migrations []SchemaMigration
	if err := d.db.Find(&migrations).Error; err != nil {
		return nil, err
	}

	applied := make(map[string]bool)
	for _, m := range migrations {
		applied[m.Filename] = true
	}

	return applied, nil
}

func (d *Database) applyMigration(filename, content string) error {
	// Start transaction using GORM
	return d.db.Transaction(func(tx *gorm.DB) error {
		// Split migration content into individual statements
		statements := splitSQLStatements(content)

		// Execute each statement using GORM Exec
		for i, stmt := range statements {
			stmt = strings.TrimSpace(stmt)
			if stmt == "" {
				continue
			}

			// NOTE: We no longer apply SQL transformations here because we now use
			// dialect-specific migration folders (migrations/sqlite, migrations/postgres, migrations/sqlserver).
			// Each migration is already written in the correct syntax for its target database.

			// Skip ALTER COLUMN statements for SQLite (not supported)
			if d.cfg.Type == DBTypeSQLite && strings.Contains(strings.ToUpper(stmt), "ALTER COLUMN") {
				slog.Debug("Skipping ALTER COLUMN statement for SQLite", "statement", stmt)
				continue
			}

			if err := tx.Exec(stmt).Error; err != nil {
				return fmt.Errorf("statement %d failed: %w\nStatement: %s", i+1, err, stmt)
			}
		}

		// Record migration using GORM
		migration := SchemaMigration{
			Filename: filename,
		}
		if err := tx.Create(&migration).Error; err != nil {
			return err
		}

		return nil
	})
}

// splitSQLStatements splits SQL content into individual statements
// It handles semicolon-separated statements and filters out comments
// Only processes the "Up" migration, stops at "Down" section
func splitSQLStatements(content string) []string {
	var statements []string
	var currentStmt strings.Builder

	lines := strings.SplitSeq(content, "\n")
	for line := range lines {
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

// GetDistinctOrganizations retrieves a list of unique organizations from repositories using GORM
func (d *Database) GetDistinctOrganizations(ctx context.Context) ([]string, error) {
	// Use dialect-specific string functions via the DialectDialer interface
	var orgs []string
	extractOrg := d.dialect.ExtractOrgFromFullName("full_name")

	query := fmt.Sprintf(`
		SELECT DISTINCT 
			CASE 
				WHEN full_name LIKE '%%/%%'
				THEN %s
				ELSE full_name
			END as organization
		FROM repositories
		WHERE full_name LIKE '%%/%%'
		ORDER BY organization ASC
	`, extractOrg)

	err := d.db.WithContext(ctx).Raw(query).Scan(&orgs).Error
	if err != nil {
		return nil, fmt.Errorf("failed to get distinct organizations: %w", err)
	}

	return orgs, nil
}

// CountRepositoriesWithFilters counts repositories matching the given filters using GORM
func (d *Database) CountRepositoriesWithFilters(ctx context.Context, filters map[string]any) (int, error) {
	var count int64

	// Start with base query
	query := d.db.WithContext(ctx).Model(&models.Repository{})

	// Apply the same scopes as ListRepositories
	query = d.applyListScopes(query, filters)

	// Execute count
	err := query.Count(&count).Error
	if err != nil {
		return 0, fmt.Errorf("failed to count repositories: %w", err)
	}

	return int(count), nil
}
