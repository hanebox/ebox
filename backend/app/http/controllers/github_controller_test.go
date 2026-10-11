package controllers

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestGithubManifest(t *testing.T) {
	manifest := githubManifest("https://panel.example.com", "11111111-2222-3333-4444-555555555555", "ebox-test")

	require.Equal(t, "ebox-test", manifest["name"])
	require.Equal(t, "https://panel.example.com", manifest["url"])
	require.Equal(t, "https://panel.example.com/api/v1/github/callback", manifest["redirect_url"])
	require.Equal(t, "https://panel.example.com/api/v1/github/setup/11111111-2222-3333-4444-555555555555", manifest["setup_url"])
	require.Equal(t, false, manifest["public"])
	require.Equal(t, []string{"push"}, manifest["default_events"])

	hook, ok := manifest["hook_attributes"].(map[string]any)
	require.True(t, ok)
	require.Equal(t, "https://panel.example.com/api/v1/webhooks/github", hook["url"])
	require.Equal(t, true, hook["active"])

	permissions, ok := manifest["default_permissions"].(map[string]any)
	require.True(t, ok)
	require.Equal(t, "read", permissions["contents"])
	require.Equal(t, "read", permissions["metadata"])
}

func TestDefaultGithubAppName(t *testing.T) {
	name := defaultGithubAppName()

	require.NotEmpty(t, name)
	require.LessOrEqual(t, len(name), 34)
	require.Regexp(t, `^[a-z0-9-]+$`, name)
	require.Equal(t, "ebox-", name[:5])
}
