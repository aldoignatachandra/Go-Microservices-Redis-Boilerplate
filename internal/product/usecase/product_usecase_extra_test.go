// Package usecase_test provides tests for the product usecase.
package usecase_test

import (
	"context"
	"errors"
	"fmt"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"

	"github.com/ignata/go-microservices-boilerplate/internal/common/constants"
	"github.com/ignata/go-microservices-boilerplate/internal/product/domain"
	"github.com/ignata/go-microservices-boilerplate/internal/product/dto"
	"github.com/ignata/go-microservices-boilerplate/internal/product/repository"
	"github.com/ignata/go-microservices-boilerplate/internal/product/usecase"
)

const (
	testUserID    = "test-user-id"
	testAdminID   = "test-admin-id"
	testUserRole  = string(constants.RoleUser)
	testAdminRole = string(constants.RoleAdmin)
)

func TestListProducts_Success(t *testing.T) {
	repo := new(MockProductRepository)
	uc := newTestProductUseCase(repo)

	req := &dto.ListProductsRequest{
		Page:  1,
		Limit: 10,
	}

	productList := &domain.ProductList{
		Products: []*domain.Product{
			{
				Model:   domain.Model{ID: "prod-1", CreatedAt: time.Now(), UpdatedAt: time.Now()},
				Name:    "Test Product",
				Price:   10.0,
				Stock:   50,
				OwnerID: testUserID,
			},
		},
		Page:       1,
		Limit:      10,
		Total:      1,
		TotalPages: 1,
	}

	repo.On("FindAll", mock.Anything, req).
		Return(productList, nil)

	resp, err := uc.ListProducts(context.Background(), testUserID, testUserRole, req)

	assert.NoError(t, err)
	assert.NotNil(t, resp)
	assert.Equal(t, 1, len(resp.Data))
	repo.AssertExpectations(t)
}

func TestListProducts_Error(t *testing.T) {
	repo := new(MockProductRepository)
	uc := newTestProductUseCase(repo)

	req := &dto.ListProductsRequest{
		Page:  1,
		Limit: 10,
	}

	repo.On("FindAll", mock.Anything, req).
		Return(nil, errors.New("db error"))

	resp, err := uc.ListProducts(context.Background(), testUserID, testUserRole, req)

	assert.Error(t, err)
	assert.Nil(t, resp)
	repo.AssertExpectations(t)
}

func TestListProducts_NonAdminOwnerFilterOverride(t *testing.T) {
	repo := new(MockProductRepository)
	uc := newTestProductUseCase(repo)

	req := &dto.ListProductsRequest{
		Page:    1,
		Limit:   10,
		OwnerID: "550e8400-e29b-41d4-a716-446655440999",
	}

	productList := &domain.ProductList{
		Products:   []*domain.Product{},
		Page:       1,
		Limit:      10,
		Total:      0,
		TotalPages: 0,
	}

	repo.On("FindAll", mock.Anything, mock.MatchedBy(func(r *dto.ListProductsRequest) bool {
		return r != nil && r.OwnerID == testUserID
	})).Return(productList, nil)

	resp, err := uc.ListProducts(context.Background(), testUserID, testUserRole, req)

	assert.NoError(t, err)
	assert.NotNil(t, resp)
	repo.AssertExpectations(t)
}

