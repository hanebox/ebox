package controllers

import (
	"strings"

	"github.com/goravel/framework/contracts/http"

	"ebox/app/services/account"
	"ebox/app/support/httpx"
)

type AccountController struct{}

func NewAccountController() *AccountController {
	return &AccountController{}
}

func (c *AccountController) Update(ctx http.Context) http.Response {
	user := httpx.User(ctx)
	if user == nil {
		return httpx.Err(ctx, http.StatusUnauthorized, "Unauthenticated.")
	}

	name := strings.TrimSpace(ctx.Request().Input("name"))
	email := strings.TrimSpace(ctx.Request().Input("email"))

	errors := newValidationErrors()
	if name == "" {
		errors.add("name", "The name field is required.")
	}
	if !validEmail(email) {
		errors.add("email", "The email field must be a valid email address.")
	}
	if !errors.empty() {
		return httpx.ValidationErrors(ctx, errors)
	}

	if err := account.Default().UpdateProfile(user, name, email); err != nil {
		if err == account.ErrEmailTaken {
			return httpx.ValidationErrors(ctx, validationErrors{"email": {"The email has already been taken."}})
		}

		return httpx.Err(ctx, http.StatusInternalServerError, "Failed to update the profile.")
	}

	return httpx.Ok(ctx, user)
}

func (c *AccountController) UpdatePassword(ctx http.Context) http.Response {
	user := httpx.User(ctx)
	session := httpx.Session(ctx)
	if user == nil || session == nil {
		return httpx.Err(ctx, http.StatusUnauthorized, "Unauthenticated.")
	}

	current := ctx.Request().Input("current_password")
	next := ctx.Request().Input("password")

	errors := newValidationErrors()
	if current == "" {
		errors.add("current_password", "The current password field is required.")
	}
	if len(next) < minPasswordLength {
		errors.add("password", "The password must be at least 10 characters.")
	}
	if !errors.empty() {
		return httpx.ValidationErrors(ctx, errors)
	}

	accountService := account.Default()
	if err := accountService.ChangePassword(user, current, next); err != nil {
		if err == account.ErrInvalidCredentials {
			return httpx.ValidationErrors(ctx, validationErrors{"current_password": {"The current password is incorrect."}})
		}

		return httpx.Err(ctx, http.StatusInternalServerError, "Failed to update the password.")
	}

	_ = accountService.RevokeOtherSessions(user.ID, session.ID)

	return httpx.Ok(ctx, http.Json{"message": "Password updated."})
}
