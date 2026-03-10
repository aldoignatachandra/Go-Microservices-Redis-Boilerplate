// Package database provides database connection management.
// It includes PostgreSQL (via GORM) and Redis connection utilities.
package database

import (
	"context"
	"fmt"
	"strings"
	"time"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

// PostgresConfig holds PostgreSQL connection configuration.
type PostgresConfig struct {
	Host            string
	Port            int
	Name            string
	User            string
	Password        string
	SSLMode         string
	MaxOpenConns    int
	MaxIdleConns    int
	ConnMaxLifetime time.Duration
}

// PostgresDB wraps the GORM DB with additional utilities.
type PostgresDB struct {
	*gorm.DB
}

// NewPostgresConnection creates a new PostgreSQL connection using GORM.
// It configures connection pooling and sets up logging based on environment.
func NewPostgresConnection(cfg *PostgresConfig) (*PostgresDB, error) {
	dsn := fmt.Sprintf(
		"host=%s port=%d user=%s password=%s dbname=%s sslmode=%s",
		cfg.Host, cfg.Port, cfg.User, cfg.Password, cfg.Name, cfg.SSLMode,
	)

	// Configure GORM logger
	var gormLogger logger.Interface
	if cfg.SSLMode == "disable" {
		// Development mode: log all queries
		gormLogger = logger.Default.LogMode(logger.Info)
	} else {
		// Production mode: only log errors
		gormLogger = logger.Default.LogMode(logger.Error)
	}

	// Open connection
	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{
		Logger: gormLogger,
		NowFunc: func() time.Time {
			return time.Now().UTC()
		},
		// Disable automatic transaction for nested transactions
		SkipDefaultTransaction: true,
		// Prepare statement cache for better performance
		PrepareStmt: true,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to connect to database: %w", err)
	}

	// Get underlying sql.DB for connection pool configuration
	sqlDB, err := db.DB()
	if err != nil {
		return nil, fmt.Errorf("failed to get sql.DB: %w", err)
	}

	// Configure connection pool
	sqlDB.SetMaxOpenConns(cfg.MaxOpenConns)
	sqlDB.SetMaxIdleConns(cfg.MaxIdleConns)
	sqlDB.SetConnMaxLifetime(cfg.ConnMaxLifetime)

	// Verify connection
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := sqlDB.PingContext(ctx); err != nil {
		return nil, fmt.Errorf("failed to ping database: %w", err)
	}

	return &PostgresDB{DB: db}, nil
}

// Ping checks if the database connection is alive.
func (db *PostgresDB) Ping(ctx context.Context) error {
	sqlDB, err := db.DB.DB()
	if err != nil {
		return fmt.Errorf("failed to get sql.DB: %w", err)
	}

	return sqlDB.PingContext(ctx)
}

// Close closes the database connection.
func (db *PostgresDB) Close() error {
	sqlDB, err := db.DB.DB()
	if err != nil {
		return fmt.Errorf("failed to get sql.DB: %w", err)
	}

	return sqlDB.Close()
}

// Health returns health status of the database connection.
func (db *PostgresDB) Health(ctx context.Context) map[string]interface{} {
	status := make(map[string]interface{})

	if err := db.Ping(ctx); err != nil {
		status["status"] = "unhealthy"
		status["error"] = err.Error()
		return status
	}

	// Get connection stats
	sqlDB, _ := db.DB.DB()
	stats := sqlDB.Stats()

	status["status"] = "healthy"
	status["open_connections"] = stats.OpenConnections
	status["in_use"] = stats.InUse
	status["idle"] = stats.Idle
	status["max_open"] = stats.MaxOpenConnections
	status["max_idle"] = stats.MaxIdleClosed
	status["wait_count"] = stats.WaitCount
	status["wait_duration"] = stats.WaitDuration.String()

	return status
}

// WithContext returns a new DB instance with the given context.
func (db *PostgresDB) WithContext(ctx context.Context) *gorm.DB {
	return db.DB.WithContext(ctx)
}

// Transaction executes a function within a database transaction.
// If the function returns an error, the transaction is rolled back.
// Otherwise, the transaction is committed.
func (db *PostgresDB) Transaction(ctx context.Context, fn func(tx *gorm.DB) error) error {
	return db.WithContext(ctx).Transaction(fn)
}

// IsUniqueViolation checks if the error is a unique constraint violation.
func IsUniqueViolation(err error) bool {
	if err == nil {
		return false
	}
	// PostgreSQL unique violation error code
	return err.Error() != "" && (containsStr(err.Error(), "duplicate key") ||
		containsStr(err.Error(), "unique constraint"))
}

// containsStr checks if s contains substr (case-insensitive).
func containsStr(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr ||
		(len(s) > len(substr) && containsSubstring(s, substr)))
}

func containsSubstring(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		match := true
		for j := 0; j < len(substr); j++ {
			sc := s[i+j]
			subc := substr[j]
			// Simple lowercase comparison
			if sc >= 'A' && sc <= 'Z' {
				sc += 32
			}
			if subc >= 'A' && subc <= 'Z' {
				subc += 32
			}
			if sc != subc {
				match = false
				break
			}
		}
		if match {
			return true
		}
	}
	return false
}

