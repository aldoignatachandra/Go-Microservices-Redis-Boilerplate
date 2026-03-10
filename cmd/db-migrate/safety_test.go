package main

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestFindDestructiveOperations_DropTable(t *testing.T) {
	queries := []string{
		"DROP TABLE IF EXISTS product_attributes;",
	}

	ops := findDestructiveOperations(queries)
	require.Len(t, ops, 1)
	require.Equal(t, operationDropTable, ops[0].Kind)
	require.Equal(t, "public", ops[0].Schema)
	require.Equal(t, "product_attributes", ops[0].Table)
}

func TestFindDestructiveOperations_DropColumn(t *testing.T) {
	queries := []string{
		"ALTER TABLE user_sessions DROP COLUMN IF EXISTS updated_at;",
	}

	ops := findDestructiveOperations(queries)
	require.Len(t, ops, 1)
	require.Equal(t, operationDropColumn, ops[0].Kind)
	require.Equal(t, "public", ops[0].Schema)
	require.Equal(t, "user_sessions", ops[0].Table)
	require.Equal(t, "updated_at", ops[0].Column)
}

func TestFindDestructiveOperations_QuotedSchemaAndIdentifiers(t *testing.T) {
	queries := []string{
		`DROP TABLE IF EXISTS "audit"."user_sessions";`,
		`ALTER TABLE "audit"."user_sessions" DROP COLUMN "deleted_at";`,
	}

	ops := findDestructiveOperations(queries)
	require.Len(t, ops, 2)

	require.Equal(t, operationDropTable, ops[0].Kind)
	require.Equal(t, "audit", ops[0].Schema)
	require.Equal(t, "user_sessions", ops[0].Table)

	require.Equal(t, operationDropColumn, ops[1].Kind)
	require.Equal(t, "audit", ops[1].Schema)
	require.Equal(t, "user_sessions", ops[1].Table)
	require.Equal(t, "deleted_at", ops[1].Column)
}

func TestFindDestructiveOperations_NonDestructiveQueriesIgnored(t *testing.T) {
	queries := []string{
		`CREATE INDEX IF NOT EXISTS user_sessions_user_id_idx ON user_sessions(user_id);`,
		`UPDATE users SET updated_at = NOW();`,
	}

	ops := findDestructiveOperations(queries)
	require.Empty(t, ops)
}
