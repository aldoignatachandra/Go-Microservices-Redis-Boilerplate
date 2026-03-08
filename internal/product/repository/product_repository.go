// Package repository provides data access for the product service.
package repository

import (
	"context"
	"errors"
	"fmt"

	"github.com/ignata/go-microservices-boilerplate/internal/product/domain"
	"github.com/ignata/go-microservices-boilerplate/internal/product/dto"
	"gorm.io/gorm"
)

// ProductRepository defines the interface for product data access.
type ProductRepository interface {
	// Create creates a new product
	Create(ctx context.Context, product *domain.Product) error
	// Update updates an existing product
	Update(ctx context.Context, product *domain.Product) error
	// Delete soft deletes a product
	Delete(ctx context.Context, id string) error
	// HardDelete permanently deletes a product
	HardDelete(ctx context.Context, id string) error
	// Restore restores a soft-deleted product
	Restore(ctx context.Context, id string) error
	// FindByID finds a product by ID
	FindByID(ctx context.Context, id string, opts *domain.ParanoidOptions) (*domain.Product, error)
	// FindAll finds all products with pagination
	FindAll(ctx context.Context, req *dto.ListProductsRequest) (*domain.ProductList, error)
	// ExistsByName checks if a product exists by name (global)
	ExistsByName(ctx context.Context, name string) (bool, error)
	// ExistsByNameAndOwner checks if a product exists by name for a specific owner
	ExistsByNameAndOwner(ctx context.Context, name string, ownerID string) (bool, error)
	// UpdateStock updates product stock
	UpdateStock(ctx context.Context, id string, stock int) error
}

// gormProductRepository implements ProductRepository using GORM.
type gormProductRepository struct {
	db *gorm.DB
}

// NewProductRepository creates a new product repository.
func NewProductRepository(db *gorm.DB) ProductRepository {
	return &gormProductRepository{db: db}
}

// Create creates a new product.
func (r *gormProductRepository) Create(ctx context.Context, product *domain.Product) error {
	result := r.db.WithContext(ctx).Create(product)
	if result.Error != nil {
		if errors.Is(result.Error, gorm.ErrDuplicatedKey) {
			return domain.ErrProductNameAlreadyUsed
		}
		return fmt.Errorf("failed to create product: %w", result.Error)
	}
	return nil
}

// Update updates an existing product.
func (r *gormProductRepository) Update(ctx context.Context, product *domain.Product) error {
	result := r.db.WithContext(ctx).
		Model(&domain.Product{}).
		Where("id = ?", product.ID).
		Updates(product)
	if result.Error != nil {
		return fmt.Errorf("failed to update product: %w", result.Error)
	}
	if result.RowsAffected == 0 {
		return domain.ErrProductNotFound
	}
	return nil
}

// Delete soft deletes a product.
func (r *gormProductRepository) Delete(ctx context.Context, id string) error {
	result := r.db.WithContext(ctx).Delete(&domain.Product{}, "id = ?", id)
	if result.Error != nil {
		return fmt.Errorf("failed to delete product: %w", result.Error)
	}
	if result.RowsAffected == 0 {
		return domain.ErrProductNotFound
	}
	return nil
}

// HardDelete permanently deletes a product.
func (r *gormProductRepository) HardDelete(ctx context.Context, id string) error {
	result := r.db.WithContext(ctx).Unscoped().Delete(&domain.Product{}, "id = ?", id)
	if result.Error != nil {
		return fmt.Errorf("failed to hard delete product: %w", result.Error)
	}
	if result.RowsAffected == 0 {
		return domain.ErrProductNotFound
	}
	return nil
}

// Restore restores a soft-deleted product.
func (r *gormProductRepository) Restore(ctx context.Context, id string) error {
	result := r.db.WithContext(ctx).
		Model(&domain.Product{}).
		Unscoped().
		Where("id = ? AND deleted_at IS NOT NULL", id).
		Update("deleted_at", nil)

	if result.Error != nil {
		return fmt.Errorf("failed to restore product: %w", result.Error)
	}
	if result.RowsAffected == 0 {
		return domain.ErrProductNotFound
	}
	return nil
}

