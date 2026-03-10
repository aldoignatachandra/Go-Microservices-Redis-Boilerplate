// Package usecase provides business logic for the auth service.
package usecase

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"go.uber.org/zap"

	"github.com/ignata/go-microservices-boilerplate/internal/auth/domain"
	"github.com/ignata/go-microservices-boilerplate/internal/auth/dto"
	"github.com/ignata/go-microservices-boilerplate/internal/auth/repository"
	"github.com/ignata/go-microservices-boilerplate/pkg/eventbus"
	"github.com/ignata/go-microservices-boilerplate/pkg/logger"
	"github.com/ignata/go-microservices-boilerplate/pkg/utils"
)

// AuthUseCase defines the interface for auth business logic.
type AuthUseCase interface {
	// Register registers a new user
	Register(ctx context.Context, req *dto.RegisterRequest) (*dto.AuthResponse, error)
	// Login authenticates a user and returns tokens
	Login(ctx context.Context, req *dto.LoginRequest, ipAddress, userAgent string) (*dto.AuthResponse, error)
	// Logout logs out a user by revoking their session
	Logout(ctx context.Context, userID string) error
	// RefreshToken refreshes access token using refresh token
	RefreshToken(ctx context.Context, req *dto.RefreshTokenRequest) (*dto.AuthResponse, error)
	// GetCurrentUser gets the current authenticated user
	GetCurrentUser(ctx context.Context, userID string) (*dto.UserResponse, error)
	// GetUser gets a user by ID
	GetUser(ctx context.Context, req *dto.GetUserRequest) (*dto.UserResponse, error)
	// ListUsers lists users with pagination
	ListUsers(ctx context.Context, req *dto.ListUsersRequest) (*dto.UserListResponse, error)
	// UpdateUser updates a user
	UpdateUser(ctx context.Context, userID string, req *dto.UpdateUserRequest) (*dto.UserResponse, error)
	// ChangePassword changes a user's password
	ChangePassword(ctx context.Context, userID string, req *dto.ChangePasswordRequest) error
	// DeleteUser deletes a user
	DeleteUser(ctx context.Context, req *dto.DeleteUserRequest) error
	// RestoreUser restores a deleted user
	RestoreUser(ctx context.Context, req *dto.RestoreUserRequest) (*dto.UserResponse, error)
}

// Config holds usecase configuration.
type Config struct {
	JWTSecret        string
	JWTExpiresIn     time.Duration
	RefreshExpiresIn time.Duration
	BcryptCost       int
	ServiceName      string
}

// authUseCase implements AuthUseCase.
type authUseCase struct {
	userRepo    repository.UserRepository
	sessionRepo repository.SessionRepository
	eventBus    eventbus.EventPublisher
	jwtManager  *utils.JWTManager
	config      Config
}

// NewAuthUseCase creates a new auth usecase.
func NewAuthUseCase(
	userRepo repository.UserRepository,
	sessionRepo repository.SessionRepository,
	eventBus eventbus.EventPublisher,
	config Config,
) AuthUseCase {
	jwtManager := utils.NewJWTManager(utils.JWTConfig{
		Secret:           config.JWTSecret,
		ExpiresIn:        config.JWTExpiresIn,
		RefreshExpiresIn: config.RefreshExpiresIn,
	})

	return &authUseCase{
		userRepo:    userRepo,
		sessionRepo: sessionRepo,
		eventBus:    eventBus,
		jwtManager:  jwtManager,
		config:      config,
	}
}

