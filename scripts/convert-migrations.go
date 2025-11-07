package main

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

// MigrationConverter converts SQLite migrations to other database dialects
type MigrationConverter struct {
	sourcePath string
	targetPath string
}

// Convert converts a SQLite migration to the specified dialect
func (mc *MigrationConverter) Convert(content, dialect string) string {
	switch dialect {
	case "postgres":
		return mc.convertToPostgreSQL(content)
	case "sqlserver":
		return mc.convertToSQLServer(content)
	default:
		return content
	}
}

// convertToPostgreSQL converts SQLite syntax to PostgreSQL
func (mc *MigrationConverter) convertToPostgreSQL(content string) string {
	// Add goose directives if missing
	if !strings.Contains(content, "-- +goose Up") {
		parts := strings.SplitN(content, "\n", 1)
		if len(parts) > 0 {
			content = "-- +goose Up\n" + content + "\n\n-- +goose Down\n-- Add rollback logic here\n"
		}
	}

	// Replace SQLite types with PostgreSQL types
	replacements := map[string]string{
		// Primary key
		`INTEGER PRIMARY KEY AUTOINCREMENT`: `SERIAL PRIMARY KEY`,
		`BIGINT PRIMARY KEY AUTOINCREMENT`:  `BIGSERIAL PRIMARY KEY`,

		// Timestamps
		`DATETIME`:          `TIMESTAMP`,
		`CURRENT_TIMESTAMP`: `CURRENT_TIMESTAMP`,

		// Boolean (SQLite stores as INTEGER)
		`BOOLEAN DEFAULT FALSE`: `BOOLEAN DEFAULT FALSE`,
		`BOOLEAN DEFAULT TRUE`:  `BOOLEAN DEFAULT TRUE`,
		`BOOLEAN NOT NULL`:      `BOOLEAN NOT NULL`,
		`BOOLEAN DEFAULT 0`:     `BOOLEAN DEFAULT FALSE`,
		`BOOLEAN DEFAULT 1`:     `BOOLEAN DEFAULT TRUE`,

		// Text types
		`TEXT`: `TEXT`,

		// Create index
		`CREATE INDEX `: `CREATE INDEX IF NOT EXISTS `,
	}

	for old, new := range replacements {
		content = strings.ReplaceAll(content, old, new)
	}

	// Convert CHECK constraints
	content = regexp.MustCompile(`CHECK\(([^)]+)\)`).ReplaceAllStringFunc(content, func(match string) string {
		return match // PostgreSQL supports CHECK constraints
	})

	// Add IF NOT EXISTS to table creation if missing
	content = regexp.MustCompile(`CREATE TABLE ([^I])`).ReplaceAllString(content, `CREATE TABLE IF NOT EXISTS $1`)

	return content
}

