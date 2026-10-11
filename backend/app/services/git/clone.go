package git

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
)

type CommitInfo struct {
	SHA      string
	ShortSHA string
	Message  string
}

type SyncOptions struct {
	Directory   string
	URL         string
	Credentials Credentials
	Branch      string
	Commit      string
	Sink        func(string)
}

// Sync brings a repository working copy to the requested branch or commit.
func (s *Service) Sync(ctx context.Context, opts SyncOptions) (CommitInfo, error) {
	var info CommitInfo

	if opts.Directory == "" || opts.URL == "" {
		return info, errors.New("git: directory and url are required")
	}

	env, err := s.Env(opts.Credentials)
	if err != nil {
		return info, err
	}

	sink := opts.Sink
	if sink == nil {
		sink = func(string) {}
	}

	branch := opts.Branch
	if branch == "" {
		branch = "main"
	}

	if _, err := os.Stat(filepath.Join(opts.Directory, ".git")); err != nil {
		if err := os.MkdirAll(filepath.Dir(opts.Directory), 0o755); err != nil {
			return info, err
		}

		sink("Cloning " + opts.URL + " (" + branch + ")")
		if err := s.run(ctx, env, sink, "", "clone", "--depth", "1", "--branch", branch, "--single-branch", opts.URL, opts.Directory); err != nil {
			return info, err
		}
	} else {
		ref := opts.Commit
		if ref == "" {
			ref = branch
		}

		sink("Fetching " + ref)
		if err := s.run(ctx, env, sink, opts.Directory, "fetch", "--depth", "1", opts.URL, ref); err != nil {
			if opts.Commit == "" {
				return info, err
			}

			sink("Commit " + ref + " is not fetchable directly, falling back to " + branch)
			if err := s.run(ctx, env, sink, opts.Directory, "fetch", "--depth", "1", opts.URL, branch); err != nil {
				return info, err
			}
		}

		if err := s.run(ctx, env, sink, opts.Directory, "checkout", "-f", "FETCH_HEAD"); err != nil {
			return info, err
		}

		if err := s.run(ctx, env, sink, opts.Directory, "clean", "-fdxq"); err != nil {
			return info, err
		}
	}

	sha, err := s.output(ctx, env, opts.Directory, "rev-parse", "HEAD")
	if err != nil {
		return info, err
	}
	info.SHA = strings.TrimSpace(sha)

	message, err := s.output(ctx, env, opts.Directory, "log", "-1", "--format=%s")
	if err != nil {
		return info, err
	}
	info.Message = strings.TrimSpace(message)

	if len(info.SHA) >= 7 {
		info.ShortSHA = info.SHA[:7]
	} else {
		info.ShortSHA = info.SHA
	}

	return info, nil
}

// Dir returns the working copy directory for a resource.
func (s *Service) Dir(kind, id string) string {
	return filepath.Join(s.dataDir, "apps", id, "repo")
}

func (s *Service) run(ctx context.Context, env []string, sink func(string), dir string, args ...string) error {
	cmd := exec.CommandContext(ctx, "git", args...)
	cmd.Dir = dir
	cmd.Env = append(env, "GIT_TERMINAL_PROMPT=0")

	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return err
	}

	stderr, err := cmd.StderrPipe()
	if err != nil {
		return err
	}

	if err := cmd.Start(); err != nil {
		return fmt.Errorf("git: %w", err)
	}

	var wg sync.WaitGroup
	stream := func(reader io.Reader) {
		defer wg.Done()

		scanner := bufio.NewScanner(reader)
		scanner.Buffer(make([]byte, 0, 64*1024), 1024*1024)

		for scanner.Scan() {
			line := strings.TrimRight(scanner.Text(), "\r")
			if line != "" {
				sink(line)
			}
		}
	}

	wg.Add(2)
	go stream(stdout)
	go stream(stderr)
	wg.Wait()

	if err := cmd.Wait(); err != nil {
		return fmt.Errorf("git %s failed: %w", args[0], err)
	}

	return nil
}

func (s *Service) output(ctx context.Context, env []string, dir string, args ...string) (string, error) {
	cmd := exec.CommandContext(ctx, "git", args...)
	cmd.Dir = dir
	cmd.Env = append(env, "GIT_TERMINAL_PROMPT=0")

	out, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("git %s failed: %w", args[0], err)
	}

	return string(out), nil
}
