package controllers

import (
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	contractshttp "github.com/goravel/framework/contracts/http"

	"ebox/app/facades"
	"ebox/app/models"
	"ebox/app/services/applications"
	"ebox/app/services/deployment"
	"ebox/app/services/docker"
	"ebox/app/services/domains"
	"ebox/app/services/envvars"
	"ebox/app/services/git"
	"ebox/app/services/sources"
	"ebox/app/support/httpx"
)

type ApplicationController struct{}

func NewApplicationController() *ApplicationController {
	return &ApplicationController{}
}

type applicationResponse struct {
	models.Application
	Runtime   applications.Status `json:"runtime"`
	DeployURL string              `json:"deploy_url,omitempty"`
}

func (c *ApplicationController) Index(ctx contractshttp.Context) contractshttp.Response {
	list, err := applications.Default().List(ctx.Request().Input("project"))
	if err != nil {
		return httpx.Err(ctx, http.StatusInternalServerError, "Failed to load applications.")
	}

	containers, _ := docker.Default().List(ctx.Request().Origin().Context(), docker.ResourceFilter(docker.ResourceApplication, ""))
	byResource := map[string]docker.Container{}
	for _, container := range containers {
		if _, exists := byResource[container.Labels[docker.LabelResourceID]]; !exists {
			byResource[container.Labels[docker.LabelResourceID]] = container
		}
	}

	responses := make([]applicationResponse, 0, len(list))
	for _, app := range list {
		runtime := applications.Status{State: "missing"}
		if container, ok := byResource[app.UUID]; ok {
			runtime.State = container.State
			runtime.Health = container.Health
			runtime.ContainerID = container.ID
			runtime.ContainerName = container.Name
			runtime.Image = container.Image
		}

		responses = append(responses, applicationResponse{Application: app, Runtime: runtime})
	}

	return httpx.Ok(ctx, responses)
}

func (c *ApplicationController) Store(ctx contractshttp.Context) contractshttp.Response {
	input, failure := applicationInput(ctx)
	if failure != nil {
		return failure
	}

	app, err := applications.Default().Create(input)
	if err != nil {
		return httpx.Err(ctx, http.StatusUnprocessableEntity, err.Error())
	}

	applicationRequests := ctx.Request().InputArray("domains")
	for index, host := range applicationRequests {
		if strings.TrimSpace(host) == "" {
			continue
		}
		if _, err := domains.Default().Add(models.ResourceTypeApplication, app.ID, host, true, index == 0); err != nil {
			return httpx.Err(ctx, http.StatusUnprocessableEntity, err.Error())
		}
	}

	envRequests := ctx.Request().InputMapArray("env")
	for _, item := range envRequests {
		key, _ := item["key"].(string)
		value, _ := item["value"].(string)
		buildTime, _ := item["is_build_time"].(bool)
		if strings.TrimSpace(key) == "" {
			continue
		}
		if _, err := envvars.Default().Set(models.ResourceTypeApplication, app.ID, key, value, buildTime); err != nil {
			return httpx.Err(ctx, http.StatusUnprocessableEntity, err.Error())
		}
	}

	if ctx.Request().InputBool("deploy", false) {
		if _, err := deployment.Default().Enqueue(app, models.DeploymentTriggerManual, ""); err != nil {
			return httpx.Err(ctx, http.StatusInternalServerError, "The application was created but the deployment could not be queued.")
		}
	}

	created, err := applications.Default().Find(app.UUID)
	if err != nil {
		return httpx.Err(ctx, http.StatusInternalServerError, "Failed to load the application.")
	}

	return httpx.Created(ctx, applicationResponse{Application: *created})
}

func (c *ApplicationController) Show(ctx contractshttp.Context) contractshttp.Response {
	app, err := applications.Default().Find(ctx.Request().Input("uuid"))
	if err != nil {
		return httpx.Err(ctx, http.StatusNotFound, "Application not found.")
	}

	resourceDomains, err := domains.Default().List(models.ResourceTypeApplication, app.ID)
	if err != nil {
		return httpx.Err(ctx, http.StatusInternalServerError, "Failed to load the domains.")
	}

	runtime := applications.Default().Status(ctx.Request().Origin().Context(), app)

	response := struct {
		applicationResponse
		Domains []models.Domain `json:"domains"`
	}{
		applicationResponse: applicationResponse{
			Application: *app,
			Runtime:     runtime,
			DeployURL:   "/api/v1/deploy/" + app.DeployToken,
		},
		Domains: resourceDomains,
	}

	return httpx.Ok(ctx, response)
}

