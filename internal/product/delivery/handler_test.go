// Package delivery provides tests for HTTP handlers for the product service.
package delivery_test

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"github.com/ignata/go-microservices-boilerplate/internal/product/delivery"
	"github.com/ignata/go-microservices-boilerplate/internal/product/domain"
	"github.com/ignata/go-microservices-boilerplate/internal/product/dto"
	productusecase "github.com/ignata/go-microservices-boilerplate/internal/product/usecase"
	productusecasemocks "github.com/ignata/go-microservices-boilerplate/internal/product/usecase/mocks"
	"github.com/ignata/go-microservices-boilerplate/pkg/server"
	"github.com/ignata/go-microservices-boilerplate/pkg/utils"
)

// setupTestRouter creates a test router with Gin in test mode.
func setupTestRouter() *gin.Engine {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	return router
}

const routeTestJWTSecret = "test-secret-key-for-product-routes"

func generateBearerToken(t *testing.T, userID, role string) string {
	t.Helper()
	manager := utils.NewJWTManager(utils.JWTConfig{
		Secret:    routeTestJWTSecret,
		ExpiresIn: time.Hour,
	})
	token, err := manager.GenerateToken(userID, "test@example.com", role)
	require.NoError(t, err)
	return "Bearer " + token
}

// TestCreateProduct_Success tests successful product creation.
func TestCreateProduct_Success(t *testing.T) {
	// Arrange
	mockUseCase := new(productusecasemocks.ProductUseCase)
	handler := delivery.NewHandler(mockUseCase)
	router := setupTestRouter()

	expectedResponse := &dto.ProductResponse{
		ID:         "550e8400-e29b-41d4-a716-446655440001",
		Name:       "Test Product",
		Price:      dto.PriceRange{Min: 29.99, Max: 29.99, Display: fmt.Sprintf("$%.2f", 29.99)},
		Stock:      100,
		OwnerID:    "550e8400-e29b-41d4-a716-446655440000",
		HasVariant: false,
	}

	mockUseCase.On("CreateProduct", mock.Anything, mock.AnythingOfType("string"), mock.AnythingOfType("*dto.CreateProductRequest")).
		Return(expectedResponse, nil)

	// Act
	reqBody := map[string]interface{}{
		"name":    "Test Product",
		"price":   29.99,
		"stock":   100,
		"ownerId": "550e8400-e29b-41d4-a716-446655440000",
	}
	bodyBytes, _ := json.Marshal(reqBody)
	req, _ := http.NewRequest("POST", "/products", bytes.NewBuffer(bodyBytes))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	// Set user context
	router.Use(func(c *gin.Context) {
		c.Set("user_id", "550e8400-e29b-41d4-a716-446655440000")
		c.Next()
	})

	router.POST("/products", handler.CreateProduct)
	router.ServeHTTP(w, req)

	// Assert
	assert.Equal(t, http.StatusCreated, w.Code)

	var response map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(t, err)

	assert.True(t, response["success"].(bool))
	data := response["data"].(map[string]interface{})
	assert.Equal(t, "550e8400-e29b-41d4-a716-446655440001", data["id"])
	assert.Equal(t, "Test Product", data["name"])
	price := data["price"].(map[string]interface{})
	assert.Equal(t, 29.99, price["min"])
	assert.Equal(t, 29.99, price["max"])
	assert.Equal(t, "$29.99", price["display"])

	mockUseCase.AssertExpectations(t)
}

// TestCreateProduct_ValidationError tests product creation with invalid input.
func TestCreateProduct_ValidationError(t *testing.T) {
	tests := []struct {
		name           string
		requestBody    map[string]interface{}
		expectedStatus int
	}{
		{
			name: "invalid JSON",
			requestBody: map[string]interface{}{
				"name": "Invalid",
			},
			expectedStatus: http.StatusBadRequest,
		},
		{
			name:           "malformed JSON",
			requestBody:    nil,
			expectedStatus: http.StatusBadRequest,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Arrange
			mockUseCase := new(productusecasemocks.ProductUseCase)
			handler := delivery.NewHandler(mockUseCase)
			router := setupTestRouter()

			var bodyBytes []byte
			if tt.requestBody == nil {
				bodyBytes = []byte("{invalid json")
			} else {
				bodyBytes, _ = json.Marshal(tt.requestBody)
			}

			// Act
			req, _ := http.NewRequest("POST", "/products", bytes.NewBuffer(bodyBytes))
			req.Header.Set("Content-Type", "application/json")
			w := httptest.NewRecorder()

			router.POST("/products", handler.CreateProduct)
			router.ServeHTTP(w, req)

			// Assert
			assert.Equal(t, tt.expectedStatus, w.Code)
		})
	}
}

// TestCreateProduct_Conflict tests product creation when name already exists.
func TestCreateProduct_Conflict(t *testing.T) {
	// Arrange
	mockUseCase := new(productusecasemocks.ProductUseCase)
	handler := delivery.NewHandler(mockUseCase)
	router := setupTestRouter()

	mockUseCase.On("CreateProduct", mock.Anything, mock.AnythingOfType("string"), mock.AnythingOfType("*dto.CreateProductRequest")).
		Return(nil, domain.ErrProductNameAlreadyUsed)

	// Act
	reqBody := map[string]interface{}{
		"name":    "Existing Product",
		"price":   19.99,
		"stock":   50,
		"ownerId": "550e8400-e29b-41d4-a716-446655440000",
	}
	bodyBytes, _ := json.Marshal(reqBody)
	req, _ := http.NewRequest("POST", "/products", bytes.NewBuffer(bodyBytes))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	// Set user context
	router.Use(func(c *gin.Context) {
		c.Set("user_id", "550e8400-e29b-41d4-a716-446655440000")
		c.Next()
	})

	router.POST("/products", handler.CreateProduct)
	router.ServeHTTP(w, req)

	// Assert
	assert.Equal(t, http.StatusConflict, w.Code)

	var response map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(t, err)
	assert.False(t, response["success"].(bool))

	mockUseCase.AssertExpectations(t)
}

// TestGetProduct_Success tests successful product retrieval.
func TestGetProduct_Success(t *testing.T) {
	// Arrange
	mockUseCase := new(productusecasemocks.ProductUseCase)
	handler := delivery.NewHandler(mockUseCase)
	router := setupTestRouter()

	expectedResponse := &dto.ProductResponse{
		ID:         "550e8400-e29b-41d4-a716-446655440001",
		Name:       "Test Product",
		OwnerID:    "550e8400-e29b-41d4-a716-446655440000",
		HasVariant: false,
	}

	mockUseCase.On("GetProduct", mock.Anything, "", "", mock.AnythingOfType("*dto.GetProductRequest")).
		Return(expectedResponse, nil)

	// Act
	req, _ := http.NewRequest("GET", "/products/550e8400-e29b-41d4-a716-446655440001", nil)
	w := httptest.NewRecorder()

	router.GET("/products/:id", handler.GetProduct)
	router.ServeHTTP(w, req)

	// Assert
	assert.Equal(t, http.StatusOK, w.Code)

	var response map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(t, err)

	assert.True(t, response["success"].(bool))
	data := response["data"].(map[string]interface{})
	assert.Equal(t, "550e8400-e29b-41d4-a716-446655440001", data["id"])

	mockUseCase.AssertExpectations(t)
}

func TestGetProduct_AccessDenied(t *testing.T) {
	mockUseCase := new(productusecasemocks.ProductUseCase)
	handler := delivery.NewHandler(mockUseCase)
	router := setupTestRouter()

	mockUseCase.On("GetProduct", mock.Anything, "", "", mock.AnythingOfType("*dto.GetProductRequest")).
		Return(nil, productusecase.ErrAccessDenied)

	req, _ := http.NewRequest("GET", "/products/550e8400-e29b-41d4-a716-446655440001", nil)
	w := httptest.NewRecorder()

	router.GET("/products/:id", handler.GetProduct)
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusForbidden, w.Code)

	var response map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(t, err)
	assert.Equal(t, "forbidden: product access denied", response["message"])

	mockUseCase.AssertExpectations(t)
}

