package docker

import (
	"bufio"
	"context"
	"encoding/json"
	"errors"
	"strings"

	"github.com/moby/moby/api/types/image"
	"github.com/moby/moby/client"
)

type Image struct {
	ID      string   `json:"id"`
	Tags    []string `json:"tags"`
	Size    int64    `json:"size"`
	Created int64    `json:"created"`
}

// PullImage pulls an image, reporting progress lines.
func (s *Service) PullImage(ctx context.Context, ref string, sink func(string)) error {
	cli, err := s.handle()
	if err != nil {
		return err
	}

	reader, err := cli.ImagePull(ctx, ref, client.ImagePullOptions{})
	if err != nil {
		return errors.New(FriendlyError(err))
	}
	defer reader.Close()

	scanner := bufio.NewScanner(reader)
	scanner.Buffer(make([]byte, 0, 64*1024), 1024*1024)

	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}

		var payload struct {
			Status   string `json:"status"`
			Progress string `json:"progress"`
			Error    string `json:"error"`
			Stream   string `json:"stream"`
		}
		if err := json.Unmarshal([]byte(line), &payload); err != nil {
			continue
		}

		if sink != nil {
			switch {
			case payload.Error != "":
				sink("error: " + payload.Error)
			case payload.Stream != "":
				sink(strings.TrimRight(payload.Stream, "\n"))
			case payload.Progress != "":
				sink(payload.Status + " " + payload.Progress)
			case payload.Status != "":
				sink(payload.Status)
			}
		}

		if payload.Error != "" {
			return errors.New(payload.Error)
		}
	}

	if err := scanner.Err(); err != nil && ctx.Err() == nil {
		return err
	}

	return ctx.Err()
}

func (s *Service) TagImage(ctx context.Context, source, target string) error {
	cli, err := s.handle()
	if err != nil {
		return err
	}

	if _, err := cli.ImageTag(ctx, client.ImageTagOptions{Source: source, Target: target}); err != nil {
		return errors.New(FriendlyError(err))
	}

	return nil
}

func (s *Service) RemoveImage(ctx context.Context, ref string, force bool) error {
	cli, err := s.handle()
	if err != nil {
		return err
	}

	if _, err := cli.ImageRemove(ctx, ref, client.ImageRemoveOptions{Force: force, PruneChildren: true}); err != nil {
		if strings.Contains(err.Error(), "No such image") {
			return nil
		}

		return errors.New(FriendlyError(err))
	}

	return nil
}

// ListImages returns images whose repository matches the given prefix.
func (s *Service) ListImages(ctx context.Context, repository string) ([]Image, error) {
	cli, err := s.handle()
	if err != nil {
		return nil, err
	}

	result, err := cli.ImageList(ctx, client.ImageListOptions{
		All:     true,
		Filters: client.Filters{}.Add("reference", repository+":*"),
	})
	if err != nil {
		return nil, errors.New(FriendlyError(err))
	}

	items := make([]Image, 0, len(result.Items))
	for _, item := range result.Items {
		items = append(items, Image{
			ID:      item.ID,
			Tags:    filterTags(item, repository),
			Size:    item.Size,
			Created: item.Created,
		})
	}

	return items, nil
}

func filterTags(item image.Summary, repository string) []string {
	tags := make([]string, 0, len(item.RepoTags))
	for _, tag := range item.RepoTags {
		if strings.HasPrefix(tag, repository+":") {
			tags = append(tags, tag)
		}
	}

	return tags
}
