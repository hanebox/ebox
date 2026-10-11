package settings

import (
	"errors"
	"sync"

	frameworkerrors "github.com/goravel/framework/errors"

	"ebox/app/facades"
	"ebox/app/models"
)

type Service struct {
	mu     sync.RWMutex
	cache  map[string]string
	loaded bool
}

var defaultService = &Service{}

func Default() *Service {
	return defaultService
}

func (s *Service) All() (map[string]string, error) {
	s.mu.RLock()
	if s.loaded {
		out := make(map[string]string, len(s.cache))
		for key, value := range s.cache {
			out[key] = value
		}
		s.mu.RUnlock()

		return out, nil
	}
	s.mu.RUnlock()

	var rows []models.Setting
	if err := facades.Orm().Query().Find(&rows); err != nil {
		return nil, err
	}

	values := make(map[string]string, len(rows))
	for _, row := range rows {
		values[row.Key] = row.Value
	}

	s.mu.Lock()
	s.cache = values
	s.loaded = true
	s.mu.Unlock()

	return values, nil
}

func (s *Service) Get(key, fallback string) string {
	values, err := s.All()
	if err != nil {
		return fallback
	}

	if value, ok := values[key]; ok && value != "" {
		return value
	}

	return fallback
}

func (s *Service) Set(key, value string) error {
	return s.SetMany(map[string]string{key: value})
}

func (s *Service) SetMany(values map[string]string) error {
	for key, value := range values {
		var existing models.Setting
		err := facades.Orm().Query().Where("key", key).FirstOrFail(&existing)
		if err == nil {
			existing.Value = value
			if err := facades.Orm().Query().Save(&existing); err != nil {
				return err
			}
			continue
		}
		if !errors.Is(err, frameworkerrors.OrmRecordNotFound) {
			return err
		}

		if err := facades.Orm().Query().Create(&models.Setting{Key: key, Value: value}); err != nil {
			return err
		}
	}

	s.mu.Lock()
	if s.loaded {
		for key, value := range values {
			s.cache[key] = value
		}
	}
	s.mu.Unlock()

	return nil
}

func (s *Service) Invalidate() {
	s.mu.Lock()
	s.loaded = false
	s.cache = nil
	s.mu.Unlock()
}
