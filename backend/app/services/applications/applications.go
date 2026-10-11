package applications

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"

	"ebox/app/facades"
	"ebox/app/models"
	"ebox/app/services/crypto"
	"ebox/app/services/docker"
	"ebox/app/services/projects"
	"ebox/app/services/sources"
)

type Input struct {
	ProjectUUID string
	Name        string
	SourceUUID  string
	Repository  string
	Branch      string
	BuildPack   string
	Config      models.ApplicationConfig
	AutoDeploy  bool
}

type Status struct {
	State         string                 `json:"state"`
	Health        string                 `json:"health,omitempty"`
	ContainerID   string                 `json:"container_id,omitempty"`
	ContainerName string                 `json:"container_name,omitempty"`
	Image         string                 `json:"image,omitempty"`
	StartedAt     string                 `json:"started_at,omitempty"`
	Uptime        string                 `json:"uptime,omitempty"`
	Deployment    *models.Deployment     `json:"latest_deployment,omitempty"`
	Limits        *models.ResourceLimits `json:"limits,omitempty"`
}

type Service struct{}

func Default() *Service {
	return &Service{}
}

func (s *Service) List(projectUUID string) ([]models.Application, error) {
	query := facades.Orm().Query().Order("name asc")

	if projectUUID != "" {
		project, err := projects.Default().Find(projectUUID)
		if err != nil {
			return nil, err
		}
		query = query.Where("project_id", project.ID)
	}

	var list []models.Application
	if err := query.Find(&list); err != nil {
		return nil, err
	}

	return s.hydrate(list)
}

func (s *Service) Find(uuidValue string) (*models.Application, error) {
	var app models.Application
	if err := facades.Orm().Query().Where("uuid", uuidValue).FirstOrFail(&app); err != nil {
		return nil, err
	}

	hydrated, err := s.hydrate([]models.Application{app})
	if err != nil {
		return nil, err
	}

	return &hydrated[0], nil
}

// hydrate attaches the related project and source records.
func (s *Service) hydrate(list []models.Application) ([]models.Application, error) {
	if len(list) == 0 {
		return list, nil
	}

	projectIDs := make([]uint64, 0, len(list))
	sourceIDs := make([]uint64, 0, len(list))
	seenProjects := map[uint64]bool{}
	seenSources := map[uint64]bool{}

	for _, app := range list {
		if !seenProjects[app.ProjectID] {
			seenProjects[app.ProjectID] = true
			projectIDs = append(projectIDs, app.ProjectID)
		}
		if app.SourceID != nil && !seenSources[*app.SourceID] {
			seenSources[*app.SourceID] = true
			sourceIDs = append(sourceIDs, *app.SourceID)
		}
	}

	projectMap := map[uint64]models.Project{}
	if len(projectIDs) > 0 {
		var records []models.Project
		if err := facades.Orm().Query().Where("id in ?", projectIDs).Find(&records); err != nil {
			return nil, err
		}
		for _, record := range records {
			projectMap[record.ID] = record
		}
	}

	sourceMap := map[uint64]models.Source{}
	if len(sourceIDs) > 0 {
		var records []models.Source
		if err := facades.Orm().Query().Where("id in ?", sourceIDs).Find(&records); err != nil {
			return nil, err
		}
		for _, record := range records {
			sourceMap[record.ID] = record
		}
	}

	for index := range list {
		list[index].Project = projectMap[list[index].ProjectID]
		if list[index].SourceID != nil {
			if source, ok := sourceMap[*list[index].SourceID]; ok {
				local := source
				list[index].Source = &local
			}
		}
	}

	return list, nil
}

func (s *Service) FindByID(id uint64) (*models.Application, error) {
	var app models.Application
	if err := facades.Orm().Query().Where("id", id).FirstOrFail(&app); err != nil {
		return nil, err
	}

	return &app, nil
}

func (s *Service) FindByDeployToken(token string) (*models.Application, error) {
	var app models.Application
	if err := facades.Orm().Query().Where("deploy_token", token).FirstOrFail(&app); err != nil {
		return nil, err
	}

	return &app, nil
}

func (s *Service) Create(input Input) (*models.Application, error) {
	project, err := projects.Default().Find(input.ProjectUUID)
	if err != nil {
		return nil, errors.New("applications: the project does not exist")
	}

	app := &models.Application{
		ProjectID:  project.ID,
		Name:       strings.TrimSpace(input.Name),
		AutoDeploy: input.AutoDeploy,
		Status:     models.ApplicationStatusIdle,
	}

	if err := s.applyInput(app, input); err != nil {
		return nil, err
	}

	app.UUID = uuid.NewString()
	app.DeployToken = newDeployToken()

	if err := facades.Orm().Query().Create(app); err != nil {
		return nil, err
	}

	return app, nil
}

