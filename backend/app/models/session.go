package models

import "time"

type Session struct {
	ID         uint64    `gorm:"primaryKey" json:"id"`
	UserID     uint64    `gorm:"index" json:"user_id"`
	TokenHash  string    `gorm:"size:64;uniqueIndex" json:"-"`
	CSRFHash   string    `gorm:"size:64" json:"-"`
	IP         string    `gorm:"size:64" json:"ip"`
	UserAgent  string    `gorm:"size:512" json:"user_agent"`
	LastSeenAt time.Time `json:"last_seen_at"`
	ExpiresAt  time.Time `gorm:"index" json:"expires_at"`
	CreatedAt  time.Time `json:"created_at"`
	UpdatedAt  time.Time `json:"updated_at"`
}
