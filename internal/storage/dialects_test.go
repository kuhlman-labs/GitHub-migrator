package storage

import (
	"testing"

	"github.com/kuhlman-labs/github-migrator/internal/config"
)

func TestNewDialectDialer(t *testing.T) {
	tests := []struct {
		name    string
		dbType  string
		wantErr bool
	}{
		{"sqlite", "sqlite", false},
		{"postgres", DBTypePostgres, false},
		{"postgresql", DBTypePostgreSQL, false},
		{"sqlserver", "sqlserver", false},
		{"mssql", "mssql", false},
		{"unknown", "mysql", true},
		{"empty", "", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := config.DatabaseConfig{
				Type: tt.dbType,
				DSN:  "test-dsn",
			}

			dialer, err := NewDialectDialer(cfg)

			if tt.wantErr {
				if err == nil {
					t.Error("NewDialectDialer() expected error, got nil")
				}
				return
			}

			if err != nil {
				t.Errorf("NewDialectDialer() unexpected error: %v", err)
				return
			}

			if dialer == nil {
				t.Error("NewDialectDialer() returned nil dialer")
			}
		})
	}
}

func TestSQLiteDialect_ExtractOrgFromFullName(t *testing.T) {
	d := &SQLiteDialect{cfg: config.DatabaseConfig{}}

	result := d.ExtractOrgFromFullName("full_name")

	expected := "SUBSTR(full_name, 1, INSTR(full_name, '/') - 1)"
	if result != expected {
		t.Errorf("ExtractOrgFromFullName() = %q, want %q", result, expected)
	}
}

func TestSQLiteDialect_FindCharPosition(t *testing.T) {
	d := &SQLiteDialect{}

	result := d.FindCharPosition("full_name", "/")

	expected := "INSTR(full_name, '/')"
	if result != expected {
		t.Errorf("FindCharPosition() = %q, want %q", result, expected)
	}
}

func TestSQLiteDialect_DateIntervalAgo(t *testing.T) {
	d := &SQLiteDialect{}

	tests := []struct {
		days int
		want string
	}{
		{7, "datetime('now', '-7 days')"},
		{30, "datetime('now', '-30 days')"},
		{0, "datetime('now', '-0 days')"},
	}

	for _, tt := range tests {
		t.Run("", func(t *testing.T) {
			result := d.DateIntervalAgo(tt.days)
			if result != tt.want {
				t.Errorf("DateIntervalAgo(%d) = %q, want %q", tt.days, result, tt.want)
			}
		})
	}
}

func TestSQLiteDialect_DateIntervalAgoParam(t *testing.T) {
	d := &SQLiteDialect{}

	sql, needsParam := d.DateIntervalAgoParam()

	if !needsParam {
		t.Error("DateIntervalAgoParam() needsParam should be true")
	}
	if sql != "datetime('now', '-' || ? || ' days')" {
		t.Errorf("DateIntervalAgoParam() sql = %q, unexpected", sql)
	}
}

func TestSQLiteDialect_BooleanLiterals(t *testing.T) {
	d := &SQLiteDialect{}

	if d.BooleanTrue() != "1" {
		t.Errorf("BooleanTrue() = %q, want %q", d.BooleanTrue(), "1")
	}
	if d.BooleanFalse() != "0" {
		t.Errorf("BooleanFalse() = %q, want %q", d.BooleanFalse(), "0")
	}
}

func TestSQLiteDialect_PercentileCont(t *testing.T) {
	d := &SQLiteDialect{}

	if d.SupportsPercentileCont() {
		t.Error("SQLite should not support PERCENTILE_CONT")
	}
	if d.PercentileMedian("column") != "" {
		t.Error("PercentileMedian() should return empty string for SQLite")
	}
}

func TestPostgresDialect_ExtractOrgFromFullName(t *testing.T) {
	d := &PostgresDialect{}

	result := d.ExtractOrgFromFullName("full_name")

	expected := "SUBSTRING(full_name, 1, POSITION('/' IN full_name) - 1)"
	if result != expected {
		t.Errorf("ExtractOrgFromFullName() = %q, want %q", result, expected)
	}
}

func TestPostgresDialect_FindCharPosition(t *testing.T) {
	d := &PostgresDialect{}

	result := d.FindCharPosition("full_name", "/")

	expected := "POSITION('/' IN full_name)"
	if result != expected {
		t.Errorf("FindCharPosition() = %q, want %q", result, expected)
	}
}

func TestPostgresDialect_DateIntervalAgo(t *testing.T) {
	d := &PostgresDialect{}

	result := d.DateIntervalAgo(30)

	expected := "NOW() - INTERVAL '30 days'"
	if result != expected {
		t.Errorf("DateIntervalAgo(30) = %q, want %q", result, expected)
	}
}

func TestPostgresDialect_DateIntervalAgoParam(t *testing.T) {
	d := &PostgresDialect{}

	sql, needsParam := d.DateIntervalAgoParam()

	if !needsParam {
		t.Error("DateIntervalAgoParam() needsParam should be true")
	}
	if sql != "NOW() - INTERVAL '1 day' * ?" {
		t.Errorf("DateIntervalAgoParam() sql = %q, unexpected", sql)
	}
}

