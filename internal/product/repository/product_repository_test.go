// Package repository_test provides tests for the product repository.
package repository_test

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/twinj/uuid"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"

	"github.com/ignata/go-microservices-boilerplate/internal/product/domain"
	"github.com/ignata/go-microservices-boilerplate/internal/product/dto"
	"github.com/ignata/go-microservices-boilerplate/internal/product/repository"
)

// setupTestDB creates an in-memory SQLite database for testing.
func setupTestDB(t *testing.T) *gorm.DB {
	t.Helper()

	// Use a unique database for each test to avoid conflicts
	dbName := fmt.Sprintf("file:test_%s.db?mode=memory&cache=shared", uuid.NewV4().String())
	db, err := gorm.Open(sqlite.Open(dbName), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Silent),
	})
	require.NoError(t, err, "Failed to open test database")

	// Migrate the schema
	err = db.AutoMigrate(&domain.Product{}, &domain.ProductVariant{}, &domain.ProductAttribute{})
	require.NoError(t, err, "Failed to migrate test database")

	return db
}

// teardownTestDB closes the database connection.
func teardownTestDB(t *testing.T, db *gorm.DB) {
	t.Helper()

	sqlDB, err := db.DB()
	require.NoError(t, err, "Failed to get sql.DB")
	err = sqlDB.Close()
	require.NoError(t, err, "Failed to close test database")
}

// createTestProduct creates a test product with a unique name.
func createTestProduct(t *testing.T, db *gorm.DB) *domain.Product {
	t.Helper()
	product := &domain.Product{
		Model: domain.Model{
			ID:        uuid.NewV4().String(),
			CreatedAt: time.Now().UTC(),
			UpdatedAt: time.Now().UTC(),
		},
		Name:       fmt.Sprintf("Product_%s", uuid.NewV4().String()),
		Price:      29.99,
		Stock:      100,
		OwnerID:    uuid.NewV4().String(),
		HasVariant: false,
		Images:     "",
	}
	err := db.Create(product).Error
	require.NoError(t, err)
	return product
}

// TestCreate tests the Create method.
func TestCreate(t *testing.T) {
	db := setupTestDB(t)
	defer teardownTestDB(t, db)

	repo := repository.NewProductRepository(db)
	ctx := context.Background()

	t.Run("successful create product", func(t *testing.T) {
		ownerID := uuid.NewV4().String()
		product := &domain.Product{
			Name:       fmt.Sprintf("Product_%s", uuid.NewV4().String()),
			Price:      19.99,
			Stock:      50,
			OwnerID:    ownerID,
			HasVariant: false,
			Images:     "",
		}

		err := repo.Create(ctx, product)
		require.NoError(t, err)
		assert.NotEmpty(t, product.ID)

		// Verify product was created
		var found domain.Product
		err = db.Where("id = ?", product.ID).First(&found).Error
		require.NoError(t, err)
		assert.Equal(t, product.Name, found.Name)
	})

	t.Run("successful create product with all fields", func(t *testing.T) {
		ownerID := uuid.NewV4().String()
		product := &domain.Product{
			Name:       fmt.Sprintf("Full Product_%s", uuid.NewV4().String()),
			Price:      99.99,
			Stock:      200,
			OwnerID:    ownerID,
			HasVariant: true,
			Images:     "http://example.com/image.jpg",
		}

		err := repo.Create(ctx, product)
		require.NoError(t, err)
	})

	t.Run("successful create product with has_variant", func(t *testing.T) {
		ownerID := uuid.NewV4().String()
		product := &domain.Product{
			Name:       fmt.Sprintf("Variant Product_%s", uuid.NewV4().String()),
			Price:      49.99,
			Stock:      30,
			OwnerID:    ownerID,
			HasVariant: true,
		}

		err := repo.Create(ctx, product)
		require.NoError(t, err)
		assert.True(t, product.HasVariant)
	})

	t.Run("successful create product with zero stock", func(t *testing.T) {
		ownerID := uuid.NewV4().String()
		product := &domain.Product{
			Name:       fmt.Sprintf("Zero Stock Product_%s", uuid.NewV4().String()),
			Price:      29.99,
			Stock:      0,
			OwnerID:    ownerID,
			HasVariant: false,
		}

		err := repo.Create(ctx, product)
		require.NoError(t, err)
		assert.Equal(t, 0, product.Stock)
	})
}

