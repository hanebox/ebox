package backups

import (
	"context"
	"sync"
	"time"

	"ebox/app/facades"
	"ebox/app/models"
)

const (
	tickInterval = 30 * time.Second
	// dueWindow guards against a schedule firing for a slot that is long past,
	// for example after the panel was offline.
	dueWindow = 24 * time.Hour
)

// Runner executes schedules while the panel is running.
type Runner struct {
	stop chan struct{}
	once sync.Once
}

func NewRunner() *Runner {
	return &Runner{stop: make(chan struct{})}
}

func (r *Runner) Signature() string {
	return "ebox:backup-scheduler"
}

func (r *Runner) ShouldRun() bool {
	return true
}

func (r *Runner) Run() error {
	// Small initial delay so the panel finishes booting before the first pass.
	timer := time.NewTimer(10 * time.Second)
	defer timer.Stop()

	for {
		select {
		case <-r.stop:
			return nil
		case <-timer.C:
		}

		r.tick()

		timer.Reset(tickInterval)
	}
}

func (r *Runner) tick() {
	now := time.Now().UTC()

	var schedules []models.Backup
	if err := facades.Orm().Query().Where("enabled", true).Find(&schedules); err != nil {
		return
	}

	for _, schedule := range schedules {
		if !Due(schedule.Schedule, schedule.LastRunAt, now) {
			continue
		}

		// Never replay slots that are older than the window: the next run wins.
		if schedule.LastRunAt != nil && now.Sub(schedule.LastRunAt.UTC()) > dueWindow {
			markSkipped(&schedule, now)

			continue
		}

		target := schedule

		go func(item models.Backup) {
			_, err := Default().Run(context.Background(), &item, models.DeploymentTriggerManual)
			if err != nil {
				if log := facades.Log(); log != nil {
					log.Warningf("ebox: scheduled backup %s failed to start: %s", item.Name, err)
				}
			}
		}(target)
	}
}

// markSkipped records that a slot passed while the panel was not running.
func markSkipped(schedule *models.Backup, now time.Time) {
	if log := facades.Log(); log != nil {
		log.Infof("ebox: skipping missed backup window for %s, next run at %s", schedule.Name, NextRun(schedule.Schedule, now).Format(time.RFC3339))
	}
}

func (r *Runner) Shutdown() error {
	r.once.Do(func() { close(r.stop) })

	return nil
}
