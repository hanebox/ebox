package models

import "time"

type Project struct {
	ID          uint64    `gorm:"primaryKey" json:"id"`
	UUID        string    `gorm:"size:36;uniqueIndex" json:"uuid"`
	Name        string    `gorm:"size:255" json:"name"`
	Description string    `gorm:"type:text" json:"description"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}
