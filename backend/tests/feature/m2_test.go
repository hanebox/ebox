package feature

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"

	"ebox/app/facades"
	"ebox/app/models"
	"ebox/app/services/applications"
	"ebox/app/services/deployment"
	"ebox/app/services/envvars"
	"ebox/app/services/projects"
	"ebox/app/services/sources"
	_ "ebox/tests"
)

func createGithubSource(t *testing.T, webhookSecret string) *models.Source {
	t.Helper()

	require.NoError(t, facades.Artisan().Call("migrate"))

	source, err := sources.Default().CreateGithubApp(sources.GithubAppInput{
		Name:          "hook-" + uuid.NewString()[:8],
		AppID:         424242,
		AppSlug:       "ebox-hook-test",
		PrivateKey:    "-----BEGIN RSA PRIVATE KEY-----\nMIIfake\n-----END RSA PRIVATE KEY-----",
		WebhookSecret: webhookSecret,
	})
	require.NoError(t, err)
	t.Cleanup(func() { _ = sources.Default().Delete(source) })

	return source
}

func createProject(t *testing.T) *models.Project {
	t.Helper()

	require.NoError(t, facades.Artisan().Call("migrate"))

	project, err := projects.Default().Create("test-"+uuid.NewString()[:8], "created by tests")
	require.NoError(t, err)
	t.Cleanup(func() { _ = projects.Default().Delete(project) })

	return project
}

func TestVerifyWebhookMatchesSignedPayload(t *testing.T) {
	source := createGithubSource(t, "top-secret")

	body := []byte(`{"ref":"refs/heads/main","repository":{"full_name":"acme/app"}}`)

	mac := hmac.New(sha256.New, []byte("top-secret"))
	mac.Write(body)
	signature := "sha256=" + hex.EncodeToString(mac.Sum(nil))

	matched, err := sources.Default().VerifyWebhook(body, signature)
	require.NoError(t, err)
	require.Equal(t, source.UUID, matched.UUID)

	_, err = sources.Default().VerifyWebhook(body, "sha256=deadbeef")
	require.Error(t, err)

	_, err = sources.Default().VerifyWebhook([]byte(`{"other":"payload"}`), signature)
	require.Error(t, err)

	_, err = sources.Default().VerifyWebhook(body, "")
	require.Error(t, err)
}

func TestEnvVarRoundTripIsEncryptedAtRest(t *testing.T) {
	require.NoError(t, facades.Artisan().Call("migrate"))

	resourceID := uint64(987654321)
	t.Cleanup(func() {
		_, _ = facades.Orm().Query().
			Where("resource_type", models.ResourceTypeApplication).
			Where("resource_id", resourceID).
			Delete(&models.EnvVar{})
	})

	created, err := envvars.Default().Set(models.ResourceTypeApplication, resourceID, "API_SECRET", "s3cr3t", false)
	require.NoError(t, err)
	require.Equal(t, "s3cr3t", created.Value)

	var raw models.EnvVar
	require.NoError(t, facades.Orm().Query().Where("id", created.ID).FirstOrFail(&raw))
	require.NotContains(t, raw.Value, "s3cr3t")
	require.Contains(t, raw.Value, "v1:")

	values, err := envvars.Default().Values(models.ResourceTypeApplication, resourceID, false)
	require.NoError(t, err)
	require.Equal(t, "s3cr3t", values["API_SECRET"])

	updated, err := envvars.Default().Set(models.ResourceTypeApplication, resourceID, "API_SECRET", "rotated", true)
	require.NoError(t, err)
	require.Equal(t, created.ID, updated.ID)
	require.True(t, updated.IsBuildTime)

	require.NoError(t, envvars.Default().Delete(models.ResourceTypeApplication, resourceID, created.ID))

	remaining, err := envvars.Default().List(models.ResourceTypeApplication, resourceID)
	require.NoError(t, err)
	require.Empty(t, remaining)
}

func TestApplicationWebhookMatching(t *testing.T) {
	project := createProject(t)
	source := createGithubSource(t, "hook-secret")

	application, err := applications.Default().Create(applications.Input{
		ProjectUUID: project.UUID,
		Name:        "webhook-" + uuid.NewString()[:8],
		SourceUUID:  source.UUID,
		Repository:  "acme/app",
		Branch:      "main",
		BuildPack:   models.BuildPackDockerfile,
		Config:      models.ApplicationConfig{ExposePort: 3000},
		AutoDeploy:  true,
	})
	require.NoError(t, err)
	t.Cleanup(func() {
		_, _ = facades.Orm().Query().Where("application_id", application.ID).Delete(&models.Deployment{})
		_ = applications.Default().Delete(application)
	})

	matched, err := applications.Default().MatchingWebhook(source.ID, "acme/app", "main")
	require.NoError(t, err)
	require.Len(t, matched, 1)
	require.Equal(t, application.UUID, matched[0].UUID)

	matched, err = applications.Default().MatchingWebhook(source.ID, "acme/app", "release")
	require.NoError(t, err)
	require.Empty(t, matched)

	matched, err = applications.Default().MatchingWebhook(source.ID, "acme/other", "main")
	require.NoError(t, err)
	require.Empty(t, matched)

	// Rollback needs a deployment that carries an image.
	_, err = deployment.Default().EnqueueRollback(application, &models.Deployment{})
	require.Error(t, err)
}

func TestDeploymentSnapshotReturnsLinesAfterSequence(t *testing.T) {
	record := &models.Deployment{
		Log: "first\nsecond\nthird",
	}

	seq, lines := deployment.Default().Snapshot(record, 0)
	require.Equal(t, 3, seq)
	require.Equal(t, []string{"first", "second", "third"}, lines)

	seq, lines = deployment.Default().Snapshot(record, 2)
	require.Equal(t, 3, seq)
	require.Equal(t, []string{"third"}, lines)

	seq, lines = deployment.Default().Snapshot(record, 3)
	require.Equal(t, 3, seq)
	require.Empty(t, lines)

	seq, lines = deployment.Default().Snapshot(&models.Deployment{}, 0)
	require.Equal(t, 0, seq)
	require.Empty(t, lines)
}
