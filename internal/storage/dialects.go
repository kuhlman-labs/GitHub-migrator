package storage

import (
	"fmt"
	"strings"
	"time"

	"github.com/kuhlman-labs/github-migrator/internal/config"
	"gorm.io/driver/postgres"
	"gorm.io/driver/sqlite"
	"gorm.io/driver/sqlserver"
	"gorm.io/gorm"
)

// DialectDialer creates a GORM dialector based on the database type
type DialectDialer interface {
	Dialect() gorm.Dialector
	ConfigureConnection(*gorm.DB) error
}

// NewDialectDialer creates a dialect dialer based on the database configuration
func NewDialectDialer(cfg config.DatabaseConfig) (DialectDialer, error) {
	switch cfg.Type {
	case "sqlite":
		return &SQLiteDialect{cfg: cfg}, nil
	case DBTypePostgres, DBTypePostgreSQL:
		return &PostgresDialect{cfg: cfg}, nil
	case "sqlserver", "mssql":
		return &SQLServerDialect{cfg: cfg}, nil
	default:
		return nil, fmt.Errorf("unsupported database type: %s", cfg.Type)
	}
}

// SQLiteDialect handles SQLite-specific configuration
type SQLiteDialect struct {
	cfg config.DatabaseConfig
}

func (d *SQLiteDialect) Dialect() gorm.Dialector {
	// Add _parseTime=true to DSN to parse DATETIME columns correctly
	dsn := d.cfg.DSN
	if !strings.Contains(dsn, "?") {
		dsn += "?_parseTime=true"
	} else if !strings.Contains(dsn, "_parseTime") {
		dsn += "&_parseTime=true"
	}
	return sqlite.Open(dsn)
}

func (d *SQLiteDialect) ConfigureConnection(db *gorm.DB) error {
	sqlDB, err := db.DB()
	if err != nil {
		return fmt.Errorf("failed to get underlying sql.DB: %w", err)
	}

	// SQLite-specific connection pool settings
	// SQLite doesn't benefit from many connections due to write serialization
	maxOpenConns := d.cfg.MaxOpenConns
	if maxOpenConns == 0 {
		maxOpenConns = 1 // SQLite works best with single writer
	}

	maxIdleConns := d.cfg.MaxIdleConns
	if maxIdleConns == 0 {
		maxIdleConns = 1
	}

	sqlDB.SetMaxOpenConns(maxOpenConns)
	sqlDB.SetMaxIdleConns(maxIdleConns)

	connMaxLifetime := time.Duration(d.cfg.ConnMaxLifetimeSeconds) * time.Second
	if connMaxLifetime == 0 {
		connMaxLifetime = 5 * time.Minute
	}
	sqlDB.SetConnMaxLifetime(connMaxLifetime)

	// Enable WAL mode for better concurrency
	if err := db.Exec("PRAGMA journal_mode=WAL").Error; err != nil {
		return fmt.Errorf("failed to enable WAL mode: %w", err)
	}

	// Enable foreign keys
	if err := db.Exec("PRAGMA foreign_keys=ON").Error; err != nil {
		return fmt.Errorf("failed to enable foreign keys: %w", err)
	}

	return nil
}

// PostgresDialect handles PostgreSQL-specific configuration
type PostgresDialect struct {
	cfg config.DatabaseConfig
}

func (d *PostgresDialect) Dialect() gorm.Dialector {
	return postgres.Open(d.cfg.DSN)
}

//nolint:dupl // Similar configuration code across dialects is expected
func (d *PostgresDialect) ConfigureConnection(db *gorm.DB) error {
	sqlDB, err := db.DB()
	if err != nil {
		return fmt.Errorf("failed to get underlying sql.DB: %w", err)
	}

	// PostgreSQL connection pool settings
	maxOpenConns := d.cfg.MaxOpenConns
	if maxOpenConns == 0 {
		maxOpenConns = 25 // Default for PostgreSQL
	}

	maxIdleConns := d.cfg.MaxIdleConns
	if maxIdleConns == 0 {
		maxIdleConns = 5
	}

	sqlDB.SetMaxOpenConns(maxOpenConns)
	sqlDB.SetMaxIdleConns(maxIdleConns)

	connMaxLifetime := time.Duration(d.cfg.ConnMaxLifetimeSeconds) * time.Second
	if connMaxLifetime == 0 {
		connMaxLifetime = 5 * time.Minute
	}
	sqlDB.SetConnMaxLifetime(connMaxLifetime)

	return nil
}

// SQLServerDialect handles SQL Server-specific configuration
type SQLServerDialect struct {
	cfg config.DatabaseConfig
}

func (d *SQLServerDialect) Dialect() gorm.Dialector {
	return sqlserver.Open(d.cfg.DSN)
}

//nolint:dupl // Similar configuration code across dialects is expected
func (d *SQLServerDialect) ConfigureConnection(db *gorm.DB) error {
	sqlDB, err := db.DB()
	if err != nil {
		return fmt.Errorf("failed to get underlying sql.DB: %w", err)
	}

	// SQL Server connection pool settings
	maxOpenConns := d.cfg.MaxOpenConns
	if maxOpenConns == 0 {
		maxOpenConns = 25 // Default for SQL Server
	}

	maxIdleConns := d.cfg.MaxIdleConns
	if maxIdleConns == 0 {
		maxIdleConns = 5
	}

	sqlDB.SetMaxOpenConns(maxOpenConns)
	sqlDB.SetMaxIdleConns(maxIdleConns)

	connMaxLifetime := time.Duration(d.cfg.ConnMaxLifetimeSeconds) * time.Second
	if connMaxLifetime == 0 {
		connMaxLifetime = 5 * time.Minute
	}
	sqlDB.SetConnMaxLifetime(connMaxLifetime)

	return nil
}