func (c *ApplicationController) Update(ctx contractshttp.Context) contractshttp.Response {
	app, err := applications.Default().Find(ctx.Request().Input("uuid"))
	if err != nil {
		return httpx.Err(ctx, http.StatusNotFound, "Application not found.")
	}

	input, failure := applicationInput(ctx)
	if failure != nil {
		return failure
	}
	input.ProjectUUID = app.Project.UUID

	if err := applications.Default().Update(app, input); err != nil {
		return httpx.Err(ctx, http.StatusUnprocessableEntity, err.Error())
	}

	return httpx.Ok(ctx, app)
}

func (c *ApplicationController) Destroy(ctx contractshttp.Context) contractshttp.Response {
	app, err := applications.Default().Find(ctx.Request().Input("uuid"))
	if err != nil {
		return httpx.Err(ctx, http.StatusNotFound, "Application not found.")
	}

	requestCtx := ctx.Request().Origin().Context()

	if app.IsCompose() {
		deployment.Default().TeardownCompose(requestCtx, app)
	}

	containers, _ := docker.Default().FindByResource(requestCtx, docker.ResourceApplication, app.UUID)
	for _, container := range containers {
		_ = docker.Default().Stop(requestCtx, container.ID, 10*time.Second)
		_ = docker.Default().Remove(requestCtx, container.ID, true)
	}

	if err := applications.Default().Delete(app); err != nil {
		return httpx.Err(ctx, http.StatusInternalServerError, "Failed to delete the application.")
	}

	_, _ = facades.Orm().Query().Where("application_id", app.ID).Delete(&models.Deployment{})
	_, _ = facades.Orm().Query().Where("resource_type", models.ResourceTypeApplication).Where("resource_id", app.ID).Delete(&models.EnvVar{})
	_, _ = facades.Orm().Query().Where("resource_type", models.ResourceTypeApplication).Where("resource_id", app.ID).Delete(&models.Domain{})

	return httpx.Ok(ctx, contractshttp.Json{"message": "Application deleted."})
}

func (c *ApplicationController) Deploy(ctx contractshttp.Context) contractshttp.Response {
	app, err := applications.Default().Find(ctx.Request().Input("uuid"))
	if err != nil {
		return httpx.Err(ctx, http.StatusNotFound, "Application not found.")
	}

	deploymentRecord, err := deployment.Default().Enqueue(app, models.DeploymentTriggerManual, ctx.Request().Input("commit"))
	if err != nil {
		return httpx.Err(ctx, http.StatusInternalServerError, "Failed to queue the deployment.")
	}

	return httpx.Created(ctx, deploymentRecord)
}

func (c *ApplicationController) Rollback(ctx contractshttp.Context) contractshttp.Response {
	app, err := applications.Default().Find(ctx.Request().Input("uuid"))
	if err != nil {
		return httpx.Err(ctx, http.StatusNotFound, "Application not found.")
	}

	var source models.Deployment
	if err := facades.Orm().Query().Where("uuid", ctx.Request().Input("deployment")).FirstOrFail(&source); err != nil {
		return httpx.Err(ctx, http.StatusNotFound, "Deployment not found.")
	}

	if source.ApplicationID != app.ID {
		return httpx.Err(ctx, http.StatusUnprocessableEntity, "The deployment belongs to another application.")
	}

	deploymentRecord, err := deployment.Default().EnqueueRollback(app, &source)
	if err != nil {
		return httpx.Err(ctx, http.StatusUnprocessableEntity, err.Error())
	}

	return httpx.Created(ctx, deploymentRecord)
}

