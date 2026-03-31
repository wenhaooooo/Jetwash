package repository

import (
	"fmt"

	"jetwash/internal/models"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// APIKeyRepository API密钥仓储接口
type APIKeyRepository interface {
	// GetAPIKeyByID 根据 ID 获取API密钥
	GetAPIKeyByID(id uuid.UUID) (*models.APIKey, error)

	// GetAPIKeyByAPIKey 根据 API Key 获取API密钥
	GetAPIKeyByAPIKey(apiKey string) (*models.APIKey, error)

	// GetAPIKeysByTenantID 根据租户 ID 获取API密钥列表
	GetAPIKeysByTenantID(tenantID uuid.UUID) ([]models.APIKey, error)

	// CreateAPIKey 创建API密钥
	CreateAPIKey(apiKey *models.APIKey) error

	// UpdateAPIKey 更新API密钥
	UpdateAPIKey(apiKey *models.APIKey) error

	// DeleteAPIKey 删除API密钥（软删除）
	DeleteAPIKey(id uuid.UUID) error

	// ListAPIKeys 列API密钥
	ListAPIKeys(tenantID uuid.UUID, offset, limit int) ([]models.APIKey, int64, error)

	// ActivateAPIKey 激活API密钥
	ActivateAPIKey(id uuid.UUID) error

	// DeactivateAPIKey 停用API密钥
	DeactivateAPIKey(id uuid.UUID) error
}

// apiKeyRepository API密钥仓储实现
type apiKeyRepository struct {
	db *gorm.DB
}

// NewAPIKeyRepository 创建API密钥仓储实例
func NewAPIKeyRepository(db *gorm.DB) APIKeyRepository {
	return &apiKeyRepository{db: db}
}

// GetAPIKeyByID 根据 ID 获取API密钥
func (r *apiKeyRepository) GetAPIKeyByID(id uuid.UUID) (*models.APIKey, error) {
	var apiKey models.APIKey
	if err := r.db.Where("id = ? AND deleted_at IS NULL", id).First(&apiKey).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, fmt.Errorf("api key not found")
		}
		return nil, fmt.Errorf("failed to get api key: %w", err)
	}
	return &apiKey, nil
}

// GetAPIKeyByAPIKey 根据 API Key 获取API密钥
func (r *apiKeyRepository) GetAPIKeyByAPIKey(apiKey string) (*models.APIKey, error) {
	var apiKeyModel models.APIKey
	if err := r.db.Where("api_key = ? AND deleted_at IS NULL", apiKey).First(&apiKeyModel).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, fmt.Errorf("api key not found")
		}
		return nil, fmt.Errorf("failed to get api key: %w", err)
	}
	return &apiKeyModel, nil
}

// GetAPIKeysByTenantID 根据租户 ID 获取API密钥列表
func (r *apiKeyRepository) GetAPIKeysByTenantID(tenantID uuid.UUID) ([]models.APIKey, error) {
	var apiKeys []models.APIKey
	if err := r.db.Where("tenant_id = ? AND deleted_at IS NULL", tenantID).Find(&apiKeys).Error; err != nil {
		return nil, fmt.Errorf("failed to get api keys: %w", err)
	}
	return apiKeys, nil
}

// CreateAPIKey 创建API密钥
func (r *apiKeyRepository) CreateAPIKey(apiKey *models.APIKey) error {
	if err := r.db.Create(apiKey).Error; err != nil {
		return fmt.Errorf("failed to create api key: %w", err)
	}
	return nil
}

// UpdateAPIKey 更新API密钥
func (r *apiKeyRepository) UpdateAPIKey(apiKey *models.APIKey) error {
	if err := r.db.Save(apiKey).Error; err != nil {
		return fmt.Errorf("failed to update api key: %w", err)
	}
	return nil
}

// DeleteAPIKey 删除API密钥（软删除）
func (r *apiKeyRepository) DeleteAPIKey(id uuid.UUID) error {
	if err := r.db.Delete(&models.APIKey{}, id).Error; err != nil {
		return fmt.Errorf("failed to delete api key: %w", err)
	}
	return nil
}

// ListAPIKeys 列API密钥
func (r *apiKeyRepository) ListAPIKeys(tenantID uuid.UUID, offset, limit int) ([]models.APIKey, int64, error) {
	var apiKeys []models.APIKey
	var total int64

	// 获取总数
	if err := r.db.Model(&models.APIKey{}).
		Where("tenant_id = ? AND deleted_at IS NULL", tenantID).
		Count(&total).Error; err != nil {
		return nil, 0, fmt.Errorf("failed to count api keys: %w", err)
	}

	// 获取分页数据
	if err := r.db.Where("tenant_id = ? AND deleted_at IS NULL", tenantID).
		Offset(offset).
		Limit(limit).
		Order("created_at DESC").
		Find(&apiKeys).Error; err != nil {
		return nil, 0, fmt.Errorf("failed to list api keys: %w", err)
	}

	return apiKeys, total, nil
}

// ActivateAPIKey 激活API密钥
func (r *apiKeyRepository) ActivateAPIKey(id uuid.UUID) error {
	if err := r.db.Model(&models.APIKey{}).Where("id = ?", id).Update("status", 1).Error; err != nil {
		return fmt.Errorf("failed to activate api key: %w", err)
	}
	return nil
}

// DeactivateAPIKey 停用API密钥
func (r *apiKeyRepository) DeactivateAPIKey(id uuid.UUID) error {
	if err := r.db.Model(&models.APIKey{}).Where("id = ?", id).Update("status", 2).Error; err != nil {
		return fmt.Errorf("failed to deactivate api key: %w", err)
	}
	return nil
}
