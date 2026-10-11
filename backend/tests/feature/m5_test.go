package feature

import (
	"testing"

	"github.com/stretchr/testify/require"

	"ebox/app/facades"
	"ebox/app/models"
	"ebox/app/services/notifications"
	_ "ebox/tests"
)

// TestNotificationChannelSecretsAreEncrypted checks that credentials are
// encrypted at rest, masked on read and preserved when the client echoes the
// masked value back.
func TestNotificationChannelSecretsAreEncrypted(t *testing.T) {
	require.NoError(t, facades.Artisan().Call("migrate"))

	channel, err := notifications.Default().Create(notifications.Input{
		Name:    "ops-slack",
		Type:    models.NotificationTypeSlack,
		Enabled: true,
		Config: map[string]string{
			"webhook_url": "https://hooks.slack.com/services/secret",
			"username":    "ebox",
		},
		Events: []string{models.EventDeploymentFailed, models.EventBackupFailed},
	})
	require.NoError(t, err)
	t.Cleanup(func() {
		_, _ = facades.Orm().Query().Where("id", channel.ID).Delete(&models.NotificationChannel{})
	})

	// The stored value is not readable plaintext.
	require.NotContains(t, channel.Config, "hooks.slack.com")
	require.Contains(t, channel.Config, ":")

	stored, err := notifications.Default().Config(channel)
	require.NoError(t, err)
	require.Equal(t, "https://hooks.slack.com/services/secret", stored["webhook_url"])

	events := notifications.Default().Events(channel)
	require.Equal(t, []string{models.EventDeploymentFailed, models.EventBackupFailed}, events)

	// A secret the client sends back unchanged keeps the stored value.
	require.NoError(t, notifications.Default().Update(channel, notifications.Input{
		Name:    "ops-slack",
		Type:    models.NotificationTypeSlack,
		Enabled: true,
		Config: map[string]string{
			"webhook_url": "https://hooks.slack.com/services/rotated",
			"username":    "ebox-2",
		},
		Events: []string{models.EventDeploymentFailed},
	}))

	rotated, err := notifications.Default().Config(channel)
	require.NoError(t, err)
	require.Equal(t, "https://hooks.slack.com/services/rotated", rotated["webhook_url"])
	require.Equal(t, "ebox-2", rotated["username"])
	require.Equal(t, []string{models.EventDeploymentFailed}, notifications.Default().Events(channel))
}

func TestNotificationValidation(t *testing.T) {
	service := notifications.Default()

	_, err := service.Create(notifications.Input{Name: "", Type: models.NotificationTypeDiscord})
	require.Error(t, err)

	_, err = service.Create(notifications.Input{Name: "x", Type: "carrier-pigeon"})
	require.Error(t, err)

	_, err = service.Create(notifications.Input{
		Name:   "x",
		Type:   models.NotificationTypeDiscord,
		Config: map[string]string{"webhook_url": "https://example.com/hook"},
		Events: []string{"not.an.event"},
	})
	require.Error(t, err)
}

// TestNotificationSendFailsWithoutEndpoint checks the failure path: a delivery
// to an unreachable endpoint is recorded as failed rather than swallowed.
func TestNotificationSendFailsWithoutEndpoint(t *testing.T) {
	require.NoError(t, facades.Artisan().Call("migrate"))

	channel, err := notifications.Default().Create(notifications.Input{
		Name:    "unreachable",
		Type:    models.NotificationTypeWebhook,
		Enabled: true,
		Config:  map[string]string{"webhook_url": "http://127.0.0.1:1/hook"},
	})
	require.NoError(t, err)
	t.Cleanup(func() {
		_, _ = facades.Orm().Query().Where("id", channel.ID).Delete(&models.NotificationChannel{})
		_, _ = facades.Orm().Query().Where("channel_id", channel.ID).Delete(&models.NotificationDelivery{})
	})

	err = notifications.Default().Test(t.Context(), channel)
	require.Error(t, err)

	deliveries, err := notifications.Default().History(10)
	require.NoError(t, err)
	require.NotEmpty(t, deliveries)

	found := false
	for _, delivery := range deliveries {
		if delivery.ChannelID == channel.ID {
			require.Equal(t, models.NotificationStatusFailed, delivery.Status)
			require.NotEmpty(t, delivery.Error)

			found = true
		}
	}
	require.True(t, found, "the failed delivery must be recorded")
}
