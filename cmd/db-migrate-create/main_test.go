package main

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestNormalizeName(t *testing.T) {
	tests := map[string]string{
		"Add Column To Users": "add_column_to_users",
		"add-column-to-users": "add_column_to_users",
		"__add$$$column__":    "add_column",
	}

	for input, expected := range tests {
		got := normalizeName(input)
		require.Equal(t, expected, got)
	}
}

func TestNextMigrationNumber(t *testing.T) {
	tempDir := t.TempDir()

	files := []string{
		"001_create_users_table.sql",
		"002_create_user_sessions_table.sql",
		"010_add_new_index.sql",
		"README.md",
	}
	for _, name := range files {
		path := filepath.Join(tempDir, name)
		require.NoError(t, os.WriteFile(path, []byte("-- test"), 0o644))
	}

	next, err := nextMigrationNumber(tempDir)
	require.NoError(t, err)
	require.Equal(t, 11, next)
}