// TestGetProduct_NotFound tests product not found scenario.
func TestGetProduct_NotFound(t *testing.T) {
	// Arrange
	mockUseCase := new(productusecasemocks.ProductUseCase)
	handler := delivery.NewHandler(mockUseCase)
	router := setupTestRouter()

	mockUseCase.On("GetProduct", mock.Anything, "", "", mock.AnythingOfType("*dto.GetProductRequest")).
		Return(nil, domain.ErrProductNotFound)

	// Act
	req, _ := http.NewRequest("GET", "/products/550e8400-e29b-41d4-a716-446655440002", nil)
	w := httptest.NewRecorder()

	router.GET("/products/:id", handler.GetProduct)
	router.ServeHTTP(w, req)

	// Assert
	assert.Equal(t, http.StatusNotFound, w.Code)

	var response map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(t, err)

	assert.False(t, response["success"].(bool))

	mockUseCase.AssertExpectations(t)
}

// TestGetProduct_IncludeDeleted tests getting a deleted product.
func TestGetProduct_IncludeDeleted(t *testing.T) {
	// Arrange
	mockUseCase := new(productusecasemocks.ProductUseCase)
	handler := delivery.NewHandler(mockUseCase)
	router := setupTestRouter()

	expectedResponse := &dto.ProductResponse{
		ID:         "550e8400-e29b-41d4-a716-446655440001",
		Name:       "Deleted Product",
		HasVariant: false,
	}

	mockUseCase.On("GetProduct", mock.Anything, "", "", mock.MatchedBy(func(r *dto.GetProductRequest) bool {
		return r.ID == "550e8400-e29b-41d4-a716-446655440001" && r.IncludeDeleted == true
	})).Return(expectedResponse, nil)

	// Act
	req, _ := http.NewRequest("GET", "/products/550e8400-e29b-41d4-a716-446655440001?include_deleted=true", nil)
	w := httptest.NewRecorder()

	router.GET("/products/:id", handler.GetProduct)
	router.ServeHTTP(w, req)

	// Assert
	assert.Equal(t, http.StatusOK, w.Code)

	var response map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(t, err)

	assert.True(t, response["success"].(bool))

	mockUseCase.AssertExpectations(t)
}

// TestListProducts_Success tests successful product list retrieval.
func TestListProducts_Success(t *testing.T) {
	// Arrange
	mockUseCase := new(productusecasemocks.ProductUseCase)
	handler := delivery.NewHandler(mockUseCase)
	router := setupTestRouter()

	expectedResponse := &dto.ProductListResponse{
		Data: []*dto.ProductResponse{
			{
				ID:         "prod-1",
				Name:       "Product 1",
				Price:      dto.PriceRange{Min: 10.99, Max: 10.99, Display: fmt.Sprintf("$%.2f", 10.99)},
				Stock:      50,
				OwnerID:    "owner-1",
				HasVariant: false,
			},
			{
				ID:         "prod-2",
				Name:       "Product 2",
				Price:      dto.PriceRange{Min: 20.99, Max: 20.99, Display: fmt.Sprintf("$%.2f", 20.99)},
				Stock:      100,
				OwnerID:    "owner-1",
				HasVariant: false,
			},
		},
		Total:      2,
		Page:       1,
		Limit:      10,
		TotalPages: 1,
	}

	mockUseCase.On("ListProducts", mock.Anything, "", "", mock.AnythingOfType("*dto.ListProductsRequest")).
		Return(expectedResponse, nil)

	// Act
	req, _ := http.NewRequest("GET", "/products?page=1&limit=10", nil)
	w := httptest.NewRecorder()

	router.GET("/products", handler.ListProducts)
	router.ServeHTTP(w, req)

	// Assert
	assert.Equal(t, http.StatusOK, w.Code)

	var response map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(t, err)

	assert.True(t, response["success"].(bool))
	data := response["data"].(map[string]interface{})
	assert.NotNil(t, data["data"])

	mockUseCase.AssertExpectations(t)
}

// TestListProducts_WithFilters tests listing products with filters.
func TestListProducts_WithFilters(t *testing.T) {
	tests := []struct {
		name  string
		query string
		setup func(*productusecasemocks.ProductUseCase)
	}{
		{
			name:  "filter by search",
			query: "/products?search=laptop",
			setup: func(m *productusecasemocks.ProductUseCase) {
				m.On("ListProducts", mock.Anything, "", "", mock.MatchedBy(func(r *dto.ListProductsRequest) bool {
					return r.Search == "laptop"
				})).Return(&dto.ProductListResponse{}, nil)
			},
		},
		{
			name:  "include deleted",
			query: "/products?include_deleted=true",
			setup: func(m *productusecasemocks.ProductUseCase) {
				m.On("ListProducts", mock.Anything, "", "", mock.MatchedBy(func(r *dto.ListProductsRequest) bool {
					return r.IncludeDeleted == true
				})).Return(&dto.ProductListResponse{}, nil)
			},
		},
		{
			name:  "only deleted",
			query: "/products?only_deleted=true",
			setup: func(m *productusecasemocks.ProductUseCase) {
				m.On("ListProducts", mock.Anything, "", "", mock.MatchedBy(func(r *dto.ListProductsRequest) bool {
					return r.OnlyDeleted == true
				})).Return(&dto.ProductListResponse{}, nil)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Arrange
			mockUseCase := new(productusecasemocks.ProductUseCase)
			handler := delivery.NewHandler(mockUseCase)
			router := setupTestRouter()

			tt.setup(mockUseCase)

			// Act
			req, _ := http.NewRequest("GET", tt.query, nil)
			w := httptest.NewRecorder()

			router.GET("/products", handler.ListProducts)
			router.ServeHTTP(w, req)

			// Assert
			assert.Equal(t, http.StatusOK, w.Code)
			mockUseCase.AssertExpectations(t)
		})
	}
}

// TestUpdateProduct_Success tests successful product update.
func TestUpdateProduct_Success(t *testing.T) {
	// Arrange
	mockUseCase := new(productusecasemocks.ProductUseCase)
	handler := delivery.NewHandler(mockUseCase)
	router := setupTestRouter()

	expectedResponse := &dto.ProductResponse{
		ID:         "550e8400-e29b-41d4-a716-446655440001",
		Name:       "Updated Product",
		Price:      dto.PriceRange{Min: 39.99, Max: 39.99, Display: fmt.Sprintf("$%.2f", 39.99)},
		Stock:      100,
		OwnerID:    "550e8400-e29b-41d4-a716-446655440000",
		HasVariant: false,
	}

	mockUseCase.On("UpdateProduct", mock.Anything, mock.AnythingOfType("string"), mock.AnythingOfType("string"), mock.AnythingOfType("string"), mock.AnythingOfType("*dto.UpdateProductRequest")).
		Return(expectedResponse, nil)

	// Act
	reqBody := map[string]interface{}{
		"name":  "Test Product",
		"price": 39.99,
		"stock": 150,
	}
	bodyBytes, _ := json.Marshal(reqBody)
	req, _ := http.NewRequest("PUT", "/products/550e8400-e29b-41d4-a716-446655440001", bytes.NewBuffer(bodyBytes))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	router.PUT("/products/:id", handler.UpdateProduct)
	router.ServeHTTP(w, req)

	// Assert
	assert.Equal(t, http.StatusOK, w.Code)

	var response map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(t, err)

	assert.True(t, response["success"].(bool))
	data := response["data"].(map[string]interface{})
	assert.Equal(t, "Updated Product", data["name"])

	mockUseCase.AssertExpectations(t)
}

// TestUpdateProduct_NotFound tests updating a non-existent product.
func TestUpdateProduct_NotFound(t *testing.T) {
	// Arrange
	mockUseCase := new(productusecasemocks.ProductUseCase)
	handler := delivery.NewHandler(mockUseCase)
	router := setupTestRouter()

	mockUseCase.On("UpdateProduct", mock.Anything, "", "", "550e8400-e29b-41d4-a716-446655440002", mock.AnythingOfType("*dto.UpdateProductRequest")).
		Return(nil, domain.ErrProductNotFound)

	// Act
	reqBody := map[string]interface{}{
		"name":  "Updated Name",
		"price": 25.99,
	}
	bodyBytes, _ := json.Marshal(reqBody)
	req, _ := http.NewRequest("PUT", "/products/550e8400-e29b-41d4-a716-446655440002", bytes.NewBuffer(bodyBytes))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	router.PUT("/products/:id", handler.UpdateProduct)
	router.ServeHTTP(w, req)

	// Assert
	assert.Equal(t, http.StatusNotFound, w.Code)

	var response map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(t, err)

	assert.False(t, response["success"].(bool))

	mockUseCase.AssertExpectations(t)
}

