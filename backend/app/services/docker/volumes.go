package docker

import (
	"context"
	"errors"
	"sort"
	"strings"

	"github.com/moby/moby/client"
)

// EnsureVolume creates a named volume when it does not exist yet.
func (s *Service) EnsureVolume(ctx context.Context, name string, labels map[string]string) error {
	cli, err := s.handle()
	if err != nil {
		return err
	}

	_, err = cli.VolumeCreate(ctx, client.VolumeCreateOptions{Name: name, Labels: labels})
	if err != nil && !strings.Contains(err.Error(), "already exists") {
		return errors.New(FriendlyError(err))
	}

	return nil
}

// RemoveVolume deletes a named volume, ignoring missing volumes.
func (s *Service) RemoveVolume(ctx context.Context, name string) error {
	if name == "" {
		return nil
	}

	cli, err := s.handle()
	if err != nil {
		return err
	}

	if _, err := cli.VolumeRemove(ctx, name, client.VolumeRemoveOptions{Force: true}); err != nil {
		if strings.Contains(err.Error(), "no such volume") {
			return nil
		}

		return errors.New(FriendlyError(err))
	}

	return nil
}

// ComposeFilter selects containers created by a compose project.
func ComposeFilter(project string) Filters {
	filters := Filters{}.Add("label", "com.docker.compose.project="+project)
	if project == "" {
		filters = Filters{}.Add("label", "com.docker.compose.project")
	}

	return filters
}

// Volume is a docker volume summary.
type Volume struct {
	Name       string            `json:"name"`
	Driver     string            `json:"driver"`
	Mountpoint string            `json:"mountpoint,omitempty"`
	CreatedAt  string            `json:"created_at,omitempty"`
	Labels     map[string]string `json:"labels,omitempty"`
}

// ListVolumes returns volumes, optionally only those labelled as managed by ebox.
func (s *Service) ListVolumes(ctx context.Context, managedOnly bool) ([]Volume, error) {
	cli, err := s.handle()
	if err != nil {
		return nil, err
	}

	filters := Filters{}
	if managedOnly {
		filters = filters.Add("label", LabelManaged+"=true")
	}

	result, err := cli.VolumeList(ctx, client.VolumeListOptions{Filters: filters})
	if err != nil {
		return nil, errors.New(FriendlyError(err))
	}

	items := make([]Volume, 0, len(result.Items))
	for _, volume := range result.Items {
		items = append(items, Volume{
			Name:       volume.Name,
			Driver:     volume.Driver,
			Mountpoint: volume.Mountpoint,
			CreatedAt:  volume.CreatedAt,
			Labels:     volume.Labels,
		})
	}

	sort.Slice(items, func(i, j int) bool { return items[i].Name < items[j].Name })

	return items, nil
}
