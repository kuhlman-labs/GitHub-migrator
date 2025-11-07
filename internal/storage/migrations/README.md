# Database Migrations

This directory contains database migrations organized by database dialect.

## Structure

```
migrations/
├── sqlite/       # SQLite-specific migrations
├── postgres/     # PostgreSQL-specific migrations
├── sqlserver/    # SQL Server-specific migrations
└── common/       # Common migrations (if any work across all databases)
```

## Migration Files

Migrations follow the naming convention: `XXX_description.sql` where XXX is a zero-padded number.

Each migration file should include:
- `-- +goose Up` section for applying the migration
- `-- +goose Down` section for rolling back the migration

## Dialect Differences

### SQLite
- Primary keys: `INTEGER PRIMARY KEY AUTOINCREMENT`
- Timestamps: `DATETIME`
- Booleans: `BOOLEAN` (stored as INTEGER 0/1)
- Text: `TEXT`
- Current timestamp: `CURRENT_TIMESTAMP`

### PostgreSQL
- Primary keys: `SERIAL PRIMARY KEY` or `BIGSERIAL PRIMARY KEY`
- Timestamps: `TIMESTAMP` or `TIMESTAMP WITH TIME ZONE`
- Booleans: `BOOLEAN` (native type)
- Text: `TEXT` or `VARCHAR(n)`
- Current timestamp: `CURRENT_TIMESTAMP` or `NOW()`
- Indexes: Use `CREATE INDEX IF NOT EXISTS`

### SQL Server
- Primary keys: `INT IDENTITY(1,1) PRIMARY KEY`
- Timestamps: `DATETIME2`
- Booleans: `BIT` (0/1)
- Text: `NVARCHAR(MAX)` or `NVARCHAR(n)`
- Current timestamp: `GETUTCDATE()`
- Table existence checks: `IF NOT EXISTS (SELECT * FROM sys.tables WHERE name = 'table_name')`
- Use `GO` statements to separate batches

## Creating New Migrations

When adding a new migration:

1. **Determine if dialect-specific**: If the migration uses standard SQL that works across all databases, you might be able to use a single version. Otherwise, create separate versions for each dialect.

2. **Create migration files**: Create the migration in each dialect folder with the same number:
   ```
   migrations/sqlite/002_add_new_field.sql
   migrations/postgres/002_add_new_field.sql
   migrations/sqlserver/002_add_new_field.sql
   ```

3. **Test**: Test the migration against each database type before committing.

## Migration Loading

The application automatically loads migrations from the correct dialect folder based on the configured database type in `config.yml`:

```yaml
database:
  type: sqlite    # or postgres, sqlserver
  dsn: data/migrator.db
```

## GORM Auto-Migration

For simple schema changes, GORM's AutoMigrate can handle schema synchronization without explicit migrations. However, for data migrations, complex schema changes, or production deployments, explicit migrations are recommended.

## Legacy Migrations

The root `migrations/` folder contains the original SQLite-only migrations. These are kept for reference but new migrations should be placed in the dialect-specific folders.

## Tools

Migrations are managed using the embedded migration system in `database.go` which:
- Tracks applied migrations in the `schema_migrations` table
- Applies migrations in order
- Supports Up migrations (Down migrations are not currently used but included for completeness)
- Automatically converts SQLite syntax to PostgreSQL when needed (legacy support)

## Best Practices

1. **Never modify existing migrations** that have been applied in any environment
2. **Always create new migrations** for schema changes
3. **Test migrations** against all supported database types
4. **Keep migrations small** and focused on a single change
5. **Use transactions** where appropriate (automatic for most operations)
6. **Document complex migrations** with comments explaining the why
7. **Consider data migrations separately** from schema migrations when they might take significant time

## Example Migration

### SQLite Version (002_add_field.sql)
```sql
-- +goose Up
ALTER TABLE repositories ADD COLUMN new_field TEXT;

-- +goose Down
ALTER TABLE repositories DROP COLUMN new_field;
```

### PostgreSQL Version (002_add_field.sql)
```sql
-- +goose Up
ALTER TABLE repositories ADD COLUMN IF NOT EXISTS new_field TEXT;

-- +goose Down
ALTER TABLE repositories DROP COLUMN IF EXISTS new_field;
```

### SQL Server Version (002_add_field.sql)
```sql
-- +goose Up
IF NOT EXISTS (SELECT * FROM sys.columns 
               WHERE object_id = OBJECT_ID('repositories') 
               AND name = 'new_field')
BEGIN
    ALTER TABLE repositories ADD new_field NVARCHAR(MAX);
END
GO

-- +goose Down
IF EXISTS (SELECT * FROM sys.columns 
           WHERE object_id = OBJECT_ID('repositories') 
           AND name = 'new_field')
BEGIN
    ALTER TABLE repositories DROP COLUMN new_field;
END
GO
```

