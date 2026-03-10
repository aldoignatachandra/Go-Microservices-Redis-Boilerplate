// Package main provides file-based database migration utility with step control.
// Usage:
//
//	go run ./cmd/db-migrate up-all
//	go run ./cmd/db-migrate up-one
//	go run ./cmd/db-migrate down-one
//	go run ./cmd/db-migrate down-all
package main

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"os"
	"time"

	_ "github.com/lib/pq"
	migrate "github.com/rubenv/sql-migrate"

	"github.com/ignata/go-microservices-boilerplate/pkg/config"
	"github.com/ignata/go-microservices-boilerplate/pkg/utils"
)

const (
	actionUpAll   = "up-all"
	actionUpOne   = "up-one"
	actionDownOne = "down-one"
	actionDownAll = "down-all"
	dialect       = "postgres"
	migrationsDir = "migrations"
)

func main() {
	utils.LoadEnv()

	action := actionUpAll
	if len(os.Args) > 1 {
		action = os.Args[1]
	}
	if !isValidAction(action) {
		log.Fatalf("invalid action %q. valid actions: %s, %s, %s, %s", action, actionUpAll, actionUpOne, actionDownOne, actionDownAll)
	}

	cfg, err := config.Load("")
	if err != nil {
		log.Fatalf("failed to load config: %v", err)
	}

	dbHost := utils.GetEnv("DB_HOST", cfg.Database.Host)
	dbPort := utils.GetEnvInt("DB_PORT", cfg.Database.Port)
	dbUser := utils.GetEnv("DB_USER", cfg.Database.User)
	dbPassword := utils.GetEnv("DB_PASSWORD", cfg.Database.Password)
	dbName := utils.GetEnv("DB_NAME", cfg.Database.Name)
	dbSSLMode := utils.GetEnv("DB_SSLMODE", cfg.Database.SSLMode)

	dsn := fmt.Sprintf("host=%s port=%d user=%s password=%s dbname=%s sslmode=%s",
		dbHost, dbPort, dbUser, dbPassword, dbName, dbSSLMode)

	db, err := sql.Open(dialect, dsn)
	if err != nil {
		log.Fatalf("failed to open database connection: %v", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)

	if err := db.PingContext(ctx); err != nil {
		cancel()
		_ = db.Close()
		log.Fatalf("failed to ping database: %v", err)
	}
	cancel()

	source := &migrate.FileMigrationSource{Dir: migrationsDir}

	applied, migrationIDs, err := runMigration(ctx, db, source, action)
	if err != nil {
		_ = db.Close()
		log.Fatalf("migration %s failed: %v", action, err)
	}
	_ = db.Close()

	if applied == 0 {
		fmt.Printf("ℹ️  No migrations applied for action '%s'.\n", action)
		return
	}

	fmt.Printf("✅ Applied %d migration(s) using action '%s'.\n", applied, action)
	printAppliedMigrations(applied, migrationIDs)
}

func runMigration(
	ctx context.Context,
	db *sql.DB,
	source *migrate.FileMigrationSource,
	action string,
) (int, []string, error) {
	switch action {
	case actionUpAll:
		planned, err := plannedMigrationIDs(db, source, migrate.Up, 0)
		if err != nil {
			return 0, nil, err
		}
		applied, err := migrate.ExecContext(ctx, db, dialect, source, migrate.Up)
		return applied, planned, err
	case actionUpOne:
		planned, err := plannedMigrationIDs(db, source, migrate.Up, 1)
		if err != nil {
			return 0, nil, err
		}
		applied, err := migrate.ExecMaxContext(ctx, db, dialect, source, migrate.Up, 1)
		return applied, planned, err
	case actionDownOne:
		if err := validateDownSafety(ctx, db, source, 1); err != nil {
			return 0, nil, err
		}
		planned, err := plannedMigrationIDs(db, source, migrate.Down, 1)
		if err != nil {
			return 0, nil, err
		}
		applied, err := migrate.ExecMaxContext(ctx, db, dialect, source, migrate.Down, 1)
		return applied, planned, err
	case actionDownAll:
		if err := validateDownSafety(ctx, db, source, 0); err != nil {
			return 0, nil, err
		}
		planned, err := plannedMigrationIDs(db, source, migrate.Down, 0)
		if err != nil {
			return 0, nil, err
		}
		applied, err := migrate.ExecContext(ctx, db, dialect, source, migrate.Down)
		return applied, planned, err
	default:
		return 0, nil, fmt.Errorf("unsupported migration action: %s", action)
	}
}

func isValidAction(action string) bool {
	return action == actionUpAll || action == actionUpOne || action == actionDownOne || action == actionDownAll
}

func plannedMigrationIDs(
	db *sql.DB,
	source *migrate.FileMigrationSource,
	dir migrate.MigrationDirection,
	limit int,
) ([]string, error) {
	planned, _, err := migrate.PlanMigration(db, dialect, source, dir, limit)
	if err != nil {
		return nil, fmt.Errorf("failed to plan migrations: %w", err)
	}

	ids := make([]string, 0, len(planned))
	for _, migrationPlan := range planned {
		ids = append(ids, migrationPlan.Id)
	}
	return ids, nil
}

func printAppliedMigrations(applied int, migrationIDs []string) {
	limit := applied
	if len(migrationIDs) < limit {
		limit = len(migrationIDs)
	}

	for i := 0; i < limit; i++ {
		fmt.Printf("   • migrations/%s\n", migrationIDs[i])
	}
}
