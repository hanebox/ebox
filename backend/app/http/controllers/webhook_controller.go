package controllers

import (
	"encoding/json"
	"io"
	"net/http"
	"strings"

	contractshttp "github.com/goravel/framework/contracts/http"

	"ebox/app/models"
	"ebox/app/services/applications"
	"ebox/app/services/deployment"
	"ebox/app/services/sources"
	"ebox/app/support/httpx"
)

type WebhookController struct{}

func NewWebhookController() *WebhookController {
	return &WebhookController{}
}

type githubPushPayload struct {
	Ref        string `json:"ref"`
	After      string `json:"after"`
	Repository struct {
		FullName string `json:"full_name"`
	} `json:"repository"`
	HeadCommit struct {
		Message string `json:"message"`
	} `json:"head_commit"`
}

// Github receives push webhooks and queues deployments for matching
// applications.
func (c *WebhookController) Github(ctx contractshttp.Context) contractshttp.Response {
	body, err := io.ReadAll(ctx.Request().Origin().Body)
	if err != nil {
		return httpx.Err(ctx, http.StatusBadRequest, "Could not read the request body.")
	}

	event := ctx.Request().Header("X-GitHub-Event")
	if event == "ping" || event == "" {
		return httpx.Ok(ctx, contractshttp.Json{"message": "pong"})
	}

	if event != "push" {
		return httpx.Ok(ctx, contractshttp.Json{"message": "ignored event " + event})
	}

	signature := ctx.Request().Header("X-Hub-Signature-256")
	source, err := sources.Default().VerifyWebhook(body, signature)
	if err != nil {
		return httpx.Err(ctx, http.StatusUnauthorized, "Invalid webhook signature.")
	}

	var payload githubPushPayload
	if err := json.Unmarshal(body, &payload); err != nil {
		return httpx.Err(ctx, http.StatusBadRequest, "Invalid webhook payload.")
	}

	branch := strings.TrimPrefix(payload.Ref, "refs/heads/")

	list, err := applications.Default().MatchingWebhook(source.ID, payload.Repository.FullName, branch)
	if err != nil {
		return httpx.Err(ctx, http.StatusInternalServerError, "Could not look up applications.")
	}

	queued := make([]string, 0, len(list))
	for index := range list {
		record, err := deployment.Default().Enqueue(&list[index], models.DeploymentTriggerWebhook, payload.After)
		if err != nil {
			continue
		}

		queued = append(queued, record.UUID)
	}

	return httpx.Ok(ctx, contractshttp.Json{
		"repository": payload.Repository.FullName,
		"branch":     branch,
		"queued":     queued,
	})
}
