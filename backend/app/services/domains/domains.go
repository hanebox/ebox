package domains

import (
	"fmt"
	"net/url"
	"regexp"
	"strings"

	"ebox/app/facades"
	"ebox/app/models"
)

var hostPattern = regexp.MustCompile(`^[a-z0-9]([a-z0-9-]*[a-z0-9])?(\.[a-z0-9]([a-z0-9-]*[a-z0-9])?)*$`)

type Service struct{}

func Default() *Service {
	return &Service{}
}

func (s *Service) List(resourceType string, resourceID uint64) ([]models.Domain, error) {
	var list []models.Domain
	if err := facades.Orm().Query().
		Where("resource_type", resourceType).
		Where("resource_id", resourceID).
		Order("is_primary desc").
		Order("host asc").
		Find(&list); err != nil {
		return nil, err
	}

	return list, nil
}

func (s *Service) Find(resourceType string, resourceID uint64, id uint64) (*models.Domain, error) {
	var domain models.Domain
	if err := facades.Orm().Query().
		Where("resource_type", resourceType).
		Where("resource_id", resourceID).
		Where("id", id).
		FirstOrFail(&domain); err != nil {
		return nil, err
	}

	return &domain, nil
}

func (s *Service) Add(resourceType string, resourceID uint64, host string, https bool, isPrimary bool) (*models.Domain, error) {
	normalized, err := NormalizeHost(host)
	if err != nil {
		return nil, err
	}

	if taken, err := s.ExistsElsewhere(normalized, 0); err != nil {
		return nil, err
	} else if taken {
		return nil, fmt.Errorf("domains: %s is already used by another resource", normalized)
	}

	domain := &models.Domain{
		ResourceType: resourceType,
		ResourceID:   resourceID,
		Host:         normalized,
		Port:         80,
		HTTPS:        https,
		IsPrimary:    isPrimary,
	}

	if err := facades.Orm().Query().Create(domain); err != nil {
		return nil, err
	}

	if isPrimary {
		if err := s.clearOtherPrimaries(resourceType, resourceID, domain.ID); err != nil {
			return nil, err
		}
	}

	return domain, nil
}

func (s *Service) Update(domain *models.Domain, host string, https bool, isPrimary bool) error {
	normalized, err := NormalizeHost(host)
	if err != nil {
		return err
	}

	if taken, err := s.ExistsElsewhere(normalized, domain.ID); err != nil {
		return err
	} else if taken {
		return fmt.Errorf("domains: %s is already used by another resource", normalized)
	}

	domain.Host = normalized
	domain.HTTPS = https
	domain.IsPrimary = isPrimary

	if err := facades.Orm().Query().Save(domain); err != nil {
		return err
	}

	if isPrimary {
		return s.clearOtherPrimaries(domain.ResourceType, domain.ResourceID, domain.ID)
	}

	return nil
}

func (s *Service) Delete(domain *models.Domain) error {
	_, err := facades.Orm().Query().Delete(domain)

	return err
}

func (s *Service) ExistsElsewhere(host string, ignoreID uint64) (bool, error) {
	query := facades.Orm().Query().Model(&models.Domain{}).Where("host", host)
	if ignoreID > 0 {
		query = query.Where("id <> ?", ignoreID)
	}

	count, err := query.Count()
	if err != nil {
		return false, err
	}

	return count > 0, nil
}

func (s *Service) clearOtherPrimaries(resourceType string, resourceID uint64, keepID uint64) error {
	_, err := facades.Orm().Query().
		Model(&models.Domain{}).
		Where("resource_type", resourceType).
		Where("resource_id", resourceID).
		Where("id <> ?", keepID).
		Update("is_primary", false)

	return err
}

// NormalizeHost accepts a hostname or URL and returns a bare hostname.
func NormalizeHost(value string) (string, error) {
	value = strings.TrimSpace(strings.ToLower(value))
	if value == "" {
		return "", fmt.Errorf("domains: the domain is required")
	}

	if strings.Contains(value, "://") {
		parsed, err := url.Parse(value)
		if err != nil {
			return "", fmt.Errorf("domains: %q is not a valid domain", value)
		}
		value = parsed.Host
	}

	if index := strings.IndexAny(value, "/?#"); index >= 0 {
		value = value[:index]
	}

	value = strings.TrimSuffix(value, ".")
	if strings.Contains(value, ":") {
		host, _, found := strings.Cut(value, ":")
		if found {
			value = host
		}
	}

	if !hostPattern.MatchString(value) {
		return "", fmt.Errorf("domains: %q is not a valid domain", value)
	}

	return value, nil
}
