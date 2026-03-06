// Package domain provides domain entities for the product service.
package domain

import (
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// ProductStatus represents the product status enum.
type ProductStatus string

const (
	ProductStatusActive   ProductStatus = "ACTIVE"
	ProductStatusInactive ProductStatus = "INACTIVE"
	ProductStatusDeleted  ProductStatus = "DELETED"
)

// IsValid checks if the status is valid.
func (s ProductStatus) IsValid() bool {
	return s == ProductStatusActive || s == ProductStatusInactive || s == ProductStatusDeleted
}

// Model is the base model for all entities.
type Model struct {
	ID        string         `gorm:"type:uuid;primary_key;" json:"id"`
	CreatedAt time.Time      `gorm:"not null" json:"created_at"`
	UpdatedAt time.Time      `gorm:"not null" json:"updated_at"`
	DeletedAt gorm.DeletedAt `gorm:"index" json:"deleted_at,omitempty"`
}

// BeforeCreate is a GORM hook that sets the UUID.
func (m *Model) BeforeCreate(tx *gorm.DB) error {
	if m.ID == "" {
		m.ID = uuid.New().String()
	}
	return nil
}

// Product represents a product entity.
type Product struct {
	Model
	Name        string        `gorm:"type:varchar(255);not null" json:"name"`
	Description string        `gorm:"type:text" json:"description"`
	Price       float64       `gorm:"type:decimal(10,2);not null" json:"price"`
	Stock       int           `gorm:"type:int;not null;default:0" json:"stock"`
	Status      ProductStatus `gorm:"type:varchar(50);not null;default:'ACTIVE'" json:"status"`
	CategoryID  string        `gorm:"type:uuid;not null" json:"category_id"`
}

// TableName specifies the table name for Product.
func (Product) TableName() string {
	return "products"
}

// IsAvailable checks if the product is available for purchase.
func (p *Product) IsAvailable() bool {
	return p.Status == ProductStatusActive && p.Stock > 0 && p.DeletedAt.Valid == false
}

// ReduceStock reduces the product stock by the given amount.
func (p *Product) ReduceStock(amount int) error {
	if amount <= 0 {
		return fmt.Errorf("invalid stock reduction amount")
	}
	if p.Stock < amount {
		return fmt.Errorf("insufficient stock")
	}
	p.Stock -= amount
	return nil
}

// IncreaseStock increases the product stock by the given amount.
func (p *Product) IncreaseStock(amount int) error {
	if amount <= 0 {
		return fmt.Errorf("invalid stock increase amount")
	}
	p.Stock += amount
	return nil
}

// BeforeCreate is a GORM hook that runs before creating a product.
func (p *Product) BeforeCreate(tx *gorm.DB) error {
	// Generate ID if empty (this handles cases where Model.BeforeCreate might not be called)
	if p.ID == "" {
		p.ID = uuid.New().String()
	}
	now := time.Now().UTC()
	p.CreatedAt = now
	p.UpdatedAt = now
	return nil
}

// BeforeUpdate is a GORM hook that runs before updating a product.
func (p *Product) BeforeUpdate(tx *gorm.DB) error {
	p.UpdatedAt = time.Now().UTC()
	return nil
}

// ToSafeProduct returns a copy of the product without sensitive fields.
func (p *Product) ToSafeProduct() *SafeProduct {
	return &SafeProduct{
		ID:          p.ID,
		Name:        p.Name,
		Description: p.Description,
		Price:       p.Price,
		Stock:       p.Stock,
		Status:      p.Status,
		CategoryID:  p.CategoryID,
		CreatedAt:   p.CreatedAt,
		UpdatedAt:   p.UpdatedAt,
	}
}

// SafeProduct represents a product without sensitive fields.
type SafeProduct struct {
	ID          string        `json:"id"`
	Name        string        `json:"name"`
	Description string        `json:"description"`
	Price       float64       `json:"price"`
	Stock       int           `json:"stock"`
	Status      ProductStatus `json:"status"`
	CategoryID  string        `json:"category_id"`
	CreatedAt   time.Time     `json:"created_at"`
	UpdatedAt   time.Time     `json:"updated_at"`
}

// ProductList represents a list of products with pagination info.
type ProductList struct {
	Products   []*Product `json:"products"`
	Total      int64      `json:"total"`
	Page       int        `json:"page"`
	Limit      int        `json:"limit"`
	TotalPages int        `json:"total_pages"`
}

// ParanoidOptions defines options for querying with soft delete support.
type ParanoidOptions struct {
	IncludeDeleted bool `form:"include_deleted" json:"include_deleted"`
	OnlyDeleted    bool `form:"only_deleted" json:"only_deleted"`
	OnlyActive     bool `form:"only_active" json:"only_active"`
}

// DefaultParanoidOptions returns default paranoid options (only active).
func DefaultParanoidOptions() *ParanoidOptions {
	return &ParanoidOptions{
		OnlyActive: true,
	}
}

// Validate validates the paranoid options.
func (p *ParanoidOptions) Validate() error {
	// Default to only active
	if !p.IncludeDeleted && !p.OnlyDeleted && !p.OnlyActive {
		p.OnlyActive = true
	}
	return nil
}

// ShouldIncludeDeleted returns true if deleted records should be included.
func (p *ParanoidOptions) ShouldIncludeDeleted() bool {
	return p.IncludeDeleted || p.OnlyDeleted
}

// ShouldOnlyDeleted returns true if only deleted records should be returned.
func (p *ParanoidOptions) ShouldOnlyDeleted() bool {
	return p.OnlyDeleted
}

// Error definitions
var (
	ErrProductNotFound        = errors.New("product not found")
	ErrProductNameAlreadyUsed = errors.New("product name already used")
)

// IsNotFoundError checks if the error is a not found error.
func IsNotFoundError(err error) bool {
	return err == ErrProductNotFound
}

// IsValidationError checks if the error is a validation error.
func IsValidationError(err error) bool {
	if err == nil {
		return false
	}
	// Check for "insufficient stock" error string as it is returned dynamically
	// We use Contains because the error might be wrapped
	errStr := err.Error()
	if strings.Contains(errStr, "insufficient stock") {
		return true
	}
	// Check for "invalid stock" errors
	if strings.Contains(errStr, "invalid stock reduction amount") || strings.Contains(errStr, "invalid stock increase amount") {
		return true
	}
	return false
}