// TestUpdateProduct_ValidationError tests product update with invalid input.
func TestUpdateProduct_ValidationError(t *testing.T) {
	tests := []struct {
		name           string
		requestBody    string
		expectedStatus int
	}{
		{
			name:           "invalid JSON",
			requestBody:    `{invalid}`,
			expectedStatus: http.StatusBadRequest,
		},
		{
			name:           "invalid status",
			requestBody:    `{"status": "INVALID"}`,
			expectedStatus: http.StatusUnprocessableEntity,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Arrange
			mockUseCase := new(productusecasemocks.ProductUseCase)
			handler := delivery.NewHandler(mockUseCase)
			router := setupTestRouter()

			// Add mock to handle the use case call (validation may not catch all errors)
			mockUseCase.On("UpdateProduct", mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(nil, domain.ErrInvalidStockReduction).Maybe()

			// Act
			req, _ := http.NewRequest("PUT", "/products/550e8400-e29b-41d4-a716-446655440001", bytes.NewBufferString(tt.requestBody))
			req.Header.Set("Content-Type", "application/json")
			w := httptest.NewRecorder()

			router.PUT("/products/:id", handler.UpdateProduct)
			router.ServeHTTP(w, req)

			// Assert
			assert.Equal(t, tt.expectedStatus, w.Code)
		})
	}
}

// TestDeleteProduct_Success tests successful product deletion (soft delete).
func TestDeleteProduct_Success(t *testing.T) {
	// Arrange
	mockUseCase := new(productusecasemocks.ProductUseCase)
	handler := delivery.NewHandler(mockUseCase)
	router := setupTestRouter()

	expectedResponse := &dto.DeleteResponse{
		Success: true,
		Message: "Product deleted successfully",
	}

	mockUseCase.On("DeleteProduct", mock.Anything, mock.Anything, mock.Anything, mock.MatchedBy(func(r *dto.DeleteProductRequest) bool {
		return r.ID == "550e8400-e29b-41d4-a716-446655440001" && r.Force == false
	})).Return(expectedResponse, nil)

	// Act
	req, _ := http.NewRequest("DELETE", "/products/550e8400-e29b-41d4-a716-446655440001", nil)
	w := httptest.NewRecorder()

	router.DELETE("/products/:id", handler.DeleteProduct)
	router.ServeHTTP(w, req)

	// Assert
	assert.Equal(t, http.StatusOK, w.Code)

	var response map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(t, err)

	assert.True(t, response["success"].(bool))
	data := response["data"].(map[string]interface{})
	assert.Equal(t, "Product deleted successfully", data["message"])

	mockUseCase.AssertExpectations(t)
}

// TestDeleteProduct_ForceDelete tests forced product deletion (hard delete).
func TestDeleteProduct_ForceDelete(t *testing.T) {
	// Arrange
	mockUseCase := new(productusecasemocks.ProductUseCase)
	handler := delivery.NewHandler(mockUseCase)
	router := setupTestRouter()

	expectedResponse := &dto.DeleteResponse{
		Success: true,
		Message: "Product permanently deleted",
	}

	mockUseCase.On("DeleteProduct", mock.Anything, mock.Anything, mock.Anything, mock.MatchedBy(func(r *dto.DeleteProductRequest) bool {
		return r.ID == "550e8400-e29b-41d4-a716-446655440001" && r.Force == true
	})).Return(expectedResponse, nil)

	// Act
	req, _ := http.NewRequest("DELETE", "/products/550e8400-e29b-41d4-a716-446655440001?force=true", nil)
	w := httptest.NewRecorder()

	router.DELETE("/products/:id", handler.DeleteProduct)
	router.ServeHTTP(w, req)

	// Assert
	assert.Equal(t, http.StatusOK, w.Code)

	var response map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(t, err)

	assert.True(t, response["success"].(bool))
	data := response["data"].(map[string]interface{})
	assert.Equal(t, "Product permanently deleted", data["message"])

	mockUseCase.AssertExpectations(t)
}

// TestDeleteProduct_NotFound tests deleting a non-existent product.
func TestDeleteProduct_NotFound(t *testing.T) {
	// Arrange
	mockUseCase := new(productusecasemocks.ProductUseCase)
	handler := delivery.NewHandler(mockUseCase)
	router := setupTestRouter()

	mockUseCase.On("DeleteProduct", mock.Anything, mock.Anything, mock.Anything, mock.AnythingOfType("*dto.DeleteProductRequest")).
		Return(nil, domain.ErrProductNotFound)

	// Act
	req, _ := http.NewRequest("DELETE", "/products/550e8400-e29b-41d4-a716-446655440002", nil)
	w := httptest.NewRecorder()

	router.DELETE("/products/:id", handler.DeleteProduct)
	router.ServeHTTP(w, req)

	// Assert
	assert.Equal(t, http.StatusNotFound, w.Code)

	var response map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(t, err)

	assert.False(t, response["success"].(bool))

	mockUseCase.AssertExpectations(t)
}

// TestRestoreProduct_Success tests successful product restoration.
func TestRestoreProduct_Success(t *testing.T) {
	// Arrange
	mockUseCase := new(productusecasemocks.ProductUseCase)
	handler := delivery.NewHandler(mockUseCase)
	router := setupTestRouter()

	expectedResponse := &dto.ProductResponse{
		ID:         "550e8400-e29b-41d4-a716-446655440001",
		Name:       "Test Product",
		Price:      dto.PriceRange{Min: 29.99, Max: 29.99, Display: fmt.Sprintf("$%.2f", 29.99)},
		Stock:      100,
		OwnerID:    "550e8400-e29b-41d4-a716-446655440000",
		HasVariant: false,
	}

	mockUseCase.On("RestoreProduct", mock.Anything, mock.AnythingOfType("string"), mock.AnythingOfType("string"), mock.AnythingOfType("*dto.RestoreProductRequest")).
		Return(expectedResponse, nil)

	// Act
	req, _ := http.NewRequest("POST", "/products/550e8400-e29b-41d4-a716-446655440001/restore", nil)
	w := httptest.NewRecorder()

	router.POST("/products/:id/restore", handler.RestoreProduct)
	router.ServeHTTP(w, req)

	// Assert
	assert.Equal(t, http.StatusOK, w.Code)

	var response map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(t, err)

	assert.True(t, response["success"].(bool))
	data := response["data"].(map[string]interface{})
	assert.Equal(t, "550e8400-e29b-41d4-a716-446655440001", data["id"])

	mockUseCase.AssertExpectations(t)
}

// TestRestoreProduct_NotFound tests restoring a non-existent product.
func TestRestoreProduct_NotFound(t *testing.T) {
	// Arrange
	mockUseCase := new(productusecasemocks.ProductUseCase)
	handler := delivery.NewHandler(mockUseCase)
	router := setupTestRouter()

	mockUseCase.On("RestoreProduct", mock.Anything, mock.AnythingOfType("string"), mock.AnythingOfType("string"), mock.AnythingOfType("*dto.RestoreProductRequest")).
		Return(nil, domain.ErrProductNotFound)

	// Act
	req, _ := http.NewRequest("POST", "/products/550e8400-e29b-41d4-a716-446655440002/restore", nil)
	w := httptest.NewRecorder()

	router.POST("/products/:id/restore", handler.RestoreProduct)
	router.ServeHTTP(w, req)

	// Assert
	assert.Equal(t, http.StatusNotFound, w.Code)

	var response map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(t, err)

	assert.False(t, response["success"].(bool))

	mockUseCase.AssertExpectations(t)
}

// TestUpdateStock_Success tests successful stock update.
func TestUpdateStock_Success(t *testing.T) {
	// Arrange
	mockUseCase := new(productusecasemocks.ProductUseCase)
	handler := delivery.NewHandler(mockUseCase)
	router := setupTestRouter()

	expectedResponse := &dto.UpdateStockResponse{
		Success: true,
		Message: "Stock updated successfully",
		Stock:   200,
	}

	mockUseCase.On("UpdateStock", mock.Anything, mock.Anything, mock.Anything, mock.MatchedBy(func(r *dto.UpdateStockRequest) bool {
		return r.ID == "550e8400-e29b-41d4-a716-446655440001" && r.Stock == 200
	})).Return(expectedResponse, nil)

	// Act
	reqBody := map[string]interface{}{
		"stock": 200,
	}
	bodyBytes, _ := json.Marshal(reqBody)
	req, _ := http.NewRequest("PUT", "/products/550e8400-e29b-41d4-a716-446655440001/stock", bytes.NewBuffer(bodyBytes))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	router.PUT("/products/:id/stock", handler.UpdateStock)
	router.ServeHTTP(w, req)

	// Assert
	if w.Code != http.StatusOK {
		t.Logf("Response status: %d", w.Code)
		t.Logf("Response body: %s", w.Body.String())
	}
	assert.Equal(t, http.StatusOK, w.Code)

	var response map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(t, err)

	assert.True(t, response["success"].(bool))
	data, ok := response["data"].(map[string]interface{})
	require.True(t, ok, "data field should be present and a map")
	assert.Equal(t, float64(200), data["stock"])

	mockUseCase.AssertExpectations(t)
}

