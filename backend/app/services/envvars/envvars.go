package envvars

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"

	"ebox/app/facades"
	"ebox/app/models"
	"ebox/app/services/crypto"
)

var keyPattern = regexp.MustCompile(`^[A-Za-z_][A-Za-z0-9_]*$`)

type Pair struct {
	Key   string
	Value string
}

type Service struct{}

func Default() *Service {
	return &Service{}
}

func (s *Service) List(resourceType string, resourceID uint64) ([]models.EnvVar, error) {
	var list []models.EnvVar
	if err := facades.Orm().Query().
		Where("resource_type", resourceType).
		Where("resource_id", resourceID).
		Order("key asc").
		Find(&list); err != nil {
		return nil, err
	}

	for index := range list {
		value, err := crypto.Decrypt(list[index].Value)
		if err != nil {
			return nil, err
		}
		list[index].Value = value
	}

	return list, nil
}

// Values returns decrypted env vars for a resource. When buildTimeOnly is set,
// only variables marked as build-time are returned.
func (s *Service) Values(resourceType string, resourceID uint64, buildTimeOnly bool) (map[string]string, error) {
	query := facades.Orm().Query().Where("resource_type", resourceType).Where("resource_id", resourceID)
	if buildTimeOnly {
		query = query.Where("is_build_time", true)
	}

	var list []models.EnvVar
	if err := query.Find(&list); err != nil {
		return nil, err
	}

	values := make(map[string]string, len(list))
	for _, item := range list {
		value, err := crypto.Decrypt(item.Value)
		if err != nil {
			return nil, err
		}
		values[item.Key] = value
	}

	return values, nil
}

// Sorted returns the env vars as a deterministic slice.
func Sorted(values map[string]string) []Pair {
	keys := make([]string, 0, len(values))
	for key := range values {
		keys = append(keys, key)
	}
	sort.Strings(keys)

	pairs := make([]Pair, 0, len(keys))
	for _, key := range keys {
		pairs = append(pairs, Pair{Key: key, Value: values[key]})
	}

	return pairs
}

// Set creates or updates a single env var.
func (s *Service) Set(resourceType string, resourceID uint64, key, value string, isBuildTime bool) (*models.EnvVar, error) {
	key = strings.TrimSpace(key)
	if !keyPattern.MatchString(key) {
		return nil, fmt.Errorf("env: %q is not a valid variable name", key)
	}

	encrypted, err := crypto.Encrypt(value)
	if err != nil {
		return nil, err
	}

	var existing models.EnvVar
	query := facades.Orm().Query().
		Where("resource_type", resourceType).
		Where("resource_id", resourceID).
		Where("key", key)

	if err := query.FirstOrFail(&existing); err != nil {
		created := &models.EnvVar{
			ResourceType: resourceType,
			ResourceID:   resourceID,
			Key:          key,
			Value:        encrypted,
			IsBuildTime:  isBuildTime,
		}
		if err := facades.Orm().Query().Create(created); err != nil {
			return nil, err
		}
		created.Value = value

		return created, nil
	}

	existing.Value = encrypted
	existing.IsBuildTime = isBuildTime
	if err := facades.Orm().Query().Save(&existing); err != nil {
		return nil, err
	}
	existing.Value = value

	return &existing, nil
}

func (s *Service) Update(resourceType string, resourceID uint64, id uint64, key, value string, isBuildTime bool) (*models.EnvVar, error) {
	var record models.EnvVar
	if err := facades.Orm().Query().
		Where("resource_type", resourceType).
		Where("resource_id", resourceID).
		Where("id", id).
		FirstOrFail(&record); err != nil {
		return nil, err
	}

	updated, err := s.Set(resourceType, resourceID, key, value, isBuildTime)
	if err != nil {
		return nil, err
	}

	if updated.ID != record.ID {
		_, _ = facades.Orm().Query().Delete(&record)
	}

	return updated, nil
}

func (s *Service) Delete(resourceType string, resourceID uint64, id uint64) error {
	_, err := facades.Orm().Query().
		Where("resource_type", resourceType).
		Where("resource_id", resourceID).
		Where("id", id).
		Delete(&models.EnvVar{})

	return err
}

// Import parses a .env payload and upserts every pair.
func (s *Service) Import(resourceType string, resourceID uint64, content string) (int, error) {
	pairs, err := ParseDotEnv(content)
	if err != nil {
		return 0, err
	}

	for _, pair := range pairs {
		if _, err := s.Set(resourceType, resourceID, pair.Key, pair.Value, false); err != nil {
			return 0, err
		}
	}

	return len(pairs), nil
}

// Export renders the env vars as a .env file.
func (s *Service) Export(resourceType string, resourceID uint64) (string, error) {
	values, err := s.Values(resourceType, resourceID, false)
	if err != nil {
		return "", err
	}

	var builder strings.Builder
	for _, pair := range Sorted(values) {
		builder.WriteString(pair.Key + "=" + quoteValue(pair.Value) + "\n")
	}

	return builder.String(), nil
}

// WriteEnvFile stores the runtime environment next to the application with
// 0600 permissions.
func (s *Service) WriteEnvFile(path string, values map[string]string) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}

	var builder strings.Builder
	for _, pair := range Sorted(values) {
		builder.WriteString(pair.Key + "=" + quoteValue(pair.Value) + "\n")
	}

	return os.WriteFile(path, []byte(builder.String()), 0o600)
}

// ParseDotEnv parses KEY=value lines, ignoring comments and blank lines.
func ParseDotEnv(content string) ([]Pair, error) {
	var pairs []Pair

	content = strings.ReplaceAll(content, "\r\n", "\n")

	for _, raw := range strings.Split(content, "\n") {
		line := strings.TrimSpace(raw)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}

		line = strings.TrimPrefix(line, "export ")

		key, value, found := strings.Cut(line, "=")
		if !found {
			return nil, fmt.Errorf("env: line %q is not KEY=value", raw)
		}

		key = strings.TrimSpace(key)
		if !keyPattern.MatchString(key) {
			return nil, fmt.Errorf("env: %q is not a valid variable name", key)
		}

		pairs = append(pairs, Pair{Key: key, Value: unquoteValue(strings.TrimSpace(value))})
	}

	return pairs, nil
}

func unquoteValue(value string) string {
	if len(value) < 2 {
		return value
	}

	first := value[0]
	last := value[len(value)-1]
	if (first != '"' && first != '\'') || first != last {
		return value
	}

	inner := value[1 : len(value)-1]
	if first == '\'' {
		return inner
	}

	replacer := strings.NewReplacer(`\n`, "\n", `\r`, "\r", `\t`, "\t", `\"`, `"`, `\\`, `\`)

	return replacer.Replace(inner)
}

func quoteValue(value string) string {
	if value == "" {
		return `""`
	}

	if strings.ContainsAny(value, " \t\n\r\"'#$`\\") {
		escaped := strings.NewReplacer(`\`, `\\`, `"`, `\"`, "\n", `\n`, "\r", `\r`, "\t", `\t`).Replace(value)

		return `"` + escaped + `"`
	}

	return value
}