// Register registers a new user.
func (uc *authUseCase) Register(ctx context.Context, req *dto.RegisterRequest) (*dto.AuthResponse, error) {
	// Check if user already exists by email
	exists, err := uc.userRepo.ExistsByEmail(ctx, req.Email)
	if err != nil {
		return nil, fmt.Errorf("failed to check user existence: %w", err)
	}
	if exists {
		return nil, domain.ErrEmailAlreadyUsed
	}

	// Check if username already exists
	exists, err = uc.userRepo.ExistsByUsername(ctx, req.Username)
	if err != nil {
		return nil, fmt.Errorf("failed to check username existence: %w", err)
	}
	if exists {
		return nil, domain.ErrUsernameAlreadyUsed
	}

	// Hash password
	passwordHash, err := utils.HashPasswordWithCost(req.Password, uc.config.BcryptCost)
	if err != nil {
		return nil, fmt.Errorf("failed to hash password: %w", err)
	}

	// Create user
	user := &domain.User{
		Email:        req.Email,
		Username:     req.Username,
		Name:         req.Name,
		PasswordHash: passwordHash,
		Role:         domain.RoleUser,
	}

	if err = uc.userRepo.Create(ctx, user); err != nil {
		return nil, fmt.Errorf("failed to create user: %w", err)
	}

	sessionID := uuid.New().String()

	// Generate tokens
	var tokenPair *utils.TokenPair
	tokenPair, err = uc.jwtManager.GenerateTokenPairWithSession(user.ID, user.Email, string(user.Role), sessionID)
	if err != nil {
		return nil, fmt.Errorf("failed to generate tokens: %w", err)
	}

	// Create session (with Token field instead of RefreshToken)
	session := &domain.Session{
		ID:        sessionID,
		UserID:    user.ID,
		Token:     tokenPair.RefreshToken,
		ExpiresAt: time.Now().UTC().Add(uc.config.RefreshExpiresIn),
	}
	if err = uc.sessionRepo.Create(ctx, session); err != nil {
		return nil, fmt.Errorf("failed to create session: %w", err)
	}

	// Publish event
	if uc.eventBus != nil {
		event := domain.NewUserCreatedEvent(user)
		if actorUserID := strings.TrimSpace(utils.GetActorUserIDFromContext(ctx)); actorUserID != "" {
			event.WithMetadata("actor_user_id", actorUserID)
		}
		uc.publishEvent(ctx, event)
	}

	return &dto.AuthResponse{
		Token:        tokenPair.AccessToken,
		RefreshToken: tokenPair.RefreshToken,
		ExpiresIn:    tokenPair.ExpiresIn,
		User:         dto.FromUser(user),
	}, nil
}

// Login authenticates a user.
func (uc *authUseCase) Login(ctx context.Context, req *dto.LoginRequest, ipAddress, userAgent string) (*dto.AuthResponse, error) {
	credential := strings.TrimSpace(req.Email)

	// Find user by email or username (include deleted to check for deleted users)
	user, err := uc.userRepo.FindByEmailOrUsername(ctx, credential, &domain.ParanoidOptions{IncludeDeleted: true})
	if err != nil {
		if errors.Is(err, domain.ErrUserNotFound) {
			return nil, domain.ErrInvalidCredentials
		}
		return nil, fmt.Errorf("failed to find user: %w", err)
	}

	// Check if user can login (soft delete based)
	if !user.CanLogin() {
		if user.DeletedAt.Valid {
			return nil, domain.ErrUserDeleted
		}
		return nil, domain.ErrInvalidCredentials
	}

	// Verify password
	if !utils.CheckPassword(req.Password, user.PasswordHash) {
		return nil, domain.ErrInvalidCredentials
	}

	// SINGLE SESSION POLICY: Delete ALL existing sessions before creating new one
	if err := uc.sessionRepo.DeleteByUserID(ctx, user.ID); err != nil {
		return nil, fmt.Errorf("failed to delete existing sessions: %w", err)
	}

	sessionID := uuid.New().String()

	// Generate tokens
	var tokenPair *utils.TokenPair
	tokenPair, err = uc.jwtManager.GenerateTokenPairWithSession(user.ID, user.Email, string(user.Role), sessionID)
	if err != nil {
		return nil, fmt.Errorf("failed to generate tokens: %w", err)
	}

	// Create session (with Token field instead of RefreshToken)
	session := &domain.Session{
		ID:         sessionID,
		UserID:     user.ID,
		Token:      tokenPair.RefreshToken,
		ExpiresAt:  time.Now().UTC().Add(uc.config.RefreshExpiresIn),
		IPAddress:  ipAddress,
		UserAgent:  userAgent,
		DeviceType: detectDeviceType(userAgent),
	}
	if err = uc.sessionRepo.Create(ctx, session); err != nil {
		return nil, fmt.Errorf("failed to create session: %w", err)
	}

	// Update last login
	user.TouchLastLogin()
	_ = uc.userRepo.Update(ctx, user)

	// Publish event
	if uc.eventBus != nil {
		uc.publishEvent(ctx, domain.NewUserLoggedInEvent(user, ipAddress, userAgent))
	}

	return &dto.AuthResponse{
		Token:        tokenPair.AccessToken,
		RefreshToken: tokenPair.RefreshToken,
		ExpiresIn:    tokenPair.ExpiresIn,
		User:         dto.FromUser(user),
	}, nil
}

// Logout logs out a user.
func (uc *authUseCase) Logout(ctx context.Context, userID string) error {
	// Require an active session so repeated logout with the same JWT is rejected.
	sessions, err := uc.sessionRepo.FindByUserID(ctx, userID)
	if err != nil {
		return fmt.Errorf("failed to validate active session: %w", err)
	}
	if len(sessions) == 0 {
		return domain.ErrInvalidToken
	}

	if err := uc.sessionRepo.RevokeAllForUser(ctx, userID); err != nil {
		return fmt.Errorf("failed to logout: %w", err)
	}

	// Publish event
	if uc.eventBus != nil {
		uc.publishEvent(ctx, domain.NewUserLoggedOutEvent(userID))
	}

	return nil
}

