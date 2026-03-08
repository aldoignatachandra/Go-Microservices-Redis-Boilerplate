// Package usecase provides tests for the product use case.
package usecase_test

import (
	"context"
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

func (m *MockProductRepository) Update(ctx context.Context, product *domain.Product) error {
	args := m.Called(ctx, product)
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
		Name:       "Test Product",
		Price:      29.99,
		Stock:      100,
		HasVariant: false,
	}

	repo.On("ExistsByNameAndOwner", mock.Anything, req.Name, testUserID).Return(false, nil)
	repo.On("Create", mock.Anything, mock.AnythingOfType("*domain.Product")).Return(nil)

	response, err := uc.CreateProduct(context.Background(), testUserID, req)

	assert.NoError(t, err)
	assert.NotNil(t, response)
	assert.Equal(t, req.Name, response.Name)
	assert.Equal(t, req.Price, response.Price)

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

	repo.On("FindByID", mock.Anything, req.ID, mock.Anything).Return(testProduct, nil)

	response, err := uc.GetProduct(context.Background(), testUserID, testUserRole, req)

	assert.NoError(t, err)
	assert.NotNil(t, response)
	assert.Equal(t, testProduct.Name, response.Name)

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
