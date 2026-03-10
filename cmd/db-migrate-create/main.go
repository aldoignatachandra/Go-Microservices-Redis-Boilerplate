// Package main generates new sequential SQL migration files.
// Usage: go run ./cmd/db-migrate-create add_column_to_users
package main

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"
)

const migrationsDir = "migrations"

var (
	migrationFileRegex = regexp.MustCompile(`^(\d+)_([a-z0-9_]+)\.sql$`)
	invalidNameRegex   = regexp.MustCompile(`[^a-z0-9_]+`)
)

func main() {
	if len(os.Args) < 2 {
		log.Fatalf("usage: go run ./cmd/db-migrate-create <migration_name>")
	}

	name := normalizeName(os.Args[1])
	if name == "" {
		log.Fatalf("migration name cannot be empty after normalization")
	}

	nextNumber, err := nextMigrationNumber(migrationsDir)
	if err != nil {
		log.Fatalf("failed to determine next migration number: %v", err)
	}

	filename := fmt.Sprintf("%03d_%s.sql", nextNumber, name)
	path := filepath.Join(migrationsDir, filename)
	if _, err := os.Stat(path); err == nil {
		log.Fatalf("migration already exists: %s", path)
	}

	content := buildTemplate()
	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		log.Fatalf("failed to create migration file: %v", err)
	}

	fmt.Printf("✅ Created migration: %s\n", path)
}

func normalizeName(input string) string {
	normalized := strings.ToLower(strings.TrimSpace(input))
	normalized = strings.ReplaceAll(normalized, "-", "_")
	normalized = strings.ReplaceAll(normalized, " ", "_")
	normalized = invalidNameRegex.ReplaceAllString(normalized, "_")
	normalized = strings.Trim(normalized, "_")
	return normalized
}

func nextMigrationNumber(dir string) (int, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return 0, err
	}

	numbers := make([]int, 0, len(entries))
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		matches := migrationFileRegex.FindStringSubmatch(entry.Name())
		if len(matches) < 2 {
			continue
		}
		number, err := strconv.Atoi(matches[1])
		if err != nil {
			return 0, fmt.Errorf("invalid migration prefix in %s: %w", entry.Name(), err)
		}
		numbers = append(numbers, number)
	}

	if len(numbers) == 0 {
		return 1, nil
	}

	sort.Ints(numbers)
	return numbers[len(numbers)-1] + 1, nil
}

func buildTemplate() string {
	return `-- +migrate Up

-- +migrate Down
`
}
