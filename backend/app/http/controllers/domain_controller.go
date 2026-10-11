package controllers

import (
	"net/http"

	contractshttp "github.com/goravel/framework/contracts/http"

	"ebox/app/services/domains"
	"ebox/app/support/httpx"
)

type DomainController struct{}

func NewDomainController() *DomainController {
	return &DomainController{}
}

func (c *DomainController) Index(ctx contractshttp.Context, resourceType string) contractshttp.Response {
	target, failure := resolveResource(ctx, resourceType)
	if failure != nil {
		return failure
	}

	list, err := domains.Default().List(target.Type, target.ID)
	if err != nil {
		return httpx.Err(ctx, http.StatusInternalServerError, "Failed to load domains.")
	}

	return httpx.Ok(ctx, list)
}

func (c *DomainController) Store(ctx contractshttp.Context, resourceType string) contractshttp.Response {
	target, failure := resolveResource(ctx, resourceType)
	if failure != nil {
		return failure
	}

	record, err := domains.Default().Add(
		target.Type,
		target.ID,
		ctx.Request().Input("host"),
		ctx.Request().InputBool("https", true),
		ctx.Request().InputBool("is_primary", false),
	)
	if err != nil {
		return httpx.Err(ctx, http.StatusUnprocessableEntity, err.Error())
	}

	return httpx.Created(ctx, record)
}

func (c *DomainController) Update(ctx contractshttp.Context, resourceType string) contractshttp.Response {
	target, failure := resolveResource(ctx, resourceType)
	if failure != nil {
		return failure
	}

	record, err := domains.Default().Find(target.Type, target.ID, uint64(ctx.Request().InputInt64("id")))
	if err != nil {
		return httpx.Err(ctx, http.StatusNotFound, "Domain not found.")
	}

	if err := domains.Default().Update(
		record,
		ctx.Request().Input("host"),
		ctx.Request().InputBool("https", record.HTTPS),
		ctx.Request().InputBool("is_primary", record.IsPrimary),
	); err != nil {
		return httpx.Err(ctx, http.StatusUnprocessableEntity, err.Error())
	}

	return httpx.Ok(ctx, record)
}

func (c *DomainController) Destroy(ctx contractshttp.Context, resourceType string) contractshttp.Response {
	target, failure := resolveResource(ctx, resourceType)
	if failure != nil {
		return failure
	}

	record, err := domains.Default().Find(target.Type, target.ID, uint64(ctx.Request().InputInt64("id")))
	if err != nil {
		return httpx.Err(ctx, http.StatusNotFound, "Domain not found.")
	}

	if err := domains.Default().Delete(record); err != nil {
		return httpx.Err(ctx, http.StatusInternalServerError, "Failed to delete the domain.")
	}

	return httpx.Ok(ctx, contractshttp.Json{"message": "Domain deleted."})
}

