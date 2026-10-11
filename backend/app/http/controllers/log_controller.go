package controllers

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	contractshttp "github.com/goravel/framework/contracts/http"

	"ebox/app/services/applications"
	"ebox/app/services/databases"
	"ebox/app/services/deployment"
	"ebox/app/services/docker"
	"ebox/app/services/stacks"
	"ebox/app/support/httpx"
)

type LogController struct{}

func NewLogController() *LogController {
	return &LogController{}
}

// resolveContainer finds the container a log request targets. Services can
// select one of their compose services with the container query parameter.
func (c *LogController) resolveContainer(ctx contractshttp.Context, resourceType string) (string, string, contractshttp.Response) {
	uuid := ctx.Request().Input("uuid")
	requestCtx := ctx.Request().Origin().Context()

	switch resourceType {
	case docker.ResourceDatabase:
		database, err := databases.Default().Find(uuid)
		if err != nil {
			return "", "", httpx.Err(ctx, http.StatusNotFound, "Database not found.")
		}

		containers, err := docker.Default().FindByResource(requestCtx, docker.ResourceDatabase, database.UUID)
		if err != nil {
			return "", "", httpx.Err(ctx, http.StatusBadGateway, docker.FriendlyError(err))
		}
		if len(containers) == 0 {
			return database.Name, "", httpx.Err(ctx, http.StatusNotFound, "This database has no container yet.")
		}

		return database.Name, containers[0].ID, nil

	case docker.ResourceService:
		service, err := stacks.Default().Find(uuid)
		if err != nil {
			return "", "", httpx.Err(ctx, http.StatusNotFound, "Service not found.")
		}

		containers, err := docker.Default().List(requestCtx, docker.ComposeFilter(service.ComposeProject()))
		if err != nil {
			return "", "", httpx.Err(ctx, http.StatusBadGateway, docker.FriendlyError(err))
		}
		if len(containers) == 0 {
			return service.Name, "", httpx.Err(ctx, http.StatusNotFound, "This stack is not running.")
		}

		requested := ctx.Request().Input("container")
		if requested != "" {
			for _, container := range containers {
				if container.ID == requested ||
					container.Name == requested ||
					container.Labels["com.docker.compose.service"] == requested {
					return service.Name + "/" + container.Labels["com.docker.compose.service"], container.ID, nil
				}
			}
		}

		entry, _, err := stacks.Default().EntryService(requestCtx, service)
		if err == nil {
			for _, container := range containers {
				if container.Labels["com.docker.compose.service"] == entry {
					return service.Name + "/" + entry, container.ID, nil
				}
			}
		}

		return service.Name, containers[0].ID, nil
	}

	app, err := applications.Default().Find(uuid)
	if err != nil {
		return "", "", httpx.Err(ctx, http.StatusNotFound, "Application not found.")
	}

	containers, err := docker.Default().FindByResource(requestCtx, docker.ResourceApplication, app.UUID)
	if err != nil {
		return "", "", httpx.Err(ctx, http.StatusBadGateway, docker.FriendlyError(err))
	}
	if len(containers) == 0 {
		// Compose applications create their containers through the compose
		// project, not the application labels alone.
		if app.IsCompose() {
			composeContainers, listErr := docker.Default().List(requestCtx, docker.ComposeFilter(deployment.ComposeProjectName(app)))
			if listErr == nil {
				containers = composeContainers
			}
		}
	}
	if len(containers) == 0 {
		return app.Name, "", httpx.Err(ctx, http.StatusNotFound, "This application has no container yet.")
	}

	if requested := ctx.Request().Input("container"); requested != "" {
		for _, container := range containers {
			if container.ID == requested ||
				container.Name == requested ||
				container.Labels["com.docker.compose.service"] == requested {
				return app.Name + "/" + container.Labels["com.docker.compose.service"], container.ID, nil
			}
		}
	}

	for _, container := range containers {
		if container.State == "running" {
			service := container.Labels["com.docker.compose.service"]
			if service != "" {
				return app.Name + "/" + service, container.ID, nil
			}

			return app.Name, container.ID, nil
		}
	}

	return app.Name, "", httpx.Err(ctx, http.StatusNotFound, "This application is not running.")
}

// Stream follows a container log over Server-Sent Events.
func (c *LogController) Stream(ctx contractshttp.Context, resourceType string) contractshttp.Response {
	name, containerID, failure := c.resolveContainer(ctx, resourceType)
	if failure != nil {
		return failure
	}

	writer := ctx.Response().Writer()
	flusher, ok := writer.(http.Flusher)
	if !ok {
		return httpx.Err(ctx, http.StatusInternalServerError, "Streaming is not supported.")
	}

	writer.Header().Set("Content-Type", "text/event-stream")
	writer.Header().Set("Cache-Control", "no-cache")
	writer.Header().Set("Connection", "keep-alive")
	writer.Header().Set("X-Accel-Buffering", "no")
	writer.WriteHeader(http.StatusOK)

	send := func(line string) {
		payload, err := json.Marshal(map[string]string{"line": line})
		if err != nil {
			return
		}

		_, _ = fmt.Fprintf(writer, "data: %s\n\n", payload)
		flusher.Flush()
	}

	send(fmt.Sprintf("\u001b[90m— following %s —\u001b[0m", name))

	err := docker.Default().StreamLogLines(ctx.Request().Origin().Context(), containerID, parseTail(ctx, 200), true, send)
	if err != nil && ctx.Request().Origin().Context().Err() == nil {
		send("\u001b[31m" + err.Error() + "\u001b[0m")
	}

	return nil
}

// Download returns a log snapshot as a file for offline reading.
func (c *LogController) Download(ctx contractshttp.Context, resourceType string) contractshttp.Response {
	name, containerID, failure := c.resolveContainer(ctx, resourceType)
	if failure != nil {
		return failure
	}

	output, err := docker.Default().Logs(ctx.Request().Origin().Context(), containerID, parseTail(ctx, 5000))
	if err != nil {
		return httpx.Err(ctx, http.StatusBadGateway, docker.FriendlyError(err))
	}

	filename := fmt.Sprintf("%s-%s.log", safeName(name), time.Now().UTC().Format("20060102-150405"))

	ctx.Response().Header("Content-Disposition", "attachment; filename=\""+filename+"\"")
	ctx.Response().Header("Content-Type", "text/plain; charset=utf-8")

	return ctx.Response().String(http.StatusOK, output)
}

func safeName(value string) string {
	return strings.Map(func(character rune) rune {
		switch {
		case character >= 'a' && character <= 'z',
			character >= 'A' && character <= 'Z',
			character >= '0' && character <= '9',
			character == '-', character == '_':
			return character
		}

		return '-'
	}, value)
}
