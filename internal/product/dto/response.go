// Package dto provides Data Transfer Objects for the product service.
package dto

import (
	"fmt"
	"time"

	"github.com/ignata/go-microservices-boilerplate/internal/product/domain"
)

// PriceRange represents a price range for products with variants.
type PriceRange struct {
	Min     float64 `json:"min"`
	Max     float64 `json:"max"`
	Display string  `json:"display"`
}

// VariantResponse represents a variant in responses.
type VariantResponse struct {
	ID              string            `json:"id"`
	SKU             string            `json:"sku"`
	Price           *float64          `json:"price,omitempty"`
	StockQuantity   int               `json:"stockQuantity"`
	AvailableStock  int               `json:"availableStock"`
	IsActive        bool              `json:"isActive"`
	AttributeValues map[string]string `json:"attributeValues"`
	Images          string            `json:"images,omitempty"`
}

// AttributeResponse represents an attribute in responses.
type AttributeResponse struct {
	ID           string   `json:"id"`
	Name         string   `json:"name"`
	Values       []string `json:"values"`
	DisplayOrder int      `json:"displayOrder"`
}

// ProductResponse represents a product response (aligned with Bun-Hono).
type ProductResponse struct {
	ID         string               `json:"id"`
	Name       string               `json:"name"`
	Price      PriceRange           `json:"price"`
	Stock      int                  `json:"stock"`
	HasVariant bool                 `json:"hasVariant"`
	OwnerID    string               `json:"ownerId"`
	Attributes []*AttributeResponse `json:"attributes,omitempty"`
	Variants   []*VariantResponse   `json:"variants,omitempty"`
	CreatedAt  time.Time            `json:"createdAt"`
	UpdatedAt  time.Time            `json:"updatedAt"`
	DeletedAt  *time.Time           `json:"deletedAt,omitempty"`
}

// ProductListResponse represents a list of products with pagination info.
type ProductListResponse struct {
	Data            []*ProductResponse `json:"data"`
	Total           int64              `json:"total"`
	Page            int                `json:"page"`
	Limit           int                `json:"limit"`
	TotalPages      int                `json:"totalPages"`
	HasNextPage     bool               `json:"hasNextPage"`
	HasPreviousPage bool               `json:"hasPreviousPage"`
}

// FromProduct converts a domain Product to a ProductResponse.
func FromProduct(product *domain.Product) *ProductResponse {
	minPrice := product.Price
	maxPrice := product.Price
	if product.HasVariant && product.PriceMin > 0 && product.PriceMax > 0 {
		minPrice = product.PriceMin
		maxPrice = product.PriceMax
	}

	price := PriceRange{
		Min:     minPrice,
		Max:     maxPrice,
		Display: formatPriceRange(minPrice, maxPrice),
	}

	resp := &ProductResponse{
		ID:         product.ID,
		Name:       product.Name,
		Price:      price,
		Stock:      product.Stock,
		HasVariant: product.HasVariant,
		OwnerID:    product.OwnerID,
		CreatedAt:  product.CreatedAt,
		UpdatedAt:  product.UpdatedAt,
	}

	if product.DeletedAt.Valid {
		resp.DeletedAt = &product.DeletedAt.Time
	}

	return resp
}

// FromProductWithVariants converts a domain Product with variants to a ProductResponse.
func FromProductWithVariants(product *domain.Product, variants []*domain.ProductVariant, attributes []*domain.ProductAttribute) *ProductResponse {
	resp := FromProduct(product)

	// Convert attributes
	if len(attributes) > 0 {
		resp.Attributes = make([]*AttributeResponse, len(attributes))
		for i, attr := range attributes {
			resp.Attributes[i] = &AttributeResponse{
				ID:           attr.ID,
				Name:         attr.Name,
				Values:       attr.Values,
				DisplayOrder: attr.DisplayOrder,
			}
		}
	}

	// Convert variants
	if len(variants) > 0 {
		resp.Variants = make([]*VariantResponse, len(variants))
		minPrice := 0.0
		maxPrice := 0.0
		hasValidPrice := false

		for i, v := range variants {
			resp.Variants[i] = &VariantResponse{
				ID:              v.ID,
				SKU:             v.SKU,
				Price:           &v.Price,
				StockQuantity:   v.StockQuantity,
				AvailableStock:  v.StockQuantity - v.StockReserved,
				IsActive:        v.IsActive,
				AttributeValues: v.AttributeValues,
				Images:          v.Images,
			}

			if v.Price <= 0 {
				continue
			}

			if !hasValidPrice {
				minPrice = v.Price
				maxPrice = v.Price
				hasValidPrice = true
				continue
			}

			// Update price range strictly from variant prices.
			if v.Price < minPrice {
				minPrice = v.Price
			}
			if v.Price > maxPrice {
				maxPrice = v.Price
			}
		}

		// Update price to PriceRange.
		if hasValidPrice {
			resp.Price = PriceRange{
				Min:     minPrice,
				Max:     maxPrice,
				Display: formatPriceRange(minPrice, maxPrice),
			}
		}
	}

	return resp
}

// FromProductList converts a domain ProductList to a ProductListResponse.
func FromProductList(productList *domain.ProductList) *ProductListResponse {
	products := make([]*ProductResponse, len(productList.Products))
	for i, product := range productList.Products {
		products[i] = FromProduct(product)
	}

	return &ProductListResponse{
		Data:            products,
		Total:           productList.Total,
		Page:            productList.Page,
		Limit:           productList.Limit,
		TotalPages:      productList.TotalPages,
		HasNextPage:     productList.Page < productList.TotalPages,
		HasPreviousPage: productList.Page > 1,
	}
}

// formatPriceRange formats a price range for display.
func formatPriceRange(minPrice, maxPrice float64) string {
	if minPrice == maxPrice {
		return formatCurrency(minPrice)
	}
	return formatCurrency(minPrice) + " - " + formatCurrency(maxPrice)
}

// formatCurrency formats a float64 as currency.
func formatCurrency(amount float64) string {
	return "$" + formatNumber(amount)
}

// formatNumber formats a number with 2 decimal places.
func formatNumber(n float64) string {
	return fmt.Sprintf("%.2f", n)
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