func TestUpdateProduct_Success(t *testing.T) {
	repo := new(MockProductRepository)
	uc := newTestProductUseCase(repo)

	testProduct := &domain.Product{
		Model:   domain.Model{ID: "prod-1"},
		Name:    "Old Name",
		Price:   10.0,
		Stock:   50,
		OwnerID: testUserID,
	}

	newName := "New Name"
	newPrice := 20.0
	req := &dto.UpdateProductRequest{
		Name:  newName,
		Price: newPrice,
	}

	repo.On("FindByID", mock.Anything, "prod-1", mock.Anything).Return(testProduct, nil)
	repo.On("ExistsByNameAndOwner", mock.Anything, req.Name, testUserID).Return(false, nil)
	repo.On("Update", mock.Anything, mock.AnythingOfType("*domain.Product")).Return(nil)
	repo.On("FindByIDWithDetails", mock.Anything, "prod-1", mock.Anything).
		Return(testProduct, []*domain.ProductVariant{}, []*domain.ProductAttribute{}, nil)

	response, err := uc.UpdateProduct(context.Background(), testUserID, testUserRole, "prod-1", req)

	assert.NoError(t, err)
	assert.NotNil(t, response)
	assert.Equal(t, req.Name, response.Name)
	assert.Equal(t, dto.PriceRange{Min: req.Price, Max: req.Price, Display: fmt.Sprintf("$%.2f", req.Price)}, response.Price)

	repo.AssertExpectations(t)
}

func TestUpdateProduct_NameAlreadyUsed(t *testing.T) {
	repo := new(MockProductRepository)
	uc := newTestProductUseCase(repo)

	testProduct := &domain.Product{
		Model:   domain.Model{ID: "prod-1"},
		Name:    "Old Name",
		OwnerID: testUserID,
	}

	existName := "Existing Name"
	req := &dto.UpdateProductRequest{
		Name: existName,
	}

	repo.On("FindByID", mock.Anything, "prod-1", mock.Anything).Return(testProduct, nil)
	repo.On("ExistsByNameAndOwner", mock.Anything, req.Name, testUserID).Return(true, nil)

	response, err := uc.UpdateProduct(context.Background(), testUserID, testUserRole, "prod-1", req)

	assert.Error(t, err)
	assert.Nil(t, response)
	assert.Equal(t, domain.ErrProductNameAlreadyUsed, err)

	repo.AssertExpectations(t)
}

func TestUpdateProduct_AccessDenied(t *testing.T) {
	repo := new(MockProductRepository)
	uc := newTestProductUseCase(repo)

	testProduct := &domain.Product{
		Model:   domain.Model{ID: "prod-1"},
		Name:    "Product",
		OwnerID: "different-user-id",
	}

	newName := "New Name"
	req := &dto.UpdateProductRequest{
		Name: newName,
	}

	repo.On("FindByID", mock.Anything, "prod-1", mock.Anything).Return(testProduct, nil)

	response, err := uc.UpdateProduct(context.Background(), testUserID, testUserRole, "prod-1", req)

	assert.Error(t, err)
	assert.Nil(t, response)
	assert.Equal(t, usecase.ErrAccessDenied, err)

	repo.AssertExpectations(t)
}

func TestUpdateProduct_DirectStockUpdateBlockedForVariantProduct(t *testing.T) {
	repo := new(MockProductRepository)
	uc := newTestProductUseCase(repo)

	testProduct := &domain.Product{
		Model:      domain.Model{ID: "prod-1"},
		Name:       "Variant Product",
		Price:      100.0,
		Stock:      45,
		OwnerID:    testUserID,
		HasVariant: true,
	}

	stock := 10
	req := &dto.UpdateProductRequest{
		Stock: &stock,
	}

	repo.On("FindByID", mock.Anything, "prod-1", mock.Anything).Return(testProduct, nil)

	response, err := uc.UpdateProduct(context.Background(), testUserID, testUserRole, "prod-1", req)

	assert.Error(t, err)
	assert.Nil(t, response)
	assert.Contains(t, err.Error(), "variant")

	repo.AssertExpectations(t)
}