func TestUpdateStock_Success_WithVariantIDInBody(t *testing.T) {
	mockUseCase := new(productusecasemocks.ProductUseCase)
	handler := delivery.NewHandler(mockUseCase)
	router := setupTestRouter()

	expectedResponse := &dto.UpdateStockResponse{
		Success: true,
		Message: "Stock updated successfully",
		Stock:   35,
	}

	mockUseCase.On("UpdateStock", mock.Anything, mock.Anything, mock.Anything, mock.MatchedBy(func(r *dto.UpdateStockRequest) bool {
		return r.ID == "550e8400-e29b-41d4-a716-446655440001" &&
			r.VariantID == "550e8400-e29b-41d4-a716-446655440101" &&
			r.Stock == 10
	})).Return(expectedResponse, nil)

	reqBody := map[string]interface{}{
		"id":    "550e8400-e29b-41d4-a716-446655440101",
		"stock": 10,
	}
	bodyBytes, _ := json.Marshal(reqBody)
	req, _ := http.NewRequest("PUT", "/products/550e8400-e29b-41d4-a716-446655440001/stock", bytes.NewBuffer(bodyBytes))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	router.PUT("/products/:id/stock", handler.UpdateStock)
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	mockUseCase.AssertExpectations(t)
}

// TestUpdateStock_NotFound tests updating stock for non-existent product.
func TestUpdateStock_NotFound(t *testing.T) {
	// Arrange
	mockUseCase := new(productusecasemocks.ProductUseCase)
	handler := delivery.NewHandler(mockUseCase)
	router := setupTestRouter()

	mockUseCase.On("UpdateStock", mock.Anything, mock.Anything, mock.Anything, mock.MatchedBy(func(r *dto.UpdateStockRequest) bool {
		return r.ID == "550e8400-e29b-41d4-a716-446655440002" && r.Stock == 100
	})).Return(nil, domain.ErrProductNotFound)

	// Act
	reqBody := map[string]interface{}{
		"stock": 100,
	}
	bodyBytes, _ := json.Marshal(reqBody)
	req, _ := http.NewRequest("PUT", "/products/550e8400-e29b-41d4-a716-446655440002/stock", bytes.NewBuffer(bodyBytes))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	router.PUT("/products/:id/stock", handler.UpdateStock)
	router.ServeHTTP(w, req)

	// Assert
	assert.Equal(t, http.StatusNotFound, w.Code)

	var response map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(t, err)

	assert.False(t, response["success"].(bool))

	mockUseCase.AssertExpectations(t)
}

// TestUpdateStock_ValidationError tests stock update with invalid input.
func TestUpdateStock_ValidationError(t *testing.T) {
	// Arrange
	mockUseCase := new(productusecasemocks.ProductUseCase)
	handler := delivery.NewHandler(mockUseCase)
	router := setupTestRouter()

	// Act - negative stock
	reqBody := map[string]interface{}{
		"stock": -1,
	}
	bodyBytes, _ := json.Marshal(reqBody)
	req, _ := http.NewRequest("PUT", "/products/550e8400-e29b-41d4-a716-446655440001/stock", bytes.NewBuffer(bodyBytes))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	router.PUT("/products/:id/stock", handler.UpdateStock)
	router.ServeHTTP(w, req)

	// Assert
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

// TestUpdateStock_InsufficientStock tests insufficient stock scenario.
// Note: This test demonstrates that domain validation errors return 422.
func TestUpdateStock_InsufficientStock(t *testing.T) {
	// Arrange
	mockUseCase := new(productusecasemocks.ProductUseCase)
	handler := delivery.NewHandler(mockUseCase)
	router := setupTestRouter()

	// Create a validation error - in the product domain, any non-nil error
	// is treated as a validation error by IsValidationError
	stockErr := errors.New("insufficient stock available")
	mockUseCase.On("UpdateStock", mock.Anything, mock.Anything, mock.Anything, mock.MatchedBy(func(r *dto.UpdateStockRequest) bool {
		return r.ID == "550e8400-e29b-41d4-a716-446655440001" && r.Stock == 1000
	})).Return(nil, stockErr)

	// Act
	reqBody := map[string]interface{}{
		"stock": 1000,
	}
	bodyBytes, _ := json.Marshal(reqBody)
	req, _ := http.NewRequest("PUT", "/products/550e8400-e29b-41d4-a716-446655440001/stock", bytes.NewBuffer(bodyBytes))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	router.PUT("/products/:id/stock", handler.UpdateStock)
	router.ServeHTTP(w, req)

	// Assert - validation errors return 422 Unprocessable Entity
	assert.Equal(t, http.StatusUnprocessableEntity, w.Code)

	var response map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(t, err)

	assert.False(t, response["success"].(bool))
	errObj := response["data"].(map[string]interface{})
	assert.Equal(t, "VALIDATION_ERROR", errObj["code"])
	assert.Contains(t, errObj["message"], "insufficient stock")

	mockUseCase.AssertExpectations(t)
}

// TestHandleError_ValidationError tests validation error handling.
func TestHandleError_ValidationError(t *testing.T) {
	// This test validates the error handling path for usecase-level validation errors
	// Arrange
	mockUseCase := new(productusecasemocks.ProductUseCase)
	handler := delivery.NewHandler(mockUseCase)
	router := setupTestRouter()

	// Create a validation error that would come from the usecase layer
	// (not from Gin binding which returns BAD_REQUEST)
	validationErr := errors.New("invalid stock reduction amount")
	mockUseCase.On("CreateProduct", mock.Anything, mock.AnythingOfType("string"), mock.AnythingOfType("*dto.CreateProductRequest")).
		Return(nil, validationErr)

	// Act - use valid data to pass Gin binding
	reqBody := map[string]interface{}{
		"name":    "Test Product",
		"price":   29.99,
		"stock":   100,
		"ownerId": "550e8400-e29b-41d4-a716-446655440000",
	}
	bodyBytes, _ := json.Marshal(reqBody)
	req, _ := http.NewRequest("POST", "/products", bytes.NewBuffer(bodyBytes))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	// Set user context
	router.Use(func(c *gin.Context) {
		c.Set("user_id", "550e8400-e29b-41d4-a716-446655440000")
		c.Next()
	})

	router.POST("/products", handler.CreateProduct)
	router.ServeHTTP(w, req)

	// Assert
	assert.Equal(t, http.StatusUnprocessableEntity, w.Code)

	var response map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(t, err)

	assert.False(t, response["success"].(bool))
	errObj := response["data"].(map[string]interface{})
	assert.Equal(t, "VALIDATION_ERROR", errObj["code"])

	mockUseCase.AssertExpectations(t)
}

// TestCreateProduct_InternalError tests create product with internal error.
func TestCreateProduct_InternalError(t *testing.T) {
	// Arrange
	mockUseCase := new(productusecasemocks.ProductUseCase)
	handler := delivery.NewHandler(mockUseCase)
	router := setupTestRouter()

	mockUseCase.On("CreateProduct", mock.Anything, mock.Anything, mock.AnythingOfType("*dto.CreateProductRequest")).
		Return(nil, errors.New("database connection failed"))

	// Act
	reqBody := map[string]interface{}{
		"name":    "Test Product",
		"price":   29.99,
		"stock":   100,
		"ownerId": "550e8400-e29b-41d4-a716-446655440000",
	}
	bodyBytes, _ := json.Marshal(reqBody)
	req, _ := http.NewRequest("POST", "/products", bytes.NewBuffer(bodyBytes))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	// Set user context
	router.Use(func(c *gin.Context) {
		c.Set("user_id", "550e8400-e29b-41d4-a716-446655440000")
		c.Next()
	})

	router.POST("/products", handler.CreateProduct)
	router.ServeHTTP(w, req)

	// Assert
	assert.Equal(t, http.StatusInternalServerError, w.Code)

	var response map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(t, err)
	assert.False(t, response["success"].(bool))

	mockUseCase.AssertExpectations(t)
}

// TestGetProduct_InvalidUUID tests get product with invalid UUID.
func TestGetProduct_InvalidUUID(t *testing.T) {
	// Arrange
	mockUseCase := new(productusecasemocks.ProductUseCase)
	handler := delivery.NewHandler(mockUseCase)
	router := setupTestRouter()

	// Act - Gin's UUID validation will fail
	req, _ := http.NewRequest("GET", "/products/invalid-uuid", nil)
	w := httptest.NewRecorder()

	router.GET("/products/:id", handler.GetProduct)
	router.ServeHTTP(w, req)

	// Assert
	assert.Equal(t, http.StatusBadRequest, w.Code)

	var response map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(t, err)
	assert.False(t, response["success"].(bool))
}

// TestListProducts_InvalidQueryParams tests list products with invalid query params.
func TestListProducts_InvalidQueryParams(t *testing.T) {
	tests := []struct {
		name           string
		query          string
		expectedStatus int
	}{
		{
			name:           "invalid page parameter",
			query:          "/products?page=invalid",
			expectedStatus: http.StatusBadRequest,
		},
		{
			name:           "invalid limit parameter",
			query:          "/products?limit=invalid",
			expectedStatus: http.StatusBadRequest,
		},
		{
			name:           "limit exceeds maximum",
			query:          "/products?limit=1000",
			expectedStatus: http.StatusBadRequest,
		},
		{
			name:           "invalid status parameter",
			query:          "/products?status=INVALID",
			expectedStatus: http.StatusBadRequest,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Arrange
			mockUseCase := new(productusecasemocks.ProductUseCase)
			handler := delivery.NewHandler(mockUseCase)
			router := setupTestRouter()

			// Act
			req, _ := http.NewRequest("GET", tt.query, nil)
			w := httptest.NewRecorder()

			router.GET("/products", handler.ListProducts)
			router.ServeHTTP(w, req)

			// Assert
			assert.Equal(t, tt.expectedStatus, w.Code)

			var response map[string]interface{}
			err := json.Unmarshal(w.Body.Bytes(), &response)
			require.NoError(t, err)
			assert.False(t, response["success"].(bool))
		})
	}
}

