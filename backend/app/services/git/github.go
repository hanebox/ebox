package git

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/bradleyfalzon/ghinstallation/v2"
	"github.com/google/go-github/v88/github"
)

const githubTimeout = 30 * time.Second

// GithubApp is a thin client over a GitHub App source. App-level requests are
// signed with a JWT; installation-level requests use cached installation
// tokens that ghinstallation refreshes on demand.
type GithubApp struct {
	UUID           string
	AppID          int64
	InstallationID int64
	PrivateKey     string
}

type InstallationInfo struct {
	ID      int64  `json:"id"`
	Account string `json:"account"`
	Type    string `json:"type"`
	HTMLURL string `json:"html_url"`
}

type RepositoryInfo struct {
	FullName      string `json:"full_name"`
	Name          string `json:"name"`
	Private       bool   `json:"private"`
	DefaultBranch string `json:"default_branch"`
	HTMLURL       string `json:"html_url"`
	CloneURL      string `json:"clone_url"`
	SSHURL        string `json:"ssh_url"`
	Description   string `json:"description"`
	UpdatedAt     string `json:"updated_at"`
}

type BranchInfo struct {
	Name      string `json:"name"`
	Protected bool   `json:"protected"`
	Default   bool   `json:"default"`
}

type ManifestApp struct {
	AppID         int64
	Slug          string
	Name          string
	HTMLURL       string
	PEM           string
	WebhookSecret string
}

var installationTransports sync.Map

func (a *GithubApp) appTransport() (*ghinstallation.AppsTransport, error) {
	transport, err := ghinstallation.NewAppsTransport(http.DefaultTransport, a.AppID, []byte(a.PrivateKey))
	if err != nil {
		return nil, fmt.Errorf("github: invalid app credentials: %w", err)
	}

	return transport, nil
}

func (a *GithubApp) appClient() (*github.Client, error) {
	transport, err := a.appTransport()
	if err != nil {
		return nil, err
	}

	return github.NewClient(github.WithTransport(transport), github.WithTimeout(githubTimeout))
}

func (a *GithubApp) installationTransport() (*ghinstallation.Transport, error) {
	if a.InstallationID == 0 {
		return nil, errors.New("github: no installation is connected to this app")
	}

	key := fmt.Sprintf("%s:%d", a.UUID, a.InstallationID)
	if cached, ok := installationTransports.Load(key); ok {
		return cached.(*ghinstallation.Transport), nil
	}

	transport, err := ghinstallation.New(http.DefaultTransport, a.AppID, a.InstallationID, []byte(a.PrivateKey))
	if err != nil {
		return nil, fmt.Errorf("github: invalid app credentials: %w", err)
	}

	actual, _ := installationTransports.LoadOrStore(key, transport)

	return actual.(*ghinstallation.Transport), nil
}

func (a *GithubApp) installationClient() (*github.Client, error) {
	transport, err := a.installationTransport()
	if err != nil {
		return nil, err
	}

	return github.NewClient(github.WithTransport(transport), github.WithTimeout(githubTimeout))
}

// InstallationToken returns a cached installation access token, refreshing it
// when it is about to expire.
func (a *GithubApp) InstallationToken(ctx context.Context) (string, error) {
	transport, err := a.installationTransport()
	if err != nil {
		return "", err
	}

	return transport.Token(ctx)
}

func (a *GithubApp) Verify(ctx context.Context) (*github.App, error) {
	client, err := a.appClient()
	if err != nil {
		return nil, err
	}

	app, _, err := client.Apps.Get(ctx, "")
	if err != nil {
		return nil, err
	}

	return app, nil
}

func (a *GithubApp) ListInstallations(ctx context.Context) ([]InstallationInfo, error) {
	client, err := a.appClient()
	if err != nil {
		return nil, err
	}

	installations, _, err := client.Apps.ListInstallations(ctx, &github.ListOptions{PerPage: 100})
	if err != nil {
		return nil, err
	}

	infos := make([]InstallationInfo, 0, len(installations))
	for _, installation := range installations {
		account := ""
		if installation.Account != nil {
			account = installation.Account.GetLogin()
		}

		infos = append(infos, InstallationInfo{
			ID:      installation.GetID(),
			Account: account,
			Type:    installation.GetTargetType(),
			HTMLURL: installation.GetHTMLURL(),
		})
	}

	return infos, nil
}

func (a *GithubApp) ListRepositories(ctx context.Context, page, perPage int) ([]RepositoryInfo, int, error) {
	client, err := a.installationClient()
	if err != nil {
		return nil, 0, err
	}

	result, _, err := client.Apps.ListRepos(ctx, &github.ListOptions{Page: page, PerPage: perPage})
	if err != nil {
		return nil, 0, err
	}

	repositories := make([]RepositoryInfo, 0, len(result.Repositories))
	for _, repository := range result.Repositories {
		repositories = append(repositories, RepositoryInfo{
			FullName:      repository.GetFullName(),
			Name:          repository.GetName(),
			Private:       repository.GetPrivate(),
			DefaultBranch: repository.GetDefaultBranch(),
			HTMLURL:       repository.GetHTMLURL(),
			CloneURL:      repository.GetCloneURL(),
			SSHURL:        repository.GetSSHURL(),
			Description:   repository.GetDescription(),
			UpdatedAt:     repository.GetUpdatedAt().Time.Format(time.RFC3339),
		})
	}

	total := len(repositories)
	if result.TotalCount != nil {
		total = *result.TotalCount
	}

	return repositories, total, nil
}

func (a *GithubApp) ListBranches(ctx context.Context, fullName, defaultBranch string) ([]BranchInfo, error) {
	owner, name, err := splitRepository(fullName)
	if err != nil {
		return nil, err
	}

	client, err := a.installationClient()
	if err != nil {
		return nil, err
	}

	branches, _, err := client.Repositories.ListBranches(ctx, owner, name, &github.BranchListOptions{
		ListOptions: github.ListOptions{PerPage: 100},
	})
	if err != nil {
		return nil, err
	}

	infos := make([]BranchInfo, 0, len(branches))
	seenDefault := false
	for _, branch := range branches {
		isDefault := branch.GetName() == defaultBranch
		if isDefault {
			seenDefault = true
		}

		infos = append(infos, BranchInfo{
			Name:      branch.GetName(),
			Protected: branch.GetProtected(),
			Default:   isDefault,
		})
	}

	if defaultBranch != "" && !seenDefault {
		infos = append([]BranchInfo{{Name: defaultBranch, Default: true}}, infos...)
	}

	return infos, nil
}

// CompleteManifest exchanges the temporary code GitHub redirects with for the
// freshly created app's credentials.
func CompleteManifest(ctx context.Context, code string) (*ManifestApp, error) {
	client, err := github.NewClient(github.WithTimeout(githubTimeout))
	if err != nil {
		return nil, err
	}

	config, _, err := client.Apps.CompleteAppManifest(ctx, code)
	if err != nil {
		return nil, err
	}

	return &ManifestApp{
		AppID:         config.GetID(),
		Slug:          config.GetSlug(),
		Name:          config.GetName(),
		HTMLURL:       config.GetHTMLURL(),
		PEM:           config.GetPEM(),
		WebhookSecret: config.GetWebhookSecret(),
	}, nil
}

func splitRepository(fullName string) (string, string, error) {
	parts := strings.SplitN(fullName, "/", 2)
	if len(parts) != 2 || parts[0] == "" || parts[1] == "" {
		return "", "", fmt.Errorf("github: invalid repository name %q", fullName)
	}

	return parts[0], parts[1], nil
}
