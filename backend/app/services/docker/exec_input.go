package docker

import (
	"context"
	"errors"
	"io"

	"github.com/moby/moby/client"
)

// CreateDetached creates a container without starting it.
func (s *Service) CreateDetached(ctx context.Context, spec Spec) (string, error) {
	cli, err := s.handle()
	if err != nil {
		return "", err
	}

	result, err := cli.ContainerCreate(ctx, s.createOptions(spec))
	if err != nil {
		return "", errors.New(FriendlyError(err))
	}

	return result.ID, nil
}

// ExecWithInput runs a command in a container, feeding stdin from a reader.
// The combined output is returned so failures can be reported with the reason.
func (s *Service) ExecWithInput(ctx context.Context, containerID string, command []string, input io.Reader) (int, string, error) {
	cli, err := s.handle()
	if err != nil {
		return 0, "", err
	}

	created, err := cli.ExecCreate(ctx, containerID, client.ExecCreateOptions{
		Cmd:          command,
		AttachStdin:  true,
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
	done := make(chan error, 1)

	go func() {
		_, err := io.Copy(attached.Conn, input)
		if err != nil {
			done <- err

			return
		}

		// The half close is what tells the process stdin is over; without it a
		// command that reads to EOF (psql, mysql) waits forever.
		done <- attached.CloseWrite()
	}()

	// Drain the multiplexed output so the process is not blocked on a full pipe,
	// keeping the tail for diagnostics.
	_ = demux(attached.Reader, captured, captured)

	select {
	case err := <-done:
		if err != nil && ctx.Err() == nil {
			return 0, captured.String(), errors.New("docker: the command input could not be written")
		}
	case <-ctx.Done():
		return 0, captured.String(), ctx.Err()
	}

	inspect, err := cli.ExecInspect(ctx, created.ID, client.ExecInspectOptions{})
	if err != nil {
		return 0, captured.String(), errors.New(FriendlyError(err))
	}

	return inspect.ExitCode, captured.String(), nil
}