// TestListProducts_InternalError tests list products with internal error.
func TestListProducts_InternalError(t *testing.T) {
	// Arrange
	mockUseCase := new(productusecasemocks.ProductUseCase)
	handler := delivery.NewHandler(mockUseCase)
	router := setupTestRouter()

	mockUseCase.On("ListProducts", mock.Anything, mock.Anything, mock.Anything, mock.AnythingOfType("*dto.ListProductsRequest")).
		Return(nil, errors.New("database error"))

	// Act
	req, _ := http.NewRequest("GET", "/products?page=1&limit=10", nil)
	w := httptest.NewRecorder()

	router.GET("/products", handler.ListProducts)
	router.ServeHTTP(w, req)

	// Assert
	assert.Equal(t, http.StatusInternalServerError, w.Code)

	var response map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(t, err)
	assert.False(t, response["success"].(bool))

	mockUseCase.AssertExpectations(t)
}

// TestUpdateProduct_InvalidUUID tests update product with invalid UUID.
func TestUpdateProduct_InvalidUUID(t *testing.T) {
	// Arrange
	mockUseCase := new(productusecasemocks.ProductUseCase)
	handler := delivery.NewHandler(mockUseCase)
	router := setupTestRouter()

	reqBody := map[string]interface{}{
		"name":  "Updated Product",
		"price": 39.99,
	}
	bodyBytes, _ := json.Marshal(reqBody)

	// Act - Gin's UUID validation will fail
	req, _ := http.NewRequest("PUT", "/products/invalid-uuid", bytes.NewBuffer(bodyBytes))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	router.PUT("/products/:id", handler.UpdateProduct)
	router.ServeHTTP(w, req)

	// Assert
	assert.Equal(t, http.StatusBadRequest, w.Code)

	var response map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(t, err)
	assert.False(t, response["success"].(bool))
}

// TestUpdateProduct_InternalError tests update product with internal error.
func TestUpdateProduct_InternalError(t *testing.T) {
	// Arrange
	mockUseCase := new(productusecasemocks.ProductUseCase)
	handler := delivery.NewHandler(mockUseCase)
	router := setupTestRouter()

	mockUseCase.On("UpdateProduct", mock.Anything, mock.Anything, mock.Anything, "550e8400-e29b-41d4-a716-446655440001", mock.AnythingOfType("*dto.UpdateProductRequest")).
		Return(nil, errors.New("database error"))

	reqBody := map[string]interface{}{
		"name":  "Updated Product",
		"price": 39.99,
	}
	bodyBytes, _ := json.Marshal(reqBody)

	// Act
	req, _ := http.NewRequest("PUT", "/products/550e8400-e29b-41d4-a716-446655440001", bytes.NewBuffer(bodyBytes))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	router.PUT("/products/:id", handler.UpdateProduct)
	router.ServeHTTP(w, req)

	// Assert
	assert.Equal(t, http.StatusInternalServerError, w.Code)

	var response map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(t, err)
	assert.False(t, response["success"].(bool))

	mockUseCase.AssertExpectations(t)
}

// TestDeleteProduct_InvalidUUID tests delete product with invalid UUID.
func TestDeleteProduct_InvalidUUID(t *testing.T) {
	// Arrange
	mockUseCase := new(productusecasemocks.ProductUseCase)
	handler := delivery.NewHandler(mockUseCase)
	router := setupTestRouter()

	// Act - Gin's UUID validation will fail
	req, _ := http.NewRequest("DELETE", "/products/invalid-uuid", nil)
	w := httptest.NewRecorder()

	router.DELETE("/products/:id", handler.DeleteProduct)
	router.ServeHTTP(w, req)

	// Assert
	assert.Equal(t, http.StatusBadRequest, w.Code)

	var response map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(t, err)
	assert.False(t, response["success"].(bool))
}

// TestDeleteProduct_InternalError tests delete product with internal error.
func TestDeleteProduct_InternalError(t *testing.T) {
	// Arrange
	mockUseCase := new(productusecasemocks.ProductUseCase)
	handler := delivery.NewHandler(mockUseCase)
	router := setupTestRouter()

	mockUseCase.On("DeleteProduct", mock.Anything, mock.Anything, mock.Anything, mock.AnythingOfType("*dto.DeleteProductRequest")).
		Return(nil, errors.New("database error"))

	// Act
	req, _ := http.NewRequest("DELETE", "/products/550e8400-e29b-41d4-a716-446655440001", nil)
	w := httptest.NewRecorder()

	router.DELETE("/products/:id", handler.DeleteProduct)
	router.ServeHTTP(w, req)

	// Assert
	assert.Equal(t, http.StatusInternalServerError, w.Code)

	var response map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(t, err)
	assert.False(t, response["success"].(bool))

	mockUseCase.AssertExpectations(t)
}

// TestRestoreProduct_InvalidUUID tests restore product with invalid UUID.
func TestRestoreProduct_InvalidUUID(t *testing.T) {
	// Arrange
	mockUseCase := new(productusecasemocks.ProductUseCase)
	handler := delivery.NewHandler(mockUseCase)
	router := setupTestRouter()

	// Act - Gin's UUID validation will fail
	req, _ := http.NewRequest("POST", "/products/invalid-uuid/restore", nil)
	w := httptest.NewRecorder()

	router.POST("/products/:id/restore", handler.RestoreProduct)
	router.ServeHTTP(w, req)

	// Assert
	assert.Equal(t, http.StatusBadRequest, w.Code)

	var response map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(t, err)
	assert.False(t, response["success"].(bool))
}

// TestRestoreProduct_InternalError tests restore product with internal error.
func TestRestoreProduct_InternalError(t *testing.T) {
	// Arrange
	mockUseCase := new(productusecasemocks.ProductUseCase)
	handler := delivery.NewHandler(mockUseCase)
	router := setupTestRouter()

	mockUseCase.On("RestoreProduct", mock.Anything, mock.AnythingOfType("string"), mock.AnythingOfType("string"), mock.AnythingOfType("*dto.RestoreProductRequest")).
		Return(nil, errors.New("database error"))

	// Act
	req, _ := http.NewRequest("POST", "/products/550e8400-e29b-41d4-a716-446655440001/restore", nil)
	w := httptest.NewRecorder()

	router.POST("/products/:id/restore", handler.RestoreProduct)
	router.ServeHTTP(w, req)

	// Assert
	assert.Equal(t, http.StatusInternalServerError, w.Code)

	var response map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(t, err)
	assert.False(t, response["success"].(bool))

	mockUseCase.AssertExpectations(t)
}

