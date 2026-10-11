package databases

import (
	"context"
	"encoding/json"
	"errors"
	"strconv"
	"strings"
	"time"

	"github.com/google/uuid"

	"ebox/app/facades"
	"ebox/app/models"
	"ebox/app/services/crypto"
	"ebox/app/services/docker"
	"ebox/app/services/projects"
)

const (
	createTimeout = 3 * time.Minute
	networkName   = "ebox"
)

type Input struct {
	ProjectUUID string
	Name        string
	Type        string
	Version     string
	Port        int
	Limits      *models.ResourceLimits
}

type Status struct {
	State         string `json:"state"`
	Health        string `json:"health,omitempty"`
	ContainerID   string `json:"container_id,omitempty"`
	ContainerName string `json:"container_name,omitempty"`
	Image         string `json:"image,omitempty"`
	Uptime        string `json:"uptime,omitempty"`
}

// Connection is everything an application needs to reach the database.
type Connection struct {
	Host           string            `json:"host"`
	Port           int               `json:"port"`
	PublicHost     string            `json:"public_host,omitempty"`
	PublicPort     int               `json:"public_port,omitempty"`
	Username       string            `json:"username"`
	Password       string            `json:"password,omitempty"`
	Database       string            `json:"database"`
	URL            string            `json:"url,omitempty"`
	PublicURL      string            `json:"public_url,omitempty"`
	Environment    map[string]string `json:"environment"`
	InternalHost   string            `json:"internal_host"`
	ExposedToWorld bool              `json:"exposed"`
}

type Service struct{}

func Default() *Service {
	return &Service{}
}

func (s *Service) List(projectUUID string) ([]models.Database, error) {
	query := facades.Orm().Query().Order("name asc")

	if projectUUID != "" {
		project, err := projects.Default().Find(projectUUID)
		if err != nil {
			return nil, err
		}
		query = query.Where("project_id", project.ID)
	}

	var list []models.Database
	if err := query.Find(&list); err != nil {
		return nil, err
	}

	return s.hydrate(list)
}

func (s *Service) Find(uuidValue string) (*models.Database, error) {
	var database models.Database
	if err := facades.Orm().Query().Where("uuid", uuidValue).FirstOrFail(&database); err != nil {
		return nil, err
	}

	hydrated, err := s.hydrate([]models.Database{database})
	if err != nil {
		return nil, err
	}

	return &hydrated[0], nil
}

func (s *Service) hydrate(list []models.Database) ([]models.Database, error) {
	if len(list) == 0 {
		return list, nil
	}

	projectIDs := make([]uint64, 0, len(list))
	seen := map[uint64]bool{}
	for _, item := range list {
		if !seen[item.ProjectID] {
			seen[item.ProjectID] = true
			projectIDs = append(projectIDs, item.ProjectID)
		}
	}

	var projects []models.Project
	if err := facades.Orm().Query().Where("id in ?", projectIDs).Find(&projects); err != nil {
		return nil, err
	}

	projectMap := map[uint64]models.Project{}
	for _, project := range projects {
		projectMap[project.ID] = project
	}

	for index := range list {
		list[index].Project = projectMap[list[index].ProjectID]
	}

	return list, nil
}

// Create provisions the container and its data volume.
func (s *Service) Create(ctx context.Context, input Input) (*models.Database, error) {
	project, err := projects.Default().Find(input.ProjectUUID)
	if err != nil {
		return nil, errors.New("databases: the project does not exist")
	}

	definition, err := DefinitionFor(input.Type)
	if err != nil {
		return nil, err
	}

	name := strings.TrimSpace(input.Name)
	if name == "" {
		return nil, errors.New("databases: the name is required")
	}

	version := strings.TrimSpace(input.Version)
	if version == "" {
		version = definition.DefaultVersion()
	}

	port := input.Port
	if port < 0 || port > 65535 {
		return nil, errors.New("databases: the port must be between 1 and 65535")
	}

	credentials, err := generateCredentials(definition, name)
	if err != nil {
		return nil, err
	}

	encoded, err := encodeCredentials(credentials)
	if err != nil {
		return nil, err
	}

	limits := input.Limits
	if limits != nil && limits.MemoryMB <= 0 && limits.CPUs <= 0 {
		limits = nil
	}

	database := &models.Database{
		UUID:        uuid.NewString(),
		ProjectID:   project.ID,
		Name:        name,
		Type:        definition.Type,
		Version:     version,
		Port:        port,
		Credentials: encoded,
		Limits:      limits,
		Status:      models.DatabaseStatusCreating,
	}

	if err := facades.Orm().Query().Create(database); err != nil {
		return nil, err
	}

	if err := s.start(ctx, database, definition, credentials); err != nil {
		database.Status = models.DatabaseStatusFailed
		database.StatusMessage = err.Error()
		_ = facades.Orm().Query().Save(database)
		_ = s.Remove(context.WithoutCancel(ctx), database)

		return nil, err
	}

	database.Status = models.DatabaseStatusReady
	database.StatusMessage = ""
	if err := facades.Orm().Query().Save(database); err != nil {
		return nil, err
	}

	database.Project = *project

	return database, nil
}

