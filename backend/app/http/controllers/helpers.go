package controllers

import (
	"strings"
	"time"

	contractshttp "github.com/goravel/framework/contracts/http"

	"ebox/app/facades"
	"ebox/app/models"
	"ebox/app/support/httpx"
)

type validationErrors map[string][]string

func newValidationErrors() validationErrors {
	return make(validationErrors)
}

func (v validationErrors) add(field, message string) {
	v[field] = append(v[field], message)
}

func (v validationErrors) empty() bool {
	return len(v) == 0
}

func respondWithSession(ctx contractshttp.Context, user *models.User, expiresAt time.Time, sessionToken, csrfToken string) contractshttp.Response {
	httpx.SetSessionCookies(ctx, sessionToken, csrfToken, cookieSecure(ctx), expiresAt)

	return httpx.Ok(ctx, contractshttp.Json{
		"user":       user,
		"csrf_token": csrfToken,
	})
}

func cookieSecure(ctx contractshttp.Context) bool {
	if facades.Config().GetString("app.env") != "production" {
		return false
	}

	request := ctx.Request().Origin()

	return request.TLS != nil || strings.EqualFold(ctx.Request().Header("X-Forwarded-Proto"), "https")
}
