package api_key

import (
	"fmt"
	"math/rand"
	"time"

	"jetwash/internal/models"
	"jetwash/internal/repository"

	"github.com/google/uuid"
)

// APIKeyService API密钥服务接口
type APIKeyService interface {
	// GenerateAPIKey 生成API密钥
	GenerateAPIKey() string

	// CreateAPIKey 创建API密钥
	CreateAPIKey(tenantID uuid.UUID, name string, expiresAt time.Time) (*models.APIKey, error)

	// GetAPIKeyByID 根据ID获取API密钥
	GetAPIKeyByID(id uuid.UUID) (*models.APIKey, error)

	// GetAPIKeyByAPIKey 根据API Key获取API密钥
	GetAPIKeyByAPIKey(apiKey string) (*models.APIKey, error)

	// GetAPIKeysByTenantID 根据租户ID获取API密钥列表
	GetAPIKeysByTenantID(tenantID uuid.UUID) ([]models.APIKey, error)

	// ListAPIKeys 列API密钥
	ListAPIKeys(tenantID uuid.UUID, page, pageSize int) ([]models.APIKey, int64, error)

	// UpdateAPIKey 更新API密钥
	UpdateAPIKey(apiKey *models.APIKey) error

	// DeleteAPIKey 删除API密钥
	DeleteAPIKey(id uuid.UUID) error

	// ActivateAPIKey 激活API密钥
	ActivateAPIKey(id uuid.UUID) error

	// DeactivateAPIKey 停用API密钥
	DeactivateAPIKey(id uuid.UUID) error
}

// apiKeyService API密钥服务实现
type apiKeyService struct {
	apiKeyRepo repository.APIKeyRepository
}

// NewAPIKeyService 创建API密钥服务实例
func NewAPIKeyService(apiKeyRepo repository.APIKeyRepository) APIKeyService {
	return &apiKeyService{
		apiKeyRepo: apiKeyRepo,
	}
}

// GenerateAPIKey 生成API密钥
func (s *apiKeyService) GenerateAPIKey() string {
	const charset = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
	const length = 32

	rng := rand.New(rand.NewSource(time.Now().UnixNano()))
	key := make([]byte, length)
	for i := range key {
		key[i] = charset[rng.Intn(len(charset))]
	}

	return string(key)
}

// CreateAPIKey 创建API密钥
func (s *apiKeyService) CreateAPIKey(tenantID uuid.UUID, name string, expiresAt time.Time) (*models.APIKey, error) {
	apiKey := &models.APIKey{
		TenantID:  tenantID,
		APIKey:    s.GenerateAPIKey(),
		Name:      name,
		Status:    1, // 1: active
		ExpiresAt: expiresAt,
	}

	if err := s.apiKeyRepo.CreateAPIKey(apiKey); err != nil {
		return nil, fmt.Errorf("failed to create api key: %w", err)
	}

	return apiKey, nil
}

// GetAPIKeyByID 根据ID获取API密钥
func (s *apiKeyService) GetAPIKeyByID(id uuid.UUID) (*models.APIKey, error) {
	apiKey, err := s.apiKeyRepo.GetAPIKeyByID(id)
	if err != nil {
		return nil, fmt.Errorf("failed to get api key: %w", err)
	}
	return apiKey, nil
}

// GetAPIKeyByAPIKey 根据API Key获取API密钥
func (s *apiKeyService) GetAPIKeyByAPIKey(apiKey string) (*models.APIKey, error) {
	apiKeyModel, err := s.apiKeyRepo.GetAPIKeyByAPIKey(apiKey)
	if err != nil {
		return nil, fmt.Errorf("failed to get api key: %w", err)
	}
	return apiKeyModel, nil
}

// GetAPIKeysByTenantID 根据租户ID获取API密钥列表
func (s *apiKeyService) GetAPIKeysByTenantID(tenantID uuid.UUID) ([]models.APIKey, error) {
	apiKeys, err := s.apiKeyRepo.GetAPIKeysByTenantID(tenantID)
	if err != nil {
		return nil, fmt.Errorf("failed to get api keys: %w", err)
	}
	return apiKeys, nil
}

// ListAPIKeys 列API密钥
func (s *apiKeyService) ListAPIKeys(tenantID uuid.UUID, page, pageSize int) ([]models.APIKey, int64, error) {
	if page < 1 {
		page = 1
	}
	if pageSize < 1 || pageSize > 100 {
		pageSize = 10
	}

	offset := (page - 1) * pageSize
	apiKeys, total, err := s.apiKeyRepo.ListAPIKeys(tenantID, offset, pageSize)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to list api keys: %w", err)
	}

	return apiKeys, total, nil
}

// UpdateAPIKey 更新API密钥
func (s *apiKeyService) UpdateAPIKey(apiKey *models.APIKey) error {
	if err := s.apiKeyRepo.UpdateAPIKey(apiKey); err != nil {
		return fmt.Errorf("failed to update api key: %w", err)
	}
	return nil
}

// DeleteAPIKey 删除API密钥
func (s *apiKeyService) DeleteAPIKey(id uuid.UUID) error {
	if err := s.apiKeyRepo.DeleteAPIKey(id); err != nil {
		return fmt.Errorf("failed to delete api key: %w", err)
	}
	return nil
}

// ActivateAPIKey 激活API密钥
func (s *apiKeyService) ActivateAPIKey(id uuid.UUID) error {
	if err := s.apiKeyRepo.ActivateAPIKey(id); err != nil {
		return fmt.Errorf("failed to activate api key: %w", err)
	}
	return nil
}

// DeactivateAPIKey 停用API密钥
func (s *apiKeyService) DeactivateAPIKey(id uuid.UUID) error {
	if err := s.apiKeyRepo.DeactivateAPIKey(id); err != nil {
		return fmt.Errorf("failed to deactivate api key: %w", err)
	}
	return nil
}
