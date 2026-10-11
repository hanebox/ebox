package docker

import (
	"context"
	"errors"
	"io"

	"github.com/moby/moby/api/types/container"
	"github.com/moby/moby/api/types/network"
	"github.com/moby/moby/client"
)

// createOptions builds the daemon request for a container spec. It is shared by
// the create paths so both honour the same spec fields.
func (s *Service) createOptions(spec Spec) client.ContainerCreateOptions {
	config := &container.Config{
		Image:  spec.Image,
		Env:    spec.Env,
		Labels: spec.Labels,
		Cmd:    spec.Command,
	}

	if len(spec.ExposedPorts) > 0 {
		ports := network.PortSet{}
		for _, port := range spec.ExposedPorts {
			if parsed, err := network.ParsePort(port); err == nil {
				ports[parsed] = struct{}{}
			}
		}
		config.ExposedPorts = ports
	}

	if spec.Healthcheck != nil {
		config.Healthcheck = &container.HealthConfig{
			Test:        spec.Healthcheck.Test,
			Interval:    spec.Healthcheck.Interval,
			Timeout:     spec.Healthcheck.Timeout,
			StartPeriod: spec.Healthcheck.StartPeriod,
			Retries:     spec.Healthcheck.Retries,
		}
	}

	restart := spec.Restart
	if restart == "" {
		restart = "unless-stopped"
	}

	hostConfig := &container.HostConfig{
		RestartPolicy: container.RestartPolicy{Name: container.RestartPolicyMode(restart)},
		Binds:         spec.Volumes,
	}

	if spec.MemoryMB > 0 {
		hostConfig.Memory = int64(spec.MemoryMB) * 1024 * 1024
	}
	if spec.CPUs > 0 {
		hostConfig.NanoCPUs = int64(spec.CPUs * 1e9)
	}
	if spec.StopTimeout > 0 {
		seconds := int(spec.StopTimeout.Seconds())
		config.StopTimeout = &seconds
	}

	var networking *network.NetworkingConfig
	if spec.Network != "" {
		hostConfig.NetworkMode = container.NetworkMode(spec.Network)
		endpoint := &network.EndpointSettings{}
		if spec.Alias != "" {
			endpoint.Aliases = []string{spec.Alias}
		}
		networking = &network.NetworkingConfig{
			EndpointsConfig: map[string]*network.EndpointSettings{spec.Network: endpoint},
		}
	}

	return client.ContainerCreateOptions{
		Config:           config,
		HostConfig:       hostConfig,
		NetworkingConfig: networking,
		Name:             spec.Name,
	}
}

// StartCreated starts a container created with CreateDetached.
func (s *Service) StartCreated(ctx context.Context, id string) error {
	cli, err := s.handle()
	if err != nil {
		return err
	}

	if _, err := cli.ContainerStart(ctx, id, client.ContainerStartOptions{}); err != nil {
		return errors.New(FriendlyError(err))
	}

	return nil
}

// ExecWithOutput runs a command in a container and copies its output.
func (s *Service) ExecWithOutput(ctx context.Context, containerID string, command []string, output io.Writer) (int, string, error) {
	return s.StreamExec(ctx, containerID, command, output)
}
