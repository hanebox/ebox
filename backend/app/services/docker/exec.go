package docker

import (
	"context"
	"errors"

	"github.com/moby/moby/api/types/events"
	"github.com/moby/moby/client"
)

// Events streams daemon events matching the filters.
func (s *Service) Events(ctx context.Context, filters Filters) (*client.EventsResult, error) {
	cli, err := s.handle()
	if err != nil {
		return nil, err
	}

	result := cli.Events(ctx, client.EventsListOptions{Filters: filters})

	return &result, nil
}

// ExecSession is an attached exec process inside a container.
type ExecSession struct {
	ID       string
	Response client.HijackedResponse
	client   *client.Client
}

func (s *Service) Exec(ctx context.Context, containerID string, command []string, tty bool, width, height uint) (*ExecSession, error) {
	cli, err := s.handle()
	if err != nil {
		return nil, err
	}

	created, err := cli.ExecCreate(ctx, containerID, client.ExecCreateOptions{
		Cmd:          command,
		TTY:          tty,
		AttachStdin:  true,
		AttachStdout: true,
		AttachStderr: true,
		ConsoleSize:  client.ConsoleSize{Width: width, Height: height},
	})
	if err != nil {
		return nil, errors.New(FriendlyError(err))
	}

	attached, err := cli.ExecAttach(ctx, created.ID, client.ExecAttachOptions{
		TTY:         tty,
		ConsoleSize: client.ConsoleSize{Width: width, Height: height},
	})
	if err != nil {
		return nil, errors.New(FriendlyError(err))
	}

	return &ExecSession{ID: created.ID, Response: attached.HijackedResponse, client: cli}, nil
}

func (s *ExecSession) Resize(ctx context.Context, width, height uint) error {
	if s == nil || s.ID == "" {
		return errors.New("docker: exec session is not attached")
	}

	_, err := s.client.ExecResize(ctx, s.ID, client.ExecResizeOptions{
		Width:  width,
		Height: height,
	})
	if err != nil {
		return errors.New(FriendlyError(err))
	}

	return nil
}

func (s *ExecSession) Close() {
	if s == nil {
		return
	}

	s.Response.Close()
}

// ExecInspect reports whether the exec process is still running.
func (s *ExecSession) Running(ctx context.Context) bool {
	if s == nil || s.ID == "" {
		return false
	}

	result, err := s.client.ExecInspect(ctx, s.ID, client.ExecInspectOptions{})
	if err != nil {
		return false
	}

	return result.Running
}

// EventAction reports whether an event is a container lifecycle action worth
// pushing to the UI.
func EventAction(action events.Action) (string, bool) {
	switch action {
	case events.ActionStart:
		return "start", true
	case events.ActionStop:
		return "stop", true
	case events.ActionDie:
		return "die", true
	case events.ActionKill:
		return "kill", true
	case events.ActionRestart:
		return "restart", true
	case events.ActionCreate:
		return "create", true
	case events.ActionDestroy:
		return "destroy", true
	case events.ActionHealthStatusHealthy:
		return "healthy", true
	case events.ActionHealthStatusUnhealthy:
		return "unhealthy", true
	}

	return "", false
}
