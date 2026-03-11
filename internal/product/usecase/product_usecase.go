// Package usecase provides business logic for the product service.
package usecase

import (
	"context"
	"errors"
	"fmt"

	"github.com/google/uuid"
	"go.uber.org/zap"

	"github.com/ignata/go-microservices-boilerplate/internal/common/constants"
	"github.com/ignata/go-microservices-boilerplate/internal/product/domain"
	"github.com/ignata/go-microservices-boilerplate/internal/product/dto"
	"github.com/ignata/go-microservices-boilerplate/internal/product/repository"
	"github.com/ignata/go-microservices-boilerplate/pkg/eventbus"
	"github.com/ignata/go-microservices-boilerplate/pkg/utils"
)

// ProductUseCase defines the interface for product business logic.
type ProductUseCase interface {
	// CreateProduct creates a new product
	CreateProduct(ctx context.Context, ownerID string, req *dto.CreateProductRequest) (*dto.ProductResponse, error)
	// GetProduct gets a product by ID
	GetProduct(ctx context.Context, userID, userRole string, req *dto.GetProductRequest) (*dto.ProductResponse, error)
	// ListProducts lists products with pagination
	ListProducts(ctx context.Context, userID, userRole string, req *dto.ListProductsRequest) (*dto.ProductListResponse, error)
	// UpdateProduct updates a product
	UpdateProduct(ctx context.Context, userID, userRole string, productID string, req *dto.UpdateProductRequest) (*dto.ProductResponse, error)
	// DeleteProduct deletes a product
	DeleteProduct(ctx context.Context, userID, userRole string, req *dto.DeleteProductRequest) (*dto.DeleteResponse, error)
	// RestoreProduct restores a deleted product
	RestoreProduct(ctx context.Context, userID, userRole string, req *dto.RestoreProductRequest) (*dto.ProductResponse, error)
	// UpdateStock updates product stock
	UpdateStock(ctx context.Context, userID, userRole string, req *dto.UpdateStockRequest) (*dto.UpdateStockResponse, error)
}

// Config holds usecase configuration.
type Config struct {
	ServiceName string
}

// productUseCase implements ProductUseCase.
type productUseCase struct {
	productRepo repository.ProductRepository
	eventBus    eventbus.EventPublisher
	config      Config
	logger      *zap.Logger
}

// NewProductUseCase creates a new product usecase.
func NewProductUseCase(
	productRepo repository.ProductRepository,
	eventBus eventbus.EventPublisher,
	config Config,
	logger *zap.Logger,
) ProductUseCase {
	return &productUseCase{
		productRepo: productRepo,
		eventBus:    eventBus,
		config:      config,
		logger:      logger,
	}
}

// CreateProduct creates a new product.
func (uc *productUseCase) CreateProduct(ctx context.Context, ownerID string, req *dto.CreateProductRequest) (*dto.ProductResponse, error) {
	if err := uc.ensureProductNameAvailable(ctx, ownerID, req.Name); err != nil {
		return nil, err
	}

	attributes := buildAttributes(req.Attributes)
	variants, totalVariantStock := buildVariants(req.Variants, req.Price)
	hasVariant := len(variants) > 0

	product := &domain.Product{
		Name:       req.Name,
		Price:      req.Price,
		Stock:      req.Stock,
		OwnerID:    ownerID,
		HasVariant: hasVariant,
		Images:     req.Images,
	}

	// Keep product stock consistent with variant stocks.
	// When product has variants, root stock is derived from variant stock sum.
	if hasVariant {
		product.Stock = totalVariantStock
	}

	if err := uc.productRepo.CreateWithDetails(ctx, product, attributes, variants); err != nil {
		return nil, fmt.Errorf("failed to create product: %w", err)
	}

	product, variants, attributes, err := uc.productRepo.FindByIDWithDetails(ctx, product.ID, domain.DefaultParanoidOptions())
	if err != nil {
		return nil, fmt.Errorf("failed to load created product details: %w", err)
	}

	// Publish event
	if uc.eventBus != nil {
		uc.publishEvent(ctx, domain.NewProductCreatedEvent(product))
	}

	return dto.FromProductWithVariants(product, variants, attributes), nil
}

// ErrAccessDenied is returned when a user tries to access a resource they don't own.
var ErrAccessDenied = errors.New("access denied: you do not have permission to perform this action")

