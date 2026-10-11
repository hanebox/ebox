package controllers

import (
	"net/http"
	"os"
	"strings"

	"github.com/google/uuid"
	contractshttp "github.com/goravel/framework/contracts/http"

	"ebox/app/services/crypto"
	"ebox/app/services/git"
	"ebox/app/services/settings"
	"ebox/app/services/sources"
	"ebox/app/support/httpx"
)

type GithubController struct{}

func NewGithubController() *GithubController {
	return &GithubController{}
}

// Manifest starts the GitHub App manifest flow and returns the URL the browser
// must POST the manifest to.
func (c *GithubController) Manifest(ctx contractshttp.Context) contractshttp.Response {
	origin := httpx.NormalizeOrigin(ctx.Request().Input("origin"))
	if !httpx.ValidOrigin(origin) {
		return httpx.Err(ctx, http.StatusUnprocessableEntity, "A valid origin is required.")
	}

	name := strings.TrimSpace(ctx.Request().Input("name"))
	if name == "" {
		name = defaultGithubAppName()
	}

	state, err := crypto.RandomToken(16)
	if err != nil {
		return httpx.Err(ctx, http.StatusInternalServerError, "Failed to start the GitHub flow.")
	}

	sourceUUID := uuid.NewString()
	pending := sources.PendingManifest{
		State:  state,
		UUID:   sourceUUID,
		UserID: httpx.User(ctx).ID,
		Origin: origin,
	}
	if err := sources.Default().SavePending(pending); err != nil {
		return httpx.Err(ctx, http.StatusInternalServerError, "Failed to start the GitHub flow.")
	}

	return httpx.Ok(ctx, contractshttp.Json{
		"action_url": "https://github.com/settings/apps/new?state=" + state,
		"manifest":   githubManifest(origin, sourceUUID, name),
		"state":      state,
	})
}

// Callback receives GitHub's redirect after the manifest app was created and
// exchanges the temporary code for app credentials.
func (c *GithubController) Callback(ctx contractshttp.Context) contractshttp.Response {
	pending, err := sources.Default().TakePending(ctx.Request().Input("state"))
	if err != nil {
		return c.failRedirect(ctx, "", "state")
	}

	code := ctx.Request().Input("code")
	if code == "" {
		return c.failRedirect(ctx, pending.Origin, "code")
	}

	app, err := git.CompleteManifest(ctx.Request().Origin().Context(), code)
	if err != nil {
		return c.failRedirect(ctx, pending.Origin, "manifest")
	}

	_, err = sources.Default().CreateGithubApp(sources.GithubAppInput{
		UUID:          pending.UUID,
		Name:          app.Name,
		AppID:         app.AppID,
		AppSlug:       app.Slug,
		HTMLURL:       app.HTMLURL,
		PrivateKey:    app.PEM,
		WebhookSecret: app.WebhookSecret,
	})
	if err != nil {
		return c.failRedirect(ctx, pending.Origin, "save")
	}

	return ctx.Response().Redirect(http.StatusFound, pending.Origin+"/sources?connected="+pending.UUID)
}

// Setup receives GitHub's redirect after the app is installed or its
// permissions change.
func (c *GithubController) Setup(ctx contractshttp.Context) contractshttp.Response {
	origin := httpx.ExternalOrigin(ctx)

	source, err := sources.Default().Find(ctx.Request().Input("source"))
	if err != nil {
		return c.failRedirect(ctx, origin, "source")
	}

	if installationID := ctx.Request().Input("installation_id"); installationID != "" {
		if err := sources.Default().SetInstallation(source, installationID); err != nil {
			return c.failRedirect(ctx, origin, "install")
		}
	}

	return ctx.Response().Redirect(http.StatusFound, origin+"/sources?installed="+source.UUID)
}

func (c *GithubController) failRedirect(ctx contractshttp.Context, origin, reason string) contractshttp.Response {
	if origin == "" {
		origin = httpx.ExternalOrigin(ctx)
	}
	if origin == "" {
		if domain := settings.Default().Get("panel_domain", ""); domain != "" {
			origin = "https://" + domain
		}
	}

	return ctx.Response().Redirect(http.StatusFound, origin+"/sources?error="+reason)
}

func githubManifest(origin, sourceUUID, name string) map[string]any {
	return map[string]any{
		"name": name,
		"url":  origin,
		"hook_attributes": map[string]any{
			"url":    origin + "/api/v1/webhooks/github",
			"active": true,
		},
		"redirect_url":             origin + "/api/v1/github/callback",
		"setup_url":                origin + "/api/v1/github/setup/" + sourceUUID,
		"public":                   false,
		"request_oauth_on_install": false,
		"default_permissions": map[string]any{
			"contents": "read",
			"metadata": "read",
		},
		"default_events": []string{"push"},
	}
}

func defaultGithubAppName() string {
	hostname, err := os.Hostname()
	if err != nil || hostname == "" {
		hostname = "server"
	}

	slug := strings.ToLower(hostname)
	slug = strings.Map(func(r rune) rune {
		if (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') || r == '-' {
			return r
		}

		return '-'
	}, slug)
	slug = strings.Trim(slug, "-")
	if slug == "" {
		slug = "server"
	}

	name := "ebox-" + slug
	if len(name) > 34 {
		name = name[:34]
	}

	return strings.Trim(name, "-")
}
