// Package main provides database seeding functionality.
// This script seeds the database with initial data for development.
package main

import (
	"context"
	"fmt"
	"log"
	"os"

	"golang.org/x/crypto/bcrypt"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"

	authDomain "github.com/ignata/go-microservices-boilerplate/internal/auth/domain"
	productDomain "github.com/ignata/go-microservices-boilerplate/internal/product/domain"
	userDomain "github.com/ignata/go-microservices-boilerplate/internal/user/domain"
)

const (
	adminEmail    = "admin@example.com"
	adminPassword = "Admin123!"
	adminRole     = "ADMIN"

	userEmail    = "user@example.com"
	userPassword = "User123!"
	userRole     = "USER"
)

// SeedData contains all seed data configuration.
type SeedData struct {
	AdminEmail    string
	AdminPassword string
	UserEmail     string
	UserPassword  string
}

// seeder handles database seeding operations.
type seeder struct {
	db *gorm.DB
}

// newSeeder creates a new seeder instance.
func newSeeder(dsn string) (*seeder, error) {
	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Info),
	})
	if err != nil {
		return nil, fmt.Errorf("failed to connect to database: %w", err)
	}

	return &seeder{db: db}, nil
}

// seed runs all seed operations.
func (s *seeder) seed(ctx context.Context) error {
	log.Println("Starting database seed...")

	if err := s.seedUsers(ctx); err != nil {
		return fmt.Errorf("failed to seed users: %w", err)
	}

	if err := s.seedUserProfiles(ctx); err != nil {
		return fmt.Errorf("failed to seed profiles: %w", err)
	}

	if err := s.seedProducts(ctx); err != nil {
		return fmt.Errorf("failed to seed products: %w", err)
	}

	log.Println("✅ Database seeding completed successfully!")
	return nil
}

// hashPassword generates a bcrypt hash of the password.
func (s *seeder) hashPassword(password string) (string, error) {
	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return "", err
	}
	return string(hash), nil
}

// seedUsers creates admin and regular users.
func (s *seeder) seedUsers(ctx context.Context) error {
	log.Println("")

	// Seed Admin User
	adminHash, err := s.hashPassword(adminPassword)
	if err != nil {
		return fmt.Errorf("failed to hash admin password: %w", err)
	}

	var adminCount int64
	s.db.WithContext(ctx).Model(&authDomain.User{}).Where("email = ?", adminEmail).Count(&adminCount)

	if adminCount == 0 {
		admin := &authDomain.User{
			Email:        adminEmail,
			PasswordHash: adminHash,
			Role:         authDomain.RoleAdmin,
		}
		if err := s.db.WithContext(ctx).Create(admin).Error; err != nil {
			return fmt.Errorf("failed to create admin user: %w", err)
		}
		log.Printf("   ✅ Admin user created: %s / %s", adminEmail, adminPassword)
	} else {
		log.Printf("   ⚠️  Admin user already exists, skipping...")
	}

	// Seed Regular User
	userHash, err := s.hashPassword(userPassword)
	if err != nil {
		return fmt.Errorf("failed to hash user password: %w", err)
	}

	var userCount int64
	s.db.WithContext(ctx).Model(&authDomain.User{}).Where("email = ?", userEmail).Count(&userCount)

	if userCount == 0 {
		user := &authDomain.User{
			Email:        userEmail,
			PasswordHash: userHash,
			Role:         authDomain.RoleUser,
		}
		if err := s.db.WithContext(ctx).Create(user).Error; err != nil {
			return fmt.Errorf("failed to create regular user: %w", err)
		}
		log.Printf("   ✅ Regular user created: %s / %s", userEmail, userPassword)
	} else {
		log.Printf("   ⚠️  Regular user already exists, skipping...")
	}

	// Log oldest user for product seeding reference
	var oldestUser authDomain.User
	if err := s.db.WithContext(ctx).Where("role = ? AND deleted_at IS NULL", userRole).
		Order("created_at ASC").First(&oldestUser).Error; err == nil {
		log.Printf("   📋 Oldest USER for product seeder: %s (ID: %s)", oldestUser.Email, oldestUser.ID)
	}

	return nil
}

// seedUserProfiles creates user profiles.
func (s *seeder) seedUserProfiles(ctx context.Context) error {
	log.Println("")

	// Get user IDs
	var users []authDomain.User
	if err := s.db.WithContext(ctx).Find(&users).Error; err != nil {
		return fmt.Errorf("failed to fetch users: %w", err)
	}

	for i := range users {
		u := &users[i]
		var profileCount int64
		s.db.WithContext(ctx).Model(&userDomain.Profile{}).Where("user_id = ?", u.ID).Count(&profileCount)

		if profileCount == 0 {
			profile := &userDomain.Profile{
				UserID:    u.ID,
				FirstName: "Test",
				LastName:  "User",
			}
			if u.Role == authDomain.RoleAdmin {
				profile.FirstName = "Admin"
				profile.LastName = "User"
			}

			if err := s.db.WithContext(ctx).Create(profile).Error; err != nil {
				return fmt.Errorf("failed to create profile for user %s: %w", u.Email, err)
			}
			log.Printf("   ✅ Profile created for: %s", u.Email)
		} else {
			log.Printf("   ⚠️  Profile already exists for: %s, skipping...", u.Email)
		}
	}

	return nil
}

