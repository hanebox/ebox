package models

import "time"

const (
	ServiceSourceInline = "inline"
	ServiceSourceGit    = "git"

	ServiceStatusIdle      = "idle"
	ServiceStatusDeploying = "deploying"
	ServiceStatusRunning   = "running"
	ServiceStatusFailed    = "failed"
)

// Service is a compose stack managed as one unit.
type Service struct {
	ID           uint64     `gorm:"primaryKey" json:"id"`
	UUID         string     `gorm:"size:36;uniqueIndex" json:"uuid"`
	ProjectID    uint64     `gorm:"index" json:"project_id"`
	SourceID     *uint64    `json:"source_id"`
	Name         string     `gorm:"size:255" json:"name"`
	SourceType   string     `gorm:"column:source;size:16" json:"source"`
	Repository   string     `gorm:"size:512" json:"repository"`
	Branch       string     `gorm:"size:255" json:"branch"`
	ComposePath  string     `gorm:"size:512" json:"compose_path"`
	Compose      string     `gorm:"type:text" json:"compose"`
	Environment  string     `gorm:"type:text" json:"environment"`
	EntryService string     `gorm:"size:255" json:"entry_service"`
	EntryPort    int        `json:"entry_port"`
	Status       string     `gorm:"size:32" json:"status"`
	LastError    string     `gorm:"type:text" json:"last_error,omitempty"`
	LastDeployed *time.Time `gorm:"column:last_deployed_at" json:"last_deployed_at,omitempty"`

	Project Project `gorm:"foreignKey:ProjectID" json:"project"`
	Source  *Source `gorm:"foreignKey:SourceID" json:"source_record,omitempty"`

	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

func (s *Service) Slug() string {
	if len(s.UUID) >= 8 {
		return s.UUID[:8]
	}

	return s.UUID
}

// ComposeProject is the docker compose project name used for the stack.
func (s *Service) ComposeProject() string {
	return "ebox-" + s.Slug()
}

func (s *Service) IsGit() bool {
	return s.SourceType == ServiceSourceGit
}
