// Package dto provides Data Transfer Objects for the product service.
package dto

import (
	"github.com/ignata/go-microservices-boilerplate/internal/product/domain"
)

// CreateProductRequest represents a product creation request.
type CreateProductRequest struct {
	Name        string  `json:"name" binding:"required,min=3,max=255"`
	Description string  `json:"description" binding:"omitempty,max=1000"`
	Price       float64 `json:"price" binding:"required,gt=0"`
	Stock       int     `json:"stock" binding:"required,gt=-1"`
	CategoryID  string  `json:"category_id" binding:"required,uuid"`
}

// UpdateProductRequest represents a product update request.
type UpdateProductRequest struct {
	Name        *string  `json:"name" binding:"omitempty,min=3,max=255"`
	Description *string  `json:"description" binding:"omitempty,max=1000"`
	Price       *float64 `json:"price" binding:"omitempty,gt=0"`
	Stock       *int     `json:"stock" binding:"omitempty,gt=-1"`
	Status      *string  `json:"status" binding:"omitempty,oneof=ACTIVE INACTIVE"`
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
	Status         string `form:"status" binding:"omitempty,oneof=ACTIVE INACTIVE"`
	Search         string `form:"search" binding:"omitempty"`
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
	Force bool   `form:"force"` // Force hard delete
}

// RestoreProductRequest represents a request to restore a deleted product.
type RestoreProductRequest struct {
	ID string `uri:"id" binding:"required,uuid"`
}

// UpdateStockRequest represents a request to update product stock.
type UpdateStockRequest struct {
	ID    string `uri:"id" binding:"required,uuid"`
	Stock int    `json:"stock" binding:"required,gt=-1"`
}