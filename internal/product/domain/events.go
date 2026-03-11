// Package domain provides event types for the product service.
package domain

import "time"

// Event types for the product service.
const (
	// Product events
	EventProductCreated      = "product.created"
	EventProductUpdated      = "product.updated"
	EventProductDeleted      = "product.deleted"
	EventProductRestored     = "product.restored"
	EventProductStockUpdated = "product.stock_updated"
)

// ProductEvent represents a product-related event.
type ProductEvent struct {
	EventType  string                 `json:"event_type"`
	ProductID  string                 `json:"product_id"`
	Name       string                 `json:"name,omitempty"`
	Price      float64                `json:"price,omitempty"`
	Stock      int                    `json:"stock,omitempty"`
	OwnerID    string                 `json:"owner_id,omitempty"`
	HasVariant bool                   `json:"has_variant,omitempty"`
	Timestamp  time.Time              `json:"timestamp"`
	Metadata   map[string]interface{} `json:"metadata,omitempty"`
}

// NewProductCreatedEvent creates a new product created event.
func NewProductCreatedEvent(product *Product) *ProductEvent {
	return &ProductEvent{
		EventType:  EventProductCreated,
		ProductID:  product.ID,
		Name:       product.Name,
		Price:      product.Price,
		Stock:      product.Stock,
		OwnerID:    product.OwnerID,
		HasVariant: product.HasVariant,
		Timestamp:  time.Now().UTC(),
		Metadata:   make(map[string]interface{}),
	}
}

// NewProductUpdatedEvent creates a new product updated event.
func NewProductUpdatedEvent(product *Product) *ProductEvent {
	return &ProductEvent{
		EventType:  EventProductUpdated,
		ProductID:  product.ID,
		Name:       product.Name,
		Price:      product.Price,
		Stock:      product.Stock,
		OwnerID:    product.OwnerID,
		HasVariant: product.HasVariant,
		Timestamp:  time.Now().UTC(),
		Metadata:   make(map[string]interface{}),
	}
}

// NewProductDeletedEvent creates a new product deleted event.
func NewProductDeletedEvent(productID, ownerID string) *ProductEvent {
	return &ProductEvent{
		EventType: EventProductDeleted,
		ProductID: productID,
		OwnerID:   ownerID,
		Timestamp: time.Now().UTC(),
		Metadata:  make(map[string]interface{}),
	}
}

// NewProductRestoredEvent creates a new product restored event.
func NewProductRestoredEvent(product *Product) *ProductEvent {
	return &ProductEvent{
		EventType:  EventProductRestored,
		ProductID:  product.ID,
		Name:       product.Name,
		Price:      product.Price,
		Stock:      product.Stock,
		OwnerID:    product.OwnerID,
		HasVariant: product.HasVariant,
		Timestamp:  time.Now().UTC(),
		Metadata:   make(map[string]interface{}),
	}
}

// NewProductStockUpdatedEvent creates a new product stock updated event.
func NewProductStockUpdatedEvent(productID string, stock int) *ProductEvent {
	return &ProductEvent{
		EventType: EventProductStockUpdated,
		ProductID: productID,
		Stock:     stock,
		Timestamp: time.Now().UTC(),
		Metadata:  make(map[string]interface{}),
	}
}

// WithMetadata adds metadata to the event.
func (e *ProductEvent) WithMetadata(key string, value interface{}) *ProductEvent {
	if e.Metadata == nil {
		e.Metadata = make(map[string]interface{})
	}
	e.Metadata[key] = value
	return e
}

// ToMap converts the event to a map for Redis storage.
func (e *ProductEvent) ToMap() map[string]interface{} {
	result := map[string]interface{}{
		"event_type": e.EventType,
		"product_id": e.ProductID,
		"timestamp":  e.Timestamp.UnixMilli(),
	}

	if e.Name != "" {
		result["name"] = e.Name
	}
	if e.Price != 0 {
		result["price"] = e.Price
	}
	if e.Stock != 0 {
		result["stock"] = e.Stock
	}
	if e.OwnerID != "" {
		result["owner_id"] = e.OwnerID
	}
	if e.Metadata != nil {
		result["metadata"] = e.Metadata
	}

	return result
}
