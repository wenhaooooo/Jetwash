package models

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// ImportTask 批量导入任务表实体
type ImportTask struct {
	ID         uuid.UUID `gorm:"type:uuid;primaryKey;default:uuid_generate_v4()" json:"id"`
	TenantID   uuid.UUID `gorm:"type:uuid;not null;index:idx_import_tenant_id" json:"tenant_id"`
	FileName   string    `gorm:"type:varchar(255);not null" json:"file_name"`
	Status     string    `gorm:"type:varchar(20);not null;default:'pending';index:idx_import_status" json:"status"` // pending, processing, completed, failed
	Total      int       `gorm:"type:int;default:0" json:"total"`
	Imported   int       `gorm:"type:int;default:0" json:"imported"`
	Failed     int       `gorm:"type:int;default:0" json:"failed"`
	ErrorMsg   string    `gorm:"type:text" json:"error_msg"`
	StartedAt  *time.Time `json:"started_at"`
	CompletedAt *time.Time `json:"completed_at"`
	CreatedAt  time.Time `json:"created_at"`
	UpdatedAt  time.Time `json:"updated_at"`
	DeletedAt  gorm.DeletedAt `gorm:"index" json:"-"`

	// 关联关系：属于某个租户
	Tenant Tenant `gorm:"foreignKey:TenantID" json:"-"`
}

// TableName 指定表名
func (ImportTask) TableName() string {
	return "import_tasks"
}