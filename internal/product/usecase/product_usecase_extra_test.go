package usecase_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"

	"github.com/ignata/go-microservices-boilerplate/internal/product/domain"
	"github.com/ignata/go-microservices-boilerplate/internal/product/dto"
	"github.com/ignata/go-microservices-boilerplate/internal/product/usecase"
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
				Model: domain.Model{ID: "prod-1", CreatedAt: time.Now(), UpdatedAt: time.Now()},
				Name:  "Test Product",
			},
		},
		Page:       1,
		Limit:      10,
		Total:      1,
		TotalPages: 1,
	}

	repo.On("FindAll", mock.Anything, req).
		Return(productList, nil)

	resp, err := uc.ListProducts(context.Background(), req)

	assert.NoError(t, err)
	assert.NotNil(t, resp)
	assert.Equal(t, 1, len(resp.Products))
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

	resp, err := uc.ListProducts(context.Background(), req)

	assert.Error(t, err)
	assert.Nil(t, resp)
	repo.AssertExpectations(t)
}

func TestUpdateProduct_Success(t *testing.T) {
	repo := new(MockProductRepository)
	uc := newTestProductUseCase(repo)

	testProduct := &domain.Product{
		Model: domain.Model{ID: "prod-1"},
		Name:  "Old Name",
		Price: 10.0,
		Stock: 50,
	}

	newName := "New Name"
	newPrice := 20.0
	req := &dto.UpdateProductRequest{
		Name:  &newName,
		Price: &newPrice,
	}

	repo.On("FindByID", mock.Anything, "prod-1", mock.Anything).Return(testProduct, nil)
	repo.On("ExistsByName", mock.Anything, *req.Name).Return(false, nil)
	repo.On("Update", mock.Anything, mock.AnythingOfType("*domain.Product")).Return(nil)

	response, err := uc.UpdateProduct(context.Background(), "prod-1", req)

	assert.NoError(t, err)
	assert.NotNil(t, response)
	assert.Equal(t, *req.Name, response.Name)
	assert.Equal(t, *req.Price, response.Price)

	repo.AssertExpectations(t)
}

func TestUpdateProduct_NameAlreadyUsed(t *testing.T) {
	repo := new(MockProductRepository)
	uc := newTestProductUseCase(repo)

	testProduct := &domain.Product{
		Model: domain.Model{ID: "prod-1"},
		Name:  "Old Name",
	}

	existName := "Existing Name"
	req := &dto.UpdateProductRequest{
		Name: &existName,
	}

	repo.On("FindByID", mock.Anything, "prod-1", mock.Anything).Return(testProduct, nil)
	repo.On("ExistsByName", mock.Anything, *req.Name).Return(true, nil)

	response, err := uc.UpdateProduct(context.Background(), "prod-1", req)

	assert.Error(t, err)
	assert.Nil(t, response)
	assert.Equal(t, domain.ErrProductNameAlreadyUsed, err)

	repo.AssertExpectations(t)
}

func TestUpdateStock_Success(t *testing.T) {
	repo := new(MockProductRepository)
	uc := newTestProductUseCase(repo)

	testProduct := &domain.Product{
		Model: domain.Model{ID: "prod-1"},
		Name:  "Test Product",
		Stock: 50,
	}

	req := &dto.UpdateStockRequest{
		ID:    "prod-1",
		Stock: 40,
	}

	repo.On("FindByID", mock.Anything, req.ID, mock.Anything).Return(testProduct, nil)
	repo.On("UpdateStock", mock.Anything, req.ID, 10).Return(nil)
	repo.On("Update", mock.Anything, mock.AnythingOfType("*domain.Product")).Return(nil).Maybe()

	response, err := uc.UpdateStock(context.Background(), req)

	assert.NoError(t, err)
	assert.NotNil(t, response)
	assert.Equal(t, 10, response.Stock)

	repo.AssertExpectations(t)
}

func TestUpdateStock_InsufficientStock(t *testing.T) {
	repo := new(MockProductRepository)
	uc := newTestProductUseCase(repo)

	testProduct := &domain.Product{
		Model: domain.Model{ID: "prod-1"},
		Name:  "Test Product",
		Stock: 5,
	}

	req := &dto.UpdateStockRequest{
		ID:    "prod-1",
		Stock: 10, // Stock to reduce by (10 > 5, so insufficient)
	}

	repo.On("FindByID", mock.Anything, req.ID, mock.Anything).Return(testProduct, nil)

	response, err := uc.UpdateStock(context.Background(), req)

	assert.Error(t, err)
	assert.Nil(t, response)
	assert.EqualError(t, err, "insufficient stock")

	repo.AssertExpectations(t)
}

