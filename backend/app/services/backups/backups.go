package backups

import (
	"compress/gzip"
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"

	"ebox/app/facades"
	"ebox/app/models"
	"ebox/app/services/databases"
	"ebox/app/services/docker"
	"ebox/app/services/notifications"
	"ebox/app/services/projects"
)

const (
	runTimeout      = 2 * time.Hour
	helperImage     = "alpine:3.20"
	volumeMountPath = "/volume"
	keepRunsPerItem = 50
)

type Input struct {
	ProjectUUID      string
	Name             string
	Target           string
	ResourceID       string
	VolumeName       string
	Schedule         string
	Retention        int
	Enabled          bool
	StopDuringBackup bool
}

type Service struct {
	mu      sync.Mutex
	running map[uint64]bool
}

var defaultService = &Service{running: map[uint64]bool{}}

func Default() *Service {
	return defaultService
}

func (s *Service) List(projectUUID string) ([]models.Backup, error) {
	query := facades.Orm().Query().Order("name asc")

	if projectUUID != "" {
		project, err := projects.Default().Find(projectUUID)
		if err != nil {
			return nil, err
		}
		query = query.Where("project_id", project.ID)
	}

	var list []models.Backup
	if err := query.Find(&list); err != nil {
		return nil, err
	}

	return list, nil
}

func (s *Service) Find(uuidValue string) (*models.Backup, error) {
	var backup models.Backup
	if err := facades.Orm().Query().Where("uuid", uuidValue).FirstOrFail(&backup); err != nil {
		return nil, err
	}

	return &backup, nil
}

func (s *Service) Create(input Input) (*models.Backup, error) {
	backup := &models.Backup{UUID: uuid.NewString()}

	if err := s.apply(backup, input); err != nil {
		return nil, err
	}

	if err := facades.Orm().Query().Create(backup); err != nil {
		return nil, err
	}

	return backup, nil
}

func (s *Service) Update(backup *models.Backup, input Input) error {
	if err := s.apply(backup, input); err != nil {
		return err
	}

	return facades.Orm().Query().Save(backup)
}

func (s *Service) Delete(backup *models.Backup) error {
	if runs, err := s.Runs(backup, 0); err == nil {
		for _, run := range runs {
			if run.File != "" {
				_ = os.Remove(run.File)
			}
		}
	}

	_ = os.RemoveAll(s.Directory(backup))
	_, _ = facades.Orm().Query().Where("backup_id", backup.ID).Delete(&models.BackupRun{})

	_, err := facades.Orm().Query().Delete(backup)

	return err
}

func (s *Service) Runs(backup *models.Backup, limit int) ([]models.BackupRun, error) {
	query := facades.Orm().Query().Where("backup_id", backup.ID).Order("id desc")
	if limit > 0 {
		query = query.Limit(limit)
	}

	var runs []models.BackupRun
	if err := query.Find(&runs); err != nil {
		return nil, err
	}

	return runs, nil
}

func (s *Service) FindRun(backup *models.Backup, id uint64) (*models.BackupRun, error) {
	var run models.BackupRun
	if err := facades.Orm().Query().
		Where("backup_id", backup.ID).
		Where("id", id).
		FirstOrFail(&run); err != nil {
		return nil, err
	}

	return &run, nil
}

// Directory is where the artefacts of one schedule live.
func (s *Service) Directory(backup *models.Backup) string {
	return filepath.Join(facades.Config().GetString("ebox.backups_dir"), backup.UUID)
}

