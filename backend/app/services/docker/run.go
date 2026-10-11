package docker

import (
	"context"
	"errors"
	"io"
	"strings"

	"github.com/moby/moby/api/pkg/stdcopy"
	"github.com/moby/moby/client"
)

// ExecResult is a completed non-interactive exec.
type ExecResult struct {
	ExitCode int
	Stdout   string
	Stderr   string
}

// demux splits the multiplexed exec stream docker uses for non-TTY processes
// into its standard output and standard error parts. Without this the raw frame
// headers end up inside the payload (a dump, for example).
func demux(reader io.Reader, stdout, stderr io.Writer) error {
	_, err := stdcopy.StdCopy(stdout, stderr, reader)

	return err
}

// Run executes a command in a container and waits for it to finish. Output is
// capped so a runaway command cannot exhaust memory.
func (s *Service) Run(ctx context.Context, containerID string, command []string) (ExecResult, error) {
	cli, err := s.handle()
	if err != nil {
		return ExecResult{}, err
	}

	created, err := cli.ExecCreate(ctx, containerID, client.ExecCreateOptions{
		Cmd:          command,
		AttachStdout: true,
		AttachStderr: true,
	})
	if err != nil {
		return ExecResult{}, errors.New(FriendlyError(err))
	}

	attached, err := cli.ExecAttach(ctx, created.ID, client.ExecAttachOptions{})
	if err != nil {
		return ExecResult{}, errors.New(FriendlyError(err))
	}
	defer attached.Close()

	stdout, stderr := &limitedBuffer{limit: 1 << 20}, &limitedBuffer{limit: 1 << 20}

	if err := demux(attached.Reader, stdout, stderr); err != nil && ctx.Err() == nil {
		// The reader ends when the process exits; a broken pipe is not fatal.
		_ = err
	}

	inspect, err := cli.ExecInspect(ctx, created.ID, client.ExecInspectOptions{})
	if err != nil {
		return ExecResult{}, errors.New(FriendlyError(err))
	}

	return ExecResult{
		ExitCode: inspect.ExitCode,
		Stdout:   stdout.String(),
		Stderr:   stderr.String(),
	}, nil
}

// StreamExec runs a command in a container and copies its standard output into
// the writer. It is used for dumps, which must not be buffered in memory. The
// captured standard error is returned for diagnostics.
func (s *Service) StreamExec(ctx context.Context, containerID string, command []string, output io.Writer) (int, string, error) {
	cli, err := s.handle()
	if err != nil {
		return 0, "", err
	}

	created, err := cli.ExecCreate(ctx, containerID, client.ExecCreateOptions{
		Cmd:          command,
		AttachStdout: true,
		AttachStderr: true,
	})
	if err != nil {
		return 0, "", errors.New(FriendlyError(err))
	}

	attached, err := cli.ExecAttach(ctx, created.ID, client.ExecAttachOptions{})
	if err != nil {
		return 0, "", errors.New(FriendlyError(err))
	}
	defer attached.Close()

	captured := &limitedBuffer{limit: 8 << 10}

	if err := demux(attached.Reader, output, captured); err != nil && ctx.Err() == nil {
		return 0, captured.String(), errors.New("docker: the command output could not be read")
	}

	inspect, err := cli.ExecInspect(ctx, created.ID, client.ExecInspectOptions{})
	if err != nil {
		return 0, captured.String(), errors.New(FriendlyError(err))
	}

	return inspect.ExitCode, captured.String(), nil
}

// CopyFromContainer streams a path out of a container as a tar archive.
func (s *Service) CopyFromContainer(ctx context.Context, containerID, path string) (io.ReadCloser, error) {
	cli, err := s.handle()
	if err != nil {
		return nil, err
	}

	reader, err := cli.CopyFromContainer(ctx, containerID, client.CopyFromContainerOptions{SourcePath: path})
	if err != nil {
		return nil, errors.New(FriendlyError(err))
	}

	return reader.Content, nil
}

// CopyToContainer writes a tar archive into a container directory.
func (s *Service) CopyToContainer(ctx context.Context, containerID, path string, content io.Reader) error {
	cli, err := s.handle()
	if err != nil {
		return err
	}

	if _, err := cli.CopyToContainer(ctx, containerID, client.CopyToContainerOptions{
		DestinationPath: path,
		Content:         content,
	}); err != nil {
		return errors.New(FriendlyError(err))
	}

	return nil
}

type limitedBuffer struct {
	builder strings.Builder
	limit   int
}

func (b *limitedBuffer) Write(payload []byte) (int, error) {
	if b.builder.Len() < b.limit {
		remaining := b.limit - b.builder.Len()
		if len(payload) > remaining {
			payload = payload[:remaining]
		}

		b.builder.Write(payload)
	}

	return len(payload), nil
}

func (b *limitedBuffer) String() string {
	return b.builder.String()
}
