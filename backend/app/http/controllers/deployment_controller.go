package controllers

import (
	"net/http"

	contractshttp "github.com/goravel/framework/contracts/http"

	"ebox/app/facades"
	"ebox/app/models"
	"ebox/app/services/deployment"
	"ebox/app/support/httpx"
)

type DeploymentController struct{}

func NewDeploymentController() *DeploymentController {
	return &DeploymentController{}
}

type deploymentResponse struct {
	models.Deployment
	ApplicationName string `json:"application_name"`
	ApplicationUUID string `json:"application_uuid"`
}

func (c *DeploymentController) Index(ctx contractshttp.Context) contractshttp.Response {
	query := facades.Orm().Query().Order("id desc")

	if applicationUUID := ctx.Request().Input("application"); applicationUUID != "" {
		var app models.Application
		if err := facades.Orm().Query().Where("uuid", applicationUUID).FirstOrFail(&app); err != nil {
			return httpx.Err(ctx, http.StatusNotFound, "Application not found.")
		}
		query = query.Where("application_id", app.ID)
	}

	limit := ctx.Request().InputInt("limit", 25)
	if limit < 1 || limit > 200 {
		limit = 25
	}

	var list []models.Deployment
	if err := query.Limit(limit).Find(&list); err != nil {
		return httpx.Err(ctx, http.StatusInternalServerError, "Failed to load deployments.")
	}

	return httpx.Ok(ctx, hydrateDeployments(list))
}

func (c *DeploymentController) Show(ctx contractshttp.Context) contractshttp.Response {
	var record models.Deployment
	if err := facades.Orm().Query().Where("uuid", ctx.Request().Input("uuid")).FirstOrFail(&record); err != nil {
		return httpx.Err(ctx, http.StatusNotFound, "Deployment not found.")
	}

	return httpx.Ok(ctx, hydrateDeployments([]models.Deployment{record})[0])
}

// Logs returns log lines after the given sequence number, for live tailing.
func (c *DeploymentController) Logs(ctx contractshttp.Context) contractshttp.Response {
	var record models.Deployment
	if err := facades.Orm().Query().Where("uuid", ctx.Request().Input("uuid")).FirstOrFail(&record); err != nil {
		return httpx.Err(ctx, http.StatusNotFound, "Deployment not found.")
	}

	since := ctx.Request().InputInt("since", 0)
	seq, lines := deployment.Default().Snapshot(&record, since)

	return httpx.Ok(ctx, contractshttp.Json{
		"seq":    seq,
		"lines":  lines,
		"status": record.Status,
	})
}

func (c *DeploymentController) Cancel(ctx contractshttp.Context) contractshttp.Response {
	uuidValue := ctx.Request().Input("uuid")

	if !deployment.Default().Cancel(uuidValue) {
		return httpx.Err(ctx, http.StatusUnprocessableEntity, "This deployment is not running.")
	}

	return httpx.Ok(ctx, contractshttp.Json{"message": "Deployment cancelled."})
}

func hydrateDeployments(list []models.Deployment) []deploymentResponse {
	responses := make([]deploymentResponse, 0, len(list))
	if len(list) == 0 {
		return responses
	}

	ids := make([]uint64, 0, len(list))
	seen := map[uint64]bool{}
	for _, record := range list {
		if !seen[record.ApplicationID] {
			seen[record.ApplicationID] = true
			ids = append(ids, record.ApplicationID)
		}
	}

	applications := map[uint64]models.Application{}
	var records []models.Application
	if err := facades.Orm().Query().Where("id in ?", ids).Find(&records); err == nil {
		for _, record := range records {
			applications[record.ID] = record
		}
	}

	for _, record := range list {
		response := deploymentResponse{Deployment: record}
		if app, ok := applications[record.ApplicationID]; ok {
			response.ApplicationName = app.Name
			response.ApplicationUUID = app.UUID
		}
		responses = append(responses, response)
	}

	return responses
}