func (s *Service) apply(backup *models.Backup, input Input) error {
	name := strings.TrimSpace(input.Name)
	if name == "" {
		return errors.New("backups: the name is required")
	}

	target := input.Target
	if target != models.BackupTargetDatabase && target != models.BackupTargetVolume {
		return errors.New("backups: the target must be a database or a volume")
	}

	cron, err := ParseSchedule(input.Schedule)
	if err != nil {
		return err
	}

	retention := input.Retention
	if retention < 1 {
		retention = 7
	}
	if retention > 365 {
		retention = 365
	}

	backup.Name = name
	backup.Target = target
	backup.Schedule = cron
	backup.Retention = retention
	backup.Enabled = input.Enabled
	backup.StopDuringBackup = input.StopDuringBackup

	if input.ProjectUUID != "" {
		project, err := projects.Default().Find(input.ProjectUUID)
		if err != nil {
			return errors.New("backups: the project does not exist")
		}
		backup.ProjectID = &project.ID
	} else {
		backup.ProjectID = nil
	}

	switch target {
	case models.BackupTargetDatabase:
		database, err := databases.Default().Find(input.ResourceID)
		if err != nil {
			return errors.New("backups: the database does not exist")
		}

		if _, err := dumpCommand(database.Type, models.DatabaseCredentials{}); err != nil {
			return err
		}

		backup.ResourceID = database.UUID
		backup.VolumeName = ""

		if backup.ProjectID == nil {
			backup.ProjectID = &database.ProjectID
		}
	case models.BackupTargetVolume:
		volume := strings.TrimSpace(input.VolumeName)
		if volume == "" {
			return errors.New("backups: select a volume")
		}

		backup.VolumeName = volume
		backup.ResourceID = ""
	}

	return nil
}

// dumpCommand builds the shell command that extracts a database.
func dumpCommand(databaseType string, credentials models.DatabaseCredentials) ([]string, error) {
	switch databaseType {
	case models.DatabaseTypePostgres:
		return []string{"sh", "-c", fmt.Sprintf(
			"PGPASSWORD=%s pg_dump --no-owner --clean --if-exists -h 127.0.0.1 -U %s -d %s",
			shellQuote(credentials.Password), shellQuote(credentials.Username), shellQuote(credentials.Database),
		)}, nil
	case models.DatabaseTypeMySQL, models.DatabaseTypeMariaDB:
		return []string{"sh", "-c", fmt.Sprintf(
			"mysqldump --single-transaction --quick --skip-lock-tables -h 127.0.0.1 -u %s -p%s %s",
			shellQuote(credentials.Username), shellQuote(credentials.Password), shellQuote(credentials.Database),
		)}, nil
	case models.DatabaseTypeRedis:
		return []string{"sh", "-c", fmt.Sprintf(
			"redis-cli -h 127.0.0.1 -a %s --no-auth-warning --rdb /dev/stdout",
			shellQuote(credentials.Password),
		)}, nil
	case models.DatabaseTypeMongo:
		return nil, errors.New("backups: MongoDB has no dump tool in its official image. Back up its data volume instead and stop the container during the backup for a consistent copy.")
	}

	return nil, fmt.Errorf("backups: %s backups are not supported", databaseType)
}

// restoreCommand builds the shell command that loads a dump back in.
func restoreCommand(databaseType string, credentials models.DatabaseCredentials) ([]string, error) {
	switch databaseType {
	case models.DatabaseTypePostgres:
		return []string{"sh", "-c", fmt.Sprintf(
			"PGPASSWORD=%s psql -v ON_ERROR_STOP=1 -h 127.0.0.1 -U %s -d %s",
			shellQuote(credentials.Password), shellQuote(credentials.Username), shellQuote(credentials.Database),
		)}, nil
	case models.DatabaseTypeMySQL, models.DatabaseTypeMariaDB:
		return []string{"sh", "-c", fmt.Sprintf(
			"mysql -h 127.0.0.1 -u %s -p%s %s",
			shellQuote(credentials.Username), shellQuote(credentials.Password), shellQuote(credentials.Database),
		)}, nil
	}

	return nil, errors.New("backups: restoring this engine automatically is not supported. Download the archive and load it manually.")
}

