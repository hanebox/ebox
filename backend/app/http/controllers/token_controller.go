package controllers

import (
	"net/http"
	"strings"
	"time"

	contractshttp "github.com/goravel/framework/contracts/http"

	"ebox/app/services/account"
	"ebox/app/support/httpx"
)

type TokenController struct{}

func NewTokenController() *TokenController {
	return &TokenController{}
}

func (c *TokenController) Index(ctx contractshttp.Context) contractshttp.Response {
	user := httpx.User(ctx)
	if user == nil {
		return httpx.Err(ctx, http.StatusUnauthorized, "Unauthenticated.")
	}

	tokens, err := account.Default().ListAccessTokens(user.ID)
	if err != nil {
		return httpx.Err(ctx, http.StatusInternalServerError, "Failed to load access tokens.")
	}

	return httpx.Ok(ctx, tokens)
}

func (c *TokenController) Store(ctx contractshttp.Context) contractshttp.Response {
	user := httpx.User(ctx)
	if user == nil {
		return httpx.Err(ctx, http.StatusUnauthorized, "Unauthenticated.")
	}

	name := strings.TrimSpace(ctx.Request().Input("name"))
	days := ctx.Request().InputInt("expires_in_days", 0)

	validation := validationErrors{}
	if name == "" {
		validation.add("name", "The name field is required.")
	}
	if days < 0 || days > 3650 {
		validation.add("expires_in_days", "The expiry must be between 0 and 3650 days.")
	}
	if !validation.empty() {
		return httpx.ValidationErrors(ctx, validation)
	}

	var expiresAt *time.Time
	if days > 0 {
		value := time.Now().UTC().Add(time.Duration(days) * 24 * time.Hour)
		expiresAt = &value
	}

	token, record, err := account.Default().CreateAccessToken(user, name, expiresAt)
	if err != nil {
		return httpx.Err(ctx, http.StatusInternalServerError, "Failed to create the access token.")
	}

	return httpx.Created(ctx, contractshttp.Json{
		"token":  token,
		"record": record,
	})
}

func (c *TokenController) Destroy(ctx contractshttp.Context) contractshttp.Response {
	user := httpx.User(ctx)
	if user == nil {
		return httpx.Err(ctx, http.StatusUnauthorized, "Unauthenticated.")
	}

	if err := account.Default().DeleteAccessToken(user.ID, uint64(ctx.Request().InputInt64("id"))); err != nil {
		return httpx.Err(ctx, http.StatusInternalServerError, "Failed to revoke the access token.")
	}

	return httpx.Ok(ctx, contractshttp.Json{"message": "Access token revoked."})
}
