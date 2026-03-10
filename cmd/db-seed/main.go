// Package main provides database seeding functionality.
// This script seeds the database with initial data for development.
package main

import (
	"context"
	"fmt"
	"io"
	"log"
	"os"
	"time"

	"golang.org/x/crypto/bcrypt"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"

	authDomain "github.com/ignata/go-microservices-boilerplate/internal/auth/domain"
	productDomain "github.com/ignata/go-microservices-boilerplate/internal/product/domain"
	"github.com/ignata/go-microservices-boilerplate/pkg/utils"
)

const (
	adminEmail    = "admin@example.com"
	adminPassword = "Admin123!"

	userEmail    = "user@example.com"
	userPassword = "User123!"
	userRole     = "USER"
)

// seeder handles database seeding operations.
type seeder struct {
	db *gorm.DB
}

func newGormLogger(writer io.Writer) logger.Interface {
	if writer == nil {
		writer = os.Stdout
	}

	return logger.New(
		log.New(writer, "", log.LstdFlags),
		logger.Config{
			SlowThreshold:             200 * time.Millisecond,
			LogLevel:                  logger.Info,
			IgnoreRecordNotFoundError: true,
			Colorful:                  false,
		},
	)
}

// newSeeder creates a new seeder instance.
func newSeeder(dsn string) (*seeder, error) {
	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{
		Logger: newGormLogger(os.Stdout),
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

	if err := s.seedProducts(ctx); err != nil {
		return fmt.Errorf("failed to seed products: %w", err)
	}

	log.Println("✅ Database seeding completed successfully!")
	return nil
}

func (s *seeder) validateRequiredTables(ctx context.Context) error {
	requiredTables := []string{
		"users",
		"user_sessions",
		"user_activity_logs",
		"products",
		"product_variants",
		"product_attributes",
	}

	for _, tableName := range requiredTables {
		var exists bool
		result := s.db.WithContext(ctx).Raw(
			`SELECT EXISTS (
				SELECT 1
				FROM information_schema.tables
				WHERE table_schema = 'public' AND table_name = ?
			)`,
			tableName,
		).Scan(&exists)
		if result.Error != nil {
			return fmt.Errorf("failed checking table %s: %w", tableName, result.Error)
		}
		if !exists {
			return fmt.Errorf("required table %s does not exist; run `make db-migrate` first", tableName)
		}
	}

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
			Username:     "admin",
			Name:         "Admin",
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
			Username:     "testuser",
			Name:         "Test User",
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

// productSeed represents a product to be seeded.
//
//nolint:govet // Field order favors readability for static seed declarations.
type productSeed struct {
	attributes []attributeSeed
	variants   []variantSeed
	name       string
	images     string
	price      float64
	stock      int
}

type attributeSeed struct {
	name         string
	values       []string
	displayOrder int
}

type variantSeed struct {
	attributeValues map[string]string
	name            string
	sku             string
	images          string
	price           float64
	stockQuantity   int
}

func defaultProductSeeds() []productSeed {
	return []productSeed{
		{
			name:   "Classic Cap",
			price:  19.99,
			stock:  150,
			images: "https://example.com/cap.jpg",
		},
		{
			name:   "Premium T-Shirt",
			price:  29.99,
			stock:  0,
			images: "https://example.com/tshirt.jpg",
			attributes: []attributeSeed{
				{name: "Color", values: []string{"Red", "Blue", "Black"}, displayOrder: 0},
				{name: "Size", values: []string{"S", "M", "L"}, displayOrder: 1},
			},
			variants: []variantSeed{
				{
					name:          "Red / S",
					sku:           "TSHIRT-RED-S",
					price:         29.99,
					stockQuantity: 40,
					attributeValues: map[string]string{
						"Color": "Red",
						"Size":  "S",
					},
					images: "https://example.com/tshirt-red-s.jpg",
				},
				{
					name:          "Blue / M",
					sku:           "TSHIRT-BLUE-M",
					price:         34.99,
					stockQuantity: 35,
					attributeValues: map[string]string{
						"Color": "Blue",
						"Size":  "M",
					},
					images: "https://example.com/tshirt-blue-m.jpg",
				},
				{
					name:          "Black / L",
					sku:           "TSHIRT-BLACK-L",
					price:         39.99,
					stockQuantity: 30,
					attributeValues: map[string]string{
						"Color": "Black",
						"Size":  "L",
					},
					images: "https://example.com/tshirt-black-l.jpg",
				},
			},
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
			stock:  0,
			images: "https://example.com/keyboard.jpg",
			attributes: []attributeSeed{
				{name: "Layout", values: []string{"TKL", "Full"}, displayOrder: 0},
				{name: "Switch", values: []string{"Brown", "Red"}, displayOrder: 1},
			},
			variants: []variantSeed{
				{
					name:          "TKL / Brown Switch",
					sku:           "KEYBOARD-TKL-BROWN",
					price:         89.99,
					stockQuantity: 20,
					attributeValues: map[string]string{
						"Layout": "TKL",
						"Switch": "Brown",
					},
					images: "https://example.com/keyboard-tkl-brown.jpg",
				},
				{
					name:          "Full / Red Switch",
					sku:           "KEYBOARD-FULL-RED",
					price:         99.99,
					stockQuantity: 15,
					attributeValues: map[string]string{
						"Layout": "Full",
						"Switch": "Red",
					},
					images: "https://example.com/keyboard-full-red.jpg",
				},
			},
		},
		{
			name:   "USB-C Hub",
			price:  39.99,
			stock:  80,
			images: "https://example.com/hub.jpg",
		},
	}
}

func totalVariantStock(variants []variantSeed) int {
	total := 0
	for _, v := range variants {
		total += v.stockQuantity
	}
	return total
}

func variantSeedRunMessage(productName string, configured, created, updated int) string {
	if configured == 0 {
		return fmt.Sprintf(
			"      📊 Variant seed result [%s]: created=%d updated=%d (no configured variants)",
			productName,
			created,
			updated,
		)
	}

	return fmt.Sprintf(
		"      📊 Variant seed result [%s]: created=%d updated=%d configured=%d",
		productName,
		created,
		updated,
		configured,
	)
}

func (s *seeder) ensureSeedProduct(ctx context.Context, ownerID string, p productSeed) (*productDomain.Product, error) {
	var existingProduct productDomain.Product
	err := s.db.WithContext(ctx).
		Where("name = ? AND owner_id = ? AND deleted_at IS NULL", p.name, ownerID).
		First(&existingProduct).Error
	if err == nil {
		log.Printf("   ⚠️  Product already exists: %s, skipping...", p.name)
		return &existingProduct, nil
	}
	if err != gorm.ErrRecordNotFound {
		return nil, fmt.Errorf("failed to check product %s: %w", p.name, err)
	}

	hasVariant := len(p.variants) > 0
	stock := p.stock
	if hasVariant {
		stock = totalVariantStock(p.variants)
	}

	product := &productDomain.Product{
		Name:       p.name,
		Price:      p.price,
		Stock:      stock,
		HasVariant: hasVariant,
		Images:     p.images,
		OwnerID:    ownerID,
	}
	if err := s.db.WithContext(ctx).Create(product).Error; err != nil {
		return nil, fmt.Errorf("failed to create product %s: %w", p.name, err)
	}
	log.Printf("   ✅ Product created: %s (Stock: %d, Price: $%.2f)", p.name, stock, p.price)
	return product, nil
}

func (s *seeder) seedProductAttributes(ctx context.Context, productID string, p productSeed) error {
	for _, a := range p.attributes {
		var existingAttribute productDomain.ProductAttribute
		attrErr := s.db.WithContext(ctx).
			Unscoped().
			Where("product_id = ? AND name = ?", productID, a.name).
			First(&existingAttribute).Error

		switch {
		case attrErr == gorm.ErrRecordNotFound:
			attribute := &productDomain.ProductAttribute{
				ProductID:    productID,
				Name:         a.name,
				Values:       a.values,
				DisplayOrder: a.displayOrder,
			}
			if err := s.db.WithContext(ctx).Create(attribute).Error; err != nil {
				return fmt.Errorf("failed to create attribute %s for product %s: %w", a.name, p.name, err)
			}
			log.Printf("      ✅ Attribute created: %s (%s)", a.name, p.name)
		case attrErr == nil:
			updatePayload := &productDomain.ProductAttribute{
				Values:       a.values,
				DisplayOrder: a.displayOrder,
				DeletedAt:    gorm.DeletedAt{},
			}
			if err := s.db.WithContext(ctx).
				Model(&productDomain.ProductAttribute{}).
				Unscoped().
				Where("id = ?", existingAttribute.ID).
				Select("values", "display_order", "deleted_at").
				Updates(updatePayload).Error; err != nil {
				return fmt.Errorf("failed to update attribute %s for product %s: %w", a.name, p.name, err)
			}
		default:
			return fmt.Errorf("failed to check attribute %s for product %s: %w", a.name, p.name, attrErr)
		}
	}

	return nil
}

func (s *seeder) seedProductVariants(ctx context.Context, productID string, p productSeed) (int, int, error) {
	createdCount := 0
	updatedCount := 0

	for _, v := range p.variants {
		var existingVariant productDomain.ProductVariant
		variantErr := s.db.WithContext(ctx).
			Unscoped().
			Where("sku = ?", v.sku).
			First(&existingVariant).Error

		switch {
		case variantErr == gorm.ErrRecordNotFound:
			variant := &productDomain.ProductVariant{
				ProductID:       productID,
				Name:            v.name,
				SKU:             v.sku,
				Price:           v.price,
				StockQuantity:   v.stockQuantity,
				StockReserved:   0,
				IsActive:        true,
				AttributeValues: v.attributeValues,
				Images:          v.images,
			}
			if err := s.db.WithContext(ctx).Create(variant).Error; err != nil {
				return 0, 0, fmt.Errorf("failed to create variant %s for product %s: %w", v.sku, p.name, err)
			}
			createdCount++
			log.Printf("      ✅ Variant created: %s (%s)", v.sku, p.name)
		case variantErr == nil:
			if existingVariant.ProductID != productID {
				return 0, 0, fmt.Errorf("variant SKU %s belongs to another product (product_id=%s)", v.sku, existingVariant.ProductID)
			}
			updatePayload := &productDomain.ProductVariant{
				Name:            v.name,
				Price:           v.price,
				StockQuantity:   v.stockQuantity,
				StockReserved:   0,
				IsActive:        true,
				AttributeValues: v.attributeValues,
				Images:          v.images,
				DeletedAt:       gorm.DeletedAt{},
			}
			updateResult := s.db.WithContext(ctx).
				Model(&productDomain.ProductVariant{}).
				Unscoped().
				Where("id = ?", existingVariant.ID).
				Select("name", "price", "stock_quantity", "stock_reserved", "is_active", "attribute_values", "images", "deleted_at").
				Updates(updatePayload)
			if updateResult.Error != nil {
				return 0, 0, fmt.Errorf("failed to update variant %s for product %s: %w", v.sku, p.name, updateResult.Error)
			}
			updatedCount++
		default:
			return 0, 0, fmt.Errorf("failed to check variant %s for product %s: %w", v.sku, p.name, variantErr)
		}
	}

	return createdCount, updatedCount, nil
}

func (s *seeder) syncProductFromVariants(ctx context.Context, productID string, variants []variantSeed, productName string) error {
	if err := s.db.WithContext(ctx).
		Model(&productDomain.Product{}).
		Where("id = ?", productID).
		Updates(map[string]interface{}{
			"has_variant": true,
			"stock":       totalVariantStock(variants),
		}).Error; err != nil {
		return fmt.Errorf("failed to sync product stock from variants for %s: %w", productName, err)
	}
	return nil
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

	products := defaultProductSeeds()
	for _, p := range products {
		seededProduct, err := s.ensureSeedProduct(ctx, owner.ID, p)
		if err != nil {
			return err
		}

		if err := s.seedProductAttributes(ctx, seededProduct.ID, p); err != nil {
			return err
		}

		variantCreatedCount, variantUpdatedCount, err := s.seedProductVariants(ctx, seededProduct.ID, p)
		if err != nil {
			return err
		}

		if len(p.variants) > 0 {
			if err := s.syncProductFromVariants(ctx, seededProduct.ID, p.variants, p.name); err != nil {
				return err
			}
		}

		log.Println(variantSeedRunMessage(p.name, len(p.variants), variantCreatedCount, variantUpdatedCount))
	}

	// Print summary
	var totalProducts int64
	s.db.WithContext(ctx).Model(&productDomain.Product{}).Where("deleted_at IS NULL").Count(&totalProducts)
	var totalVariants int64
	s.db.WithContext(ctx).Model(&productDomain.ProductVariant{}).Where("deleted_at IS NULL").Count(&totalVariants)
	log.Printf("   📊 Total products in database: %d", totalProducts)
	log.Printf("   📊 Total variants in database: %d", totalVariants)

	return nil
}

func main() {
	// Load .env file
	utils.LoadEnv()

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

	// Validate schema exists (SQL migrations should have already run).
	log.Println("Validating schema...")
	if err := seeder.validateRequiredTables(ctx); err != nil {
		log.Fatalf("Schema validation failed: %v", err)
	}
	log.Println("✅ Schema validation completed!")

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
