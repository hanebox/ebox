package config

import (
	"os"
	"path/filepath"

	"github.com/goravel/framework/support/path"

	"ebox/app/facades"
)

// eboxDataDir is also used by the database config, which runs earlier in the
// Go init order, so it must not depend on registered configuration values.
func eboxDataDir() string {
	if dir := os.Getenv("EBOX_DATA_DIR"); dir != "" {
		return dir
	}

	return path.Storage("ebox")
}

func init() {
	config := facades.Config()
	dataDir := eboxDataDir()

	config.Add("ebox", map[string]any{
		"data_dir":     dataDir,
		"apps_dir":     filepath.Join(dataDir, "apps"),
		"services_dir": filepath.Join(dataDir, "services"),
		"volumes_dir":  filepath.Join(dataDir, "volumes"),
		"backups_dir":  filepath.Join(dataDir, "backups"),
		"traefik_dir":  config.EnvString("EBOX_TRAEFIK_DIR", path.Storage("ebox/traefik")),

		"traefik": map[string]any{
			"api_port": config.EnvString("EBOX_TRAEFIK_API_PORT", "8080"),
			"api_url":  config.EnvString("EBOX_TRAEFIK_API", ""),
		},
		"host": config.EnvString("EBOX_HOST", "127.0.0.1"),
		"port": config.EnvString("EBOX_PORT", "8000"),

		"docker": map[string]any{
			"network": config.EnvString("EBOX_DOCKER_NETWORK", "ebox"),
		},

		"deploy": map[string]any{
			"connection":         config.EnvString("EBOX_QUEUE_CONNECTION", "database"),
			"queue":              "deployments",
			"retain_images":      5,
			"retain_deployments": 50,
		},
	})
}
