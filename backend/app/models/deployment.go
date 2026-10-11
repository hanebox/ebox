package models

import "time"

const (
	DeploymentStatusQueued    = "queued"
	DeploymentStatusBuilding  = "building"
	DeploymentStatusDeploying = "deploying"
	DeploymentStatusSuccess   = "success"
	DeploymentStatusFailed    = "failed"
	DeploymentStatusCancelled = "cancelled"

	DeploymentTriggerManual   = "manual"
	DeploymentTriggerWebhook  = "webhook"
	DeploymentTriggerToken    = "token"
	DeploymentTriggerRollback = "rollback"
)

type Deployment struct {
	ID             uint64     `gorm:"primaryKey" json:"id"`
	UUID           string     `gorm:"size:36;uniqueIndex" json:"uuid"`
	ApplicationID  uint64     `gorm:"index" json:"application_id"`
	Status         string     `gorm:"size:32" json:"status"`
	Trigger        string     `gorm:"size:16" json:"trigger"`
	Commit         string     `gorm:"size:64" json:"commit"`
	CommitMessage  string     `gorm:"size:512" json:"commit_message"`
	ImageTag       string     `gorm:"size:255" json:"image_tag"`
	ComposeProject string     `gorm:"size:128" json:"compose_project"`
	ContainerID    string     `gorm:"size:64" json:"container_id"`
	ContainerName  string     `gorm:"size:128" json:"container_name"`
	Log            string     `gorm:"type:text" json:"-"`
	Error          string     `gorm:"type:text" json:"error"`
	StartedAt      *time.Time `json:"started_at"`
	FinishedAt     *time.Time `json:"finished_at"`
	DurationMS     int64      `json:"duration_ms"`

	Application Application `gorm:"foreignKey:ApplicationID" json:"-"`

	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

func (d *Deployment) Active() bool {
	switch d.Status {
	case DeploymentStatusQueued, DeploymentStatusBuilding, DeploymentStatusDeploying:
		return true
	}

	return false
}

func (d *Deployment) ShortTag() string {
	if len(d.UUID) >= 7 {
		return d.UUID[:7]
	}

	return d.UUID
}
