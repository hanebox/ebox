package controllers

import (
	"github.com/goravel/framework/contracts/http"

	"ebox/app/services/metrics"
	"ebox/app/support/httpx"
)

type MetricsController struct{}

func NewMetricsController() *MetricsController {
	return &MetricsController{}
}

func (c *MetricsController) Host(ctx http.Context) http.Response {
	window := ctx.Request().Query("window", "20m")
	if window != "24h" {
		window = "20m"
	}

	collector := metrics.Default()
	current, _ := collector.Current()

	return httpx.Ok(ctx, http.Json{
		"current": current,
		"history": collector.History(window),
	})
}
