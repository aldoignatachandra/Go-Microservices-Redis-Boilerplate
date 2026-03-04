// Package delivery provides HTTP handlers for the auth service.
package delivery

import (
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"github.com/ignata/go-microservices-boilerplate/internal/auth/dto"
	"github.com/ignata/go-microservices-boilerplate/internal/auth/domain"
	"github.com/ignata/go-microservices-boilerplate/internal/auth/usecase"
	"github.com/ignata/go-microservices-boilerplate/pkg/utils"
)

// Handler provides HTTP handlers for auth endpoints.
type Handler struct {
	authUseCase usecase.AuthUseCase
}

// NewHandler creates a new handler.
func NewHandler(authUseCase usecase.AuthUseCase) *Handler {
	return &Handler{
		authUseCase: authUseCase,
	}
}

// Register handles user registration.
// @Summary Register a new user
// @Description Register a new user with email and password
// @Tags auth
// @Accept json
// @Produce json
// @Param request body dto.RegisterRequest true "Registration data"
// @Success 201 {object} dto.AuthResponse
// @Failure 400 {object} utils.Response
// @Failure 409 {object} utils.Response
// @Router /auth/register [post]
func (h *Handler) Register(c *gin.Context) {
	var req dto.RegisterRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.BadRequest(c, "Invalid request body", err.Error())
		return
	}

	response, err := h.authUseCase.Register(c.Request.Context(), &req)
	if err != nil {
		h.handleError(c, err)
		return
	}

	utils.Created(c, response)
}

// Login handles user login.
// @Summary Login
// @Description Authenticate user and return tokens
// @Tags auth
// @Accept json
// @Produce json
// @Param request body dto.LoginRequest true "Login credentials"
// @Success 200 {object} dto.AuthResponse
// @Failure 400 {object} utils.Response
// @Failure 401 {object} utils.Response
// @Router /auth/login [post]
func (h *Handler) Login(c *gin.Context) {
	var req dto.LoginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.BadRequest(c, "Invalid request body", err.Error())
		return
	}

	ipAddress := c.ClientIP()
	userAgent := c.GetHeader("User-Agent")

	response, err := h.authUseCase.Login(c.Request.Context(), &req, ipAddress, userAgent)
	if err != nil {
		h.handleError(c, err)
		return
	}

	utils.OK(c, response)
}

// Logout handles user logout.
// @Summary Logout
// @Description Logout user and invalidate session
// @Tags auth
// @Produce json
// @Success 200 {object} utils.Response
// @Failure 401 {object} utils.Response
// @Router /auth/logout [post]
// @Security BearerAuth
func (h *Handler) Logout(c *gin.Context) {
	userID := c.GetString("user_id")
	if userID == "" {
		utils.Unauthorized(c, "")
		return
	}

	if err := h.authUseCase.Logout(c.Request.Context(), userID); err != nil {
		h.handleError(c, err)
		return
	}

	utils.OK(c, dto.MessageResponse{Message: "Successfully logged out"})
}

// RefreshToken handles token refresh.
// @Summary Refresh token
// @Description Refresh access token using refresh token
// @Tags auth
// @Accept json
// @Produce json
// @Param request body dto.RefreshTokenRequest true "Refresh token"
// @Success 200 {object} dto.AuthResponse
// @Failure 400 {object} utils.Response
// @Failure 401 {object} utils.Response
// @Router /auth/refresh [post]
func (h *Handler) RefreshToken(c *gin.Context) {
	var req dto.RefreshTokenRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.BadRequest(c, "Invalid request body", err.Error())
		return
	}

	response, err := h.authUseCase.RefreshToken(c.Request.Context(), &req)
	if err != nil {
		h.handleError(c, err)
		return
	}

	utils.OK(c, response)
}

// GetCurrentUser gets the current authenticated user.
// @Summary Get current user
// @Description Get the currently authenticated user's profile
// @Tags auth
// @Produce json
// @Success 200 {object} dto.UserResponse
// @Failure 401 {object} utils.Response
// @Router /auth/me [get]
// @Security BearerAuth
func (h *Handler) GetCurrentUser(c *gin.Context) {
	userID := c.GetString("user_id")
	if userID == "" {
		utils.Unauthorized(c, "")
		return
	}

	response, err := h.authUseCase.GetCurrentUser(c.Request.Context(), userID)
	if err != nil {
		h.handleError(c, err)
		return
	}

	utils.OK(c, response)
}

// ChangePassword changes the user's password.
// @Summary Change password
// @Description Change the current user's password
// @Tags auth
// @Accept json
// @Produce json
// @Param request body dto.ChangePasswordRequest true "Password change data"
// @Success 200 {object} utils.Response
// @Failure 400 {object} utils.Response
// @Failure 401 {object} utils.Response
// @Router /auth/change-password [post]
// @Security BearerAuth
func (h *Handler) ChangePassword(c *gin.Context) {
	userID := c.GetString("user_id")
	if userID == "" {
		utils.Unauthorized(c, "")
		return
	}

	var req dto.ChangePasswordRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.BadRequest(c, "Invalid request body", err.Error())
		return
	}

	if err := h.authUseCase.ChangePassword(c.Request.Context(), userID, &req); err != nil {
		h.handleError(c, err)
		return
	}

	utils.OK(c, dto.MessageResponse{Message: "Password changed successfully"})
}

