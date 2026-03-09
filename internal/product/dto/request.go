// Package dto provides Data Transfer Objects for the product service.
package dto

import (
	"github.com/ignata/go-microservices-boilerplate/internal/product/domain"
)

// CreateVariantRequest represents a variant in create/update requests (aligned with Bun-Hono).
type CreateVariantRequest struct {
	Name            string            `json:"name" binding:"required,min=1,max=255"`
	SKU             string            `json:"sku" binding:"required,min=1,max=100"`
	Price           *float64          `json:"price" binding:"omitempty,gt=0"`
	StockQuantity   int               `json:"stockQuantity" binding:"omitempty,gte=0"`
	IsActive        bool              `json:"isActive" binding:"omitempty"`
	AttributeValues map[string]string `json:"attributeValues" binding:"omitempty"`
	Images          string            `json:"images" binding:"omitempty"`
}

// CreateAttributeRequest represents an attribute in create/update requests (aligned with Bun-Hono).
type CreateAttributeRequest struct {
	Name         string   `json:"name" binding:"required,min=1,max=100"`
	Values       []string `json:"values" binding:"required,min=1"`
	DisplayOrder int      `json:"displayOrder" binding:"omitempty,gte=0"`
}

// CreateProductRequest represents a product creation request (aligned with Bun-Hono).
type CreateProductRequest struct {
	Name       string                    `json:"name" binding:"required,min=2,max=255"`
	Price      float64                   `json:"price" binding:"required,gt=0"`
	Stock      int                       `json:"stock" binding:"omitempty,gte=0"`
	OwnerID    string                    `json:"ownerId" binding:"required,uuid"`
	Images     string                    `json:"images" binding:"omitempty"`
	Attributes []*CreateAttributeRequest `json:"attributes" binding:"omitempty"`
	Variants   []*CreateVariantRequest   `json:"variants" binding:"omitempty"`
}

// UpdateProductRequest represents a product update request (aligned with Bun-Hono).
type UpdateProductRequest struct {
	ID         string                    `uri:"id" binding:"required,uuid"`
	Name       string                    `json:"name" binding:"omitempty,min=2,max=255"`
	Price      float64                   `json:"price" binding:"omitempty,gt=0"`
	Stock      int                       `json:"stock" binding:"omitempty,gte=0"`
	Images     string                    `json:"images" binding:"omitempty"`
	Attributes []*CreateAttributeRequest `json:"attributes" binding:"omitempty"`
	Variants   []*CreateVariantRequest   `json:"variants" binding:"omitempty"`
}

// GetProductRequest represents a request to get a product.
type GetProductRequest struct {
	ID             string `uri:"id" binding:"required,uuid"`
	IncludeDeleted bool   `form:"include_deleted"`
}

// GetParanoidOptions returns paranoid options from the request.
func (r *GetProductRequest) GetParanoidOptions() *domain.ParanoidOptions {
	return &domain.ParanoidOptions{
		IncludeDeleted: r.IncludeDeleted,
		OnlyActive:     !r.IncludeDeleted,
	}
}

// ListProductsRequest represents a request to list products.
type ListProductsRequest struct {
	Page           int    `form:"page" binding:"omitempty,min=1"`
	Limit          int    `form:"limit" binding:"omitempty,min=1,max=100"`
	OwnerID        string `form:"owner_id" binding:"omitempty,uuid"`
	Search         string `form:"search" binding:"omitempty"`
	Status         string `form:"status" binding:"omitempty,oneof=ACTIVE INACTIVE"`
	IncludeDeleted bool   `form:"include_deleted"`
	OnlyDeleted    bool   `form:"only_deleted"`
}

// GetPage returns the page number with default.
func (r *ListProductsRequest) GetPage() int {
	if r.Page < 1 {
		return 1
	}
	return r.Page
}

// GetLimit returns the limit with default.
func (r *ListProductsRequest) GetLimit() int {
	if r.Limit < 1 {
		return 10
	}
	if r.Limit > 100 {
		return 100
	}
	return r.Limit
}

// GetParanoidOptions returns paranoid options from the request.
func (r *ListProductsRequest) GetParanoidOptions() *domain.ParanoidOptions {
	return &domain.ParanoidOptions{
		IncludeDeleted: r.IncludeDeleted,
		OnlyDeleted:    r.OnlyDeleted,
		OnlyActive:     !r.IncludeDeleted && !r.OnlyDeleted,
	}
}

// DeleteProductRequest represents a request to delete a product.
type DeleteProductRequest struct {
	ID    string `uri:"id" binding:"required,uuid"`
	Force bool   `form:"force"`
}

// RestoreProductRequest represents a request to restore a deleted product.
type RestoreProductRequest struct {
	ID string `uri:"id" binding:"required,uuid"`
}

// UpdateStockRequest represents a request to update product stock.
type UpdateStockRequest struct {
	ID    string `uri:"id" binding:"required,uuid"`
	Stock int    `json:"stock" binding:"required,min=0"`
}
