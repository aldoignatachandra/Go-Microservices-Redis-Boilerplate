// Package usecase provides tests for the product use case.
package usecase_test

import (
	"context"
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"go.uber.org/zap"

	"github.com/ignata/go-microservices-boilerplate/internal/product/domain"
	"github.com/ignata/go-microservices-boilerplate/internal/product/dto"
	"github.com/ignata/go-microservices-boilerplate/internal/product/usecase"
	"github.com/ignata/go-microservices-boilerplate/pkg/eventbus"
)

// --- Mock Repository ---

type MockProductRepository struct {
	mock.Mock
}

func (m *MockProductRepository) Create(ctx context.Context, product *domain.Product) error {
	args := m.Called(ctx, product)
	return args.Error(0)
}

func (m *MockProductRepository) CreateWithDetails(
	ctx context.Context,
	product *domain.Product,
	attributes []*domain.ProductAttribute,
	variants []*domain.ProductVariant,
) error {
	args := m.Called(ctx, product, attributes, variants)
	return args.Error(0)
}

func (m *MockProductRepository) Update(ctx context.Context, product *domain.Product) error {
	args := m.Called(ctx, product)
	return args.Error(0)
}

func (m *MockProductRepository) UpdateWithDetails(
	ctx context.Context,
	product *domain.Product,
	attributes []*domain.ProductAttribute,
	variants []*domain.ProductVariant,
	replaceAttributes bool,
	replaceVariants bool,
) error {
	args := m.Called(ctx, product, attributes, variants, replaceAttributes, replaceVariants)
	return args.Error(0)
}

func (m *MockProductRepository) Delete(ctx context.Context, id string) error {
	args := m.Called(ctx, id)
	return args.Error(0)
}

func (m *MockProductRepository) HardDelete(ctx context.Context, id string) error {
	args := m.Called(ctx, id)
	return args.Error(0)
}

func (m *MockProductRepository) Restore(ctx context.Context, id string) error {
	args := m.Called(ctx, id)
	return args.Error(0)
}

func (m *MockProductRepository) FindByID(ctx context.Context, id string, opts *domain.ParanoidOptions) (*domain.Product, error) {
	args := m.Called(ctx, id, opts)
	if p, ok := args.Get(0).(*domain.Product); ok {
		return p, args.Error(1)
	}
	return nil, args.Error(1)
}

func (m *MockProductRepository) FindByIDWithDetails(
	ctx context.Context,
	id string,
	opts *domain.ParanoidOptions,
) (*domain.Product, []*domain.ProductVariant, []*domain.ProductAttribute, error) {
	args := m.Called(ctx, id, opts)
	var product *domain.Product
	if p, ok := args.Get(0).(*domain.Product); ok {
		product = p
	}
	var variants []*domain.ProductVariant
	if v, ok := args.Get(1).([]*domain.ProductVariant); ok {
		variants = v
	}
	var attributes []*domain.ProductAttribute
	if a, ok := args.Get(2).([]*domain.ProductAttribute); ok {
		attributes = a
	}
	return product, variants, attributes, args.Error(3)
}

func (m *MockProductRepository) FindAll(ctx context.Context, req *dto.ListProductsRequest) (*domain.ProductList, error) {
	args := m.Called(ctx, req)
	if list, ok := args.Get(0).(*domain.ProductList); ok {
		return list, args.Error(1)
	}
	return nil, args.Error(1)
}

func (m *MockProductRepository) ExistsByName(ctx context.Context, name string) (bool, error) {
	args := m.Called(ctx, name)
	return args.Bool(0), args.Error(1)
}

func (m *MockProductRepository) ExistsByNameAndOwner(ctx context.Context, name string, ownerID string) (bool, error) {
	args := m.Called(ctx, name, ownerID)
	return args.Bool(0), args.Error(1)
}

func (m *MockProductRepository) UpdateStock(ctx context.Context, id string, stock int) error {
	args := m.Called(ctx, id, stock)
	return args.Error(0)
}