// TestUpdateStock_InvalidUUID tests update stock with invalid UUID.
func TestUpdateStock_InvalidUUID(t *testing.T) {
	// Arrange
	mockUseCase := new(productusecasemocks.ProductUseCase)
	handler := delivery.NewHandler(mockUseCase)
	router := setupTestRouter()

	reqBody := map[string]interface{}{
		"stock": 100,
	}
	bodyBytes, _ := json.Marshal(reqBody)

	// Act - Gin's UUID validation will fail
	req, _ := http.NewRequest("PUT", "/products/invalid-uuid/stock", bytes.NewBuffer(bodyBytes))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	router.PUT("/products/:id/stock", handler.UpdateStock)
	router.ServeHTTP(w, req)

	// Assert
	assert.Equal(t, http.StatusBadRequest, w.Code)

	var response map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(t, err)
	assert.False(t, response["success"].(bool))
}

// TestUpdateStock_MissingStock tests update stock with missing stock field.
func TestUpdateStock_MissingStock(t *testing.T) {
	// Arrange
	mockUseCase := new(productusecasemocks.ProductUseCase)
	handler := delivery.NewHandler(mockUseCase)
	router := setupTestRouter()

	reqBody := map[string]interface{}{} // Missing stock field
	bodyBytes, _ := json.Marshal(reqBody)

	// Act
	req, _ := http.NewRequest("PUT", "/products/550e8400-e29b-41d4-a716-446655440001/stock", bytes.NewBuffer(bodyBytes))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	router.PUT("/products/:id/stock", handler.UpdateStock)
	router.ServeHTTP(w, req)

	// Assert - Gin validation should fail
	assert.Equal(t, http.StatusBadRequest, w.Code)

	var response map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(t, err)
	assert.False(t, response["success"].(bool))
}

// TestUpdateStock_InternalError tests update stock with internal error.
func TestUpdateStock_InternalError(t *testing.T) {
	// Arrange
	mockUseCase := new(productusecasemocks.ProductUseCase)
	handler := delivery.NewHandler(mockUseCase)
	router := setupTestRouter()

	mockUseCase.On("UpdateStock", mock.Anything, mock.Anything, mock.Anything, mock.AnythingOfType("*dto.UpdateStockRequest")).
		Return(nil, errors.New("database error"))

	reqBody := map[string]interface{}{
		"stock": 100,
	}
	bodyBytes, _ := json.Marshal(reqBody)

	// Act
	req, _ := http.NewRequest("PUT", "/products/550e8400-e29b-41d4-a716-446655440001/stock", bytes.NewBuffer(bodyBytes))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	router.PUT("/products/:id/stock", handler.UpdateStock)
	router.ServeHTTP(w, req)

	// Assert
	assert.Equal(t, http.StatusInternalServerError, w.Code)

	var response map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(t, err)
	assert.False(t, response["success"].(bool))

	mockUseCase.AssertExpectations(t)
}

// TestCORSMiddleware tests the CORS middleware.
func TestCORSMiddleware(t *testing.T) {
	tests := []struct {
		name           string
		method         string
		expectedStatus int
		checkHeaders   bool
	}{
		{
			name:           "OPTIONS request returns 204",
			method:         "OPTIONS",
			expectedStatus: http.StatusNoContent,
			checkHeaders:   true,
		},
		{
			name:           "GET request passes through",
			method:         "GET",
			expectedStatus: http.StatusOK,
			checkHeaders:   false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Arrange
			router := setupTestRouter()
			router.Use(delivery.CORSMiddleware())
			router.GET("/test", func(c *gin.Context) {
				c.Status(http.StatusOK)
			})

			// Act
			req, _ := http.NewRequest(tt.method, "/test", nil)
			w := httptest.NewRecorder()
			router.ServeHTTP(w, req)

			// Assert
			assert.Equal(t, tt.expectedStatus, w.Code)

			if tt.checkHeaders {
				assert.Equal(t, "*", w.Header().Get("Access-Control-Allow-Origin"))
				assert.Equal(t, "true", w.Header().Get("Access-Control-Allow-Credentials"))
				assert.Contains(t, w.Header().Get("Access-Control-Allow-Headers"), "Content-Type")
				assert.Contains(t, w.Header().Get("Access-Control-Allow-Methods"), "POST")
			}
		})
	}
}

