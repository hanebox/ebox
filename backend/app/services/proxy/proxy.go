package proxy

import (
	"context"
	"time"
)

type Status struct {
	Installed    bool     `json:"installed"`
	Running      bool     `json:"running"`
	State        string   `json:"state"`
	ContainerID  string   `json:"container_id,omitempty"`
	Container    string   `json:"container,omitempty"`
	Image        string   `json:"image,omitempty"`
	Version      string   `json:"version,omitempty"`
	Ports        []string `json:"ports,omitempty"`
	APIReachable bool     `json:"api_reachable"`
	Error        string   `json:"error,omitempty"`
}

type Route struct {
	Name        string   `json:"name"`
	Rule        string   `json:"rule"`
	Service     string   `json:"service"`
	Entrypoints []string `json:"entrypoints"`
	Status      string   `json:"status"`
	TLS         bool     `json:"tls"`
	Provider    string   `json:"provider"`
	Priority    int      `json:"priority,omitempty"`
}

type Certificate struct {
	Domain      string    `json:"domain"`
	SANs        []string  `json:"sans"`
	NotAfter    time.Time `json:"not_after"`
	DaysLeft    int       `json:"days_left"`
	Issuer      string    `json:"issuer"`
	ResolvedVia string    `json:"resolved_via"`
}

type Middleware struct {
	Name        string         `json:"name"`
	Type        string         `json:"type"`
	Config      map[string]any `json:"config"`
	Description string         `json:"description,omitempty"`
	Provider    string         `json:"provider"`
	Editable    bool           `json:"editable"`
}

type Settings struct {
	PanelDomain string `json:"panel_domain"`
	ACMEEmail   string `json:"acme_email"`
	HTTPPort    int    `json:"http_port"`
	HTTPSPort   int    `json:"https_port"`
}

// Proxy is the ingress implementation the panel manages. Traefik is the only
// implementation today; Caddy can be added behind this interface later.
type Proxy interface {
	Ensure(ctx context.Context) error
	Status(ctx context.Context) Status
	Start(ctx context.Context) error
	Stop(ctx context.Context) error
	Restart(ctx context.Context) error
	Routes(ctx context.Context) ([]Route, error)
	Certificates(ctx context.Context) ([]Certificate, error)
	Middlewares(ctx context.Context) ([]Middleware, error)
	SaveMiddleware(ctx context.Context, middleware Middleware) error
	DeleteMiddleware(ctx context.Context, name string) error
	Logs(ctx context.Context, tail int) (string, error)
}

var defaultProxy = NewTraefik()

// Default returns the active ingress implementation.
func Default() Proxy {
	return defaultProxy
}

// Traefik returns the Traefik implementation directly.
func TraefikService() *Traefik {
	return defaultProxy
}
