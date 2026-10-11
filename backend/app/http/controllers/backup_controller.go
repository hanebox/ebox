package controllers

import (
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	contractshttp "github.com/goravel/framework/contracts/http"

	"ebox/app/facades"
	"ebox/app/models"
	"ebox/app/services/backups"
	"ebox/app/services/docker"
	"ebox/app/support/httpx"
)

type BackupController struct{}

func NewBackupController() *BackupController {
	return &BackupController{}
}

type backupResponse struct {
	models.Backup
	NextRunAt    string `json:"next_run_at,omitempty"`
	ScheduleHint string `json:"schedule_hint,omitempty"`
}

func (c *BackupController) Index(ctx contractshttp.Context) contractshttp.Response {
	list, err := backups.Default().List(ctx.Request().Input("project"))
	if err != nil {
		return httpx.Err(ctx, http.StatusInternalServerError, "Failed to load backups.")
	}

	responses := make([]backupResponse, 0, len(list))
	for _, backup := range list {
		responses = append(responses, describe(backup))
	}

	return httpx.Ok(ctx, responses)
}

func (c *BackupController) Show(ctx contractshttp.Context) contractshttp.Response {
	backup, err := backups.Default().Find(ctx.Request().Input("uuid"))
	if err != nil {
		return httpx.Err(ctx, http.StatusNotFound, "Backup not found.")
	}

	runs, err := backups.Default().Runs(backup, 25)
	if err != nil {
		return httpx.Err(ctx, http.StatusInternalServerError, "Failed to load backup runs.")
	}

	return httpx.Ok(ctx, struct {
		backupResponse
		Runs []models.BackupRun `json:"runs"`
	}{
		backupResponse: describe(*backup),
		Runs:           runs,
	})
}

// Targets lists what can be backed up: databases and docker volumes.
func (c *BackupController) Targets(ctx contractshttp.Context) contractshttp.Response {
	requestCtx := ctx.Request().Origin().Context()

	var databases []models.Database
	if err := facades.Orm().Query().Order("name asc").Find(&databases); err != nil {
		return httpx.Err(ctx, http.StatusInternalServerError, "Failed to load databases.")
	}

	type databaseTarget struct {
		UUID      string `json:"uuid"`
		Name      string `json:"name"`
		Type      string `json:"type"`
		ProjectID uint64 `json:"project_id"`
		Port      int    `json:"port"`
	}

	targets := make([]databaseTarget, 0, len(databases))
	for _, database := range databases {
		targets = append(targets, databaseTarget{
			UUID:      database.UUID,
			Name:      database.Name,
			Type:      database.Type,
			ProjectID: database.ProjectID,
			Port:      database.Port,
		})
	}

	volumes, err := docker.Default().ListVolumes(requestCtx, true)
	if err != nil {
		volumes = nil
	}

	return httpx.Ok(ctx, contractshttp.Json{
		"databases": targets,
		"volumes":   volumes,
	})
}

func (c *BackupController) Store(ctx contractshttp.Context) contractshttp.Response {
	input, failure := backupInput(ctx)
	if failure != nil {
		return failure
	}

	backup, err := backups.Default().Create(input)
	if err != nil {
		return httpx.Err(ctx, http.StatusUnprocessableEntity, err.Error())
	}

	return httpx.Created(ctx, describe(*backup))
}

func (c *BackupController) Update(ctx contractshttp.Context) contractshttp.Response {
	backup, err := backups.Default().Find(ctx.Request().Input("uuid"))
	if err != nil {
		return httpx.Err(ctx, http.StatusNotFound, "Backup not found.")
	}

	input, failure := backupInput(ctx)
	if failure != nil {
		return failure
	}

	if err := backups.Default().Update(backup, input); err != nil {
		return httpx.Err(ctx, http.StatusUnprocessableEntity, err.Error())
	}

	return httpx.Ok(ctx, describe(*backup))
}

func (c *BackupController) Destroy(ctx contractshttp.Context) contractshttp.Response {
	backup, err := backups.Default().Find(ctx.Request().Input("uuid"))
	if err != nil {
		return httpx.Err(ctx, http.StatusNotFound, "Backup not found.")
	}

	if err := backups.Default().Delete(backup); err != nil {
		return httpx.Err(ctx, http.StatusInternalServerError, "Failed to delete the backup.")
	}

	return httpx.Ok(ctx, contractshttp.Json{"message": "Backup deleted."})
}

