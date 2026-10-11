package proxy

import (
	"sort"
	"strconv"

	"ebox/app/models"
	"ebox/app/services/docker"
)

const (
	EntrypointHTTP  = "web"
	EntrypointHTTPS = "websecure"
	CertResolver    = "letsencrypt"
	DefaultNetwork  = "ebox"
)

// RouterName is the stable Traefik service/router prefix for an application.
func RouterName(app *models.Application) string {
	return "ebox-" + app.Slug()
}

// AppLabels builds the Traefik routing labels for an application container.
func AppLabels(app *models.Application, domains []models.Domain, network string) map[string]string {
	return IngressLabels(Ingress{
		ResourceType: docker.ResourceApplication,
		ResourceID:   app.UUID,
		Name:         app.Name,
		Router:       RouterName(app),
		Port:         app.ExposePort(),
		Domains:      domains,
		Network:      network,
	})
}

// Ingress describes the routing labels for one container.
type Ingress struct {
	ResourceType string
	ResourceID   string
	Name         string
	Router       string
	Port         int
	Domains      []models.Domain
	Network      string
}

// IngressLabels builds the Traefik routing labels for a managed container.
// Domains are sorted so the generated labels are deterministic.
func IngressLabels(ingress Ingress) map[string]string {
	network := ingress.Network
	if network == "" {
		network = DefaultNetwork
	}

	labels := docker.ManagedLabels(ingress.ResourceType, ingress.ResourceID, ingress.Name)

	hosts := make([]models.Domain, 0, len(ingress.Domains))
	for _, domain := range ingress.Domains {
		if domain.Host != "" {
			hosts = append(hosts, domain)
		}
	}

	if len(hosts) == 0 {
		labels["traefik.enable"] = "false"

		return labels
	}

	sort.Slice(hosts, func(i, j int) bool { return hosts[i].Host < hosts[j].Host })

	router := ingress.Router
	port := strconv.Itoa(ingress.Port)
	if ingress.Port <= 0 {
		port = "80"
	}

	labels["traefik.enable"] = "true"
	labels["traefik.docker.network"] = network
	labels["traefik.http.services."+router+".loadbalancer.server.port"] = port
	labels["traefik.http.services."+router+".loadbalancer.server.scheme"] = "http"

	redirect := router + "-redirect"
	needsRedirect := false

	for index, domain := range hosts {
		rule := "Host(`" + domain.Host + "`)"
		httpRouter := router + "-" + strconv.Itoa(index)
		secureRouter := httpRouter + "-secure"

		labels["traefik.http.routers."+httpRouter+".rule"] = rule
		labels["traefik.http.routers."+httpRouter+".entrypoints"] = EntrypointHTTP
		labels["traefik.http.routers."+httpRouter+".service"] = router

		if !domain.HTTPS {
			continue
		}

		needsRedirect = true
		labels["traefik.http.routers."+httpRouter+".middlewares"] = redirect

		labels["traefik.http.routers."+secureRouter+".rule"] = rule
		labels["traefik.http.routers."+secureRouter+".entrypoints"] = EntrypointHTTPS
		labels["traefik.http.routers."+secureRouter+".service"] = router
		labels["traefik.http.routers."+secureRouter+".tls.certresolver"] = CertResolver
	}

	if needsRedirect {
		labels["traefik.http.middlewares."+redirect+".redirectscheme.scheme"] = "https"
		labels["traefik.http.middlewares."+redirect+".redirectscheme.permanent"] = "true"
	}

	return labels
}
