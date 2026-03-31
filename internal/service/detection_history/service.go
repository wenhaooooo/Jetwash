package detection_history

import (
	"encoding/json"
	"fmt"

	"github.com/google/uuid"
	"gorm.io/gorm"

	"jetwash/internal/models"
	"jetwash/internal/types"
)

type DetectionHistoryService interface {
	SaveDetectionHistory(tenantID uuid.UUID, text string, mode string, result *types.OrchestratorResult, duration int64) error
	GetDetectionHistories(tenantID uuid.UUID, offset, limit int) ([]models.DetectionHistory, int64, error)
	GetDetectionHistoryByID(id int64, tenantID uuid.UUID) (*models.DetectionHistory, error)
	GetByTenantID(tenantID uuid.UUID, filters map[string]interface{}, offset, limit int) ([]models.DetectionHistory, int64, error)
	GetByID(id int64) (*models.DetectionHistory, error)
	Delete(id int64) error
	ClearByTenantID(tenantID uuid.UUID) error
}

type detectionHistoryService struct {
	db *gorm.DB
}

func NewDetectionHistoryService(db *gorm.DB) DetectionHistoryService {
	return &detectionHistoryService{
		db: db,
	}
}

func (s *detectionHistoryService) SaveDetectionHistory(tenantID uuid.UUID, text string, mode string, result *types.OrchestratorResult, duration int64) error {
	resultJSON, err := json.Marshal(result)
	if err != nil {
		return fmt.Errorf("failed to marshal result: %w", err)
	}

	history := &models.DetectionHistory{
		TenantID:    tenantID,
		Text:        text,
		Mode:        mode,
		IsOffensive: !result.Passed,
		ResultJSON:  string(resultJSON),
		Duration:    duration,
	}

	tx := s.db.Begin()
	defer func() {
		if r := recover(); r != nil {
			tx.Rollback()
		}
	}()

	if err := tx.Create(history).Error; err != nil {
		tx.Rollback()
		return fmt.Errorf("failed to create detection history: %w", err)
	}

	matches := s.buildMatches(history.ID, result)
	if len(matches) > 0 {
		if err := tx.Create(&matches).Error; err != nil {
			tx.Rollback()
			return fmt.Errorf("failed to create detection matches: %w", err)
		}
	}

	if err := tx.Commit().Error; err != nil {
		return fmt.Errorf("failed to commit transaction: %w", err)
	}

	return nil
}

func (s *detectionHistoryService) buildMatches(historyID int64, result *types.OrchestratorResult) []models.DetectionMatch {
	matches := make([]models.DetectionMatch, 0)

	if result.Layer1Result != nil && result.Layer1Result.HasMatch {
		for _, match := range result.Layer1Result.MatchedWords {
			matches = append(matches, models.DetectionMatch{
				HistoryID:  historyID,
				Type:       "layer1",
				Text:       match.Matched,
				Confidence: 1.0,
			})
		}
	}

	if result.Layer2Result != nil && result.Layer2Result.HasMatch {
		for _, match := range result.Layer2Result.MatchedWords {
			confidence := 1.0 - match.Distance
			if confidence < 0 {
				confidence = 0
			}
			matches = append(matches, models.DetectionMatch{
				HistoryID:  historyID,
				Type:       "layer2",
				Text:       match.WordText,
				Confidence: confidence,
			})
		}
	}

	if result.Layer3Result != nil && result.Layer3Result.HasRisk {
		matches = append(matches, models.DetectionMatch{
			HistoryID:  historyID,
			Type:       "layer3",
			Text:       result.Layer3Result.RiskReason,
			Confidence: float64(result.Layer3Result.RiskLevel) / 3.0,
		})
	}

	return matches
}

func (s *detectionHistoryService) GetDetectionHistories(tenantID uuid.UUID, offset, limit int) ([]models.DetectionHistory, int64, error) {
	var histories []models.DetectionHistory
	var total int64

	if err := s.db.Model(&models.DetectionHistory{}).
		Where("tenant_id = ? AND deleted_at IS NULL", tenantID).
		Count(&total).Error; err != nil {
		return nil, 0, fmt.Errorf("failed to count detection histories: %w", err)
	}

	if err := s.db.Where("tenant_id = ? AND deleted_at IS NULL", tenantID).
		Order("created_at DESC").
		Offset(offset).
		Limit(limit).
		Find(&histories).Error; err != nil {
		return nil, 0, fmt.Errorf("failed to get detection histories: %w", err)
	}

	return histories, total, nil
}

func (s *detectionHistoryService) GetDetectionHistoryByID(id int64, tenantID uuid.UUID) (*models.DetectionHistory, error) {
	var history models.DetectionHistory

	if err := s.db.Where("id = ? AND tenant_id = ? AND deleted_at IS NULL", id, tenantID).
		Preload("Matches").
		First(&history).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, fmt.Errorf("detection history not found")
		}
		return nil, fmt.Errorf("failed to get detection history: %w", err)
	}

	return &history, nil
}

func (s *detectionHistoryService) GetByTenantID(tenantID uuid.UUID, filters map[string]interface{}, offset, limit int) ([]models.DetectionHistory, int64, error) {
	var histories []models.DetectionHistory
	var total int64

	query := s.db.Model(&models.DetectionHistory{}).Where("tenant_id = ? AND deleted_at IS NULL", tenantID)

	if startTime, ok := filters["start_time"].(string); ok && startTime != "" {
		query = query.Where("created_at >= ?", startTime)
	}
	if endTime, ok := filters["end_time"].(string); ok && endTime != "" {
		query = query.Where("created_at <= ?", endTime)
	}
	if mode, ok := filters["mode"].(string); ok && mode != "" {
		query = query.Where("mode = ?", mode)
	}
	if isOffensive, ok := filters["is_offensive"].(bool); ok {
		query = query.Where("is_offensive = ?", isOffensive)
	}

	if err := query.Count(&total).Error; err != nil {
		return nil, 0, fmt.Errorf("failed to count detection histories: %w", err)
	}

	if err := query.Order("created_at DESC").Offset(offset).Limit(limit).Find(&histories).Error; err != nil {
		return nil, 0, fmt.Errorf("failed to get detection histories: %w", err)
	}

	return histories, total, nil
}

func (s *detectionHistoryService) GetByID(id int64) (*models.DetectionHistory, error) {
	var history models.DetectionHistory
	if err := s.db.Where("id = ? AND deleted_at IS NULL", id).First(&history).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, fmt.Errorf("detection history not found")
		}
		return nil, fmt.Errorf("failed to get detection history: %w", err)
	}
	return &history, nil
}

func (s *detectionHistoryService) Delete(id int64) error {
	if err := s.db.Delete(&models.DetectionHistory{}, id).Error; err != nil {
		return fmt.Errorf("failed to delete detection history: %w", err)
	}
	return nil
}

func (s *detectionHistoryService) ClearByTenantID(tenantID uuid.UUID) error {
	if err := s.db.Where("tenant_id = ?", tenantID).Delete(&models.DetectionHistory{}).Error; err != nil {
		return fmt.Errorf("failed to clear detection histories: %w", err)
	}
	return nil
}
