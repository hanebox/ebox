package controllers

import (
	"net/http"

	contractshttp "github.com/goravel/framework/contracts/http"

	"ebox/app/services/applications"
	"ebox/app/services/databases"
	"ebox/app/services/docker"
	"ebox/app/services/stacks"
	"ebox/app/support/httpx"
)

// resourceTarget identifies the record an environment or domain request belongs
// to. Applications, databases and services share these endpoints.
type resourceTarget struct {
	Type string
	ID   uint64
	Name string
}

// resolveResource loads the resource addressed by the uuid route parameter for
// the given resource type.
func resolveResource(ctx contractshttp.Context, resourceType string) (resourceTarget, contractshttp.Response) {
	uuid := ctx.Request().Input("uuid")

	switch resourceType {
	case docker.ResourceDatabase:
		database, err := databases.Default().Find(uuid)
		if err != nil {
			return resourceTarget{}, httpx.Err(ctx, http.StatusNotFound, "Database not found.")
		}

		return resourceTarget{Type: resourceType, ID: database.ID, Name: database.Name}, nil

	case docker.ResourceService:
		service, err := stacks.Default().Find(uuid)
		if err != nil {
			return resourceTarget{}, httpx.Err(ctx, http.StatusNotFound, "Service not found.")
		}

		return resourceTarget{Type: resourceType, ID: service.ID, Name: service.Name}, nil
	}

	app, err := applications.Default().Find(uuid)
	if err != nil {
		return resourceTarget{}, httpx.Err(ctx, http.StatusNotFound, "Application not found.")
	}

	return resourceTarget{Type: docker.ResourceApplication, ID: app.ID, Name: app.Name}, nil
}
