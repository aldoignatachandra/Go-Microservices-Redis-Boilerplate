package main

import (
	"context"
	"database/sql"
	"fmt"
	"os"
	"regexp"
	"strings"

	migrate "github.com/rubenv/sql-migrate"
)

const (
	operationDropTable  = "drop_table"
	operationDropColumn = "drop_column"
	forceMigrationEnv   = "MIGRATION_FORCE"
)

var (
	dropTableRegex  = regexp.MustCompile(`(?is)^\s*DROP\s+TABLE\s+(?:IF\s+EXISTS\s+)?((?:"?[A-Za-z_][A-Za-z0-9_]*"?\.)?"?[A-Za-z_][A-Za-z0-9_]*"?)`)
	dropColumnRegex = regexp.MustCompile(`(?is)^\s*ALTER\s+TABLE\s+((?:"?[A-Za-z_][A-Za-z0-9_]*"?\.)?"?[A-Za-z_][A-Za-z0-9_]*"?)\s+DROP\s+COLUMN\s+(?:IF\s+EXISTS\s+)?("?([A-Za-z_][A-Za-z0-9_]*)"?)`)
)

type destructiveOperation struct {
	Kind     string
	Schema   string
	Table    string
	Column   string
	RawQuery string
}

func validateDownSafety(
	ctx context.Context,
	db *sql.DB,
	source *migrate.FileMigrationSource,
	limit int,
) error {
	if isForceMigrationEnabled() {
		return nil
	}

	planned, _, err := migrate.PlanMigration(db, dialect, source, migrate.Down, limit)
	if err != nil {
		return fmt.Errorf("failed to plan down migration safety checks: %w", err)
	}
	if len(planned) == 0 {
		return nil
	}

	for _, migrationPlan := range planned {
		operations := findDestructiveOperations(migrationPlan.Queries)
		if len(operations) == 0 {
			continue
		}

		if err := validateDestructiveOperations(ctx, db, migrationPlan, operations); err != nil {
			return err
		}
	}

	return nil
}

func validateDestructiveOperations(
	ctx context.Context,
	db *sql.DB,
	migrationPlan *migrate.PlannedMigration,
	operations []destructiveOperation,
) error {
	for _, operation := range operations {
		switch operation.Kind {
		case operationDropTable:
			hasRows, err := tableHasRows(ctx, db, operation.Schema, operation.Table)
			if err != nil {
				return err
			}
			if hasRows {
				return fmt.Errorf(
					"safety check blocked down migration %s: table %s.%s contains data; set %s=1 to force",
					migrationPlan.Id,
					operation.Schema,
					operation.Table,
					forceMigrationEnv,
				)
			}
		case operationDropColumn:
			hasValues, err := columnHasValues(ctx, db, operation.Schema, operation.Table, operation.Column)
			if err != nil {
				return err
			}
			if hasValues {
				return fmt.Errorf(
					"safety check blocked down migration %s: column %s.%s.%s contains data; set %s=1 to force",
					migrationPlan.Id,
					operation.Schema,
					operation.Table,
					operation.Column,
					forceMigrationEnv,
				)
			}
		}
	}

	return nil
}

func findDestructiveOperations(queries []string) []destructiveOperation {
	operations := make([]destructiveOperation, 0)

	for _, query := range queries {
		normalized := strings.TrimSpace(query)
		if normalized == "" {
			continue
		}

		if match := dropTableRegex.FindStringSubmatch(normalized); len(match) > 1 {
			schema, table := splitQualifiedIdentifier(match[1])
			operations = append(operations, destructiveOperation{
				Kind:     operationDropTable,
				Schema:   schema,
				Table:    table,
				RawQuery: normalized,
			})
			continue
		}

		if match := dropColumnRegex.FindStringSubmatch(normalized); len(match) > 3 {
			schema, table := splitQualifiedIdentifier(match[1])
			column := unquoteIdentifier(match[3])
			operations = append(operations, destructiveOperation{
				Kind:     operationDropColumn,
				Schema:   schema,
				Table:    table,
				Column:   column,
				RawQuery: normalized,
			})
		}
	}

	return operations
}

func tableHasRows(ctx context.Context, db *sql.DB, schema, table string) (bool, error) {
	exists, err := tableExists(ctx, db, schema, table)
	if err != nil {
		return false, err
	}
	if !exists {
		return false, nil
	}

	// #nosec G201 -- identifiers are internally parsed and quoted before interpolation.
	countQuery := fmt.Sprintf("SELECT COUNT(1) FROM %s", qualifiedIdentifier(schema, table))
	var count int64
	if err := db.QueryRowContext(ctx, countQuery).Scan(&count); err != nil {
		return false, fmt.Errorf("failed counting rows for %s.%s: %w", schema, table, err)
	}

	return count > 0, nil
}

func columnHasValues(ctx context.Context, db *sql.DB, schema, table, column string) (bool, error) {
	columnExists, err := tableColumnExists(ctx, db, schema, table, column)
	if err != nil {
		return false, err
	}
	if !columnExists {
		return false, nil
	}

	// #nosec G201 -- identifiers are internally parsed and quoted before interpolation.
	countQuery := fmt.Sprintf(
		"SELECT COUNT(1) FROM %s WHERE %s IS NOT NULL",
		qualifiedIdentifier(schema, table),
		quoteIdentifier(column),
	)
	var count int64
	if err := db.QueryRowContext(ctx, countQuery).Scan(&count); err != nil {
		return false, fmt.Errorf("failed counting non-null values for %s.%s.%s: %w", schema, table, column, err)
	}

	return count > 0, nil
}

func tableExists(ctx context.Context, db *sql.DB, schema, table string) (bool, error) {
	var exists bool
	err := db.QueryRowContext(
		ctx,
		`SELECT EXISTS (
			SELECT 1
			FROM information_schema.tables
			WHERE table_schema = $1 AND table_name = $2
		)`,
		schema,
		table,
	).Scan(&exists)
	if err != nil {
		return false, fmt.Errorf("failed checking table existence for %s.%s: %w", schema, table, err)
	}
	return exists, nil
}

func tableColumnExists(ctx context.Context, db *sql.DB, schema, table, column string) (bool, error) {
	var exists bool
	err := db.QueryRowContext(
		ctx,
		`SELECT EXISTS (
			SELECT 1
			FROM information_schema.columns
			WHERE table_schema = $1 AND table_name = $2 AND column_name = $3
		)`,
		schema,
		table,
		column,
	).Scan(&exists)
	if err != nil {
		return false, fmt.Errorf("failed checking column existence for %s.%s.%s: %w", schema, table, column, err)
	}
	return exists, nil
}

func splitQualifiedIdentifier(raw string) (schema string, table string) {
	parts := strings.SplitN(strings.TrimSpace(raw), ".", 2)
	if len(parts) == 1 {
		return "public", unquoteIdentifier(parts[0])
	}

	return unquoteIdentifier(parts[0]), unquoteIdentifier(parts[1])
}

func unquoteIdentifier(value string) string {
	return strings.Trim(strings.TrimSpace(value), `"`)
}

func qualifiedIdentifier(schema, table string) string {
	return quoteIdentifier(schema) + "." + quoteIdentifier(table)
}

func quoteIdentifier(identifier string) string {
	return `"` + strings.ReplaceAll(identifier, `"`, `""`) + `"`
}

func isForceMigrationEnabled() bool {
	value := strings.ToLower(strings.TrimSpace(os.Getenv(forceMigrationEnv)))
	return value == "1" || value == "true" || value == "yes"
}
