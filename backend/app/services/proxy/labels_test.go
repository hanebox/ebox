package proxy

import (
	"testing"

	"github.com/stretchr/testify/require"

	"ebox/app/models"
)

func testApplication() *models.Application {
	return &models.Application{
		UUID:      "11111111-2222-3333-4444-555555555555",
		Name:      "hello",
		BuildPack: models.BuildPackDockerfile,
		Config:    models.ApplicationConfig{ExposePort: 8080},
	}
}

func TestAppLabelsWithoutDomains(t *testing.T) {
	labels := AppLabels(testApplication(), nil, "ebox")

	require.Equal(t, "false", labels["traefik.enable"])
	require.Equal(t, "true", labels["ebox.managed"])
	require.Equal(t, "application", labels["ebox.resource-type"])
	require.Equal(t, "11111111-2222-3333-4444-555555555555", labels["ebox.resource-id"])
	require.NotContains(t, labels, "traefik.http.services.ebox-11111111.loadbalancer.server.port")
}

func TestAppLabelsWithHTTPSDomain(t *testing.T) {
	app := testApplication()
	labels := AppLabels(app, []models.Domain{
		{Host: "app.example.com", HTTPS: true, Port: 80},
	}, "ebox")

	require.Equal(t, "true", labels["traefik.enable"])
	require.Equal(t, "ebox", labels["traefik.docker.network"])
	require.Equal(t, "8080", labels["traefik.http.services.ebox-11111111.loadbalancer.server.port"])
	require.Equal(t, "http", labels["traefik.http.services.ebox-11111111.loadbalancer.server.scheme"])

	require.Equal(t, "Host(`app.example.com`)", labels["traefik.http.routers.ebox-11111111-0.rule"])
	require.Equal(t, "web", labels["traefik.http.routers.ebox-11111111-0.entrypoints"])
	require.Equal(t, "ebox-11111111", labels["traefik.http.routers.ebox-11111111-0.service"])
	require.Equal(t, "ebox-11111111-redirect", labels["traefik.http.routers.ebox-11111111-0.middlewares"])

	require.Equal(t, "Host(`app.example.com`)", labels["traefik.http.routers.ebox-11111111-0-secure.rule"])
	require.Equal(t, "websecure", labels["traefik.http.routers.ebox-11111111-0-secure.entrypoints"])
	require.Equal(t, "letsencrypt", labels["traefik.http.routers.ebox-11111111-0-secure.tls.certresolver"])

	require.Equal(t, "https", labels["traefik.http.middlewares.ebox-11111111-redirect.redirectscheme.scheme"])
	require.Equal(t, "true", labels["traefik.http.middlewares.ebox-11111111-redirect.redirectscheme.permanent"])
}

func TestAppLabelsWithPlainHTTPDomain(t *testing.T) {
	labels := AppLabels(testApplication(), []models.Domain{
		{Host: "app.example.com", HTTPS: false},
	}, "ebox")

	require.Equal(t, "web", labels["traefik.http.routers.ebox-11111111-0.entrypoints"])
	require.NotContains(t, labels, "traefik.http.routers.ebox-11111111-0.middlewares")
	require.NotContains(t, labels, "traefik.http.routers.ebox-11111111-0-secure.rule")
	require.NotContains(t, labels, "traefik.http.middlewares.ebox-11111111-redirect.redirectscheme.scheme")
}

func TestAppLabelsAreDeterministicForMultipleDomains(t *testing.T) {
	domains := []models.Domain{
		{Host: "z.example.com", HTTPS: true},
		{Host: "a.example.com", HTTPS: false},
	}

	first := AppLabels(testApplication(), domains, "ebox")
	second := AppLabels(testApplication(), []models.Domain{domains[1], domains[0]}, "ebox")

	require.Equal(t, first, second)
	require.Equal(t, "Host(`a.example.com`)", first["traefik.http.routers.ebox-11111111-0.rule"])
	require.Equal(t, "Host(`z.example.com`)", first["traefik.http.routers.ebox-11111111-1.rule"])
	require.Equal(t, "websecure", first["traefik.http.routers.ebox-11111111-1-secure.entrypoints"])
}
