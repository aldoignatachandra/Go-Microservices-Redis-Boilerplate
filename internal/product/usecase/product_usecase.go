// Package usecase provides business logic for the product service.
package usecase

import (
	"context"
	"errors"
	"fmt"

	"github.com/google/uuid"

	"github.com/ignata/go-microservices-boilerplate/internal/product/domain"
	"github.com/ignata/go-microservices-boilerplate/internal/product/dto"
	"github.com/ignata/go-microservices-boilerplate/internal/product/repository"
	"github.com/ignata/go-microservices-boilerplate/pkg/eventbus"
)

// ProductUseCase defines the interface for product business logic.
type ProductUseCase interface {
	// CreateProduct creates a new product
	CreateProduct(ctx context.Context, req *dto.CreateProductRequest) (*dto.ProductResponse, error)
	// GetProduct gets a product by ID
	GetProduct(ctx context.Context, req *dto.GetProductRequest) (*dto.ProductResponse, error)
	// ListProducts lists products with pagination
	ListProducts(ctx context.Context, req *dto.ListProductsRequest) (*dto.ProductListResponse, error)
	// UpdateProduct updates a product
	UpdateProduct(ctx context.Context, productID string, req *dto.UpdateProductRequest) (*dto.ProductResponse, error)
	// DeleteProduct deletes a product
	DeleteProduct(ctx context.Context, req *dto.DeleteProductRequest) (*dto.DeleteResponse, error)
	// RestoreProduct restores a deleted product
	RestoreProduct(ctx context.Context, req *dto.RestoreProductRequest) (*dto.ProductResponse, error)
	// UpdateStock updates product stock
	UpdateStock(ctx context.Context, req *dto.UpdateStockRequest) (*dto.UpdateStockResponse, error)
}

// Config holds usecase configuration.
type Config struct {
	ServiceName string
}

// productUseCase implements ProductUseCase.
type productUseCase struct {
	productRepo repository.ProductRepository
	eventBus    *eventbus.Producer
	config      Config
}

// NewProductUseCase creates a new product usecase.
func NewProductUseCase(
	productRepo repository.ProductRepository,
	eventBus *eventbus.Producer,
	config Config,
) ProductUseCase {
	return &productUseCase{
		productRepo: productRepo,
		eventBus:    eventBus,
		config:      config,
	}
}

// CreateProduct creates a new product.
func (uc *productUseCase) CreateProduct(ctx context.Context, req *dto.CreateProductRequest) (*dto.ProductResponse, error) {
	// Check if product already exists
	exists, err := uc.productRepo.ExistsByName(ctx, req.Name)
	if err != nil {
		return nil, fmt.Errorf("failed to check product existence: %w", err)
	}
	if exists {
		return nil, domain.ErrProductNameAlreadyUsed
	}

	// Create product
	product := &domain.Product{
		Name:        req.Name,
		Description: req.Description,
		Price:       req.Price,
		Stock:       req.Stock,
		Status:      domain.ProductStatusActive,
		CategoryID:  req.CategoryID,
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

// GetProduct gets a product by ID.
func (uc *productUseCase) GetProduct(ctx context.Context, req *dto.GetProductRequest) (*dto.ProductResponse, error) {
	product, err := uc.productRepo.FindByID(ctx, req.ID, req.GetParanoidOptions())
	if err != nil {
		return nil, err
	}
	return dto.FromProduct(product), nil
}

// ListProducts lists products with pagination.
func (uc *productUseCase) ListProducts(ctx context.Context, req *dto.ListProductsRequest) (*dto.ProductListResponse, error) {
	list, err := uc.productRepo.FindAll(ctx, req)
	if err != nil {
		return nil, err
	}
	return dto.FromProductList(list), nil
}

// UpdateProduct updates a product.
func (uc *productUseCase) UpdateProduct(ctx context.Context, productID string, req *dto.UpdateProductRequest) (*dto.ProductResponse, error) {
	product, err := uc.productRepo.FindByID(ctx, productID, domain.DefaultParanoidOptions())
	if err != nil {
		return nil, err
	}

	// Update fields
	if req.Name != nil {
		// Check if name is already used by another product
		if *req.Name != product.Name {
			existingProduct, err := uc.productRepo.ExistsByName(ctx, *req.Name)
			if err == nil && existingProduct {
				return nil, domain.ErrProductNameAlreadyUsed
			}
		}
		product.Name = *req.Name
	}

	if req.Description != nil {
		product.Description = *req.Description
	}

	if req.Price != nil {
		product.Price = *req.Price
	}

	if req.Status != nil {
		status := domain.ProductStatus(*req.Status)
		if status.IsValid() {
			product.Status = status
		}
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
func (uc *productUseCase) DeleteProduct(ctx context.Context, req *dto.DeleteProductRequest) (*dto.DeleteResponse, error) {
	// Check if product exists
	product, err := uc.productRepo.FindByID(ctx, req.ID, &domain.ParanoidOptions{IncludeDeleted: true})
	if err != nil {
		return nil, err
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
func (uc *productUseCase) RestoreProduct(ctx context.Context, req *dto.RestoreProductRequest) (*dto.ProductResponse, error) {
	if err := uc.productRepo.Restore(ctx, req.ID); err != nil {
		return nil, err
	}

	// Get restored product
	product, err := uc.productRepo.FindByID(ctx, req.ID, domain.DefaultParanoidOptions())
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
func (uc *productUseCase) UpdateStock(ctx context.Context, req *dto.UpdateStockRequest) (*dto.UpdateStockResponse, error) {
	product, err := uc.productRepo.FindByID(ctx, req.ID, domain.DefaultParanoidOptions())
	if err != nil {
		return nil, err
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

	// Add correlation ID from context if available
	if correlationID, ok := ctx.Value("correlation_id").(string); ok && correlationID != "" {
		ebEvent.WithCorrelationID(correlationID)
	}

	// Publish asynchronously
	go func() {
		_, _ = uc.eventBus.Publish(context.Background(), eventbus.StreamProductEvents, ebEvent)
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