// GetProduct gets a product by ID.
func (uc *productUseCase) GetProduct(ctx context.Context, userID, userRole string, req *dto.GetProductRequest) (*dto.ProductResponse, error) {
	product, variants, attributes, err := uc.productRepo.FindByIDWithDetails(ctx, req.ID, req.GetParanoidOptions())
	if err != nil {
		return nil, err
	}

	// IDOR PROTECTION: Non-admin users can only access their own products
	if userRole != constants.RoleAdmin && product.OwnerID != userID {
		return nil, ErrAccessDenied
	}

	return dto.FromProductWithVariants(product, variants, attributes), nil
}

// ListProducts lists products with pagination.
func (uc *productUseCase) ListProducts(ctx context.Context, userID, userRole string, req *dto.ListProductsRequest) (*dto.ProductListResponse, error) {
	// For non-admin users, always filter by their own user ID, ignoring user-supplied owner_id.
	if userRole != constants.RoleAdmin {
		req.OwnerID = userID
	}

	list, err := uc.productRepo.FindAll(ctx, req)
	if err != nil {
		return nil, err
	}
	return dto.FromProductList(list), nil
}

// UpdateProduct updates a product.
func (uc *productUseCase) UpdateProduct(ctx context.Context, userID, _ string, productID string, req *dto.UpdateProductRequest) (*dto.ProductResponse, error) {
	product, err := uc.productRepo.FindByID(ctx, productID, domain.DefaultParanoidOptions())
	if err != nil {
		return nil, err
	}

	// IDOR PROTECTION: Only owner can update (admin CANNOT update user products)
	if product.OwnerID != userID {
		return nil, ErrAccessDenied
	}

	if err := uc.applyUpdateBaseFields(ctx, product, req); err != nil {
		return nil, err
	}

	details, err := prepareUpdateProductDetails(product.Price, req)
	if err != nil {
		return nil, err
	}

	if err := applyUpdateProductStockRules(product, req, details); err != nil {
		return nil, err
	}

	if err := uc.persistUpdatedProduct(ctx, product, details); err != nil {
		return nil, err
	}

	updatedProduct, updatedVariants, updatedAttributes, err := uc.productRepo.FindByIDWithDetails(ctx, productID, domain.DefaultParanoidOptions())
	if err != nil {
		return nil, err
	}

	// Publish event
	if uc.eventBus != nil {
		uc.publishEvent(ctx, domain.NewProductUpdatedEvent(updatedProduct))
	}

	return dto.FromProductWithVariants(updatedProduct, updatedVariants, updatedAttributes), nil
}

// DeleteProduct deletes a product.
// DeleteProduct deletes a product.
func (uc *productUseCase) DeleteProduct(ctx context.Context, userID, _ string, req *dto.DeleteProductRequest) (*dto.DeleteResponse, error) {
	// Check if product exists
	product, err := uc.productRepo.FindByID(ctx, req.ID, &domain.ParanoidOptions{IncludeDeleted: true})
	if err != nil {
		return nil, err
	}

	// IDOR PROTECTION: Only owner can delete
	if product.OwnerID != userID {
		return nil, ErrAccessDenied
	}

	// Delete product
	if req.Force {
		if err := uc.productRepo.HardDelete(ctx, req.ID); err != nil {
			return nil, err
		}
	} else {
		if err := uc.productRepo.Delete(ctx, req.ID); err != nil {
			return nil, err
		}
	}

	// Publish event
	if uc.eventBus != nil {
		uc.publishEvent(ctx, domain.NewProductDeletedEvent(product.ID, product.OwnerID))
	}

	message := "Product deleted successfully"
	if req.Force {
		message = "Product permanently deleted"
	}

	return &dto.DeleteResponse{
		Success: true,
		Message: message,
	}, nil
}

// RestoreProduct restores a deleted product.
// RestoreProduct restores a deleted product.
func (uc *productUseCase) RestoreProduct(ctx context.Context, userID, _ string, req *dto.RestoreProductRequest) (*dto.ProductResponse, error) {
	// Check if product exists
	product, err := uc.productRepo.FindByID(ctx, req.ID, &domain.ParanoidOptions{IncludeDeleted: true})
	if err != nil {
		return nil, err
	}

	// IDOR PROTECTION: Only owner can restore
	if product.OwnerID != userID {
		return nil, ErrAccessDenied
	}

	if err := uc.productRepo.Restore(ctx, req.ID); err != nil {
		return nil, err
	}

	// Get restored product
	product, err = uc.productRepo.FindByID(ctx, req.ID, domain.DefaultParanoidOptions())
	if err != nil {
		return nil, err
	}

	// Publish event
	if uc.eventBus != nil {
		uc.publishEvent(ctx, domain.NewProductRestoredEvent(product))
	}

	return dto.FromProduct(product), nil
}

