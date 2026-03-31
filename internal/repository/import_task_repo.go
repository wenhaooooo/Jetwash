package repository

import (
	"fmt"

	"jetwash/internal/models"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// ImportTaskRepository 导入任务仓储接口
type ImportTaskRepository interface {
	// Create 创建导入任务
	Create(task *models.ImportTask) error
	// Update 更新导入任务
	Update(task *models.ImportTask) error
	// GetByID 根据ID获取导入任务
	GetByID(id uuid.UUID) (*models.ImportTask, error)
	// GetByTenantID 获取租户的所有导入任务
	GetByTenantID(tenantID uuid.UUID, offset, limit int) ([]models.ImportTask, int64, error)
}

// importTaskRepository 导入任务仓储实现
type importTaskRepository struct {
	db *gorm.DB
}

// NewImportTaskRepository 创建导入任务仓储实例
func NewImportTaskRepository(db *gorm.DB) ImportTaskRepository {
	return &importTaskRepository{db: db}
}

// Create 创建导入任务
func (r *importTaskRepository) Create(task *models.ImportTask) error {
	if err := r.db.Create(task).Error; err != nil {
		return fmt.Errorf("failed to create import task: %w", err)
	}
	return nil
}

// Update 更新导入任务
func (r *importTaskRepository) Update(task *models.ImportTask) error {
	if err := r.db.Save(task).Error; err != nil {
		return fmt.Errorf("failed to update import task: %w", err)
	}
	return nil
}

// GetByID 根据ID获取导入任务
func (r *importTaskRepository) GetByID(id uuid.UUID) (*models.ImportTask, error) {
	var task models.ImportTask
	if err := r.db.Where("id = ? AND deleted_at IS NULL", id).First(&task).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, fmt.Errorf("import task not found")
		}
		return nil, fmt.Errorf("failed to get import task: %w", err)
	}
	return &task, nil
}

// GetByTenantID 获取租户的所有导入任务
func (r *importTaskRepository) GetByTenantID(tenantID uuid.UUID, offset, limit int) ([]models.ImportTask, int64, error) {
	var tasks []models.ImportTask
	var total int64

	// 获取总数
	if err := r.db.Model(&models.ImportTask{}).
		Where("tenant_id = ? AND deleted_at IS NULL", tenantID).
		Count(&total).Error; err != nil {
		return nil, 0, fmt.Errorf("failed to count import tasks: %w", err)
	}

	// 获取分页数据
	if err := r.db.Where("tenant_id = ? AND deleted_at IS NULL", tenantID).
		Offset(offset).
		Limit(limit).
		Order("created_at DESC").
		Find(&tasks).Error; err != nil {
		return nil, 0, fmt.Errorf("failed to get import tasks: %w", err)
	}

	return tasks, total, nil
}