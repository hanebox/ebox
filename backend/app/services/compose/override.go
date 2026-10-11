package compose

import (
	"sort"

	"gopkg.in/yaml.v3"
)

// Override describes what ebox adds on top of a user compose file.
type Override struct {
	Project     string
	Network     string
	Labels      map[string]string
	Environment map[string]string
	// EntryService receives the ingress labels and published environment.
	EntryService string
	EntryLabels  map[string]string
	EntryEnv     map[string]string
	// EntryPort overrides the container port Traefik should target.
	EntryPort int
}

type overrideDocument struct {
	Services map[string]overrideService `yaml:"services"`
	Networks map[string]overrideNetwork `yaml:"networks"`
}

type overrideService struct {
	Labels      map[string]string `yaml:"labels,omitempty"`
	Environment map[string]string `yaml:"environment,omitempty"`
	Networks    []string          `yaml:"networks,omitempty"`
}

type overrideNetwork struct {
	External bool   `yaml:"external"`
	Name     string `yaml:"name,omitempty"`
}

// Render produces the override file that makes a compose project a managed
// ebox resource: consistent labels, the shared network and ingress routing.
func Render(document *Document, options Override) (string, error) {
	if options.Network == "" {
		options.Network = "ebox"
	}

	services := map[string]overrideService{}

	for _, name := range document.ServiceNames() {
		labels := map[string]string{}
		for key, value := range options.Labels {
			labels[key] = value
		}
		labels["ebox.service"] = name

		environment := map[string]string{}
		for key, value := range options.Environment {
			environment[key] = value
		}

		if name == options.EntryService {
			for key, value := range options.EntryLabels {
				labels[key] = value
			}
			for key, value := range options.EntryEnv {
				environment[key] = value
			}
		}

		service := overrideService{Networks: []string{options.Network}}
		if len(labels) > 0 {
			service.Labels = labels
		}
		if len(environment) > 0 {
			service.Environment = environment
		}

		services[name] = service
	}

	document_ := overrideDocument{
		Services: services,
		Networks: map[string]overrideNetwork{
			options.Network: {External: true},
		},
	}

	content, err := yaml.Marshal(document_)
	if err != nil {
		return "", err
	}

	return string(content), nil
}

// ServiceContainerName returns the container name compose gives a service in a
// project: <project>-<service>-<index>.
func ServiceContainerName(project, service string) string {
	return project + "-" + service + "-1"
}

// SortedLabels renders labels for logging.
func SortedLabels(labels map[string]string) []string {
	keys := make([]string, 0, len(labels))
	for key := range labels {
		keys = append(keys, key)
	}

	sort.Strings(keys)

	return keys
}
