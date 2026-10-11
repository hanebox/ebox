package controllers

import (
	"net/http"
	"strings"

	contractshttp "github.com/goravel/framework/contracts/http"

	"ebox/app/facades"
	"ebox/app/models"
	"ebox/app/services/docker"
	"ebox/app/services/stacks"
	"ebox/app/support/httpx"
)

type StackController struct{}

func NewStackController() *StackController {
	return &StackController{}
}

type stackResponse struct {
	models.Service
	Runtime stacks.Status `json:"runtime"`
}

func (c *StackController) Index(ctx contractshttp.Context) contractshttp.Response {
	list, err := stacks.Default().List(ctx.Request().Input("project"))
	if err != nil {
		return httpx.Err(ctx, http.StatusInternalServerError, "Failed to load services.")
	}

	requestCtx := ctx.Request().Origin().Context()
	responses := make([]stackResponse, 0, len(list))
	for _, service := range list {
		responses = append(responses, stackResponse{
			Service: service,
			Runtime: stacks.Default().Status(requestCtx, &service),
		})
	}

	return httpx.Ok(ctx, responses)
}

func (c *StackController) Show(ctx contractshttp.Context) contractshttp.Response {
	service, err := stacks.Default().Find(ctx.Request().Input("uuid"))
	if err != nil {
		return httpx.Err(ctx, http.StatusNotFound, "Service not found.")
	}

	requestCtx := ctx.Request().Origin().Context()

	document, documentErr := stacks.Default().Document(requestCtx, service, false)
	definitions := []composeInfo{}
	if documentErr == nil {
		for _, info := range document.Info() {
			definitions = append(definitions, composeInfo{
				Name:      info.Name,
				Image:     info.Image,
				Built:     info.Built,
				Ports:     info.Ports,
				DependsOn: info.DependsOn,
			})
		}
	}

	entry, entryPort, _ := stacks.Default().EntryService(requestCtx, service)

	return httpx.Ok(ctx, struct {
		stackResponse
		Definitions []composeInfo `json:"definitions"`
		Entry       string        `json:"entry_service"`
		EntryPort   int           `json:"entry_port"`
	}{
		stackResponse: stackResponse{
			Service: *service,
			Runtime: stacks.Default().Status(requestCtx, service),
		},
		Definitions: definitions,
		Entry:       entry,
		EntryPort:   entryPort,
	})
}

type composeInfo struct {
	Name      string   `json:"name"`
	Image     string   `json:"image"`
	Built     bool     `json:"build"`
	Ports     []string `json:"ports"`
	DependsOn []string `json:"depends_on"`
}

func (c *StackController) Store(ctx contractshttp.Context) contractshttp.Response {
	input, failure := stackInput(ctx)
	if failure != nil {
		return failure
	}

	service, err := stacks.Default().Create(ctx.Request().Origin().Context(), input)
	if err != nil {
		if service != nil {
			return httpx.Err(ctx, http.StatusBadGateway, err.Error())
		}

		return httpx.Err(ctx, http.StatusUnprocessableEntity, err.Error())
	}

	return httpx.Created(ctx, service)
}

func (c *StackController) Update(ctx contractshttp.Context) contractshttp.Response {
	service, err := stacks.Default().Find(ctx.Request().Input("uuid"))
	if err != nil {
		return httpx.Err(ctx, http.StatusNotFound, "Service not found.")
	}

	input, failure := stackInput(ctx)
	if failure != nil {
		return failure
	}
	input.ProjectUUID = service.Project.UUID

	if err := stacks.Default().Update(ctx.Request().Origin().Context(), service, input); err != nil {
		return httpx.Err(ctx, http.StatusUnprocessableEntity, err.Error())
	}

	return httpx.Ok(ctx, service)
}

func (c *StackController) Destroy(ctx contractshttp.Context) contractshttp.Response {
	service, err := stacks.Default().Find(ctx.Request().Input("uuid"))
	if err != nil {
		return httpx.Err(ctx, http.StatusNotFound, "Service not found.")
	}

	if err := stacks.Default().Delete(ctx.Request().Origin().Context(), service); err != nil {
		return httpx.Err(ctx, http.StatusInternalServerError, "Failed to delete the service.")
	}

	_, _ = facades.Orm().Query().Where("resource_type", models.ResourceTypeService).Where("resource_id", service.ID).Delete(&models.EnvVar{})
	_, _ = facades.Orm().Query().Where("resource_type", models.ResourceTypeService).Where("resource_id", service.ID).Delete(&models.Domain{})

	return httpx.Ok(ctx, contractshttp.Json{"message": "Service deleted."})
}

