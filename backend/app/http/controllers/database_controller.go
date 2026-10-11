package controllers

import (
	"net/http"
	"strings"

	contractshttp "github.com/goravel/framework/contracts/http"

	"ebox/app/facades"
	"ebox/app/models"
	"ebox/app/services/databases"
	"ebox/app/services/docker"
	"ebox/app/services/envvars"
	"ebox/app/support/httpx"
)

type DatabaseController struct{}

func NewDatabaseController() *DatabaseController {
	return &DatabaseController{}
}

type databaseResponse struct {
	models.Database
	Runtime databases.Status `json:"runtime"`
}

func (c *DatabaseController) Index(ctx contractshttp.Context) contractshttp.Response {
	list, err := databases.Default().List(ctx.Request().Input("project"))
	if err != nil {
		return httpx.Err(ctx, http.StatusInternalServerError, "Failed to load databases.")
	}

	requestCtx := ctx.Request().Origin().Context()
	responses := make([]databaseResponse, 0, len(list))
	for _, database := range list {
		responses = append(responses, databaseResponse{
			Database: database,
			Runtime:  databases.Default().Status(requestCtx, &database),
		})
	}

	return httpx.Ok(ctx, responses)
}

func (c *DatabaseController) Definitions(ctx contractshttp.Context) contractshttp.Response {
	return httpx.Ok(ctx, databases.Definitions())
}

func (c *DatabaseController) Store(ctx contractshttp.Context) contractshttp.Response {
	input, failure := databaseInput(ctx)
	if failure != nil {
		return failure
	}

	database, err := databases.Default().Create(ctx.Request().Origin().Context(), input)
	if err != nil {
		return httpx.Err(ctx, http.StatusUnprocessableEntity, err.Error())
	}

	// Waiting for the healthcheck keeps the wizard honest: the user leaves with
	// a database that is actually accepting connections.
	if ctx.Request().InputBool("wait", true) {
		if err := databases.Default().WaitReady(ctx.Request().Origin().Context(), database); err != nil {
			return httpx.Err(ctx, http.StatusBadGateway, "The database container started but never became healthy: "+err.Error())
		}
	}

	return httpx.Created(ctx, databaseResponse{
		Database: *database,
		Runtime:  databases.Default().Status(ctx.Request().Origin().Context(), database),
	})
}

func (c *DatabaseController) Show(ctx contractshttp.Context) contractshttp.Response {
	database, err := databases.Default().Find(ctx.Request().Input("uuid"))
	if err != nil {
		return httpx.Err(ctx, http.StatusNotFound, "Database not found.")
	}

	requestCtx := ctx.Request().Origin().Context()
	connection, err := databases.Default().Connection(requestCtx, database, false)
	if err != nil {
		return httpx.Err(ctx, http.StatusInternalServerError, err.Error())
	}

	definition, _ := databases.DefinitionFor(database.Type)

	return httpx.Ok(ctx, struct {
		databaseResponse
		Definition databases.Definition `json:"definition"`
		Connection databases.Connection `json:"connection"`
	}{
		databaseResponse: databaseResponse{
			Database: *database,
			Runtime:  databases.Default().Status(requestCtx, database),
		},
		Definition: definition,
		Connection: connection,
	})
}

// Credentials reveals the generated password and connection URLs.
func (c *DatabaseController) Credentials(ctx contractshttp.Context) contractshttp.Response {
	database, err := databases.Default().Find(ctx.Request().Input("uuid"))
	if err != nil {
		return httpx.Err(ctx, http.StatusNotFound, "Database not found.")
	}

	connection, err := databases.Default().Connection(ctx.Request().Origin().Context(), database, true)
	if err != nil {
		return httpx.Err(ctx, http.StatusInternalServerError, err.Error())
	}

	return httpx.Ok(ctx, connection)
}

func (c *DatabaseController) Update(ctx contractshttp.Context) contractshttp.Response {
	database, err := databases.Default().Find(ctx.Request().Input("uuid"))
	if err != nil {
		return httpx.Err(ctx, http.StatusNotFound, "Database not found.")
	}

	input, failure := databaseInput(ctx)
	if failure != nil {
		return failure
	}
	input.ProjectUUID = database.Project.UUID

	if err := databases.Default().Update(database, input); err != nil {
		return httpx.Err(ctx, http.StatusUnprocessableEntity, err.Error())
	}

	return httpx.Ok(ctx, database)
}

func (c *DatabaseController) Destroy(ctx contractshttp.Context) contractshttp.Response {
	database, err := databases.Default().Find(ctx.Request().Input("uuid"))
	if err != nil {
		return httpx.Err(ctx, http.StatusNotFound, "Database not found.")
	}

	if err := databases.Default().Remove(ctx.Request().Origin().Context(), database); err != nil {
		return httpx.Err(ctx, http.StatusInternalServerError, "Failed to delete the database.")
	}

	_, _ = facades.Orm().Query().Where("resource_type", models.ResourceTypeDatabase).Where("resource_id", database.ID).Delete(&models.EnvVar{})

	return httpx.Ok(ctx, contractshttp.Json{"message": "Database deleted."})
}

func (c *DatabaseController) Start(ctx contractshttp.Context) contractshttp.Response {
	database, definition, failure := c.target(ctx)
	if failure != nil {
		return failure
	}

	if err := databases.Default().Start(ctx.Request().Origin().Context(), database, definition); err != nil {
		return httpx.Err(ctx, http.StatusBadGateway, docker.FriendlyError(err))
	}

	return httpx.Ok(ctx, contractshttp.Json{"message": "Database started."})
}