func TestUpdateStock_Success(t *testing.T) {
	repo := new(MockProductRepository)
	uc := newTestProductUseCase(repo)

	testProduct := &domain.Product{
		Model:   domain.Model{ID: "prod-1"},
		Name:    "Test Product",
		Stock:   50,
		OwnerID: testUserID,
	}

	req := &dto.UpdateStockRequest{
		ID:    "prod-1",
		Stock: 40,
	}

	repo.On("FindByID", mock.Anything, req.ID, mock.Anything).Return(testProduct, nil)
	repo.On("UpdateStock", mock.Anything, req.ID, 10).Return(nil)
	repo.On("Update", mock.Anything, mock.AnythingOfType("*domain.Product")).Return(nil).Maybe()

	response, err := uc.UpdateStock(context.Background(), testUserID, testUserRole, req)

	assert.NoError(t, err)
	assert.NotNil(t, response)
	assert.Equal(t, 10, response.Stock)

	repo.AssertExpectations(t)
}

func TestUpdateStock_InsufficientStock(t *testing.T) {
	repo := new(MockProductRepository)
	uc := newTestProductUseCase(repo)

	testProduct := &domain.Product{
		Model:   domain.Model{ID: "prod-1"},
		Name:    "Test Product",
		Stock:   5,
		OwnerID: testUserID,
	}

	req := &dto.UpdateStockRequest{
		ID:    "prod-1",
		Stock: 10,
	}

	repo.On("FindByID", mock.Anything, req.ID, mock.Anything).Return(testProduct, nil)

	response, err := uc.UpdateStock(context.Background(), testUserID, testUserRole, req)

	assert.Error(t, err)
	assert.Nil(t, response)
	assert.EqualError(t, err, "insufficient stock")

	repo.AssertExpectations(t)
}

func TestUpdateStock_ProductWithVariants_RequiresVariantID(t *testing.T) {
	repo := new(MockProductRepository)
	uc := newTestProductUseCase(repo)

	testProduct := &domain.Product{
		Model:      domain.Model{ID: "prod-1"},
		Name:       "Variant Product",
		Stock:      50,
		OwnerID:    testUserID,
		HasVariant: true,
	}

	req := &dto.UpdateStockRequest{
		ID:    "prod-1",
		Stock: 10,
	}

	repo.On("FindByID", mock.Anything, req.ID, mock.Anything).Return(testProduct, nil)

	response, err := uc.UpdateStock(context.Background(), testUserID, testUserRole, req)

	assert.Error(t, err)
	assert.Nil(t, response)
	assert.Contains(t, err.Error(), "variant")

	repo.AssertExpectations(t)
}

func TestUpdateStock_ProductWithVariants_VariantMustBelongToProduct(t *testing.T) {
	repo := new(MockProductRepository)
	uc := newTestProductUseCase(repo)

	testProduct := &domain.Product{
		Model:      domain.Model{ID: "prod-1"},
		Name:       "Variant Product",
		Stock:      50,
		OwnerID:    testUserID,
		HasVariant: true,
	}

	req := &dto.UpdateStockRequest{
		ID:        "prod-1",
		VariantID: "var-2",
		Stock:     10,
	}

	repo.On("FindByID", mock.Anything, req.ID, mock.Anything).Return(testProduct, nil)
	repo.On("FindByIDWithDetails", mock.Anything, req.ID, mock.Anything).
		Return(testProduct, []*domain.ProductVariant{
			{ID: "var-1", ProductID: "prod-1", StockQuantity: 30, IsActive: true},
		}, []*domain.ProductAttribute{}, nil)

	response, err := uc.UpdateStock(context.Background(), testUserID, testUserRole, req)

	assert.Error(t, err)
	assert.Nil(t, response)
	assert.Contains(t, err.Error(), "variant")

	repo.AssertExpectations(t)
}

