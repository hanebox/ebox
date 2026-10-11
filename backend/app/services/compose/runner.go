package compose

import (
	"context"
	"errors"
	"fmt"
	"path/filepath"
	"strings"
	"time"

	"ebox/app/services/build"
)

const commandTimeout = 15 * time.Minute

// Runner shells out to the Docker Compose plugin. Compose is the only part of
// ebox that uses the Docker CLI: container, image and volume lifecycle stay on
// the Docker API.
type Runner struct{}

func NewRunner() *Runner {
	return &Runner{}
}

// Project describes a compose project on disk.
type Project struct {
	Name    string
	Dir     string
	Files   []string
	EnvFile string
}

// Dir returns the directory holding the project files.
func (p Project) Directory() string {
	if p.Dir != "" {
		return p.Dir
	}

	return filepath.Dir(p.Files[0])
}

// Absolute returns a copy of the project with absolute file paths so the compose
// CLI never resolves them against its working directory.
func (p Project) Absolute() Project {
	absolute := p

	if p.Dir == "" && len(p.Files) > 0 {
		absolute.Dir = filepath.Dir(p.Files[0])
	}

	files := make([]string, 0, len(p.Files))
	for _, file := range p.Files {
		if path, err := filepath.Abs(file); err == nil {
			files = append(files, path)
			continue
		}

		files = append(files, file)
	}
	absolute.Files = files

	if p.EnvFile != "" {
		if path, err := filepath.Abs(p.EnvFile); err == nil {
			absolute.EnvFile = path
		}
	}

	return absolute
}

func (r *Runner) args(project Project, command ...string) []string {
	args := []string{"compose", "-p", project.Name}

	for _, file := range project.Files {
		args = append(args, "-f", file)
	}

	if project.EnvFile != "" {
		args = append(args, "--env-file", project.EnvFile)
	}

	return append(args, command...)
}

func (r *Runner) run(ctx context.Context, project Project, sink func(string), command ...string) error {
	project = project.Absolute()
	args := r.args(project, command...)

	// Keep the tail of the output so failures carry the real reason (a port
	// clash, a bad image, a compose error) and not just the exit status.
	tail := newTailBuffer(12)

	err := build.Run(ctx, build.Command{Dir: project.Directory(), Sink: func(line string) {
		tail.add(line)
		if sink != nil {
			sink(line)
		}
	}}, "docker", args...)
	if err != nil {
		return composeError(command, err, tail.lines())
	}

	return nil
}

// composeError renders a readable failure for a compose command.
func composeError(command []string, err error, output []string) error {
	message := strings.Join(output, "\n")

	switch {
	case strings.Contains(message, "port is already allocated"):
		for _, line := range output {
			if strings.Contains(line, "port is already allocated") {
				message = "A container port is already in use on this host, so the stack could not start: " + line
				break
			}
		}
	case strings.TrimSpace(message) == "":
		message = err.Error()
	}

	return fmt.Errorf("compose: docker %s failed: %s", strings.Join(command, " "), message)
}

// tailBuffer keeps the last lines of a command's output.
type tailBuffer struct {
	limit   int
	entries []string
}

func newTailBuffer(limit int) *tailBuffer {
	return &tailBuffer{limit: limit}
}

func (t *tailBuffer) add(line string) {
	if strings.TrimSpace(line) == "" {
		return
	}

	t.entries = append(t.entries, line)
	if len(t.entries) > t.limit {
		t.entries = t.entries[len(t.entries)-t.limit:]
	}
}

func (t *tailBuffer) lines() []string {
	return t.entries
}

// Up creates or updates the project, building images when needed.
func (r *Runner) Up(ctx context.Context, project Project, buildImages bool, sink func(string)) error {
	command := []string{"up", "-d", "--remove-orphans"}
	if buildImages {
		command = append(command, "--build")
	}

	return r.run(ctx, project, sink, command...)
}

// Down stops and removes the project containers and networks.
func (r *Runner) Down(ctx context.Context, project Project, sink func(string)) error {
	return r.run(ctx, project, sink, "down", "--remove-orphans", "--volumes")
}

// Stop stops the project containers without removing them.
func (r *Runner) Stop(ctx context.Context, project Project, sink func(string)) error {
	return r.run(ctx, project, sink, "stop")
}

// Start starts existing project containers.
func (r *Runner) Start(ctx context.Context, project Project, sink func(string)) error {
	return r.run(ctx, project, sink, "start")
}

// Restart restarts every project container.
func (r *Runner) Restart(ctx context.Context, project Project, sink func(string)) error {
	return r.run(ctx, project, sink, "restart")
}

// Build builds the project images without starting containers.
func (r *Runner) Build(ctx context.Context, project Project, sink func(string)) error {
	return r.run(ctx, project, sink, "build")
}

// Pull pulls images referenced by the project.
func (r *Runner) Pull(ctx context.Context, project Project, sink func(string)) error {
	return r.run(ctx, project, sink, "pull", "--ignore-buildable", "--ignore-pull-failures")
}

// Config validates the merged project configuration.
func (r *Runner) Config(ctx context.Context, project Project) (string, error) {
	output := &strings.Builder{}
	args := r.args(project.Absolute(), "config", "--quiet")

	err := build.Run(ctx, build.Command{Dir: project.Directory(), Sink: func(line string) {
		output.WriteString(line)
		output.WriteString("\n")
	}}, "docker", args...)
	if err != nil {
		return output.String(), err
	}

	return output.String(), nil
}

// Validate checks that the project configuration resolves.
func (r *Runner) Validate(ctx context.Context, project Project) error {
	return r.run(ctx, project, nil, "config", "--quiet")
}

// Available reports whether the docker compose plugin can be used.
func (r *Runner) Available(ctx context.Context) error {
	output := &strings.Builder{}

	err := build.Run(ctx, build.Command{Sink: func(line string) { output.WriteString(line) }}, "docker", "compose", "version")
	if err != nil {
		return errors.New("compose: the docker compose plugin is not available")
	}

	return nil
}