// productSeed represents a product to be seeded.
type productSeed struct {
	name   string
	images string
	price  float64
	stock  int
}

// seedProducts creates sample products.
func (s *seeder) seedProducts(ctx context.Context) error {
	log.Println("")

	// Get a regular user to be the owner
	var owner authDomain.User
	if err := s.db.WithContext(ctx).Where("role = ? AND deleted_at IS NULL", userRole).
		Order("created_at ASC").First(&owner).Error; err != nil {
		return fmt.Errorf("failed to find owner user: %w", err)
	}

	products := []productSeed{
		{
			name:   "Classic Cap",
			price:  19.99,
			stock:  150,
			images: "https://example.com/cap.jpg",
		},
		{
			name:   "Premium T-Shirt",
			price:  29.99,
			stock:  100,
			images: "https://example.com/tshirt.jpg",
		},
		{
			name:   "Wireless Mouse",
			price:  49.99,
			stock:  75,
			images: "https://example.com/mouse.jpg",
		},
		{
			name:   "Mechanical Keyboard",
			price:  89.99,
			stock:  50,
			images: "https://example.com/keyboard.jpg",
		},
		{
			name:   "USB-C Hub",
			price:  39.99,
			stock:  80,
			images: "https://example.com/hub.jpg",
		},
	}

	for _, p := range products {
		var existingProduct productDomain.Product
		err := s.db.WithContext(ctx).Where("name = ? AND deleted_at IS NULL", p.name).First(&existingProduct).Error

		if err == gorm.ErrRecordNotFound {
			product := &productDomain.Product{
				Name:    p.name,
				Price:   p.price,
				Stock:   p.stock,
				Images:  p.images,
				OwnerID: owner.ID,
			}
			if err := s.db.WithContext(ctx).Create(product).Error; err != nil {
				return fmt.Errorf("failed to create product %s: %w", p.name, err)
			}
			log.Printf("   ✅ Product created: %s (Stock: %d, Price: $%.2f)", p.name, p.stock, p.price)
		} else if err == nil {
			log.Printf("   ⚠️  Product already exists: %s, skipping...", p.name)
		} else {
			return fmt.Errorf("failed to check product %s: %w", p.name, err)
		}
	}

	// Print summary
	var totalProducts int64
	s.db.WithContext(ctx).Model(&productDomain.Product{}).Where("deleted_at IS NULL").Count(&totalProducts)
	log.Printf("   📊 Total products in database: %d", totalProducts)

	return nil
}

func main() {
	ctx := context.Background()

	// Get database connection from environment or use default
	host := getEnv("DB_HOST", "localhost")
	port := getEnv("DB_PORT", "5432")
	user := getEnv("DB_USER", "postgres")
	password := getEnv("DB_PASSWORD", "postgres")
	dbName := getEnv("DB_NAME", "microservices_db")
	sslmode := getEnv("DB_SSLMODE", "disable")

	dsn := fmt.Sprintf("host=%s port=%s user=%s password=%s dbname=%s sslmode=%s",
		host, port, user, password, dbName, sslmode)

	log.Println("=====================================")
	log.Println("🚀 Database Seeder")
	log.Println("=====================================")
	log.Printf("   Database: %s", dbName)
	log.Printf("   Host: %s:%s", host, port)
	log.Println("=====================================")

	seeder, err := newSeeder(dsn)
	if err != nil {
		log.Fatalf("Failed to initialize seeder: %v", err)
	}

	// Auto-migrate tables
	log.Println("Running auto-migration...")
	if err := seeder.db.AutoMigrate(
		&authDomain.User{},
		&authDomain.Session{},
		&userDomain.User{},
		&userDomain.Profile{},
		&userDomain.ActivityLog{},
		&productDomain.Product{},
	); err != nil {
		log.Fatalf("Failed to run auto-migration: %v", err)
	}
	log.Println("✅ Auto-migration completed!")

	// Run seeding
	if err := seeder.seed(ctx); err != nil {
		log.Fatalf("Seeding failed: %v", err)
	}

	log.Println("=====================================")
	log.Println("🎉 All seeding operations completed!")
	log.Println("=====================================")
}

func getEnv(key, defaultValue string) string {
	if value, exists := os.LookupEnv(key); exists {
		return value
	}
	return defaultValue
}
