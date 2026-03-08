// Package delivery tests HTTP handlers for the product service.
package delivery_test

import (
	"bytes"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"github.com/ignata/go-microservices-boilerplate/internal/product/delivery"
	"github.com/ignata/go-microservices-boilerplate/internal/product/domain"
	"github.com/ignata/go-microservices-boilerplate/internal/product/dto"
	productusecasemocks "github.com/ignata/go-microservices-boilerplate/internal/product/usecase/mocks"
)

// setupTestRouter creates a test router with Gin in test mode.
func setupTestRouter() *gin.Engine {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	return router
}

// TestCreateProduct_Success tests successful product creation.
func TestCreateProduct_Success(t *testing.T) {
	// Arrange
	mockUseCase := new(productusecasemocks.ProductUseCase)
	handler := delivery.NewHandler(mockUseCase)
	router := setupTestRouter()

	expectedResponse := &dto.ProductResponse{
		ID:          "550e8400-e29b-41d4-a716-446655440001",
		Name:        "Test Product",
		Description: "A test product",
		Price:       29.99,
		Stock:       100,
		Status:      "ACTIVE",
		CategoryID:  "550e8400-e29b-41d4-a716-446655440000",
	}

	mockUseCase.On("CreateProduct", mock.Anything, mock.AnythingOfType("*dto.CreateProductRequest")).
		Return(expectedResponse, nil)

	// Act
	reqBody := map[string]interface{}{
		"name":        "Test Product",
		"description": "A test product",
		"price":       29.99,
		"stock":       100,
		"category_id": "550e8400-e29b-41d4-a716-446655440000",
	}
	bodyBytes, _ := json.Marshal(reqBody)
	req, _ := http.NewRequest("POST", "/products", bytes.NewBuffer(bodyBytes))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	router.POST("/products", handler.CreateProduct)
	router.ServeHTTP(w, req)

	// Assert
	assert.Equal(t, http.StatusCreated, w.Code)

	var response map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(t, err)

	assert.True(t, response["Success"].(bool))
	data := response["Data"].(map[string]interface{})
	assert.Equal(t, "550e8400-e29b-41d4-a716-446655440001", data["id"])
	assert.Equal(t, "Test Product", data["name"])
	assert.Equal(t, 29.99, data["price"])

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

	mockUseCase.On("CreateProduct", mock.Anything, mock.AnythingOfType("*dto.CreateProductRequest")).
		Return(nil, domain.ErrProductNameAlreadyUsed)

	// Act
	reqBody := map[string]interface{}{
		"name":        "Existing Product",
		"description": "A product",
		"price":       19.99,
		"stock":       50,
		"category_id": "550e8400-e29b-41d4-a716-446655440000",
	}
	bodyBytes, _ := json.Marshal(reqBody)
	req, _ := http.NewRequest("POST", "/products", bytes.NewBuffer(bodyBytes))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	router.POST("/products", handler.CreateProduct)
	router.ServeHTTP(w, req)

	// Assert
	assert.Equal(t, http.StatusConflict, w.Code)

	var response map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(t, err)
	assert.False(t, response["Success"].(bool))

	mockUseCase.AssertExpectations(t)
}

// TestGetProduct_Success tests successful product retrieval.
func TestGetProduct_Success(t *testing.T) {
	// Arrange
	mockUseCase := new(productusecasemocks.ProductUseCase)
	handler := delivery.NewHandler(mockUseCase)
	router := setupTestRouter()

	expectedResponse := &dto.ProductResponse{
		ID:          "550e8400-e29b-41d4-a716-446655440001",
		Name:        "Test Product",
		Description: "A test product",
		Price:       29.99,
		Stock:       100,
		Status:      "ACTIVE",
		CategoryID:  "550e8400-e29b-41d4-a716-446655440000",
	}

	mockUseCase.On("GetProduct", mock.Anything, mock.AnythingOfType("*dto.GetProductRequest")).
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

	assert.True(t, response["Success"].(bool))
	data := response["Data"].(map[string]interface{})
	assert.Equal(t, "550e8400-e29b-41d4-a716-446655440001", data["id"])

	mockUseCase.AssertExpectations(t)
}

// TestGetProduct_NotFound tests product not found scenario.
func TestGetProduct_NotFound(t *testing.T) {
	// Arrange
	mockUseCase := new(productusecasemocks.ProductUseCase)
	handler := delivery.NewHandler(mockUseCase)
	router := setupTestRouter()

	mockUseCase.On("GetProduct", mock.Anything, mock.AnythingOfType("*dto.GetProductRequest")).
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

	assert.False(t, response["Success"].(bool))

	mockUseCase.AssertExpectations(t)
}

