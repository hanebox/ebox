package deployment

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/goravel/framework/contracts/queue"

	"ebox/app/facades"
	"ebox/app/models"
	"ebox/app/services/build"
	"ebox/app/services/docker"
	"ebox/app/services/domains"
	"ebox/app/services/envvars"
	"ebox/app/services/git"
	"ebox/app/services/notifications"
	"ebox/app/services/proxy"
	"ebox/app/services/realtime"
	"ebox/app/services/sources"
)

const (
	containerStartTimeout = 60 * time.Second
	healthTimeout         = 90 * time.Second
	probeAttempts         = 6
)

type Service struct {
	cancels sync.Map
}

var defaultService = &Service{}

func Default() *Service {
	return defaultService
}

// Enqueue records a deployment and dispatches the deploy job.
func (s *Service) Enqueue(app *models.Application, trigger string, commit string) (*models.Deployment, error) {
	deployment := &models.Deployment{
		UUID:          uuid.NewString(),
		ApplicationID: app.ID,
		Status:        models.DeploymentStatusQueued,
		Trigger:       trigger,
		Commit:        commit,
	}

	if err := facades.Orm().Query().Create(deployment); err != nil {
		return nil, err
	}

	if err := s.dispatch(deployment); err != nil {
		_, _ = facades.Orm().Query().Delete(deployment)

		return nil, err
	}

	s.publish(deployment, "queued")

	return deployment, nil
}

// EnqueueRollback starts a deployment that reuses the image of an earlier
// deployment instead of building again.
func (s *Service) EnqueueRollback(app *models.Application, source *models.Deployment) (*models.Deployment, error) {
	if app.IsCompose() {
		if source.Commit == "" {
			return nil, errors.New("deployment: the selected deployment has no revision to roll back to")
		}
	} else if source.ImageTag == "" {
		return nil, errors.New("deployment: the selected deployment has no image to roll back to")
	}

	deployment := &models.Deployment{
		UUID:          uuid.NewString(),
		ApplicationID: app.ID,
		Status:        models.DeploymentStatusQueued,
		Trigger:       models.DeploymentTriggerRollback,
		Commit:        source.Commit,
		CommitMessage: source.CommitMessage,
		ImageTag:      source.ImageTag,
	}

	if err := facades.Orm().Query().Create(deployment); err != nil {
		return nil, err
	}

	if err := s.dispatch(deployment); err != nil {
		_, _ = facades.Orm().Query().Delete(deployment)

		return nil, err
	}

	s.publish(deployment, "queued")

	return deployment, nil
}

func (s *Service) dispatch(deployment *models.Deployment) error {
	connection := facades.Config().GetString("ebox.deploy.connection", "database")
	queueName := facades.Config().GetString("ebox.deploy.queue", "deployments")

	return facades.Queue().
		Job(&DeployApplicationJob{}, []queue.Arg{{Type: "string", Value: deployment.UUID}}).
		OnConnection(connection).
		OnQueue(queueName).
		Dispatch()
}

// Cancel stops a running deployment.
func (s *Service) Cancel(deploymentUUID string) bool {
	if cancel, ok := s.cancels.Load(deploymentUUID); ok {
		cancel.(context.CancelFunc)()

		return true
	}

	return false
}

// Run executes the deployment pipeline. It is called by the queue job.
func (s *Service) Run(deploymentUUID string) error {
	var deployment models.Deployment
	if err := facades.Orm().Query().Where("uuid", deploymentUUID).FirstOrFail(&deployment); err != nil {
		return err
	}

	if !deployment.Active() && deployment.Status != models.DeploymentStatusQueued {
		return nil
	}

	var app models.Application
	if err := facades.Orm().Query().Where("id", deployment.ApplicationID).FirstOrFail(&app); err != nil {
		s.fail(&deployment, nil, errors.New("the application no longer exists"))

		return nil
	}

	ctx, cancel := context.WithCancel(context.Background())
	s.cancels.Store(deploymentUUID, cancel)
	defer func() {
		s.cancels.Delete(deploymentUUID)
		cancel()
	}()

	log := newLogWriter(&deployment)
	defer log.Close()

	log.Line(fmt.Sprintf("Deployment %s started (%s)", deployment.UUID, deployment.Trigger))

	if err := s.execute(ctx, &app, &deployment, log); err != nil {
		if errors.Is(err, context.Canceled) {
			s.cancelDeployment(&deployment, log)

			return nil
		}

		s.fail(&deployment, log, err)

		return nil
	}

	s.succeed(&deployment, log)

	return nil
}