func (c *StackController) Deploy(ctx contractshttp.Context) contractshttp.Response {
	service, err := stacks.Default().Find(ctx.Request().Input("uuid"))
	if err != nil {
		return httpx.Err(ctx, http.StatusNotFound, "Service not found.")
	}

	if err := stacks.Default().Deploy(ctx.Request().Origin().Context(), service, nil); err != nil {
		return httpx.Err(ctx, http.StatusBadGateway, err.Error())
	}

	return httpx.Ok(ctx, contractshttp.Json{"message": "Service deployed."})
}

func (c *StackController) Stop(ctx contractshttp.Context) contractshttp.Response {
	return c.lifecycle(ctx, func(service *models.Service) error {
		return stacks.Default().Stop(ctx.Request().Origin().Context(), service, nil)
	}, "Service stopped.")
}

func (c *StackController) Start(ctx contractshttp.Context) contractshttp.Response {
	return c.lifecycle(ctx, func(service *models.Service) error {
		return stacks.Default().Start(ctx.Request().Origin().Context(), service, nil)
	}, "Service started.")
}

func (c *StackController) Restart(ctx contractshttp.Context) contractshttp.Response {
	return c.lifecycle(ctx, func(service *models.Service) error {
		return stacks.Default().Restart(ctx.Request().Origin().Context(), service, nil)
	}, "Service restarted.")
}

func (c *StackController) Containers(ctx contractshttp.Context) contractshttp.Response {
	service, err := stacks.Default().Find(ctx.Request().Input("uuid"))
	if err != nil {
		return httpx.Err(ctx, http.StatusNotFound, "Service not found.")
	}

	containers, err := docker.Default().List(ctx.Request().Origin().Context(), docker.ComposeFilter(service.ComposeProject()))
	if err != nil {
		return httpx.Err(ctx, http.StatusBadGateway, docker.FriendlyError(err))
	}

	type containerResponse struct {
		ID      string   `json:"id"`
		Name    string   `json:"name"`
		Service string   `json:"service"`
		State   string   `json:"state"`
		Health  string   `json:"health,omitempty"`
		Image   string   `json:"image"`
		Ports   []string `json:"ports,omitempty"`
	}

	list := make([]containerResponse, 0, len(containers))
	for _, container := range containers {
		list = append(list, containerResponse{
			ID:      container.ID,
			Name:    container.Name,
			Service: container.Labels["com.docker.compose.service"],
			State:   container.State,
			Health:  container.Health,
			Image:   container.Image,
			Ports:   container.Ports,
		})
	}

	return httpx.Ok(ctx, list)
}

func (c *StackController) lifecycle(ctx contractshttp.Context, action func(*models.Service) error, message string) contractshttp.Response {
	service, err := stacks.Default().Find(ctx.Request().Input("uuid"))
	if err != nil {
		return httpx.Err(ctx, http.StatusNotFound, "Service not found.")
	}

	if err := action(service); err != nil {
		return httpx.Err(ctx, http.StatusBadGateway, docker.FriendlyError(err))
	}

	return httpx.Ok(ctx, contractshttp.Json{"message": message})
}

func stackInput(ctx contractshttp.Context) (stacks.Input, contractshttp.Response) {
	entryPort := ctx.Request().InputInt("entry_port", 0)

	input := stacks.Input{
		ProjectUUID:  ctx.Request().Input("project"),
		Name:         strings.TrimSpace(ctx.Request().Input("name")),
		Source:       ctx.Request().Input("source"),
		Compose:      ctx.Request().Input("compose"),
		SourceUUID:   ctx.Request().Input("source_uuid"),
		Repository:   strings.TrimSpace(ctx.Request().Input("repository")),
		Branch:       strings.TrimSpace(ctx.Request().Input("branch")),
		ComposePath:  strings.TrimSpace(ctx.Request().Input("compose_path")),
		Environment:  ctx.Request().Input("environment"),
		EntryService: strings.TrimSpace(ctx.Request().Input("entry_service")),
		EntryPort:    entryPort,
	}

	if input.Name == "" {
		return input, httpx.ValidationErrors(ctx, validationErrors{"name": {"The name is required."}})
	}

	if input.Source == models.ServiceSourceGit {
		if input.SourceUUID == "" {
			return input, httpx.ValidationErrors(ctx, validationErrors{"source_uuid": {"Select a source."}})
		}
	} else if strings.TrimSpace(input.Compose) == "" {
		return input, httpx.ValidationErrors(ctx, validationErrors{"compose": {"Paste a compose file."}})
	}

	return input, nil
}