// TestUpdate tests the Update method.
func TestUpdate(t *testing.T) {
	db := setupTestDB(t)
	defer teardownTestDB(t, db)

	repo := repository.NewProductRepository(db)
	ctx := context.Background()

	t.Run("successful update product name", func(t *testing.T) {
		product := createTestProduct(t, db)
		originalName := product.Name
		product.Name = fmt.Sprintf("Updated Product_%s", uuid.NewV4().String())

		err := repo.Update(ctx, product)
		require.NoError(t, err)

		var found domain.Product
		err = db.Where("id = ?", product.ID).First(&found).Error
		require.NoError(t, err)
		assert.NotEqual(t, originalName, found.Name)
		assert.Equal(t, product.Name, found.Name)
	})

	t.Run("successful update product price", func(t *testing.T) {
		product := createTestProduct(t, db)
		product.Price = 39.99

		err := repo.Update(ctx, product)
		require.NoError(t, err)

		var found domain.Product
		err = db.Where("id = ?", product.ID).First(&found).Error
		require.NoError(t, err)
		assert.Equal(t, 39.99, found.Price)
	})

	t.Run("successful update product stock", func(t *testing.T) {
		product := createTestProduct(t, db)
		product.Stock = 150

		err := repo.Update(ctx, product)
		require.NoError(t, err)

		var found domain.Product
		err = db.Where("id = ?", product.ID).First(&found).Error
		require.NoError(t, err)
		assert.Equal(t, 150, found.Stock)
	})

	t.Run("successful update product status", func(t *testing.T) {
		product := createTestProduct(t, db)
		product.HasVariant = true

		err := repo.Update(ctx, product)
		require.NoError(t, err)

		var found domain.Product
		err = db.Where("id = ?", product.ID).First(&found).Error
		require.NoError(t, err)
		assert.True(t, found.HasVariant)
	})

	t.Run("successful update multiple fields", func(t *testing.T) {
		product := createTestProduct(t, db)
		product.Price = 59.99
		product.Stock = 75
		product.HasVariant = true
		product.Images = "http://example.com/new-image.jpg"

		err := repo.Update(ctx, product)
		require.NoError(t, err)

		var found domain.Product
		err = db.Where("id = ?", product.ID).First(&found).Error
		require.NoError(t, err)
		assert.Equal(t, 59.99, found.Price)
		assert.Equal(t, 75, found.Stock)
		assert.True(t, found.HasVariant)
	})

	t.Run("fail - product not found", func(t *testing.T) {
		ownerID := uuid.NewV4().String()
		product := &domain.Product{
			Model:      domain.Model{ID: uuid.NewV4().String()},
			Name:       "Non-existent Product",
			Price:      19.99,
			Stock:      50,
			OwnerID:    ownerID,
			HasVariant: false,
		}

		err := repo.Update(ctx, product)
		assert.Error(t, err)
		assert.ErrorIs(t, err, domain.ErrProductNotFound)
	})
}

// TestDelete tests the Delete (soft delete) method.
func TestDelete(t *testing.T) {
	db := setupTestDB(t)
	defer teardownTestDB(t, db)

	repo := repository.NewProductRepository(db)
	ctx := context.Background()

	t.Run("successful soft delete product", func(t *testing.T) {
		product := createTestProduct(t, db)

		err := repo.Delete(ctx, product.ID)
		require.NoError(t, err)

		// Product should not be found in normal queries
		var found domain.Product
		err = db.Where("id = ?", product.ID).First(&found).Error
		assert.Error(t, err)
		assert.Equal(t, gorm.ErrRecordNotFound, err)

		// Product should still exist in database (soft delete)
		var deleted domain.Product
		err = db.Unscoped().Where("id = ?", product.ID).First(&deleted).Error
		require.NoError(t, err)
		assert.NotNil(t, deleted.DeletedAt)
		assert.True(t, deleted.DeletedAt.Valid)
	})

	t.Run("fail - product not found", func(t *testing.T) {
		err := repo.Delete(ctx, uuid.NewV4().String())
		assert.Error(t, err)
		assert.ErrorIs(t, err, domain.ErrProductNotFound)
	})

	t.Run("fail - already deleted product", func(t *testing.T) {
		product := createTestProduct(t, db)
		err := db.Delete(product).Error
		require.NoError(t, err)

		err = repo.Delete(ctx, product.ID)
		assert.Error(t, err)
		assert.ErrorIs(t, err, domain.ErrProductNotFound)
	})
}