// Run executes a backup now.
func (s *Service) Run(ctx context.Context, backup *models.Backup, trigger string) (*models.BackupRun, error) {
	s.mu.Lock()
	if s.running[backup.ID] {
		s.mu.Unlock()

		return nil, errors.New("backups: this backup is already running")
	}
	s.running[backup.ID] = true
	s.mu.Unlock()

	defer func() {
		s.mu.Lock()
		delete(s.running, backup.ID)
		s.mu.Unlock()
	}()

	started := time.Now().UTC()
	run := &models.BackupRun{
		BackupID:  backup.ID,
		Status:    models.BackupStatusRunning,
		Trigger:   trigger,
		StartedAt: &started,
	}

	if err := facades.Orm().Query().Create(run); err != nil {
		return nil, err
	}

	runCtx, cancel := context.WithTimeout(context.WithoutCancel(ctx), runTimeout)
	defer cancel()

	path, size, err := s.capture(runCtx, backup, run)

	finished := time.Now().UTC()
	run.FinishedAt = &finished
	run.DurationMS = finished.Sub(started).Milliseconds()

	if err != nil {
		run.Status = models.BackupStatusFailed
		run.Error = err.Error()
	} else {
		run.Status = models.BackupStatusSuccess
		run.File = path
		run.SizeBytes = size
	}

	_, err = facades.Orm().Query().
		Model(&models.BackupRun{}).
		Where("id", run.ID).
		Update(map[string]any{
			"status":      run.Status,
			"file":        run.File,
			"size_bytes":  run.SizeBytes,
			"duration_ms": run.DurationMS,
			"error":       run.Error,
			"finished_at": run.FinishedAt,
		})
	if err != nil {
		if log := facades.Log(); log != nil {
			log.Errorf("ebox: recording backup run %d failed: %s", run.ID, err)
		}
	}

	backup.LastRunAt = &finished
	backup.LastStatus = run.Status
	backup.LastError = run.Error
	_ = facades.Orm().Query().Save(backup)

	s.notify(backup, run)
	s.prune(backup)
	s.trimRuns(backup)

	return run, nil
}

// capture writes the artefact for a run and returns its path and size.
func (s *Service) capture(ctx context.Context, backup *models.Backup, run *models.BackupRun) (string, int64, error) {
	directory := s.Directory(backup)
	if err := os.MkdirAll(directory, 0o700); err != nil {
		return "", 0, err
	}

	extension := "tar.gz"
	if backup.Target == models.BackupTargetDatabase {
		extension = "sql.gz"
	}

	name := fmt.Sprintf("%s-%s-%s.%s", backup.Slug(), backup.Target, run.CreatedAt.UTC().Format("20060102-150405"), extension)
	path := filepath.Join(directory, name)

	file, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0o600)
	if err != nil {
		return "", 0, err
	}

	var captureErr error

	if backup.Target == models.BackupTargetDatabase {
		captureErr = s.captureDatabase(ctx, backup, file)
	} else {
		captureErr = s.captureVolume(ctx, backup, file)
	}

	if captureErr != nil {
		_ = file.Close()
		_ = os.Remove(path)

		return "", 0, captureErr
	}

	if err := file.Sync(); err != nil {
		_ = file.Close()

		return "", 0, err
	}

	if err := file.Close(); err != nil {
		return "", 0, err
	}

	info, err := os.Stat(path)
	if err != nil {
		return "", 0, err
	}

	if info.Size() == 0 {
		_ = os.Remove(path)

		return "", 0, errors.New("backups: the backup produced an empty file")
	}

	return path, info.Size(), nil
}

func (s *Service) captureDatabase(ctx context.Context, backup *models.Backup, output io.Writer) error {
	database, err := databases.Default().Find(backup.ResourceID)
	if err != nil {
		return errors.New("backups: the database no longer exists")
	}

	credentials, err := databases.Default().Credentials(database)
	if err != nil {
		return err
	}

	command, err := dumpCommand(database.Type, credentials)
	if err != nil {
		return err
	}

	containers, err := docker.Default().FindByResource(ctx, docker.ResourceDatabase, database.UUID)
	if err != nil || len(containers) == 0 {
		return errors.New("backups: the database container does not exist")
	}

	if containers[0].State != "running" {
		return errors.New("backups: the database container is not running")
	}

	compressed, err := gzip.NewWriterLevel(output, gzip.BestSpeed)
	if err != nil {
		return err
	}

	exitCode, stderr, err := docker.Default().StreamExec(ctx, containers[0].ID, command, compressed)
	if err != nil {
		return err
	}

	if exitCode != 0 {
		return fmt.Errorf("backups: the dump command exited with code %d: %s", exitCode, lastLines(stderr, 3))
	}

	if err := compressed.Close(); err != nil {
		return err
	}

	return nil
}

