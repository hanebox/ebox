package controllers

import (
	"strings"

	"github.com/goravel/framework/contracts/http"

	"ebox/app/services/settings"
	"ebox/app/support/httpx"
)

var allowedSettingKeys = []string{"panel_domain", "acme_email", "timezone"}

type SettingsController struct{}

func NewSettingsController() *SettingsController {
	return &SettingsController{}
}

func (c *SettingsController) Index(ctx http.Context) http.Response {
	values, err := settings.Default().All()
	if err != nil {
		return httpx.Err(ctx, http.StatusInternalServerError, "Failed to load settings.")
	}

	out := make(map[string]string, len(allowedSettingKeys))
	for _, key := range allowedSettingKeys {
		out[key] = values[key]
	}

	return httpx.Ok(ctx, out)
}

func (c *SettingsController) Update(ctx http.Context) http.Response {
	payload := ctx.Request().All()

	update := make(map[string]string)
	errors := newValidationErrors()

	for _, key := range allowedSettingKeys {
		raw, ok := payload[key]
		if !ok {
			continue
		}

		value, ok := raw.(string)
		if !ok {
			errors.add(key, "The "+key+" field must be a string.")
			continue
		}
		update[key] = strings.TrimSpace(value)
	}

	if acmeEmail, ok := update["acme_email"]; ok && acmeEmail != "" && !validEmail(acmeEmail) {
		errors.add("acme_email", "The ACME email field must be a valid email address.")
	}
	if !errors.empty() {
		return httpx.ValidationErrors(ctx, errors)
	}

	if err := settings.Default().SetMany(update); err != nil {
		return httpx.Err(ctx, http.StatusInternalServerError, "Failed to update settings.")
	}

	return c.Index(ctx)
}