// TestHardDelete tests the HardDelete method.
func TestHardDelete(t *testing.T) {
	db := setupTestDB(t)
	defer teardownTestDB(t, db)

	repo := repository.NewProductRepository(db)
	ctx := context.Background()

	t.Run("successful hard delete product", func(t *testing.T) {
		product := createTestProduct(t, db)

		err := repo.HardDelete(ctx, product.ID)
		require.NoError(t, err)

		// Product should not exist at all (hard delete)
		var deleted domain.Product
		err = db.Unscoped().Where("id = ?", product.ID).First(&deleted).Error
		assert.Error(t, err)
		assert.Equal(t, gorm.ErrRecordNotFound, err)
	})

	t.Run("successful hard delete soft-deleted product", func(t *testing.T) {
		product := createTestProduct(t, db)
		err := db.Delete(product).Error
		require.NoError(t, err)

		err = repo.HardDelete(ctx, product.ID)
		require.NoError(t, err)

		// Product should not exist at all
		var deleted domain.Product
		err = db.Unscoped().Where("id = ?", product.ID).First(&deleted).Error
		assert.Error(t, err)
	})

	t.Run("fail - product not found", func(t *testing.T) {
		err := repo.HardDelete(ctx, uuid.NewV4().String())
		assert.Error(t, err)
		assert.ErrorIs(t, err, domain.ErrProductNotFound)
	})
}

// TestRestore tests the Restore method.
func TestRestore(t *testing.T) {
	db := setupTestDB(t)
	defer teardownTestDB(t, db)

	repo := repository.NewProductRepository(db)
	ctx := context.Background()

	t.Run("successful restore soft-deleted product", func(t *testing.T) {
		product := createTestProduct(t, db)
		err := db.Delete(product).Error
		require.NoError(t, err)

		err = repo.Restore(ctx, product.ID)
		require.NoError(t, err)

		// Product should now be findable in normal queries
		var found domain.Product
		err = db.Where("id = ?", product.ID).First(&found).Error
		require.NoError(t, err)
		assert.False(t, found.DeletedAt.Valid)
	})

	t.Run("fail - product not found", func(t *testing.T) {
		err := repo.Restore(ctx, uuid.NewV4().String())
		assert.Error(t, err)
		assert.ErrorIs(t, err, domain.ErrProductNotFound)
	})

	t.Run("fail - restore active product", func(t *testing.T) {
		product := createTestProduct(t, db)

		err := repo.Restore(ctx, product.ID)
		assert.Error(t, err)
		assert.ErrorIs(t, err, domain.ErrProductNotFound)
	})
}

