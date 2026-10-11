package controllers

import (
	"net/http"
	"time"

	contractshttp "github.com/goravel/framework/contracts/http"

	"ebox/app/facades"
	"ebox/app/services/docker"
	"ebox/app/services/realtime"
	"ebox/app/services/system"
	"ebox/app/support/httpx"
	"ebox/app/support/version"
)

type SystemController struct{}

func NewSystemController() *SystemController {
	return &SystemController{}
}

func (c *SystemController) Health(ctx contractshttp.Context) contractshttp.Response {
	return httpx.Ok(ctx, contractshttp.Json{
		"status":  "ok",
		"version": version.Version,
		"time":    time.Now(),
	})
}

func (c *SystemController) Info(ctx contractshttp.Context) contractshttp.Response {
	dataDir := facades.Config().GetString("ebox.data_dir")

	return httpx.Ok(ctx, system.Get(dataDir))
}

func (c *SystemController) DiskUsage(ctx contractshttp.Context) contractshttp.Response {
	usage, err := docker.Default().DiskUsage(ctx.Request().Origin().Context())
	if err != nil {
		return httpx.Err(ctx, http.StatusBadGateway, docker.FriendlyError(err))
	}

	return httpx.Ok(ctx, usage)
}

// Prune reclaims unused Docker data. Managed containers, images in use and the
// ebox image namespace are never touched.
func (c *SystemController) Prune(ctx contractshttp.Context) contractshttp.Response {
	targets := ctx.Request().InputArray("targets")
	if len(targets) == 0 {
		return httpx.ValidationErrors(ctx, validationErrors{
			"targets": {"Select at least one target to prune."},
		})
	}

	result, err := docker.Default().Prune(ctx.Request().Origin().Context(), targets)
	if err != nil {
		return httpx.Err(ctx, http.StatusBadGateway, docker.FriendlyError(err))
	}

	realtime.Default().Publish(realtime.Event{
		Topic: "system",
		Type:  "prune",
		Data:  map[string]any{"reclaimed": result.SpaceReclaimed},
	})

	return httpx.Ok(ctx, result)
}
