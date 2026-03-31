package models

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// APIKey API密钥表实体
type APIKey struct {
	ID        uuid.UUID      `gorm:"type:uuid;primaryKey;default:gen_random_uuid()" json:"id"`
	TenantID  uuid.UUID      `gorm:"type:uuid;not null;index:idx_api_key_tenant_id" json:"tenant_id"`
	APIKey    string         `gorm:"type:varchar(64);uniqueIndex;not null" json:"api_key"`
	Name      string         `gorm:"type:varchar(255);not null" json:"name"`
	Status    int            `gorm:"type:smallint;default:1;index" json:"status"` // 1: active, 2: inactive
	ExpiresAt time.Time      `gorm:"index" json:"expires_at"`
	CreatedAt time.Time      `json:"created_at"`
	UpdatedAt time.Time      `json:"updated_at"`
	DeletedAt gorm.DeletedAt `gorm:"index" json:"-"`

	// 关联关系：多对一
	Tenant Tenant `gorm:"foreignKey:TenantID;constraint:OnUpdate:CASCADE,OnDelete:CASCADE;" json:"-"`
}

// TableName 指定表名
func (APIKey) TableName() string {
	return "api_keys"
}