// EnsureDatabase creates the database if it doesn't exist.
// It connects to the default 'postgres' database to check and create the target database.
func EnsureDatabase(cfg *PostgresConfig) error {
	dbName := strings.TrimSpace(cfg.Name)
	if dbName == "" {
		return fmt.Errorf("database name cannot be empty")
	}

	// Connect to the 'postgres' maintenance database to check/create target DB
	maintenanceDSN := fmt.Sprintf(
		"host=%s port=%d user=%s password=%s dbname=postgres sslmode=%s",
		cfg.Host, cfg.Port, cfg.User, cfg.Password, cfg.SSLMode,
	)

	db, err := gorm.Open(postgres.Open(maintenanceDSN), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Silent),
	})
	if err != nil {
		return fmt.Errorf("failed to connect to maintenance database: %w", err)
	}

	sqlDB, err := db.DB()
	if err != nil {
		return fmt.Errorf("failed to get sql.DB: %w", err)
	}
	defer func() { _ = sqlDB.Close() }()

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	exists, err := databaseExists(ctx, db, dbName)
	if err != nil {
		return err
	}
	if !exists {
		if err := createDatabaseAndVerify(ctx, db, dbName); err != nil {
			return err
		}
		fmt.Printf("✨ Database '%s' created successfully!\n", dbName)
	} else {
		fmt.Printf("ℹ️  Database '%s' already exists. Skipping creation.\n", dbName)
	}

	// Connect to the target database to enable extension
	newDBDSN := fmt.Sprintf(
		"host=%s port=%d user=%s password=%s dbname=%s sslmode=%s",
		cfg.Host, cfg.Port, cfg.User, cfg.Password, dbName, cfg.SSLMode,
	)
	newDB, err := gorm.Open(postgres.Open(newDBDSN), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Silent),
	})
	if err != nil {
		return fmt.Errorf("failed to connect to new database: %w", err)
	}

	newSQLDB, err := newDB.DB()
	if err != nil {
		return fmt.Errorf("failed to get new sql.DB: %w", err)
	}
	defer func() { _ = newSQLDB.Close() }()

	extensionResult := newDB.WithContext(ctx).Exec("CREATE EXTENSION IF NOT EXISTS \"uuid-ossp\"")
	if extensionResult.Error != nil {
		fmt.Printf("⚠️  Warning: failed to enable uuid-ossp extension: %v\n", extensionResult.Error)
	}

	return nil
}

// createDatabaseAndVerify creates a database and verifies it is accessible in pg_database.
func createDatabaseAndVerify(ctx context.Context, db *gorm.DB, dbName string) error {
	createSQL := fmt.Sprintf("CREATE DATABASE %s", quoteIdentifier(dbName))
	createResult := db.WithContext(ctx).Exec(createSQL)
	if createResult.Error != nil {
		return fmt.Errorf("failed to create database %s: %w", dbName, createResult.Error)
	}

	exists, err := databaseExists(ctx, db, dbName)
	if err != nil {
		return err
	}
	if !exists {
		return fmt.Errorf("database %s still does not exist after create attempt", dbName)
	}

	return nil
}

// DropDatabase drops the database if it exists.
// It connects to the default 'postgres' database to drop the target database.
func DropDatabase(cfg *PostgresConfig) error {
	dbName := strings.TrimSpace(cfg.Name)
	if dbName == "" {
		return fmt.Errorf("database name cannot be empty")
	}

	// Connect to the 'postgres' maintenance database to drop target DB
	maintenanceDSN := fmt.Sprintf(
		"host=%s port=%d user=%s password=%s dbname=postgres sslmode=%s",
		cfg.Host, cfg.Port, cfg.User, cfg.Password, cfg.SSLMode,
	)

	db, err := gorm.Open(postgres.Open(maintenanceDSN), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Silent),
	})
	if err != nil {
		return fmt.Errorf("failed to connect to maintenance database: %w", err)
	}

	sqlDB, err := db.DB()
	if err != nil {
		return fmt.Errorf("failed to get sql.DB: %w", err)
	}
	defer func() { _ = sqlDB.Close() }()

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	exists, err := databaseExists(ctx, db, dbName)
	if err != nil {
		return err
	}
	if !exists {
		fmt.Printf("ℹ️  Database '%s' does not exist. Nothing to drop.\n", dbName)
		return nil
	}

	if err := dropDatabaseAndVerify(ctx, db, dbName); err != nil {
		return err
	}

	fmt.Printf("✨ Database '%s' dropped successfully!\n", dbName)
	return nil
}

// dropDatabaseAndVerify drops a database and verifies it is truly removed.
func dropDatabaseAndVerify(ctx context.Context, db *gorm.DB, dbName string) error {
	// Terminate active connections so DROP DATABASE can succeed.
	terminateResult := db.WithContext(ctx).Exec(
		"SELECT pg_terminate_backend(pid) FROM pg_stat_activity WHERE datname = ? AND pid <> pg_backend_pid()",
		dbName,
	)
	if terminateResult.Error != nil {
		return fmt.Errorf("failed to terminate active connections for database %s: %w", dbName, terminateResult.Error)
	}

	dropSQL := fmt.Sprintf("DROP DATABASE %s", quoteIdentifier(dbName))
	dropResult := db.WithContext(ctx).Exec(dropSQL)
	if dropResult.Error != nil {
		return fmt.Errorf("failed to drop database %s: %w", dbName, dropResult.Error)
	}

	exists, err := databaseExists(ctx, db, dbName)
	if err != nil {
		return err
	}
	if exists {
		return fmt.Errorf("database %s still exists after drop attempt", dbName)
	}

	return nil
}

// databaseExists checks if a PostgreSQL database exists by name.
func databaseExists(ctx context.Context, db *gorm.DB, dbName string) (bool, error) {
	var count int64
	result := db.WithContext(ctx).
		Raw("SELECT COUNT(*) FROM pg_database WHERE datname = ?", dbName).
		Scan(&count)
	if result.Error != nil {
		return false, fmt.Errorf("failed to check database existence for %s: %w", dbName, result.Error)
	}
	return count > 0, nil
}

// quoteIdentifier safely quotes PostgreSQL identifiers (e.g. database names).
func quoteIdentifier(identifier string) string {
	return `"` + strings.ReplaceAll(identifier, `"`, `""`) + `"`
}
