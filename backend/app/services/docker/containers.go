package docker

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/netip"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/moby/moby/api/pkg/stdcopy"
	"github.com/moby/moby/api/types/container"
	"github.com/moby/moby/api/types/network"
	"github.com/moby/moby/client"
)

type Filters = client.Filters

type Container struct {
	ID      string            `json:"id"`
	Name    string            `json:"name"`
	Image   string            `json:"image"`
	State   string            `json:"state"`
	Status  string            `json:"status"`
	Health  string            `json:"health,omitempty"`
	Created int64             `json:"created"`
	Ports   []string          `json:"ports,omitempty"`
	Labels  map[string]string `json:"labels,omitempty"`
}

type ContainerStats struct {
	CPUPercent    float64 `json:"cpu_percent"`
	MemoryUsage   uint64  `json:"memory_usage"`
	MemoryLimit   uint64  `json:"memory_limit"`
	MemoryPercent float64 `json:"memory_percent"`
}

// PortBinding publishes a container port on the host.
type PortBinding struct {
	ContainerPort string
	HostPort      string
	HostIP        string
}

// Healthcheck describes the container health probe.
type Healthcheck struct {
	Test        []string
	Interval    time.Duration
	Timeout     time.Duration
	StartPeriod time.Duration
	Retries     int
}

type Spec struct {
	Name         string
	Image        string
	Env          []string
	Labels       map[string]string
	Network      string
	Alias        string
	Volumes      []string
	Restart      string
	MemoryMB     int
	CPUs         float64
	StopTimeout  time.Duration
	ExposedPorts []string
	PortBindings []PortBinding
	Healthcheck  *Healthcheck
	Command      []string
}

// List returns containers matching the given Docker filters.
func (s *Service) List(ctx context.Context, filters Filters) ([]Container, error) {
	cli, err := s.handle()
	if err != nil {
		return nil, err
	}

	result, err := cli.ContainerList(ctx, client.ContainerListOptions{All: true, Filters: filters})
	if err != nil {
		return nil, errors.New(FriendlyError(err))
	}

	items := make([]Container, 0, len(result.Items))
	for _, item := range result.Items {
		name := ""
		if len(item.Names) > 0 {
			name = strings.TrimPrefix(item.Names[0], "/")
		}

		items = append(items, Container{
			ID:      item.ID,
			Name:    name,
			Image:   item.Image,
			State:   string(item.State),
			Status:  item.Status,
			Health:  string(item.Health.Status),
			Created: item.Created,
			Ports:   containerPorts(item.Ports),
			Labels:  item.Labels,
		})
	}

	sort.Slice(items, func(i, j int) bool { return items[i].Created > items[j].Created })

	return items, nil
}

func containerPorts(ports []container.PortSummary) []string {
	result := []string{}

	for _, port := range ports {
		if port.PublicPort > 0 {
			result = append(result, fmt.Sprintf("%d->%d/%s", port.PublicPort, port.PrivatePort, port.Type))
			continue
		}

		result = append(result, fmt.Sprintf("%d/%s", port.PrivatePort, port.Type))
	}

	return result
}

// FindByResource returns the managed container of a resource, newest first.
func (s *Service) FindByResource(ctx context.Context, resourceType, resourceID string) ([]Container, error) {
	return s.List(ctx, ResourceFilter(resourceType, resourceID))
}

func (s *Service) Inspect(ctx context.Context, id string) (*container.InspectResponse, error) {
	cli, err := s.handle()
	if err != nil {
		return nil, err
	}

	result, err := cli.ContainerInspect(ctx, id, client.ContainerInspectOptions{})
	if err != nil {
		return nil, errors.New(FriendlyError(err))
	}

	return &result.Container, nil
}

