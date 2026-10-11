package webserve

import (
	"io/fs"
	"net/http"
	"path"
	"strings"

	contractshttp "github.com/goravel/framework/contracts/http"
	"github.com/goravel/framework/contracts/route"

	"ebox/web"
)

func Register(router route.Route) {
	dist, err := web.Dist()
	if err != nil {
		router.Fallback(func(ctx contractshttp.Context) contractshttp.Response {
			if isAPIPath(ctx) {
				return notFound(ctx)
			}

			return ctx.Response().String(http.StatusOK, "ebox API is running. The web UI is not embedded in this build.\n")
		})
		return
	}

	fileServer := http.FileServer(http.FS(dist))
	index, indexErr := fs.ReadFile(dist, "index.html")

	router.Fallback(func(ctx contractshttp.Context) contractshttp.Response {
		if isAPIPath(ctx) {
			return notFound(ctx)
		}

		writer := ctx.Response().Writer()
		request := ctx.Request().Origin()
		assetPath := strings.TrimPrefix(path.Clean("/"+request.URL.Path), "/")

		if assetPath != "" {
			if file, err := dist.Open(assetPath); err == nil {
				_ = file.Close()
				if strings.HasPrefix(request.URL.Path, "/assets/") {
					writer.Header().Set("Cache-Control", "public, max-age=31536000, immutable")
				}
				fileServer.ServeHTTP(writer, request)
				return nil
			}

			// Missing files under /assets/ are real 404s, never SPA routes.
			if strings.HasPrefix(request.URL.Path, "/assets/") {
				return ctx.Response().Status(http.StatusNotFound).String("asset not found")
			}
		}

		if indexErr != nil {
			return ctx.Response().Status(http.StatusNotFound).String("index.html is missing from the embedded frontend")
		}

		// gin pre-sets the 404 status on the NoRoute path, so the SPA document
		// must explicitly reset it to 200 before writing the body.
		writer.Header().Set("Content-Type", "text/html; charset=utf-8")
		writer.Header().Set("Cache-Control", "no-cache")
		writer.WriteHeader(http.StatusOK)
		_, _ = writer.Write(index)

		return nil
	})
}

func notFound(ctx contractshttp.Context) contractshttp.Response {
	return ctx.Response().Status(http.StatusNotFound).Json(contractshttp.Json{"message": "Not found."})
}

func isAPIPath(ctx contractshttp.Context) bool {
	return strings.HasPrefix(ctx.Request().Path(), "/api/")
}
