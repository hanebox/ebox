package proxy

import (
	"context"
	"crypto/x509"
	"encoding/json"
	"encoding/pem"
	"errors"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"gopkg.in/yaml.v3"

	"ebox/app/facades"
	"ebox/app/services/build"
	"ebox/app/services/docker"
	"ebox/app/services/settings"
)

const (
	containerName = "ebox-traefik"
	composeFile   = "docker-compose.yml"
	panelFile     = "panel.yml"
	middlewareDir = "dynamic"
	defaultImage  = "traefik:v3"
)

// Traefik manages the Traefik container that fronts every application.
type Traefik struct{}

func NewTraefik() *Traefik {
	return &Traefik{}
}

func (t *Traefik) Directory() string {
	return facades.Config().GetString("ebox.traefik_dir", "storage/ebox/traefik")
}

func (t *Traefik) dynamicDirectory() string {
	return filepath.Join(t.Directory(), middlewareDir)
}

func (t *Traefik) composePath() string {
	return filepath.Join(t.Directory(), composeFile)
}

func (t *Traefik) acmePath() string {
	return filepath.Join(t.Directory(), "acme", "acme.json")
}

// Settings collects the proxy configuration from the panel settings.
func (t *Traefik) Settings() Settings {
	return Settings{
		PanelDomain: settings.Default().Get("panel_domain", ""),
		ACMEEmail:   settings.Default().Get("acme_email", ""),
		HTTPPort:    80,
		HTTPSPort:   443,
	}
}

func (t *Traefik) network() string {
	network := facades.Config().GetString("ebox.docker.network", DefaultNetwork)

	return network
}

// Ensure writes the compose and dynamic configuration and starts the proxy.
func (t *Traefik) Ensure(ctx context.Context) error {
	if err := t.WriteConfiguration(); err != nil {
		return err
	}

	if err := docker.Default().EnsureNetwork(ctx, t.network()); err != nil {
		return err
	}

	return t.compose(ctx, "up", "-d", "--remove-orphans")
}

func (t *Traefik) Start(ctx context.Context) error {
	if _, err := os.Stat(t.composePath()); err != nil {
		return t.Ensure(ctx)
	}

	return t.compose(ctx, "up", "-d", "--remove-orphans")
}

func (t *Traefik) Stop(ctx context.Context) error {
	if _, err := os.Stat(t.composePath()); err != nil {
		return nil
	}

	return t.compose(ctx, "down", "--remove-orphans")
}

func (t *Traefik) Restart(ctx context.Context) error {
	if err := t.WriteConfiguration(); err != nil {
		return err
	}

	return t.compose(ctx, "up", "-d", "--force-recreate", "--remove-orphans")
}

// WriteConfiguration (re)generates the compose file, the panel route and the
// middleware directory so a restart always reflects the current settings.
func (t *Traefik) WriteConfiguration() error {
	if err := os.MkdirAll(t.dynamicDirectory(), 0o755); err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(t.acmePath()), 0o700); err != nil {
		return err
	}

	if _, err := os.Stat(t.acmePath()); err != nil {
		if err := os.WriteFile(t.acmePath(), []byte("{}"), 0o600); err != nil {
			return err
		}
	}

	if err := os.WriteFile(t.composePath(), []byte(t.ComposeFile(t.Settings())), 0o600); err != nil {
		return err
	}

	return t.writePanelRoute(t.Settings())
}