func (m *MockProductRepository) UpdateVariantStockAndSyncProduct(
	ctx context.Context,
	productID string,
	variantID string,
	stock int,
) (int, error) {
	args := m.Called(ctx, productID, variantID, stock)
	return args.Int(0), args.Error(1)
}

// --- Mock Event Publisher ---

type MockEventPublisher struct {
	mock.Mock
}

func (m *MockEventPublisher) Publish(ctx context.Context, topic string, event *eventbus.Event) (string, error) {
	args := m.Called(ctx, topic, event)
	return args.String(0), args.Error(1)
}

func (m *MockEventPublisher) Close() error {
	args := m.Called()
	return args.Error(0)
}

// --- Helper ---

func newTestProductUseCase(repo *MockProductRepository) usecase.ProductUseCase {
	logger, _ := zap.NewDevelopment()
	eb := new(MockEventPublisher)
	eb.On("Publish", mock.Anything, mock.Anything, mock.Anything).Return("mock-event-id", nil).Maybe()
	return usecase.NewProductUseCase(
		repo,
		eb,
		usecase.Config{ServiceName: "product-service-test"},
		logger,
	)
}

// --- Tests ---

func TestCreateProduct_Success(t *testing.T) {
	repo := new(MockProductRepository)
	uc := newTestProductUseCase(repo)

	testUserID := "user-123"

	req := &dto.CreateProductRequest{
		Name:    "Test Product",
		Price:   29.99,
		Stock:   100,
		OwnerID: testUserID,
	}

	repo.On("ExistsByNameAndOwner", mock.Anything, req.Name, testUserID).Return(false, nil)
	repo.On(
		"CreateWithDetails",
		mock.Anything,
		mock.AnythingOfType("*domain.Product"),
		mock.AnythingOfType("[]*domain.ProductAttribute"),
		mock.AnythingOfType("[]*domain.ProductVariant"),
	).Return(nil)
	repo.On("FindByIDWithDetails", mock.Anything, mock.AnythingOfType("string"), mock.AnythingOfType("*domain.ParanoidOptions")).
		Return(&domain.Product{
			Model: domain.Model{ID: "prod-1"},
			Name:  req.Name,
			Price: req.Price,
			Stock: req.Stock,
		}, []*domain.ProductVariant{}, []*domain.ProductAttribute{}, nil)

	response, err := uc.CreateProduct(context.Background(), testUserID, req)

	assert.NoError(t, err)
	assert.NotNil(t, response)
	assert.Equal(t, req.Name, response.Name)
	assert.Equal(t, dto.PriceRange{Min: req.Price, Max: req.Price, Display: fmt.Sprintf("$%.2f", req.Price)}, response.Price)

	repo.AssertExpectations(t)
}

func TestCreateProduct_NameAlreadyUsed(t *testing.T) {
	repo := new(MockProductRepository)
	uc := newTestProductUseCase(repo)

	testUserID := "user-123"

	req := &dto.CreateProductRequest{
		Name:  "Existing Product",
		Price: 9.99,
	}

	repo.On("ExistsByNameAndOwner", mock.Anything, req.Name, testUserID).Return(true, nil)

	response, err := uc.CreateProduct(context.Background(), testUserID, req)

	assert.Error(t, err)
	assert.Nil(t, response)
	assert.Equal(t, domain.ErrProductNameAlreadyUsed, err)

	repo.AssertExpectations(t)
}

