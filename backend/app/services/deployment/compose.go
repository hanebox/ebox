package deployment

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"ebox/app/facades"
	"ebox/app/models"
	"ebox/app/services/compose"
	"ebox/app/services/docker"
	"ebox/app/services/domains"
	"ebox/app/services/envvars"
	"ebox/app/services/git"
	"ebox/app/services/proxy"
)

const (
	composeOverrideName = ".ebox-override.yml"
	composeEnvName      = ".ebox.env"
)

var composeCandidates = []string{"docker-compose.yml", "docker-compose.yaml", "compose.yml", "compose.yaml"}

// composeFile locates the compose file of an application inside its checkout.
func composeFile(app *models.Application, directory string) (string, error) {
	base := directory
	if app.BuildContextDirectory() != "." {
		base = filepath.Join(directory, app.BuildContextDirectory())
	}

	candidates := composeCandidates
	if path := strings.TrimSpace(app.Config.ComposePath); path != "" {
		candidates = []string{path}
	}

	for _, candidate := range candidates {
		path := filepath.Join(base, candidate)
		if _, err := os.Stat(path); err == nil {
			return path, nil
		}
	}

	return "", errors.New("deployment: no compose file was found (looked for " + strings.Join(candidates, ", ") + ")")
}

// deployCompose builds and starts the application's compose project.
func (s *Service) deployCompose(ctx context.Context, app *models.Application, deployment *models.Deployment, log *logWriter, directory, network string) error {
	if err := docker.Default().EnsureNetwork(ctx, network); err != nil {
		return err
	}

	path, err := composeFile(app, directory)
	if err != nil {
		return err
	}

	content, err := os.ReadFile(path)
	if err != nil {
		return err
	}

	document, err := compose.Parse(string(content))
	if err != nil {
		return err
	}

	log.Line("Compose file: " + strings.TrimPrefix(path, directory+string(os.PathSeparator)))
	for _, info := range document.Info() {
		description := info.Name
		if info.Image != "" {
			description += " (" + info.Image + ")"
		} else if info.Built {
			description += " (build)"
		}
		if len(info.Ports) > 0 {
			description += " ports " + strings.Join(info.Ports, ", ")
		}

		log.Line("  service " + description)
	}

	entryService, entryPort, err := composeEntry(app, document)
	if err != nil {
		return err
	}

	log.Line(fmt.Sprintf("Ingress service %s on port %d", entryService, entryPort))

	resourceDomains, err := domains.Default().List(models.ResourceTypeApplication, app.ID)
	if err != nil {
		return err
	}

	values, err := envvars.Default().Values(models.ResourceTypeApplication, app.ID, false)
	if err != nil {
		return err
	}

	labels := proxy.AppLabels(app, resourceDomains, network)
	labels[docker.LabelDeploymentID] = deployment.UUID

	override, err := compose.Render(document, compose.Override{
		Project:      composeProjectName(app),
		Network:      network,
		Labels:       labels,
		Environment:  values,
		EntryService: entryService,
		EntryPort:    entryPort,
		EntryLabels:  map[string]string{},
		EntryEnv:     values,
	})
	if err != nil {
		return err
	}

	overridePath := filepath.Join(directory, composeOverrideName)
	if err := os.WriteFile(overridePath, []byte(override), 0o600); err != nil {
		return err
	}

	envPath := filepath.Join(directory, composeEnvName)
	if err := envvars.Default().WriteEnvFile(envPath, values); err != nil {
		return err
	}

	project := compose.Project{
		Name:    composeProjectName(app),
		Dir:     directory,
		Files:   []string{path, overridePath},
		EnvFile: envPath,
	}

	deployment.ComposeProject = project.Name
	s.save(deployment)

	runner := compose.NewRunner()
	if err := runner.Pull(ctx, project, log.Line); err != nil {
		log.Line("Warning: " + err.Error())
	}

	log.Line("Running docker compose up")
	if err := runner.Up(ctx, project, true, log.Line); err != nil {
		return err
	}

	running, err := s.composeServices(ctx, project.Name)
	if err != nil {
		return err
	}

	if running == 0 {
		s.dumpComposeLogs(ctx, project.Name, log)

		return errors.New("deployment: no compose service is running")
	}

	log.Line(fmt.Sprintf("%d compose service(s) running", running))

	s.probeCompose(ctx, project.Name, entryService, entryPort, log)

	return nil
}

// composeEntry resolves which compose service fronts the application.
func composeEntry(app *models.Application, document *compose.Document) (string, int, error) {
	if configured := strings.TrimSpace(app.Config.ComposeService); configured != "" {
		definition, ok := document.Services[configured]
		if !ok {
			return "", 0, fmt.Errorf("deployment: the compose file has no service %q", configured)
		}

		return configured, composeEntryPort(app, definition), nil
	}

	candidates := document.EntryCandidates()
	if len(candidates) == 0 {
		return "", 0, errors.New("deployment: the compose file has no service")
	}

	definition := document.Services[candidates[0].Name]

	return candidates[0].Name, composeEntryPort(app, definition), nil
}