func TestUpdateStock_ProductWithVariants_Success(t *testing.T) {
	repo := new(MockProductRepository)
	uc := newTestProductUseCase(repo)

	testProduct := &domain.Product{
		Model:      domain.Model{ID: "prod-1"},
		Name:       "Variant Product",
		Stock:      50,
		OwnerID:    testUserID,
		HasVariant: true,
	}

	req := &dto.UpdateStockRequest{
		ID:        "prod-1",
		VariantID: "var-1",
		Stock:     10,
	}

	repo.On("FindByID", mock.Anything, req.ID, mock.Anything).Return(testProduct, nil)
	repo.On("FindByIDWithDetails", mock.Anything, req.ID, mock.Anything).
		Return(testProduct, []*domain.ProductVariant{
			{ID: "var-1", ProductID: "prod-1", StockQuantity: 30, IsActive: true},
			{ID: "var-2", ProductID: "prod-1", StockQuantity: 15, IsActive: true},
		}, []*domain.ProductAttribute{}, nil)
	repo.On("UpdateVariantStockAndSyncProduct", mock.Anything, req.ID, req.VariantID, 20).Return(35, nil)

	response, err := uc.UpdateStock(context.Background(), testUserID, testUserRole, req)

	assert.NoError(t, err)
	assert.NotNil(t, response)
	assert.Equal(t, 35, response.Stock)

	repo.AssertExpectations(t)
}

func TestGenerateUUID(t *testing.T) {
	uuidStr := usecase.GenerateUUID()
	assert.NotEmpty(t, uuidStr)
	assert.Len(t, uuidStr, 36)
}

func TestValidateStock(t *testing.T) {
	err1 := usecase.ValidateStock(10)
	assert.NoError(t, err1)

	err2 := usecase.ValidateStock(0)
	assert.NoError(t, err2)

	err3 := usecase.ValidateStock(-5)
	assert.Error(t, err3)
	assert.EqualError(t, err3, "stock cannot be negative")
}

func TestUpdateProduct_NotFoundError(t *testing.T) {
	repo := new(MockProductRepository)
	uc := newTestProductUseCase(repo)

	req := &dto.UpdateProductRequest{}
	repo.On("FindByID", mock.Anything, "prod-1", mock.Anything).Return(nil, errors.New("not found"))

	response, err := uc.UpdateProduct(context.Background(), testUserID, testUserRole, "prod-1", req)

	assert.Error(t, err)
	assert.Nil(t, response)
	repo.AssertExpectations(t)
}

func TestDeleteProduct_NotFoundError(t *testing.T) {
	repo := new(MockProductRepository)
	uc := newTestProductUseCase(repo)

	req := &dto.DeleteProductRequest{ID: "prod-1"}
	repo.On("FindByID", mock.Anything, req.ID, mock.Anything).Return(nil, errors.New("not found"))

	response, err := uc.DeleteProduct(context.Background(), testUserID, testUserRole, req)

	assert.Error(t, err)
	assert.Nil(t, response)
	repo.AssertExpectations(t)
}

func TestRestoreProduct_RestoreError(t *testing.T) {
	repo := new(MockProductRepository)
	uc := newTestProductUseCase(repo)

	req := &dto.RestoreProductRequest{ID: "prod-1"}
	product := &domain.Product{
		Model:   domain.Model{ID: "prod-1"},
		OwnerID: testUserID,
	}
	repo.On("FindByID", mock.Anything, req.ID, mock.AnythingOfType("*domain.ParanoidOptions")).Return(product, nil)
	repo.On("Restore", mock.Anything, req.ID).Return(errors.New("restore failed"))

	response, err := uc.RestoreProduct(context.Background(), testUserID, testUserRole, req)

	assert.Error(t, err)
	assert.Nil(t, response)
	repo.AssertExpectations(t)
}

func TestCreateProduct_CreateError(t *testing.T) {
	repo := new(MockProductRepository)
	uc := newTestProductUseCase(repo)

	req := &dto.CreateProductRequest{
		Name:  "Error Product",
		Price: 10.0,
	}

	repo.On("ExistsByNameAndOwner", mock.Anything, req.Name, testUserID).Return(false, nil)
	repo.On(
		"CreateWithDetails",
		mock.Anything,
		mock.AnythingOfType("*domain.Product"),
		mock.AnythingOfType("[]*domain.ProductAttribute"),
		mock.AnythingOfType("[]*domain.ProductVariant"),
	).Return(errors.New("create error"))

	response, err := uc.CreateProduct(context.Background(), testUserID, req)

	assert.Error(t, err)
	assert.Nil(t, response)
	repo.AssertExpectations(t)
}