// Run starts a backup right away.
func (c *BackupController) Run(ctx contractshttp.Context) contractshttp.Response {
	backup, err := backups.Default().Find(ctx.Request().Input("uuid"))
	if err != nil {
		return httpx.Err(ctx, http.StatusNotFound, "Backup not found.")
	}

	run, err := backups.Default().Run(ctx.Request().Origin().Context(), backup, models.DeploymentTriggerManual)
	if err != nil {
		return httpx.Err(ctx, http.StatusBadGateway, err.Error())
	}

	if run.Status == models.BackupStatusFailed {
		return httpx.Err(ctx, http.StatusBadGateway, run.Error)
	}

	return httpx.Ok(ctx, run)
}

// Download streams a backup artefact.
func (c *BackupController) Download(ctx contractshttp.Context) contractshttp.Response {
	_, run, failure := c.target(ctx)
	if failure != nil {
		return failure
	}

	if run.File == "" {
		return httpx.Err(ctx, http.StatusNotFound, "This run has no file on disk anymore.")
	}

	file, err := os.Open(run.File)
	if err != nil {
		return httpx.Err(ctx, http.StatusNotFound, "The backup file is no longer available.")
	}
	defer func() { _ = file.Close() }()

	name := filepath.Base(run.File)

	ctx.Response().Header("Content-Disposition", "attachment; filename=\""+name+"\"")
	ctx.Response().Header("Content-Type", "application/gzip")

	return ctx.Response().Stream(http.StatusOK, func(writer contractshttp.StreamWriter) error {
		_, err := io.Copy(writer, file)

		return err
	})
}

// Restore loads a backup back into its database or volume.
func (c *BackupController) Restore(ctx contractshttp.Context) contractshttp.Response {
	backup, run, failure := c.target(ctx)
	if failure != nil {
		return failure
	}

	if err := backups.Default().Restore(ctx.Request().Origin().Context(), backup, run); err != nil {
		return httpx.Err(ctx, http.StatusBadGateway, err.Error())
	}

	return httpx.Ok(ctx, contractshttp.Json{"message": "Backup restored into " + backup.Name + "."})
}

// DeleteRun removes one run and its file.
func (c *BackupController) DeleteRun(ctx contractshttp.Context) contractshttp.Response {
	_, run, failure := c.target(ctx)
	if failure != nil {
		return failure
	}

	if run.File != "" {
		_ = os.Remove(run.File)
	}

	_, _ = facades.Orm().Query().Where("id", run.ID).Delete(&models.BackupRun{})

	return httpx.Ok(ctx, contractshttp.Json{"message": "Backup run deleted."})
}

func (c *BackupController) target(ctx contractshttp.Context) (*models.Backup, *models.BackupRun, contractshttp.Response) {
	backup, err := backups.Default().Find(ctx.Request().Input("uuid"))
	if err != nil {
		return nil, nil, httpx.Err(ctx, http.StatusNotFound, "Backup not found.")
	}

	run, err := backups.Default().FindRun(backup, uint64(ctx.Request().InputInt64("run", 0)))
	if err != nil {
		return nil, nil, httpx.Err(ctx, http.StatusNotFound, "Backup run not found.")
	}

	return backup, run, nil
}

func describe(backup models.Backup) backupResponse {
	response := backupResponse{Backup: backup}

	if hint := backups.Describe(backup.Schedule); hint != "" {
		response.ScheduleHint = hint
	}

	if backup.Enabled {
		reference := backup.UpdatedAt.UTC()
		if backup.LastRunAt != nil {
			reference = backup.LastRunAt.UTC()
		}

		if next := backups.NextRun(backup.Schedule, reference); !next.IsZero() {
			response.NextRunAt = next.Format("2006-01-02T15:04:05Z07:00")
		}
	}

	return response
}

func backupInput(ctx contractshttp.Context) (backups.Input, contractshttp.Response) {
	input := backups.Input{
		ProjectUUID:      ctx.Request().Input("project"),
		Name:             strings.TrimSpace(ctx.Request().Input("name")),
		Target:           ctx.Request().Input("target"),
		ResourceID:       ctx.Request().Input("resource_id"),
		VolumeName:       ctx.Request().Input("volume_name"),
		Schedule:         ctx.Request().Input("schedule"),
		Retention:        ctx.Request().InputInt("retention", 7),
		Enabled:          ctx.Request().InputBool("enabled", true),
		StopDuringBackup: ctx.Request().InputBool("stop_during_backup", false),
	}

	if input.Name == "" {
		return input, httpx.ValidationErrors(ctx, validationErrors{"name": {"The name is required."}})
	}

	return input, nil
}