// UpdateStock updates product stock.
// UpdateStock updates product stock.
func (uc *productUseCase) UpdateStock(ctx context.Context, userID, _ string, req *dto.UpdateStockRequest) (*dto.UpdateStockResponse, error) {
	product, err := uc.productRepo.FindByID(ctx, req.ID, domain.DefaultParanoidOptions())
	if err != nil {
		return nil, err
	}

	// IDOR PROTECTION: Only owner can update stock
	if product.OwnerID != userID {
		return nil, ErrAccessDenied
	}

	if product.HasVariant {
		return uc.updateVariantProductStock(ctx, req)
	}

	return uc.updateSimpleProductStock(ctx, req, product)
}

type updateProductDetails struct {
	attributes        []*domain.ProductAttribute
	variants          []*domain.ProductVariant
	replaceAttributes bool
	replaceVariants   bool
	totalVariantStock int
}

func (uc *productUseCase) ensureProductNameAvailable(ctx context.Context, ownerID, name string) error {
	exists, err := uc.productRepo.ExistsByNameAndOwner(ctx, name, ownerID)
	if err != nil {
		return fmt.Errorf("failed to check product existence: %w", err)
	}
	if exists {
		return domain.ErrProductNameAlreadyUsed
	}
	return nil
}

func buildAttributes(reqs []*dto.CreateAttributeRequest) []*domain.ProductAttribute {
	attributes := make([]*domain.ProductAttribute, 0, len(reqs))
	for i, attrReq := range reqs {
		if attrReq == nil {
			continue
		}
		displayOrder := attrReq.DisplayOrder
		if displayOrder == 0 && i > 0 {
			displayOrder = i
		}
		attributes = append(attributes, &domain.ProductAttribute{
			Name:         attrReq.Name,
			Values:       attrReq.Values,
			DisplayOrder: displayOrder,
		})
	}
	return attributes
}

func buildVariants(reqs []*dto.CreateVariantRequest, defaultPrice float64) ([]*domain.ProductVariant, int) {
	variants := make([]*domain.ProductVariant, 0, len(reqs))
	totalStock := 0

	for _, variantReq := range reqs {
		if variantReq == nil {
			continue
		}

		variantName := variantReq.Name
		if variantName == "" {
			variantName = variantReq.SKU
		}

		variantPrice := defaultPrice
		if variantReq.Price != nil && *variantReq.Price > 0 {
			variantPrice = *variantReq.Price
		}

		stockQty := variantReq.ResolveStockQuantity()
		variants = append(variants, &domain.ProductVariant{
			Name:            variantName,
			SKU:             variantReq.SKU,
			Price:           variantPrice,
			StockQuantity:   stockQty,
			IsActive:        variantReq.ResolveIsActive(),
			AttributeValues: variantReq.AttributeValues,
			Images:          variantReq.Images,
		})
		totalStock += stockQty
	}

	return variants, totalStock
}

func (uc *productUseCase) applyUpdateBaseFields(ctx context.Context, product *domain.Product, req *dto.UpdateProductRequest) error {
	if req.Name != "" {
		if req.Name != product.Name {
			if err := uc.ensureProductNameAvailable(ctx, product.OwnerID, req.Name); err != nil {
				return err
			}
		}
		product.Name = req.Name
	}

	if req.Price > 0 {
		product.Price = req.Price
	}

	if req.Images != "" {
		product.Images = req.Images
	}

	return nil
}

func prepareUpdateProductDetails(basePrice float64, req *dto.UpdateProductRequest) (*updateProductDetails, error) {
	details := &updateProductDetails{
		replaceAttributes: req.Attributes != nil,
		replaceVariants:   req.Variants != nil,
	}

	if details.replaceVariants && len(req.Variants) > 0 && !details.replaceAttributes {
		return nil, domain.ErrAttributesRequired
	}

	if details.replaceAttributes {
		details.attributes = buildAttributes(req.Attributes)
	}
	if details.replaceVariants {
		details.variants, details.totalVariantStock = buildVariants(req.Variants, basePrice)
	}

	return details, nil
}

