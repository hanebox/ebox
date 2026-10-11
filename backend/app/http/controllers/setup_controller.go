package controllers

import (
	"net/mail"
	"strings"

	"github.com/goravel/framework/contracts/http"

	"ebox/app/services/account"
	"ebox/app/services/settings"
	"ebox/app/support/httpx"
)

const minPasswordLength = 10

type SetupController struct{}

func NewSetupController() *SetupController {
	return &SetupController{}
}

func (c *SetupController) Status(ctx http.Context) http.Response {
	return httpx.Ok(ctx, http.Json{"initialized": account.Default().Initialized()})
}

func (c *SetupController) Store(ctx http.Context) http.Response {
	accountService := account.Default()
	if accountService.Initialized() {
		return httpx.Err(ctx, http.StatusForbidden, "ebox has already been set up.")
	}

	name := strings.TrimSpace(ctx.Request().Input("name"))
	email := strings.TrimSpace(ctx.Request().Input("email"))
	password := ctx.Request().Input("password")
	panelDomain := strings.TrimSpace(ctx.Request().Input("panel_domain"))
	acmeEmail := strings.TrimSpace(ctx.Request().Input("acme_email"))

	errors := newValidationErrors()
	if name == "" {
		errors.add("name", "The name field is required.")
	}
	if !validEmail(email) {
		errors.add("email", "The email field must be a valid email address.")
	}
	if len(password) < minPasswordLength {
		errors.add("password", "The password must be at least 10 characters.")
	}
	if acmeEmail != "" && !validEmail(acmeEmail) {
		errors.add("acme_email", "The ACME email field must be a valid email address.")
	}
	if !errors.empty() {
		return httpx.ValidationErrors(ctx, errors)
	}

	user, err := accountService.CreateUser(name, email, password)
	if err != nil {
		return httpx.Err(ctx, http.StatusInternalServerError, "Failed to create the account.")
	}

	values := make(map[string]string)
	if panelDomain != "" {
		values["panel_domain"] = panelDomain
	}
	if acmeEmail != "" {
		values["acme_email"] = acmeEmail
	}
	if len(values) > 0 {
		if err := settings.Default().SetMany(values); err != nil {
			return httpx.Err(ctx, http.StatusInternalServerError, "Failed to store the panel settings.")
		}
	}

	session, token, csrf, err := accountService.CreateSession(user, ctx.Request().Ip(), ctx.Request().Header("User-Agent"))
	if err != nil {
		return httpx.Err(ctx, http.StatusInternalServerError, "Failed to create a session.")
	}

	return respondWithSession(ctx, user, session.ExpiresAt, token, csrf)
}

func validEmail(email string) bool {
	if strings.TrimSpace(email) == "" {
		return false
	}

	_, err := mail.ParseAddress(email)

	return err == nil
}
