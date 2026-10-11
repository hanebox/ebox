package build

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"

	"ebox/app/models"
)

func applicationWithConfig(config models.ApplicationConfig) *models.Application {
	return &models.Application{
		UUID:      "11111111-2222-3333-4444-555555555555",
		Name:      "site",
		BuildPack: models.BuildPackStatic,
		Config:    config,
	}
}

func TestStaticDockerfileUsesConfiguredCommand(t *testing.T) {
	dir := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(dir, "package.json"), []byte("{}"), 0o644))

	dockerfile, err := StaticDockerfile(applicationWithConfig(models.ApplicationConfig{
		BuildCommand:     "pnpm run build",
		PublishDirectory: "build",
	}), dir)
	require.NoError(t, err)

	require.Contains(t, dockerfile, "FROM node:22-alpine AS build")
	require.Contains(t, dockerfile, "RUN pnpm run build")
	require.Contains(t, dockerfile, "COPY --from=build /app/build /srv")
	require.Contains(t, dockerfile, "FROM caddy:2-alpine")
	require.Contains(t, dockerfile, "COPY ebox.Caddyfile /etc/caddy/Caddyfile")
	require.NotContains(t, dockerfile, "/app/dist")
}

func TestStaticDockerfileDefaults(t *testing.T) {
	dir := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(dir, "package.json"), []byte("{}"), 0o644))

	dockerfile, err := StaticDockerfile(applicationWithConfig(models.ApplicationConfig{}), dir)
	require.NoError(t, err)

	require.Contains(t, dockerfile, "RUN npm install && npm run build")
	require.Contains(t, dockerfile, "COPY --from=build /app/dist /srv")
}

func TestStaticDockerfileDetectsPackageManager(t *testing.T) {
	cases := []struct {
		lockfile string
		image    string
		command  string
	}{
		{"bun.lockb", "oven/bun:1-alpine", "bun install --frozen-lockfile && bun run build"},
		{"pnpm-lock.yaml", "node:22-alpine", "corepack enable && pnpm install --frozen-lockfile && pnpm run build"},
		{"yarn.lock", "node:22-alpine", "corepack enable && yarn install --immutable && yarn run build"},
		{"package-lock.json", "node:22-alpine", "npm ci && npm run build"},
	}

	for _, testCase := range cases {
		dir := t.TempDir()
		require.NoError(t, os.WriteFile(filepath.Join(dir, testCase.lockfile), []byte("x"), 0o644))

		dockerfile, err := StaticDockerfile(applicationWithConfig(models.ApplicationConfig{}), dir)
		require.NoError(t, err, "lockfile %s", testCase.lockfile)

		require.Contains(t, dockerfile, "FROM "+testCase.image+" AS build", "lockfile %s", testCase.lockfile)
		require.Contains(t, dockerfile, "RUN "+testCase.command, "lockfile %s", testCase.lockfile)
	}
}

func TestStaticDockerfileRequiresBuildTarget(t *testing.T) {
	_, err := StaticDockerfile(applicationWithConfig(models.ApplicationConfig{}), t.TempDir())
	require.Error(t, err)
	require.Contains(t, err.Error(), "build command")
}

func TestStaticCaddyfileHasSPAFallback(t *testing.T) {
	require.Contains(t, StaticCaddyfile, "try_files {path} /index.html")
	require.Contains(t, StaticCaddyfile, "root * /srv")
	require.Contains(t, StaticCaddyfile, ":80 {")
}
