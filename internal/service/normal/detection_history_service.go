package normal

import (
	"jetwash/internal/models"
	"jetwash/internal/repository"

	"github.com/google/uuid"
)

type DetectionHistoryService interface {
	Create(history *models.DetectionHistory) error
	GetByID(id int64) (*models.DetectionHistory, error)
	GetByTenantID(tenantID uuid.UUID, filters map[string]interface{}, offset, limit int) ([]models.DetectionHistory, int64, error)
	Delete(id int64) error
	ClearByTenantID(tenantID uuid.UUID) error
}

type detectionHistoryService struct {
	repo repository.DetectionHistoryRepository
}

func NewDetectionHistoryService(repo repository.DetectionHistoryRepository) DetectionHistoryService {
	return &detectionHistoryService{repo: repo}
}

func (s *detectionHistoryService) Create(history *models.DetectionHistory) error {
	return s.repo.Create(history)
}

func (s *detectionHistoryService) GetByID(id int64) (*models.DetectionHistory, error) {
	return s.repo.GetByID(id)
}

func (s *detectionHistoryService) GetByTenantID(tenantID uuid.UUID, filters map[string]interface{}, offset, limit int) ([]models.DetectionHistory, int64, error) {
	return s.repo.GetByTenantID(tenantID, filters, offset, limit)
}

func (s *detectionHistoryService) Delete(id int64) error {
	return s.repo.Delete(id)
}

func (s *detectionHistoryService) ClearByTenantID(tenantID uuid.UUID) error {
	return s.repo.ClearByTenantID(tenantID)
}
