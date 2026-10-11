package models

import "time"

const (
	BackupTargetDatabase = "database"
	BackupTargetVolume   = "volume"

	BackupStatusPending = "pending"
	BackupStatusRunning = "running"
	BackupStatusSuccess = "success"
	BackupStatusFailed  = "failed"
)

// Backup is a scheduled backup of a database or a docker volume.
type Backup struct {
	ID          uint64 `gorm:"primaryKey" json:"id"`
	UUID        string `gorm:"size:36;uniqueIndex" json:"uuid"`
	ProjectID   *uint64 `gorm:"index" json:"project_id,omitempty"`
	Name        string `gorm:"size:255" json:"name"`
	Target      string `gorm:"size:16" json:"target"`
	// ResourceID is the database uuid for database targets.
	ResourceID  string `gorm:"size:64" json:"resource_id"`
	VolumeName  string `gorm:"size:255" json:"volume_name"`
	Schedule    string `gorm:"size:64" json:"schedule"`
	Retention   int    `json:"retention"`
	Enabled     bool   `json:"enabled"`
	// StopDuringBackup pauses the container so file level copies are consistent.
	StopDuringBackup bool       `json:"stop_during_backup"`
	LastRunAt        *time.Time `json:"last_run_at,omitempty"`
	LastStatus       string     `gorm:"size:32" json:"last_status,omitempty"`
	LastError        string     `gorm:"type:text" json:"last_error,omitempty"`

	Project  *Project `gorm:"foreignKey:ProjectID" json:"project,omitempty"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// BackupRun is one execution of a backup.
type BackupRun struct {
	ID          uint64     `gorm:"primaryKey" json:"id"`
	BackupID    uint64     `gorm:"index" json:"backup_id"`
	Status      string     `gorm:"size:32;index" json:"status"`
	Trigger     string     `gorm:"size:16" json:"trigger"`
	File        string     `gorm:"size:512" json:"file"`
	SizeBytes   int64      `json:"size_bytes"`
	DurationMS  int64      `json:"duration_ms"`
	Error       string     `gorm:"type:text" json:"error,omitempty"`
	StartedAt   *time.Time `json:"started_at,omitempty"`
	FinishedAt  *time.Time `json:"finished_at,omitempty"`
	CreatedAt   time.Time  `json:"created_at"`
	UpdatedAt   time.Time  `json:"updated_at"`
}

func (b *Backup) Slug() string {
	if len(b.UUID) >= 8 {
		return b.UUID[:8]
	}

	return b.UUID
}
