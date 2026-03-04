// Package delivery provides HTTP handlers for the user service.
package delivery

import (
	"strconv"

	"github.com/gin-gonic/gin"

	"github.com/ignata/go-microservices-boilerplate/internal/user/domain"
	"github.com/ignata/go-microservices-boilerplate/internal/user/dto"
	"github.com/ignata/go-microservices-boilerplate/internal/user/usecase"
	"github.com/ignata/go-microservices-boilerplate/pkg/utils"
)

// UserHandler handles user-related HTTP requests.
type UserHandler struct {
	userUseCase usecase.UserUseCase
}

// NewUserHandler creates a new user handler.
func NewUserHandler(userUseCase usecase.UserUseCase) *UserHandler {
	return &UserHandler{
		userUseCase: userUseCase,
	}
}

// UpdateProfile handles profile update requests.
// @Summary Update user profile
// @Description Update a user's profile information
// @Tags users
// @Accept json
// @Produce json
// @Param request body dto.UpdateProfileRequest true "Profile update request"
// @Success 200 {object} dto.ProfileResponse
// @Failure 400 {object} utils.Response
// @Failure 404 {object} utils.Response
// @Failure 500 {object} utils.Response
// @Router /api/v1/users/profile [put]
func (h *UserHandler) UpdateProfile(c *gin.Context) {
	var req dto.UpdateProfileRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.BadRequest(c, "Invalid request body", err.Error())
		return
	}

	// Extract user ID from context (set by auth middleware)
	userID, exists := c.Get("user_id")
	if !exists {
		utils.Unauthorized(c, "Unauthorized")
		return
	}
	req.UserID = userID.(string)

	// Extract request info
	req.IPAddress = c.ClientIP()
	req.UserAgent = c.GetHeader("User-Agent")

	if err := h.userUseCase.UpdateProfile(c.Request.Context(), &req); err != nil {
		if domain.IsNotFoundError(err) {
			utils.NotFound(c, "User")
			return
		}
		if domain.IsValidationError(err) {
			utils.ValidationError(c, err.Error())
			return
		}
		utils.InternalError(c, "Failed to update profile")
		return
	}

	utils.OK(c, dto.MessageResponse{Message: "Profile updated successfully"})
}

// GetProfile handles profile retrieval requests.
// @Summary Get user profile
// @Description Get a user's profile information
// @Tags users
// @Produce json
// @Param id path string true "User ID"
// @Success 200 {object} dto.ProfileResponse
// @Failure 400 {object} utils.Response
// @Failure 404 {object} utils.Response
// @Failure 500 {object} utils.Response
// @Router /api/v1/users/{id}/profile [get]
func (h *UserHandler) GetProfile(c *gin.Context) {
	req := dto.GetUserRequest{
		ID: c.Param("id"),
	}

	profile, err := h.userUseCase.GetProfile(c.Request.Context(), &req)
	if err != nil {
		if domain.IsNotFoundError(err) {
			utils.NotFound(c, "Profile")
			return
		}
		utils.InternalError(c, "Failed to get profile")
		return
	}

	utils.OK(c, profile)
}

// GetUser handles user retrieval requests.
// @Summary Get user by ID
// @Description Get a user by ID
// @Tags users
// @Produce json
// @Param id path string true "User ID"
// @Param include_deleted query bool false "Include deleted users"
// @Success 200 {object} dto.UserResponse
// @Failure 400 {object} utils.Response
// @Failure 404 {object} utils.Response
// @Failure 500 {object} utils.Response
// @Router /api/v1/users/{id} [get]
func (h *UserHandler) GetUser(c *gin.Context) {
	req := dto.GetUserRequest{
		ID: c.Param("id"),
	}

	if includeDeleted := c.Query("include_deleted"); includeDeleted == "true" {
		req.IncludeDeleted = true
	}

	user, err := h.userUseCase.GetUser(c.Request.Context(), &req)
	if err != nil {
		if domain.IsNotFoundError(err) {
			utils.NotFound(c, "User")
			return
		}
		utils.InternalError(c, "Failed to get user")
		return
	}

	utils.OK(c, user)
}

// ListUsers handles user list requests.
// @Summary List users
// @Description Get a paginated list of users
// @Tags users
// @Produce json
// @Param page query int false "Page number" default(1)
// @Param limit query int false "Items per page" default(20)
// @Param role query string false "Filter by role"
// @Param search query string false "Search by email or name"
// @Param include_deleted query bool false "Include deleted users"
// @Param only_deleted query bool false "Only show deleted users"
// @Success 200 {object} dto.UserListResponse
// @Failure 400 {object} utils.Response
// @Failure 500 {object} utils.Response
// @Router /api/v1/users [get]
func (h *UserHandler) ListUsers(c *gin.Context) {
	req := dto.ListUsersRequest{}

	// Parse pagination
	if page, err := strconv.Atoi(c.DefaultQuery("page", "1")); err == nil {
		req.Page = page
	}
	if limit, err := strconv.Atoi(c.DefaultQuery("limit", "20")); err == nil {
		req.Limit = limit
	}

	// Parse filters
	req.Role = c.Query("role")
	req.Search = c.Query("search")

	// Parse paranoid options
	if includeDeleted := c.Query("include_deleted"); includeDeleted == "true" {
		req.IncludeDeleted = true
	}
	if onlyDeleted := c.Query("only_deleted"); onlyDeleted == "true" {
		req.OnlyDeleted = true
	}

	users, err := h.userUseCase.ListUsers(c.Request.Context(), &req)
	if err != nil {
		utils.InternalError(c, "Failed to list users")
		return
	}

	utils.OK(c, users)
}

