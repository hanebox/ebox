package feature

import (
	"encoding/json"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"

	"ebox/app/facades"
	"ebox/app/models"
	"ebox/app/services/account"
	"ebox/app/services/sources"
	_ "ebox/tests"
)

func TestGitSourceSecretsAreEncryptedAtRest(t *testing.T) {
	require.NoError(t, facades.Artisan().Call("migrate"))

	service := sources.Default()
	source, err := service.CreateGit("git-"+uuid.NewString()[:8], "https://example.com/repo.git", models.SourceAuthToken, "super-secret-token", "")
	require.NoError(t, err)
	t.Cleanup(func() { _ = service.Delete(source) })

	require.NotContains(t, source.TokenEncrypted, "super-secret-token")
	require.True(t, strings.HasPrefix(source.TokenEncrypted, "v1:"))

	token, err := service.Token(source)
	require.NoError(t, err)
	require.Equal(t, "super-secret-token", token)

	payload, err := json.Marshal(source)
	require.NoError(t, err)
	require.NotContains(t, string(payload), "super-secret-token")
	require.NotContains(t, string(payload), source.TokenEncrypted)
}

func TestGithubAppSourceSecretsAreEncryptedAtRest(t *testing.T) {
	require.NoError(t, facades.Artisan().Call("migrate"))

	service := sources.Default()
	source, err := service.CreateGithubApp(sources.GithubAppInput{
		Name:          "app-" + uuid.NewString()[:8],
		AppID:         123456,
		AppSlug:       "ebox-test",
		PrivateKey:    "-----BEGIN RSA PRIVATE KEY-----\nMIIfake\n-----END RSA PRIVATE KEY-----",
		WebhookSecret: "hook-secret",
	})
	require.NoError(t, err)
	t.Cleanup(func() { _ = service.Delete(source) })

	key, err := service.PrivateKey(source)
	require.NoError(t, err)
	require.Contains(t, key, "BEGIN RSA PRIVATE KEY")

	secret, err := service.WebhookSecret(source)
	require.NoError(t, err)
	require.Equal(t, "hook-secret", secret)

	require.NoError(t, service.SetInstallation(source, "42"))
	reloaded, err := service.Find(source.UUID)
	require.NoError(t, err)
	require.Equal(t, "42", reloaded.InstallationID)
}

func TestAccessTokens(t *testing.T) {
	require.NoError(t, facades.Artisan().Call("migrate"))

	service := account.Default()
	email := "token-" + uuid.NewString()[:8] + "@example.com"
	user, err := service.CreateUser("Token Tester", email, "correct-horse-battery-staple")
	require.NoError(t, err)
	t.Cleanup(func() {
		_, _ = facades.Orm().Query().Where("email", email).Delete(&models.User{})
	})

	token, record, err := service.CreateAccessToken(user, "ci", nil)
	require.NoError(t, err)
	require.True(t, strings.HasPrefix(token, "ebox_pat_"))
	t.Cleanup(func() { _ = service.DeleteAccessToken(user.ID, record.ID) })

	authenticated, authenticatedUser, err := service.AuthenticateAccessToken(token)
	require.NoError(t, err)
	require.Equal(t, record.ID, authenticated.ID)
	require.Equal(t, user.ID, authenticatedUser.ID)

	_, _, err = service.AuthenticateAccessToken("ebox_pat_wrong")
	require.Error(t, err)

	expiredAt := time.Now().Add(-time.Hour)
	expiredToken, _, err := service.CreateAccessToken(user, "expired", &expiredAt)
	require.NoError(t, err)

	_, _, err = service.AuthenticateAccessToken(expiredToken)
	require.Error(t, err)
}

func TestPendingManifestIsSingleUse(t *testing.T) {
	require.NoError(t, facades.Artisan().Call("migrate"))

	service := sources.Default()
	pending := sources.PendingManifest{
		State:  uuid.NewString(),
		UUID:   uuid.NewString(),
		UserID: 1,
		Origin: "https://panel.example.com",
	}
	require.NoError(t, service.SavePending(pending))

	taken, err := service.TakePending(pending.State)
	require.NoError(t, err)
	require.Equal(t, pending.UUID, taken.UUID)
	require.Equal(t, pending.Origin, taken.Origin)

	_, err = service.TakePending(pending.State)
	require.Error(t, err)
}
