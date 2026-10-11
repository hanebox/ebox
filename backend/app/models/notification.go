package models

import "time"

const (
	NotificationTypeEmail    = "email"
	NotificationTypeDiscord  = "discord"
	NotificationTypeSlack    = "slack"
	NotificationTypeTelegram = "telegram"
	NotificationTypeWebhook  = "webhook"

	NotificationStatusPending = "pending"
	NotificationStatusSent    = "sent"
	NotificationStatusFailed  = "failed"
)

// Notification events a channel can subscribe to.
const (
	EventDeploymentSuccess = "deployment.success"
	EventDeploymentFailed  = "deployment.failed"
	EventBackupSuccess     = "backup.success"
	EventBackupFailed      = "backup.failed"
	EventContainerDown     = "container.down"
	EventCertificateExpiry = "certificate.expiring"
	EventDiskHigh          = "disk.high"
	EventUpdateAvailable   = "update.available"
)

// NotificationEvents lists every event the panel can announce.
func NotificationEvents() []string {
	return []string{
		EventDeploymentSuccess,
		EventDeploymentFailed,
		EventBackupSuccess,
		EventBackupFailed,
		EventContainerDown,
		EventCertificateExpiry,
		EventDiskHigh,
		EventUpdateAvailable,
	}
}

type NotificationChannel struct {
	ID      uint64 `gorm:"primaryKey" json:"id"`
	UUID    string `gorm:"size:36;uniqueIndex" json:"uuid"`
	Name    string `gorm:"size:255" json:"name"`
	Type    string `gorm:"size:32" json:"type"`
	Enabled bool   `json:"enabled"`
	// Config holds the type specific settings. Values marked as secret are
	// stored encrypted.
	Config string `gorm:"type:text" json:"-"`
	// Events lists the subscribed event names. Empty means every event.
	Events    string    `gorm:"type:text" json:"-"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

type NotificationDelivery struct {
	ID        uint64    `gorm:"primaryKey" json:"id"`
	ChannelID uint64    `gorm:"index" json:"channel_id"`
	Channel   string    `gorm:"size:255" json:"channel"`
	Event     string    `gorm:"size:64;index" json:"event"`
	Title     string    `gorm:"size:512" json:"title"`
	Body      string    `gorm:"type:text" json:"body"`
	Status    string    `gorm:"size:32;index" json:"status"`
	Error     string    `gorm:"type:text" json:"error,omitempty"`
	Attempts  int       `json:"attempts"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}