// TestFindByID tests the FindByID method.
func TestFindByID(t *testing.T) {
	db := setupTestDB(t)
	defer teardownTestDB(t, db)

	repo := repository.NewProductRepository(db)
	ctx := context.Background()

	t.Run("successful find active product by ID", func(t *testing.T) {
		product := createTestProduct(t, db)

		found, err := repo.FindByID(ctx, product.ID, domain.DefaultParanoidOptions())
		require.NoError(t, err)
		assert.Equal(t, product.ID, found.ID)
		assert.Equal(t, product.Name, found.Name)
		assert.Equal(t, product.Price, found.Price)
	})

	t.Run("successful find variant product", func(t *testing.T) {
		product := createTestProduct(t, db)
		product.HasVariant = true
		err := db.Save(product).Error
		require.NoError(t, err)

		found, err := repo.FindByID(ctx, product.ID, domain.DefaultParanoidOptions())
		require.NoError(t, err)
		assert.True(t, found.HasVariant)
	})

	t.Run("successful find deleted product with include deleted", func(t *testing.T) {
		product := createTestProduct(t, db)
		err := db.Delete(product).Error
		require.NoError(t, err)

		found, err := repo.FindByID(ctx, product.ID, &domain.ParanoidOptions{IncludeDeleted: true})
		require.NoError(t, err)
		assert.Equal(t, product.ID, found.ID)
	})

	t.Run("fail - find deleted product without include deleted", func(t *testing.T) {
		product := createTestProduct(t, db)
		err := db.Delete(product).Error
		require.NoError(t, err)

		_, err = repo.FindByID(ctx, product.ID, domain.DefaultParanoidOptions())
		assert.Error(t, err)
		assert.ErrorIs(t, err, domain.ErrProductNotFound)
	})

	t.Run("fail - product not found", func(t *testing.T) {
		_, err := repo.FindByID(ctx, uuid.NewV4().String(), domain.DefaultParanoidOptions())
		assert.Error(t, err)
		assert.ErrorIs(t, err, domain.ErrProductNotFound)
	})

	t.Run("successful find with nil options", func(t *testing.T) {
		product := createTestProduct(t, db)

		found, err := repo.FindByID(ctx, product.ID, nil)
		require.NoError(t, err)
		assert.Equal(t, product.ID, found.ID)
	})
}

// TestFindAll tests the FindAll method.
func TestFindAll(t *testing.T) {
	db := setupTestDB(t)
	defer teardownTestDB(t, db)

	repo := repository.NewProductRepository(db)
	ctx := context.Background()

	// Setup test data
	for range 5 {
		_ = createTestProduct(t, db)
	}

	// Create a soft-deleted product
	deletedProduct := createTestProduct(t, db)
	err := db.Delete(deletedProduct).Error
	require.NoError(t, err)

	t.Run("successful find all products - first page", func(t *testing.T) {
		result, err := repo.FindAll(ctx, &dto.ListProductsRequest{Page: 1, Limit: 2})
		require.NoError(t, err)
		assert.Len(t, result.Products, 2)
		assert.GreaterOrEqual(t, result.Total, int64(5))
		assert.Equal(t, 1, result.Page)
		assert.Equal(t, 2, result.Limit)
	})

	t.Run("successful find all products - second page", func(t *testing.T) {
		result, err := repo.FindAll(ctx, &dto.ListProductsRequest{Page: 2, Limit: 2})
		require.NoError(t, err)
		assert.Len(t, result.Products, 2)
		assert.Equal(t, 2, result.Page)
	})

	t.Run("successful find all products with include deleted", func(t *testing.T) {
		result, err := repo.FindAll(ctx, &dto.ListProductsRequest{Page: 1, Limit: 10, IncludeDeleted: true})
		require.NoError(t, err)
		assert.Greater(t, result.Total, int64(5))
	})

	t.Run("successful find all products with only deleted", func(t *testing.T) {
		result, err := repo.FindAll(ctx, &dto.ListProductsRequest{Page: 1, Limit: 10, OnlyDeleted: true})
		require.NoError(t, err)
		assert.Equal(t, int64(1), result.Total)
		if len(result.Products) > 0 {
			assert.NotNil(t, result.Products[0].DeletedAt)
			assert.True(t, result.Products[0].DeletedAt.Valid)
		}
	})

	t.Run("successful find all with nil request", func(t *testing.T) {
		result, err := repo.FindAll(ctx, nil)
		require.NoError(t, err)
		assert.NotNil(t, result)
		assert.GreaterOrEqual(t, result.Total, int64(5))
	})

	t.Run("successful find all with pagination defaults", func(t *testing.T) {
		result, err := repo.FindAll(ctx, &dto.ListProductsRequest{Page: 0, Limit: 0})
		require.NoError(t, err)
		assert.Equal(t, 1, result.Page)
		assert.Equal(t, 10, result.Limit)
	})

	t.Run("successful find all with empty result", func(t *testing.T) {
		result, err := repo.FindAll(ctx, &dto.ListProductsRequest{Page: 1, Limit: 10, Search: "nonexistentproduct123456"})
		require.NoError(t, err)
		assert.Equal(t, int64(0), result.Total)
		assert.Empty(t, result.Products)
	})

	t.Run("find all attaches variant min/max price range", func(t *testing.T) {
		product := createTestProduct(t, db)
		product.HasVariant = true
		product.Price = 89.99
		require.NoError(t, db.Save(product).Error)

		variants := []*domain.ProductVariant{
			{
				ID:            uuid.NewV4().String(),
				ProductID:     product.ID,
				Name:          "TKL / Brown Switch",
				SKU:           "KEY-TKL-BROWN",
				Price:         89.99,
				StockQuantity: 20,
				IsActive:      true,
			},
			{
				ID:            uuid.NewV4().String(),
				ProductID:     product.ID,
				Name:          "Full / Red Switch",
				SKU:           "KEY-FULL-RED",
				Price:         99.99,
				StockQuantity: 15,
				IsActive:      true,
			},
		}
		require.NoError(t, db.Create(&variants).Error)

		result, err := repo.FindAll(ctx, &dto.ListProductsRequest{
			Page:   1,
			Limit:  20,
			Search: product.Name,
		})
		require.NoError(t, err)
		require.NotEmpty(t, result.Products)

		var found *domain.Product
		for _, p := range result.Products {
			if p.ID == product.ID {
				found = p
				break
			}
		}
		require.NotNil(t, found)
		assert.True(t, found.HasVariant)
		assert.Equal(t, 89.99, found.PriceMin)
		assert.Equal(t, 99.99, found.PriceMax)
	})
}

