package controllers

import (
	"fmt"
	"net/http"
	"strings"

	contractshttp "github.com/goravel/framework/contracts/http"
	"golang.org/x/crypto/bcrypt"

	"ebox/app/services/docker"
	"ebox/app/services/proxy"
	"ebox/app/services/settings"
	"ebox/app/support/httpx"
)

type ProxyController struct{}

func NewProxyController() *ProxyController {
	return &ProxyController{}
}

func (c *ProxyController) Status(ctx contractshttp.Context) contractshttp.Response {
	status := proxy.Default().Status(ctx.Request().Origin().Context())
	config := proxy.TraefikService().Settings()

	return httpx.Ok(ctx, contractshttp.Json{
		"status":   status,
		"settings": config,
	})
}

func (c *ProxyController) Routes(ctx contractshttp.Context) contractshttp.Response {
	routes, err := proxy.Default().Routes(ctx.Request().Origin().Context())
	if err != nil {
		return httpx.Err(ctx, http.StatusBadGateway, err.Error())
	}

	return httpx.Ok(ctx, routes)
}

func (c *ProxyController) Certificates(ctx contractshttp.Context) contractshttp.Response {
	certificates, err := proxy.Default().Certificates(ctx.Request().Origin().Context())
	if err != nil {
		return httpx.Err(ctx, http.StatusInternalServerError, err.Error())
	}

	return httpx.Ok(ctx, certificates)
}

func (c *ProxyController) Middlewares(ctx contractshttp.Context) contractshttp.Response {
	middlewares, err := proxy.Default().Middlewares(ctx.Request().Origin().Context())
	if err != nil {
		return httpx.Err(ctx, http.StatusBadGateway, err.Error())
	}

	return httpx.Ok(ctx, middlewares)
}

func (c *ProxyController) StoreMiddleware(ctx contractshttp.Context) contractshttp.Response {
	middleware, failure := middlewareInput(ctx)
	if failure != nil {
		return failure
	}

	if err := proxy.Default().SaveMiddleware(ctx.Request().Origin().Context(), middleware); err != nil {
		return httpx.Err(ctx, http.StatusUnprocessableEntity, err.Error())
	}

	return httpx.Created(ctx, middleware)
}

func (c *ProxyController) DestroyMiddleware(ctx contractshttp.Context) contractshttp.Response {
	if err := proxy.Default().DeleteMiddleware(ctx.Request().Origin().Context(), ctx.Request().Input("name")); err != nil {
		return httpx.Err(ctx, http.StatusInternalServerError, err.Error())
	}

	return httpx.Ok(ctx, contractshttp.Json{"message": "Middleware deleted."})
}

func (c *ProxyController) Start(ctx contractshttp.Context) contractshttp.Response {
	return c.lifecycle(ctx, func() error { return proxy.Default().Start(ctx.Request().Origin().Context()) }, "Proxy started.")
}

func (c *ProxyController) Stop(ctx contractshttp.Context) contractshttp.Response {
	return c.lifecycle(ctx, func() error { return proxy.Default().Stop(ctx.Request().Origin().Context()) }, "Proxy stopped.")
}

func (c *ProxyController) Restart(ctx contractshttp.Context) contractshttp.Response {
	return c.lifecycle(ctx, func() error { return proxy.Default().Restart(ctx.Request().Origin().Context()) }, "Proxy restarted.")
}

func (c *ProxyController) Ensure(ctx contractshttp.Context) contractshttp.Response {
	return c.lifecycle(ctx, func() error { return proxy.Default().Ensure(ctx.Request().Origin().Context()) }, "Proxy configured and started.")
}

func (c *ProxyController) Logs(ctx contractshttp.Context) contractshttp.Response {
	output, err := proxy.Default().Logs(ctx.Request().Origin().Context(), parseTail(ctx, 200))
	if err != nil {
		return httpx.Err(ctx, http.StatusBadGateway, err.Error())
	}

	lines := []string{}
	if strings.TrimSpace(output) != "" {
		lines = strings.Split(strings.TrimRight(output, "\n"), "\n")
	}

	return httpx.Ok(ctx, contractshttp.Json{"lines": lines})
}

