package git

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestEnvTokenAuthCreatesAskpass(t *testing.T) {
	dir := t.TempDir()
	service := New(dir)

	env, err := service.Env(Credentials{AuthType: "token", Token: "secret-token"})
	require.NoError(t, err)

	askpass := filepath.Join(dir, "git-askpass.sh")
	info, err := os.Stat(askpass)
	require.NoError(t, err)
	require.Equal(t, os.FileMode(0o700), info.Mode().Perm())

	require.Contains(t, env, "GIT_ASKPASS="+askpass)
	require.Contains(t, env, "EB0X_GIT_TOKEN=secret-token")
	require.Contains(t, env, "GIT_TERMINAL_PROMPT=0")
}

func TestEnvTokenAuthRequiresToken(t *testing.T) {
	_, err := New(t.TempDir()).Env(Credentials{AuthType: "token"})
	require.Error(t, err)
}

func TestEnvSSHAuthWritesKey(t *testing.T) {
	dir := t.TempDir()
	service := New(dir)

	env, err := service.Env(Credentials{AuthType: "ssh", SSHKey: "-----BEGIN OPENSSH PRIVATE KEY-----", KeyName: "abc"})
	require.NoError(t, err)

	keyPath := filepath.Join(dir, "keys", "abc")
	content, err := os.ReadFile(keyPath)
	require.NoError(t, err)
	require.True(t, strings.HasSuffix(string(content), "\n"))

	info, err := os.Stat(keyPath)
	require.NoError(t, err)
	require.Equal(t, os.FileMode(0o600), info.Mode().Perm())

	_, err = os.Stat(filepath.Join(dir, "ssh_known_hosts"))
	require.NoError(t, err)

	command := ""
	for _, entry := range env {
		if strings.HasPrefix(entry, "GIT_SSH_COMMAND=") {
			command = strings.TrimPrefix(entry, "GIT_SSH_COMMAND=")
		}
	}
	require.Contains(t, command, "-i '"+keyPath+"'")
	require.Contains(t, command, "StrictHostKeyChecking=accept-new")
	require.Contains(t, command, "IdentitiesOnly=yes")
}

func TestEnvSSHAuthRequiresKey(t *testing.T) {
	_, err := New(t.TempDir()).Env(Credentials{AuthType: "ssh"})
	require.Error(t, err)
}

func TestEnvNoneAuth(t *testing.T) {
	env, err := New(t.TempDir()).Env(Credentials{AuthType: "none"})
	require.NoError(t, err)
	require.Contains(t, env, "GIT_TERMINAL_PROMPT=0")
}

func TestTestRemoteAgainstLocalRepository(t *testing.T) {
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git is not installed")
	}

	dir := t.TempDir()
	bare := filepath.Join(dir, "demo.git")
	require.NoError(t, exec.Command("git", "init", "--bare", "-q", bare).Run())

	service := New(t.TempDir())
	require.NoError(t, service.Test(context.Background(), "file://"+bare, Credentials{AuthType: "none"}))
	require.Error(t, service.Test(context.Background(), "file://"+filepath.Join(dir, "missing.git"), Credentials{AuthType: "none"}))
}

func TestRemoveKey(t *testing.T) {
	dir := t.TempDir()
	service := New(dir)

	_, err := service.Env(Credentials{AuthType: "ssh", SSHKey: "key-material", KeyName: "gone"})
	require.NoError(t, err)

	service.RemoveKey("gone")

	_, err = os.Stat(filepath.Join(dir, "keys", "gone"))
	require.True(t, os.IsNotExist(err))
}