func (c *DatabaseController) Stop(ctx contractshttp.Context) contractshttp.Response {
	database, _, failure := c.target(ctx)
	if failure != nil {
		return failure
	}

	if err := databases.Default().Stop(ctx.Request().Origin().Context(), database); err != nil {
		return httpx.Err(ctx, http.StatusBadGateway, docker.FriendlyError(err))
	}

	return httpx.Ok(ctx, contractshttp.Json{"message": "Database stopped."})
}

func (c *DatabaseController) Restart(ctx contractshttp.Context) contractshttp.Response {
	database, _, failure := c.target(ctx)
	if failure != nil {
		return failure
	}

	if err := databases.Default().Restart(ctx.Request().Origin().Context(), database); err != nil {
		return httpx.Err(ctx, http.StatusBadGateway, docker.FriendlyError(err))
	}

	return httpx.Ok(ctx, contractshttp.Json{"message": "Database restarted."})
}

// Attach copies the database connection environment into an application.
func (c *DatabaseController) Attach(ctx contractshttp.Context) contractshttp.Response {
	database, err := databases.Default().Find(ctx.Request().Input("uuid"))
	if err != nil {
		return httpx.Err(ctx, http.StatusNotFound, "Database not found.")
	}

	applicationUUID := ctx.Request().Input("application")
	if strings.TrimSpace(applicationUUID) == "" {
		return httpx.ValidationErrors(ctx, validationErrors{"application": {"Select an application."}})
	}

	var app models.Application
	if err := facades.Orm().Query().Where("uuid", applicationUUID).FirstOrFail(&app); err != nil {
		return httpx.Err(ctx, http.StatusNotFound, "Application not found.")
	}

	connection, err := databases.Default().Connection(ctx.Request().Origin().Context(), database, true)
	if err != nil {
		return httpx.Err(ctx, http.StatusInternalServerError, err.Error())
	}

	definition, _ := databases.DefinitionFor(database.Type)
	credentials, err := databases.Default().Credentials(database)
	if err != nil {
		return httpx.Err(ctx, http.StatusInternalServerError, err.Error())
	}

	values := definition.EnvironmentVariables(database.ContainerName(), definition.InternalPort, credentials)
	applied := []string{}

	for key, value := range values {
		if _, err := envvars.Default().Set(models.ResourceTypeApplication, app.ID, key, value, false); err != nil {
			return httpx.Err(ctx, http.StatusInternalServerError, "Failed to save the environment variables.")
		}

		applied = append(applied, key)
	}

	_ = connection

	return httpx.Ok(ctx, contractshttp.Json{
		"message":     "Environment variables copied to " + app.Name + ". Redeploy the application to apply them.",
		"variables":   applied,
		"application": app.UUID,
	})
}

// Logs returns a snapshot of the database container logs.
func (c *DatabaseController) Logs(ctx contractshttp.Context) contractshttp.Response {
	database, err := databases.Default().Find(ctx.Request().Input("uuid"))
	if err != nil {
		return httpx.Err(ctx, http.StatusNotFound, "Database not found.")
	}

	containers, err := docker.Default().FindByResource(ctx.Request().Origin().Context(), docker.ResourceDatabase, database.UUID)
	if err != nil {
		return httpx.Err(ctx, http.StatusBadGateway, docker.FriendlyError(err))
	}
	if len(containers) == 0 {
		return httpx.Ok(ctx, contractshttp.Json{"lines": []string{}, "container": nil})
	}

	output, err := docker.Default().Logs(ctx.Request().Origin().Context(), containers[0].ID, ctx.Request().InputInt("tail", 300))
	if err != nil {
		return httpx.Err(ctx, http.StatusBadGateway, docker.FriendlyError(err))
	}

	lines := []string{}
	if strings.TrimSpace(output) != "" {
		lines = strings.Split(strings.TrimRight(output, "\n"), "\n")
	}

	return httpx.Ok(ctx, contractshttp.Json{"lines": lines, "container": containers[0]})
}

func (c *DatabaseController) target(ctx contractshttp.Context) (*models.Database, databases.Definition, contractshttp.Response) {
	database, err := databases.Default().Find(ctx.Request().Input("uuid"))
	if err != nil {
		return nil, databases.Definition{}, httpx.Err(ctx, http.StatusNotFound, "Database not found.")
	}

	definition, err := databases.DefinitionFor(database.Type)
	if err != nil {
		return nil, databases.Definition{}, httpx.Err(ctx, http.StatusUnprocessableEntity, err.Error())
	}

	return database, definition, nil
}

func databaseInput(ctx contractshttp.Context) (databases.Input, contractshttp.Response) {
	input := databases.Input{
		ProjectUUID: ctx.Request().Input("project"),
		Name:        strings.TrimSpace(ctx.Request().Input("name")),
		Type:        ctx.Request().Input("type"),
		Version:     strings.TrimSpace(ctx.Request().Input("version")),
		Port:        ctx.Request().InputInt("port", 0),
	}

	if input.Name == "" {
		return input, httpx.ValidationErrors(ctx, validationErrors{"name": {"The name is required."}})
	}

	if input.Type == "" {
		return input, httpx.ValidationErrors(ctx, validationErrors{"type": {"Select a database engine."}})
	}

	if limits := ctx.Request().InputMap("limits"); limits != nil {
		memory, _ := limits["memory_mb"].(float64)
		cpus, _ := limits["cpus"].(float64)

		if memory > 0 || cpus > 0 {
			input.Limits = &models.ResourceLimits{MemoryMB: int(memory), CPUs: cpus}
		}
	}

	return input, nil
}