func (s *Service) execute(ctx context.Context, app *models.Application, deployment *models.Deployment, log *logWriter) error {
	cfg := facades.Config()
	network := cfg.GetString("ebox.docker.network", proxy.DefaultNetwork)

	if err := ctx.Err(); err != nil {
		return err
	}

	started := time.Now()
	deployment.StartedAt = &started
	s.setStatus(deployment, models.DeploymentStatusBuilding, log)
	s.markApplication(app, models.ApplicationStatusDeploying)

	rollingBack := deployment.Trigger == models.DeploymentTriggerRollback && deployment.ImageTag != ""
	composeApp := app.IsCompose()
	repositoryDir := git.Default().Dir("apps", app.UUID)

	if rollingBack && !composeApp {
		log.Line("Rolling back to image " + deployment.ImageTag)
	} else {
		if app.Kind == models.ApplicationKindGit {
			info, err := sources.Default().CloneInfo(ctx, app)
			if err != nil {
				return err
			}

			commit, err := git.Default().Sync(ctx, git.SyncOptions{
				Directory:   repositoryDir,
				URL:         info.URL,
				Credentials: info.Credentials,
				Branch:      app.Branch,
				Commit:      deployment.Commit,
				Sink:        log.Line,
			})
			if err != nil {
				return err
			}

			deployment.Commit = commit.SHA
			deployment.CommitMessage = commit.Message
			log.Line("Checked out " + commit.ShortSHA + ": " + commit.Message)
		}

		buildArgs, err := envvars.Default().Values(models.ResourceTypeApplication, app.ID, true)
		if err != nil {
			return err
		}

		if composeApp {
			log.Line("Compose application: images are built by docker compose")
		} else {
			tag := fmt.Sprintf("ebox/%s:%s", app.Slug(), deployment.ShortTag())
			log.Line("Building image " + tag)

			if err := build.Image(ctx, build.Request{
				App:       app,
				Directory: repositoryDir,
				Tag:       tag,
				BuildArgs: buildArgs,
				Env:       os.Environ(),
				Sink:      log.Line,
			}); err != nil {
				return err
			}

			deployment.ImageTag = tag
		}
	}

	s.setStatus(deployment, models.DeploymentStatusDeploying, log)

	if composeApp {
		if err := s.deployCompose(ctx, app, deployment, log, repositoryDir, network); err != nil {
			return err
		}
	} else if err := s.deployContainer(ctx, app, deployment, log, network); err != nil {
		return err
	}

	if !composeApp {
		s.pruneImages(context.WithoutCancel(ctx), app, log)
	}
	s.markApplication(app, models.ApplicationStatusIdle)

	return nil
}