func (s *Service) Update(app *models.Application, input Input) error {
	if err := s.applyInput(app, input); err != nil {
		return err
	}

	app.AutoDeploy = input.AutoDeploy

	return facades.Orm().Query().Save(app)
}

func (s *Service) Delete(app *models.Application) error {
	_, err := facades.Orm().Query().Delete(app)

	return err
}

func (s *Service) RegenerateDeployToken(app *models.Application) (string, error) {
	app.DeployToken = newDeployToken()

	if err := facades.Orm().Query().Save(app); err != nil {
		return "", err
	}

	return app.DeployToken, nil
}

// MatchingWebhook returns auto-deploying applications for a repository push.
func (s *Service) MatchingWebhook(sourceID uint64, repository, branch string) ([]models.Application, error) {
	var list []models.Application
	if err := facades.Orm().Query().
		Where("source_id", sourceID).
		Where("repository", repository).
		Where("branch", branch).
		Where("auto_deploy", true).
		Find(&list); err != nil {
		return nil, err
	}

	return list, nil
}

// Status reports the live container state and the latest deployment.
func (s *Service) Status(ctx context.Context, app *models.Application) Status {
	status := Status{State: "missing"}

	containers, err := docker.Default().FindByResource(ctx, docker.ResourceApplication, app.UUID)
	if err == nil {
		for _, item := range containers {
			if item.State == "running" {
				status.State = "running"
				status.Health = item.Health
				status.ContainerID = item.ID
				status.ContainerName = item.Name
				status.Image = item.Image
				status.StartedAt = time.Unix(item.Created, 0).UTC().Format(time.RFC3339)
				break
			}

			if status.State == "missing" {
				status.State = item.State
				status.ContainerID = item.ID
				status.ContainerName = item.Name
				status.Image = item.Image
			}
		}
	} else {
		status.State = "unknown"
	}

	var latest models.Deployment
	if err := facades.Orm().Query().
		Where("application_id", app.ID).
		Order("id desc").
		FirstOrFail(&latest); err == nil {
		status.Deployment = &latest
	}

	status.Limits = app.Config.Limits

	return status
}

func (s *Service) applyInput(app *models.Application, input Input) error {
	buildPack := input.BuildPack
	if buildPack == "" {
		buildPack = models.BuildPackDockerfile
	}

	switch buildPack {
	case models.BuildPackDockerfile, models.BuildPackStatic, models.BuildPackCompose:
		if input.SourceUUID == "" {
			return errors.New("applications: a source is required")
		}
	case models.BuildPackImage:
		if strings.TrimSpace(input.Config.Image) == "" {
			return errors.New("applications: a docker image is required")
		}
	default:
		return fmt.Errorf("applications: unknown build pack %q", buildPack)
	}

	app.Name = strings.TrimSpace(input.Name)
	if app.Name == "" {
		return errors.New("applications: the name is required")
	}
	if len(app.Name) > 255 {
		return errors.New("applications: the name may not be longer than 255 characters")
	}

	app.BuildPack = buildPack
	app.Repository = strings.TrimSpace(input.Repository)
	app.Branch = strings.TrimSpace(input.Branch)
	if app.Branch == "" {
		app.Branch = "main"
	}

	config := input.Config
	if config.ExposePort < 1 || config.ExposePort > 65535 {
		config.ExposePort = 80
	}
	if config.Limits != nil && config.Limits.MemoryMB <= 0 && config.Limits.CPUs <= 0 {
		config.Limits = nil
	}
	app.Config = config

	if buildPack == models.BuildPackImage {
		app.Kind = models.ApplicationKindImage
	} else {
		app.Kind = models.ApplicationKindGit
	}

	if input.SourceUUID != "" {
		source, err := sources.Default().Find(input.SourceUUID)
		if err != nil {
			return errors.New("applications: the source does not exist")
		}
		app.SourceID = &source.ID

		if source.Type == models.SourceTypeGit && app.Repository == "" {
			app.Repository = source.GitURL
		}
	} else {
		app.SourceID = nil
	}

	if app.Kind == models.ApplicationKindGit && app.Repository == "" {
		return errors.New("applications: a repository is required")
	}

	return nil
}

func newDeployToken() string {
	secret, err := crypto.RandomToken(16)
	if err != nil {
		return uuid.NewString()
	}

	return secret
}
