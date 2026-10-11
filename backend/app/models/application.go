package models

import "time"

const (
	BuildPackDockerfile = "dockerfile"
	BuildPackStatic     = "static"
	BuildPackImage      = "image"
	BuildPackCompose    = "compose"

	ApplicationKindGit   = "git"
	ApplicationKindImage = "image"

	ApplicationStatusIdle      = "idle"
	ApplicationStatusDeploying = "deploying"
	ApplicationStatusFailed    = "failed"
)

type ResourceLimits struct {
	MemoryMB int     `json:"memory_mb,omitempty"`
	CPUs     float64 `json:"cpus,omitempty"`
}

type ApplicationConfig struct {
	BaseDirectory    string          `json:"base_directory"`
	DockerfilePath   string          `json:"dockerfile_path"`
	ComposePath      string          `json:"compose_path"`
	ComposeService   string          `json:"compose_service"`
	BuildCommand     string          `json:"build_command"`
	PublishDirectory string          `json:"publish_directory"`
	Image            string          `json:"image"`
	ExposePort       int             `json:"expose_port"`
	Limits           *ResourceLimits `json:"limits,omitempty"`
}

type Application struct {
	ID          uint64            `gorm:"primaryKey" json:"id"`
	UUID        string            `gorm:"size:36;uniqueIndex" json:"uuid"`
	ProjectID   uint64            `gorm:"index" json:"project_id"`
	SourceID    *uint64           `json:"source_id"`
	Name        string            `gorm:"size:255" json:"name"`
	Kind        string            `gorm:"size:16" json:"kind"`
	Repository  string            `gorm:"size:512" json:"repository"`
	Branch      string            `gorm:"size:255" json:"branch"`
	BuildPack   string            `gorm:"size:32" json:"build_pack"`
	Config      ApplicationConfig `gorm:"serializer:json;type:text" json:"config"`
	AutoDeploy  bool              `json:"auto_deploy"`
	DeployToken string            `gorm:"size:64;uniqueIndex" json:"-"`
	Status      string            `gorm:"size:32" json:"status"`

	Project Project `gorm:"foreignKey:ProjectID" json:"project"`
	Source  *Source `gorm:"foreignKey:SourceID" json:"source,omitempty"`

	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// IsCompose reports whether the application deploys a compose project.
func (a *Application) IsCompose() bool {
	return a.BuildPack == BuildPackCompose
}

func (a *Application) ExposePort() int {
	if a.Config.ExposePort > 0 {
		return a.Config.ExposePort
	}

	return 80
}

func (a *Application) Slug() string {
	if len(a.UUID) >= 8 {
		return a.UUID[:8]
	}

	return a.UUID
}

func (a *Application) ContainerName(tag string) string {
	name := "ebox-" + a.Slug()
	if tag != "" {
		name += "-" + tag
	}

	return name
}

func (a *Application) BuildContextDirectory() string {
	if a.Config.BaseDirectory != "" && a.Config.BaseDirectory != "/" {
		return a.Config.BaseDirectory
	}

	return "."
}