// ComposeFile renders the Traefik compose file for the given settings.
func (t *Traefik) ComposeFile(config Settings) string {
	email := config.ACMEEmail
	if email == "" {
		email = "admin@example.com"
	}

	httpPort := config.HTTPPort
	if httpPort <= 0 {
		httpPort = 80
	}
	httpsPort := config.HTTPSPort
	if httpsPort <= 0 {
		httpsPort = 443
	}

	directory := t.Directory()

	apiPort := t.apiPort()

	return fmt.Sprintf(`services:
  traefik:
    image: %s
    container_name: %s
    restart: unless-stopped
    ports:
      - "%d:80"
      - "%d:443"
      - "127.0.0.1:%s:8080"
    extra_hosts:
      - "host.docker.internal:host-gateway"
    command:
      - --global.checknewversion=false
      - --global.sendanonymoususage=false
      - --api.dashboard=true
      - --api.insecure=true
      - --providers.docker=true
      - --providers.docker.exposedbydefault=false
      - --providers.docker.network=%s
      - --providers.file.directory=/etc/traefik/dynamic
      - --providers.file.watch=true
      - --entrypoints.web.address=:80
      - --entrypoints.websecure.address=:443
      - --certificatesresolvers.%s.acme.email=%s
      - --certificatesresolvers.%s.acme.storage=/acme/acme.json
      - --certificatesresolvers.%s.acme.httpchallenge.entrypoint=web
    volumes:
      - /var/run/docker.sock:/var/run/docker.sock:ro
      - %s:/etc/traefik/dynamic
      - %s:/acme
    labels:
      ebox.managed: "true"
      ebox.resource-type: "proxy"
      ebox.name: "traefik"
    networks:
      - %s

networks:
  %s:
    external: true
`, defaultImage, containerName, httpPort, httpsPort, apiPort, t.network(), CertResolver, email, CertResolver, CertResolver,
		filepath.Join(directory, middlewareDir), filepath.Join(directory, "acme"), t.network(), t.network())
}

// writePanelRoute publishes the panel itself through the proxy when a panel
// domain is configured.
func (t *Traefik) writePanelRoute(config Settings) error {
	path := filepath.Join(t.dynamicDirectory(), panelFile)

	if config.PanelDomain == "" {
		if err := os.Remove(path); err != nil && !os.IsNotExist(err) {
			return err
		}

		return nil
	}

	port := facades.Config().GetString("ebox.port", "8000")

	content := fmt.Sprintf(`http:
  routers:
    ebox-panel:
      rule: Host(`+"`%s`"+`)
      entryPoints:
        - web
        - websecure
      service: ebox-panel
      tls:
        certResolver: %s
  services:
    ebox-panel:
      loadBalancer:
        servers:
          - url: http://host.docker.internal:%s
`, config.PanelDomain, CertResolver, port)

	return os.WriteFile(path, []byte(content), 0o600)
}

func (t *Traefik) compose(ctx context.Context, args ...string) error {
	if _, err := os.Stat(t.composePath()); err != nil {
		return errors.New("proxy: the proxy is not configured yet")
	}

	command := append([]string{"compose", "-f", t.composePath()}, args...)

	output := &strings.Builder{}
	err := build.Run(ctx, build.Command{Dir: t.Directory(), Sink: func(line string) {
		output.WriteString(line)
		output.WriteString("\n")
	}}, "docker", command...)
	if err != nil {
		message := strings.TrimSpace(output.String())
		if strings.Contains(message, "port is already allocated") {
			message = "A port the proxy needs is already in use. Free it or set EBOX_TRAEFIK_API_PORT to a free port. " + message
		}
		if message == "" {
			message = err.Error()
		}

		facades.Log().Errorf("ebox: docker %s failed: %s", strings.Join(command, " "), err)

		return errors.New(message)
	}

	return nil
}

func (t *Traefik) Status(ctx context.Context) Status {
	status := Status{State: "missing"}

	if _, err := os.Stat(t.composePath()); err == nil {
		status.Installed = true
	}

	containers, err := docker.Default().FindByResource(ctx, docker.ResourceProxy, "")
	if err == nil && len(containers) > 0 {
		container := containers[0]
		status.ContainerID = container.ID
		status.Container = container.Name
		status.State = container.State
		status.Running = container.State == "running"
		status.Image = container.Image
	}

	if version, err := t.version(ctx); err == nil {
		status.Version = version
		status.APIReachable = true
	}

	return status
}

// apiPort is the local port the Traefik API is published on.
func (t *Traefik) apiPort() string {
	return facades.Config().GetString("ebox.traefik.api_port", "8080")
}

// apiURL is the local address of the Traefik API.
func (t *Traefik) apiURL() string {
	if override := facades.Config().GetString("ebox.traefik.api_url", ""); override != "" {
		return strings.TrimRight(override, "/")
	}

	return "http://127.0.0.1:" + t.apiPort()
}

