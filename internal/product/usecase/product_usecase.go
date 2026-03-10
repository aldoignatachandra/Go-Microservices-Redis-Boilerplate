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
	// Check if product already exists for this owner
	exists, err := uc.productRepo.ExistsByNameAndOwner(ctx, req.Name, ownerID)
	if err != nil {
		return nil, fmt.Errorf("failed to check product existence: %w", err)
	}
	if exists {
		return nil, domain.ErrProductNameAlreadyUsed
	}

	// Create product
	hasVariant := len(req.Variants) > 0
	product := &domain.Product{
		Name:       req.Name,
		Price:      req.Price,
		Stock:      req.Stock,
		OwnerID:    ownerID,
		HasVariant: hasVariant,
		Images:     req.Images,
	}

	if err := uc.productRepo.Create(ctx, product); err != nil {
		return nil, fmt.Errorf("failed to create product: %w", err)
	}

	// Publish event
	if uc.eventBus != nil {
		uc.publishEvent(ctx, domain.NewProductCreatedEvent(product))
	}

	return dto.FromProduct(product), nil
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

	// Update fields
	if req.Name != "" {
		// Check if name is already used by another product of same owner
		if req.Name != product.Name {
			exists, err := uc.productRepo.ExistsByNameAndOwner(ctx, req.Name, product.OwnerID)
			if err == nil && exists {
				return nil, domain.ErrProductNameAlreadyUsed
			}
		}
		product.Name = req.Name
	}

	if req.Price > 0 {
		product.Price = req.Price
	}

	if req.Stock >= 0 {
		product.Stock = req.Stock
	}

	if req.Images != "" {
		product.Images = req.Images
	}

	if err := uc.productRepo.Update(ctx, product); err != nil {
		return nil, err
	}

	// Publish event
	if uc.eventBus != nil {
		uc.publishEvent(ctx, domain.NewProductUpdatedEvent(product))
	}

	return dto.FromProduct(product), nil
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
		uc.publishEvent(ctx, domain.NewProductDeletedEvent(product.ID))
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

	if err := product.ReduceStock(req.Stock); err != nil {
		return nil, err
	}

	if err := uc.productRepo.UpdateStock(ctx, req.ID, product.Stock); err != nil {
		return nil, err
	}

	// Publish event
	if uc.eventBus != nil {
		uc.publishEvent(ctx, domain.NewProductStockUpdatedEvent(req.ID, product.Stock))
	}

	return &dto.UpdateStockResponse{
		Success: true,
		Message: "Stock updated successfully",
		Stock:   product.Stock,
	}, nil
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
