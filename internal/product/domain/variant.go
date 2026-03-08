// Package domain provides domain entities for the product service.
package domain

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// ProductVariant represents a product variant entity.
type ProductVariant struct {
	ID              string            `gorm:"type:uuid;primary_key;" json:"id"`
	ProductID       string            `gorm:"type:uuid;not null;index" json:"product_id"`
	Name            string            `gorm:"type:varchar(255);not null" json:"name"`
	SKU             string            `gorm:"type:varchar(100);not null" json:"sku"`
	Price           float64           `gorm:"type:decimal(10,2)" json:"price"`
	StockQuantity   int               `gorm:"type:int;not null;default:0" json:"stock_quantity"`
	StockReserved   int               `gorm:"type:int;not null;default:0" json:"stock_reserved"`
	IsActive        bool              `gorm:"default:true;not null" json:"is_active"`
	AttributeValues map[string]string `gorm:"type:jsonb" json:"attribute_values"`
	Images          string            `gorm:"type:text" json:"images"`
	CreatedAt       time.Time         `gorm:"not null" json:"created_at"`
	UpdatedAt       time.Time         `gorm:"not null" json:"updated_at"`
	DeletedAt       gorm.DeletedAt    `gorm:"index" json:"deleted_at,omitempty"`
}

// TableName specifies the table name for ProductVariant.
func (ProductVariant) TableName() string {
	return "product_variants"
}

// IsAvailable checks if the variant is available for purchase.
func (v *ProductVariant) IsAvailable() bool {
	return v.StockQuantity > 0 && v.IsActive && !v.DeletedAt.Valid
}

// ReduceStock reduces the variant stock by the given amount.
func (v *ProductVariant) ReduceStock(amount int) error {
	if amount <= 0 {
		return ErrInvalidStockReduction
	}
	if v.StockQuantity < amount {
		return ErrInsufficientStock
	}
	v.StockQuantity -= amount
	return nil
}

// ReserveStock reserves stock for an order.
func (v *ProductVariant) ReserveStock(amount int) error {
	if amount <= 0 {
		return ErrInvalidStockReduction
	}
	available := v.StockQuantity - v.StockReserved
	if available < amount {
		return ErrInsufficientStock
	}
	v.StockReserved += amount
	return nil
}

// ReleaseReservedStock releases reserved stock.
func (v *ProductVariant) ReleaseReservedStock(amount int) error {
	if amount <= 0 {
		return ErrInvalidStockReduction
	}
	if v.StockReserved < amount {
		return ErrInvalidStockReduction
	}
	v.StockReserved -= amount
	return nil
}

// BeforeCreate is a GORM hook that runs before creating a variant.
func (v *ProductVariant) BeforeCreate(_ *gorm.DB) error {
	if v.ID == "" {
		v.ID = uuid.New().String()
	}
	now := time.Now().UTC()
	v.CreatedAt = now
	v.UpdatedAt = now
	return nil
}

// BeforeUpdate is a GORM hook that runs before updating a variant.
func (v *ProductVariant) BeforeUpdate(_ *gorm.DB) error {
	v.UpdatedAt = time.Now().UTC()
	return nil
}

// ToSafeVariant returns a copy of the variant without sensitive fields.
func (v *ProductVariant) ToSafeVariant() *SafeVariant {
	return &SafeVariant{
		ID:              v.ID,
		ProductID:       v.ProductID,
		Name:            v.Name,
		SKU:             v.SKU,
		Price:           v.Price,
		StockQuantity:   v.StockQuantity,
		StockReserved:   v.StockReserved,
		IsActive:        v.IsActive,
		AttributeValues: v.AttributeValues,
		Images:          v.Images,
		CreatedAt:       v.CreatedAt,
		UpdatedAt:       v.UpdatedAt,
	}
}

// SafeVariant represents a variant without sensitive fields.
type SafeVariant struct {
	ID              string            `json:"id"`
	ProductID       string            `json:"product_id"`
	Name            string            `json:"name"`
	SKU             string            `json:"sku"`
	Price           float64           `json:"price"`
	StockQuantity   int               `json:"stock_quantity"`
	StockReserved   int               `json:"stock_reserved"`
	IsActive        bool              `json:"is_active"`
	AttributeValues map[string]string `json:"attribute_values"`
	Images          string            `json:"images"`
	CreatedAt       time.Time         `json:"created_at"`
	UpdatedAt       time.Time         `json:"updated_at"`
}

// VariantList represents a list of variants.
type VariantList struct {
	Variants   []*ProductVariant `json:"variants"`
	Total      int64             `json:"total"`
	Page       int               `json:"page"`
	Limit      int               `json:"limit"`
	TotalPages int               `json:"total_pages"`
}
