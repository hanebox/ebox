package controllers

import (
	"net/http"
	"strconv"
	"strings"

	contractshttp "github.com/goravel/framework/contracts/http"

	"ebox/app/models"
	"ebox/app/services/git"
	"ebox/app/services/sources"
	"ebox/app/support/httpx"
)

type SourceController struct{}

func NewSourceController() *SourceController {
	return &SourceController{}
}

func (c *SourceController) Index(ctx contractshttp.Context) contractshttp.Response {
	list, err := sources.Default().List()
	if err != nil {
		return httpx.Err(ctx, http.StatusInternalServerError, "Failed to load sources.")
	}

	return httpx.Ok(ctx, list)
}

func (c *SourceController) Store(ctx contractshttp.Context) contractshttp.Response {
	switch ctx.Request().Input("type") {
	case models.SourceTypeGit:
		return c.storeGit(ctx)
	case models.SourceTypeGithubApp:
		return c.storeGithubApp(ctx)
	}

	return httpx.Err(ctx, http.StatusUnprocessableEntity, "Unknown source type.")
}

func (c *SourceController) Show(ctx contractshttp.Context) contractshttp.Response {
	source, err := sources.Default().Find(ctx.Request().Input("uuid"))
	if err != nil {
		return httpx.Err(ctx, http.StatusNotFound, "Source not found.")
	}

	return httpx.Ok(ctx, source)
}

func (c *SourceController) Update(ctx contractshttp.Context) contractshttp.Response {
	source, err := sources.Default().Find(ctx.Request().Input("uuid"))
	if err != nil {
		return httpx.Err(ctx, http.StatusNotFound, "Source not found.")
	}

	name := strings.TrimSpace(ctx.Request().Input("name"))
	if name == "" {
		return httpx.ValidationErrors(ctx, validationErrors{"name": {"The name field is required."}})
	}

	if err := sources.Default().Update(source, name); err != nil {
		return httpx.Err(ctx, http.StatusInternalServerError, "Failed to update the source.")
	}

	return httpx.Ok(ctx, source)
}

func (c *SourceController) SetInstallation(ctx contractshttp.Context) contractshttp.Response {
	source, err := sources.Default().Find(ctx.Request().Input("uuid"))
	if err != nil {
		return httpx.Err(ctx, http.StatusNotFound, "Source not found.")
	}
	if source.Type != models.SourceTypeGithubApp {
		return httpx.Err(ctx, http.StatusUnprocessableEntity, "Only GitHub App sources have installations.")
	}

	installationID := strings.TrimSpace(ctx.Request().Input("installation_id"))
	if _, err := strconv.ParseInt(installationID, 10, 64); err != nil {
		return httpx.ValidationErrors(ctx, validationErrors{"installation_id": {"The installation ID must be a number."}})
	}

	if err := sources.Default().SetInstallation(source, installationID); err != nil {
		return httpx.Err(ctx, http.StatusInternalServerError, "Failed to save the installation.")
	}

	return httpx.Ok(ctx, source)
}

func (c *SourceController) Destroy(ctx contractshttp.Context) contractshttp.Response {
	source, err := sources.Default().Find(ctx.Request().Input("uuid"))
	if err != nil {
		return httpx.Err(ctx, http.StatusNotFound, "Source not found.")
	}

	if err := sources.Default().Delete(source); err != nil {
		return httpx.Err(ctx, http.StatusInternalServerError, "Failed to delete the source.")
	}

	return httpx.Ok(ctx, contractshttp.Json{"message": "Source deleted."})
}

func (c *SourceController) Test(ctx contractshttp.Context) contractshttp.Response {
	source, err := sources.Default().Find(ctx.Request().Input("uuid"))
	if err != nil {
		return httpx.Err(ctx, http.StatusNotFound, "Source not found.")
	}

	message, err := sources.Default().Test(ctx.Request().Origin().Context(), source)
	if err != nil {
		return httpx.Err(ctx, http.StatusUnprocessableEntity, err.Error())
	}

	return httpx.Ok(ctx, contractshttp.Json{"message": message})
}

func (c *SourceController) Repositories(ctx contractshttp.Context) contractshttp.Response {
	source, client, failure := c.githubSource(ctx)
	if failure != nil {
		return failure
	}
	_ = source

	page := ctx.Request().InputInt("page", 1)
	if page < 1 {
		page = 1
	}
	perPage := ctx.Request().InputInt("per_page", 100)
	if perPage < 1 || perPage > 100 {
		perPage = 100
	}

	repositories, total, err := client.ListRepositories(ctx.Request().Origin().Context(), page, perPage)
	if err != nil {
		return httpx.Err(ctx, http.StatusBadGateway, "GitHub request failed: "+err.Error())
	}

	return httpx.Ok(ctx, contractshttp.Json{
		"repositories": repositories,
		"total":        total,
		"page":         page,
	})
}

func (c *SourceController) Branches(ctx contractshttp.Context) contractshttp.Response {
	_, client, failure := c.githubSource(ctx)
	if failure != nil {
		return failure
	}

	repository := strings.TrimSpace(ctx.Request().Input("repo"))
	if repository == "" {
		return httpx.ValidationErrors(ctx, validationErrors{"repo": {"The repository is required."}})
	}

	branches, err := client.ListBranches(ctx.Request().Origin().Context(), repository, ctx.Request().Input("default_branch"))
	if err != nil {
		return httpx.Err(ctx, http.StatusBadGateway, "GitHub request failed: "+err.Error())
	}

	return httpx.Ok(ctx, branches)
}

