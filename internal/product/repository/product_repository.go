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
	// CreateWithDetails creates a new product plus optional attributes/variants in one transaction.
	CreateWithDetails(
		ctx context.Context,
		product *domain.Product,
		attributes []*domain.ProductAttribute,
		variants []*domain.ProductVariant,
	) error
	// Update updates an existing product
	Update(ctx context.Context, product *domain.Product) error
	// UpdateWithDetails updates product and optionally replaces attributes/variants atomically.
	UpdateWithDetails(
		ctx context.Context,
		product *domain.Product,
		attributes []*domain.ProductAttribute,
		variants []*domain.ProductVariant,
		replaceAttributes bool,
		replaceVariants bool,
	) error
	// Delete soft deletes a product
	Delete(ctx context.Context, id string) error
	// HardDelete permanently deletes a product
	HardDelete(ctx context.Context, id string) error
	// Restore restores a soft-deleted product
	Restore(ctx context.Context, id string) error
	// FindByID finds a product by ID
	FindByID(ctx context.Context, id string, opts *domain.ParanoidOptions) (*domain.Product, error)
	// FindByIDWithDetails finds a product by ID and includes attributes + variants.
	FindByIDWithDetails(
		ctx context.Context,
		id string,
		opts *domain.ParanoidOptions,
	) (*domain.Product, []*domain.ProductVariant, []*domain.ProductAttribute, error)
	// FindAll finds all products with pagination
	FindAll(ctx context.Context, req *dto.ListProductsRequest) (*domain.ProductList, error)
	// ExistsByName checks if a product exists by name (global)
	ExistsByName(ctx context.Context, name string) (bool, error)
	// ExistsByNameAndOwner checks if a product exists by name for a specific owner
	ExistsByNameAndOwner(ctx context.Context, name string, ownerID string) (bool, error)
	// UpdateStock updates product stock
	UpdateStock(ctx context.Context, id string, stock int) error
	// UpdateVariantStockAndSyncProduct updates one variant stock and re-syncs parent product stock.
	UpdateVariantStockAndSyncProduct(
		ctx context.Context,
		productID string,
		variantID string,
		stock int,
	) (int, error)
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

// CreateWithDetails creates a product with attributes and variants atomically.
func (r *gormProductRepository) CreateWithDetails(
	ctx context.Context,
	product *domain.Product,
	attributes []*domain.ProductAttribute,
	variants []*domain.ProductVariant,
) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Create(product).Error; err != nil {
			if errors.Is(err, gorm.ErrDuplicatedKey) {
				return domain.ErrProductNameAlreadyUsed
			}
			return fmt.Errorf("failed to create product: %w", err)
		}

		if len(attributes) > 0 {
			filteredAttributes := make([]*domain.ProductAttribute, 0, len(attributes))
			for _, attr := range attributes {
				if attr == nil {
					continue
				}
				attr.ProductID = product.ID
				filteredAttributes = append(filteredAttributes, attr)
			}
			if len(filteredAttributes) > 0 {
				if err := tx.Create(&filteredAttributes).Error; err != nil {
					return fmt.Errorf("failed to create product attributes: %w", err)
				}
			}
		}

		if len(variants) > 0 {
			filteredVariants := make([]*domain.ProductVariant, 0, len(variants))
			for _, variant := range variants {
				if variant == nil {
					continue
				}
				variant.ProductID = product.ID
				filteredVariants = append(filteredVariants, variant)
			}
			if len(filteredVariants) > 0 {
				if err := tx.Create(&filteredVariants).Error; err != nil {
					return fmt.Errorf("failed to create product variants: %w", err)
				}
			}
		}

		return nil
	})
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

// UpdateWithDetails updates product and optionally replaces attributes/variants atomically.
func (r *gormProductRepository) UpdateWithDetails(
	ctx context.Context,
	product *domain.Product,
	attributes []*domain.ProductAttribute,
	variants []*domain.ProductVariant,
	replaceAttributes bool,
	replaceVariants bool,
) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		productUpdate := tx.Model(&domain.Product{}).
			Where("id = ?", product.ID).
			Updates(map[string]interface{}{
				"name":        product.Name,
				"price":       product.Price,
				"stock":       product.Stock,
				"has_variant": product.HasVariant,
				"images":      product.Images,
			})
		if productUpdate.Error != nil {
			return fmt.Errorf("failed to update product: %w", productUpdate.Error)
		}
		if productUpdate.RowsAffected == 0 {
			return domain.ErrProductNotFound
		}

		if replaceAttributes {
			if err := tx.Unscoped().
				Where("product_id = ?", product.ID).
				Delete(&domain.ProductAttribute{}).Error; err != nil {
				return fmt.Errorf("failed to replace product attributes: %w", err)
			}

			filteredAttributes := make([]*domain.ProductAttribute, 0, len(attributes))
			for _, attr := range attributes {
				if attr == nil {
					continue
				}
				attr.ProductID = product.ID
				filteredAttributes = append(filteredAttributes, attr)
			}
			if len(filteredAttributes) > 0 {
				if err := tx.Create(&filteredAttributes).Error; err != nil {
					return fmt.Errorf("failed to create replacement attributes: %w", err)
				}
			}
		}

		if replaceVariants {
			if err := tx.Unscoped().
				Where("product_id = ?", product.ID).
				Delete(&domain.ProductVariant{}).Error; err != nil {
				return fmt.Errorf("failed to replace product variants: %w", err)
			}

			filteredVariants := make([]*domain.ProductVariant, 0, len(variants))
			for _, variant := range variants {
				if variant == nil {
					continue
				}
				variant.ProductID = product.ID
				filteredVariants = append(filteredVariants, variant)
			}
			if len(filteredVariants) > 0 {
				if err := tx.Create(&filteredVariants).Error; err != nil {
					return fmt.Errorf("failed to create replacement variants: %w", err)
				}
			}
		}

		return nil
	})
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