// ComposePreview lists the services of a compose application so the UI can pick
// the ingress service without guessing.
func (c *ApplicationController) ComposePreview(ctx contractshttp.Context) contractshttp.Response {
	app, err := applications.Default().Find(ctx.Request().Input("uuid"))
	if err != nil {
		return httpx.Err(ctx, http.StatusNotFound, "Application not found.")
	}

	if !app.IsCompose() {
		return httpx.Err(ctx, http.StatusUnprocessableEntity, "This application does not use the compose build pack.")
	}

	requestCtx := ctx.Request().Origin().Context()
	directory := git.Default().Dir("apps", app.UUID)

	if !composeCheckoutExists(app, directory) {
		info, err := sources.Default().CloneInfo(requestCtx, app)
		if err != nil {
			return httpx.Err(ctx, http.StatusUnprocessableEntity, err.Error())
		}

		if _, err := git.Default().Sync(requestCtx, git.SyncOptions{
			Directory:   directory,
			URL:         info.URL,
			Credentials: info.Credentials,
			Branch:      app.Branch,
			Sink:        nil,
		}); err != nil {
			return httpx.Err(ctx, http.StatusBadGateway, err.Error())
		}
	}

	services, entry, err := deployment.ComposeInfo(app, directory)
	if err != nil {
		return httpx.Err(ctx, http.StatusUnprocessableEntity, err.Error())
	}

	return httpx.Ok(ctx, contractshttp.Json{
		"services":      services,
		"entry_service": entry,
	})
}

func (c *ApplicationController) Start(ctx contractshttp.Context) contractshttp.Response {
	return c.containerAction(ctx, "start")
}

func (c *ApplicationController) Stop(ctx contractshttp.Context) contractshttp.Response {
	return c.containerAction(ctx, "stop")
}

func (c *ApplicationController) Restart(ctx contractshttp.Context) contractshttp.Response {
	return c.containerAction(ctx, "restart")
}

func (c *ApplicationController) Logs(ctx contractshttp.Context) contractshttp.Response {
	app, err := applications.Default().Find(ctx.Request().Input("uuid"))
	if err != nil {
		return httpx.Err(ctx, http.StatusNotFound, "Application not found.")
	}

	requestCtx := ctx.Request().Origin().Context()
	containers, err := docker.Default().FindByResource(requestCtx, docker.ResourceApplication, app.UUID)
	if err != nil {
		return httpx.Err(ctx, http.StatusBadGateway, err.Error())
	}
	if len(containers) == 0 {
		return httpx.Ok(ctx, contractshttp.Json{"lines": []string{}, "container": nil})
	}

	tail := ctx.Request().InputInt("tail", 300)
	output, err := docker.Default().Logs(requestCtx, containers[0].ID, tail)
	if err != nil {
		return httpx.Err(ctx, http.StatusBadGateway, err.Error())
	}

	lines := []string{}
	if strings.TrimSpace(output) != "" {
		lines = strings.Split(strings.TrimRight(output, "\n"), "\n")
	}

	return httpx.Ok(ctx, contractshttp.Json{
		"lines":     lines,
		"container": containers[0],
	})
}

func (c *ApplicationController) RegenerateToken(ctx contractshttp.Context) contractshttp.Response {
	app, err := applications.Default().Find(ctx.Request().Input("uuid"))
	if err != nil {
		return httpx.Err(ctx, http.StatusNotFound, "Application not found.")
	}

	token, err := applications.Default().RegenerateDeployToken(app)
	if err != nil {
		return httpx.Err(ctx, http.StatusInternalServerError, "Failed to regenerate the deploy token.")
	}

	return httpx.Ok(ctx, contractshttp.Json{
		"deploy_token": token,
		"deploy_url":   "/api/v1/deploy/" + token,
	})
}

