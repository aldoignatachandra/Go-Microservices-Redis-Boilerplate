// Package delivery provides HTTP handlers for the product service.
package delivery

import (
	"github.com/gin-gonic/gin"

	"github.com/ignata/go-microservices-boilerplate/internal/product/dto"
	"github.com/ignata/go-microservices-boilerplate/internal/product/domain"
	"github.com/ignata/go-microservices-boilerplate/internal/product/usecase"
	"github.com/ignata/go-microservices-boilerplate/pkg/utils"
)

// Handler provides HTTP handlers for product endpoints.
type Handler struct {
	productUseCase usecase.ProductUseCase
}

// NewHandler creates a new handler.
func NewHandler(productUseCase usecase.ProductUseCase) *Handler {
	return &Handler{
		productUseCase: productUseCase,
	}
}

// CreateProduct handles product creation.
// @Summary Create a new product
// @Description Create a new product with name, description, price, stock and category
// @Tags products
// @Accept json
// @Produce json
// @Param request body dto.CreateProductRequest true "Product creation data"
// @Success 201 {object} dto.ProductResponse
// @Failure 400 {object} utils.Response
// @Failure 409 {object} utils.Response
// @Router /products [post]
func (h *Handler) CreateProduct(c *gin.Context) {
	var req dto.CreateProductRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.BadRequest(c, "Invalid request body", err.Error())
		return
	}

	response, err := h.productUseCase.CreateProduct(c.Request.Context(), &req)
	if err != nil {
		h.handleError(c, err)
		return
	}

	utils.Created(c, response)
}

// GetProduct handles getting a product by ID.
// @Summary Get product by ID
// @Description Get a specific product by ID
// @Tags products
// @Produce json
// @Param id path string true "Product ID"
// @Param include_deleted query bool false "Include deleted products"
// @Success 200 {object} dto.ProductResponse
// @Failure 400 {object} utils.Response
// @Failure 404 {object} utils.Response
// @Router /products/{id} [get]
func (h *Handler) GetProduct(c *gin.Context) {
	var req dto.GetProductRequest
	if err := c.ShouldBindUri(&req); err != nil {
		utils.BadRequest(c, "Invalid product ID", err.Error())
		return
	}

	if err := c.ShouldBindQuery(&req); err != nil {
		utils.BadRequest(c, "Invalid query parameters", err.Error())
		return
	}

	response, err := h.productUseCase.GetProduct(c.Request.Context(), &req)
	if err != nil {
		h.handleError(c, err)
		return
	}

	utils.OK(c, response)
}

// ListProducts handles listing products.
// @Summary List products
// @Description List products with pagination and filters
// @Tags products
// @Produce json
// @Param page query int false "Page number" default(1)
// @Param limit query int false "Items per page" default(10)
// @Param status query string false "Filter by status" Enums(ACTIVE, INACTIVE)
// @Param search query string false "Search by name"
// @Param include_deleted query bool false "Include deleted products"
// @Param only_deleted query bool false "Only deleted products"
// @Success 200 {object} dto.ProductListResponse
// @Router /products [get]
func (h *Handler) ListProducts(c *gin.Context) {
	var req dto.ListProductsRequest
	if err := c.ShouldBindQuery(&req); err != nil {
		utils.BadRequest(c, "Invalid query parameters", err.Error())
		return
	}

	response, err := h.productUseCase.ListProducts(c.Request.Context(), &req)
	if err != nil {
		h.handleError(c, err)
		return
	}

	utils.OK(c, response)
}

// UpdateProduct handles updating a product.
// @Summary Update product
// @Description Update product details
// @Tags products
// @Accept json
// @Produce json
// @Param id path string true "Product ID"
// @Param request body dto.UpdateProductRequest true "Product update data"
// @Success 200 {object} dto.ProductResponse
// @Failure 400 {object} utils.Response
// @Failure 404 {object} utils.Response
// @Router /products/{id} [put]
func (h *Handler) UpdateProduct(c *gin.Context) {
	var req dto.UpdateProductRequest
	if err := c.ShouldBindUri(&req); err != nil {
		utils.BadRequest(c, "Invalid product ID", err.Error())
		return
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		utils.BadRequest(c, "Invalid request body", err.Error())
		return
	}

	response, err := h.productUseCase.UpdateProduct(c.Request.Context(), c.Param("id"), &req)
	if err != nil {
		h.handleError(c, err)
		return
	}

	utils.OK(c, response)
}

// DeleteProduct handles deleting a product.
// @Summary Delete product
// @Description Delete a product (soft delete by default)
// @Tags products
// @Produce json
// @Param id path string true "Product ID"
// @Param force query bool false "Force hard delete"
// @Success 200 {object} dto.DeleteResponse
// @Failure 400 {object} utils.Response
// @Failure 404 {object} utils.Response
// @Router /products/{id} [delete]
func (h *Handler) DeleteProduct(c *gin.Context) {
	var req dto.DeleteProductRequest
	if err := c.ShouldBindUri(&req); err != nil {
		utils.BadRequest(c, "Invalid product ID", err.Error())
		return
	}

	if err := c.ShouldBindQuery(&req); err != nil {
		utils.BadRequest(c, "Invalid query parameters", err.Error())
		return
	}

	response, err := h.productUseCase.DeleteProduct(c.Request.Context(), &req)
	if err != nil {
		h.handleError(c, err)
		return
	}

	utils.OK(c, response)
}

// RestoreProduct handles restoring a deleted product.
// @Summary Restore product
// @Description Restore a soft-deleted product
// @Tags products
// @Produce json
// @Param id path string true "Product ID"
// @Success 200 {object} dto.ProductResponse
// @Failure 400 {object} utils.Response
// @Failure 404 {object} utils.Response
// @Router /products/{id}/restore [post]
func (h *Handler) RestoreProduct(c *gin.Context) {
	var req dto.RestoreProductRequest
	if err := c.ShouldBindUri(&req); err != nil {
		utils.BadRequest(c, "Invalid product ID", err.Error())
		return
	}

	response, err := h.productUseCase.RestoreProduct(c.Request.Context(), &req)
	if err != nil {
		h.handleError(c, err)
		return
	}

	utils.OK(c, response)
}

// UpdateStock handles updating product stock.
// @Summary Update product stock
// @Description Update product stock quantity
// @Tags products
// @Accept json
// @Produce json
// @Param id path string true "Product ID"
// @Param request body dto.UpdateStockRequest true "Stock update data"
// @Success 200 {object} dto.UpdateStockResponse
// @Failure 400 {object} utils.Response
// @Failure 404 {object} utils.Response
// @Router /products/{id}/stock [put]
func (h *Handler) UpdateStock(c *gin.Context) {
	var req dto.UpdateStockRequest
	if err := c.ShouldBindUri(&req); err != nil {
		utils.BadRequest(c, "Invalid product ID", err.Error())
		return
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		utils.BadRequest(c, "Invalid request body", err.Error())
		return
	}

	response, err := h.productUseCase.UpdateStock(c.Request.Context(), &req)
	if err != nil {
		h.handleError(c, err)
		return
	}

	utils.OK(c, response)
}

// handleError handles errors and sends appropriate responses.
func (h *Handler) handleError(c *gin.Context, err error) {
	switch {
	case domain.IsNotFoundError(err):
		utils.NotFound(c, "Product")
	case domain.IsValidationError(err):
		utils.ValidationError(c, err.Error())
	case err == domain.ErrProductNameAlreadyUsed:
		utils.Conflict(c, "Product name already in use")
	default:
		utils.InternalError(c, "An unexpected error occurred")
	}
}