func TestUpdateProduct_UpdateError(t *testing.T) {
	repo := new(MockProductRepository)
	uc := newTestProductUseCase(repo)

	testProduct := &domain.Product{Model: domain.Model{ID: "prod-1"}, OwnerID: testUserID}
	name := "New Name"
	req := &dto.UpdateProductRequest{
		Name: name,
	}

	repo.On("FindByID", mock.Anything, "prod-1", mock.Anything).Return(testProduct, nil)
	repo.On("ExistsByNameAndOwner", mock.Anything, req.Name, testUserID).Return(false, nil)
	repo.On("Update", mock.Anything, mock.AnythingOfType("*domain.Product")).Return(errors.New("update error"))

	response, err := uc.UpdateProduct(context.Background(), testUserID, testUserRole, "prod-1", req)

	assert.Error(t, err)
	assert.Nil(t, response)
	repo.AssertExpectations(t)
}

func TestUpdateStock_UpdateStockError(t *testing.T) {
	repo := new(MockProductRepository)
	uc := newTestProductUseCase(repo)

	testProduct := &domain.Product{Model: domain.Model{ID: "prod-1"}, Stock: 50, OwnerID: testUserID}
	req := &dto.UpdateStockRequest{ID: "prod-1", Stock: 10}

	repo.On("FindByID", mock.Anything, req.ID, mock.Anything).Return(testProduct, nil)
	repo.On("UpdateStock", mock.Anything, req.ID, 40).Return(errors.New("update stock error"))

	response, err := uc.UpdateStock(context.Background(), testUserID, testUserRole, req)

	assert.Error(t, err)
	assert.Nil(t, response)
	repo.AssertExpectations(t)
}

func TestGetProduct_NotFoundError(t *testing.T) {
	repo := new(MockProductRepository)
	uc := newTestProductUseCase(repo)

	req := &dto.GetProductRequest{ID: "prod-1"}
	repo.On("FindByIDWithDetails", mock.Anything, req.ID, mock.Anything).
		Return(nil, nil, nil, errors.New("not found"))

	response, err := uc.GetProduct(context.Background(), testUserID, testUserRole, req)

	assert.Error(t, err)
	assert.Nil(t, response)
	repo.AssertExpectations(t)
}

func TestGetProduct_AccessDenied(t *testing.T) {
	repo := new(MockProductRepository)
	uc := newTestProductUseCase(repo)

	testProduct := &domain.Product{
		Model:   domain.Model{ID: "prod-1"},
		OwnerID: "different-user-id",
	}

	req := &dto.GetProductRequest{ID: "prod-1"}
	repo.On("FindByIDWithDetails", mock.Anything, req.ID, mock.Anything).
		Return(testProduct, nil, nil, nil)

	response, err := uc.GetProduct(context.Background(), testUserID, testUserRole, req)

	assert.Error(t, err)
	assert.Nil(t, response)
	assert.Equal(t, usecase.ErrAccessDenied, err)

	repo.AssertExpectations(t)
}

func TestGetProduct_AdminCanAccessAny(t *testing.T) {
	repo := new(MockProductRepository)
	uc := newTestProductUseCase(repo)

	testProduct := &domain.Product{
		Model:   domain.Model{ID: "prod-1", CreatedAt: time.Now(), UpdatedAt: time.Now()},
		Name:    "Test Product",
		Price:   10.0,
		Stock:   50,
		OwnerID: "different-user-id",
	}

	req := &dto.GetProductRequest{ID: "prod-1"}
	repo.On("FindByIDWithDetails", mock.Anything, req.ID, mock.Anything).
		Return(testProduct, nil, nil, nil)

	response, err := uc.GetProduct(context.Background(), testAdminID, testAdminRole, req)

	assert.NoError(t, err)
	assert.NotNil(t, response)
	assert.Equal(t, testProduct.Name, response.Name)

	repo.AssertExpectations(t)
}

