package docker

import (
	"context"
	"errors"

	"github.com/moby/moby/client"
)

type UsageSection struct {
	Count       int64 `json:"count"`
	Active      int64 `json:"active"`
	Size        int64 `json:"size"`
	Reclaimable int64 `json:"reclaimable"`
}

type DiskUsage struct {
	Images     UsageSection `json:"images"`
	Containers UsageSection `json:"containers"`
	Volumes    UsageSection `json:"volumes"`
	BuildCache UsageSection `json:"build_cache"`
	TotalSize  int64        `json:"total_size"`
}

type PruneResult struct {
	Images         int   `json:"images"`
	Containers     int   `json:"containers"`
	Volumes        int   `json:"volumes"`
	BuildCache     int   `json:"build_cache"`
	SpaceReclaimed int64 `json:"space_reclaimed"`
}

func (s *Service) DiskUsage(ctx context.Context) (DiskUsage, error) {
	cli, err := s.handle()
	if err != nil {
		return DiskUsage{}, err
	}

	result, err := cli.DiskUsage(ctx, client.DiskUsageOptions{
		Containers: true,
		Images:     true,
		Volumes:    true,
		BuildCache: true,
	})
	if err != nil {
		return DiskUsage{}, errors.New(FriendlyError(err))
	}

	usage := DiskUsage{
		Images: UsageSection{
			Count:       result.Images.TotalCount,
			Active:      result.Images.ActiveCount,
			Size:        result.Images.TotalSize,
			Reclaimable: result.Images.Reclaimable,
		},
		Containers: UsageSection{
			Count:       result.Containers.TotalCount,
			Active:      result.Containers.ActiveCount,
			Size:        result.Containers.TotalSize,
			Reclaimable: result.Containers.Reclaimable,
		},
		Volumes: UsageSection{
			Count:       result.Volumes.TotalCount,
			Active:      result.Volumes.ActiveCount,
			Size:        result.Volumes.TotalSize,
			Reclaimable: result.Volumes.Reclaimable,
		},
		BuildCache: UsageSection{
			Count:       result.BuildCache.TotalCount,
			Active:      result.BuildCache.ActiveCount,
			Size:        result.BuildCache.TotalSize,
			Reclaimable: result.BuildCache.Reclaimable,
		},
	}

	usage.TotalSize = usage.Images.Size + usage.Containers.Size + usage.Volumes.Size + usage.BuildCache.Size

	return usage, nil
}

// Prune removes unused data for the requested targets. Supported targets are
// images, containers, volumes and build_cache. Managed containers, images in
// use and the ebox image namespace are never removed.
func (s *Service) Prune(ctx context.Context, targets []string) (PruneResult, error) {
	cli, err := s.handle()
	if err != nil {
		return PruneResult{}, err
	}

	result := PruneResult{}

	for _, target := range targets {
		switch target {
		case "containers":
			pruned, err := cli.ContainerPrune(ctx, client.ContainerPruneOptions{
				Filters: Filters{}.Add("label!", LabelManaged),
			})
			if err != nil {
				return result, errors.New(FriendlyError(err))
			}
			result.Containers += len(pruned.Report.ContainersDeleted)
			result.SpaceReclaimed += int64(pruned.Report.SpaceReclaimed)
		case "images":
			pruned, err := cli.ImagePrune(ctx, client.ImagePruneOptions{
				Filters: Filters{}.Add("dangling", "false"),
			})
			if err != nil {
				return result, errors.New(FriendlyError(err))
			}
			result.Images += len(pruned.Report.ImagesDeleted)
			result.SpaceReclaimed += int64(pruned.Report.SpaceReclaimed)
		case "volumes":
			pruned, err := cli.VolumePrune(ctx, client.VolumePruneOptions{})
			if err != nil {
				return result, errors.New(FriendlyError(err))
			}
			result.Volumes += len(pruned.Report.VolumesDeleted)
			result.SpaceReclaimed += int64(pruned.Report.SpaceReclaimed)
		case "build_cache":
			pruned, err := cli.BuildCachePrune(ctx, client.BuildCachePruneOptions{All: true})
			if err != nil {
				return result, errors.New(FriendlyError(err))
			}
			result.BuildCache += len(pruned.Report.CachesDeleted)
			result.SpaceReclaimed += int64(pruned.Report.SpaceReclaimed)
		default:
			return result, errors.New("docker: unsupported prune target " + target)
		}
	}

	return result, nil
}
