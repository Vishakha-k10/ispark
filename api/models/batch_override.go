package models

import (
	"time"

	"gorm.io/gorm"
)

// BatchOverride allows admins to customize or override status and notes for a specific batch.
type BatchOverride struct {
	ID        uint           `gorm:"primaryKey" json:"id"`
	BatchName string         `gorm:"type:varchar(100);uniqueIndex;not null" json:"batch_name"`
	Status    string         `gorm:"type:varchar(50)" json:"status"` // Excellent, Good, At Risk
	Notes     string         `gorm:"type:text" json:"notes"`
	CreatedAt time.Time      `json:"created_at"`
	UpdatedAt time.Time      `json:"updated_at"`
	DeletedAt gorm.DeletedAt `gorm:"index" json:"-"`
}
