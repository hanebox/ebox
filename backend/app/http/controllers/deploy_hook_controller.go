package controllers

import (
	"net/http"

	contractshttp "github.com/goravel/framework/contracts/http"

	"ebox/app/models"
	"ebox/app/services/applications"
	"ebox/app/services/deployment"
	"ebox/app/support/httpx"
)

type DeployHookController struct{}

func NewDeployHookController() *DeployHookController {
	return &DeployHookController{}
}

// Deploy queues a deployment for the application that owns the token. Used by
// CI pipelines.
func (c *DeployHookController) Deploy(ctx contractshttp.Context) contractshttp.Response {
	app, err := applications.Default().FindByDeployToken(ctx.Request().Input("token"))
	if err != nil {
		return httpx.Err(ctx, http.StatusNotFound, "Unknown deploy token.")
	}

	record, err := deployment.Default().Enqueue(app, models.DeploymentTriggerToken, ctx.Request().Input("commit"))
	if err != nil {
		return httpx.Err(ctx, http.StatusInternalServerError, "Failed to queue the deployment.")
	}

	return httpx.Created(ctx, contractshttp.Json{
		"application": app.Name,
		"deployment":  record.UUID,
	})
}
