package repository

import (
	"fmt"
	"jetwash/internal/models"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type DetectionHistoryRepository interface {
	Create(history *models.DetectionHistory) error
	GetByID(id int64) (*models.DetectionHistory, error)
	GetByTenantID(tenantID uuid.UUID, filters map[string]interface{}, offset, limit int) ([]models.DetectionHistory, int64, error)
	Delete(id int64) error
	ClearByTenantID(tenantID uuid.UUID) error
}

type detectionHistoryRepository struct {
	db *gorm.DB
}

func NewDetectionHistoryRepository(db *gorm.DB) DetectionHistoryRepository {
	return &detectionHistoryRepository{db: db}
}

func (r *detectionHistoryRepository) Create(history *models.DetectionHistory) error {
	if err := r.db.Create(history).Error; err != nil {
		return fmt.Errorf("failed to create detection history: %w", err)
	}
	return nil
}

func (r *detectionHistoryRepository) GetByID(id int64) (*models.DetectionHistory, error) {
	var history models.DetectionHistory
	if err := r.db.Preload("Matches").Where("id = ? AND deleted_at IS NULL", id).First(&history).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, fmt.Errorf("detection history not found")
		}
		return nil, fmt.Errorf("failed to get detection history: %w", err)
	}
	return &history, nil
}

func (r *detectionHistoryRepository) GetByTenantID(tenantID uuid.UUID, filters map[string]interface{}, offset, limit int) ([]models.DetectionHistory, int64, error) {
	var histories []models.DetectionHistory
	var total int64

	query := r.db.Model(&models.DetectionHistory{}).Where("tenant_id = ? AND deleted_at IS NULL", tenantID)

	if filters != nil {
		if mode, ok := filters["mode"]; ok && mode != "" {
			query = query.Where("mode = ?", mode)
		}
		if startTime, ok := filters["start_time"]; ok && startTime != "" {
			query = query.Where("created_at >= ?", startTime)
		}
		if endTime, ok := filters["end_time"]; ok && endTime != "" {
			query = query.Where("created_at <= ?", endTime)
		}
	}

	if err := query.Count(&total).Error; err != nil {
		return nil, 0, fmt.Errorf("failed to count detection histories: %w", err)
	}

	if err := query.Preload("Matches").Offset(offset).Limit(limit).Order("created_at DESC").Find(&histories).Error; err != nil {
		return nil, 0, fmt.Errorf("failed to get detection histories: %w", err)
	}

	return histories, total, nil
}

func (r *detectionHistoryRepository) Delete(id int64) error {
	if err := r.db.Delete(&models.DetectionHistory{}, id).Error; err != nil {
		return fmt.Errorf("failed to delete detection history: %w", err)
	}
	return nil
}

func (r *detectionHistoryRepository) ClearByTenantID(tenantID uuid.UUID) error {
	if err := r.db.Where("tenant_id = ?", tenantID).Delete(&models.DetectionHistory{}).Error; err != nil {
		return fmt.Errorf("failed to clear detection histories: %w", err)
	}
	return nil
}