// RefreshToken refreshes access token.
func (uc *authUseCase) RefreshToken(ctx context.Context, req *dto.RefreshTokenRequest) (*dto.AuthResponse, error) {
	// Validate refresh token
	userID, err := uc.jwtManager.ValidateRefreshToken(req.Token)
	if err != nil {
		return nil, domain.ErrInvalidToken
	}

	// Find session
	session, err := uc.sessionRepo.FindByRefreshToken(ctx, req.Token)
	if err != nil {
		return nil, domain.ErrInvalidToken
	}

	// Check if session is valid
	if !session.IsValid() || session.UserID != userID {
		return nil, domain.ErrInvalidToken
	}

	// Get user
	user, err := uc.userRepo.FindByID(ctx, userID, domain.DefaultParanoidOptions())
	if err != nil {
		return nil, domain.ErrInvalidToken
	}

	// Check if user can login
	if !user.CanLogin() {
		return nil, domain.ErrUserInactive
	}

	// Revoke all active sessions so every previous access token is invalid immediately.
	if err := uc.sessionRepo.RevokeAllForUser(ctx, user.ID); err != nil {
		return nil, fmt.Errorf("failed to revoke existing sessions: %w", err)
	}

	newSessionID := uuid.New().String()

	// Generate new tokens
	tokenPair, err := uc.jwtManager.GenerateTokenPairWithSession(user.ID, user.Email, string(user.Role), newSessionID)
	if err != nil {
		return nil, fmt.Errorf("failed to generate tokens: %w", err)
	}

	// Create new session
	newSession := &domain.Session{
		ID:        newSessionID,
		UserID:    user.ID,
		Token:     tokenPair.RefreshToken,
		ExpiresAt: time.Now().UTC().Add(uc.config.RefreshExpiresIn),
	}
	if err := uc.sessionRepo.Create(ctx, newSession); err != nil {
		return nil, fmt.Errorf("failed to create session: %w", err)
	}

	// Publish event
	if uc.eventBus != nil {
		uc.publishEvent(ctx, domain.NewUserRefreshedTokenEvent(user))
	}

	return &dto.AuthResponse{
		Token:        tokenPair.AccessToken,
		RefreshToken: tokenPair.RefreshToken,
		ExpiresIn:    tokenPair.ExpiresIn,
		User:         dto.FromUser(user),
	}, nil
}

// GetCurrentUser gets the current authenticated user.
func (uc *authUseCase) GetCurrentUser(ctx context.Context, userID string) (*dto.UserResponse, error) {
	user, err := uc.userRepo.FindByID(ctx, userID, domain.DefaultParanoidOptions())
	if err != nil {
		return nil, err
	}
	return dto.FromUser(user), nil
}

// GetUser gets a user by ID.
func (uc *authUseCase) GetUser(ctx context.Context, req *dto.GetUserRequest) (*dto.UserResponse, error) {
	user, err := uc.userRepo.FindByID(ctx, req.ID, req.GetParanoidOptions())
	if err != nil {
		return nil, err
	}
	return dto.FromUser(user), nil
}

// ListUsers lists users with pagination.
func (uc *authUseCase) ListUsers(ctx context.Context, req *dto.ListUsersRequest) (*dto.UserListResponse, error) {
	list, err := uc.userRepo.FindAll(ctx, req)
	if err != nil {
		return nil, err
	}
	return dto.FromUserList(list), nil
}

// UpdateUser updates a user.
func (uc *authUseCase) UpdateUser(ctx context.Context, userID string, req *dto.UpdateUserRequest) (*dto.UserResponse, error) {
	user, err := uc.userRepo.FindByID(ctx, userID, domain.DefaultParanoidOptions())
	if err != nil {
		return nil, err
	}

	// Update fields
	if req.Email != "" {
		// Check if email is already used by another user
		existingUser, err := uc.userRepo.FindByEmail(ctx, req.Email, domain.DefaultParanoidOptions())
		if err == nil && existingUser.ID != userID {
			return nil, domain.ErrEmailAlreadyUsed
		}
		user.Email = req.Email
	}

	if req.Name != "" {
		user.Name = req.Name
	}

	if req.Password != "" {
		passwordHash, err := utils.HashPasswordWithCost(req.Password, uc.config.BcryptCost)
		if err != nil {
			return nil, fmt.Errorf("failed to hash password: %w", err)
		}
		user.PasswordHash = passwordHash
	}

	if err := uc.userRepo.Update(ctx, user); err != nil {
		return nil, err
	}

	// Publish event
	if uc.eventBus != nil {
		uc.publishEvent(ctx, domain.NewUserUpdatedEvent(user))
	}

	return dto.FromUser(user), nil
}

