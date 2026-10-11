package build

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"ebox/app/models"
)

// StaticCaddyfile serves the published directory over port 80 with a SPA
// fallback.
const StaticCaddyfile = `:80 {
	root * /srv
	encode gzip
	try_files {path} /index.html
	file_server
}
`

// StaticDockerfile generates the multi-stage Dockerfile used by the static
// build pack.
func StaticDockerfile(app *models.Application, contextDir string) (string, error) {
	command := strings.TrimSpace(app.Config.BuildCommand)
	base := "node:22-alpine"

	if command == "" {
		detectedBase, detectedCommand, err := detectNodeBuild(contextDir)
		if err != nil {
			return "", err
		}

		base = detectedBase
		command = detectedCommand
	}

	publish := strings.TrimPrefix(strings.TrimSpace(app.Config.PublishDirectory), "./")
	publish = strings.Trim(publish, "/")
	if publish == "" {
		publish = "dist"
	}

	return fmt.Sprintf(`FROM %s AS build
WORKDIR /app
COPY . .
RUN %s

FROM caddy:2-alpine
COPY --from=build /app/%s /srv
COPY ebox.Caddyfile /etc/caddy/Caddyfile
EXPOSE 80
`, base, command, publish), nil
}

func detectNodeBuild(dir string) (string, string, error) {
	switch {
	case exists(dir, "bun.lockb"), exists(dir, "bun.lock"):
		return "oven/bun:1-alpine", "bun install --frozen-lockfile && bun run build", nil
	case exists(dir, "pnpm-lock.yaml"):
		return "node:22-alpine", "corepack enable && pnpm install --frozen-lockfile && pnpm run build", nil
	case exists(dir, "yarn.lock"):
		return "node:22-alpine", "corepack enable && yarn install --immutable && yarn run build", nil
	case exists(dir, "package-lock.json"):
		return "node:22-alpine", "npm ci && npm run build", nil
	case exists(dir, "package.json"):
		return "node:22-alpine", "npm install && npm run build", nil
	}

	return "", "", errors.New("build: no package.json found - set a build command for this static site")
}

func exists(dir string, name string) bool {
	_, err := os.Stat(filepath.Join(dir, name))

	return err == nil
}