// ActivateUser handles user activation requests.
// @Summary Activate user
// @Description Activate a user account
// @Tags users
// @Produce json
// @Param id path string true "User ID"
// @Success 200 {object} dto.MessageResponse
// @Failure 400 {object} utils.Response
// @Failure 404 {object} utils.Response
// @Failure 500 {object} utils.Response
// @Router /api/v1/users/{id}/activate [post]
func (h *UserHandler) ActivateUser(c *gin.Context) {
	req := dto.ActivateUserRequest{
		ID: c.Param("id"),
	}

	if err := h.userUseCase.ActivateUser(c.Request.Context(), &req); err != nil {
		if domain.IsNotFoundError(err) {
			utils.NotFound(c, "User")
			return
		}
		utils.InternalError(c, "Failed to activate user")
		return
	}

	utils.OK(c, dto.MessageResponse{Message: "User activated successfully"})
}

// DeactivateUser handles user deactivation requests.
// @Summary Deactivate user
// @Description Deactivate a user account
// @Tags users
// @Produce json
// @Param id path string true "User ID"
// @Success 200 {object} dto.MessageResponse
// @Failure 400 {object} utils.Response
// @Failure 404 {object} utils.Response
// @Failure 500 {object} utils.Response
// @Router /api/v1/users/{id}/deactivate [post]
func (h *UserHandler) DeactivateUser(c *gin.Context) {
	req := dto.DeactivateUserRequest{
		ID: c.Param("id"),
	}

	if err := h.userUseCase.DeactivateUser(c.Request.Context(), &req); err != nil {
		if domain.IsNotFoundError(err) {
			utils.NotFound(c, "User")
			return
		}
		utils.InternalError(c, "Failed to deactivate user")
		return
	}

	utils.OK(c, dto.MessageResponse{Message: "User deactivated successfully"})
}

// DeleteUser handles user deletion requests.
// @Summary Delete user
// @Description Soft delete a user account
// @Tags users
// @Produce json
// @Param id path string true "User ID"
// @Success 200 {object} dto.MessageResponse
// @Failure 400 {object} utils.Response
// @Failure 404 {object} utils.Response
// @Failure 500 {object} utils.Response
// @Router /api/v1/users/{id} [delete]
func (h *UserHandler) DeleteUser(c *gin.Context) {
	req := dto.DeleteUserRequest{
		ID: c.Param("id"),
	}

	if force := c.Query("force"); force == "true" {
		req.Force = true
	}

	if err := h.userUseCase.DeleteUser(c.Request.Context(), &req); err != nil {
		if domain.IsNotFoundError(err) {
			utils.NotFound(c, "User")
			return
		}
		utils.InternalError(c, "Failed to delete user")
		return
	}

	utils.OK(c, dto.MessageResponse{Message: "User deleted successfully"})
}

// RestoreUser handles user restoration requests.
// @Summary Restore user
// @Description Restore a soft-deleted user account
// @Tags users
// @Produce json
// @Param id path string true "User ID"
// @Success 200 {object} dto.RestoreResponse
// @Failure 400 {object} utils.Response
// @Failure 404 {object} utils.Response
// @Failure 500 {object} utils.Response
// @Router /api/v1/users/{id}/restore [post]
func (h *UserHandler) RestoreUser(c *gin.Context) {
	req := dto.RestoreUserRequest{
		ID: c.Param("id"),
	}

	result, err := h.userUseCase.RestoreUser(c.Request.Context(), &req)
	if err != nil {
		utils.InternalError(c, "Failed to restore user")
		return
	}

	utils.OK(c, result)
}

// GetActivityLogs handles activity log retrieval requests.
// @Summary Get activity logs
// @Description Get activity logs with optional filtering
// @Tags users
// @Produce json
// @Param user_id query string false "Filter by user ID"
// @Param action query string false "Filter by action"
// @Param resource query string false "Filter by resource"
// @Param page query int false "Page number" default(1)
// @Param limit query int false "Items per page" default(20)
// @Success 200 {object} dto.ActivityLogListResponse
// @Failure 400 {object} utils.Response
// @Failure 500 {object} utils.Response
// @Router /api/v1/activity-logs [get]
func (h *UserHandler) GetActivityLogs(c *gin.Context) {
	req := dto.ListActivityLogsRequest{}

	// Parse pagination
	if page, err := strconv.Atoi(c.DefaultQuery("page", "1")); err == nil {
		req.Page = page
	}
	if limit, err := strconv.Atoi(c.DefaultQuery("limit", "20")); err == nil {
		req.Limit = limit
	}

	// Parse filters
	req.UserID = c.Query("user_id")
	req.Action = c.Query("action")
	req.Resource = c.Query("resource")

	logs, err := h.userUseCase.GetActivityLogs(c.Request.Context(), &req)
	if err != nil {
		utils.InternalError(c, "Failed to get activity logs")
		return
	}

	utils.OK(c, logs)
}
