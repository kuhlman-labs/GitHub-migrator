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

// DialectDialer creates a GORM dialector and provides dialect-specific SQL expressions.
// This interface abstracts database-specific syntax differences for portable queries.
type DialectDialer interface {
	// Dialect returns the GORM dialector for this database type.
	Dialect() gorm.Dialector

	// ConfigureConnection sets up connection pooling and database-specific settings.
	ConfigureConnection(*gorm.DB) error

	// SQL Expression Helpers

	// ExtractOrgFromFullName returns SQL to extract org from "org/repo" format.
	// Example output: "SUBSTRING(full_name, 1, POSITION('/' IN full_name) - 1)" (Postgres)
	ExtractOrgFromFullName(column string) string

	// FindCharPosition returns SQL to find position of char in a column.
	// Example: FindCharPosition("full_name", "/") -> "POSITION('/' IN full_name)" (Postgres)
	FindCharPosition(column, char string) string

	// DateIntervalAgo returns SQL for a date N days ago.
	// Example: DateIntervalAgo(30) -> "NOW() - INTERVAL '30 days'" (Postgres)
	DateIntervalAgo(days int) string

	// DateIntervalAgoParam returns SQL with placeholder for parameterized days.
	// Returns (sql, needsParam) - if needsParam is true, caller must add days to args.
	DateIntervalAgoParam() (sql string, needsParam bool)

	// BooleanTrue returns the SQL literal for true.
	BooleanTrue() string

	// BooleanFalse returns the SQL literal for false.
	BooleanFalse() string

	// SupportsPercentileCont returns true if the database supports PERCENTILE_CONT.
	SupportsPercentileCont() bool

	// PercentileMedian returns SQL for calculating median, or empty if not supported.
	// The caller should fall back to ordering/limiting for databases without support.
	PercentileMedian(column string) string
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

// ExtractOrgFromFullName returns SQLite SQL to extract org from "org/repo" format.
func (d *SQLiteDialect) ExtractOrgFromFullName(column string) string {
	return fmt.Sprintf("SUBSTR(%s, 1, INSTR(%s, '/') - 1)", column, column)
}

// FindCharPosition returns SQLite SQL to find position of char in a column.
func (d *SQLiteDialect) FindCharPosition(column, char string) string {
	return fmt.Sprintf("INSTR(%s, '%s')", column, char)
}

// DateIntervalAgo returns SQLite SQL for a date N days ago.
func (d *SQLiteDialect) DateIntervalAgo(days int) string {
	return fmt.Sprintf("datetime('now', '-%d days')", days)
}

// DateIntervalAgoParam returns SQLite SQL with placeholder for parameterized days.
func (d *SQLiteDialect) DateIntervalAgoParam() (string, bool) {
	return "datetime('now', '-' || ? || ' days')", true
}

// BooleanTrue returns the SQLite literal for true.
func (d *SQLiteDialect) BooleanTrue() string {
	return "1"
}

// BooleanFalse returns the SQLite literal for false.
func (d *SQLiteDialect) BooleanFalse() string {
	return "0"
}

// SupportsPercentileCont returns false - SQLite doesn't support PERCENTILE_CONT.
func (d *SQLiteDialect) SupportsPercentileCont() bool {
	return false
}

// PercentileMedian returns empty string - SQLite doesn't support PERCENTILE_CONT.
func (d *SQLiteDialect) PercentileMedian(column string) string {
	return ""
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

// ExtractOrgFromFullName returns PostgreSQL SQL to extract org from "org/repo" format.
func (d *PostgresDialect) ExtractOrgFromFullName(column string) string {
	return fmt.Sprintf("SUBSTRING(%s, 1, POSITION('/' IN %s) - 1)", column, column)
}

// FindCharPosition returns PostgreSQL SQL to find position of char in a column.
func (d *PostgresDialect) FindCharPosition(column, char string) string {
	return fmt.Sprintf("POSITION('%s' IN %s)", char, column)
}

// DateIntervalAgo returns PostgreSQL SQL for a date N days ago.
func (d *PostgresDialect) DateIntervalAgo(days int) string {
	return fmt.Sprintf("NOW() - INTERVAL '%d days'", days)
}

// DateIntervalAgoParam returns PostgreSQL SQL with placeholder for parameterized days.
func (d *PostgresDialect) DateIntervalAgoParam() (string, bool) {
	// PostgreSQL can use interval arithmetic with concatenation
	return "NOW() - INTERVAL '1 day' * ?", true
}

// BooleanTrue returns the PostgreSQL literal for true.
func (d *PostgresDialect) BooleanTrue() string {
	return "TRUE"
}

// BooleanFalse returns the PostgreSQL literal for false.
func (d *PostgresDialect) BooleanFalse() string {
	return "FALSE"
}

// SupportsPercentileCont returns true - PostgreSQL supports PERCENTILE_CONT.
func (d *PostgresDialect) SupportsPercentileCont() bool {
	return true
}

// PercentileMedian returns PostgreSQL SQL for calculating median.
func (d *PostgresDialect) PercentileMedian(column string) string {
	return fmt.Sprintf("PERCENTILE_CONT(0.5) WITHIN GROUP (ORDER BY %s)", column)
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

// ExtractOrgFromFullName returns SQL Server SQL to extract org from "org/repo" format.
func (d *SQLServerDialect) ExtractOrgFromFullName(column string) string {
	return fmt.Sprintf("SUBSTRING(%s, 1, CHARINDEX('/', %s) - 1)", column, column)
}

// FindCharPosition returns SQL Server SQL to find position of char in a column.
func (d *SQLServerDialect) FindCharPosition(column, char string) string {
	return fmt.Sprintf("CHARINDEX('%s', %s)", char, column)
}

// DateIntervalAgo returns SQL Server SQL for a date N days ago.
func (d *SQLServerDialect) DateIntervalAgo(days int) string {
	return fmt.Sprintf("DATEADD(day, -%d, GETUTCDATE())", days)
}

// DateIntervalAgoParam returns SQL Server SQL with placeholder for parameterized days.
func (d *SQLServerDialect) DateIntervalAgoParam() (string, bool) {
	return "DATEADD(day, -?, GETUTCDATE())", true
}

// BooleanTrue returns the SQL Server literal for true.
func (d *SQLServerDialect) BooleanTrue() string {
	return "1" // SQL Server uses 1/0 for BIT type
}

// BooleanFalse returns the SQL Server literal for false.
func (d *SQLServerDialect) BooleanFalse() string {
	return "0"
}

// SupportsPercentileCont returns true - SQL Server 2012+ supports PERCENTILE_CONT.
func (d *SQLServerDialect) SupportsPercentileCont() bool {
	return true
}

// PercentileMedian returns SQL Server SQL for calculating median.
func (d *SQLServerDialect) PercentileMedian(column string) string {
	return fmt.Sprintf("PERCENTILE_CONT(0.5) WITHIN GROUP (ORDER BY %s) OVER ()", column)
}
