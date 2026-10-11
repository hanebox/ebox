package compose

import (
	"errors"
	"fmt"
	"sort"
	"strconv"
	"strings"

	"gopkg.in/yaml.v3"
)

// Document is the subset of a compose file ebox needs to reason about.
type Document struct {
	Services map[string]Service `yaml:"services"`
}

type Service struct {
	Image       string         `yaml:"image"`
	Build       any            `yaml:"build"`
	Ports       []any          `yaml:"ports"`
	Expose      []any          `yaml:"expose"`
	Environment any            `yaml:"environment"`
	DependsOn   any            `yaml:"depends_on"`
	Command     any            `yaml:"command"`
	Volumes     []any          `yaml:"volumes"`
	Restart     string         `yaml:"restart"`
	Profiles    []string       `yaml:"profiles"`
	Healthcheck map[string]any `yaml:"healthcheck"`
	Networks    any            `yaml:"networks"`
	Extra       map[string]any `yaml:",inline"`
}

// Info summarises one service for the UI.
type Info struct {
	Name       string   `json:"name"`
	Image      string   `json:"image"`
	Built      bool     `json:"build"`
	Ports      []string `json:"ports"`
	DependsOn  []string `json:"depends_on"`
	HasVolumes bool     `json:"has_volumes"`
}

// Parse reads a compose file, rejecting files without services.
func Parse(content string) (*Document, error) {
	if strings.TrimSpace(content) == "" {
		return nil, errors.New("compose: the file is empty")
	}

	var document Document
	if err := yaml.Unmarshal([]byte(content), &document); err != nil {
		return nil, fmt.Errorf("compose: the file is not valid YAML: %w", err)
	}

	if len(document.Services) == 0 {
		return nil, errors.New("compose: the file does not define any service")
	}

	return &document, nil
}

// ServiceNames returns the service names in a stable order.
func (d *Document) ServiceNames() []string {
	names := make([]string, 0, len(d.Services))
	for name := range d.Services {
		names = append(names, name)
	}

	sort.Strings(names)

	return names
}

// Info summarises every service in the document.
func (d *Document) Info() []Info {
	names := d.ServiceNames()
	list := make([]Info, 0, len(names))

	for _, name := range names {
		service := d.Services[name]

		info := Info{
			Name:       name,
			Image:      service.Image,
			Built:      service.Build != nil,
			Ports:      PortStrings(service.Ports),
			DependsOn:  DependencyNames(service.DependsOn),
			HasVolumes: len(service.Volumes) > 0,
		}

		list = append(list, info)
	}

	return list
}

// PortStrings normalises the many port notations into "published:target" or
// "target" strings.
func PortStrings(ports []any) []string {
	result := []string{}

	for _, entry := range ports {
		switch value := entry.(type) {
		case string:
			result = append(result, value)
		case int:
			result = append(result, strconv.Itoa(value))
		case map[string]any:
			target := fmt.Sprint(value["target"])
			published := value["published"]
			if published == nil || fmt.Sprint(published) == "" {
				result = append(result, target)

				continue
			}
			result = append(result, fmt.Sprint(published)+":"+target)
		}
	}

	return result
}

// DependencyNames normalises depends_on notations into a name list.
func DependencyNames(value any) []string {
	names := []string{}

	switch typed := value.(type) {
	case []any:
		for _, entry := range typed {
			names = append(names, fmt.Sprint(entry))
		}
	case map[string]any:
		for name := range typed {
			names = append(names, name)
		}
	}

	sort.Strings(names)

	return names
}

// PublishedPort returns the host port a service publishes, when any. The first
// published port wins, which matches the common "one exposed port" convention.
func (s Service) PublishedPort() int {
	for _, entry := range PortStrings(s.Ports) {
		parts := strings.Split(entry, ":")
		if len(parts) < 2 {
			continue
		}

		host := parts[len(parts)-2]
		if port, err := strconv.Atoi(host); err == nil && port > 0 && port < 65536 {
			return port
		}
	}

	return 0
}

// ContainerPort returns the container side port of a service.
func (s Service) ContainerPort() int {
	for _, entry := range PortStrings(s.Ports) {
		parts := strings.Split(entry, ":")
		port, err := strconv.Atoi(parts[len(parts)-1])
		if err == nil && port > 0 && port < 65536 {
			return port
		}
	}

	for _, entry := range s.Expose {
		port, err := strconv.Atoi(fmt.Sprint(entry))
		if err == nil && port > 0 && port < 65536 {
			return port
		}
	}

	return 0
}

// EntryCandidates returns the services that look like the public entrypoint,
// ordered by suitability: published ports first, then exposed, then alphabetical.
func (d *Document) EntryCandidates() []Info {
	withPublished := []Info{}
	withExposed := []Info{}
	rest := []Info{}

	for _, info := range d.Info() {
		service := d.Services[info.Name]

		switch {
		case service.PublishedPort() > 0:
			withPublished = append(withPublished, info)
		case service.ContainerPort() > 0:
			withExposed = append(withExposed, info)
		default:
			rest = append(rest, info)
		}
	}

	return append(append(withPublished, withExposed...), rest...)
}