func (s *Service) start(ctx context.Context, database *models.Database, definition Definition, credentials models.DatabaseCredentials) error {
	if err := docker.Default().EnsureNetwork(ctx, networkName); err != nil {
		return err
	}

	if err := docker.Default().EnsureVolume(ctx, database.VolumeName(), docker.ManagedLabels(docker.ResourceDatabase, database.UUID, database.Name)); err != nil {
		return err
	}

	labels := docker.ManagedLabels(docker.ResourceDatabase, database.UUID, database.Name)

	imageRef := definition.ImageRef(database.Version)

	if err := docker.Default().PullImage(ctx, imageRef, nil); err != nil {
		return err
	}

	spec := docker.Spec{
		Name:         database.ContainerName(),
		Image:        imageRef,
		Env:          definition.Environment(credentials),
		Command:      definition.Command(credentials),
		Labels:       labels,
		Network:      networkName,
		Alias:        database.ContainerName(),
		Restart:      "unless-stopped",
		StopTimeout:  30 * time.Second,
		Volumes:      []string{database.VolumeName() + ":" + definition.VolumePath},
		Healthcheck:  definition.Healthcheck,
		ExposedPorts: []string{strconv.Itoa(definition.InternalPort) + "/tcp"},
	}

	if database.Public() {
		spec.PortBindings = []docker.PortBinding{{
			ContainerPort: strconv.Itoa(definition.InternalPort) + "/tcp",
			HostPort:      strconv.Itoa(database.Port),
			HostIP:        "0.0.0.0",
		}}
	}

	if database.Limits != nil {
		spec.MemoryMB = database.Limits.MemoryMB
		spec.CPUs = database.Limits.CPUs
	}

	if err := docker.Default().RemoveByName(ctx, spec.Name, true); err != nil {
		return err
	}

	containerID, err := docker.Default().Create(ctx, spec)
	if err != nil {
		return err
	}

	if _, err := docker.Default().WaitRunning(ctx, containerID, time.Minute); err != nil {
		return err
	}

	return nil
}

// WaitReady blocks until the database container reports healthy.
func (s *Service) WaitReady(ctx context.Context, database *models.Database) error {
	containers, err := docker.Default().FindByResource(ctx, docker.ResourceDatabase, database.UUID)
	if err != nil || len(containers) == 0 {
		return errors.New("databases: the container is not available")
	}

	return docker.Default().WaitHealthy(ctx, containers[0].ID, createTimeout)
}

func (s *Service) Update(database *models.Database, input Input) error {
	name := strings.TrimSpace(input.Name)
	if name == "" {
		return errors.New("databases: the name is required")
	}

	version := strings.TrimSpace(input.Version)
	if version == "" {
		version = database.Version
	}

	port := input.Port
	if port < 0 || port > 65535 {
		return errors.New("databases: the port must be between 1 and 65535")
	}

	limits := input.Limits
	if limits != nil && limits.MemoryMB <= 0 && limits.CPUs <= 0 {
		limits = nil
	}

	database.Name = name
	database.Port = port
	database.Limits = limits
	requiresRestart := database.Version != version || database.Public() != (port > 0)
	database.Version = version

	if err := facades.Orm().Query().Save(database); err != nil {
		return err
	}

	if requiresRestart {
		return s.Restart(context.Background(), database)
	}

	return nil
}

func (s *Service) Start(ctx context.Context, database *models.Database, definition Definition) error {
	containers, err := docker.Default().FindByResource(ctx, docker.ResourceDatabase, database.UUID)
	if err == nil && len(containers) > 0 {
		return docker.Default().Start(ctx, containers[0].ID)
	}

	credentials, err := s.Credentials(database)
	if err != nil {
		return err
	}

	return s.start(ctx, database, definition, credentials)
}

func (s *Service) Stop(ctx context.Context, database *models.Database) error {
	containers, err := docker.Default().FindByResource(ctx, docker.ResourceDatabase, database.UUID)
	if err != nil || len(containers) == 0 {
		return nil
	}

	return docker.Default().Stop(ctx, containers[0].ID, 30*time.Second)
}

