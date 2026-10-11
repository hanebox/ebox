package feature

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"

	"ebox/app/facades"
	"ebox/app/models"
	"ebox/app/services/compose"
	"ebox/app/services/domains"
	"ebox/app/services/stacks"
	_ "ebox/tests"
)

const stackCompose = `services:
  web:
    image: nginx:alpine
    ports:
      - "8080:80"
  worker:
    image: alpine:3.20
    command: ["sleep", "infinity"]
`

// TestStackProjectRendersOverride checks the generated compose override for a
// stack without touching Docker: labels, network, ingress and domains.
func TestStackProjectRendersOverride(t *testing.T) {
	require.NoError(t, facades.Artisan().Call("migrate"))

	project := models.Project{UUID: "stack-test-project", Name: "Stack test"}
	require.NoError(t, facades.Orm().Query().Create(&project))
	t.Cleanup(func() { _, _ = facades.Orm().Query().Where("id", project.ID).Delete(&models.Project{}) })

	service := &models.Service{
		UUID:        "stack-test-service-uuid-with-length",
		ProjectID:   project.ID,
		Name:        "stack-test",
		SourceType:  models.ServiceSourceInline,
		Compose:     stackCompose,
		EntryPort:   0,
		Environment: "APP_ENV=testing\n",
	}
	require.NoError(t, facades.Orm().Query().Create(service))
	t.Cleanup(func() {
		_, _ = facades.Orm().Query().Where("id", service.ID).Delete(&models.Service{})
		_ = os.RemoveAll(stacks.Default().Directory(service))
	})

	_, err := domains.Default().Add(models.ResourceTypeService, service.ID, "stack.test.example", false, true)
	require.NoError(t, err)
	t.Cleanup(func() {
		_, _ = facades.Orm().Query().
			Where("resource_type", models.ResourceTypeService).
			Where("resource_id", service.ID).
			Delete(&models.Domain{})
	})

	rendered, err := stacks.Default().Project(context.Background(), service, false)
	require.NoError(t, err)
	require.Len(t, rendered.Files, 2)

	override, err := os.ReadFile(rendered.Files[1])
	require.NoError(t, err)
	content := string(override)

	require.Contains(t, content, "ebox.managed: \"true\"")
	require.Contains(t, content, "ebox.resource-type: service")
	require.Contains(t, content, "ebox.resource-id: "+service.UUID)
	require.Contains(t, content, "ebox.service: web")
	require.Contains(t, content, "ebox.service: worker")
	require.Contains(t, content, "external: true")
	require.Contains(t, content, "traefik.enable: \"true\"")
	require.Contains(t, content, "Host(`stack.test.example`)")
	require.Contains(t, content, "loadbalancer.server.port: \"80\"")

	// The entry service is the one publishing a port.
	entry, port, err := stacks.Default().EntryService(context.Background(), service)
	require.NoError(t, err)
	require.Equal(t, "web", entry)
	require.Equal(t, 80, port)

	// The compose file and env file are on disk with strict permissions.
	base, err := os.ReadFile(rendered.Files[0])
	require.NoError(t, err)
	require.Equal(t, stackCompose, string(base))

	env, err := os.ReadFile(rendered.EnvFile)
	require.NoError(t, err)
	require.Contains(t, string(env), "APP_ENV=testing")

	info, err := os.Stat(rendered.EnvFile)
	require.NoError(t, err)
	require.Equal(t, os.FileMode(0o600), info.Mode().Perm())

	// The override must be valid compose the CLI can merge.
	_, err = compose.Parse(content)
	require.NoError(t, err)

	directory := stacks.Default().Directory(service)
	require.Equal(t, filepath.Join(facades.Config().GetString("ebox.services_dir"), service.UUID), directory)
	require.True(t, strings.HasPrefix(rendered.Files[0], directory))
}

func TestStackEntryPortOverride(t *testing.T) {
	require.NoError(t, facades.Artisan().Call("migrate"))

	project := models.Project{UUID: "stack-port-project", Name: "Stack port"}
	require.NoError(t, facades.Orm().Query().Create(&project))
	t.Cleanup(func() { _, _ = facades.Orm().Query().Where("id", project.ID).Delete(&models.Project{}) })

	service := &models.Service{
		UUID:         "stack-port-service-uuid-with-leng",
		ProjectID:    project.ID,
		Name:         "stack-port",
		SourceType:   models.ServiceSourceInline,
		Compose:      stackCompose,
		EntryPort:    9000,
		EntryService: "worker",
	}
	require.NoError(t, facades.Orm().Query().Create(service))
	t.Cleanup(func() {
		_, _ = facades.Orm().Query().Where("id", service.ID).Delete(&models.Service{})
		_ = os.RemoveAll(stacks.Default().Directory(service))
	})

	_, err := domains.Default().Add(models.ResourceTypeService, service.ID, "worker.test.example", false, true)
	require.NoError(t, err)
	t.Cleanup(func() {
		_, _ = facades.Orm().Query().
			Where("resource_type", models.ResourceTypeService).
			Where("resource_id", service.ID).
			Delete(&models.Domain{})
	})

	entry, port, err := stacks.Default().EntryService(context.Background(), service)
	require.NoError(t, err)
	require.Equal(t, "worker", entry)
	require.Equal(t, 9000, port)

	rendered, err := stacks.Default().Project(context.Background(), service, false)
	require.NoError(t, err)

	override, err := os.ReadFile(rendered.Files[1])
	require.NoError(t, err)
	require.Contains(t, string(override), "loadbalancer.server.port: \"9000\"")
}