func TestUpdateStock_NotFoundError(t *testing.T) {
	repo := new(MockProductRepository)
	uc := newTestProductUseCase(repo)

	req := &dto.UpdateStockRequest{ID: "prod-1", Stock: 10}
	repo.On("FindByID", mock.Anything, req.ID, mock.Anything).Return(nil, errors.New("not found"))

	response, err := uc.UpdateStock(context.Background(), testUserID, testUserRole, req)

	assert.Error(t, err)
	assert.Nil(t, response)
	repo.AssertExpectations(t)
}

func TestCreateProduct_ExistsByNameError(t *testing.T) {
	repo := new(MockProductRepository)
	uc := newTestProductUseCase(repo)

	req := &dto.CreateProductRequest{
		Name:  "Test Product",
		Price: 10.0,
	}

	repo.On("ExistsByNameAndOwner", mock.Anything, req.Name, testUserID).Return(false, errors.New("db error"))

	response, err := uc.CreateProduct(context.Background(), testUserID, req)

	assert.Error(t, err)
	assert.Nil(t, response)
	repo.AssertExpectations(t)
}

func TestCreateProduct_WithAttributesAndVariants_PersistsAndReturnsDetail(t *testing.T) {
	dbName := fmt.Sprintf("file:test_usecase_create_with_details_%s.db?mode=memory&cache=shared", uuid.NewString())
	db, err := gorm.Open(sqlite.Open(dbName), &gorm.Config{})
	require.NoError(t, err)
	require.NoError(t, db.AutoMigrate(&domain.Product{}, &domain.ProductVariant{}, &domain.ProductAttribute{}))

	sqlDB, err := db.DB()
	require.NoError(t, err)
	defer sqlDB.Close()

	productRepo := repository.NewProductRepository(db)
	uc := usecase.NewProductUseCase(productRepo, nil, usecase.Config{ServiceName: "product-service-test"}, zap.NewNop())

	ownerID := uuid.NewString()
	variantPrice1 := 399.99
	variantPrice2 := 429.99

	req := &dto.CreateProductRequest{
		OwnerID: ownerID,
		Name:    "Galaxy Watch 6 Classic",
		Price:   399.99,
		Stock:   100,
		Attributes: []*dto.CreateAttributeRequest{
			{Name: "Color", Values: []string{"Black", "Silver"}, DisplayOrder: 1},
			{Name: "Strap Size", Values: []string{"S/M", "M/L"}, DisplayOrder: 2},
		},
		Variants: []*dto.CreateVariantRequest{
			{
				SKU:   "GW6-BLK-SM",
				Price: &variantPrice1,
				Stock: 30,
				AttributeValues: map[string]string{
					"Color":      "Black",
					"Strap Size": "S/M",
				},
			},
			{
				SKU:   "GW6-SLV-ML",
				Price: &variantPrice2,
				Stock: 15,
				AttributeValues: map[string]string{
					"Color":      "Silver",
					"Strap Size": "M/L",
				},
			},
		},
	}

	created, err := uc.CreateProduct(context.Background(), ownerID, req)
	require.NoError(t, err)
	require.NotNil(t, created)

	assert.True(t, created.HasVariant)
	assert.Len(t, created.Attributes, 2)
	assert.Len(t, created.Variants, 2)
	assert.True(t, created.Variants[0].IsActive)
	assert.True(t, created.Variants[1].IsActive)
	assert.Equal(t, 45, created.Stock)
	assert.Equal(t, 399.99, created.Price.Min)
	assert.Equal(t, 429.99, created.Price.Max)

	detail, err := uc.GetProduct(context.Background(), ownerID, testUserRole, &dto.GetProductRequest{ID: created.ID})
	require.NoError(t, err)
	require.NotNil(t, detail)
	assert.Len(t, detail.Attributes, 2)
	assert.Len(t, detail.Variants, 2)
	assert.Equal(t, 45, detail.Stock)
	assert.Equal(t, 399.99, detail.Price.Min)
	assert.Equal(t, 429.99, detail.Price.Max)
}