func TestGetProduct_Success(t *testing.T) {
	repo := new(MockProductRepository)
	uc := newTestProductUseCase(repo)

	testUserID := "user-123"
	testUserRole := "USER"

	testProduct := &domain.Product{
		Name:    "Found Product",
		Price:   19.99,
		Stock:   50,
		OwnerID: testUserID,
	}

	req := &dto.GetProductRequest{ID: "prod-1"}

	variants := []*domain.ProductVariant{
		{ID: "var-1", ProductID: "prod-1", SKU: "KEY-TKL-BROWN", Price: 89.99, StockQuantity: 20, IsActive: true},
		{ID: "var-2", ProductID: "prod-1", SKU: "KEY-FULL-RED", Price: 99.99, StockQuantity: 15, IsActive: true},
	}
	attributes := []*domain.ProductAttribute{
		{ID: "attr-1", ProductID: "prod-1", Name: "Layout", Values: []string{"TKL", "Full"}, DisplayOrder: 0},
	}
	repo.On("FindByIDWithDetails", mock.Anything, req.ID, mock.Anything).Return(testProduct, variants, attributes, nil)

	response, err := uc.GetProduct(context.Background(), testUserID, testUserRole, req)

	assert.NoError(t, err)
	assert.NotNil(t, response)
	assert.Equal(t, testProduct.Name, response.Name)
	assert.Equal(t, 89.99, response.Price.Min)
	assert.Equal(t, 99.99, response.Price.Max)
	assert.Len(t, response.Variants, 2)
	assert.Len(t, response.Attributes, 1)

	repo.AssertExpectations(t)
}

func TestDeleteProduct_SoftDelete(t *testing.T) {
	repo := new(MockProductRepository)
	uc := newTestProductUseCase(repo)

	testUserID := "user-123"
	testUserRole := "USER"

	testProduct := &domain.Product{
		Name:    "Delete Me",
		OwnerID: testUserID,
	}
	testProduct.ID = "prod-del-1"

	req := &dto.DeleteProductRequest{
		ID:    "prod-del-1",
		Force: false,
	}

	repo.On("FindByID", mock.Anything, req.ID, mock.Anything).Return(testProduct, nil)
	repo.On("Delete", mock.Anything, req.ID).Return(nil)

	response, err := uc.DeleteProduct(context.Background(), testUserID, testUserRole, req)

	assert.NoError(t, err)
	assert.NotNil(t, response)
	assert.True(t, response.Success)
	assert.Equal(t, "Product deleted successfully", response.Message)

	repo.AssertExpectations(t)
}

func TestDeleteProduct_HardDelete(t *testing.T) {
	repo := new(MockProductRepository)
	uc := newTestProductUseCase(repo)

	testUserID := "user-123"
	testUserRole := "USER"

	testProduct := &domain.Product{
		Name:    "Hard Delete Me",
		OwnerID: testUserID,
	}
	testProduct.ID = "prod-hard-del"

	req := &dto.DeleteProductRequest{
		ID:    "prod-hard-del",
		Force: true,
	}

	repo.On("FindByID", mock.Anything, req.ID, mock.Anything).Return(testProduct, nil)
	repo.On("HardDelete", mock.Anything, req.ID).Return(nil)

	response, err := uc.DeleteProduct(context.Background(), testUserID, testUserRole, req)

	assert.NoError(t, err)
	assert.NotNil(t, response)
	assert.True(t, response.Success)
	assert.Equal(t, "Product permanently deleted", response.Message)

	repo.AssertExpectations(t)
}

func TestRestoreProduct_Success(t *testing.T) {
	repo := new(MockProductRepository)
	uc := newTestProductUseCase(repo)

	testUserID := "user-123"
	testUserRole := "USER"

	restoredProduct := &domain.Product{
		Name:    "Restored Product",
		Price:   14.99,
		OwnerID: testUserID,
	}

	req := &dto.RestoreProductRequest{ID: "restore-me"}

	repo.On("Restore", mock.Anything, req.ID).Return(nil)
	repo.On("FindByID", mock.Anything, req.ID, mock.Anything).Return(restoredProduct, nil)

	response, err := uc.RestoreProduct(context.Background(), testUserID, testUserRole, req)

	assert.NoError(t, err)
	assert.NotNil(t, response)
	assert.Equal(t, restoredProduct.Name, response.Name)

	repo.AssertExpectations(t)
}