// captureVolume archives a named volume through a short lived helper container.
func (s *Service) captureVolume(ctx context.Context, backup *models.Backup, output io.Writer) error {
	if err := docker.Default().PullImage(ctx, helperImage, nil); err != nil {
		return err
	}

	helperID, err := s.helper(ctx, backup, "sleep 600")
	if err != nil {
		return err
	}
	defer func() {
		_ = docker.Default().Remove(context.WithoutCancel(ctx), helperID, true)
	}()

	stream, err := docker.Default().CopyFromContainer(ctx, helperID, volumeMountPath)
	if err != nil {
		return errors.New("backups: the volume could not be read: " + err.Error())
	}
	defer func() { _ = stream.Close() }()

	compressed, err := gzip.NewWriterLevel(output, gzip.BestSpeed)
	if err != nil {
		return err
	}

	if err := rewriteTar(stream, compressed, volumeMountPath, ""); err != nil {
		return err
	}

	return compressed.Close()
}

func (s *Service) restoreDatabase(ctx context.Context, backup *models.Backup, input io.Reader) error {
	database, err := databases.Default().Find(backup.ResourceID)
	if err != nil {
		return errors.New("backups: the database no longer exists")
	}

	credentials, err := databases.Default().Credentials(database)
	if err != nil {
		return err
	}

	command, err := restoreCommand(database.Type, credentials)
	if err != nil {
		return err
	}

	containers, err := docker.Default().FindByResource(ctx, docker.ResourceDatabase, database.UUID)
	if err != nil || len(containers) == 0 {
		return errors.New("backups: the database container does not exist")
	}

	if containers[0].State != "running" {
		return errors.New("backups: the database container is not running")
	}

	reader, err := gzip.NewReader(input)
	if err != nil {
		return errors.New("backups: the archive is not a valid gzip file")
	}
	defer func() { _ = reader.Close() }()

	exitCode, output, err := docker.Default().ExecWithInput(ctx, containers[0].ID, command, reader)
	if err != nil {
		return err
	}

	if exitCode != 0 {
		return fmt.Errorf("backups: the restore command exited with code %d: %s", exitCode, lastLines(output, 5))
	}

	return nil
}

func (s *Service) restoreVolume(ctx context.Context, backup *models.Backup, input io.Reader) error {
	if backup.VolumeName == "" {
		return errors.New("backups: this run has no volume to restore")
	}

	if err := docker.Default().PullImage(ctx, helperImage, nil); err != nil {
		return err
	}

	helperID, err := s.helper(ctx, backup, "sleep 600")
	if err != nil {
		return err
	}
	defer func() {
		_ = docker.Default().Remove(context.WithoutCancel(ctx), helperID, true)
	}()

	reader, err := gzip.NewReader(input)
	if err != nil {
		return errors.New("backups: the archive is not a valid gzip file")
	}
	defer func() { _ = reader.Close() }()

	// The volume is emptied first so the archive defines its full contents, then the
	// tar is re-prefixed and unpacked into the mount point by docker.
	if _, err := docker.Default().Run(ctx, helperID, []string{"sh", "-c", "rm -rf " + volumeMountPath + "/* " + volumeMountPath + "/.[!.]* 2>/dev/null || true"}); err != nil {
		return err
	}

	pipeReader, pipeWriter := io.Pipe()

	go func() {
		_ = rewriteTar(reader, pipeWriter, "", volumeMountPath)
		_ = pipeWriter.Close()
	}()

	if err := docker.Default().CopyToContainer(ctx, helperID, "/", pipeReader); err != nil {
		return errors.New("backups: the archive could not be written back: " + err.Error())
	}

	return nil
}

// helper starts a throwaway container with the volume mounted.
func (s *Service) helper(ctx context.Context, backup *models.Backup, command string) (string, error) {
	spec := docker.Spec{
		Name:    "ebox-backup-" + backup.Slug(),
		Image:   helperImage,
		Command: []string{"sh", "-c", command},
		Labels:  docker.ManagedLabels("backup", backup.UUID, backup.Name),
		Volumes: []string{backup.VolumeName + ":" + volumeMountPath},
		Restart: "no",
	}

	_ = docker.Default().RemoveByName(ctx, spec.Name, true)

	containerID, err := docker.Default().Create(ctx, spec)
	if err != nil {
		return "", err
	}

	if _, err := docker.Default().WaitRunning(ctx, containerID, time.Minute); err != nil {
		_ = docker.Default().Remove(context.WithoutCancel(ctx), containerID, true)

		return "", err
	}

	return containerID, nil
}