func (s *Service) deployContainer(ctx context.Context, app *models.Application, deployment *models.Deployment, log *logWriter, network string) error {
	if err := docker.Default().EnsureNetwork(ctx, network); err != nil {
		return err
	}

	resourceDomains, err := domains.Default().List(models.ResourceTypeApplication, app.ID)
	if err != nil {
		return err
	}

	values, err := envvars.Default().Values(models.ResourceTypeApplication, app.ID, false)
	if err != nil {
		return err
	}

	envFile := filepath.Join(facades.Config().GetString("ebox.apps_dir"), app.UUID, "env")
	if err := envvars.Default().WriteEnvFile(envFile, values); err != nil {
		return err
	}

	labels := proxy.AppLabels(app, resourceDomains, network)
	labels[docker.LabelDeploymentID] = deployment.UUID

	spec := docker.Spec{
		Name:         app.ContainerName(deployment.ShortTag()),
		Image:        deployment.ImageTag,
		Env:          EnvLines(values),
		Labels:       labels,
		Network:      network,
		Alias:        "ebox-" + app.Slug(),
		Restart:      "unless-stopped",
		StopTimeout:  15 * time.Second,
		ExposedPorts: []string{strconv.Itoa(app.ExposePort()) + "/tcp"},
	}
	if app.Config.Limits != nil {
		spec.MemoryMB = app.Config.Limits.MemoryMB
		spec.CPUs = app.Config.Limits.CPUs
	}

	log.Line("Starting container " + spec.Name)

	containerID, err := docker.Default().Create(ctx, spec)
	if err != nil {
		return err
	}

	deployment.ContainerID = containerID
	deployment.ContainerName = spec.Name
	s.save(deployment)

	inspect, err := docker.Default().WaitRunning(ctx, containerID, containerStartTimeout)
	if err != nil {
		s.dumpContainerLogs(ctx, containerID, log)
		_ = docker.Default().Remove(context.WithoutCancel(ctx), containerID, true)

		return err
	}

	log.Line("Container is running")

	if err := docker.Default().WaitHealthy(ctx, containerID, healthTimeout); err != nil {
		s.dumpContainerLogs(ctx, containerID, log)
		_ = docker.Default().Remove(context.WithoutCancel(ctx), containerID, true)

		return err
	}

	if inspect.State != nil && inspect.State.Health != nil && inspect.State.Health.Status == "healthy" {
		log.Line("Healthcheck passed")
	}

	s.probe(ctx, app, containerID, log, network)
	s.removePreviousContainers(context.WithoutCancel(ctx), app, containerID, log)

	return nil
}

func (s *Service) probe(ctx context.Context, app *models.Application, containerID string, log *logWriter, network string) {
	if ctx.Err() != nil {
		return
	}

	inspect, err := docker.Default().Inspect(ctx, containerID)
	if err != nil {
		return
	}

	ip := docker.Default().ContainerIP(inspect, network)
	if ip == "" {
		log.Line("Skipping HTTP probe: the container has no address on the " + network + " network")

		return
	}

	target := fmt.Sprintf("http://%s:%d/", ip, app.ExposePort())
	client := &http.Client{Timeout: time.Second}

	for attempt := 1; attempt <= probeAttempts; attempt++ {
		response, err := client.Get(target)
		if err == nil {
			_ = response.Body.Close()
			if response.StatusCode < 500 {
				log.Line(fmt.Sprintf("HTTP probe succeeded (%d) on %s", response.StatusCode, target))

				return
			}
		}

		current, inspectErr := docker.Default().Inspect(ctx, containerID)
		if inspectErr == nil && current.State != nil && !current.State.Running {
			log.Line("Container stopped while probing")

			return
		}

		select {
		case <-ctx.Done():
			return
		case <-time.After(500 * time.Millisecond):
		}
	}

	log.Line("Warning: " + target + " did not answer the HTTP probe from the panel. The container is running and routing may still work (container networks are not always reachable from the host, for example on Docker Desktop).")
}

func (s *Service) dumpContainerLogs(ctx context.Context, containerID string, log *logWriter) {
	output, err := docker.Default().Logs(context.WithoutCancel(ctx), containerID, 80)
	if err != nil || strings.TrimSpace(output) == "" {
		return
	}

	log.Line("--- container output ---")
	for _, line := range strings.Split(strings.TrimRight(output, "\n"), "\n") {
		log.Line(line)
	}
}