func (t *Traefik) version(ctx context.Context) (string, error) {
	var payload struct {
		Version string `json:"Version"`
	}

	if err := t.get(ctx, "/api/version", &payload); err != nil {
		return "", err
	}

	return payload.Version, nil
}

func (t *Traefik) get(ctx context.Context, path string, target any) error {
	requestCtx, cancel := context.WithTimeout(ctx, 3*time.Second)
	defer cancel()

	request, err := http.NewRequestWithContext(requestCtx, http.MethodGet, t.apiURL()+path, nil)
	if err != nil {
		return err
	}

	response, err := http.DefaultClient.Do(request)
	if err != nil {
		return errors.New("proxy: the proxy API is not reachable")
	}
	defer response.Body.Close()

	if response.StatusCode >= 400 {
		return fmt.Errorf("proxy: the proxy API answered with %d", response.StatusCode)
	}

	if err := json.NewDecoder(response.Body).Decode(target); err != nil {
		return errors.New("proxy: the proxy API returned an unexpected payload")
	}

	return nil
}

func (t *Traefik) Routes(ctx context.Context) ([]Route, error) {
	var payload []struct {
		Name        string   `json:"name"`
		Rule        string   `json:"rule"`
		Service     string   `json:"service"`
		Status      string   `json:"status"`
		Entrypoints []string `json:"entryPoints"`
		Provider    string   `json:"provider"`
		Priority    int      `json:"priority"`
		TLS         *struct {
			CertResolver string `json:"certResolver"`
		} `json:"tls"`
	}

	if err := t.get(ctx, "/api/http/routers", &payload); err != nil {
		return nil, err
	}

	routes := make([]Route, 0, len(payload))
	for _, item := range payload {
		routes = append(routes, Route{
			Name:        item.Name,
			Rule:        item.Rule,
			Service:     item.Service,
			Entrypoints: item.Entrypoints,
			Status:      item.Status,
			Provider:    item.Provider,
			Priority:    item.Priority,
			TLS:         item.TLS != nil && item.TLS.CertResolver != "",
		})
	}

	return routes, nil
}

func (t *Traefik) Certificates(_ context.Context) ([]Certificate, error) {
	content, err := os.ReadFile(t.acmePath())
	if err != nil {
		if os.IsNotExist(err) {
			return []Certificate{}, nil
		}

		return nil, err
	}

	if strings.TrimSpace(string(content)) == "" {
		return []Certificate{}, nil
	}

	var parsed map[string]struct {
		Certificates []struct {
			Domain struct {
				Main string   `json:"main"`
				SANs []string `json:"sans"`
			} `json:"domain"`
			Certificate []byte `json:"certificate"`
		} `json:"Certificates"`
	}

	if err := json.Unmarshal(content, &parsed); err != nil {
		return nil, errors.New("proxy: acme.json could not be parsed")
	}

	certificates := []Certificate{}
	for resolver, store := range parsed {
		for _, item := range store.Certificates {
			certificate := Certificate{
				Domain:      item.Domain.Main,
				SANs:        item.Domain.SANs,
				ResolvedVia: resolver,
			}

			if block, _ := pem.Decode(item.Certificate); block != nil {
				if leaf, err := x509.ParseCertificate(block.Bytes); err == nil {
					certificate.NotAfter = leaf.NotAfter
					certificate.Issuer = leaf.Issuer.CommonName
					certificate.DaysLeft = int(time.Until(leaf.NotAfter).Hours() / 24)
				}
			}

			certificates = append(certificates, certificate)
		}
	}

	return certificates, nil
}

func (t *Traefik) Logs(ctx context.Context, tail int) (string, error) {
	containers, err := docker.Default().FindByResource(ctx, docker.ResourceProxy, "")
	if err != nil {
		return "", err
	}
	if len(containers) == 0 {
		return "", errors.New("proxy: the proxy container is not running")
	}

	return docker.Default().Logs(ctx, containers[0].ID, tail)
}

// MiddlewareFile is the YAML shape of a middleware definition.
type MiddlewareFile struct {
	HTTP struct {
		Middlewares map[string]map[string]any `yaml:"middlewares"`
	} `yaml:"http"`
}

