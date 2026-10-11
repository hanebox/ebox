package models

import "time"

const (
	ResourceTypeApplication = "application"
	ResourceTypeDatabase    = "database"
	ResourceTypeService     = "service"
)

type EnvVar struct {
	ID           uint64 `gorm:"primaryKey" json:"id"`
	ResourceType string `gorm:"size:32;index" json:"resource_type"`
	ResourceID   uint64 `gorm:"index" json:"resource_id"`
	Key          string `gorm:"size:255" json:"key"`
	Value        string `gorm:"type:text" json:"value"`
	IsBuildTime  bool   `json:"is_build_time"`

	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

type Domain struct {
	ID           uint64 `gorm:"primaryKey" json:"id"`
	ResourceType string `gorm:"size:32;index" json:"resource_type"`
	ResourceID   uint64 `gorm:"index" json:"resource_id"`
	Host         string `gorm:"size:255;uniqueIndex" json:"host"`
	Port         int    `json:"port"`
	HTTPS        bool   `gorm:"column:https" json:"https"`
	IsPrimary    bool   `json:"is_primary"`

	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}
