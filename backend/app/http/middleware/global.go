package middleware

import (
	"strings"

	"github.com/goravel/framework/contracts/http"

	"ebox/app/facades"
)

// ConfigureGlobal removes the framework's default request timeout middleware.
// That middleware buffers every response in memory and never flushes it until
// the handler returns, which breaks streaming responses (SSE) and hijacked
// connections (WebSockets). ebox serves both, so the middleware is dropped
// before any route is registered and the HTTP server timeouts are relied on.
func ConfigureGlobal() {
	global := facades.Route().GetGlobalMiddleware()

	filtered := make([]http.Middleware, 0, len(global))
	for _, item := range global {
		if strings.HasPrefix(item.Signature(), "goravel:timeout:") {
			continue
		}

		filtered = append(filtered, item)
	}

	facades.Route().SetGlobalMiddleware(filtered)
}