func (s *Service) removePreviousContainers(ctx context.Context, app *models.Application, keepID string, log *logWriter) {
	containers, err := docker.Default().FindByResource(ctx, docker.ResourceApplication, app.UUID)
	if err != nil {
		return
	}

	for _, item := range containers {
		if item.ID == keepID {
			continue
		}

		log.Line("Stopping previous container " + item.Name)
		if err := docker.Default().Stop(ctx, item.ID, 15*time.Second); err != nil {
			_ = docker.Default().Remove(ctx, item.ID, true)
			continue
		}
		_ = docker.Default().Remove(ctx, item.ID, false)
	}
}

func (s *Service) pruneImages(ctx context.Context, app *models.Application, log *logWriter) {
	keep := facades.Config().GetInt("ebox.deploy.retain_images", 5)
	if keep <= 0 {
		return
	}

	images, err := docker.Default().ListImages(ctx, "ebox/"+app.Slug())
	if err != nil || len(images) <= keep {
		return
	}

	for _, item := range images[keep:] {
		for _, tag := range item.Tags {
			if err := docker.Default().RemoveImage(ctx, tag, false); err == nil {
				log.Line("Pruned old image " + tag)
			}
		}
	}
}

func (s *Service) succeed(deployment *models.Deployment, log *logWriter) {
	finished := time.Now()
	deployment.FinishedAt = &finished
	if deployment.StartedAt != nil {
		deployment.DurationMS = finished.Sub(*deployment.StartedAt).Milliseconds()
	}

	log.Line(fmt.Sprintf("Deployment succeeded in %s", (time.Duration(deployment.DurationMS) * time.Millisecond).Round(time.Second)))
	s.setStatus(deployment, models.DeploymentStatusSuccess, log)

	s.announce(deployment, models.EventDeploymentSuccess, "Deployment succeeded: "+deployment.ContainerName,
		fmt.Sprintf("Deployment %s finished in %s.", deployment.ShortTag(), (time.Duration(deployment.DurationMS)*time.Millisecond).Round(time.Second)))

	s.retainDeployments(deployment.ApplicationID)
}

func (s *Service) fail(deployment *models.Deployment, log *logWriter, cause error) {
	finished := time.Now()
	deployment.FinishedAt = &finished
	deployment.Error = cause.Error()
	if deployment.StartedAt != nil {
		deployment.DurationMS = finished.Sub(*deployment.StartedAt).Milliseconds()
	}

	if log != nil {
		log.Line("ERROR: " + cause.Error())
		log.Line("Deployment failed. The previous container keeps serving traffic.")
	}

	s.setStatus(deployment, models.DeploymentStatusFailed, log)

	var name string
	if loaded, err := s.applicationOf(deployment); err == nil {
		name = loaded.Name
	}

	s.announce(deployment, models.EventDeploymentFailed, "Deployment failed: "+name, cause.Error())

	var app models.Application
	if err := facades.Orm().Query().Where("id", deployment.ApplicationID).FirstOrFail(&app); err == nil {
		s.markApplication(&app, models.ApplicationStatusFailed)
	}
}

func (s *Service) cancelDeployment(deployment *models.Deployment, log *logWriter) {
	finished := time.Now()
	deployment.FinishedAt = &finished
	deployment.Error = "cancelled"
	if deployment.StartedAt != nil {
		deployment.DurationMS = finished.Sub(*deployment.StartedAt).Milliseconds()
	}

	if log != nil {
		log.Line("Deployment cancelled")
	}

	s.setStatus(deployment, models.DeploymentStatusCancelled, log)
}

// announce publishes a notification about a deployment.
func (s *Service) announce(deployment *models.Deployment, event, title, body string) {
	notifications.Default().Notify(event, title, body, map[string]any{
		"application_id": deployment.ApplicationID,
		"deployment":     deployment.UUID,
		"trigger":        deployment.Trigger,
		"commit":         deployment.Commit,
		"commit_message": deployment.CommitMessage,
		"duration_ms":    deployment.DurationMS,
	})
}

