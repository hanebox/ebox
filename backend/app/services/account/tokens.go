package account

import (
	"errors"
	"strings"
	"sync"
	"time"

	"ebox/app/facades"
	"ebox/app/models"
	"ebox/app/services/crypto"
)

const accessTokenPrefix = "ebox_pat_"

var lastTokenTouch sync.Map

func (s *Service) CreateAccessToken(user *models.User, name string, expiresAt *time.Time) (string, *models.PersonalAccessToken, error) {
	secret, err := crypto.RandomToken(24)
	if err != nil {
		return "", nil, err
	}

	token := accessTokenPrefix + secret
	record := &models.PersonalAccessToken{
		UserID:    user.ID,
		Name:      strings.TrimSpace(name),
		TokenHash: crypto.HashToken(token),
		ExpiresAt: expiresAt,
	}

	if err := facades.Orm().Query().Create(record); err != nil {
		return "", nil, err
	}

	return token, record, nil
}

func (s *Service) ListAccessTokens(userID uint64) ([]models.PersonalAccessToken, error) {
	var tokens []models.PersonalAccessToken
	if err := facades.Orm().Query().Where("user_id", userID).Order("created_at desc").Find(&tokens); err != nil {
		return nil, err
	}

	return tokens, nil
}

func (s *Service) DeleteAccessToken(userID, id uint64) error {
	_, err := facades.Orm().Query().
		Where("user_id", userID).
		Where("id", id).
		Delete(&models.PersonalAccessToken{})

	return err
}

func (s *Service) AuthenticateAccessToken(token string) (*models.PersonalAccessToken, *models.User, error) {
	if !strings.HasPrefix(token, accessTokenPrefix) {
		return nil, nil, errors.New("account: not an access token")
	}

	var record models.PersonalAccessToken
	if err := facades.Orm().Query().Where("token_hash", crypto.HashToken(token)).FirstOrFail(&record); err != nil {
		return nil, nil, errors.New("account: access token not found")
	}

	if record.ExpiresAt != nil && time.Now().After(*record.ExpiresAt) {
		_, _ = facades.Orm().Query().Delete(&record)

		return nil, nil, errors.New("account: access token expired")
	}

	user, err := s.UserByID(record.UserID)
	if err != nil {
		return nil, nil, err
	}

	return &record, user, nil
}

// TouchAccessToken records token usage, throttled to one write per minute.
func (s *Service) TouchAccessToken(record *models.PersonalAccessToken) {
	if record == nil || record.ID == 0 {
		return
	}

	if last, ok := lastTokenTouch.Load(record.ID); ok {
		if time.Since(last.(time.Time)) < time.Minute {
			return
		}
	}
	lastTokenTouch.Store(record.ID, time.Now())

	now := time.Now().UTC()
	record.LastUsedAt = &now
	_ = facades.Orm().Query().Save(record)
}
