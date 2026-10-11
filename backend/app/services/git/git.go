package git

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"ebox/app/facades"
)

const commandTimeout = 45 * time.Second

// Credentials describe how to authenticate a git command against a remote.
type Credentials struct {
	AuthType string
	Token    string
	SSHKey   string
	KeyName  string
}

type Service struct {
	dataDir string
}

func New(dataDir string) *Service {
	return &Service{dataDir: dataDir}
}

func Default() *Service {
	return New(facades.Config().GetString("ebox.data_dir"))
}

// Test verifies that a remote is reachable with the given credentials.
func (s *Service) Test(ctx context.Context, url string, creds Credentials) error {
	ctx, cancel := context.WithTimeout(ctx, commandTimeout)
	defer cancel()

	env, err := s.Env(creds)
	if err != nil {
		return err
	}

	cmd := exec.CommandContext(ctx, "git", "ls-remote", "--heads", url)
	cmd.Env = env

	output, err := cmd.CombinedOutput()
	if err != nil {
		message := strings.TrimSpace(string(output))
		if message == "" {
			message = err.Error()
		}

		return errors.New(message)
	}

	return nil
}

// Env builds the environment for a git command, injecting credentials without
// exposing them in process arguments.
func (s *Service) Env(creds Credentials) ([]string, error) {
	env := append(os.Environ(), "GIT_TERMINAL_PROMPT=0")

	switch creds.AuthType {
	case "token":
		if creds.Token == "" {
			return nil, errors.New("git: token credentials are missing the token")
		}

		askpass, err := s.askpassScript()
		if err != nil {
			return nil, err
		}

		env = append(env, "GIT_ASKPASS="+askpass, "EB0X_GIT_TOKEN="+creds.Token)
	case "ssh":
		if creds.SSHKey == "" {
			return nil, errors.New("git: ssh credentials are missing the private key")
		}

		keyPath, err := s.sshKeyFile(creds)
		if err != nil {
			return nil, err
		}

		env = append(env, "GIT_SSH_COMMAND="+s.sshCommand(keyPath))
	}

	return env, nil
}

// RemoveKey deletes any SSH key material stored for a source.
func (s *Service) RemoveKey(name string) {
	if name == "" {
		return
	}

	_ = os.Remove(filepath.Join(s.dataDir, "keys", filepath.Base(name)))
}

func (s *Service) askpassScript() (string, error) {
	path := filepath.Join(s.dataDir, "git-askpass.sh")
	if _, err := os.Stat(path); err == nil {
		return path, nil
	}

	if err := os.MkdirAll(s.dataDir, 0o755); err != nil {
		return "", err
	}

	script := "#!/bin/sh\n" +
		"case \"$1\" in\n" +
		"  Username*) printf '%s' \"${EB0X_GIT_USERNAME:-x-access-token}\" ;;\n" +
		"  Password*) printf '%s' \"${EB0X_GIT_TOKEN}\" ;;\n" +
		"esac\n"

	if err := os.WriteFile(path, []byte(script), 0o700); err != nil {
		return "", err
	}

	return path, nil
}

func (s *Service) sshKeyFile(creds Credentials) (string, error) {
	name := creds.KeyName
	if name == "" {
		name = "default"
	}

	dir := filepath.Join(s.dataDir, "keys")
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return "", err
	}

	content := creds.SSHKey
	if !strings.HasSuffix(content, "\n") {
		content += "\n"
	}

	path := filepath.Join(dir, filepath.Base(name))
	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		return "", err
	}

	knownHosts := filepath.Join(s.dataDir, "ssh_known_hosts")
	if _, err := os.Stat(knownHosts); os.IsNotExist(err) {
		_ = os.WriteFile(knownHosts, nil, 0o600)
	}

	return path, nil
}

func (s *Service) sshCommand(keyPath string) string {
	knownHosts := filepath.Join(s.dataDir, "ssh_known_hosts")

	return fmt.Sprintf(
		"ssh -i %s -o IdentitiesOnly=yes -o StrictHostKeyChecking=accept-new -o UserKnownHostsFile=%s",
		shellQuote(keyPath),
		shellQuote(knownHosts),
	)
}

func shellQuote(value string) string {
	return "'" + strings.ReplaceAll(value, "'", `'\''`) + "'"
}