func (c *ProxyController) UpdateSettings(ctx contractshttp.Context) contractshttp.Response {
	update := map[string]string{}

	if domain := strings.TrimSpace(ctx.Request().Input("panel_domain")); domain != "" {
		lowered := strings.ToLower(domain)
		if strings.Contains(lowered, "://") || strings.ContainsAny(lowered, " /") {
			return httpx.ValidationErrors(ctx, validationErrors{"panel_domain": {"Enter a bare hostname, for example panel.example.com."}})
		}
		update["panel_domain"] = lowered
	} else {
		update["panel_domain"] = ""
	}

	email := strings.TrimSpace(ctx.Request().Input("acme_email"))
	if email != "" && !validEmail(email) {
		return httpx.ValidationErrors(ctx, validationErrors{"acme_email": {"The ACME email field must be a valid email address."}})
	}
	update["acme_email"] = email

	if err := settings.Default().SetMany(update); err != nil {
		return httpx.Err(ctx, http.StatusInternalServerError, "Failed to save the proxy settings.")
	}

	status := proxy.Default().Status(ctx.Request().Origin().Context())
	if status.Installed {
		if err := proxy.Default().Restart(ctx.Request().Origin().Context()); err != nil {
			return httpx.Err(ctx, http.StatusBadGateway, docker.FriendlyError(err))
		}
	}

	return httpx.Ok(ctx, contractshttp.Json{"message": "Proxy settings saved."})
}

func (c *ProxyController) lifecycle(ctx contractshttp.Context, action func() error, message string) contractshttp.Response {
	if err := action(); err != nil {
		return httpx.Err(ctx, http.StatusBadGateway, docker.FriendlyError(err))
	}

	return httpx.Ok(ctx, contractshttp.Json{"message": message})
}

func middlewareInput(ctx contractshttp.Context) (proxy.Middleware, contractshttp.Response) {
	middleware := proxy.Middleware{
		Name:   strings.TrimSpace(ctx.Request().Input("name")),
		Type:   ctx.Request().Input("type"),
		Config: map[string]any{},
	}

	if middleware.Name == "" || middleware.Type == "" {
		return middleware, httpx.ValidationErrors(ctx, validationErrors{
			"name": {"A name and a type are required."},
		})
	}

	config := ctx.Request().InputMap("config")

	switch middleware.Type {
	case "basicAuth":
		users := []string{}
		if list, ok := config["users"].([]any); ok {
			for _, entry := range list {
				item, ok := entry.(map[string]any)
				if !ok {
					continue
				}

				username, _ := item["username"].(string)
				password, _ := item["password"].(string)
				if username == "" || password == "" {
					continue
				}

				hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
				if err != nil {
					return middleware, httpx.Err(ctx, http.StatusInternalServerError, "Could not hash the password.")
				}

				users = append(users, username+":"+string(hash))
			}
		}

		if len(users) == 0 {
			return middleware, httpx.ValidationErrors(ctx, validationErrors{
				"config.users": {"Add at least one user with a password."},
			})
		}

		middleware.Config = map[string]any{"users": users}
	case "headers":
		headers := map[string]any{}
		if list, ok := config["customRequestHeaders"].(map[string]any); ok {
			for key, value := range list {
				headers[key] = fmt.Sprint(value)
			}
		}
		if len(headers) == 0 {
			return middleware, httpx.ValidationErrors(ctx, validationErrors{
				"config.customRequestHeaders": {"Add at least one header."},
			})
		}
		middleware.Config = map[string]any{"customRequestHeaders": headers}
	case "redirectScheme":
		scheme := "https"
		if value, ok := config["scheme"].(string); ok && value != "" {
			scheme = value
		}
		permanent := true
		if value, ok := config["permanent"].(bool); ok {
			permanent = value
		}
		middleware.Config = map[string]any{"scheme": scheme, "permanent": permanent}
	case "rateLimit":
		average := 100
		if value, ok := config["average"].(float64); ok && value > 0 {
			average = int(value)
		}
		burst := average * 2
		if value, ok := config["burst"].(float64); ok && value > 0 {
			burst = int(value)
		}
		middleware.Config = map[string]any{"average": average, "burst": burst}
	case "stripPrefix":
		prefixes := []string{}
		if value, ok := config["prefixes"].(string); ok && strings.TrimSpace(value) != "" {
			for _, prefix := range strings.Split(value, ",") {
				trimmed := strings.TrimSpace(prefix)
				if trimmed != "" {
					prefixes = append(prefixes, trimmed)
				}
			}
		}
		if len(prefixes) == 0 {
			return middleware, httpx.ValidationErrors(ctx, validationErrors{
				"config.prefixes": {"Add at least one prefix."},
			})
		}
		middleware.Config = map[string]any{"prefixes": prefixes}
	default:
		return middleware, httpx.ValidationErrors(ctx, validationErrors{
			"type": {"Unsupported middleware type."},
		})
	}

	return middleware, nil
}