func applyUpdateProductStockRules(product *domain.Product, req *dto.UpdateProductRequest, details *updateProductDetails) error {
	if details.replaceVariants {
		if len(details.variants) > 0 {
			product.HasVariant = true
			product.Stock = details.totalVariantStock
			return nil
		}

		product.HasVariant = false
		if req.Stock != nil {
			product.Stock = *req.Stock
		} else {
			product.Stock = 0
		}
		return nil
	}

	if req.Stock != nil {
		if product.HasVariant {
			return domain.ErrDirectStockUpdate
		}
		product.Stock = *req.Stock
	}

	return nil
}

func (uc *productUseCase) persistUpdatedProduct(ctx context.Context, product *domain.Product, details *updateProductDetails) error {
	if details.replaceAttributes || details.replaceVariants {
		return uc.productRepo.UpdateWithDetails(
			ctx,
			product,
			details.attributes,
			details.variants,
			details.replaceAttributes,
			details.replaceVariants,
		)
	}
	return uc.productRepo.Update(ctx, product)
}

func (uc *productUseCase) updateVariantProductStock(ctx context.Context, req *dto.UpdateStockRequest) (*dto.UpdateStockResponse, error) {
	if req.VariantID == "" || req.VariantID == req.ID {
		return nil, domain.ErrVariantIDRequired
	}

	_, variants, _, err := uc.productRepo.FindByIDWithDetails(ctx, req.ID, domain.DefaultParanoidOptions())
	if err != nil {
		return nil, err
	}

	targetVariant := findVariantByID(variants, req.VariantID)
	if targetVariant == nil {
		return nil, domain.ErrVariantNotInProduct
	}
	if err := targetVariant.ReduceStock(req.Stock); err != nil {
		return nil, err
	}

	updatedProductStock, err := uc.productRepo.UpdateVariantStockAndSyncProduct(ctx, req.ID, req.VariantID, targetVariant.StockQuantity)
	if err != nil {
		return nil, err
	}

	uc.publishStockUpdatedEvent(ctx, req.ID, updatedProductStock)
	return &dto.UpdateStockResponse{
		Success: true,
		Message: "Stock updated successfully",
		Stock:   updatedProductStock,
	}, nil
}

func (uc *productUseCase) updateSimpleProductStock(
	ctx context.Context,
	req *dto.UpdateStockRequest,
	product *domain.Product,
) (*dto.UpdateStockResponse, error) {
	if err := product.ReduceStock(req.Stock); err != nil {
		return nil, err
	}
	if err := uc.productRepo.UpdateStock(ctx, req.ID, product.Stock); err != nil {
		return nil, err
	}

	uc.publishStockUpdatedEvent(ctx, req.ID, product.Stock)
	return &dto.UpdateStockResponse{
		Success: true,
		Message: "Stock updated successfully",
		Stock:   product.Stock,
	}, nil
}

func findVariantByID(variants []*domain.ProductVariant, variantID string) *domain.ProductVariant {
	for _, variant := range variants {
		if variant == nil {
			continue
		}
		if variant.ID == variantID {
			return variant
		}
	}
	return nil
}

func (uc *productUseCase) publishStockUpdatedEvent(ctx context.Context, productID string, stock int) {
	if uc.eventBus == nil {
		return
	}
	uc.publishEvent(ctx, domain.NewProductStockUpdatedEvent(productID, stock))
}

// publishEvent publishes an event to the event bus.
func (uc *productUseCase) publishEvent(ctx context.Context, event *domain.ProductEvent) {
	if uc.eventBus == nil {
		return
	}

	// Create event bus event
	ebEvent := eventbus.NewEvent(event.EventType, uc.config.ServiceName, event.ToMap())
	utils.ApplyRequestMetadataToEvent(ctx, ebEvent)

	// Publish asynchronously with error logging
	go func() {
		if _, err := uc.eventBus.Publish(context.Background(), eventbus.StreamProductEvents, ebEvent); err != nil {
			uc.logger.Error("failed to publish event",
				zap.String("event_type", event.EventType),
				zap.Error(err),
			)
		}
	}()
}

// GenerateUUID generates a new UUID.
func GenerateUUID() string {
	return uuid.New().String()
}

// ValidateStock validates stock value.
func ValidateStock(stock int) error {
	if stock < 0 {
		return errors.New("stock cannot be negative")
	}
	return nil
}
