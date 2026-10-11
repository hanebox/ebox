package build

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"ebox/app/models"
	"ebox/app/services/docker"
)

type Request struct {
	App       *models.Application
	Directory string
	Tag       string
	BuildArgs map[string]string
	Env       []string
	Sink      LineSink
}

// Image builds (or pulls) the image for an application revision.
func Image(ctx context.Context, request Request) error {
	switch request.App.BuildPack {
	case models.BuildPackDockerfile:
		return buildDockerfile(ctx, request)
	case models.BuildPackStatic:
		return buildStatic(ctx, request)
	case models.BuildPackImage:
		return pullImage(ctx, request)
	}

	return fmt.Errorf("build: unsupported build pack %q", request.App.BuildPack)
}

func buildDockerfile(ctx context.Context, request Request) error {
	contextDir := filepath.Join(request.Directory, request.App.BuildContextDirectory())

	dockerfile := request.App.Config.DockerfilePath
	if dockerfile == "" {
		dockerfile = "Dockerfile"
	}

	if _, err := os.Stat(filepath.Join(contextDir, dockerfile)); err != nil {
		return fmt.Errorf("build: dockerfile %q not found in the repository", dockerfile)
	}

	args := []string{"build", "--progress=plain", "-t", request.Tag, "-f", dockerfile}
	args = append(args, labelFlags(request.App)...)
	args = append(args, buildArgFlags(request.BuildArgs)...)
	args = append(args, ".")

	sinkLine(request.Sink, "$ docker "+strings.Join(args, " "))

	return Run(ctx, Command{Dir: contextDir, Env: request.Env, Sink: request.Sink}, "docker", args...)
}

func buildStatic(ctx context.Context, request Request) error {
	contextDir := filepath.Join(request.Directory, request.App.BuildContextDirectory())

	dockerfile, err := StaticDockerfile(request.App, contextDir)
	if err != nil {
		return err
	}

	if err := os.WriteFile(filepath.Join(contextDir, "ebox.static.Dockerfile"), []byte(dockerfile), 0o644); err != nil {
		return err
	}

	if err := os.WriteFile(filepath.Join(contextDir, "ebox.Caddyfile"), []byte(StaticCaddyfile), 0o644); err != nil {
		return err
	}

	for _, line := range strings.Split(strings.TrimRight(dockerfile, "\n"), "\n") {
		sinkLine(request.Sink, "| "+line)
	}

	args := []string{"build", "--progress=plain", "-t", request.Tag, "-f", "ebox.static.Dockerfile"}
	args = append(args, labelFlags(request.App)...)
	args = append(args, buildArgFlags(request.BuildArgs)...)
	args = append(args, ".")

	sinkLine(request.Sink, "$ docker "+strings.Join(args, " "))

	return Run(ctx, Command{Dir: contextDir, Env: request.Env, Sink: request.Sink}, "docker", args...)
}

func pullImage(ctx context.Context, request Request) error {
	image := strings.TrimSpace(request.App.Config.Image)
	if image == "" {
		return errors.New("build: no image configured for this application")
	}

	sinkLine(request.Sink, "$ docker pull "+image)
	if err := docker.Default().PullImage(ctx, image, request.Sink); err != nil {
		return err
	}

	if err := docker.Default().TagImage(ctx, image, request.Tag); err != nil {
		return err
	}

	sinkLine(request.Sink, "Tagged "+request.Tag)

	return nil
}

func labelFlags(app *models.Application) []string {
	labels := docker.ManagedLabels(docker.ResourceApplication, app.UUID, app.Name)

	keys := make([]string, 0, len(labels))
	for key := range labels {
		keys = append(keys, key)
	}
	sort.Strings(keys)

	flags := make([]string, 0, len(keys)*2)
	for _, key := range keys {
		flags = append(flags, "--label", key+"="+labels[key])
	}

	return flags
}

func buildArgFlags(args map[string]string) []string {
	if len(args) == 0 {
		return nil
	}

	keys := make([]string, 0, len(args))
	for key := range args {
		keys = append(keys, key)
	}
	sort.Strings(keys)

	flags := make([]string, 0, len(keys)*2)
	for _, key := range keys {
		flags = append(flags, "--build-arg", key+"="+args[key])
	}

	return flags
}

func sinkLine(sink LineSink, line string) {
	if sink != nil {
		sink(line)
	}
}
