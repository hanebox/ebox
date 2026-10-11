package build

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"os/exec"
	"strings"
	"sync"
)

type LineSink func(line string)

type Command struct {
	Dir   string
	Env   []string
	Sink  LineSink
	Quiet bool
}

// Run executes a command and streams its combined output line by line.
func Run(ctx context.Context, command Command, name string, args ...string) error {
	cmd := exec.CommandContext(ctx, name, args...)
	cmd.Dir = command.Dir
	cmd.Env = command.Env

	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return err
	}

	stderr, err := cmd.StderrPipe()
	if err != nil {
		return err
	}

	if err := cmd.Start(); err != nil {
		return fmt.Errorf("%s: %w", name, err)
	}

	var (
		wg    sync.WaitGroup
		lines []string
		mu    sync.Mutex
	)

	stream := func(reader io.Reader) {
		defer wg.Done()

		scanner := bufio.NewScanner(reader)
		scanner.Buffer(make([]byte, 0, 64*1024), 1024*1024)

		for scanner.Scan() {
			line := strings.TrimRight(scanner.Text(), "\r")
			if line == "" {
				continue
			}

			mu.Lock()
			if len(lines) < 40 {
				lines = append(lines, line)
			}
			mu.Unlock()

			if command.Sink != nil {
				command.Sink(line)
			}
		}
	}

	wg.Add(2)
	go stream(stdout)
	go stream(stderr)
	wg.Wait()

	if err := cmd.Wait(); err != nil {
		if ctx.Err() != nil {
			return ctx.Err()
		}

		mu.Lock()
		tail := strings.Join(lines, "\n")
		mu.Unlock()

		if tail == "" {
			return fmt.Errorf("%s %s failed: %w", name, strings.Join(args, " "), err)
		}

		return fmt.Errorf("%s %s failed: %w\n%s", name, strings.Join(args, " "), err, tail)
	}

	return nil
}

// DockerEnv builds the environment for docker CLI invocations.
func DockerEnv(extra []string) []string {
	return extra
}
