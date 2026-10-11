package stacks

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/google/uuid"

	"ebox/app/facades"
	"ebox/app/models"
	"ebox/app/services/compose"
	"ebox/app/services/docker"
	"ebox/app/services/domains"
	"ebox/app/services/envvars"
	"ebox/app/services/git"
	"ebox/app/services/projects"
	"ebox/app/services/proxy"
	"ebox/app/services/sources"
)

const (
	composeFileName  = "compose.yml"
	overrideFileName = "override.yml"
	envFileName      = ".env"
)

type Input struct {
	ProjectUUID string
	Name        string
	Source      string
	Compose     string
	SourceUUID  string
	Repository  string
	Branch      string
	ComposePath string
	Environment string
	// EntryService receives the ingress labels for the stack domains.
	EntryService string
	// EntryPort is the container port Traefik targets.
	EntryPort int
}

// ServiceStatus is the live state of one container of the stack.
type ServiceStatus struct {
	Name          string   `json:"name"`
	State         string   `json:"state"`
	Health        string   `json:"health,omitempty"`
	ContainerID   string   `json:"container_id,omitempty"`
	ContainerName string   `json:"container_name,omitempty"`
	Image         string   `json:"image,omitempty"`
	Ports         []string `json:"ports,omitempty"`
}

// Status aggregates the stack state.
type Status struct {
	State    string          `json:"state"`
	Services []ServiceStatus `json:"services"`
}

type Service struct{}

func Default() *Service {
	return &Service{}
}

func (s *Service) List(projectUUID string) ([]models.Service, error) {
	query := facades.Orm().Query().Order("name asc")

	if projectUUID != "" {
		project, err := projects.Default().Find(projectUUID)
		if err != nil {
			return nil, err
		}
		query = query.Where("project_id", project.ID)
	}

	var list []models.Service
	if err := query.Find(&list); err != nil {
		return nil, err
	}

	return s.hydrate(list)
}

func (s *Service) Find(uuidValue string) (*models.Service, error) {
	var service models.Service
	if err := facades.Orm().Query().Where("uuid", uuidValue).FirstOrFail(&service); err != nil {
		return nil, err
	}

	hydrated, err := s.hydrate([]models.Service{service})
	if err != nil {
		return nil, err
	}

	return &hydrated[0], nil
}

