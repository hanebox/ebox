package models

import "time"

const (
	SourceTypeGithubApp = "github_app"
	SourceTypeGit       = "git"

	SourceAuthNone  = "none"
	SourceAuthToken = "token"
	SourceAuthSSH   = "ssh"
)

type Source struct {
	ID   uint64 `gorm:"primaryKey" json:"id"`
	UUID string `gorm:"size:36;uniqueIndex" json:"uuid"`
	Name string `gorm:"size:255" json:"name"`
	Type string `gorm:"size:32" json:"type"`

	AppID          int64  `json:"app_id"`
	AppSlug        string `gorm:"size:255" json:"app_slug"`
	InstallationID string `gorm:"size:32" json:"installation_id"`
	HTMLURL        string `gorm:"size:512" json:"html_url"`

	GitURL   string `gorm:"size:512" json:"git_url"`
	AuthType string `gorm:"size:16" json:"auth_type"`

	PrivateKeyEncrypted    string `gorm:"type:text" json:"-"`
	WebhookSecretEncrypted string `gorm:"type:text" json:"-"`
	TokenEncrypted         string `gorm:"type:text" json:"-"`
	SSHKeyEncrypted        string `gorm:"type:text" json:"-"`

	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}