// ChangePassword changes a user's password.
func (uc *authUseCase) ChangePassword(ctx context.Context, userID string, req *dto.ChangePasswordRequest) error {
	user, err := uc.userRepo.FindByID(ctx, userID, domain.DefaultParanoidOptions())
	if err != nil {
		return err
	}

	// Verify current password
	if !utils.CheckPassword(req.OldPassword, user.PasswordHash) {
		return domain.ErrInvalidPassword
	}

	// Hash new password
	passwordHash, err := utils.HashPasswordWithCost(req.NewPassword, uc.config.BcryptCost)
	if err != nil {
		return fmt.Errorf("failed to hash password: %w", err)
	}

	user.PasswordHash = passwordHash

	if err := uc.userRepo.Update(ctx, user); err != nil {
		return err
	}

	// Revoke all sessions (force re-login)
	_ = uc.sessionRepo.RevokeAllForUser(ctx, userID)

	// Publish event
	if uc.eventBus != nil {
		uc.publishEvent(ctx, domain.NewUserUpdatedEvent(user).
			WithMetadata("update_type", "password_changed"))
	}

	return nil
}

// DeleteUser deletes a user.
func (uc *authUseCase) DeleteUser(ctx context.Context, req *dto.DeleteUserRequest) error {
	// Check if user exists
	user, err := uc.userRepo.FindByID(ctx, req.ID, &domain.ParanoidOptions{IncludeDeleted: true})
	if err != nil {
		return err
	}

	// Delete user
	if req.Force {
		if err := uc.userRepo.HardDelete(ctx, req.ID); err != nil {
			return err
		}
	} else {
		if err := uc.userRepo.Delete(ctx, req.ID); err != nil {
			return err
		}
	}

	// Revoke all sessions
	_ = uc.sessionRepo.RevokeAllForUser(ctx, req.ID)

	// Publish event
	if uc.eventBus != nil {
		uc.publishEvent(ctx, domain.NewUserDeletedEvent(user.ID))
	}

	return nil
}

// RestoreUser restores a deleted user.
func (uc *authUseCase) RestoreUser(ctx context.Context, req *dto.RestoreUserRequest) (*dto.UserResponse, error) {
	if err := uc.userRepo.Restore(ctx, req.ID); err != nil {
		return nil, err
	}

	// Get restored user
	user, err := uc.userRepo.FindByID(ctx, req.ID, domain.DefaultParanoidOptions())
	if err != nil {
		return nil, err
	}

	// Publish event
	if uc.eventBus != nil {
		uc.publishEvent(ctx, domain.NewUserRestoredEvent(user))
	}

	return dto.FromUser(user), nil
}

// publishEvent publishes an event to the event bus.
func (uc *authUseCase) publishEvent(ctx context.Context, event *domain.UserEvent) {
	if uc.eventBus == nil {
		return
	}

	// Create event bus event
	ebEvent := eventbus.NewEvent(event.EventType, uc.config.ServiceName, event.ToMap())
	utils.ApplyRequestMetadataToEvent(ctx, ebEvent)

	// Publish asynchronously
	go func() {
		eventID, err := uc.eventBus.Publish(context.Background(), eventbus.StreamAuthEvents, ebEvent)
		if err != nil {
			logger.Error("Failed to publish auth event",
				zap.String("stream", eventbus.StreamAuthEvents),
				zap.String("event_type", event.EventType),
				zap.String("user_id", event.UserID),
				zap.Error(err),
			)
			return
		}

		logger.Info("Published auth event",
			zap.String("stream", eventbus.StreamAuthEvents),
			zap.String("event_id", eventID),
			zap.String("event_type", event.EventType),
			zap.String("user_id", event.UserID),
		)
	}()
}

// ValidateToken validates a JWT token and returns the claims.
func (uc *authUseCase) ValidateToken(token string) (*utils.Claims, error) {
	return uc.jwtManager.ValidateToken(token)
}

// GenerateUUID generates a new UUID.
func GenerateUUID() string {
	return uuid.New().String()
}

// detectDeviceType determines the device type from user agent.
func detectDeviceType(userAgent string) string {
	if userAgent == "" {
		return "unknown"
	}
	lowerUA := strings.ToLower(userAgent)
	if strings.Contains(lowerUA, "mobile") || strings.Contains(lowerUA, "android") || strings.Contains(lowerUA, "iphone") {
		return "mobile"
	}
	if strings.Contains(lowerUA, "tablet") || strings.Contains(lowerUA, "ipad") {
		return "tablet"
	}
	if strings.Contains(lowerUA, "bot") || strings.Contains(lowerUA, "spider") || strings.Contains(lowerUA, "crawler") {
		return "bot"
	}
	return "desktop"
}