func (s *Service) hydrate(list []models.Service) ([]models.Service, error) {
	if len(list) == 0 {
		return list, nil
	}

	projectIDs := []uint64{}
	sourceIDs := []uint64{}
	seenProjects := map[uint64]bool{}
	seenSources := map[uint64]bool{}

	for _, item := range list {
		if !seenProjects[item.ProjectID] {
			seenProjects[item.ProjectID] = true
			projectIDs = append(projectIDs, item.ProjectID)
		}
		if item.SourceID != nil && !seenSources[*item.SourceID] {
			seenSources[*item.SourceID] = true
			sourceIDs = append(sourceIDs, *item.SourceID)
		}
	}

	var projectRecords []models.Project
	if err := facades.Orm().Query().Where("id in ?", projectIDs).Find(&projectRecords); err != nil {
		return nil, err
	}

	projectMap := map[uint64]models.Project{}
	for _, project := range projectRecords {
		projectMap[project.ID] = project
	}

	sourceMap := map[uint64]models.Source{}
	if len(sourceIDs) > 0 {
		var sourceRecords []models.Source
		if err := facades.Orm().Query().Where("id in ?", sourceIDs).Find(&sourceRecords); err != nil {
			return nil, err
		}
		for _, source := range sourceRecords {
			sourceMap[source.ID] = source
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

// Create stores the stack and runs it for the first time.
func (s *Service) Create(ctx context.Context, input Input) (*models.Service, error) {
	project, err := projects.Default().Find(input.ProjectUUID)
	if err != nil {
		return nil, errors.New("services: the project does not exist")
	}

	service := &models.Service{
		UUID:        uuid.NewString(),
		ProjectID:   project.ID,
		Status:      models.ServiceStatusIdle,
		SourceType:  models.ServiceSourceInline,
		Branch:      "main",
		Environment: input.Environment,
	}

	if err := s.applyInput(service, input); err != nil {
		return nil, err
	}

	if err := facades.Orm().Query().Create(service); err != nil {
		return nil, err
	}

	// Validate before touching Docker so obvious mistakes fail fast.
	if _, err := s.Project(ctx, service, true); err != nil {
		_, _ = facades.Orm().Query().Delete(service)

		return nil, err
	}

	if err := s.Deploy(ctx, service, nil); err != nil {
		service.Status = models.ServiceStatusFailed
		service.LastError = err.Error()
		_ = facades.Orm().Query().Save(service)

		return service, err
	}

	service.Project = *project

	return service, nil
}

func (s *Service) Update(ctx context.Context, service *models.Service, input Input) error {
	if err := s.applyInput(service, input); err != nil {
		return err
	}

	return facades.Orm().Query().Save(service)
}

func (s *Service) applyInput(service *models.Service, input Input) error {
	name := strings.TrimSpace(input.Name)
	if name == "" {
		return errors.New("services: the name is required")
	}

	service.Name = name
	service.Environment = input.Environment

	source := input.Source
	if source == "" {
		source = models.ServiceSourceInline
	}

	switch source {
	case models.ServiceSourceInline:
		for _, candidate := range []string{input.Compose, service.Compose} {
			if strings.TrimSpace(candidate) == "" {
				continue
			}

			if _, err := compose.Parse(candidate); err != nil {
				return err
			}

			service.Compose = candidate
			service.SourceType = models.ServiceSourceInline
			service.SourceID = nil
			service.Repository = ""
			service.ComposePath = ""
			applyEntry(service, input)

			return nil
		}

		return errors.New("services: a compose file is required")
	case models.ServiceSourceGit:
		if input.SourceUUID == "" {
			return errors.New("services: a source is required")
		}

		sourceRecord, err := sources.Default().Find(input.SourceUUID)
		if err != nil {
			return errors.New("services: the source does not exist")
		}

		path := strings.TrimSpace(input.ComposePath)
		if path == "" {
			path = "docker-compose.yml"
		}

		service.SourceType = models.ServiceSourceGit
		service.SourceID = &sourceRecord.ID
		service.Repository = strings.TrimSpace(input.Repository)
		service.Branch = strings.TrimSpace(input.Branch)
		if service.Branch == "" {
			service.Branch = "main"
		}
		service.ComposePath = path
		service.Compose = ""
		applyEntry(service, input)

		return nil
	default:
		return fmt.Errorf("services: unsupported source %q", source)
	}
}

// EntryService returns the compose service that receives ingress labels.
func (s *Service) EntryService(ctx context.Context, service *models.Service) (string, int, error) {
	document, err := s.Document(ctx, service, false)
	if err != nil {
		return "", 0, err
	}

	if service.EntryService != "" {
		if definition, ok := document.Services[service.EntryService]; ok {
			return service.EntryService, s.entryPort(service, definition), nil
		}
	}

	candidates := document.EntryCandidates()
	if len(candidates) == 0 {
		return "", 0, errors.New("services: the compose file has no services")
	}

	definition := document.Services[candidates[0].Name]

	return candidates[0].Name, s.entryPort(service, definition), nil
}

func (s *Service) entryPort(service *models.Service, definition compose.Service) int {
	if service.EntryPort > 0 {
		return service.EntryPort
	}

	if port := definition.ContainerPort(); port > 0 {
		return port
	}

	if port := definition.PublishedPort(); port > 0 {
		return port
	}

	return 80
}

func atoi(value string) (int, error) {
	var parsed int
	for _, character := range value {
		if character < '0' || character > '9' {
			return 0, errors.New("not a number")
		}
		parsed = parsed*10 + int(character-'0')
	}

	if value == "" {
		return 0, errors.New("empty")
	}

	return parsed, nil
}

// File returns the compose document for a stack, from storage or the checkout.
func (s *Service) Document(ctx context.Context, service *models.Service, fetch bool) (*compose.Document, error) {
	content, err := s.composeContent(ctx, service, fetch)
	if err != nil {
		return nil, err
	}

	return compose.Parse(content)
}

func (s *Service) composeContent(ctx context.Context, service *models.Service, fetch bool) (string, error) {
	if !service.IsGit() {
		if strings.TrimSpace(service.Compose) == "" {
			return "", errors.New("services: the stack has no compose file yet")
		}

		return service.Compose, nil
	}

	directory := s.sourceDirectory(service)
	path := filepath.Join(directory, service.ComposePath)

	if fetch || !fileExists(path) {
		if err := s.fetch(ctx, service, nil); err != nil {
			return "", err
		}
	}

	content, err := os.ReadFile(filepath.Join(directory, service.ComposePath))
	if err != nil {
		return "", fmt.Errorf("services: %s was not found in the repository", service.ComposePath)
	}

	return string(content), nil
}

func (s *Service) fetch(ctx context.Context, service *models.Service, sink func(string)) error {
	if !service.IsGit() || service.SourceID == nil {
		return nil
	}

	clone, err := sources.Default().CloneInfoFor(ctx, *service.SourceID, service.Repository)
	if err != nil {
		return err
	}

	_, err = git.Default().Sync(ctx, git.SyncOptions{
		Directory:   s.sourceDirectory(service),
		URL:         clone.URL,
		Credentials: clone.Credentials,
		Branch:      service.Branch,
		Sink:        sink,
	})

	return err
}

func (s *Service) sourceDirectory(service *models.Service) string {
	return filepath.Join(s.Directory(service), "source")
}

// Directory is the working directory that holds the stack files.
func (s *Service) Directory(service *models.Service) string {
	return filepath.Join(facades.Config().GetString("ebox.services_dir"), service.UUID)
}

// Project renders the compose files on disk and returns the project handle.
func (s *Service) Project(ctx context.Context, service *models.Service, fetch bool) (compose.Project, error) {
	directory := s.Directory(service)
	if err := os.MkdirAll(directory, 0o755); err != nil {
		return compose.Project{}, err
	}

	content, err := s.composeContent(ctx, service, fetch)
	if err != nil {
		return compose.Project{}, err
	}

	document, err := compose.Parse(content)
	if err != nil {
		return compose.Project{}, err
	}

	basePath := filepath.Join(directory, composeFileName)
	if service.IsGit() {
		basePath = filepath.Join(s.sourceDirectory(service), service.ComposePath)
	}

	if !service.IsGit() {
		if err := os.WriteFile(basePath, []byte(content), 0o600); err != nil {
			return compose.Project{}, err
		}
	}

	resourceDomains, err := domains.Default().List(models.ResourceTypeService, service.ID)
	if err != nil {
		return compose.Project{}, err
	}

	values, err := envvars.Default().Values(models.ResourceTypeService, service.ID, false)
	if err != nil {
		return compose.Project{}, err
	}

	environment := map[string]string{}
	for key, value := range values {
		environment[key] = value
	}

	entryService, entryPort, err := s.EntryService(ctx, service)
	if err != nil {
		return compose.Project{}, err
	}

	override, err := compose.Render(document, compose.Override{
		Project:      service.ComposeProject(),
		Network:      proxy.DefaultNetwork,
		Labels:       docker.ManagedLabels(docker.ResourceService, service.UUID, service.Name),
		Environment:  environment,
		EntryService: entryService,
		EntryPort:    entryPort,
		EntryLabels: proxy.IngressLabels(proxy.Ingress{
			ResourceType: docker.ResourceService,
			ResourceID:   service.UUID,
			Name:         service.Name + "/" + entryService,
			Router:       "ebox-" + service.Slug(),
			Port:         entryPort,
			Domains:      resourceDomains,
			Network:      proxy.DefaultNetwork,
		}),
	})
	if err != nil {
		return compose.Project{}, err
	}

	overridePath := filepath.Join(directory, overrideFileName)
	if err := os.WriteFile(overridePath, []byte(override), 0o600); err != nil {
		return compose.Project{}, err
	}

	envPath := filepath.Join(directory, envFileName)
	if err := writeEnvFile(envPath, service.Environment); err != nil {
		return compose.Project{}, err
	}

	return compose.Project{
		Name:    service.ComposeProject(),
		Dir:     directory,
		Files:   []string{basePath, overridePath},
		EnvFile: envPath,
	}, nil
}

// Deploy brings the stack up, building images when needed.
func (s *Service) Deploy(ctx context.Context, service *models.Service, sink func(string)) error {
	project, err := s.Project(ctx, service, true)
	if err != nil {
		s.markFailed(service, err)

		return err
	}

	if err := docker.Default().EnsureNetwork(ctx, proxy.DefaultNetwork); err != nil {
		s.markFailed(service, err)

		return err
	}

	line(sink, "Starting compose project "+project.Name)

	service.Status = models.ServiceStatusDeploying
	service.LastError = ""
	_ = facades.Orm().Query().Save(service)

	if err := compose.NewRunner().Up(ctx, project, true, sink); err != nil {
		s.markFailed(service, err)

		return err
	}

	now := time.Now().UTC()
	service.Status = models.ServiceStatusRunning
	service.LastDeployed = &now
	service.LastError = ""

	return facades.Orm().Query().Save(service)
}

func (s *Service) Stop(ctx context.Context, service *models.Service, sink func(string)) error {
	project, err := s.Project(ctx, service, false)
	if err != nil {
		return err
	}

	if err := compose.NewRunner().Stop(ctx, project, sink); err != nil {
		return err
	}

	service.Status = models.ServiceStatusIdle

	return facades.Orm().Query().Save(service)
}

func (s *Service) Start(ctx context.Context, service *models.Service, sink func(string)) error {
	project, err := s.Project(ctx, service, false)
	if err != nil {
		return err
	}

	if err := compose.NewRunner().Start(ctx, project, sink); err != nil {
		return err
	}

	service.Status = models.ServiceStatusRunning

	return facades.Orm().Query().Save(service)
}

func (s *Service) Restart(ctx context.Context, service *models.Service, sink func(string)) error {
	project, err := s.Project(ctx, service, false)
	if err != nil {
		return err
	}

	if err := compose.NewRunner().Restart(ctx, project, sink); err != nil {
		return err
	}

	service.Status = models.ServiceStatusRunning

	return facades.Orm().Query().Save(service)
}

// Delete tears the stack down and removes its files.
func (s *Service) Delete(ctx context.Context, service *models.Service) error {
	if project, err := s.Project(ctx, service, false); err == nil {
		_ = compose.NewRunner().Down(ctx, project, nil)
	} else {
		// Fall back to removing containers by label when the files are gone.
		containers, listErr := docker.Default().List(ctx, docker.ComposeFilter(service.ComposeProject()))
		if listErr == nil {
			for _, container := range containers {
				_ = docker.Default().Remove(context.WithoutCancel(ctx), container.ID, true)
			}
		}
	}

	_ = os.RemoveAll(s.Directory(service))

	_, err := facades.Orm().Query().Delete(service)

	return err
}

// Status reports the state of every container in the stack.
func (s *Service) Status(ctx context.Context, service *models.Service) Status {
	status := Status{State: "missing", Services: []ServiceStatus{}}

	containers, err := docker.Default().List(ctx, docker.ComposeFilter(service.ComposeProject()))
	if err != nil {
		status.State = "unknown"

		return status
	}

	if len(containers) == 0 {
		return status
	}

	expected := expectedServices(ctx, s, service)
	running := 0
	states := map[string]string{}

	for _, container := range containers {
		name := container.Labels["com.docker.compose.service"]
		state := container.State
		if state == "running" {
			running++
		}

		states[name] = state

		status.Services = append(status.Services, ServiceStatus{
			Name:          name,
			State:         state,
			Health:        container.Health,
			ContainerID:   container.ID,
			ContainerName: container.Name,
			Image:         container.Image,
			Ports:         container.Ports,
		})
	}

	for _, name := range expected {
		if _, ok := states[name]; !ok {
			status.Services = append(status.Services, ServiceStatus{Name: name, State: "missing"})
		}
	}

	sort.Slice(status.Services, func(i, j int) bool { return status.Services[i].Name < status.Services[j].Name })

	switch {
	case running == 0:
		status.State = "stopped"
	case running < len(expected) && len(expected) > 0:
		status.State = "partial"
	default:
		status.State = "running"
	}

	return status
}

func expectedServices(ctx context.Context, s *Service, service *models.Service) []string {
	document, err := s.Document(ctx, service, false)
	if err != nil {
		return nil
	}

	return document.ServiceNames()
}

func (s *Service) markFailed(service *models.Service, err error) {
	service.Status = models.ServiceStatusFailed
	service.LastError = err.Error()
	_ = facades.Orm().Query().Save(service)
}

func writeEnvFile(path, content string) error {
	if content != "" && !strings.HasSuffix(content, "\n") {
		content += "\n"
	}

	return os.WriteFile(path, []byte(content), 0o600)
}

func fileExists(path string) bool {
	_, err := os.Stat(path)

	return err == nil
}

func line(sink func(string), value string) {
	if sink != nil {
		sink(value)
	}
}

// applyEntry records the ingress entry service chosen in the panel. An empty
// value keeps the detected default.
func applyEntry(service *models.Service, input Input) {
	service.EntryService = strings.TrimSpace(input.EntryService)

	service.EntryPort = input.EntryPort
	if service.EntryPort < 0 || service.EntryPort > 65535 {
		service.EntryPort = 0
	}
}