func TestPostgresDialect_BooleanLiterals(t *testing.T) {
	d := &PostgresDialect{}

	if d.BooleanTrue() != "TRUE" {
		t.Errorf("BooleanTrue() = %q, want %q", d.BooleanTrue(), "TRUE")
	}
	if d.BooleanFalse() != "FALSE" {
		t.Errorf("BooleanFalse() = %q, want %q", d.BooleanFalse(), "FALSE")
	}
}

func TestPostgresDialect_PercentileCont(t *testing.T) {
	d := &PostgresDialect{}

	if !d.SupportsPercentileCont() {
		t.Error("PostgreSQL should support PERCENTILE_CONT")
	}

	result := d.PercentileMedian("duration")
	expected := "PERCENTILE_CONT(0.5) WITHIN GROUP (ORDER BY duration)"
	if result != expected {
		t.Errorf("PercentileMedian() = %q, want %q", result, expected)
	}
}

func TestSQLServerDialect_ExtractOrgFromFullName(t *testing.T) {
	d := &SQLServerDialect{}

	result := d.ExtractOrgFromFullName("full_name")

	expected := "SUBSTRING(full_name, 1, CHARINDEX('/', full_name) - 1)"
	if result != expected {
		t.Errorf("ExtractOrgFromFullName() = %q, want %q", result, expected)
	}
}

func TestSQLServerDialect_FindCharPosition(t *testing.T) {
	d := &SQLServerDialect{}

	result := d.FindCharPosition("full_name", "/")

	expected := "CHARINDEX('/', full_name)"
	if result != expected {
		t.Errorf("FindCharPosition() = %q, want %q", result, expected)
	}
}

func TestSQLServerDialect_DateIntervalAgo(t *testing.T) {
	d := &SQLServerDialect{}

	result := d.DateIntervalAgo(30)

	expected := "DATEADD(day, -30, GETUTCDATE())"
	if result != expected {
		t.Errorf("DateIntervalAgo(30) = %q, want %q", result, expected)
	}
}

func TestSQLServerDialect_DateIntervalAgoParam(t *testing.T) {
	d := &SQLServerDialect{}

	sql, needsParam := d.DateIntervalAgoParam()

	if !needsParam {
		t.Error("DateIntervalAgoParam() needsParam should be true")
	}
	if sql != "DATEADD(day, -?, GETUTCDATE())" {
		t.Errorf("DateIntervalAgoParam() sql = %q, unexpected", sql)
	}
}

func TestSQLServerDialect_BooleanLiterals(t *testing.T) {
	d := &SQLServerDialect{}

	if d.BooleanTrue() != "1" {
		t.Errorf("BooleanTrue() = %q, want %q", d.BooleanTrue(), "1")
	}
	if d.BooleanFalse() != "0" {
		t.Errorf("BooleanFalse() = %q, want %q", d.BooleanFalse(), "0")
	}
}

func TestSQLServerDialect_PercentileCont(t *testing.T) {
	d := &SQLServerDialect{}

	if !d.SupportsPercentileCont() {
		t.Error("SQL Server should support PERCENTILE_CONT")
	}

	result := d.PercentileMedian("duration")
	expected := "PERCENTILE_CONT(0.5) WITHIN GROUP (ORDER BY duration) OVER ()"
	if result != expected {
		t.Errorf("PercentileMedian() = %q, want %q", result, expected)
	}
}

func TestSQLiteDialect_Dialect(t *testing.T) {
	d := &SQLiteDialect{cfg: config.DatabaseConfig{DSN: "test.db"}}

	dialector := d.Dialect()

	if dialector == nil {
		t.Error("Dialect() returned nil")
	}
}

func TestSQLiteDialect_Dialect_WithExistingParams(t *testing.T) {
	d := &SQLiteDialect{cfg: config.DatabaseConfig{DSN: "test.db?mode=memory"}}

	dialector := d.Dialect()

	if dialector == nil {
		t.Error("Dialect() returned nil")
	}
}

func TestSQLiteDialect_Dialect_WithExistingParseTime(t *testing.T) {
	d := &SQLiteDialect{cfg: config.DatabaseConfig{DSN: "test.db?_parseTime=true"}}

	dialector := d.Dialect()

	if dialector == nil {
		t.Error("Dialect() returned nil")
	}
}

func TestPostgresDialect_Dialect(t *testing.T) {
	d := &PostgresDialect{cfg: config.DatabaseConfig{DSN: "postgresql://test"}}

	dialector := d.Dialect()

	if dialector == nil {
		t.Error("Dialect() returned nil")
	}
}

func TestSQLServerDialect_Dialect(t *testing.T) {
	d := &SQLServerDialect{cfg: config.DatabaseConfig{DSN: "sqlserver://test"}}

	dialector := d.Dialect()

	if dialector == nil {
		t.Error("Dialect() returned nil")
	}
}
