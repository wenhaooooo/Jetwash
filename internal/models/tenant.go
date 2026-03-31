package models

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// Tenant 租户表实体
type Tenant struct {
	ID        uuid.UUID      `gorm:"type:uuid;primaryKey;default:gen_random_uuid()" json:"id"`
	ParentID  *uuid.UUID     `gorm:"type:uuid;index" json:"parent_id"` // 父租户ID，用于多租户层级结构
	APIKey    string         `gorm:"type:varchar(64);uniqueIndex;not null" json:"api_key"`
	Name      string         `gorm:"type:varchar(255);uniqueIndex;not null" json:"name"`
	Email     string         `gorm:"type:varchar(255);uniqueIndex;not null" json:"email"`
	Password  string         `gorm:"type:varchar(255);not null" json:"-"`         // 密码哈希，不序列化
	Status    int            `gorm:"type:smallint;default:1;index" json:"status"` // 1: active, 2: inactive, 3: suspended
	CreatedAt time.Time      `json:"created_at"`
	UpdatedAt time.Time      `json:"updated_at"`
	DeletedAt gorm.DeletedAt `gorm:"index" json:"-"`

	// 关联关系：一对多
	SensitiveWords []SensitiveWord `gorm:"foreignKey:TenantID;constraint:OnUpdate:CASCADE,OnDelete:CASCADE;" json:"-"`
}

// TableName 指定表名
func (Tenant) TableName() string {
	return "tenants"
}
