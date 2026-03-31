package models

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type DetectionHistory struct {
	ID         int64             `json:"id" gorm:"primaryKey;autoIncrement"`
	TenantID   uuid.UUID         `json:"tenant_id" gorm:"type:uuid;not null;index:idx_history_tenant_id"`
	Text       string            `json:"text" gorm:"type:text;not null"`
	Mode       string            `json:"mode" gorm:"type:varchar(20);not null;index:idx_history_mode"`
	IsOffensive bool             `json:"is_offensive" gorm:"not null"`
	ResultJSON  string           `json:"result_json" gorm:"type:json"`
	Duration   int64            `json:"duration" gorm:"not null"`
	CreatedAt  time.Time         `json:"created_at" gorm:"autoCreateTime;index:idx_history_created_at"`
	UpdatedAt  time.Time         `json:"updated_at" gorm:"autoUpdateTime"`
	DeletedAt  gorm.DeletedAt   `json:"-" gorm:"index"`
	Matches    []DetectionMatch  `json:"matches" gorm:"foreignKey:HistoryID;constraint:OnDelete:CASCADE"`
}

func (DetectionHistory) TableName() string {
	return "detection_history"
}

type DetectionMatch struct {
	ID         int64   `json:"id" gorm:"primaryKey;autoIncrement"`
	HistoryID  int64   `json:"history_id" gorm:"not null;index:idx_match_history_id"`
	Type       string   `json:"type" gorm:"type:varchar(50);not null"`
	Text       string   `json:"text" gorm:"type:varchar(255);not null"`
	Confidence float64 `json:"confidence" gorm:"not null"`
	CreatedAt time.Time `json:"created_at" gorm:"autoCreateTime"`
}

func (DetectionMatch) TableName() string {
	return "detection_match"
}