// GetUser gets a user by ID (admin only).
// @Summary Get user by ID
// @Description Get a specific user by ID (admin only)
// @Tags admin
// @Produce json
// @Param id path string true "User ID"
// @Param include_deleted query bool false "Include deleted users"
// @Success 200 {object} dto.UserResponse
// @Failure 400 {object} utils.Response
// @Failure 401 {object} utils.Response
// @Failure 404 {object} utils.Response
// @Router /admin/users/{id} [get]
// @Security BearerAuth
func (h *Handler) GetUser(c *gin.Context) {
	var req dto.GetUserRequest
	if err := c.ShouldBindUri(&req); err != nil {
		utils.BadRequest(c, "Invalid user ID", err.Error())
		return
	}

	if err := c.ShouldBindQuery(&req); err != nil {
		utils.BadRequest(c, "Invalid query parameters", err.Error())
		return
	}

	response, err := h.authUseCase.GetUser(c.Request.Context(), &req)
	if err != nil {
		h.handleError(c, err)
		return
	}

	utils.OK(c, response)
}

// ListUsers lists all users (admin only).
// @Summary List users
// @Description List all users with pagination (admin only)
// @Tags admin
// @Produce json
// @Param page query int false "Page number" default(1)
// @Param limit query int false "Items per page" default(10)
// @Param role query string false "Filter by role" Enums(ADMIN, USER)
// @Param search query string false "Search by email"
// @Param include_deleted query bool false "Include deleted users"
// @Param only_deleted query bool false "Only deleted users"
// @Success 200 {object} dto.UserListResponse
// @Failure 401 {object} utils.Response
// @Router /admin/users [get]
// @Security BearerAuth
func (h *Handler) ListUsers(c *gin.Context) {
	var req dto.ListUsersRequest
	if err := c.ShouldBindQuery(&req); err != nil {
		utils.BadRequest(c, "Invalid query parameters", err.Error())
		return
	}

	response, err := h.authUseCase.ListUsers(c.Request.Context(), &req)
	if err != nil {
		h.handleError(c, err)
		return
	}

	utils.OK(c, response)
}

// DeleteUser deletes a user (admin only).
// @Summary Delete user
// @Description Delete a user (soft delete by default) (admin only)
// @Tags admin
// @Produce json
// @Param id path string true "User ID"
// @Param force query bool false "Force hard delete"
// @Success 200 {object} dto.DeleteResponse
// @Failure 400 {object} utils.Response
// @Failure 401 {object} utils.Response
// @Failure 404 {object} utils.Response
// @Router /admin/users/{id} [delete]
// @Security BearerAuth
func (h *Handler) DeleteUser(c *gin.Context) {
	var req dto.DeleteUserRequest
	if err := c.ShouldBindUri(&req); err != nil {
		utils.BadRequest(c, "Invalid user ID", err.Error())
		return
	}

	if err := c.ShouldBindQuery(&req); err != nil {
		utils.BadRequest(c, "Invalid query parameters", err.Error())
		return
	}

	if err := h.authUseCase.DeleteUser(c.Request.Context(), &req); err != nil {
		h.handleError(c, err)
		return
	}

	message := "User deleted successfully"
	if req.Force {
		message = "User permanently deleted"
	}

	utils.OK(c, dto.DeleteResponse{
		Success: true,
		Message: message,
	})
}

// RestoreUser restores a deleted user (admin only).
// @Summary Restore user
// @Description Restore a soft-deleted user (admin only)
// @Tags admin
// @Produce json
// @Param id path string true "User ID"
// @Success 200 {object} dto.RestoreResponse
// @Failure 400 {object} utils.Response
// @Failure 401 {object} utils.Response
// @Failure 404 {object} utils.Response
// @Router /admin/users/{id}/restore [post]
// @Security BearerAuth
func (h *Handler) RestoreUser(c *gin.Context) {
	var req dto.RestoreUserRequest
	if err := c.ShouldBindUri(&req); err != nil {
		utils.BadRequest(c, "Invalid user ID", err.Error())
		return
	}

	response, err := h.authUseCase.RestoreUser(c.Request.Context(), &req)
	if err != nil {
		h.handleError(c, err)
		return
	}

	utils.OK(c, dto.RestoreResponse{
		Success: true,
		Message: "User restored successfully",
		User:    response,
	})
}

// handleError handles errors and sends appropriate responses.
func (h *Handler) handleError(c *gin.Context, err error) {
	switch {
	case domain.IsNotFoundError(err):
		utils.NotFound(c, "User")
	case domain.IsAuthError(err):
		utils.Unauthorized(c, err.Error())
	case domain.IsValidationError(err):
		utils.ValidationError(c, err.Error())
	case err == domain.ErrEmailAlreadyUsed:
		utils.Conflict(c, "Email already in use")
	case err == domain.ErrUserDeleted:
		utils.Gone(c, "User")
	default:
		utils.InternalError(c, "An unexpected error occurred")
	}
}

// isValidUUID checks if a string is a valid UUID.
func isValidUUID(s string) bool {
	_, err := uuid.Parse(s)
	return err == nil
}