// convertToSQLServer converts SQLite syntax to SQL Server
func (mc *MigrationConverter) convertToSQLServer(content string) string {
	// Add goose directives if missing
	if !strings.Contains(content, "-- +goose Up") {
		parts := strings.SplitN(content, "\n", 1)
		if len(parts) > 0 {
			content = "-- +goose Up\n" + content + "\n\nGO\n\n-- +goose Down\n-- Add rollback logic here\nGO\n"
		}
	}

	var result strings.Builder
	lines := strings.Split(content, "\n")
	inTable := false
	tableName := ""

	for i, line := range lines {
		trimmed := strings.TrimSpace(line)

		// Handle CREATE TABLE
		if strings.HasPrefix(trimmed, "CREATE TABLE") {
			inTable = true
			tableNameMatch := regexp.MustCompile(`CREATE TABLE (?:IF NOT EXISTS )?(\w+)`).FindStringSubmatch(trimmed)
			if len(tableNameMatch) > 1 {
				tableName = tableNameMatch[1]
				result.WriteString(fmt.Sprintf("IF NOT EXISTS (SELECT * FROM sys.tables WHERE name = '%s')\n", tableName))
				result.WriteString("BEGIN\n")
				result.WriteString(fmt.Sprintf("    CREATE TABLE %s (\n", tableName))
				continue
			}
		}

		// Handle end of CREATE TABLE
		if inTable && strings.HasPrefix(trimmed, ");") {
			result.WriteString("    );\n")
			result.WriteString("END\n")
			result.WriteString("GO\n")
			inTable = false
			tableName = ""
			continue
		}

		// Convert data types inside table definition
		if inTable {
			line = strings.ReplaceAll(line, "INTEGER PRIMARY KEY AUTOINCREMENT", "INT IDENTITY(1,1) PRIMARY KEY")
			line = strings.ReplaceAll(line, "INTEGER", "INT")
			line = strings.ReplaceAll(line, "BIGINT PRIMARY KEY AUTOINCREMENT", "BIGINT IDENTITY(1,1) PRIMARY KEY")
			line = strings.ReplaceAll(line, "DATETIME", "DATETIME2")
			line = strings.ReplaceAll(line, "TIMESTAMP", "DATETIME2")
			line = strings.ReplaceAll(line, "TEXT", "NVARCHAR(MAX)")
			line = strings.ReplaceAll(line, "BOOLEAN DEFAULT FALSE", "BIT DEFAULT 0")
			line = strings.ReplaceAll(line, "BOOLEAN DEFAULT TRUE", "BIT DEFAULT 1")
			line = strings.ReplaceAll(line, "BOOLEAN DEFAULT 0", "BIT DEFAULT 0")
			line = strings.ReplaceAll(line, "BOOLEAN DEFAULT 1", "BIT DEFAULT 1")
			line = strings.ReplaceAll(line, "BOOLEAN NOT NULL", "BIT NOT NULL")
			line = strings.ReplaceAll(line, "BOOLEAN", "BIT")
			line = strings.ReplaceAll(line, "CURRENT_TIMESTAMP", "GETUTCDATE()")
			result.WriteString(line)
			if i < len(lines)-1 {
				result.WriteString("\n")
			}
			continue
		}

		// Handle CREATE INDEX
		if strings.HasPrefix(trimmed, "CREATE INDEX") {
			indexMatch := regexp.MustCompile(`CREATE INDEX (?:IF NOT EXISTS )?(\w+) ON (\w+)`).FindStringSubmatch(trimmed)
			if len(indexMatch) > 2 {
				indexName := indexMatch[1]
				tableName := indexMatch[2]
				result.WriteString(fmt.Sprintf("IF NOT EXISTS (SELECT * FROM sys.indexes WHERE name = '%s' AND object_id = OBJECT_ID('%s'))\n", indexName, tableName))
				result.WriteString("BEGIN\n")
				result.WriteString("    " + strings.TrimSuffix(strings.ReplaceAll(trimmed, "CREATE INDEX IF NOT EXISTS ", "CREATE INDEX "), ";") + ";\n")
				result.WriteString("END\n")
				result.WriteString("GO\n")
				continue
			}
		}

		// Handle ALTER TABLE ADD COLUMN
		if strings.HasPrefix(trimmed, "ALTER TABLE") && strings.Contains(trimmed, "ADD COLUMN") {
			alterMatch := regexp.MustCompile(`ALTER TABLE (\w+) ADD COLUMN (\w+) (.+);`).FindStringSubmatch(trimmed)
			if len(alterMatch) > 3 {
				tableName := alterMatch[1]
				columnName := alterMatch[2]
				columnDef := alterMatch[3]

				// Convert data types
				columnDef = strings.ReplaceAll(columnDef, "INTEGER", "INT")
				columnDef = strings.ReplaceAll(columnDef, "DATETIME", "DATETIME2")
				columnDef = strings.ReplaceAll(columnDef, "TIMESTAMP", "DATETIME2")
				columnDef = strings.ReplaceAll(columnDef, "TEXT", "NVARCHAR(MAX)")
				columnDef = strings.ReplaceAll(columnDef, "BOOLEAN DEFAULT FALSE", "BIT DEFAULT 0")
				columnDef = strings.ReplaceAll(columnDef, "BOOLEAN DEFAULT TRUE", "BIT DEFAULT 1")
				columnDef = strings.ReplaceAll(columnDef, "BOOLEAN DEFAULT 0", "BIT DEFAULT 0")
				columnDef = strings.ReplaceAll(columnDef, "BOOLEAN DEFAULT 1", "BIT DEFAULT 1")
				columnDef = strings.ReplaceAll(columnDef, "BOOLEAN", "BIT")
				columnDef = strings.ReplaceAll(columnDef, "CURRENT_TIMESTAMP", "GETUTCDATE()")

				result.WriteString(fmt.Sprintf("IF NOT EXISTS (SELECT * FROM sys.columns WHERE object_id = OBJECT_ID('%s') AND name = '%s')\n", tableName, columnName))
				result.WriteString("BEGIN\n")
				result.WriteString(fmt.Sprintf("    ALTER TABLE %s ADD %s %s;\n", tableName, columnName, columnDef))
				result.WriteString("END\n")
				result.WriteString("GO\n")
				continue
			}
		}

		// Handle DROP statements
		if strings.HasPrefix(trimmed, "DROP TABLE") {
			tableMatch := regexp.MustCompile(`DROP TABLE (?:IF EXISTS )?(\w+)`).FindStringSubmatch(trimmed)
			if len(tableMatch) > 1 {
				result.WriteString(fmt.Sprintf("DROP TABLE IF EXISTS %s;\n", tableMatch[1]))
				result.WriteString("GO\n")
				continue
			}
		}

		if strings.HasPrefix(trimmed, "DROP INDEX") {
			result.WriteString(trimmed + "\n")
			result.WriteString("GO\n")
			continue
		}

		// Default: write line as-is
		result.WriteString(line)
		if i < len(lines)-1 {
			result.WriteString("\n")
		}
	}

	return result.String()
}