func TestGenerateUUID(t *testing.T) {
	uuidStr := usecase.GenerateUUID()
	assert.NotEmpty(t, uuidStr)
	assert.Len(t, uuidStr, 36) // standard UUID length
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

	response, err := uc.UpdateProduct(context.Background(), "prod-1", req)

	assert.Error(t, err)
	assert.Nil(t, response)
	repo.AssertExpectations(t)
}

func TestDeleteProduct_NotFoundError(t *testing.T) {
	repo := new(MockProductRepository)
	uc := newTestProductUseCase(repo)

	req := &dto.DeleteProductRequest{ID: "prod-1"}
	repo.On("FindByID", mock.Anything, req.ID, mock.Anything).Return(nil, errors.New("not found"))

	response, err := uc.DeleteProduct(context.Background(), req)

	assert.Error(t, err)
	assert.Nil(t, response)
	repo.AssertExpectations(t)
}

func TestRestoreProduct_RestoreError(t *testing.T) {
	repo := new(MockProductRepository)
	uc := newTestProductUseCase(repo)

	req := &dto.RestoreProductRequest{ID: "prod-1"}
	repo.On("Restore", mock.Anything, req.ID).Return(errors.New("restore failed"))

	response, err := uc.RestoreProduct(context.Background(), req)

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

	repo.On("ExistsByName", mock.Anything, req.Name).Return(false, nil)
	repo.On("Create", mock.Anything, mock.AnythingOfType("*domain.Product")).Return(errors.New("create error"))

	response, err := uc.CreateProduct(context.Background(), req)

	assert.Error(t, err)
	assert.Nil(t, response)
	repo.AssertExpectations(t)
}

func TestUpdateProduct_UpdateError(t *testing.T) {
	repo := new(MockProductRepository)
	uc := newTestProductUseCase(repo)

	testProduct := &domain.Product{Model: domain.Model{ID: "prod-1"}}
	name := "New Name"
	req := &dto.UpdateProductRequest{
		Name: &name,
	}

	repo.On("FindByID", mock.Anything, "prod-1", mock.Anything).Return(testProduct, nil)
	repo.On("ExistsByName", mock.Anything, *req.Name).Return(false, nil)
	repo.On("Update", mock.Anything, mock.AnythingOfType("*domain.Product")).Return(errors.New("update error"))

	response, err := uc.UpdateProduct(context.Background(), "prod-1", req)

	assert.Error(t, err)
	assert.Nil(t, response)
	repo.AssertExpectations(t)
}

func TestUpdateStock_UpdateStockError(t *testing.T) {
	repo := new(MockProductRepository)
	uc := newTestProductUseCase(repo)

	testProduct := &domain.Product{Model: domain.Model{ID: "prod-1"}, Stock: 50}
	req := &dto.UpdateStockRequest{ID: "prod-1", Stock: 10}

	repo.On("FindByID", mock.Anything, req.ID, mock.Anything).Return(testProduct, nil)
	// Expecting 50 - 10 = 40
	repo.On("UpdateStock", mock.Anything, req.ID, 40).Return(errors.New("update stock error"))

	response, err := uc.UpdateStock(context.Background(), req)

	assert.Error(t, err)
	assert.Nil(t, response)
	repo.AssertExpectations(t)
}

func TestGetProduct_NotFoundError(t *testing.T) {
	repo := new(MockProductRepository)
	uc := newTestProductUseCase(repo)

	req := &dto.GetProductRequest{ID: "prod-1"}
	repo.On("FindByID", mock.Anything, req.ID, mock.Anything).Return(nil, errors.New("not found"))

	response, err := uc.GetProduct(context.Background(), req)

	assert.Error(t, err)
	assert.Nil(t, response)
	repo.AssertExpectations(t)
}

func TestUpdateStock_NotFoundError(t *testing.T) {
	repo := new(MockProductRepository)
	uc := newTestProductUseCase(repo)

	req := &dto.UpdateStockRequest{ID: "prod-1", Stock: 10}
	repo.On("FindByID", mock.Anything, req.ID, mock.Anything).Return(nil, errors.New("not found"))

	response, err := uc.UpdateStock(context.Background(), req)

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

	repo.On("ExistsByName", mock.Anything, req.Name).Return(false, errors.New("db error"))

	response, err := uc.CreateProduct(context.Background(), req)

	assert.Error(t, err)
	assert.Nil(t, response)
	repo.AssertExpectations(t)
}