func TestFindByIDWithDetails(t *testing.T) {
	db := setupTestDB(t)
	defer teardownTestDB(t, db)

	repo := repository.NewProductRepository(db)
	ctx := context.Background()

	t.Run("returns product with variants, attributes, and variant price range", func(t *testing.T) {
		product := createTestProduct(t, db)
		product.HasVariant = true
		product.Price = 89.99
		require.NoError(t, db.Save(product).Error)

		attributes := []*domain.ProductAttribute{
			{
				ID:           uuid.NewV4().String(),
				ProductID:    product.ID,
				Name:         "Layout",
				Values:       []string{"TKL", "Full"},
				DisplayOrder: 0,
			},
		}
		require.NoError(t, db.Create(&attributes).Error)

		variants := []*domain.ProductVariant{
			{
				ID:            uuid.NewV4().String(),
				ProductID:     product.ID,
				Name:          "TKL / Brown Switch",
				SKU:           "KEY-TKL-BROWN",
				Price:         89.99,
				StockQuantity: 20,
				IsActive:      true,
			},
			{
				ID:            uuid.NewV4().String(),
				ProductID:     product.ID,
				Name:          "Full / Red Switch",
				SKU:           "KEY-FULL-RED",
				Price:         99.99,
				StockQuantity: 15,
				IsActive:      true,
			},
		}
		require.NoError(t, db.Create(&variants).Error)

		found, foundVariants, foundAttributes, err := repo.FindByIDWithDetails(
			ctx,
			product.ID,
			domain.DefaultParanoidOptions(),
		)
		require.NoError(t, err)
		require.NotNil(t, found)
		assert.Equal(t, product.ID, found.ID)
		assert.Equal(t, 89.99, found.PriceMin)
		assert.Equal(t, 99.99, found.PriceMax)
		assert.Len(t, foundVariants, 2)
		assert.Len(t, foundAttributes, 1)
	})
}