func (c *SourceController) Installations(ctx contractshttp.Context) contractshttp.Response {
	_, client, failure := c.githubSource(ctx)
	if failure != nil {
		return failure
	}

	installations, err := client.ListInstallations(ctx.Request().Origin().Context())
	if err != nil {
		return httpx.Err(ctx, http.StatusBadGateway, "GitHub request failed: "+err.Error())
	}

	return httpx.Ok(ctx, installations)
}

func (c *SourceController) storeGit(ctx contractshttp.Context) contractshttp.Response {
	name := strings.TrimSpace(ctx.Request().Input("name"))
	gitURL := strings.TrimSpace(ctx.Request().Input("git_url"))
	authType := ctx.Request().Input("auth_type", models.SourceAuthNone)
	token := ctx.Request().Input("token")
	sshKey := ctx.Request().Input("ssh_key")

	validation := validationErrors{}
	if name == "" {
		validation.add("name", "The name field is required.")
	}
	if gitURL == "" {
		validation.add("git_url", "The repository URL is required.")
	}
	switch authType {
	case models.SourceAuthNone, models.SourceAuthToken, models.SourceAuthSSH:
	default:
		validation.add("auth_type", "The authentication type is invalid.")
	}
	if authType == models.SourceAuthToken && token == "" {
		validation.add("token", "A token is required for token authentication.")
	}
	if authType == models.SourceAuthSSH && sshKey == "" {
		validation.add("ssh_key", "A private key is required for SSH authentication.")
	}
	if !validation.empty() {
		return httpx.ValidationErrors(ctx, validation)
	}

	if ctx.Request().InputBool("test_connection", true) {
		creds := git.Credentials{AuthType: authType, Token: token, SSHKey: sshKey, KeyName: "probe"}
		if err := git.Default().Test(ctx.Request().Origin().Context(), gitURL, creds); err != nil {
			return httpx.Err(ctx, http.StatusUnprocessableEntity, "Could not reach the repository: "+err.Error())
		}
	}

	source, err := sources.Default().CreateGit(name, gitURL, authType, token, sshKey)
	if err != nil {
		return httpx.Err(ctx, http.StatusInternalServerError, "Failed to create the source.")
	}

	return httpx.Created(ctx, source)
}

func (c *SourceController) storeGithubApp(ctx contractshttp.Context) contractshttp.Response {
	name := strings.TrimSpace(ctx.Request().Input("name"))
	appID := ctx.Request().InputInt64("app_id")
	privateKey := ctx.Request().Input("private_key")
	webhookSecret := ctx.Request().Input("webhook_secret")
	installationID := strings.TrimSpace(ctx.Request().Input("installation_id"))

	validation := validationErrors{}
	if name == "" {
		validation.add("name", "The name field is required.")
	}
	if appID <= 0 {
		validation.add("app_id", "The GitHub App ID is required.")
	}
	if !strings.Contains(privateKey, "PRIVATE KEY") {
		validation.add("private_key", "A PEM encoded private key is required.")
	}
	if installationID != "" {
		if _, err := strconv.ParseInt(installationID, 10, 64); err != nil {
			validation.add("installation_id", "The installation ID must be a number.")
		}
	}
	if !validation.empty() {
		return httpx.ValidationErrors(ctx, validation)
	}

	installation, _ := strconv.ParseInt(installationID, 10, 64)
	client := &git.GithubApp{AppID: appID, InstallationID: installation, PrivateKey: privateKey}
	app, err := client.Verify(ctx.Request().Origin().Context())
	if err != nil {
		return httpx.Err(ctx, http.StatusUnprocessableEntity, "GitHub rejected these credentials: "+err.Error())
	}

	source, err := sources.Default().CreateGithubApp(sources.GithubAppInput{
		Name:           name,
		AppID:          appID,
		AppSlug:        app.GetSlug(),
		HTMLURL:        app.GetHTMLURL(),
		PrivateKey:     privateKey,
		WebhookSecret:  webhookSecret,
		InstallationID: installationID,
	})
	if err != nil {
		return httpx.Err(ctx, http.StatusInternalServerError, "Failed to create the source.")
	}

	return httpx.Created(ctx, source)
}

func (c *SourceController) githubSource(ctx contractshttp.Context) (*models.Source, *git.GithubApp, contractshttp.Response) {
	source, err := sources.Default().Find(ctx.Request().Input("uuid"))
	if err != nil {
		return nil, nil, httpx.Err(ctx, http.StatusNotFound, "Source not found.")
	}

	if source.Type != models.SourceTypeGithubApp {
		return nil, nil, httpx.Err(ctx, http.StatusUnprocessableEntity, "This source does not support the GitHub API.")
	}

	client, err := sources.Default().GithubClient(source)
	if err != nil {
		return nil, nil, httpx.Err(ctx, http.StatusUnprocessableEntity, err.Error())
	}

	return source, client, nil
}