// Restore puts a backup back into the database or volume it came from.
func (s *Service) Restore(ctx context.Context, backup *models.Backup, run *models.BackupRun) error {
	if run.File == "" {
		return errors.New("backups: this run has no file on disk anymore")
	}

	file, err := os.Open(run.File)
	if err != nil {
		return errors.New("backups: the backup file could not be opened")
	}
	defer func() { _ = file.Close() }()

	if backup.Target == models.BackupTargetDatabase {
		return s.restoreDatabase(ctx, backup, file)
	}

	return s.restoreVolume(ctx, backup, file)
}

// notify announces the result of a run.
func (s *Service) notify(backup *models.Backup, run *models.BackupRun) {
	if run.Status == models.BackupStatusSuccess {
		notifications.Default().Notify(models.EventBackupSuccess,
			"Backup finished: "+backup.Name,
			fmt.Sprintf("%s backed up in %s (%s).", backup.Name, shortDuration(run.DurationMS), humanSize(run.SizeBytes)),
			map[string]any{"backup": backup.Name, "target": backup.Target, "size": run.SizeBytes, "file": filepath.Base(run.File)})

		return
	}

	notifications.Default().Notify(models.EventBackupFailed,
		"Backup failed: "+backup.Name,
		run.Error,
		map[string]any{"backup": backup.Name, "target": backup.Target, "error": run.Error})
}

// prune enforces the retention count.
func (s *Service) prune(backup *models.Backup) {
	runs, err := s.Runs(backup, 0)
	if err != nil {
		return
	}

	kept := 0
	for _, run := range runs {
		if run.Status != models.BackupStatusSuccess || run.File == "" {
			continue
		}

		kept++
		if kept <= backup.Retention {
			continue
		}

		_ = os.Remove(run.File)
		_, _ = facades.Orm().Query().
			Model(&models.BackupRun{}).
			Where("id", run.ID).
			Update(map[string]any{"file": ""})
	}
}

func (s *Service) trimRuns(backup *models.Backup) {
	var stale []models.BackupRun
	if err := facades.Orm().Query().Where("backup_id", backup.ID).Order("id desc").Offset(keepRunsPerItem).Find(&stale); err != nil || len(stale) == 0 {
		return
	}

	ids := make([]uint64, 0, len(stale))
	for _, run := range stale {
		ids = append(ids, run.ID)
	}

	_, _ = facades.Orm().Query().Where("id in ?", ids).Delete(&models.BackupRun{})
}

// lastLines trims command output for an error message.
func lastLines(output string, count int) string {
	lines := []string{}
	for _, line := range strings.Split(strings.TrimSpace(output), "\n") {
		if trimmed := strings.TrimSpace(line); trimmed != "" {
			lines = append(lines, trimmed)
		}
	}

	if len(lines) == 0 {
		return "no output"
	}

	if len(lines) > count {
		lines = lines[len(lines)-count:]
	}

	return strings.Join(lines, " | ")
}

func shellQuote(value string) string {
	return "'" + strings.ReplaceAll(value, "'", `'\''`) + "'"
}

func shortDuration(milliseconds int64) string {
	duration := time.Duration(milliseconds) * time.Millisecond
	if duration < time.Second {
		return duration.Round(time.Millisecond).String()
	}

	return duration.Round(time.Second).String()
}

func humanSize(bytes int64) string {
	const unit = 1024

	if bytes < unit {
		return fmt.Sprintf("%d B", bytes)
	}

	value := float64(bytes)
	units := []string{"kB", "MB", "GB", "TB"}
	index := -1

	for value >= unit && index < len(units)-1 {
		value /= unit
		index++
	}

	return fmt.Sprintf("%.1f %s", value, units[index])
}