// TestExistsByName tests the ExistsByName method.
func TestExistsByName(t *testing.T) {
	db := setupTestDB(t)
	defer teardownTestDB(t, db)

	repo := repository.NewProductRepository(db)
	ctx := context.Background()

	t.Run("product name exists", func(t *testing.T) {
		product := createTestProduct(t, db)

		exists, err := repo.ExistsByName(ctx, product.Name)
		require.NoError(t, err)
		assert.True(t, exists)
	})

	t.Run("product name does not exist", func(t *testing.T) {
		exists, err := repo.ExistsByName(ctx, "Non-existent Product")
		require.NoError(t, err)
		assert.False(t, exists)
	})

	t.Run("soft deleted product should not exist", func(t *testing.T) {
		product := createTestProduct(t, db)
		err := db.Delete(product).Error
		require.NoError(t, err)

		exists, err := repo.ExistsByName(ctx, product.Name)
		require.NoError(t, err)
		assert.False(t, exists)
	})

	t.Run("case sensitive product name check", func(t *testing.T) {
		product := createTestProduct(t, db)

		// Change first character to uppercase to test case sensitivity
		runes := []rune(product.Name)
		if len(runes) > 0 {
			runes[0] = runes[0] - 32 // Convert to uppercase (ASCII only)
		}
		upperName := string(runes)

		exists, err := repo.ExistsByName(ctx, upperName)
		require.NoError(t, err)
		assert.False(t, exists)
	})
}

// TestUpdateStock tests the UpdateStock method.
func TestUpdateStock(t *testing.T) {
	db := setupTestDB(t)
	defer teardownTestDB(t, db)

	repo := repository.NewProductRepository(db)
	ctx := context.Background()

	t.Run("successful update stock to higher value", func(t *testing.T) {
		product := createTestProduct(t, db)

		err := repo.UpdateStock(ctx, product.ID, 150)
		require.NoError(t, err)

		var found domain.Product
		err = db.Where("id = ?", product.ID).First(&found).Error
		require.NoError(t, err)
		assert.Equal(t, 150, found.Stock)
	})

	t.Run("successful update stock to lower value", func(t *testing.T) {
		product := createTestProduct(t, db)

		err := repo.UpdateStock(ctx, product.ID, 25)
		require.NoError(t, err)

		var found domain.Product
		err = db.Where("id = ?", product.ID).First(&found).Error
		require.NoError(t, err)
		assert.Equal(t, 25, found.Stock)
	})

	t.Run("successful update stock to zero", func(t *testing.T) {
		product := createTestProduct(t, db)

		err := repo.UpdateStock(ctx, product.ID, 0)
		require.NoError(t, err)

		var found domain.Product
		err = db.Where("id = ?", product.ID).First(&found).Error
		require.NoError(t, err)
		assert.Equal(t, 0, found.Stock)
	})

	t.Run("successful update stock to very high value", func(t *testing.T) {
		product := createTestProduct(t, db)

		err := repo.UpdateStock(ctx, product.ID, 10000)
		require.NoError(t, err)

		var found domain.Product
		err = db.Where("id = ?", product.ID).First(&found).Error
		require.NoError(t, err)
		assert.Equal(t, 10000, found.Stock)
	})

	t.Run("fail - product not found", func(t *testing.T) {
		err := repo.UpdateStock(ctx, uuid.NewV4().String(), 50)
		assert.Error(t, err)
		assert.ErrorIs(t, err, domain.ErrProductNotFound)
	})
}

