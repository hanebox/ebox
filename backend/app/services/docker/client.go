package docker

import (
	"context"
	"errors"
	"strings"
	"sync"
	"time"

	"github.com/moby/moby/client"
)

type Service struct {
	once   sync.Once
	client *client.Client
	err    error
}

var defaultService = &Service{}

func Default() *Service {
	return defaultService
}

func New() *Service {
	return &Service{}
}

func (s *Service) handle() (*client.Client, error) {
	s.once.Do(func() {
		s.client, s.err = client.NewClientWithOpts(
			client.FromEnv,
			client.WithAPIVersionNegotiation(),
			client.WithTimeout(60*time.Second),
		)
	})

	return s.client, s.err
}

// Available reports whether the Docker daemon can be reached.
func (s *Service) Available(ctx context.Context) error {
	cli, err := s.handle()
	if err != nil {
		return err
	}

	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	if _, err := cli.Ping(ctx, client.PingOptions{}); err != nil {
		return errors.New(FriendlyError(err))
	}

	return nil
}

// EnsureNetwork creates the network managed containers are attached to.
func (s *Service) EnsureNetwork(ctx context.Context, name string) error {
	if name == "" {
		return errors.New("docker: network name is empty")
	}

	cli, err := s.handle()
	if err != nil {
		return err
	}

	networks, err := cli.NetworkList(ctx, client.NetworkListOptions{
		Filters: client.Filters{}.Add("name", name),
	})
	if err != nil {
		return errors.New(FriendlyError(err))
	}

	for _, item := range networks.Items {
		if item.Name == name {
			return nil
		}
	}

	enableIPv4 := true
	if _, err := cli.NetworkCreate(ctx, name, client.NetworkCreateOptions{
		Driver:     "bridge",
		EnableIPv4: &enableIPv4,
		Labels:     map[string]string{LabelManaged: "true"},
	}); err != nil {
		if strings.Contains(err.Error(), "already exists") {
			return nil
		}

		return errors.New(FriendlyError(err))
	}

	return nil
}

// FriendlyError trims the boilerplate Docker daemon errors down to something
// readable in the UI.
func FriendlyError(err error) string {
	message := err.Error()
	message = strings.TrimPrefix(message, "Error response from daemon: ")

	if index := strings.Index(message, "Cannot connect to the Docker daemon"); index >= 0 {
		return "Cannot connect to the Docker daemon. Is Docker running?"
	}

	if len(message) > 500 {
		message = message[:500] + "…"
	}

	return message
}
