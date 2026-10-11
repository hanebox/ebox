package sources

import (
	"context"
	"errors"
	"fmt"

	"ebox/app/facades"
	"ebox/app/models"
	"ebox/app/services/git"
)

type CloneInfo struct {
	URL         string
	Credentials git.Credentials
}

// FindByID loads a source by its primary key.
func (s *Service) FindByID(id uint64) (*models.Source, error) {
	var source models.Source
	if err := facades.Orm().Query().Where("id", id).FirstOrFail(&source); err != nil {
		return nil, err
	}

	return &source, nil
}

// CloneInfo resolves the clone URL and credentials for an application.
func (s *Service) CloneInfo(ctx context.Context, app *models.Application) (CloneInfo, error) {
	if app.Kind == models.ApplicationKindImage {
		return CloneInfo{}, errors.New("sources: image applications have no repository")
	}
	if app.SourceID == nil {
		return CloneInfo{}, errors.New("sources: this application has no source connected")
	}

	return s.CloneInfoFor(ctx, *app.SourceID, app.Repository)
}

// CloneInfoFor resolves the clone URL and credentials for a source and
// repository pair. It is shared by applications and compose services.
func (s *Service) CloneInfoFor(ctx context.Context, sourceID uint64, repository string) (CloneInfo, error) {
	source, err := s.FindByID(sourceID)
	if err != nil {
		return CloneInfo{}, err
	}

	switch source.Type {
	case models.SourceTypeGit:
		credentials, err := s.GitCredentials(source)
		if err != nil {
			return CloneInfo{}, err
		}

		return CloneInfo{URL: source.GitURL, Credentials: credentials}, nil
	case models.SourceTypeGithubApp:
		if repository == "" {
			return CloneInfo{}, errors.New("sources: this application has no repository selected")
		}

		client, err := s.GithubClient(source)
		if err != nil {
			return CloneInfo{}, err
		}

		token, err := client.InstallationToken(ctx)
		if err != nil {
			return CloneInfo{}, err
		}

		return CloneInfo{
			URL:         fmt.Sprintf("https://github.com/%s.git", repository),
			Credentials: git.Credentials{AuthType: models.SourceAuthToken, Token: token, KeyName: source.UUID},
		}, nil
	}

	return CloneInfo{}, fmt.Errorf("sources: unsupported source type %q", source.Type)
}