// FindByID finds a product by ID.
func (r *gormProductRepository) FindByID(ctx context.Context, id string, opts *domain.ParanoidOptions) (*domain.Product, error) {
	if opts == nil {
		opts = domain.DefaultParanoidOptions()
	}

	query := r.db.WithContext(ctx)

	if opts.ShouldIncludeDeleted() {
		query = query.Unscoped()
		if opts.ShouldOnlyDeleted() {
			query = query.Where("deleted_at IS NOT NULL")
		}
	}

	var product domain.Product
	result := query.Where("id = ?", id).First(&product)
	if result.Error != nil {
		if errors.Is(result.Error, gorm.ErrRecordNotFound) {
			return nil, domain.ErrProductNotFound
		}
		return nil, fmt.Errorf("failed to find product: %w", result.Error)
	}

	return &product, nil
}

// FindAll finds all products with pagination.
func (r *gormProductRepository) FindAll(ctx context.Context, req *dto.ListProductsRequest) (*domain.ProductList, error) {
	if req == nil {
		req = &dto.ListProductsRequest{}
	}

	page := req.GetPage()
	limit := req.GetLimit()
	opts := req.GetParanoidOptions()

	query := r.db.WithContext(ctx).Model(&domain.Product{})

	// Apply paranoid options
	if opts.ShouldIncludeDeleted() {
		query = query.Unscoped()
		if opts.ShouldOnlyDeleted() {
			query = query.Where("deleted_at IS NOT NULL")
		}
	}

	// Apply filters
	if req.OwnerID != "" {
		query = query.Where("owner_id = ?", req.OwnerID)
	}
	if req.Search != "" {
		// Use LIKE for SQLite compatibility (case-insensitive for ASCII)
		query = query.Where("name LIKE ?", "%"+req.Search+"%")
	}

	// Count total
	var total int64
	if err := query.Count(&total).Error; err != nil {
		return nil, fmt.Errorf("failed to count products: %w", err)
	}

	// Calculate pagination
	totalPages := int(total) / limit
	if int(total)%limit > 0 {
		totalPages++
	}

	// Get paginated results
	var products []*domain.Product
	offset := (page - 1) * limit
	result := query.Order("created_at DESC").Offset(offset).Limit(limit).Find(&products)
	if result.Error != nil {
		return nil, fmt.Errorf("failed to find products: %w", result.Error)
	}

	return &domain.ProductList{
		Products:   products,
		Total:      total,
		Page:       page,
		Limit:      limit,
		TotalPages: totalPages,
	}, nil
}

// ExistsByName checks if a product exists by name.
func (r *gormProductRepository) ExistsByName(ctx context.Context, name string) (bool, error) {
	var count int64
	result := r.db.WithContext(ctx).
		Model(&domain.Product{}).
		Where("name = ?", name).
		Count(&count)

	if result.Error != nil {
		return false, fmt.Errorf("failed to check product existence: %w", result.Error)
	}

	return count > 0, nil
}

// ExistsByNameAndOwner checks if a product exists by name for a specific owner.
func (r *gormProductRepository) ExistsByNameAndOwner(ctx context.Context, name string, ownerID string) (bool, error) {
	var count int64
	result := r.db.WithContext(ctx).
		Model(&domain.Product{}).
		Where("name = ? AND owner_id = ?", name, ownerID).
		Count(&count)

	if result.Error != nil {
		return false, fmt.Errorf("failed to check product existence: %w", result.Error)
	}

	return count > 0, nil
}

// UpdateStock updates product stock.
func (r *gormProductRepository) UpdateStock(ctx context.Context, id string, stock int) error {
	result := r.db.WithContext(ctx).
		Model(&domain.Product{}).
		Where("id = ?", id).
		Update("stock", stock)

	if result.Error != nil {
		return fmt.Errorf("failed to update product stock: %w", result.Error)
	}
	if result.RowsAffected == 0 {
		return domain.ErrProductNotFound
	}
	return nil
}
