package models

import (
	"time"

	"github.com/google/uuid"
	"github.com/pgvector/pgvector-go"
	"gorm.io/gorm"
)

// SensitiveWord 敏感词表实体
type SensitiveWord struct {
	// 注意：ID 从 UUID 改为了 uint，并设置了 autoIncrement
	ID        uint      `gorm:"primaryKey;autoIncrement" json:"id"`
	TenantID  uuid.UUID `gorm:"type:uuid;not null;index:idx_words_tenant_id" json:"tenant_id"`
	WordText  string    `gorm:"type:text;not null" json:"word_text"`
	Category  string    `gorm:"type:varchar(50);not null;index:idx_words_category" json:"category"`
	RiskLevel int       `gorm:"type:int;not null;default:1" json:"risk_level"` // 1-5, 5为最高风险
	// 注意：维度已调整为 768 以匹配 nomic-embed-text:v1.5
	Embedding pgvector.Vector `gorm:"type:vector(768);not null" json:"-"`
	Status    int             `gorm:"type:smallint;default:1;index:idx_words_status" json:"status"` // 1: 启用, 2: 停用, 3: 归档
	CreatedAt time.Time       `json:"created_at"`
	UpdatedAt time.Time       `json:"updated_at"`
	DeletedAt gorm.DeletedAt  `gorm:"index" json:"-"`

	// 关联关系：属于某个租户
	Tenant Tenant `gorm:"foreignKey:TenantID" json:"-"`
}

// TableName 指定表名
func (SensitiveWord) TableName() string {
	return "sensitive_words"
}
