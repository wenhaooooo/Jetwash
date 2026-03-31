package repository

import (
	"fmt"

	"jetwash/internal/models"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// TenantRepository 租户仓储接口
type TenantRepository interface {
	// GetTenantByID 根据 ID 获取租户
	GetTenantByID(id uuid.UUID) (*models.Tenant, error)

	// GetTenantByAPIKey 根据 API Key 获取租户
	GetTenantByAPIKey(apiKey string) (*models.Tenant, error)

	// GetTenantByName 根据名称获取租户
	GetTenantByName(name string) (*models.Tenant, error)

	// GetTenantByEmail 根据邮箱获取租户
	GetTenantByEmail(email string) (*models.Tenant, error)

	// CreateTenant 创建租户
	CreateTenant(tenant *models.Tenant) error

	// UpdateTenant 更新租户
	UpdateTenant(tenant *models.Tenant) error

	// DeleteTenant 删除租户（软删除）
	DeleteTenant(id uuid.UUID) error

	// ListTenants 列出租户
	ListTenants(offset, limit int) ([]models.Tenant, int64, error)

	// GetTenantLevel 获取租户层级深度（顶级租户为0）
	GetTenantLevel(tenantID uuid.UUID) (int, error)
}

// tenantRepository 租户仓储实现
type tenantRepository struct {
	db *gorm.DB
}

// NewTenantRepository 创建租户仓储实例
func NewTenantRepository(db *gorm.DB) TenantRepository {
	return &tenantRepository{db: db}
}

// GetTenantByID 根据 ID 获取租户
func (r *tenantRepository) GetTenantByID(id uuid.UUID) (*models.Tenant, error) {
	var tenant models.Tenant
	if err := r.db.Where("id = ? AND deleted_at IS NULL", id).First(&tenant).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, fmt.Errorf("tenant not found")
		}
		return nil, fmt.Errorf("failed to get tenant: %w", err)
	}
	return &tenant, nil
}

// GetTenantByAPIKey 根据 API Key 获取租户
func (r *tenantRepository) GetTenantByAPIKey(apiKey string) (*models.Tenant, error) {
	var tenant models.Tenant
	if err := r.db.Where("api_key = ? AND deleted_at IS NULL", apiKey).First(&tenant).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, fmt.Errorf("tenant not found")
		}
		return nil, fmt.Errorf("failed to get tenant: %w", err)
	}
	return &tenant, nil
}

// GetTenantByName 根据名称获取租户
func (r *tenantRepository) GetTenantByName(name string) (*models.Tenant, error) {
	var tenant models.Tenant
	if err := r.db.Where("name = ? AND deleted_at IS NULL", name).First(&tenant).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, fmt.Errorf("tenant not found")
		}
		return nil, fmt.Errorf("failed to get tenant: %w", err)
	}
	return &tenant, nil
}

// GetTenantByEmail 根据邮箱获取租户
func (r *tenantRepository) GetTenantByEmail(email string) (*models.Tenant, error) {
	var tenant models.Tenant
	if err := r.db.Where("email = ? AND deleted_at IS NULL", email).First(&tenant).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, fmt.Errorf("tenant not found")
		}
		return nil, fmt.Errorf("failed to get tenant: %w", err)
	}
	return &tenant, nil
}

// CreateTenant 创建租户
func (r *tenantRepository) CreateTenant(tenant *models.Tenant) error {
	if err := r.db.Create(tenant).Error; err != nil {
		return fmt.Errorf("failed to create tenant: %w", err)
	}
	return nil
}

// UpdateTenant 更新租户
func (r *tenantRepository) UpdateTenant(tenant *models.Tenant) error {
	if err := r.db.Save(tenant).Error; err != nil {
		return fmt.Errorf("failed to update tenant: %w", err)
	}
	return nil
}

// DeleteTenant 删除租户（软删除）
func (r *tenantRepository) DeleteTenant(id uuid.UUID) error {
	if err := r.db.Delete(&models.Tenant{}, id).Error; err != nil {
		return fmt.Errorf("failed to delete tenant: %w", err)
	}
	return nil
}

// ListTenants 列出租户
func (r *tenantRepository) ListTenants(offset, limit int) ([]models.Tenant, int64, error) {
	var tenants []models.Tenant
	var total int64

	// 获取总数
	if err := r.db.Model(&models.Tenant{}).
		Where("deleted_at IS NULL").
		Count(&total).Error; err != nil {
		return nil, 0, fmt.Errorf("failed to count tenants: %w", err)
	}

	// 获取分页数据
	if err := r.db.Where("deleted_at IS NULL").
		Offset(offset).
		Limit(limit).
		Order("created_at DESC").
		Find(&tenants).Error; err != nil {
		return nil, 0, fmt.Errorf("failed to list tenants: %w", err)
	}

	return tenants, total, nil
}

// GetTenantLevel 获取租户层级深度（顶级租户为1）
func (r *tenantRepository) GetTenantLevel(tenantID uuid.UUID) (int, error) {
	level := 1
	currentID := tenantID

	for {
		var tenant models.Tenant
		if err := r.db.Where("id = ? AND deleted_at IS NULL", currentID).First(&tenant).Error; err != nil {
			if err == gorm.ErrRecordNotFound {
				return 0, fmt.Errorf("tenant not found")
			}
			return 0, fmt.Errorf("failed to get tenant: %w", err)
		}

		if tenant.ParentID == nil {
			break
		}

		level++
		currentID = *tenant.ParentID

		if level > 5 {
			return level, nil
		}
	}

	return level, nil
}