// applicationOf loads the application a deployment belongs to.
func (s *Service) applicationOf(deployment *models.Deployment) (*models.Application, error) {
	var app models.Application
	if err := facades.Orm().Query().Where("id", deployment.ApplicationID).FirstOrFail(&app); err != nil {
		return nil, err
	}

	return &app, nil
}

func (s *Service) setStatus(deployment *models.Deployment, status string, log *logWriter) {
	deployment.Status = status

	updates := map[string]any{
		"status":          status,
		"commit":          deployment.Commit,
		"commit_message":  deployment.CommitMessage,
		"image_tag":       deployment.ImageTag,
		"compose_project": deployment.ComposeProject,
		"container_id":    deployment.ContainerID,
		"container_name":  deployment.ContainerName,
		"error":           deployment.Error,
		"started_at":      deployment.StartedAt,
		"finished_at":     deployment.FinishedAt,
		"duration_ms":     deployment.DurationMS,
	}

	_, _ = facades.Orm().Query().
		Model(&models.Deployment{}).
		Where("id", deployment.ID).
		Update(updates)

	_ = log
	s.publish(deployment, status)
}

func (s *Service) save(deployment *models.Deployment) {
	_, _ = facades.Orm().Query().
		Model(&models.Deployment{}).
		Where("id", deployment.ID).
		Update(map[string]any{
			"commit":          deployment.Commit,
			"commit_message":  deployment.CommitMessage,
			"image_tag":       deployment.ImageTag,
			"compose_project": deployment.ComposeProject,
			"container_id":    deployment.ContainerID,
			"container_name":  deployment.ContainerName,
		})
}

func (s *Service) publish(deployment *models.Deployment, status string) {
	realtime.Default().Publish(realtime.Event{
		Topic: "deployment." + deployment.UUID,
		Type:  "status",
		Data: map[string]any{
			"deployment_uuid": deployment.UUID,
			"application_id":  deployment.ApplicationID,
			"status":          status,
		},
	})
}

func (s *Service) markApplication(app *models.Application, status string) {
	app.Status = status
	_, _ = facades.Orm().Query().
		Model(&models.Application{}).
		Where("id", app.ID).
		Update("status", status)

	realtime.Default().Publish(realtime.Event{
		Topic: "application." + app.UUID,
		Type:  "status",
		Data:  map[string]any{"application_uuid": app.UUID, "status": status},
	})
}

// retainDeployments keeps the most recent deployments per application.
func (s *Service) retainDeployments(applicationID uint64) {
	keep := facades.Config().GetInt("ebox.deploy.retain_deployments", 50)
	if keep <= 0 {
		return
	}

	var deployments []models.Deployment
	if err := facades.Orm().Query().
		Where("application_id", applicationID).
		Order("id desc").
		Find(&deployments); err != nil {
		return
	}

	if len(deployments) <= keep {
		return
	}

	for _, old := range deployments[keep:] {
		if old.Active() {
			continue
		}

		_, _ = facades.Orm().Query().Where("id", old.ID).Delete(&models.Deployment{})
	}
}

// EnvLines renders env vars as the KEY=value list Docker expects.
func EnvLines(values map[string]string) []string {
	lines := make([]string, 0, len(values))
	for _, pair := range envvars.Sorted(values) {
		lines = append(lines, pair.Key+"="+pair.Value)
	}

	return lines
}

// Snapshot returns deployment log lines after a sequence number.
func (s *Service) Snapshot(deployment *models.Deployment, since int) (int, []string) {
	lines := []string{}
	if strings.TrimSpace(deployment.Log) != "" {
		lines = strings.Split(strings.TrimRight(deployment.Log, "\n"), "\n")
	}

	if since < 0 {
		since = 0
	}
	if since >= len(lines) {
		return len(lines), nil
	}

	remaining := make([]string, len(lines)-since)
	copy(remaining, lines[since:])

	return len(lines), remaining
}