func main() {
	if len(os.Args) < 2 {
		fmt.Println("Usage: go run convert-migrations.go <migrations_directory>")
		fmt.Println("Example: go run convert-migrations.go internal/storage/migrations")
		os.Exit(1)
	}

	migrationsDir := os.Args[1]

	// Read all SQLite migration files from the root migrations directory
	files, err := os.ReadDir(migrationsDir)
	if err != nil {
		fmt.Printf("Error reading migrations directory: %v\n", err)
		os.Exit(1)
	}

	converter := &MigrationConverter{
		sourcePath: migrationsDir,
		targetPath: migrationsDir,
	}

	sqliteFiles := []string{}
	for _, file := range files {
		if !file.IsDir() && strings.HasSuffix(file.Name(), ".sql") {
			sqliteFiles = append(sqliteFiles, file.Name())
		}
	}

	fmt.Printf("Found %d SQLite migration files\n", len(sqliteFiles))

	// Convert each file
	for _, filename := range sqliteFiles {
		sourcePath := filepath.Join(migrationsDir, filename)
		content, err := os.ReadFile(sourcePath)
		if err != nil {
			fmt.Printf("Error reading %s: %v\n", filename, err)
			continue
		}

		// SQLite version (copy as-is to sqlite folder)
		sqliteDir := filepath.Join(migrationsDir, "sqlite")
		sqlitePath := filepath.Join(sqliteDir, filename)
		if err := os.WriteFile(sqlitePath, content, 0644); err != nil {
			fmt.Printf("Error writing SQLite version of %s: %v\n", filename, err)
			continue
		}
		fmt.Printf("✓ Copied SQLite version: %s\n", filename)

		// PostgreSQL version
		postgresContent := converter.Convert(string(content), "postgres")
		postgresDir := filepath.Join(migrationsDir, "postgres")
		postgresPath := filepath.Join(postgresDir, filename)
		if err := os.WriteFile(postgresPath, []byte(postgresContent), 0644); err != nil {
			fmt.Printf("Error writing PostgreSQL version of %s: %v\n", filename, err)
			continue
		}
		fmt.Printf("✓ Converted to PostgreSQL: %s\n", filename)

		// SQL Server version
		sqlserverContent := converter.Convert(string(content), "sqlserver")
		sqlserverDir := filepath.Join(migrationsDir, "sqlserver")
		sqlserverPath := filepath.Join(sqlserverDir, filename)
		if err := os.WriteFile(sqlserverPath, []byte(sqlserverContent), 0644); err != nil {
			fmt.Printf("Error writing SQL Server version of %s: %v\n", filename, err)
			continue
		}
		fmt.Printf("✓ Converted to SQL Server: %s\n", filename)
	}

	fmt.Printf("\n✅ Successfully converted %d migrations to all three dialects\n", len(sqliteFiles))
}
