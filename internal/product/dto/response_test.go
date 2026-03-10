package dto

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"github.com/ignata/go-microservices-boilerplate/internal/product/domain"
)

func TestFromProduct_UsesVariantPriceRangeWhenAvailable(t *testing.T) {
	product := &domain.Product{
		Model: domain.Model{
			ID:        "prod-1",
			CreatedAt: time.Now().UTC(),
			UpdatedAt: time.Now().UTC(),
		},
		Name:       "Mechanical Keyboard",
		Price:      89.99,
		Stock:      35,
		OwnerID:    "owner-1",
		HasVariant: true,
		PriceMin:   89.99,
		PriceMax:   99.99,
	}

	resp := FromProduct(product)
	assert.Equal(t, 89.99, resp.Price.Min)
	assert.Equal(t, 99.99, resp.Price.Max)
	assert.Equal(t, "$89.99 - $99.99", resp.Price.Display)
}

func TestFromProduct_FallsBackToBasePriceWhenRangeMissing(t *testing.T) {
	product := &domain.Product{
		Model: domain.Model{
			ID:        "prod-2",
			CreatedAt: time.Now().UTC(),
			UpdatedAt: time.Now().UTC(),
		},
		Name:       "Classic Cap",
		Price:      19.99,
		Stock:      150,
		OwnerID:    "owner-1",
		HasVariant: false,
	}

	resp := FromProduct(product)
	assert.Equal(t, 19.99, resp.Price.Min)
	assert.Equal(t, 19.99, resp.Price.Max)
	assert.Equal(t, "$19.99", resp.Price.Display)
}

func TestFromProductWithVariants_UsesOnlyVariantPricesForRange(t *testing.T) {
	product := &domain.Product{
		Model: domain.Model{
			ID:        "prod-3",
			CreatedAt: time.Now().UTC(),
			UpdatedAt: time.Now().UTC(),
		},
		Name:       "Gaming Keyboard",
		Price:      19.99, // intentionally lower than variant prices
		Stock:      35,
		OwnerID:    "owner-1",
		HasVariant: true,
	}

	variants := []*domain.ProductVariant{
		{ID: "var-1", ProductID: "prod-3", SKU: "KEY-TKL-BROWN", Price: 89.99, StockQuantity: 10, IsActive: true},
		{ID: "var-2", ProductID: "prod-3", SKU: "KEY-FULL-RED", Price: 99.99, StockQuantity: 5, IsActive: true},
	}

	resp := FromProductWithVariants(product, variants, nil)
	assert.Equal(t, 89.99, resp.Price.Min)
	assert.Equal(t, 99.99, resp.Price.Max)
	assert.Equal(t, "$89.99 - $99.99", resp.Price.Display)
}
