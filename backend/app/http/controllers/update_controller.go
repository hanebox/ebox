package controllers

import (
	"net/http"
	"os"
	"time"

	contractshttp "github.com/goravel/framework/contracts/http"

	"ebox/app/facades"
	"ebox/app/models"
	"ebox/app/services/notifications"
	"ebox/app/services/system"
	"ebox/app/support/httpx"
)

type UpdateController struct{}

func NewUpdateController() *UpdateController {
	return &UpdateController{}
}

// Status reports the running version and the newest release.
func (c *UpdateController) Status(ctx contractshttp.Context) contractshttp.Response {
	check := ctx.Request().InputBool("check", false)

	return httpx.Ok(ctx, system.Default().Status(ctx.Request().Origin().Context(), check))
}

// Apply downloads and installs the newest release, then restarts the panel.
func (c *UpdateController) Apply(ctx contractshttp.Context) contractshttp.Response {
	updater := system.Default()
	status := updater.Status(ctx.Request().Origin().Context(), true)
	if status.CheckError != "" {
		return httpx.Err(ctx, http.StatusBadGateway, status.CheckError)
	}
	if status.Latest == nil {
		return httpx.Err(ctx, http.StatusNotFound, "No release was found for this platform.")
	}
	if !status.UpdateAvailable && !ctx.Request().InputBool("force", false) {
		return httpx.Err(ctx, http.StatusUnprocessableEntity, "The panel already runs the newest version.")
	}

	release := status.Latest

	// The download happens inline so the caller sees failures. The restart is
	// scheduled a moment later, long enough for this response to reach the client.
	err := updater.Apply(ctx.Request().Origin().Context(), release, nil)
	if err != nil {
		notifications.Default().Notify(models.EventUpdateAvailable,
			"Panel update failed",
			err.Error(),
			map[string]any{"version": release.Version, "error": err.Error()})

		return httpx.Err(ctx, http.StatusBadGateway, err.Error())
	}

	notifications.Default().Notify(models.EventUpdateAvailable,
		"Panel updated to "+release.Version,
		"The panel is restarting with the new version.",
		map[string]any{"version": release.Version, "from": status.CurrentVersion})

	go restartLater()

	return httpx.Ok(ctx, contractshttp.Json{
		"message": "Update installed. The panel is restarting.",
		"version": release.Version,
	})
}

// Rollback restores the previous binary.
func (c *UpdateController) Rollback(ctx contractshttp.Context) contractshttp.Response {
	if err := system.Default().Rollback(); err != nil {
		return httpx.Err(ctx, http.StatusBadGateway, err.Error())
	}

	go restartLater()

	return httpx.Ok(ctx, contractshttp.Json{"message": "Previous version restored. The panel is restarting."})
}

// restartLater gives the HTTP response time to leave, then restarts the unit.
// When systemd cannot be asked directly the process exits, which the unit's
// Restart policy turns into a fresh start of the new binary.
func restartLater() {
	time.Sleep(1500 * time.Millisecond)

	unit := facades.Config().GetString("ebox.systemd_unit", "ebox.service")

	if err := system.Default().Restart(unit); err != nil {
		if log := facades.Log(); log != nil {
			log.Errorf("ebox: the panel could not restart itself: %s", err)
		}

		os.Exit(0)
	}
}