// Create creates and starts a container, replacing any container with the same
// name.
func (s *Service) Create(ctx context.Context, spec Spec) (string, error) {
	cli, err := s.handle()
	if err != nil {
		return "", err
	}

	restart := spec.Restart
	if restart == "" {
		restart = "unless-stopped"
	}

	config := &container.Config{
		Image:  spec.Image,
		Env:    spec.Env,
		Labels: spec.Labels,
		Cmd:    spec.Command,
	}

	if len(spec.ExposedPorts) > 0 {
		ports := network.PortSet{}
		for _, port := range spec.ExposedPorts {
			parsed, err := network.ParsePort(port)
			if err != nil {
				return "", fmt.Errorf("docker: invalid port %q: %w", port, err)
			}
			ports[parsed] = struct{}{}
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

	if spec.StopTimeout > 0 {
		seconds := int(spec.StopTimeout.Seconds())
		config.StopTimeout = &seconds
	}

	hostConfig := &container.HostConfig{
		RestartPolicy: container.RestartPolicy{Name: container.RestartPolicyMode(restart)},
		Binds:         spec.Volumes,
	}

	if len(spec.PortBindings) > 0 {
		bindings := network.PortMap{}
		exposed := config.ExposedPorts
		if exposed == nil {
			exposed = network.PortSet{}
		}

		for _, binding := range spec.PortBindings {
			parsed, err := network.ParsePort(binding.ContainerPort)
			if err != nil {
				return "", fmt.Errorf("docker: invalid port %q: %w", binding.ContainerPort, err)
			}

			hostIP := netip.Addr{}
			if binding.HostIP != "" {
				if address, err := netip.ParseAddr(binding.HostIP); err == nil {
					hostIP = address
				}
			}

			exposed[parsed] = struct{}{}
			bindings[parsed] = append(bindings[parsed], network.PortBinding{
				HostIP:   hostIP,
				HostPort: binding.HostPort,
			})
		}

		config.ExposedPorts = exposed
		hostConfig.PortBindings = bindings
	}

	if spec.MemoryMB > 0 {
		hostConfig.Memory = int64(spec.MemoryMB) * 1024 * 1024
	}
	if spec.CPUs > 0 {
		hostConfig.NanoCPUs = int64(spec.CPUs * 1e9)
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

	result, err := cli.ContainerCreate(ctx, client.ContainerCreateOptions{
		Config:           config,
		HostConfig:       hostConfig,
		NetworkingConfig: networking,
		Name:             spec.Name,
	})
	if err != nil {
		return "", errors.New(FriendlyError(err))
	}

	if _, err := cli.ContainerStart(ctx, result.ID, client.ContainerStartOptions{}); err != nil {
		_ = s.Remove(context.WithoutCancel(ctx), result.ID, true)

		return "", errors.New(FriendlyError(err))
	}

	return result.ID, nil
}

func (s *Service) Start(ctx context.Context, id string) error {
	cli, err := s.handle()
	if err != nil {
		return err
	}

	if _, err := cli.ContainerStart(ctx, id, client.ContainerStartOptions{}); err != nil {
		return errors.New(FriendlyError(err))
	}

	return nil
}

func (s *Service) Stop(ctx context.Context, id string, timeout time.Duration) error {
	cli, err := s.handle()
	if err != nil {
		return err
	}

	options := client.ContainerStopOptions{}
	if timeout > 0 {
		seconds := int(timeout.Seconds())
		options.Timeout = &seconds
	}

	if _, err := cli.ContainerStop(ctx, id, options); err != nil {
		return errors.New(FriendlyError(err))
	}

	return nil
}

func (s *Service) Restart(ctx context.Context, id string, timeout time.Duration) error {
	cli, err := s.handle()
	if err != nil {
		return err
	}

	options := client.ContainerRestartOptions{}
	if timeout > 0 {
		seconds := int(timeout.Seconds())
		options.Timeout = &seconds
	}

	if _, err := cli.ContainerRestart(ctx, id, options); err != nil {
		return errors.New(FriendlyError(err))
	}

	return nil
}

func (s *Service) Remove(ctx context.Context, id string, force bool) error {
	cli, err := s.handle()
	if err != nil {
		return err
	}

	if _, err := cli.ContainerRemove(ctx, id, client.ContainerRemoveOptions{
		Force:         force,
		RemoveVolumes: false,
	}); err != nil {
		if strings.Contains(err.Error(), "No such container") {
			return nil
		}

		return errors.New(FriendlyError(err))
	}

	return nil
}

// RemoveByName deletes a container by name, ignoring missing containers.
func (s *Service) RemoveByName(ctx context.Context, name string, force bool) error {
	cli, err := s.handle()
	if err != nil {
		return err
	}

	if _, err := cli.ContainerRemove(ctx, name, client.ContainerRemoveOptions{Force: force}); err != nil {
		if strings.Contains(err.Error(), "No such container") {
			return nil
		}

		return errors.New(FriendlyError(err))
	}

	return nil
}

// WaitRunning polls until the container is running, exits, or the timeout
// elapses.
func (s *Service) WaitRunning(ctx context.Context, id string, timeout time.Duration) (*container.InspectResponse, error) {
	deadline := time.Now().Add(timeout)

	for {
		inspect, err := s.Inspect(ctx, id)
		if err != nil {
			return nil, err
		}

		if inspect.State != nil {
			if inspect.State.Running {
				return inspect, nil
			}
			if inspect.State.Status == container.StateExited || inspect.State.Status == container.StateDead {
				return inspect, fmt.Errorf("docker: container exited with code %d", inspect.State.ExitCode)
			}
		}

		if time.Now().After(deadline) {
			return inspect, fmt.Errorf("docker: container did not start within %s", timeout)
		}

		select {
		case <-ctx.Done():
			return inspect, ctx.Err()
		case <-time.After(400 * time.Millisecond):
		}
	}
}

// WaitHealthy waits for a container healthcheck to report healthy, if the
// container defines one.
func (s *Service) WaitHealthy(ctx context.Context, id string, timeout time.Duration) error {
	deadline := time.Now().Add(timeout)

	for {
		inspect, err := s.Inspect(ctx, id)
		if err != nil {
			return err
		}

		if inspect.State == nil || inspect.State.Health == nil {
			return nil
		}

		status := inspect.State.Health.Status
		if status == "healthy" {
			return nil
		}
		if status == "unhealthy" {
			return errors.New("docker: container healthcheck reported unhealthy")
		}

		if time.Now().After(deadline) {
			return fmt.Errorf("docker: container did not become healthy within %s", timeout)
		}

		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(time.Second):
		}
	}
}

// Logs returns a log snapshot; tail -1 returns everything.
func (s *Service) Logs(ctx context.Context, id string, tail int) (string, error) {
	cli, err := s.handle()
	if err != nil {
		return "", err
	}

	tailValue := "all"
	if tail > 0 {
		tailValue = fmt.Sprintf("%d", tail)
	}

	reader, err := cli.ContainerLogs(ctx, id, client.ContainerLogsOptions{
		ShowStdout: true,
		ShowStderr: true,
		Tail:       tailValue,
	})
	if err != nil {
		return "", errors.New(FriendlyError(err))
	}
	defer reader.Close()

	inspect, err := s.Inspect(ctx, id)
	if err != nil {
		return "", err
	}

	var builder strings.Builder
	if inspect.Config != nil && inspect.Config.Tty {
		if _, err := io.Copy(&builder, reader); err != nil {
			return "", err
		}
	} else if _, err := stdcopy.StdCopy(&builder, &builder, reader); err != nil {
		return "", err
	}

	return builder.String(), nil
}

// StreamLogs follows container logs, writing demultiplexed output to writer
// until the context is cancelled.
func (s *Service) StreamLogs(ctx context.Context, id string, tail int, writer io.Writer) error {
	cli, err := s.handle()
	if err != nil {
		return err
	}

	tailValue := "all"
	if tail > 0 {
		tailValue = fmt.Sprintf("%d", tail)
	}

	reader, err := cli.ContainerLogs(ctx, id, client.ContainerLogsOptions{
		ShowStdout: true,
		ShowStderr: true,
		Follow:     true,
		Tail:       tailValue,
	})
	if err != nil {
		return errors.New(FriendlyError(err))
	}
	defer reader.Close()

	inspect, err := s.Inspect(ctx, id)
	if err != nil {
		return err
	}

	if inspect.Config != nil && inspect.Config.Tty {
		_, err = io.Copy(writer, reader)
	} else {
		_, err = stdcopy.StdCopy(writer, writer, reader)
	}
	if err != nil && ctx.Err() == nil && !strings.Contains(err.Error(), "use of closed") {
		return err
	}

	return nil
}

// StatsBatch collects stats for many containers in parallel. Container stats
// sampling is dominated by waiting on the daemon, so fanning out keeps the
// endpoint responsive when many containers are running.
func (s *Service) StatsBatch(ctx context.Context, ids []string) map[string]ContainerStats {
	results := make(map[string]ContainerStats, len(ids))

	if len(ids) == 0 {
		return results
	}

	var (
		mu       sync.Mutex
		wait     sync.WaitGroup
		limiters = make(chan struct{}, 8)
	)

	for _, id := range ids {
		wait.Add(1)
		limiters <- struct{}{}

		go func(containerID string) {
			defer wait.Done()
			defer func() { <-limiters }()

			stats, err := s.Stats(ctx, containerID)
			if err != nil {
				return
			}

			mu.Lock()
			results[containerID] = stats
			mu.Unlock()
		}(id)
	}

	wait.Wait()

	return results
}

func (s *Service) Stats(ctx context.Context, id string) (ContainerStats, error) {
	cli, err := s.handle()
	if err != nil {
		return ContainerStats{}, err
	}

	result, err := cli.ContainerStats(ctx, id, client.ContainerStatsOptions{IncludePreviousSample: true})
	if err != nil {
		return ContainerStats{}, errors.New(FriendlyError(err))
	}
	defer result.Body.Close()

	var response container.StatsResponse
	if err := json.NewDecoder(result.Body).Decode(&response); err != nil {
		return ContainerStats{}, err
	}

	return calculateStats(response), nil
}

// ContainerIP returns the container IP on the given network, empty when the
// container is not attached.
func (s *Service) ContainerIP(inspect *container.InspectResponse, networkName string) string {
	if inspect == nil || inspect.NetworkSettings == nil {
		return ""
	}

	if networkName != "" {
		if endpoint, ok := inspect.NetworkSettings.Networks[networkName]; ok && endpoint != nil {
			return endpoint.IPAddress.String()
		}

		return ""
	}

	for _, endpoint := range inspect.NetworkSettings.Networks {
		if endpoint != nil && endpoint.IPAddress.IsValid() {
			return endpoint.IPAddress.String()
		}
	}

	return ""
}

func calculateStats(response container.StatsResponse) ContainerStats {
	stats := ContainerStats{
		MemoryUsage: response.MemoryStats.Usage,
		MemoryLimit: response.MemoryStats.Limit,
	}

	if response.MemoryStats.Stats != nil {
		if cache, ok := response.MemoryStats.Stats["inactive_file"]; ok && cache < stats.MemoryUsage {
			stats.MemoryUsage -= cache
		}
	}

	if stats.MemoryLimit > 0 {
		stats.MemoryPercent = float64(stats.MemoryUsage) / float64(stats.MemoryLimit) * 100
	}

	cpuDelta := float64(response.CPUStats.CPUUsage.TotalUsage) - float64(response.PreCPUStats.CPUUsage.TotalUsage)
	systemDelta := float64(response.CPUStats.SystemUsage) - float64(response.PreCPUStats.SystemUsage)

	if cpuDelta > 0 && systemDelta > 0 {
		cpus := float64(response.CPUStats.OnlineCPUs)
		if cpus == 0 {
			cpus = float64(len(response.CPUStats.CPUUsage.PercpuUsage))
		}
		if cpus == 0 {
			cpus = 1
		}

		stats.CPUPercent = cpuDelta / systemDelta * cpus * 100
	}

	return stats
}
