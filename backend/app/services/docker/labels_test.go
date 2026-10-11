package docker

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestResourceFilterUsesLabelKey(t *testing.T) {
	filters := ResourceFilter(ResourceApplication, "abc-123")

	require.Equal(t, map[string]bool{
		"ebox.managed=true":              true,
		"ebox.resource-type=application": true,
		"ebox.resource-id=abc-123":       true,
	}, filters["label"])

	managed := ResourceFilter("", "")
	require.Equal(t, map[string]bool{"ebox.managed=true": true}, managed["label"])
}

func TestManagedLabels(t *testing.T) {
	labels := ManagedLabels(ResourceApplication, "abc", "hello")

	require.Equal(t, "true", labels[LabelManaged])
	require.Equal(t, ResourceApplication, labels[LabelResourceType])
	require.Equal(t, "abc", labels[LabelResourceID])
	require.Equal(t, "hello", labels[LabelName])
}

func TestFriendlyError(t *testing.T) {
	require.Contains(t, FriendlyError(errString("Cannot connect to the Docker daemon at unix:///var/run/docker.sock. Is the docker daemon running?")), "Is Docker running?")
	require.Equal(t, "boom", FriendlyError(errString("Error response from daemon: boom")))
}

type errString string

func (e errString) Error() string { return string(e) }
