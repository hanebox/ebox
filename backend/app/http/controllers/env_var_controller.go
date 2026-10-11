package controllers

import (
	"net/http"
	"strings"

	contractshttp "github.com/goravel/framework/contracts/http"

	"ebox/app/services/envvars"
	"ebox/app/support/httpx"
)

type EnvVarController struct{}

func NewEnvVarController() *EnvVarController {
	return &EnvVarController{}
}

func (c *EnvVarController) Index(ctx contractshttp.Context, resourceType string) contractshttp.Response {
	target, failure := resolveResource(ctx, resourceType)
	if failure != nil {
		return failure
	}

	list, err := envvars.Default().List(target.Type, target.ID)
	if err != nil {
		return httpx.Err(ctx, http.StatusInternalServerError, "Failed to load environment variables.")
	}

	return httpx.Ok(ctx, list)
}

func (c *EnvVarController) Store(ctx contractshttp.Context, resourceType string) contractshttp.Response {
	target, failure := resolveResource(ctx, resourceType)
	if failure != nil {
		return failure
	}

	key := strings.TrimSpace(ctx.Request().Input("key"))
	if key == "" {
		return httpx.ValidationErrors(ctx, validationErrors{"key": {"The key field is required."}})
	}

	record, err := envvars.Default().Set(
		target.Type,
		target.ID,
		key,
		ctx.Request().Input("value"),
		ctx.Request().InputBool("is_build_time", false),
	)
	if err != nil {
		return httpx.Err(ctx, http.StatusUnprocessableEntity, err.Error())
	}

	return httpx.Created(ctx, record)
}

func (c *EnvVarController) Update(ctx contractshttp.Context, resourceType string) contractshttp.Response {
	target, failure := resolveResource(ctx, resourceType)
	if failure != nil {
		return failure
	}

	record, err := envvars.Default().Update(
		target.Type,
		target.ID,
		uint64(ctx.Request().InputInt64("id")),
		strings.TrimSpace(ctx.Request().Input("key")),
		ctx.Request().Input("value"),
		ctx.Request().InputBool("is_build_time", false),
	)
	if err != nil {
		return httpx.Err(ctx, http.StatusUnprocessableEntity, err.Error())
	}

	return httpx.Ok(ctx, record)
}

func (c *EnvVarController) Destroy(ctx contractshttp.Context, resourceType string) contractshttp.Response {
	target, failure := resolveResource(ctx, resourceType)
	if failure != nil {
		return failure
	}

	if err := envvars.Default().Delete(target.Type, target.ID, uint64(ctx.Request().InputInt64("id"))); err != nil {
		return httpx.Err(ctx, http.StatusInternalServerError, "Failed to delete the environment variable.")
	}

	return httpx.Ok(ctx, contractshttp.Json{"message": "Environment variable deleted."})
}

func (c *EnvVarController) Import(ctx contractshttp.Context, resourceType string) contractshttp.Response {
	target, failure := resolveResource(ctx, resourceType)
	if failure != nil {
		return failure
	}

	count, err := envvars.Default().Import(target.Type, target.ID, ctx.Request().Input("content"))
	if err != nil {
		return httpx.Err(ctx, http.StatusUnprocessableEntity, err.Error())
	}

	return httpx.Ok(ctx, contractshttp.Json{"imported": count})
}

func (c *EnvVarController) Export(ctx contractshttp.Context, resourceType string) contractshttp.Response {
	target, failure := resolveResource(ctx, resourceType)
	if failure != nil {
		return failure
	}

	content, err := envvars.Default().Export(target.Type, target.ID)
	if err != nil {
		return httpx.Err(ctx, http.StatusInternalServerError, "Failed to export the environment.")
	}

	return httpx.Ok(ctx, contractshttp.Json{"content": content})
}
