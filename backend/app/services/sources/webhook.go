package sources

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"errors"

	"ebox/app/models"
)

// VerifyWebhook matches a GitHub webhook payload against the stored webhook
// secrets and returns the source that signed it.
func (s *Service) VerifyWebhook(body []byte, signature string) (*models.Source, error) {
	if signature == "" {
		return nil, errors.New("sources: missing webhook signature")
	}

	list, err := s.List()
	if err != nil {
		return nil, err
	}

	for index := range list {
		if list[index].Type != models.SourceTypeGithubApp {
			continue
		}

		secret, err := s.WebhookSecret(&list[index])
		if err != nil || secret == "" {
			continue
		}

		mac := hmac.New(sha256.New, []byte(secret))
		mac.Write(body)
		expected := "sha256=" + hex.EncodeToString(mac.Sum(nil))

		if hmac.Equal([]byte(expected), []byte(signature)) {
			return &list[index], nil
		}
	}

	return nil, errors.New("sources: no source matches the webhook signature")
}