func (c *ApplicationController) containerAction(ctx contractshttp.Context, action string) contractshttp.Response {
	app, err := applications.Default().Find(ctx.Request().Input("uuid"))
	if err != nil {
		return httpx.Err(ctx, http.StatusNotFound, "Application not found.")
	}

	requestCtx := ctx.Request().Origin().Context()

	// Compose applications are controlled as a project, not as one container.
	if app.IsCompose() {
		composeContainers, err := docker.Default().List(requestCtx, docker.ComposeFilter(deployment.ComposeProjectName(app)))
		if err != nil {
			return httpx.Err(ctx, http.StatusBadGateway, docker.FriendlyError(err))
		}
		if len(composeContainers) == 0 {
			return httpx.Err(ctx, http.StatusNotFound, "This application has no running stack. Deploy it first.")
		}

		if actionErr := deployment.Default().ComposeAction(requestCtx, app, action); actionErr != nil {
			return httpx.Err(ctx, http.StatusBadGateway, docker.FriendlyError(actionErr))
		}

		return httpx.Ok(ctx, contractshttp.Json{"message": "Stack " + action + "ed."})
	}

	containers, err := docker.Default().FindByResource(requestCtx, docker.ResourceApplication, app.UUID)
	if err != nil {
		return httpx.Err(ctx, http.StatusBadGateway, err.Error())
	}
	if len(containers) == 0 {
		return httpx.Err(ctx, http.StatusNotFound, "No container found for this application. Deploy it first.")
	}

	container := containers[0]
	var actionErr error

	switch action {
	case "start":
		actionErr = docker.Default().Start(requestCtx, container.ID)
	case "stop":
		actionErr = docker.Default().Stop(requestCtx, container.ID, 15*time.Second)
	case "restart":
		actionErr = docker.Default().Restart(requestCtx, container.ID, 15*time.Second)
	}

	if actionErr != nil {
		return httpx.Err(ctx, http.StatusBadGateway, docker.FriendlyError(actionErr))
	}

	messages := map[string]string{
		"start":   "Container started.",
		"stop":    "Container stopped.",
		"restart": "Container restarted.",
	}

	return httpx.Ok(ctx, contractshttp.Json{"message": messages[action]})
}

func applicationInput(ctx contractshttp.Context) (applications.Input, contractshttp.Response) {
	input := applications.Input{
		ProjectUUID: ctx.Request().Input("project"),
		Name:        strings.TrimSpace(ctx.Request().Input("name")),
		SourceUUID:  ctx.Request().Input("source"),
		Repository:  strings.TrimSpace(ctx.Request().Input("repository")),
		Branch:      strings.TrimSpace(ctx.Request().Input("branch")),
		BuildPack:   ctx.Request().Input("build_pack"),
		AutoDeploy:  ctx.Request().InputBool("auto_deploy", false),
	}

	configMap := ctx.Request().InputMap("config")
	if configMap != nil {
		if value, ok := configMap["base_directory"].(string); ok {
			input.Config.BaseDirectory = value
		}
		if value, ok := configMap["dockerfile_path"].(string); ok {
			input.Config.DockerfilePath = value
		}
		if value, ok := configMap["compose_path"].(string); ok {
			input.Config.ComposePath = value
		}
		if value, ok := configMap["compose_service"].(string); ok {
			input.Config.ComposeService = value
		}
		if value, ok := configMap["build_command"].(string); ok {
			input.Config.BuildCommand = value
		}
		if value, ok := configMap["publish_directory"].(string); ok {
			input.Config.PublishDirectory = value
		}
		if value, ok := configMap["image"].(string); ok {
			input.Config.Image = value
		}
		if value, ok := configMap["expose_port"].(float64); ok {
			input.Config.ExposePort = int(value)
		}
		if limits, ok := configMap["limits"].(map[string]any); ok {
			resourceLimits := &models.ResourceLimits{}
			if value, ok := limits["memory_mb"].(float64); ok {
				resourceLimits.MemoryMB = int(value)
			}
			if value, ok := limits["cpus"].(float64); ok {
				resourceLimits.CPUs = value
			}
			input.Config.Limits = resourceLimits
		}
	}

	if input.Name == "" {
		return input, httpx.ValidationErrors(ctx, validationErrors{"name": {"The name field is required."}})
	}

	if input.BuildPack == models.BuildPackImage && input.Config.Image == "" {
		return input, httpx.ValidationErrors(ctx, validationErrors{"config.image": {"A docker image is required."}})
	}

	if input.BuildPack != models.BuildPackImage && input.SourceUUID == "" {
		return input, httpx.ValidationErrors(ctx, validationErrors{"source": {"A source is required."}})
	}

	if input.ProjectUUID == "" {
		return input, httpx.ValidationErrors(ctx, validationErrors{"project": {"A project is required."}})
	}

	if port, err := strconv.Atoi(ctx.Request().Input("port")); err == nil && port > 0 {
		input.Config.ExposePort = port
	}

	return input, nil
}

// composeCheckoutExists reports whether the repository is already checked out.
func composeCheckoutExists(_ *models.Application, directory string) bool {
	_, err := os.Stat(filepath.Join(directory, ".git"))

	return err == nil
}
