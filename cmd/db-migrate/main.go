// Package main provides database migration utility using GORM AutoMigrate.
// Usage: go run ./cmd/db-migrate
package main

import (
	"fmt"
	"log"

	authdomain "github.com/ignata/go-microservices-boilerplate/internal/auth/domain"
	productdomain "github.com/ignata/go-microservices-boilerplate/internal/product/domain"
	userdomain "github.com/ignata/go-microservices-boilerplate/internal/user/domain"
	"github.com/ignata/go-microservices-boilerplate/pkg/config"
	"github.com/ignata/go-microservices-boilerplate/pkg/database"
	"github.com/ignata/go-microservices-boilerplate/pkg/utils"
)

func main() {
	utils.LoadEnv()

	cfg, err := config.Load("")
	if err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}

	db, err := database.NewPostgresConnection(&database.PostgresConfig{
		Host:            cfg.Database.Host,
		Port:            cfg.Database.Port,
		Name:            cfg.Database.Name,
		User:            cfg.Database.User,
		Password:        cfg.Database.Password,
		SSLMode:         cfg.Database.SSLMode,
		MaxOpenConns:    cfg.Database.MaxOpenConns,
		MaxIdleConns:    cfg.Database.MaxIdleConns,
		ConnMaxLifetime: cfg.Database.ConnMaxLifetime,
	})
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}

	fmt.Println("Running AutoMigrate...")

	// AutoMigrate all domains
	err = db.AutoMigrate(
		// Auth domain
		&authdomain.User{},
		&authdomain.Session{},
		// User domain
		&userdomain.ActivityLog{},
		// Product domain
		&productdomain.Product{},
		&productdomain.ProductVariant{},
		&productdomain.ProductAttribute{},
	)
	if err != nil {
		log.Fatalf("Failed to run AutoMigrate: %v", err)
	}

	fmt.Println("✅ AutoMigrate completed successfully!")
}