func (s *Service) Restart(ctx context.Context, database *models.Database) error {
	definition, err := DefinitionFor(database.Type)
	if err != nil {
		return err
	}

	credentials, err := s.Credentials(database)
	if err != nil {
		return err
	}

	containers, _ := docker.Default().FindByResource(ctx, docker.ResourceDatabase, database.UUID)
	_ = docker.Default().RemoveByName(ctx, database.ContainerName(), true)
	for _, container := range containers {
		_ = docker.Default().Remove(context.WithoutCancel(ctx), container.ID, true)
	}

	if err := s.start(ctx, database, definition, credentials); err != nil {
		return err
	}

	return s.WaitReady(ctx, database)
}

// Remove deletes the container and, when drop is set, the data volume.
func (s *Service) Remove(ctx context.Context, database *models.Database) error {
	containers, err := docker.Default().FindByResource(ctx, docker.ResourceDatabase, database.UUID)
	if err == nil {
		for _, container := range containers {
			_ = docker.Default().Remove(ctx, container.ID, true)
		}
	}

	_ = docker.Default().RemoveByName(ctx, database.ContainerName(), true)
	_ = docker.Default().RemoveVolume(context.WithoutCancel(ctx), database.VolumeName())

	_, err = facades.Orm().Query().Delete(database)

	return err
}

// Status reports the live container state.
func (s *Service) Status(ctx context.Context, database *models.Database) Status {
	definition, _ := DefinitionFor(database.Type)

	status := Status{State: "missing"}

	containers, err := docker.Default().FindByResource(ctx, docker.ResourceDatabase, database.UUID)
	if err != nil {
		status.State = "unknown"

		return status
	}

	for _, container := range containers {
		if container.State == "running" {
			status.State = "running"
			status.Health = container.Health
			status.ContainerID = container.ID
			status.ContainerName = container.Name
			status.Image = container.Image

			if container.Created > 0 {
				uptime := time.Since(time.Unix(container.Created, 0))
				status.Uptime = uptime.Truncate(time.Second).String()
			}

			return status
		}

		status.State = container.State
		status.ContainerID = container.ID
		status.ContainerName = container.Name
		status.Image = container.Image
	}

	if status.Image == "" {
		status.Image = definition.ImageRef(database.Version)
	}

	return status
}

// Credentials decrypts the stored credentials.
func (s *Service) Credentials(database *models.Database) (models.DatabaseCredentials, error) {
	var credentials models.DatabaseCredentials

	decoded, err := crypto.Decrypt(database.Credentials)
	if err != nil {
		return credentials, errors.New("databases: the stored credentials could not be read")
	}

	if err := json.Unmarshal([]byte(decoded), &credentials); err != nil {
		return credentials, errors.New("databases: the stored credentials are invalid")
	}

	return credentials, nil
}

// Connection assembles the connection details, optionally including secrets.
func (s *Service) Connection(ctx context.Context, database *models.Database, reveal bool) (Connection, error) {
	definition, err := DefinitionFor(database.Type)
	if err != nil {
		return Connection{}, err
	}

	credentials, err := s.Credentials(database)
	if err != nil {
		return Connection{}, err
	}

	host := database.ContainerName()
	connection := Connection{
		Host:         host,
		InternalHost: host,
		Port:         definition.InternalPort,
		Username:     credentials.Username,
		Database:     credentials.Database,
		Environment:  definition.EnvironmentVariables(host, definition.InternalPort, credentials),
	}

	if database.Public() {
		connection.PublicHost = externalHost(ctx)
		connection.PublicPort = database.Port
		connection.ExposedToWorld = true
	}

	// Everything that embeds the password is only returned when the caller asked
	// for secrets, so a masked view can never leak them.
	if reveal {
		connection.Password = credentials.Password
		connection.URL = definition.URL(host, definition.InternalPort, credentials, false)

		if database.Public() {
			connection.PublicURL = definition.URL(connection.PublicHost, database.Port, credentials, true)
		}
	} else {
		connection.Environment = map[string]string{}
	}

	return connection, nil
}

// externalHost reports the address clients should use to reach published
// ports, falling back to the panel host.
func externalHost(ctx context.Context) string {
	host := facades.Config().GetString("ebox.public_host", "")
	if host != "" {
		return host
	}

	return "localhost"
}

func generateCredentials(definition Definition, name string) (models.DatabaseCredentials, error) {
	password, err := crypto.RandomToken(24)
	if err != nil {
		return models.DatabaseCredentials{}, err
	}

	return models.DatabaseCredentials{
		Username: definition.Username(),
		Password: password,
		Database: NormalizeDatabaseName(name),
	}, nil
}

func encodeCredentials(credentials models.DatabaseCredentials) (string, error) {
	payload, err := json.Marshal(credentials)
	if err != nil {
		return "", err
	}

	return crypto.Encrypt(string(payload))
}
