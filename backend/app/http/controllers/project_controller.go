package controllers

import (
	"net/http"
	"strings"

	contractshttp "github.com/goravel/framework/contracts/http"

	"ebox/app/services/projects"
	"ebox/app/support/httpx"
)

type ProjectController struct{}

func NewProjectController() *ProjectController {
	return &ProjectController{}
}

func (c *ProjectController) Index(ctx contractshttp.Context) contractshttp.Response {
	list, err := projects.Default().List()
	if err != nil {
		return httpx.Err(ctx, http.StatusInternalServerError, "Failed to load projects.")
	}

	return httpx.Ok(ctx, list)
}

func (c *ProjectController) Store(ctx contractshttp.Context) contractshttp.Response {
	name := strings.TrimSpace(ctx.Request().Input("name"))
	description := strings.TrimSpace(ctx.Request().Input("description"))

	validation := validationErrors{}
	if name == "" {
		validation.add("name", "The name field is required.")
	}
	if len(name) > 255 {
		validation.add("name", "The name may not be greater than 255 characters.")
	}
	if !validation.empty() {
		return httpx.ValidationErrors(ctx, validation)
	}

	project, err := projects.Default().Create(name, description)
	if err != nil {
		return httpx.Err(ctx, http.StatusInternalServerError, "Failed to create the project.")
	}

	return httpx.Created(ctx, project)
}

func (c *ProjectController) Show(ctx contractshttp.Context) contractshttp.Response {
	project, err := projects.Default().Find(ctx.Request().Input("uuid"))
	if err != nil {
		return httpx.Err(ctx, http.StatusNotFound, "Project not found.")
	}

	return httpx.Ok(ctx, project)
}

func (c *ProjectController) Update(ctx contractshttp.Context) contractshttp.Response {
	project, err := projects.Default().Find(ctx.Request().Input("uuid"))
	if err != nil {
		return httpx.Err(ctx, http.StatusNotFound, "Project not found.")
	}

	name := strings.TrimSpace(ctx.Request().Input("name"))
	if name == "" {
		return httpx.ValidationErrors(ctx, validationErrors{"name": {"The name field is required."}})
	}
	if len(name) > 255 {
		return httpx.ValidationErrors(ctx, validationErrors{"name": {"The name may not be greater than 255 characters."}})
	}

	if err := projects.Default().Update(project, name, strings.TrimSpace(ctx.Request().Input("description"))); err != nil {
		return httpx.Err(ctx, http.StatusInternalServerError, "Failed to update the project.")
	}

	return httpx.Ok(ctx, project)
}

func (c *ProjectController) Destroy(ctx contractshttp.Context) contractshttp.Response {
	project, err := projects.Default().Find(ctx.Request().Input("uuid"))
	if err != nil {
		return httpx.Err(ctx, http.StatusNotFound, "Project not found.")
	}

	if err := projects.Default().Delete(project); err != nil {
		return httpx.Err(ctx, http.StatusInternalServerError, "Failed to delete the project.")
	}

	return httpx.Ok(ctx, contractshttp.Json{"message": "Project deleted."})
}
