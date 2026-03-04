// Package repository provides activity log data access.
package repository

import (
	"context"
	"fmt"

	"github.com/ignata/go-microservices-boilerplate/internal/user/domain"
	"github.com/ignata/go-microservices-boilerplate/internal/user/dto"
	"gorm.io/gorm"
)

// ActivityRepository defines the interface for activity log data access.
type ActivityRepository interface {
	// Create creates a new activity log
	Create(ctx context.Context, log *domain.ActivityLog) error
	// FindByUserID finds activity logs by user ID
	FindByUserID(ctx context.Context, req *dto.ListActivityLogsRequest) (*domain.ActivityLogList, error)
	// FindAll finds all activity logs with pagination
	FindAll(ctx context.Context, req *dto.ListActivityLogsRequest) (*domain.ActivityLogList, error)
	// DeleteOlderThan deletes activity logs older than the specified duration
	DeleteOlderThan(ctx context.Context, days int) error
}

// gormActivityRepository implements ActivityRepository using GORM.
type gormActivityRepository struct {
	db *gorm.DB
}

// NewActivityRepository creates a new activity repository.
func NewActivityRepository(db *gorm.DB) ActivityRepository {
	return &gormActivityRepository{db: db}
}

// Create creates a new activity log.
func (r *gormActivityRepository) Create(ctx context.Context, log *domain.ActivityLog) error {
	result := r.db.WithContext(ctx).Create(log)
	if result.Error != nil {
		return fmt.Errorf("failed to create activity log: %w", result.Error)
	}
	return nil
}

// FindByUserID finds activity logs by user ID.
func (r *gormActivityRepository) FindByUserID(ctx context.Context, req *dto.ListActivityLogsRequest) (*domain.ActivityLogList, error) {
	if req == nil {
		req = &dto.ListActivityLogsRequest{}
	}

	page := req.GetPage()
	limit := req.GetLimit()

	query := r.db.WithContext(ctx).Model(&domain.ActivityLog{}).Where("user_id = ?", req.UserID)

	// Apply filters
	if req.Action != "" {
		query = query.Where("action = ?", req.Action)
	}
	if req.Resource != "" {
		query = query.Where("resource = ?", req.Resource)
	}

	// Count total
	var total int64
	if err := query.Count(&total).Error; err != nil {
		return nil, fmt.Errorf("failed to count activity logs: %w", err)
	}

	// Calculate pagination
	totalPages := int(total) / limit
	if int(total)%limit > 0 {
		totalPages++
	}

	// Get paginated results
	var logs []*domain.ActivityLog
	offset := (page - 1) * limit
	result := query.Order("created_at DESC").Offset(offset).Limit(limit).Find(&logs)
	if result.Error != nil {
		return nil, fmt.Errorf("failed to find activity logs: %w", result.Error)
	}

	return &domain.ActivityLogList{
		Logs:       logs,
		Total:      total,
		Page:       page,
		Limit:      limit,
		TotalPages: totalPages,
	}, nil
}

// FindAll finds all activity logs with pagination.
func (r *gormActivityRepository) FindAll(ctx context.Context, req *dto.ListActivityLogsRequest) (*domain.ActivityLogList, error) {
	if req == nil {
		req = &dto.ListActivityLogsRequest{}
	}

	page := req.GetPage()
	limit := req.GetLimit()

	query := r.db.WithContext(ctx).Model(&domain.ActivityLog{})

	// Apply filters
	if req.UserID != "" {
		query = query.Where("user_id = ?", req.UserID)
	}
	if req.Action != "" {
		query = query.Where("action = ?", req.Action)
	}
	if req.Resource != "" {
		query = query.Where("resource = ?", req.Resource)
	}

	// Count total
	var total int64
	if err := query.Count(&total).Error; err != nil {
		return nil, fmt.Errorf("failed to count activity logs: %w", err)
	}

	// Calculate pagination
	totalPages := int(total) / limit
	if int(total)%limit > 0 {
		totalPages++
	}

	// Get paginated results
	var logs []*domain.ActivityLog
	offset := (page - 1) * limit
	result := query.Order("created_at DESC").Offset(offset).Limit(limit).Find(&logs)
	if result.Error != nil {
		return nil, fmt.Errorf("failed to find activity logs: %w", result.Error)
	}

	return &domain.ActivityLogList{
		Logs:       logs,
		Total:      total,
		Page:       page,
		Limit:      limit,
		TotalPages: totalPages,
	}, nil
}

// DeleteOlderThan deletes activity logs older than the specified number of days.
func (r *gormActivityRepository) DeleteOlderThan(ctx context.Context, days int) error {
	result := r.db.WithContext(ctx).
		Where("created_at < NOW() - INTERVAL '? days'", days).
		Delete(&domain.ActivityLog{})

	if result.Error != nil {
		return fmt.Errorf("failed to delete old activity logs: %w", result.Error)
	}

	return nil
}
