package models

import "time"

type PersonalAccessToken struct {
	ID         uint64     `gorm:"primaryKey" json:"id"`
	UserID     uint64     `gorm:"index" json:"user_id"`
	Name       string     `gorm:"size:255" json:"name"`
	TokenHash  string     `gorm:"size:64;uniqueIndex" json:"-"`
	LastUsedAt *time.Time `json:"last_used_at"`
	ExpiresAt  *time.Time `json:"expires_at"`
	CreatedAt  time.Time  `json:"created_at"`
	UpdatedAt  time.Time  `json:"updated_at"`
}
