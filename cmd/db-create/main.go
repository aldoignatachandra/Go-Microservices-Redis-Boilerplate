// Package main provides database creation utility.
// Usage: go run ./cmd/db-create
package main

import (
	"fmt"
	"log"

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

	dbHost := utils.GetEnv("DB_HOST", cfg.Database.Host)
	dbPort := utils.GetEnvInt("DB_PORT", cfg.Database.Port)
	dbUser := utils.GetEnv("DB_USER", cfg.Database.User)
	dbPassword := utils.GetEnv("DB_PASSWORD", cfg.Database.Password)
	dbName := utils.GetEnv("DB_NAME", cfg.Database.Name)
	dbSSLMode := utils.GetEnv("DB_SSLMODE", cfg.Database.SSLMode)

	fmt.Printf("Using database config - Host: %s, User: %s, Name: %s\n",
		dbHost, dbUser, dbName)

	dbConfig := &database.PostgresConfig{
		Host:     dbHost,
		Port:     dbPort,
		Name:     dbName,
		User:     dbUser,
		Password: dbPassword,
		SSLMode:  dbSSLMode,
	}

	fmt.Printf("Checking database '%s'...\n", dbConfig.Name)

	if err := database.EnsureDatabase(dbConfig); err != nil {
		log.Fatalf("Failed to ensure database: %v", err)
	}

	fmt.Println("Database is ready!")
}