func TestUpdateProduct_WithVariants_ReplacesDetailsAndSyncsStock(t *testing.T) {
	dbName := fmt.Sprintf("file:test_usecase_update_with_details_%s.db?mode=memory&cache=shared", uuid.NewString())
	db, err := gorm.Open(sqlite.Open(dbName), &gorm.Config{})
	require.NoError(t, err)
	require.NoError(t, db.AutoMigrate(&domain.Product{}, &domain.ProductVariant{}, &domain.ProductAttribute{}))

	sqlDB, err := db.DB()
	require.NoError(t, err)
	defer sqlDB.Close()

	productRepo := repository.NewProductRepository(db)
	uc := usecase.NewProductUseCase(productRepo, nil, usecase.Config{ServiceName: "product-service-test"}, zap.NewNop())

	ownerID := uuid.NewString()
	initialVariantPrice1 := 100.0
	initialVariantPrice2 := 120.0

	created, err := uc.CreateProduct(context.Background(), ownerID, &dto.CreateProductRequest{
		OwnerID: ownerID,
		Name:    "Initial Product",
		Price:   99.0,
		Stock:   999, // should be ignored because variants exist
		Attributes: []*dto.CreateAttributeRequest{
			{Name: "Color", Values: []string{"Black", "Silver"}, DisplayOrder: 1},
		},
		Variants: []*dto.CreateVariantRequest{
			{SKU: "INIT-BLK", Price: &initialVariantPrice1, Stock: 30, AttributeValues: map[string]string{"Color": "Black"}},
			{SKU: "INIT-SLV", Price: &initialVariantPrice2, Stock: 15, AttributeValues: map[string]string{"Color": "Silver"}},
		},
	})
	require.NoError(t, err)
	require.NotNil(t, created)
	assert.Equal(t, 45, created.Stock)

	newStock := 777
	updateVariantPrice1 := 149.0
	updateVariantPrice2 := 159.0
	updated, err := uc.UpdateProduct(context.Background(), ownerID, testUserRole, created.ID, &dto.UpdateProductRequest{
		Name:  "Updated Product",
		Price: 149.0,
		Stock: &newStock, // ignored because variants are provided
		Attributes: []*dto.CreateAttributeRequest{
			{Name: "Color", Values: []string{"Blue", "Green"}, DisplayOrder: 1},
		},
		Variants: []*dto.CreateVariantRequest{
			{SKU: "UPD-BLU", Price: &updateVariantPrice1, Stock: 5, AttributeValues: map[string]string{"Color": "Blue"}},
			{SKU: "UPD-GRN", Price: &updateVariantPrice2, Stock: 10, AttributeValues: map[string]string{"Color": "Green"}},
		},
	})
	require.NoError(t, err)
	require.NotNil(t, updated)

	assert.Equal(t, "Updated Product", updated.Name)
	assert.Equal(t, 15, updated.Stock)
	assert.True(t, updated.HasVariant)
	assert.Len(t, updated.Attributes, 1)
	assert.Len(t, updated.Variants, 2)
	assert.Equal(t, 149.0, updated.Price.Min)
	assert.Equal(t, 159.0, updated.Price.Max)

	detail, err := uc.GetProduct(context.Background(), ownerID, testUserRole, &dto.GetProductRequest{ID: created.ID})
	require.NoError(t, err)
	require.NotNil(t, detail)
	assert.Equal(t, 15, detail.Stock)
	assert.Len(t, detail.Variants, 2)
	assert.Equal(t, "UPD-BLU", detail.Variants[0].SKU)
	assert.Equal(t, "UPD-GRN", detail.Variants[1].SKU)
}
