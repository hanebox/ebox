package models

import "time"

const (
	DatabaseTypePostgres = "postgres"
	DatabaseTypeMySQL    = "mysql"
	DatabaseTypeMariaDB  = "mariadb"
	DatabaseTypeRedis    = "redis"
	DatabaseTypeMongo    = "mongo"

	DatabaseStatusCreating = "creating"
	DatabaseStatusReady    = "ready"
	DatabaseStatusFailed   = "failed"
)

// DatabaseCredentials holds the generated superuser credentials. The struct is
// stored encrypted at rest.
type DatabaseCredentials struct {
	Username string `json:"username"`
	Password string `json:"password"`
	Database string `json:"database"`
}

type Database struct {
	ID            uint64          `gorm:"primaryKey" json:"id"`
	UUID          string          `gorm:"size:36;uniqueIndex" json:"uuid"`
	ProjectID     uint64          `gorm:"index" json:"project_id"`
	Name          string          `gorm:"size:255" json:"name"`
	Type          string          `gorm:"size:32" json:"type"`
	Version       string          `gorm:"size:32" json:"version"`
	Port          int             `json:"port"`
	Credentials   string          `gorm:"type:text" json:"-"`
	Limits        *ResourceLimits `gorm:"serializer:json;type:text" json:"limits,omitempty"`
	Status        string          `gorm:"size:32" json:"status"`
	StatusMessage string          `gorm:"type:text" json:"status_message,omitempty"`

	Project Project `gorm:"foreignKey:ProjectID" json:"project"`

	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

func (d *Database) Slug() string {
	if len(d.UUID) >= 8 {
		return d.UUID[:8]
	}

	return d.UUID
}

// ContainerName is both the container name and the network alias applications
// use to reach the database.
func (d *Database) ContainerName() string {
	return "ebox-db-" + d.Slug()
}

func (d *Database) VolumeName() string {
	return "ebox-db-" + d.Slug() + "-data"
}

// Public reports whether the database publishes a host port.
func (d *Database) Public() bool {
	return d.Port > 0
}
