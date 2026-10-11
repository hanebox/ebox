package routes

import (
	contractshttp "github.com/goravel/framework/contracts/http"
	contractsroute "github.com/goravel/framework/contracts/route"

	"ebox/app/facades"
	"ebox/app/http/controllers"
	"ebox/app/http/middleware"
	"ebox/app/services/docker"
)

// resourceHandler adapts a resource scoped controller method to a route handler.
func resourceHandler(
	action func(contractshttp.Context, string) contractshttp.Response,
	resourceType string,
) contractshttp.HandlerFunc {
	return func(ctx contractshttp.Context) contractshttp.Response {
		return action(ctx, resourceType)
	}
}

func Api() {
	route := facades.Route()

	setupController := controllers.NewSetupController()
	authController := controllers.NewAuthController()
	sessionController := controllers.NewSessionController()
	accountController := controllers.NewAccountController()
	settingsController := controllers.NewSettingsController()
	metricsController := controllers.NewMetricsController()
	systemController := controllers.NewSystemController()
	eventController := controllers.NewEventController()
	projectController := controllers.NewProjectController()
	sourceController := controllers.NewSourceController()
	githubController := controllers.NewGithubController()
	tokenController := controllers.NewTokenController()
	applicationController := controllers.NewApplicationController()
	deploymentController := controllers.NewDeploymentController()
	envVarController := controllers.NewEnvVarController()
	domainController := controllers.NewDomainController()
	webhookController := controllers.NewWebhookController()
	deployHookController := controllers.NewDeployHookController()
	terminalController := controllers.NewTerminalController()
	containerController := controllers.NewContainerController()
	logController := controllers.NewLogController()
	proxyController := controllers.NewProxyController()
	databaseController := controllers.NewDatabaseController()
	stackController := controllers.NewStackController()
	notificationController := controllers.NewNotificationController()
	backupController := controllers.NewBackupController()
	updateController := controllers.NewUpdateController()

	route.Get("/api/v1/health", systemController.Health)
	route.Get("/health", systemController.Health)

	route.Prefix("api/v1").Group(func(router contractsroute.Router) {
		router.Get("/setup/status", setupController.Status)
		router.Post("/setup", setupController.Store)
		router.Post("/auth/login", authController.Login)

		router.Post("/webhooks/github", webhookController.Github)
		router.Post("/deploy/{token}", deployHookController.Deploy)

		router.Middleware(&middleware.Authenticate{}).Group(func(router contractsroute.Router) {
			router.Get("/auth/me", authController.Me)
			router.Post("/auth/logout", authController.Logout)

			router.Get("/events", eventController.Stream)
			router.Get("/metrics/host", metricsController.Host)
			router.Get("/system/info", systemController.Info)
			router.Get("/system/disk", systemController.DiskUsage)
			router.Get("/terminal/targets", terminalController.Targets)
			router.Get("/terminal/ws", terminalController.Connect)
			router.Get("/containers", containerController.Index)
			router.Get("/proxy", proxyController.Status)
			router.Get("/proxy/routes", proxyController.Routes)
			router.Get("/proxy/certificates", proxyController.Certificates)
			router.Get("/proxy/middlewares", proxyController.Middlewares)
			router.Get("/proxy/logs", proxyController.Logs)
			router.Get("/sessions", sessionController.Index)

			router.Get("/projects", projectController.Index)
			router.Get("/projects/{uuid}", projectController.Show)

			router.Get("/sources", sourceController.Index)
			router.Get("/sources/{uuid}", sourceController.Show)
			router.Get("/sources/{uuid}/repositories", sourceController.Repositories)
			router.Get("/sources/{uuid}/branches", sourceController.Branches)
			router.Get("/sources/{uuid}/github/installations", sourceController.Installations)

			router.Get("/tokens", tokenController.Index)

			router.Get("/applications", applicationController.Index)
			router.Get("/applications/{uuid}", applicationController.Show)
			router.Get("/applications/{uuid}/logs", applicationController.Logs)
			router.Get("/applications/{uuid}/logs/stream", func(ctx contractshttp.Context) contractshttp.Response {
				return logController.Stream(ctx, docker.ResourceApplication)
			})
			router.Get("/applications/{uuid}/logs/download", func(ctx contractshttp.Context) contractshttp.Response {
				return logController.Download(ctx, docker.ResourceApplication)
			})
			router.Get("/applications/{uuid}/compose", applicationController.ComposePreview)
			router.Get("/applications/{uuid}/stats", containerController.Stats)
			router.Get("/applications/{uuid}/env", resourceHandler(envVarController.Index, docker.ResourceApplication))
			router.Get("/applications/{uuid}/env/export", resourceHandler(envVarController.Export, docker.ResourceApplication))
			router.Get("/applications/{uuid}/domains", resourceHandler(domainController.Index, docker.ResourceApplication))

			router.Get("/databases", databaseController.Index)
			router.Get("/databases/definitions", databaseController.Definitions)
			router.Get("/databases/{uuid}", databaseController.Show)
			router.Get("/databases/{uuid}/credentials", databaseController.Credentials)
			router.Get("/databases/{uuid}/logs", databaseController.Logs)
			router.Get("/databases/{uuid}/logs/stream", func(ctx contractshttp.Context) contractshttp.Response {
				return logController.Stream(ctx, docker.ResourceDatabase)
			})
			router.Get("/databases/{uuid}/logs/download", func(ctx contractshttp.Context) contractshttp.Response {
				return logController.Download(ctx, docker.ResourceDatabase)
			})

			router.Get("/services", stackController.Index)
			router.Get("/services/{uuid}", stackController.Show)
			router.Get("/services/{uuid}/containers", stackController.Containers)
			router.Get("/services/{uuid}/logs/stream", func(ctx contractshttp.Context) contractshttp.Response {
				return logController.Stream(ctx, docker.ResourceService)
			})
			router.Get("/services/{uuid}/logs/download", func(ctx contractshttp.Context) contractshttp.Response {
				return logController.Download(ctx, docker.ResourceService)
			})
			router.Get("/services/{uuid}/env", resourceHandler(envVarController.Index, docker.ResourceService))
			router.Get("/services/{uuid}/env/export", resourceHandler(envVarController.Export, docker.ResourceService))
			router.Get("/services/{uuid}/domains", resourceHandler(domainController.Index, docker.ResourceService))

			router.Get("/notifications/channels", notificationController.Index)
			router.Get("/notifications/events", notificationController.Events)
			router.Get("/notifications/history", notificationController.History)

			router.Get("/backups", backupController.Index)
			router.Get("/backups/targets", backupController.Targets)
			router.Get("/backups/{uuid}", backupController.Show)
			router.Get("/backups/{uuid}/runs/{run}/download", backupController.Download)

			router.Get("/system/update", updateController.Status)

			router.Get("/deployments", deploymentController.Index)
			router.Get("/deployments/{uuid}", deploymentController.Show)
			router.Get("/deployments/{uuid}/logs", deploymentController.Logs)

			router.Get("/github/callback", githubController.Callback)
			router.Get("/github/setup/{source}", githubController.Setup)

			router.Middleware(&middleware.Csrf{}).Group(func(router contractsroute.Router) {
				router.Delete("/sessions/{id}", sessionController.Destroy)
				router.Put("/account", accountController.Update)
				router.Put("/account/password", accountController.UpdatePassword)
				router.Get("/settings", settingsController.Index)
				router.Put("/settings", settingsController.Update)

				router.Post("/projects", projectController.Store)
				router.Put("/projects/{uuid}", projectController.Update)
				router.Delete("/projects/{uuid}", projectController.Destroy)

				router.Post("/sources", sourceController.Store)
				router.Put("/sources/{uuid}", sourceController.Update)
				router.Put("/sources/{uuid}/installation", sourceController.SetInstallation)
				router.Delete("/sources/{uuid}", sourceController.Destroy)
				router.Post("/sources/{uuid}/test", sourceController.Test)

				router.Post("/tokens", tokenController.Store)
				router.Delete("/tokens/{id}", tokenController.Destroy)

				router.Post("/applications", applicationController.Store)
				router.Put("/applications/{uuid}", applicationController.Update)
				router.Delete("/applications/{uuid}", applicationController.Destroy)
				router.Post("/applications/{uuid}/deploy", applicationController.Deploy)
				router.Post("/applications/{uuid}/rollback", applicationController.Rollback)
				router.Post("/applications/{uuid}/start", applicationController.Start)
				router.Post("/applications/{uuid}/stop", applicationController.Stop)
				router.Post("/applications/{uuid}/restart", applicationController.Restart)
				router.Post("/applications/{uuid}/token/regenerate", applicationController.RegenerateToken)

				router.Post("/applications/{uuid}/env", resourceHandler(envVarController.Store, docker.ResourceApplication))
				router.Post("/applications/{uuid}/env/import", resourceHandler(envVarController.Import, docker.ResourceApplication))
				router.Put("/applications/{uuid}/env/{id}", resourceHandler(envVarController.Update, docker.ResourceApplication))
				router.Delete("/applications/{uuid}/env/{id}", resourceHandler(envVarController.Destroy, docker.ResourceApplication))

				router.Post("/applications/{uuid}/domains", resourceHandler(domainController.Store, docker.ResourceApplication))
				router.Put("/applications/{uuid}/domains/{id}", resourceHandler(domainController.Update, docker.ResourceApplication))
				router.Delete("/applications/{uuid}/domains/{id}", resourceHandler(domainController.Destroy, docker.ResourceApplication))

				router.Post("/databases", databaseController.Store)
				router.Put("/databases/{uuid}", databaseController.Update)
				router.Delete("/databases/{uuid}", databaseController.Destroy)
				router.Post("/databases/{uuid}/start", databaseController.Start)
				router.Post("/databases/{uuid}/stop", databaseController.Stop)
				router.Post("/databases/{uuid}/restart", databaseController.Restart)
				router.Post("/databases/{uuid}/attach", databaseController.Attach)

				router.Post("/services", stackController.Store)
				router.Put("/services/{uuid}", stackController.Update)
				router.Delete("/services/{uuid}", stackController.Destroy)
				router.Post("/services/{uuid}/deploy", stackController.Deploy)
				router.Post("/services/{uuid}/start", stackController.Start)
				router.Post("/services/{uuid}/stop", stackController.Stop)
				router.Post("/services/{uuid}/restart", stackController.Restart)
				router.Post("/services/{uuid}/env", resourceHandler(envVarController.Store, docker.ResourceService))
				router.Post("/services/{uuid}/env/import", resourceHandler(envVarController.Import, docker.ResourceService))
				router.Put("/services/{uuid}/env/{id}", resourceHandler(envVarController.Update, docker.ResourceService))
				router.Delete("/services/{uuid}/env/{id}", resourceHandler(envVarController.Destroy, docker.ResourceService))
				router.Post("/services/{uuid}/domains", resourceHandler(domainController.Store, docker.ResourceService))
				router.Put("/services/{uuid}/domains/{id}", resourceHandler(domainController.Update, docker.ResourceService))
				router.Delete("/services/{uuid}/domains/{id}", resourceHandler(domainController.Destroy, docker.ResourceService))

				router.Post("/notifications/channels", notificationController.Store)
				router.Put("/notifications/channels/{uuid}", notificationController.Update)
				router.Delete("/notifications/channels/{uuid}", notificationController.Destroy)
				router.Post("/notifications/channels/{uuid}/toggle", notificationController.Toggle)
				router.Post("/notifications/channels/{uuid}/test", notificationController.Test)
				router.Post("/notifications/send", notificationController.Send)

				router.Post("/backups", backupController.Store)
				router.Put("/backups/{uuid}", backupController.Update)
				router.Delete("/backups/{uuid}", backupController.Destroy)
				router.Post("/backups/{uuid}/run", backupController.Run)
				router.Post("/backups/{uuid}/runs/{run}/restore", backupController.Restore)
				router.Delete("/backups/{uuid}/runs/{run}", backupController.DeleteRun)

				router.Post("/system/update", updateController.Apply)
				router.Post("/system/update/rollback", updateController.Rollback)

				router.Post("/deployments/{uuid}/cancel", deploymentController.Cancel)

				router.Post("/system/prune", systemController.Prune)
				router.Post("/proxy/ensure", proxyController.Ensure)
				router.Post("/proxy/start", proxyController.Start)
				router.Post("/proxy/stop", proxyController.Stop)
				router.Post("/proxy/restart", proxyController.Restart)
				router.Put("/proxy/settings", proxyController.UpdateSettings)
				router.Post("/proxy/middlewares", proxyController.StoreMiddleware)
				router.Delete("/proxy/middlewares/{name}", proxyController.DestroyMiddleware)

				router.Post("/github/manifest", githubController.Manifest)
			})
		})
	})
}
