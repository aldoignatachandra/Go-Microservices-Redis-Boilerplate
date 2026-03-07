// Package dto provides Data Transfer Objects for the product service.
package dto

import (
	"time"

	"github.com/ignata/go-microservices-boilerplate/internal/product/domain"
)

// ProductResponse represents a product response.
type ProductResponse struct {
	ID          string  `json:"id"`
	Name        string  `json:"name"`
	Description string  `json:"description"`
	Price       float64 `json:"price"`
	Stock       int     `json:"stock"`
	Status      string  `json:"status"`
	CategoryID  string  `json:"category_id"`
	CreatedAt   string  `json:"created_at"`
	UpdatedAt   string  `json:"updated_at"`
}

// ProductListResponse represents a list of products with pagination info.
type ProductListResponse struct {
	Products   []*ProductResponse `json:"products"`
	Total      int64              `json:"total"`
	Page       int                `json:"page"`
	Limit      int                `json:"limit"`
	TotalPages int                `json:"total_pages"`
}

// FromProduct converts a domain Product to a ProductResponse.
func FromProduct(product *domain.Product) *ProductResponse {
	return &ProductResponse{
		ID:          product.ID,
		Name:        product.Name,
		Description: product.Description,
		Price:       product.Price,
		Stock:       product.Stock,
		Status:      string(product.Status),
		CategoryID:  product.CategoryID,
		CreatedAt:   product.CreatedAt.Format(time.RFC3339),
		UpdatedAt:   product.UpdatedAt.Format(time.RFC3339),
	}
}

// FromProductList converts a domain ProductList to a ProductListResponse.
func FromProductList(productList *domain.ProductList) *ProductListResponse {
	products := make([]*ProductResponse, len(productList.Products))
	for i, product := range productList.Products {
		products[i] = FromProduct(product)
	}

	return &ProductListResponse{
		Products:   products,
		Total:      productList.Total,
		Page:       productList.Page,
		Limit:      productList.Limit,
		TotalPages: productList.TotalPages,
	}
}

// MessageResponse represents a simple message response.
type MessageResponse struct {
	Message string `json:"message"`
}

// SuccessResponse represents a success response.
type SuccessResponse struct {
	Success bool   `json:"success"`
	Message string `json:"message"`
}

// DeleteResponse represents a delete response.
type DeleteResponse struct {
	Success bool   `json:"success"`
	Message string `json:"message"`
}

// UpdateStockResponse represents an update stock response.
type UpdateStockResponse struct {
	Success bool   `json:"success"`
	Message string `json:"message"`
	Stock   int    `json:"stock"`
}