// TestPublicHealth tests the public health endpoint.
func TestPublicHealth(t *testing.T) {
	// Arrange
	mockUseCase := new(productusecasemocks.ProductUseCase)
	handler := delivery.NewHandler(mockUseCase)
	router := setupTestRouter()

	router.GET("/health", handler.PublicHealth)

	// Act
	req, _ := http.NewRequest("GET", "/health", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	// Assert
	assert.Equal(t, http.StatusOK, w.Code)

	var response map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(t, err)

	assert.Equal(t, "ok", response["status"])
	assert.Equal(t, "service-product", response["service"])
}

// TestReadyProbe tests the readiness probe endpoint.
func TestReadyProbe(t *testing.T) {
	// Arrange
	mockUseCase := new(productusecasemocks.ProductUseCase)
	handler := delivery.NewHandler(mockUseCase)
	router := setupTestRouter()

	router.GET("/ready", handler.ReadyProbe)

	// Act
	req, _ := http.NewRequest("GET", "/ready", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	// Assert
	assert.Equal(t, http.StatusOK, w.Code)

	var response map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(t, err)

	assert.Equal(t, true, response["ready"])
}

// TestLiveProbe tests the liveness probe endpoint.
func TestLiveProbe(t *testing.T) {
	// Arrange
	mockUseCase := new(productusecasemocks.ProductUseCase)
	handler := delivery.NewHandler(mockUseCase)
	router := setupTestRouter()

	router.GET("/live", handler.LiveProbe)

	// Act
	req, _ := http.NewRequest("GET", "/live", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	// Assert
	assert.Equal(t, http.StatusOK, w.Code)

	var response map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(t, err)

	assert.Equal(t, true, response["alive"])
}

// TestRegisterRoutes tests route registration.
func TestRegisterRoutes(t *testing.T) {
	// Arrange
	mockUseCase := new(productusecasemocks.ProductUseCase)
	router := setupTestRouter()

	// Set up mock expectations for all possible route calls
	mockUseCase.On("ListProducts", mock.Anything, "user-123", "USER", mock.AnythingOfType("*dto.ListProductsRequest")).
		Return(&dto.ProductListResponse{}, nil).
		Once()

	// Act
	delivery.RegisterRoutes(router, mockUseCase, routeTestJWTSecret, nil)

	// Assert - verify business routes are registered and health routes are not
	routes := router.Routes()
	routePaths := make(map[string]bool, len(routes))
	for _, route := range routes {
		routePaths[route.Path] = true
	}

	assert.False(t, routePaths["/health"], "Health route should not be registered by delivery routes")
	assert.False(t, routePaths["/ready"], "Ready route should not be registered by delivery routes")
	assert.False(t, routePaths["/live"], "Live route should not be registered by delivery routes")
	assert.True(t, routePaths["/api/v1/products"], "Versioned products route should be registered")
	assert.False(t, routePaths["/products"], "Legacy products route should not be registered")

	// Product routes require JWT.
	w := httptest.NewRecorder()
	reqUnauth, _ := http.NewRequest("GET", "/api/v1/products", nil)
	router.ServeHTTP(w, reqUnauth)
	assert.Equal(t, http.StatusUnauthorized, w.Code)

	// Authenticated request can access list endpoint.
	w = httptest.NewRecorder()
	reqAuth, _ := http.NewRequest("GET", "/api/v1/products", nil)
	reqAuth.Header.Set("Authorization", generateBearerToken(t, "user-123", "USER"))
	router.ServeHTTP(w, reqAuth)
	assert.Equal(t, http.StatusOK, w.Code)

	// Legacy endpoint should no longer exist.
	w = httptest.NewRecorder()
	reqLegacy, _ := http.NewRequest("GET", "/products", nil)
	router.ServeHTTP(w, reqLegacy)
	assert.Equal(t, http.StatusNotFound, w.Code)

	mockUseCase.AssertExpectations(t)
}

// TestGetProduct_Unauthorized tests GetProduct without user context.
func TestGetProduct_Unauthorized(t *testing.T) {
	// Arrange
	mockUseCase := new(productusecasemocks.ProductUseCase)
	handler := delivery.NewHandler(mockUseCase)
	router := setupTestRouter()

	expectedResponse := &dto.ProductResponse{
		ID:         "550e8400-e29b-41d4-a716-446655440001",
		Name:       "Test Product",
		OwnerID:    "550e8400-e29b-41d4-a716-446655440000",
		HasVariant: false,
	}

	mockUseCase.On("GetProduct", mock.Anything, "", "", mock.AnythingOfType("*dto.GetProductRequest")).
		Return(expectedResponse, nil)

	// Act - no user context set
	req, _ := http.NewRequest("GET", "/products/550e8400-e29b-41d4-a716-446655440001", nil)
	w := httptest.NewRecorder()

	router.GET("/products/:id", handler.GetProduct)
	router.ServeHTTP(w, req)

	// Assert
	assert.Equal(t, http.StatusOK, w.Code)

	mockUseCase.AssertExpectations(t)
}

// TestListProducts_Unauthorized tests ListProducts without user context.
func TestListProducts_Unauthorized(t *testing.T) {
	// Arrange
	mockUseCase := new(productusecasemocks.ProductUseCase)
	handler := delivery.NewHandler(mockUseCase)
	router := setupTestRouter()

	expectedResponse := &dto.ProductListResponse{
		Data:       []*dto.ProductResponse{},
		Total:      0,
		Page:       1,
		Limit:      10,
		TotalPages: 0,
	}

	mockUseCase.On("ListProducts", mock.Anything, "", "", mock.AnythingOfType("*dto.ListProductsRequest")).
		Return(expectedResponse, nil)

	// Act - no user context set
	req, _ := http.NewRequest("GET", "/products", nil)
	w := httptest.NewRecorder()

	router.GET("/products", handler.ListProducts)
	router.ServeHTTP(w, req)

	// Assert
	assert.Equal(t, http.StatusOK, w.Code)

	mockUseCase.AssertExpectations(t)
}

// TestDeleteProduct_Unauthorized tests DeleteProduct without user context.
func TestDeleteProduct_Unauthorized(t *testing.T) {
	// Arrange
	mockUseCase := new(productusecasemocks.ProductUseCase)
	handler := delivery.NewHandler(mockUseCase)
	router := setupTestRouter()

	expectedResponse := &dto.DeleteResponse{
		Success: true,
		Message: "Product deleted successfully",
	}

	mockUseCase.On("DeleteProduct", mock.Anything, "", "", mock.MatchedBy(func(r *dto.DeleteProductRequest) bool {
		return r.ID == "550e8400-e29b-41d4-a716-446655440001"
	})).Return(expectedResponse, nil)

	// Act - no user context set
	req, _ := http.NewRequest("DELETE", "/products/550e8400-e29b-41d4-a716-446655440001", nil)
	w := httptest.NewRecorder()

	router.DELETE("/products/:id", handler.DeleteProduct)
	router.ServeHTTP(w, req)

	// Assert
	assert.Equal(t, http.StatusOK, w.Code)

	mockUseCase.AssertExpectations(t)
}

// TestRestoreProduct_Unauthorized tests RestoreProduct without user context.
func TestRestoreProduct_Unauthorized(t *testing.T) {
	// Arrange
	mockUseCase := new(productusecasemocks.ProductUseCase)
	handler := delivery.NewHandler(mockUseCase)
	router := setupTestRouter()

	expectedResponse := &dto.ProductResponse{
		ID:         "550e8400-e29b-41d4-a716-446655440001",
		Name:       "Test Product",
		HasVariant: false,
	}

	mockUseCase.On("RestoreProduct", mock.Anything, "", "", mock.AnythingOfType("*dto.RestoreProductRequest")).
		Return(expectedResponse, nil)

	// Act - no user context set
	req, _ := http.NewRequest("POST", "/products/550e8400-e29b-41d4-a716-446655440001/restore", nil)
	w := httptest.NewRecorder()

	router.POST("/products/:id/restore", handler.RestoreProduct)
	router.ServeHTTP(w, req)

	// Assert
	assert.Equal(t, http.StatusOK, w.Code)

	mockUseCase.AssertExpectations(t)
}

// TestUpdateStock_Unauthorized tests UpdateStock without user context.
func TestUpdateStock_Unauthorized(t *testing.T) {
	// Arrange
	mockUseCase := new(productusecasemocks.ProductUseCase)
	handler := delivery.NewHandler(mockUseCase)
	router := setupTestRouter()

	expectedResponse := &dto.UpdateStockResponse{
		Success: true,
		Message: "Stock updated successfully",
		Stock:   150,
	}

	mockUseCase.On("UpdateStock", mock.Anything, "", "", mock.MatchedBy(func(r *dto.UpdateStockRequest) bool {
		return r.ID == "550e8400-e29b-41d4-a716-446655440001" && r.Stock == 150
	})).Return(expectedResponse, nil)

	// Act - no user context set
	reqBody := map[string]interface{}{"stock": 150}
	bodyBytes, _ := json.Marshal(reqBody)
	req, _ := http.NewRequest("PUT", "/products/550e8400-e29b-41d4-a716-446655440001/stock", bytes.NewBuffer(bodyBytes))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	router.PUT("/products/:id/stock", handler.UpdateStock)
	router.ServeHTTP(w, req)

	// Assert
	assert.Equal(t, http.StatusOK, w.Code)

	mockUseCase.AssertExpectations(t)
}

// TestGetProduct_AdminRole tests GetProduct with admin role.
func TestGetProduct_AdminRole(t *testing.T) {
	// Arrange
	mockUseCase := new(productusecasemocks.ProductUseCase)
	handler := delivery.NewHandler(mockUseCase)
	router := setupTestRouter()

	expectedResponse := &dto.ProductResponse{
		ID:         "550e8400-e29b-41d4-a716-446655440001",
		Name:       "Test Product",
		OwnerID:    "550e8400-e29b-41d4-a716-446655440000",
		HasVariant: false,
	}

	mockUseCase.On("GetProduct", mock.Anything, "", "ADMIN", mock.AnythingOfType("*dto.GetProductRequest")).
		Return(expectedResponse, nil)

	// Act
	req, _ := http.NewRequest("GET", "/products/550e8400-e29b-41d4-a716-446655440001", nil)
	w := httptest.NewRecorder()

	router.Use(func(c *gin.Context) {
		c.Set("user_role", "ADMIN")
		c.Next()
	})
	router.GET("/products/:id", handler.GetProduct)
	router.ServeHTTP(w, req)

	// Assert
	assert.Equal(t, http.StatusOK, w.Code)

	mockUseCase.AssertExpectations(t)
}

// TestListProducts_AdminRole tests ListProducts with admin role.
func TestListProducts_AdminRole(t *testing.T) {
	// Arrange
	mockUseCase := new(productusecasemocks.ProductUseCase)
	handler := delivery.NewHandler(mockUseCase)
	router := setupTestRouter()

	expectedResponse := &dto.ProductListResponse{
		Data:       []*dto.ProductResponse{},
		Total:      0,
		Page:       1,
		Limit:      10,
		TotalPages: 0,
	}

	mockUseCase.On("ListProducts", mock.Anything, "", "ADMIN", mock.AnythingOfType("*dto.ListProductsRequest")).
		Return(expectedResponse, nil)

	// Act
	req, _ := http.NewRequest("GET", "/products", nil)
	w := httptest.NewRecorder()

	router.Use(func(c *gin.Context) {
		c.Set("user_role", "ADMIN")
		c.Next()
	})
	router.GET("/products", handler.ListProducts)
	router.ServeHTTP(w, req)

	// Assert
	assert.Equal(t, http.StatusOK, w.Code)

	mockUseCase.AssertExpectations(t)
}

// TestGetProduct_BindQueryError tests GetProduct with bind query error.
func TestGetProduct_BindQueryError(t *testing.T) {
	// Arrange
	mockUseCase := new(productusecasemocks.ProductUseCase)
	handler := delivery.NewHandler(mockUseCase)
	router := setupTestRouter()

	// This test checks the case when ShouldBindQuery fails but ShouldBindUri succeeds
	// Since Gin validation catches most query errors at the router level,
	// this is a theoretical edge case

	// Act - request with malformed query that might cause bind issues
	req, _ := http.NewRequest("GET", "/products/550e8400-e29b-41d4-a716-446655440001?include_deleted=invalid", nil)
	w := httptest.NewRecorder()

	router.GET("/products/:id", handler.GetProduct)
	router.ServeHTTP(w, req)

	// Should return 400 due to invalid boolean binding
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

// TestListProducts_EmptyResults tests ListProducts returning empty results.
func TestListProducts_EmptyResults(t *testing.T) {
	// Arrange
	mockUseCase := new(productusecasemocks.ProductUseCase)
	handler := delivery.NewHandler(mockUseCase)
	router := setupTestRouter()

	expectedResponse := &dto.ProductListResponse{
		Data:       []*dto.ProductResponse{},
		Total:      0,
		Page:       1,
		Limit:      10,
		TotalPages: 0,
	}

	mockUseCase.On("ListProducts", mock.Anything, "", "", mock.AnythingOfType("*dto.ListProductsRequest")).
		Return(expectedResponse, nil)

	// Act
	req, _ := http.NewRequest("GET", "/products", nil)
	w := httptest.NewRecorder()

	router.GET("/products", handler.ListProducts)
	router.ServeHTTP(w, req)

	// Assert
	assert.Equal(t, http.StatusOK, w.Code)

	var response map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(t, err)

	assert.True(t, response["success"].(bool))
	data := response["data"].(map[string]interface{})
	assert.Equal(t, float64(0), data["total"])

	mockUseCase.AssertExpectations(t)
}

// TestUpdateProduct_Conflict tests UpdateProduct with name conflict.
func TestUpdateProduct_Conflict(t *testing.T) {
	// Arrange
	mockUseCase := new(productusecasemocks.ProductUseCase)
	handler := delivery.NewHandler(mockUseCase)
	router := setupTestRouter()

	mockUseCase.On("UpdateProduct", mock.Anything, mock.Anything, mock.Anything, "550e8400-e29b-41d4-a716-446655440001", mock.AnythingOfType("*dto.UpdateProductRequest")).
		Return(nil, domain.ErrProductNameAlreadyUsed)

	// Act
	reqBody := map[string]interface{}{
		"name":  "Existing Name",
		"price": 25.99,
	}
	bodyBytes, _ := json.Marshal(reqBody)
	req, _ := http.NewRequest("PUT", "/products/550e8400-e29b-41d4-a716-446655440001", bytes.NewBuffer(bodyBytes))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	router.PUT("/products/:id", handler.UpdateProduct)
	router.ServeHTTP(w, req)

	// Assert
	assert.Equal(t, http.StatusConflict, w.Code)

	var response map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(t, err)
	assert.False(t, response["success"].(bool))

	mockUseCase.AssertExpectations(t)
}

// TestUpdateProduct_Validation tests UpdateProduct with validation error.
func TestUpdateProduct_Validation(t *testing.T) {
	// Arrange
	mockUseCase := new(productusecasemocks.ProductUseCase)
	handler := delivery.NewHandler(mockUseCase)
	router := setupTestRouter()

	validationErr := errors.New("invalid stock reduction amount")
	mockUseCase.On("UpdateProduct", mock.Anything, mock.Anything, mock.Anything, "550e8400-e29b-41d4-a716-446655440001", mock.AnythingOfType("*dto.UpdateProductRequest")).
		Return(nil, validationErr)

	// Act
	reqBody := map[string]interface{}{
		"stockReduction": 1000,
	}
	bodyBytes, _ := json.Marshal(reqBody)
	req, _ := http.NewRequest("PUT", "/products/550e8400-e29b-41d4-a716-446655440001", bytes.NewBuffer(bodyBytes))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	router.PUT("/products/:id", handler.UpdateProduct)
	router.ServeHTTP(w, req)

	// Assert
	assert.Equal(t, http.StatusUnprocessableEntity, w.Code)

	mockUseCase.AssertExpectations(t)
}

// TestRestoreProduct_Validation tests RestoreProduct with validation error.
func TestRestoreProduct_Validation(t *testing.T) {
	// Arrange
	mockUseCase := new(productusecasemocks.ProductUseCase)
	handler := delivery.NewHandler(mockUseCase)
	router := setupTestRouter()

	// Use an actual validation error (insufficient stock is recognized as validation)
	validationErr := errors.New("invalid stock reduction amount")
	mockUseCase.On("RestoreProduct", mock.Anything, mock.AnythingOfType("string"), mock.AnythingOfType("string"), mock.AnythingOfType("*dto.RestoreProductRequest")).
		Return(nil, validationErr)

	// Act
	req, _ := http.NewRequest("POST", "/products/550e8400-e29b-41d4-a716-446655440001/restore", nil)
	w := httptest.NewRecorder()

	router.POST("/products/:id/restore", handler.RestoreProduct)
	router.ServeHTTP(w, req)

	// Assert - validation error returns 422
	assert.Equal(t, http.StatusUnprocessableEntity, w.Code)

	mockUseCase.AssertExpectations(t)
}

// TestDeleteProduct_Validation tests DeleteProduct with validation error.
func TestDeleteProduct_Validation(t *testing.T) {
	// Arrange
	mockUseCase := new(productusecasemocks.ProductUseCase)
	handler := delivery.NewHandler(mockUseCase)
	router := setupTestRouter()

	// Use an actual validation error (insufficient stock is recognized as validation)
	validationErr := errors.New("invalid stock reduction amount")
	mockUseCase.On("DeleteProduct", mock.Anything, mock.Anything, mock.Anything, mock.AnythingOfType("*dto.DeleteProductRequest")).
		Return(nil, validationErr)

	// Act
	req, _ := http.NewRequest("DELETE", "/products/550e8400-e29b-41d4-a716-446655440001", nil)
	w := httptest.NewRecorder()

	router.DELETE("/products/:id", handler.DeleteProduct)
	router.ServeHTTP(w, req)

	// Assert - validation error returns 422
	assert.Equal(t, http.StatusUnprocessableEntity, w.Code)

	var response map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(t, err)
	assert.False(t, response["success"].(bool))

	mockUseCase.AssertExpectations(t)
}

// TestRegisterRoutesWithRateLimit tests route registration with rate limiting.
func TestRegisterRoutesWithRateLimit(t *testing.T) {
	// This test verifies the function is callable
	// We don't actually test rate limiting as it requires Redis
	mockUseCase := new(productusecasemocks.ProductUseCase)
	router := setupTestRouter()

	// Set up mock expectations for all possible route calls
	mockUseCase.On("ListProducts", mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(&dto.ProductListResponse{}, nil).Maybe()

	// This should not panic - it will just register routes.
	delivery.RegisterRoutesWithRateLimit(router, mockUseCase, routeTestJWTSecret, nil, nil, 100, time.Second)

	// Verify business routes are registered and health routes are not
	routes := router.Routes()
	routePaths := make(map[string]bool, len(routes))
	for _, route := range routes {
		routePaths[route.Path] = true
	}

	assert.False(t, routePaths["/health"], "Health route should not be registered by delivery routes")
	assert.False(t, routePaths["/ready"], "Ready route should not be registered by delivery routes")
	assert.False(t, routePaths["/live"], "Live route should not be registered by delivery routes")
	assert.True(t, routePaths["/api/v1/products"], "Versioned products route should be registered")
	assert.False(t, routePaths["/products"], "Legacy products route should not be registered")
}

// TestRegisterHealthRoutes tests health route registration.
func TestRegisterHealthRoutes(t *testing.T) {
	// This test verifies the function is callable
	// Arrange
	router := setupTestRouter()

	// Create a mock health handler using the constructor
	healthHandler := server.NewHealthHandler(server.HealthHandlerConfig{
		ServiceName: "service-product",
		Version:     "1.0.0",
	})

	// Act - this should not panic
	delivery.RegisterHealthRoutes(router, healthHandler)

	// Assert - verify health routes are registered
	w := httptest.NewRecorder()

	// Test public health endpoint
	req1, _ := http.NewRequest("GET", "/health", nil)
	router.ServeHTTP(w, req1)
	assert.Equal(t, http.StatusOK, w.Code)

	// Test ready endpoint
	w = httptest.NewRecorder()
	req2, _ := http.NewRequest("GET", "/ready", nil)
	router.ServeHTTP(w, req2)
	assert.Equal(t, http.StatusOK, w.Code)

	// Test live endpoint
	w = httptest.NewRecorder()
	req3, _ := http.NewRequest("GET", "/live", nil)
	router.ServeHTTP(w, req3)
	assert.Equal(t, http.StatusOK, w.Code)

	// Test admin health endpoint
	w = httptest.NewRecorder()
	req4, _ := http.NewRequest("GET", "/admin/health", nil)
	router.ServeHTTP(w, req4)
	assert.Equal(t, http.StatusOK, w.Code)
}
