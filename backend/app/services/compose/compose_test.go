package compose

import (
	"testing"

	"github.com/stretchr/testify/require"
)

const sampleCompose = `
services:
  web:
    build: .
    ports:
      - "3000:3000"
    environment:
      NODE_ENV: production
    depends_on:
      - db
  db:
    image: postgres:17-alpine
    environment:
      POSTGRES_PASSWORD: secret
    volumes:
      - data:/var/lib/postgresql/data
  worker:
    build:
      context: .
      dockerfile: Dockerfile.worker
    depends_on:
      db:
        condition: service_healthy
volumes:
  data:
`

func TestParseSummarisesServices(t *testing.T) {
	document, err := Parse(sampleCompose)
	require.NoError(t, err)

	require.Equal(t, []string{"db", "web", "worker"}, document.ServiceNames())

	info := document.Info()
	require.Len(t, info, 3)

	byName := map[string]Info{}
	for _, item := range info {
		byName[item.Name] = item
	}

	require.Equal(t, []string{"3000:3000"}, byName["web"].Ports)
	require.False(t, byName["web"].Built == false, "web uses build")
	require.Equal(t, []string{"db"}, byName["web"].DependsOn)
	require.Equal(t, "postgres:17-alpine", byName["db"].Image)
	require.True(t, byName["db"].HasVolumes)
	require.Equal(t, []string{"db"}, byName["worker"].DependsOn)
	require.True(t, byName["worker"].Built)
}

func TestParseRejectsInvalidInput(t *testing.T) {
	_, err := Parse("")
	require.Error(t, err)

	_, err = Parse("services: {}")
	require.Error(t, err)

	_, err = Parse("not: [valid")
	require.Error(t, err)
}

func TestPortNotations(t *testing.T) {
	document, err := Parse(`
services:
  a:
    image: nginx
    ports:
      - 8080
  b:
    image: nginx
    ports:
      - "127.0.0.1:9000:80"
  c:
    image: nginx
    ports:
      - target: 5000
        published: "5001"
        protocol: tcp
  d:
    image: nginx
    expose:
      - "7000"
`)
	require.NoError(t, err)

	require.Equal(t, 0, document.Services["a"].PublishedPort())
	require.Equal(t, 8080, document.Services["a"].ContainerPort())

	require.Equal(t, 9000, document.Services["b"].PublishedPort())
	require.Equal(t, 80, document.Services["b"].ContainerPort())

	require.Equal(t, 5001, document.Services["c"].PublishedPort())
	require.Equal(t, 5000, document.Services["c"].ContainerPort())

	require.Equal(t, 7000, document.Services["d"].ContainerPort())
}

func TestEntryCandidatesPrefersPublishedPorts(t *testing.T) {
	document, err := Parse(sampleCompose)
	require.NoError(t, err)

	candidates := document.EntryCandidates()
	require.NotEmpty(t, candidates)

	// worker and db come first alphabetically but web publishes a port.
	require.Equal(t, "web", candidates[0].Name)
}

func TestRenderOverrideAppliesLabelsNetworkAndEntry(t *testing.T) {
	document, err := Parse(sampleCompose)
	require.NoError(t, err)

	override, err := Render(document, Override{
		Network:      "ebox",
		Labels:       map[string]string{"ebox.managed": "true", "ebox.resource-type": "service"},
		Environment:  map[string]string{"GLOBAL": "1"},
		EntryService: "web",
		EntryPort:    3000,
		EntryLabels:  map[string]string{"traefik.enable": "true"},
		EntryEnv:     map[string]string{"APP_URL": "https://example.com"},
	})
	require.NoError(t, err)

	parsed, err := Parse(override)
	require.NoError(t, err)
	require.Equal(t, []string{"db", "web", "worker"}, parsed.ServiceNames())

	require.Contains(t, override, "ebox.managed: \"true\"")
	require.Contains(t, override, "ebox.service: web")
	require.Contains(t, override, "ebox.service: db")
	require.Contains(t, override, "traefik.enable: \"true\"")
	require.Contains(t, override, "external: true")

	web := parsed.Services["web"]
	db := parsed.Services["db"]

	environment, ok := web.Environment.(map[string]any)
	require.True(t, ok)
	require.Equal(t, "1", environment["GLOBAL"])
	require.Equal(t, "https://example.com", environment["APP_URL"])

	dbEnvironment, ok := db.Environment.(map[string]any)
	require.True(t, ok)
	require.Equal(t, "1", dbEnvironment["GLOBAL"])
	require.NotContains(t, dbEnvironment, "APP_URL")

	// Every service joins the shared network.
	require.Equal(t, []any{"ebox"}, web.Networks)
	require.Equal(t, []any{"ebox"}, db.Networks)
}

func TestRenderOverrideWithoutEntryService(t *testing.T) {
	document, err := Parse("services:\n  api:\n    image: nginx\n")
	require.NoError(t, err)

	override, err := Render(document, Override{Labels: map[string]string{"ebox.managed": "true"}})
	require.NoError(t, err)

	require.Contains(t, override, "external: true")
	require.Contains(t, override, "ebox.service: api")
	require.NotContains(t, override, "traefik.enable")
}

func TestServiceContainerName(t *testing.T) {
	require.Equal(t, "ebox-abc12345-web-1", ServiceContainerName("ebox-abc12345", "web"))
	require.Equal(t, []string{"a", "b"}, SortedLabels(map[string]string{"b": "1", "a": "2"}))
}

func TestParseKeepsUnknownKeys(t *testing.T) {
	document, err := Parse(`
services:
  web:
    image: nginx
    deploy:
      replicas: 2
    healthcheck:
      test: ["CMD", "curl", "-f", "http://localhost"]
`)
	require.NoError(t, err)
	require.Equal(t, "nginx", document.Services["web"].Image)
	require.NotNil(t, document.Services["web"].Healthcheck)
	keys := []string{}
	for key := range document.Services["web"].Extra {
		keys = append(keys, key)
	}
	require.Contains(t, keys, "deploy")
}