type middlewareDocument struct {
	HTTP middlewareHTTP `yaml:"http"`
}

type middlewareHTTP struct {
	Middlewares map[string]middlewareDefinition `yaml:"middlewares"`
}

type middlewareDefinition map[string]any

func (t *Traefik) Middlewares(ctx context.Context) ([]Middleware, error) {
	middlewares := []Middleware{}

	files, err := os.ReadDir(t.dynamicDirectory())
	if err == nil {
		for _, file := range files {
			if file.IsDir() || !strings.HasPrefix(file.Name(), "middleware-") {
				continue
			}

			content, err := os.ReadFile(filepath.Join(t.dynamicDirectory(), file.Name()))
			if err != nil {
				continue
			}

			var document middlewareDocument
			if err := yaml.Unmarshal(content, &document); err != nil {
				continue
			}

			for name, definition := range document.HTTP.Middlewares {
				for kind, config := range definition {
					middlewares = append(middlewares, Middleware{
						Name:     name,
						Type:     kind,
						Config:   normalizeConfig(config),
						Provider: "file",
						Editable: true,
					})
				}
			}
		}
	}

	var remote []struct {
		Name     string         `json:"name"`
		Type     string         `json:"type"`
		Status   string         `json:"status"`
		Provider string         `json:"provider"`
		Config   map[string]any `json:"config"`
	}
	if err := t.get(ctx, "/api/http/middlewares", &remote); err == nil {
		for _, item := range remote {
			if item.Provider == "file" {
				continue
			}

			middlewares = append(middlewares, Middleware{
				Name:     item.Name,
				Type:     item.Type,
				Config:   item.Config,
				Provider: item.Provider,
				Editable: false,
			})
		}
	}

	return middlewares, nil
}

func (t *Traefik) SaveMiddleware(_ context.Context, middleware Middleware) error {
	name := strings.TrimSpace(middleware.Name)
	if name == "" {
		return errors.New("proxy: the middleware needs a name")
	}
	if middleware.Type == "" {
		return errors.New("proxy: the middleware needs a type")
	}

	directory := t.dynamicDirectory()
	if err := os.MkdirAll(directory, 0o755); err != nil {
		return err
	}

	document := middlewareDocument{}
	document.HTTP.Middlewares = map[string]middlewareDefinition{
		name: {middleware.Type: middleware.Config},
	}

	content, err := yaml.Marshal(document)
	if err != nil {
		return err
	}

	return os.WriteFile(filepath.Join(directory, "middleware-"+safeFileName(name)+".yml"), content, 0o600)
}

func (t *Traefik) DeleteMiddleware(_ context.Context, name string) error {
	path := filepath.Join(t.dynamicDirectory(), "middleware-"+safeFileName(name)+".yml")

	if err := os.Remove(path); err != nil && !os.IsNotExist(err) {
		return err
	}

	return nil
}

func normalizeConfig(value any) map[string]any {
	config := map[string]any{}

	switch typed := value.(type) {
	case map[string]any:
		config = typed
	case middlewareDefinition:
		config = map[string]any(typed)
	default:
		return map[string]any{"value": value}
	}

	for key, item := range config {
		if list, ok := item.([]any); ok {
			strings := make([]string, 0, len(list))
			for _, entry := range list {
				strings = append(strings, fmt.Sprint(entry))
			}
			config[key] = strings
		}
	}

	return config
}

func safeFileName(name string) string {
	return strings.Map(func(character rune) rune {
		switch {
		case character >= 'a' && character <= 'z',
			character >= 'A' && character <= 'Z',
			character >= '0' && character <= '9',
			character == '-', character == '_', character == '.':
			return character
		}

		return '-'
	}, name)
}

// FormatBytes renders a byte count for log messages.
func FormatBytes(bytes int64) string {
	units := []string{"B", "KB", "MB", "GB", "TB"}
	value := float64(bytes)
	unit := 0

	for value >= 1024 && unit < len(units)-1 {
		value /= 1024
		unit++
	}

	return strconv.FormatFloat(value, 'f', 1, 64) + " " + units[unit]
}