// TestGetProduct_IncludeDeleted tests getting a deleted product.
func TestGetProduct_IncludeDeleted(t *testing.T) {
	// Arrange
	mockUseCase := new(productusecasemocks.ProductUseCase)
	handler := delivery.NewHandler(mockUseCase)
	router := setupTestRouter()

	expectedResponse := &dto.ProductResponse{
		ID:          "550e8400-e29b-41d4-a716-446655440001",
		Name:        "Deleted Product",
		Description: "A deleted product",
		Price:       29.99,
		Stock:       0,
		Status:      "DELETED",
		CategoryID:  "550e8400-e29b-41d4-a716-446655440000",
	}

	mockUseCase.On("GetProduct", mock.Anything, mock.MatchedBy(func(r *dto.GetProductRequest) bool {
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

	assert.True(t, response["Success"].(bool))

	mockUseCase.AssertExpectations(t)
}

// TestListProducts_Success tests successful product list retrieval.
func TestListProducts_Success(t *testing.T) {
	// Arrange
	mockUseCase := new(productusecasemocks.ProductUseCase)
	handler := delivery.NewHandler(mockUseCase)
	router := setupTestRouter()

	expectedResponse := &dto.ProductListResponse{
		Products: []*dto.ProductResponse{
			{
				ID:         "prod-1",
				Name:       "Product 1",
				Price:      10.99,
				Stock:      50,
				Status:     "ACTIVE",
				CategoryID: "cat-1",
			},
			{
				ID:         "prod-2",
				Name:       "Product 2",
				Price:      20.99,
				Stock:      100,
				Status:     "ACTIVE",
				CategoryID: "cat-1",
			},
		},
		Total:      2,
		Page:       1,
		Limit:      10,
		TotalPages: 1,
	}

	mockUseCase.On("ListProducts", mock.Anything, mock.AnythingOfType("*dto.ListProductsRequest")).
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

	assert.True(t, response["Success"].(bool))
	data := response["Data"].(map[string]interface{})
	assert.NotNil(t, data["products"])

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
			name:  "filter by status",
			query: "/products?status=ACTIVE",
			setup: func(m *productusecasemocks.ProductUseCase) {
				m.On("ListProducts", mock.Anything, mock.MatchedBy(func(r *dto.ListProductsRequest) bool {
					return r.Status == "ACTIVE"
				})).Return(&dto.ProductListResponse{}, nil)
			},
		},
		{
			name:  "filter by search",
			query: "/products?search=laptop",
			setup: func(m *productusecasemocks.ProductUseCase) {
				m.On("ListProducts", mock.Anything, mock.MatchedBy(func(r *dto.ListProductsRequest) bool {
					return r.Search == "laptop"
				})).Return(&dto.ProductListResponse{}, nil)
			},
		},
		{
			name:  "include deleted",
			query: "/products?include_deleted=true",
			setup: func(m *productusecasemocks.ProductUseCase) {
				m.On("ListProducts", mock.Anything, mock.MatchedBy(func(r *dto.ListProductsRequest) bool {
					return r.IncludeDeleted == true
				})).Return(&dto.ProductListResponse{}, nil)
			},
		},
		{
			name:  "only deleted",
			query: "/products?only_deleted=true",
			setup: func(m *productusecasemocks.ProductUseCase) {
				m.On("ListProducts", mock.Anything, mock.MatchedBy(func(r *dto.ListProductsRequest) bool {
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
		ID:          "550e8400-e29b-41d4-a716-446655440001",
		Name:        "Updated Product",
		Description: "Updated description",
		Price:       39.99,
		Stock:       150,
		Status:      "ACTIVE",
		CategoryID:  "550e8400-e29b-41d4-a716-446655440000",
	}

	mockUseCase.On("UpdateProduct", mock.Anything, "550e8400-e29b-41d4-a716-446655440001", mock.AnythingOfType("*dto.UpdateProductRequest")).
		Return(expectedResponse, nil)

	// Act
	reqBody := map[string]interface{}{
		"name":        "Updated Product",
		"description": "Updated description",
		"price":       39.99,
		"stock":       150,
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

	assert.True(t, response["Success"].(bool))
	data := response["Data"].(map[string]interface{})
	assert.Equal(t, "Updated Product", data["name"])

	mockUseCase.AssertExpectations(t)
}

// TestUpdateProduct_NotFound tests updating a non-existent product.
func TestUpdateProduct_NotFound(t *testing.T) {
	// Arrange
	mockUseCase := new(productusecasemocks.ProductUseCase)
	handler := delivery.NewHandler(mockUseCase)
	router := setupTestRouter()

	mockUseCase.On("UpdateProduct", mock.Anything, "550e8400-e29b-41d4-a716-446655440002", mock.AnythingOfType("*dto.UpdateProductRequest")).
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

	assert.False(t, response["Success"].(bool))

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

	mockUseCase.On("DeleteProduct", mock.Anything, mock.MatchedBy(func(r *dto.DeleteProductRequest) bool {
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

	assert.True(t, response["Success"].(bool))
	data := response["Data"].(map[string]interface{})
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

	mockUseCase.On("DeleteProduct", mock.Anything, mock.MatchedBy(func(r *dto.DeleteProductRequest) bool {
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

	assert.True(t, response["Success"].(bool))
	data := response["Data"].(map[string]interface{})
	assert.Equal(t, "Product permanently deleted", data["message"])

	mockUseCase.AssertExpectations(t)
}

// TestDeleteProduct_NotFound tests deleting a non-existent product.
func TestDeleteProduct_NotFound(t *testing.T) {
	// Arrange
	mockUseCase := new(productusecasemocks.ProductUseCase)
	handler := delivery.NewHandler(mockUseCase)
	router := setupTestRouter()

	mockUseCase.On("DeleteProduct", mock.Anything, mock.AnythingOfType("*dto.DeleteProductRequest")).
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

	assert.False(t, response["Success"].(bool))

	mockUseCase.AssertExpectations(t)
}

// TestRestoreProduct_Success tests successful product restoration.
func TestRestoreProduct_Success(t *testing.T) {
	// Arrange
	mockUseCase := new(productusecasemocks.ProductUseCase)
	handler := delivery.NewHandler(mockUseCase)
	router := setupTestRouter()

	expectedResponse := &dto.ProductResponse{
		ID:          "550e8400-e29b-41d4-a716-446655440001",
		Name:        "Restored Product",
		Description: "A restored product",
		Price:       29.99,
		Stock:       100,
		Status:      "ACTIVE",
		CategoryID:  "550e8400-e29b-41d4-a716-446655440000",
	}

	mockUseCase.On("RestoreProduct", mock.Anything, mock.AnythingOfType("*dto.RestoreProductRequest")).
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

	assert.True(t, response["Success"].(bool))
	data := response["Data"].(map[string]interface{})
	assert.Equal(t, "550e8400-e29b-41d4-a716-446655440001", data["id"])

	mockUseCase.AssertExpectations(t)
}

// TestRestoreProduct_NotFound tests restoring a non-existent product.
func TestRestoreProduct_NotFound(t *testing.T) {
	// Arrange
	mockUseCase := new(productusecasemocks.ProductUseCase)
	handler := delivery.NewHandler(mockUseCase)
	router := setupTestRouter()

	mockUseCase.On("RestoreProduct", mock.Anything, mock.AnythingOfType("*dto.RestoreProductRequest")).
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

	assert.False(t, response["Success"].(bool))

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

	mockUseCase.On("UpdateStock", mock.Anything, mock.MatchedBy(func(r *dto.UpdateStockRequest) bool {
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

	assert.True(t, response["Success"].(bool))
	data, ok := response["Data"].(map[string]interface{})
	require.True(t, ok, "data field should be present and a map")
	assert.Equal(t, float64(200), data["stock"])

	mockUseCase.AssertExpectations(t)
}

// TestUpdateStock_NotFound tests updating stock for non-existent product.
func TestUpdateStock_NotFound(t *testing.T) {
	// Arrange
	mockUseCase := new(productusecasemocks.ProductUseCase)
	handler := delivery.NewHandler(mockUseCase)
	router := setupTestRouter()

	mockUseCase.On("UpdateStock", mock.Anything, mock.MatchedBy(func(r *dto.UpdateStockRequest) bool {
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

	assert.False(t, response["Success"].(bool))

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
	mockUseCase.On("UpdateStock", mock.Anything, mock.MatchedBy(func(r *dto.UpdateStockRequest) bool {
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

	assert.False(t, response["Success"].(bool))
	errObj := response["Error"].(map[string]interface{})
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
	mockUseCase.On("CreateProduct", mock.Anything, mock.AnythingOfType("*dto.CreateProductRequest")).
		Return(nil, validationErr)

	// Act - use valid data to pass Gin binding
	reqBody := map[string]interface{}{
		"name":        "Test Product",
		"price":       29.99,
		"stock":       100,
		"category_id": "550e8400-e29b-41d4-a716-446655440000",
	}
	bodyBytes, _ := json.Marshal(reqBody)
	req, _ := http.NewRequest("POST", "/products", bytes.NewBuffer(bodyBytes))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	router.POST("/products", handler.CreateProduct)
	router.ServeHTTP(w, req)

	// Assert
	assert.Equal(t, http.StatusUnprocessableEntity, w.Code)

	var response map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(t, err)

	assert.False(t, response["Success"].(bool))
	errObj := response["Error"].(map[string]interface{})
	assert.Equal(t, "VALIDATION_ERROR", errObj["code"])

	mockUseCase.AssertExpectations(t)
}

// TestCreateProduct_InternalError tests create product with internal error.
func TestCreateProduct_InternalError(t *testing.T) {
	// Arrange
	mockUseCase := new(productusecasemocks.ProductUseCase)
	handler := delivery.NewHandler(mockUseCase)
	router := setupTestRouter()

	mockUseCase.On("CreateProduct", mock.Anything, mock.AnythingOfType("*dto.CreateProductRequest")).
		Return(nil, errors.New("database connection failed"))

	// Act
	reqBody := map[string]interface{}{
		"name":        "Test Product",
		"description": "A test product",
		"price":       29.99,
		"stock":       100,
		"category_id": "550e8400-e29b-41d4-a716-446655440000",
	}
	bodyBytes, _ := json.Marshal(reqBody)
	req, _ := http.NewRequest("POST", "/products", bytes.NewBuffer(bodyBytes))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	router.POST("/products", handler.CreateProduct)
	router.ServeHTTP(w, req)

	// Assert
	assert.Equal(t, http.StatusInternalServerError, w.Code)

	var response map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(t, err)
	assert.False(t, response["Success"].(bool))

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
	assert.False(t, response["Success"].(bool))
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
			assert.False(t, response["Success"].(bool))
		})
	}
}

// TestListProducts_InternalError tests list products with internal error.
func TestListProducts_InternalError(t *testing.T) {
	// Arrange
	mockUseCase := new(productusecasemocks.ProductUseCase)
	handler := delivery.NewHandler(mockUseCase)
	router := setupTestRouter()

	mockUseCase.On("ListProducts", mock.Anything, mock.AnythingOfType("*dto.ListProductsRequest")).
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
	assert.False(t, response["Success"].(bool))

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
	assert.False(t, response["Success"].(bool))
}

// TestUpdateProduct_InternalError tests update product with internal error.
func TestUpdateProduct_InternalError(t *testing.T) {
	// Arrange
	mockUseCase := new(productusecasemocks.ProductUseCase)
	handler := delivery.NewHandler(mockUseCase)
	router := setupTestRouter()

	mockUseCase.On("UpdateProduct", mock.Anything, "550e8400-e29b-41d4-a716-446655440001", mock.AnythingOfType("*dto.UpdateProductRequest")).
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
	assert.False(t, response["Success"].(bool))

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
	assert.False(t, response["Success"].(bool))
}

// TestDeleteProduct_InternalError tests delete product with internal error.
func TestDeleteProduct_InternalError(t *testing.T) {
	// Arrange
	mockUseCase := new(productusecasemocks.ProductUseCase)
	handler := delivery.NewHandler(mockUseCase)
	router := setupTestRouter()

	mockUseCase.On("DeleteProduct", mock.Anything, mock.AnythingOfType("*dto.DeleteProductRequest")).
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
	assert.False(t, response["Success"].(bool))

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
	assert.False(t, response["Success"].(bool))
}

// TestRestoreProduct_InternalError tests restore product with internal error.
func TestRestoreProduct_InternalError(t *testing.T) {
	// Arrange
	mockUseCase := new(productusecasemocks.ProductUseCase)
	handler := delivery.NewHandler(mockUseCase)
	router := setupTestRouter()

	mockUseCase.On("RestoreProduct", mock.Anything, mock.AnythingOfType("*dto.RestoreProductRequest")).
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
	assert.False(t, response["Success"].(bool))

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
	assert.False(t, response["Success"].(bool))
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
	assert.False(t, response["Success"].(bool))
}

// TestUpdateStock_InternalError tests update stock with internal error.
func TestUpdateStock_InternalError(t *testing.T) {
	// Arrange
	mockUseCase := new(productusecasemocks.ProductUseCase)
	handler := delivery.NewHandler(mockUseCase)
	router := setupTestRouter()

	mockUseCase.On("UpdateStock", mock.Anything, mock.AnythingOfType("*dto.UpdateStockRequest")).
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
	assert.False(t, response["Success"].(bool))

	mockUseCase.AssertExpectations(t)
}