// TestProductRepositoryIntegration tests the repository with multiple operations.
func TestProductRepositoryIntegration(t *testing.T) {
	db := setupTestDB(t)
	defer teardownTestDB(t, db)

	repo := repository.NewProductRepository(db)
	ctx := context.Background()

	t.Run("full lifecycle: create, find, update, delete, restore", func(t *testing.T) {
		// Create
		product := createTestProduct(t, db)

		// Find by ID
		found, err := repo.FindByID(ctx, product.ID, domain.DefaultParanoidOptions())
		require.NoError(t, err)
		assert.Equal(t, product.Name, found.Name)

		// Update
		product.Price = 59.99
		err = repo.Update(ctx, product)
		require.NoError(t, err)

		// Verify update
		updated, err := repo.FindByID(ctx, product.ID, domain.DefaultParanoidOptions())
		require.NoError(t, err)
		assert.Equal(t, 59.99, updated.Price)

		// Update stock
		err = repo.UpdateStock(ctx, product.ID, 150)
		require.NoError(t, err)

		// Verify stock update
		stockUpdated, err := repo.FindByID(ctx, product.ID, domain.DefaultParanoidOptions())
		require.NoError(t, err)
		assert.Equal(t, 150, stockUpdated.Stock)

		// Delete (soft)
		err = repo.Delete(ctx, product.ID)
		require.NoError(t, err)

		// Verify soft delete
		_, err = repo.FindByID(ctx, product.ID, domain.DefaultParanoidOptions())
		assert.Error(t, err)
		assert.ErrorIs(t, err, domain.ErrProductNotFound)

		// Restore
		err = repo.Restore(ctx, product.ID)
		require.NoError(t, err)

		// Verify restore
		restored, err := repo.FindByID(ctx, product.ID, domain.DefaultParanoidOptions())
		require.NoError(t, err)
		assert.Equal(t, product.ID, restored.ID)

		// Hard delete
		err = repo.HardDelete(ctx, product.ID)
		require.NoError(t, err)

		// Verify hard delete
		_, err = repo.FindByID(ctx, product.ID, &domain.ParanoidOptions{IncludeDeleted: true})
		assert.Error(t, err)
	})

	t.Run("exists by name after operations", func(t *testing.T) {
		// Initial state - should not exist
		productName := fmt.Sprintf("Lifecycle Product_%s", uuid.NewV4().String())
		exists, err := repo.ExistsByName(ctx, productName)
		require.NoError(t, err)
		assert.False(t, exists)

		// Create product
		ownerID := uuid.NewV4().String()
		product := &domain.Product{
			Name:       productName,
			Price:      29.99,
			Stock:      75,
			OwnerID:    ownerID,
			HasVariant: false,
			Images:     "",
		}
		err = repo.Create(ctx, product)
		require.NoError(t, err)

		// Should exist now
		exists, err = repo.ExistsByName(ctx, productName)
		require.NoError(t, err)
		assert.True(t, exists)

		// Soft delete
		err = repo.Delete(ctx, product.ID)
		require.NoError(t, err)

		// Should not exist after soft delete
		exists, err = repo.ExistsByName(ctx, productName)
		require.NoError(t, err)
		assert.False(t, exists)
	})
}

// TestEdgeCases tests edge cases and error conditions.
func TestEdgeCases(t *testing.T) {
	db := setupTestDB(t)
	defer teardownTestDB(t, db)

	repo := repository.NewProductRepository(db)
	ctx := context.Background()

	t.Run("create product with empty ID", func(t *testing.T) {
		ownerID := uuid.NewV4().String()
		product := &domain.Product{
			Name:       fmt.Sprintf("Empty ID Product_%s", uuid.NewV4().String()),
			Price:      19.99,
			Stock:      50,
			OwnerID:    ownerID,
			HasVariant: false,
		}

		err := repo.Create(ctx, product)
		require.NoError(t, err)
		assert.NotEmpty(t, product.ID)
	})

	t.Run("update non-existent product", func(t *testing.T) {
		ownerID := uuid.NewV4().String()
		product := &domain.Product{
			Model:      domain.Model{ID: uuid.NewV4().String()},
			Name:       "Non-existent Product",
			Price:      19.99,
			Stock:      50,
			OwnerID:    ownerID,
			HasVariant: false,
		}

		err := repo.Update(ctx, product)
		assert.Error(t, err)
		assert.ErrorIs(t, err, domain.ErrProductNotFound)
	})

	t.Run("delete non-existent product", func(t *testing.T) {
		err := repo.Delete(ctx, uuid.NewV4().String())
		assert.Error(t, err)
		assert.ErrorIs(t, err, domain.ErrProductNotFound)
	})

	t.Run("restore non-existent product", func(t *testing.T) {
		err := repo.Restore(ctx, uuid.NewV4().String())
		assert.Error(t, err)
		assert.ErrorIs(t, err, domain.ErrProductNotFound)
	})

	t.Run("hard delete non-existent product", func(t *testing.T) {
		err := repo.HardDelete(ctx, uuid.NewV4().String())
		assert.Error(t, err)
		assert.ErrorIs(t, err, domain.ErrProductNotFound)
	})

	t.Run("update stock non-existent product", func(t *testing.T) {
		err := repo.UpdateStock(ctx, uuid.NewV4().String(), 100)
		assert.Error(t, err)
		assert.ErrorIs(t, err, domain.ErrProductNotFound)
	})

	t.Run("FindAll with large page number", func(t *testing.T) {
		result, err := repo.FindAll(ctx, &dto.ListProductsRequest{Page: 999, Limit: 10})
		require.NoError(t, err)
		// Products slice should be empty since we're beyond available pages
		assert.Empty(t, result.Products)
		// Total may be > 0 due to other tests in same parent test creating products
		assert.GreaterOrEqual(t, result.Total, int64(0))
	})
}

