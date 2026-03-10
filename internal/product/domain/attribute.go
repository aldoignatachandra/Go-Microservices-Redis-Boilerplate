// Package domain provides domain entities for the product service.
package domain

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// ProductAttribute represents a product attribute entity.
type ProductAttribute struct {
	ID           string         `gorm:"type:uuid;primary_key;" json:"id"`
	ProductID    string         `gorm:"type:uuid;not null;index" json:"product_id"`
	Name         string         `gorm:"type:varchar(100);not null" json:"name"`
	Values       []string       `gorm:"type:jsonb;serializer:json" json:"values"`
	DisplayOrder int            `gorm:"type:int;default:0" json:"display_order"`
	CreatedAt    time.Time      `gorm:"not null" json:"created_at"`
	UpdatedAt    time.Time      `gorm:"not null" json:"updated_at"`
	DeletedAt    gorm.DeletedAt `gorm:"index" json:"deleted_at,omitempty"`
}

// TableName specifies the table name for ProductAttribute.
func (ProductAttribute) TableName() string {
	return "product_attributes"
}

// BeforeCreate is a GORM hook that runs before creating an attribute.
func (a *ProductAttribute) BeforeCreate(_ *gorm.DB) error {
	if a.ID == "" {
		a.ID = uuid.New().String()
	}
	now := time.Now().UTC()
	a.CreatedAt = now
	a.UpdatedAt = now
	return nil
}

// BeforeUpdate is a GORM hook that runs before updating an attribute.
func (a *ProductAttribute) BeforeUpdate(_ *gorm.DB) error {
	a.UpdatedAt = time.Now().UTC()
	return nil
}

// ToSafeAttribute returns a copy of the attribute without sensitive fields.
func (a *ProductAttribute) ToSafeAttribute() *SafeAttribute {
	return &SafeAttribute{
		ID:           a.ID,
		ProductID:    a.ProductID,
		Name:         a.Name,
		Values:       a.Values,
		DisplayOrder: a.DisplayOrder,
		CreatedAt:    a.CreatedAt,
		UpdatedAt:    a.UpdatedAt,
	}
}

// SafeAttribute represents an attribute without sensitive fields.
type SafeAttribute struct {
	ID           string    `json:"id"`
	ProductID    string    `json:"product_id"`
	Name         string    `json:"name"`
	Values       []string  `json:"values"`
	DisplayOrder int       `json:"display_order"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
}

// AttributeList represents a list of attributes.
type AttributeList struct {
	Attributes []*ProductAttribute `json:"attributes"`
	Total      int64               `json:"total"`
	Page       int                 `json:"page"`
	Limit      int                 `json:"limit"`
	TotalPages int                 `json:"total_pages"`
}
