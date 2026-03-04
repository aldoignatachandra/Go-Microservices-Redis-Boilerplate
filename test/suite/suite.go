// Package suite provides test suite utilities for the project.
package suite

import (
	"context"
	"fmt"
	"os"
	"strconv"
	"testing"
	"time"

	"github.com/stretchr/testify/suite"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"

	"github.com/ignata/go-microservices-boilerplate/internal/user/domain"
	"github.com/ignata/go-microservices-boilerplate/pkg/config"
)

// TestSuite is a base test suite for integration tests.
type TestSuite struct {
	suite.Suite
	DB          *gorm.DB
	Config      *config.Config
	CleanupFunc func()
}

// SetupSuite runs once before all tests.
func (s *TestSuite) SetupSuite() {
	// Load test configuration
	cfg, err := config.Load("")
	s.Require().NoError(err)

	// Override with test values
	cfg.Database.Host = GetEnv("TEST_DB_HOST", "localhost")
	cfg.Database.Port = GetEnvInt("TEST_DB_PORT", 5432)
	cfg.Database.Name = GetEnv("TEST_DB_NAME", "test_db")
	cfg.Database.User = GetEnv("TEST_DB_USER", "test")
	cfg.Database.Password = GetEnv("TEST_DB_PASSWORD", "test")

	s.Config = cfg

	// Setup database
	s.setupDatabase()
}

// TearDownSuite runs once after all tests.
func (s *TestSuite) TearDownSuite() {
	if s.CleanupFunc != nil {
		s.CleanupFunc()
	}
}

// SetupTest runs before each test.
func (s *TestSuite) SetupTest() {
	// Clean database before each test
	s.cleanupDatabase()
}

// TearDownTest runs after each test.
func (s *TestSuite) TearDownTest() {
	// Clean database after each test
	s.cleanupDatabase()
}

// setupDatabase sets up the test database.
func (s *TestSuite) setupDatabase() {
	dsn := fmt.Sprintf(
		"host=%s port=%d user=%s password=%s dbname=%s sslmode=disable",
		s.Config.Database.Host,
		s.Config.Database.Port,
		s.Config.Database.User,
		s.Config.Database.Password,
		s.Config.Database.Name,
	)

	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Silent),
	})
	s.Require().NoError(err)

	// Run migrations
	err = db.AutoMigrate(
		&domain.User{},
		&domain.Profile{},
		&domain.ActivityLog{},
	)
	s.Require().NoError(err)

	s.DB = db

	s.CleanupFunc = func() {
		// Close database connection
		if sqlDB, err := db.DB(); err == nil {
			sqlDB.Close()
		}
	}
}

// cleanupDatabase cleans all test data.
func (s *TestSuite) cleanupDatabase() {
	if s.DB == nil {
		return
	}

	s.DB.Unscoped().Where("1 = 1").Delete(&domain.ActivityLog{})
	s.DB.Unscoped().Where("1 = 1").Delete(&domain.Profile{})
	s.DB.Unscoped().Where("1 = 1").Delete(&domain.User{})
}

// GetContext returns a context with timeout.
func (s *TestSuite) GetContext() context.Context {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	s.T().Cleanup(cancel)
	return ctx
}

// RunTestSuite runs the test suite.
func RunTestSuite(t *testing.T, s suite.TestingSuite) {
	suite.Run(t, s)
}

// GetEnv gets an environment variable with default.
func GetEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

// GetEnvInt gets an environment variable as integer with default.
func GetEnvInt(key string, defaultValue int) int {
	if value := os.Getenv(key); value != "" {
		if intVal, err := strconv.Atoi(value); err == nil {
			return intVal
		}
	}
	return defaultValue
}

// RequireEnv requires an environment variable to be set.
func RequireEnv(t *testing.T, key string) string {
	value := os.Getenv(key)
	if value == "" {
		t.Fatalf("Required environment variable %s is not set", key)
	}
	return value
}

// SkipIfShort skips the test if -short flag is set.
func SkipIfShort(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping test in short mode")
	}
}

// SkipUnlessEnv skips the test unless the environment variable is set.
func SkipUnlessEnv(t *testing.T, key string) {
	if os.Getenv(key) == "" {
		t.Skipf("Skipping test: %s environment variable not set", key)
	}
}

// Retry runs a function with retries.
func Retry(t *testing.T, maxAttempts int, delay time.Duration, fn func() error) {
	var err error
	for i := 0; i < maxAttempts; i++ {
		if err = fn(); err == nil {
			return
		}
		if i < maxAttempts-1 {
			time.Sleep(delay)
		}
	}
	t.Fatalf("Function failed after %d attempts: %v", maxAttempts, err)
}