// TestCreate_DatabaseError tests creating product with database error.
func TestCreate_DatabaseError(t *testing.T) {
	db := setupTestDB(t)
	defer teardownTestDB(t, db)

	repo := repository.NewProductRepository(db)
	ctx := context.Background()

	// Close the database to simulate an error
	sqlDB, _ := db.DB()
	sqlDB.Close()

	ownerID := uuid.NewV4().String()
	product := &domain.Product{
		Model:      domain.Model{ID: uuid.NewV4().String()},
		Name:       "Test Product",
		Price:      19.99,
		Stock:      50,
		OwnerID:    ownerID,
		HasVariant: false,
	}

	err := repo.Create(ctx, product)
	assert.Error(t, err)
}

// TestUpdate_DatabaseError tests updating product with database error.
func TestUpdate_DatabaseError(t *testing.T) {
	db := setupTestDB(t)
	defer teardownTestDB(t, db)

	repo := repository.NewProductRepository(db)
	ctx := context.Background()

	product := createTestProduct(t, db)

	// Close the database to simulate an error
	sqlDB, _ := db.DB()
	sqlDB.Close()

	product.Name = "Updated Product"
	err := repo.Update(ctx, product)
	assert.Error(t, err)
}

// TestDelete_DatabaseError tests deleting product with database error.
func TestDelete_DatabaseError(t *testing.T) {
	db := setupTestDB(t)
	defer teardownTestDB(t, db)

	repo := repository.NewProductRepository(db)
	ctx := context.Background()

	product := createTestProduct(t, db)

	// Close the database to simulate an error
	sqlDB, _ := db.DB()
	sqlDB.Close()

	err := repo.Delete(ctx, product.ID)
	assert.Error(t, err)
}

// TestHardDelete_DatabaseError tests hard deleting with database error.
func TestHardDelete_DatabaseError(t *testing.T) {
	db := setupTestDB(t)
	defer teardownTestDB(t, db)

	repo := repository.NewProductRepository(db)
	ctx := context.Background()

	product := createTestProduct(t, db)

	// Close the database to simulate an error
	sqlDB, _ := db.DB()
	sqlDB.Close()

	err := repo.HardDelete(ctx, product.ID)
	assert.Error(t, err)
}

// TestRestore_DatabaseError tests restoring with database error.
func TestRestore_DatabaseError(t *testing.T) {
	db := setupTestDB(t)
	defer teardownTestDB(t, db)

	repo := repository.NewProductRepository(db)
	ctx := context.Background()

	product := createTestProduct(t, db)
	_ = repo.Delete(ctx, product.ID)

	// Close the database to simulate an error
	sqlDB, _ := db.DB()
	sqlDB.Close()

	err := repo.Restore(ctx, product.ID)
	assert.Error(t, err)
}

// TestExistsByName_DatabaseError tests checking existence with database error.
func TestExistsByName_DatabaseError(t *testing.T) {
	db := setupTestDB(t)
	defer teardownTestDB(t, db)

	repo := repository.NewProductRepository(db)
	ctx := context.Background()

	// Close the database to simulate an error
	sqlDB, _ := db.DB()
	sqlDB.Close()

	_, err := repo.ExistsByName(ctx, "Test Product")
	assert.Error(t, err)
}

// TestUpdateStock_DatabaseError tests updating stock with database error.
func TestUpdateStock_DatabaseError(t *testing.T) {
	db := setupTestDB(t)
	defer teardownTestDB(t, db)

	repo := repository.NewProductRepository(db)
	ctx := context.Background()

	product := createTestProduct(t, db)

	// Close the database to simulate an error
	sqlDB, _ := db.DB()
	sqlDB.Close()

	err := repo.UpdateStock(ctx, product.ID, 50)
	assert.Error(t, err)
}