func composeEntryPort(app *models.Application, definition compose.Service) int {
	if app.Config.ExposePort > 0 {
		return app.Config.ExposePort
	}

	if port := definition.ContainerPort(); port > 0 {
		return port
	}

	return 80
}

// ComposeProjectName is the compose project for an application.
func ComposeProjectName(app *models.Application) string {
	return "ebox-" + app.Slug()
}

func composeProjectName(app *models.Application) string {
	return ComposeProjectName(app)
}

// composeServices counts running containers of the project.
func (s *Service) composeServices(ctx context.Context, project string) (int, error) {
	containers, err := docker.Default().List(ctx, docker.ComposeFilter(project))
	if err != nil {
		return 0, err
	}

	running := 0
	for _, container := range containers {
		service := container.Labels["com.docker.compose.service"]

		if container.State == "running" {
			running++
			continue
		}

		failure := container.State
		if container.Status != "" {
			failure += ": " + container.Status
		}

		return running, fmt.Errorf("deployment: compose service %s is not running (%s)", service, failure)
	}

	return running, nil
}

func (s *Service) probeCompose(ctx context.Context, project, service string, port int, log *logWriter) {
	containers, err := docker.Default().List(ctx, docker.ComposeFilter(project))
	if err != nil {
		return
	}

	for _, container := range containers {
		if container.Labels["com.docker.compose.service"] != service || container.State != "running" {
			continue
		}

		s.probe(ctx, &models.Application{Config: models.ApplicationConfig{ExposePort: port}}, container.ID, log, facades.Config().GetString("ebox.docker.network", proxy.DefaultNetwork))

		return
	}
}

func (s *Service) dumpComposeLogs(ctx context.Context, project string, log *logWriter) {
	containers, err := docker.Default().List(context.WithoutCancel(ctx), docker.ComposeFilter(project))
	if err != nil {
		return
	}

	for _, container := range containers {
		output, err := docker.Default().Logs(context.WithoutCancel(ctx), container.ID, 40)
		if err != nil || strings.TrimSpace(output) == "" {
			continue
		}

		log.Line("--- last logs of " + container.Labels["com.docker.compose.service"] + " ---")
		for _, line := range strings.Split(strings.TrimRight(output, "\n"), "\n") {
			log.Line("  " + line)
		}
	}
}

// TeardownCompose removes every container of a compose application.
func (s *Service) TeardownCompose(ctx context.Context, app *models.Application) {
	containers, err := docker.Default().List(ctx, docker.ComposeFilter(composeProjectName(app)))
	if err != nil {
		return
	}

	for _, container := range containers {
		_ = docker.Default().Remove(context.WithoutCancel(ctx), container.ID, true)
	}
}

// ComposeInfo lists the services of a compose application and the detected
// ingress service.
func ComposeInfo(app *models.Application, directory string) ([]compose.Info, string, error) {
	path, err := composeFile(app, directory)
	if err != nil {
		return nil, "", err
	}

	content, err := os.ReadFile(path)
	if err != nil {
		return nil, "", err
	}

	document, err := compose.Parse(string(content))
	if err != nil {
		return nil, "", err
	}

	entry, _, err := composeEntry(app, document)
	if err != nil {
		return nil, "", err
	}

	return document.Info(), entry, nil
}

// ComposeAction starts, stops or restarts a compose application project.
func (s *Service) ComposeAction(ctx context.Context, app *models.Application, action string) error {
	project, err := s.composeProjectFiles(app)
	if err != nil {
		return err
	}

	runner := compose.NewRunner()

	switch action {
	case "start":
		return runner.Start(ctx, project, nil)
	case "stop":
		return runner.Stop(ctx, project, nil)
	case "restart":
		return runner.Restart(ctx, project, nil)
	}

	return errors.New("deployment: unsupported compose action " + action)
}

// composeProjectFiles rebuilds the project handle from the last checkout.
func (s *Service) composeProjectFiles(app *models.Application) (compose.Project, error) {
	directory := git.Default().Dir("apps", app.UUID)

	path, err := composeFile(app, directory)
	if err != nil {
		return compose.Project{}, err
	}

	overridePath := filepath.Join(directory, composeOverrideName)
	if _, err := os.Stat(overridePath); err != nil {
		return compose.Project{}, errors.New("deployment: this stack has no generated configuration yet, deploy it first")
	}

	return compose.Project{
		Name:    composeProjectName(app),
		Dir:     directory,
		Files:   []string{path, overridePath},
		EnvFile: filepath.Join(directory, composeEnvName),
	}, nil
}
