package sources

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/google/uuid"

	"ebox/app/facades"
	"ebox/app/models"
	"ebox/app/services/crypto"
	"ebox/app/services/git"
)

const (
	pendingManifestTTL    = 20 * time.Minute
	pendingManifestPrefix = "ebox:github-manifest:"
)

var ErrNotGithubApp = errors.New("sources: not a github app source")

// PendingManifest carries the state of an in-flight GitHub App manifest flow.
type PendingManifest struct {
	State  string `json:"state"`
	UUID   string `json:"uuid"`
	UserID uint64 `json:"user_id"`
	Origin string `json:"origin"`
}

type GithubAppInput struct {
	UUID           string
	Name           string
	AppID          int64
	AppSlug        string
	HTMLURL        string
	PrivateKey     string
	WebhookSecret  string
	InstallationID string
}

type Service struct{}

func Default() *Service {
	return &Service{}
}

func (s *Service) List() ([]models.Source, error) {
	var list []models.Source
	if err := facades.Orm().Query().Order("name asc").Find(&list); err != nil {
		return nil, err
	}

	return list, nil
}

func (s *Service) Find(uuidValue string) (*models.Source, error) {
	var source models.Source
	if err := facades.Orm().Query().Where("uuid", uuidValue).FirstOrFail(&source); err != nil {
		return nil, err
	}

	return &source, nil
}

func (s *Service) CreateGit(name, gitURL, authType, token, sshKey string) (*models.Source, error) {
	source := &models.Source{
		UUID:     uuid.NewString(),
		Name:     strings.TrimSpace(name),
		Type:     models.SourceTypeGit,
		GitURL:   strings.TrimSpace(gitURL),
		AuthType: authType,
	}

	var err error
	if token != "" {
		if source.TokenEncrypted, err = crypto.Encrypt(token); err != nil {
			return nil, err
		}
	}
	if sshKey != "" {
		if source.SSHKeyEncrypted, err = crypto.Encrypt(sshKey); err != nil {
			return nil, err
		}
	}

	if err := facades.Orm().Query().Create(source); err != nil {
		return nil, err
	}

	return source, nil
}

func (s *Service) CreateGithubApp(input GithubAppInput) (*models.Source, error) {
	uuidValue := input.UUID
	if uuidValue == "" {
		uuidValue = uuid.NewString()
	}

	privateKey, err := crypto.Encrypt(input.PrivateKey)
	if err != nil {
		return nil, err
	}

	source := &models.Source{
		UUID:                uuidValue,
		Name:                strings.TrimSpace(input.Name),
		Type:                models.SourceTypeGithubApp,
		AppID:               input.AppID,
		AppSlug:             input.AppSlug,
		HTMLURL:             input.HTMLURL,
		InstallationID:      input.InstallationID,
		PrivateKeyEncrypted: privateKey,
	}

	if input.WebhookSecret != "" {
		encrypted, err := crypto.Encrypt(input.WebhookSecret)
		if err != nil {
			return nil, err
		}
		source.WebhookSecretEncrypted = encrypted
	}

	if err := facades.Orm().Query().Create(source); err != nil {
		return nil, err
	}

	return source, nil
}

func (s *Service) Update(source *models.Source, name string) error {
	if name != "" {
		source.Name = strings.TrimSpace(name)
	}

	return facades.Orm().Query().Save(source)
}

func (s *Service) SetInstallation(source *models.Source, installationID string) error {
	source.InstallationID = strings.TrimSpace(installationID)

	return facades.Orm().Query().Save(source)
}

func (s *Service) Delete(source *models.Source) error {
	if _, err := facades.Orm().Query().Delete(source); err != nil {
		return err
	}

	git.Default().RemoveKey(source.UUID)

	return nil
}

func (s *Service) PrivateKey(source *models.Source) (string, error) {
	return s.decrypt(source.PrivateKeyEncrypted)
}

func (s *Service) WebhookSecret(source *models.Source) (string, error) {
	return s.decrypt(source.WebhookSecretEncrypted)
}

func (s *Service) Token(source *models.Source) (string, error) {
	return s.decrypt(source.TokenEncrypted)
}

func (s *Service) SSHKey(source *models.Source) (string, error) {
	return s.decrypt(source.SSHKeyEncrypted)
}

// GithubClient builds a GitHub App client for a source.
func (s *Service) GithubClient(source *models.Source) (*git.GithubApp, error) {
	if source.Type != models.SourceTypeGithubApp {
		return nil, ErrNotGithubApp
	}

	privateKey, err := s.PrivateKey(source)
	if err != nil {
		return nil, err
	}

	installationID, _ := strconv.ParseInt(source.InstallationID, 10, 64)

	return &git.GithubApp{
		UUID:           source.UUID,
		AppID:          source.AppID,
		InstallationID: installationID,
		PrivateKey:     privateKey,
	}, nil
}

// GitCredentials returns the credentials a git command needs for a source.
func (s *Service) GitCredentials(source *models.Source) (git.Credentials, error) {
	creds := git.Credentials{
		AuthType: source.AuthType,
		KeyName:  source.UUID,
	}

	var err error
	switch source.AuthType {
	case models.SourceAuthToken:
		creds.Token, err = s.Token(source)
	case models.SourceAuthSSH:
		creds.SSHKey, err = s.SSHKey(source)
	}

	return creds, err
}

// Test checks that a source's credentials work and returns a summary.
func (s *Service) Test(ctx context.Context, source *models.Source) (string, error) {
	switch source.Type {
	case models.SourceTypeGithubApp:
		return s.testGithubApp(ctx, source)
	case models.SourceTypeGit:
		creds, err := s.GitCredentials(source)
		if err != nil {
			return "", err
		}

		if err := git.Default().Test(ctx, source.GitURL, creds); err != nil {
			return "", err
		}

		return "Remote is reachable.", nil
	}

	return "", fmt.Errorf("sources: unknown source type %q", source.Type)
}

func (s *Service) testGithubApp(ctx context.Context, source *models.Source) (string, error) {
	client, err := s.GithubClient(source)
	if err != nil {
		return "", err
	}

	app, err := client.Verify(ctx)
	if err != nil {
		return "", err
	}

	summary := fmt.Sprintf("Authenticated as app %q.", app.GetSlug())
	if source.InstallationID == "" {
		return summary + " No installation is connected yet.", nil
	}

	_, total, err := client.ListRepositories(ctx, 1, 1)
	if err != nil {
		return "", err
	}

	return fmt.Sprintf("%s Installation connected, %d repositories accessible.", summary, total), nil
}

func (s *Service) SavePending(pending PendingManifest) error {
	payload, err := json.Marshal(pending)
	if err != nil {
		return err
	}

	return facades.Cache().Put(pendingManifestPrefix+pending.State, string(payload), pendingManifestTTL)
}

// TakePending consumes a pending manifest state, so it can only be used once.
func (s *Service) TakePending(state string) (*PendingManifest, error) {
	if state == "" {
		return nil, errors.New("sources: missing state")
	}

	value, ok := facades.Cache().Pull(pendingManifestPrefix + state).(string)
	if !ok || value == "" {
		return nil, errors.New("sources: pending manifest not found")
	}

	var pending PendingManifest
	if err := json.Unmarshal([]byte(value), &pending); err != nil {
		return nil, err
	}

	return &pending, nil
}

func (s *Service) decrypt(value string) (string, error) {
	if value == "" {
		return "", nil
	}

	return crypto.Decrypt(value)
}
