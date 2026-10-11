package controllers

import (
	"net/http"
	"strconv"

	contractshttp "github.com/goravel/framework/contracts/http"

	"ebox/app/services/applications"
	"ebox/app/services/docker"
	"ebox/app/services/terminal"
	"ebox/app/support/httpx"
)

type TerminalController struct{}

func NewTerminalController() *TerminalController {
	return &TerminalController{}
}

func (c *TerminalController) Targets(ctx contractshttp.Context) contractshttp.Response {
	targets, err := terminal.Default().Targets(ctx.Request().Origin().Context())
	if err != nil && len(targets) == 0 {
		return httpx.Err(ctx, http.StatusBadGateway, docker.FriendlyError(err))
	}

	return httpx.Ok(ctx, targets)
}

// Connect upgrades the request to a WebSocket and bridges it to a shell.
func (c *TerminalController) Connect(ctx contractshttp.Context) contractshttp.Response {
	if err := terminal.Default().Connect(ctx); err != nil {
		// The connection is already upgraded or the failure was reported on the
		// socket, so only log here.
		return nil
	}

	return nil
}

type ContainerController struct{}

func NewContainerController() *ContainerController {
	return &ContainerController{}
}

func (c *ContainerController) Index(ctx contractshttp.Context) contractshttp.Response {
	requestCtx := ctx.Request().Origin().Context()

	containers, err := docker.Default().List(requestCtx, docker.ResourceFilter("", ""))
	if err != nil {
		return httpx.Err(ctx, http.StatusBadGateway, docker.FriendlyError(err))
	}

	type response struct {
		docker.Container
		Stats *docker.ContainerStats `json:"stats,omitempty"`
	}

	runningIDs := []string{}
	for _, container := range containers {
		if container.State == "running" {
			runningIDs = append(runningIDs, container.ID)
		}
	}

	stats := docker.Default().StatsBatch(requestCtx, runningIDs)

	list := make([]response, 0, len(containers))
	for _, container := range containers {
		item := response{Container: container}

		if value, ok := stats[container.ID]; ok {
			statsCopy := value
			item.Stats = &statsCopy
		}

		list = append(list, item)
	}

	return httpx.Ok(ctx, list)
}

func (c *ContainerController) Stats(ctx contractshttp.Context) contractshttp.Response {
	app, err := applications.Default().Find(ctx.Request().Input("uuid"))
	if err != nil {
		return httpx.Err(ctx, http.StatusNotFound, "Application not found.")
	}

	requestCtx := ctx.Request().Origin().Context()

	containers, err := docker.Default().FindByResource(requestCtx, docker.ResourceApplication, app.UUID)
	if err != nil {
		return httpx.Err(ctx, http.StatusBadGateway, docker.FriendlyError(err))
	}

	for _, container := range containers {
		if container.State != "running" {
			continue
		}

		stats, err := docker.Default().Stats(requestCtx, container.ID)
		if err != nil {
			return httpx.Err(ctx, http.StatusBadGateway, docker.FriendlyError(err))
		}

		return httpx.Ok(ctx, contractshttp.Json{
			"container": container,
			"stats":     stats,
			"limits":    app.Config.Limits,
		})
	}

	return httpx.Ok(ctx, contractshttp.Json{"container": nil, "stats": nil, "limits": app.Config.Limits})
}

func parseTail(ctx contractshttp.Context, fallback int) int {
	value := ctx.Request().Input("tail")
	if value == "" {
		return fallback
	}

	parsed, err := strconv.Atoi(value)
	if err != nil || parsed < 0 {
		return fallback
	}

	return parsed
}