// FindByIDWithDetails finds a product by ID and loads active attributes + variants.
func (r *gormProductRepository) FindByIDWithDetails(
	ctx context.Context,
	id string,
	opts *domain.ParanoidOptions,
) (*domain.Product, []*domain.ProductVariant, []*domain.ProductAttribute, error) {
	product, err := r.FindByID(ctx, id, opts)
	if err != nil {
		return nil, nil, nil, err
	}

	var attributes []*domain.ProductAttribute
	if err := r.db.WithContext(ctx).
		Where("product_id = ? AND deleted_at IS NULL", product.ID).
		Order("display_order ASC, created_at ASC").
		Find(&attributes).Error; err != nil {
		return nil, nil, nil, fmt.Errorf("failed to find product attributes: %w", err)
	}

	var variants []*domain.ProductVariant
	if err := r.db.WithContext(ctx).
		Where("product_id = ? AND deleted_at IS NULL", product.ID).
		Order("created_at ASC").
		Find(&variants).Error; err != nil {
		return nil, nil, nil, fmt.Errorf("failed to find product variants: %w", err)
	}

	// Keep detail price consistent with list response:
	// when variants exist, expose min/max from variant prices.
	if product.HasVariant && len(variants) > 0 {
		minPrice := product.Price
		maxPrice := product.Price
		hasValidVariantPrice := false
		for _, v := range variants {
			if v == nil || v.Price <= 0 {
				continue
			}
			if !hasValidVariantPrice {
				minPrice = v.Price
				maxPrice = v.Price
				hasValidVariantPrice = true
				continue
			}
			if v.Price < minPrice {
				minPrice = v.Price
			}
			if v.Price > maxPrice {
				maxPrice = v.Price
			}
		}
		if hasValidVariantPrice {
			product.PriceMin = minPrice
			product.PriceMax = maxPrice
		}
	}

	return product, variants, attributes, nil
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

	if err := r.attachVariantPriceRanges(ctx, products); err != nil {
		return nil, err
	}

	return &domain.ProductList{
		Products:   products,
		Total:      total,
		Page:       page,
		Limit:      limit,
		TotalPages: totalPages,
	}, nil
}

func (r *gormProductRepository) attachVariantPriceRanges(ctx context.Context, products []*domain.Product) error {
	// Attach variant-based price ranges for list responses.
	// For products with variants, list API should expose min/max variant price.
	productIDs := make([]string, 0, len(products))
	for _, product := range products {
		if product != nil && product.HasVariant {
			productIDs = append(productIDs, product.ID)
		}
	}

	if len(productIDs) == 0 {
		return nil
	}

	type variantPriceRangeRow struct {
		ProductID string  `gorm:"column:product_id"`
		MinPrice  float64 `gorm:"column:min_price"`
		MaxPrice  float64 `gorm:"column:max_price"`
	}

	var rows []variantPriceRangeRow
	err := r.db.WithContext(ctx).
		Table("product_variants").
		Select("product_id, MIN(price) AS min_price, MAX(price) AS max_price").
		Where("product_id IN ?", productIDs).
		Where("deleted_at IS NULL").
		Group("product_id").
		Scan(&rows).Error
	if err != nil {
		return fmt.Errorf("failed to load variant price ranges: %w", err)
	}

	priceRanges := make(map[string]variantPriceRangeRow, len(rows))
	for _, row := range rows {
		priceRanges[row.ProductID] = row
	}

	for _, product := range products {
		if product == nil || !product.HasVariant {
			continue
		}
		if row, ok := priceRanges[product.ID]; ok {
			product.PriceMin = row.MinPrice
			product.PriceMax = row.MaxPrice
		}
	}

	return nil
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

// UpdateVariantStockAndSyncProduct updates a variant stock and syncs parent product stock in one transaction.
func (r *gormProductRepository) UpdateVariantStockAndSyncProduct(
	ctx context.Context,
	productID string,
	variantID string,
	stock int,
) (int, error) {
	var updatedProductStock int

	err := r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		variantUpdate := tx.Model(&domain.ProductVariant{}).
			Where("id = ? AND product_id = ? AND deleted_at IS NULL", variantID, productID).
			Update("stock_quantity", stock)
		if variantUpdate.Error != nil {
			return fmt.Errorf("failed to update variant stock: %w", variantUpdate.Error)
		}
		if variantUpdate.RowsAffected == 0 {
			return domain.ErrVariantNotInProduct
		}

		type stockRow struct {
			Total int `gorm:"column:total_stock"`
		}
		var row stockRow
		if err := tx.Model(&domain.ProductVariant{}).
			Select("COALESCE(SUM(stock_quantity), 0) AS total_stock").
			Where("product_id = ? AND deleted_at IS NULL", productID).
			Scan(&row).Error; err != nil {
			return fmt.Errorf("failed to calculate product stock from variants: %w", err)
		}

		productUpdate := tx.Model(&domain.Product{}).
			Where("id = ?", productID).
			Update("stock", row.Total)
		if productUpdate.Error != nil {
			return fmt.Errorf("failed to sync product stock: %w", productUpdate.Error)
		}
		if productUpdate.RowsAffected == 0 {
			return domain.ErrProductNotFound
		}

		updatedProductStock = row.Total
		return nil
	})
	if err != nil {
		return 0, err
	}

	return updatedProductStock, nil
}
