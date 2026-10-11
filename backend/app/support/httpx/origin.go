package httpx

import (
	"net/url"
	"strings"

	contractshttp "github.com/goravel/framework/contracts/http"

	"ebox/app/models"
)

const ContextAccessToken = "ebox:access_token"

func AccessToken(ctx contractshttp.Context) *models.PersonalAccessToken {
	token, ok := ctx.Value(ContextAccessToken).(*models.PersonalAccessToken)
	if !ok {
		return nil
	}

	return token
}

// ExternalOrigin returns the origin the client used to reach the panel,
// honouring reverse proxy headers.
func ExternalOrigin(ctx contractshttp.Context) string {
	origin := ctx.Request().Origin()

	proto := ctx.Request().Header("X-Forwarded-Proto")
	if proto == "" {
		if origin.TLS != nil {
			proto = "https"
		} else {
			proto = "http"
		}
	}

	host := ctx.Request().Header("X-Forwarded-Host")
	if host == "" {
		host = origin.Host
	}
	if host == "" {
		return ""
	}

	return proto + "://" + host
}

// ValidOrigin reports whether value is a plain http(s) origin.
func ValidOrigin(value string) bool {
	parsed, err := url.Parse(value)
	if err != nil {
		return false
	}

	if parsed.Scheme != "http" && parsed.Scheme != "https" {
		return false
	}
	if parsed.Host == "" {
		return false
	}
	if parsed.Path != "" && parsed.Path != "/" {
		return false
	}

	return parsed.RawQuery == "" && parsed.Fragment == ""
}

func NormalizeOrigin(value string) string {
	return strings.TrimRight(strings.TrimSpace(value), "/")
}
