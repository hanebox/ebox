package backups

import (
	"errors"
	"strings"
	"time"

	"github.com/robfig/cron/v3"
)

// scheduleParser accepts the standard five field cron syntax plus @hourly style
// descriptors, without seconds.
var scheduleParser = cron.NewParser(cron.Minute | cron.Hour | cron.Dom | cron.Month | cron.Dow | cron.Descriptor)

// ParseSchedule validates a cron expression and returns its normalised form.
func ParseSchedule(value string) (string, error) {
	expression := strings.TrimSpace(value)
	if expression == "" {
		expression = "0 3 * * *"
	}

	if _, err := scheduleParser.Parse(expression); err != nil {
		return "", errors.New("backups: the schedule is not a valid cron expression (for example \"0 3 * * *\" runs daily at 03:00)")
	}

	return expression, nil
}

// NextRun reports when a schedule fires next after a point in time.
func NextRun(expression string, after time.Time) time.Time {
	schedule, err := scheduleParser.Parse(expression)
	if err != nil {
		return time.Time{}
	}

	return schedule.Next(after)
}

// Due reports whether a schedule should run, given the last execution time.
func Due(expression string, last *time.Time, now time.Time) bool {
	schedule, err := scheduleParser.Parse(expression)
	if err != nil {
		return false
	}

	if last == nil {
		// A brand new schedule waits for its next slot instead of firing at once.
		return false
	}

	return !schedule.Next(last.UTC()).After(now.UTC())
}

// Describe renders a human readable form for the UI when possible.
func Describe(expression string) string {
	switch strings.TrimSpace(expression) {
	case "@hourly":
		return "Every hour"
	case "@daily", "@midnight":
		return "Every day at midnight"
	case "@weekly":
		return "Every week"
	case "@monthly":
		return "Every month"
	case "@yearly", "@annually":
		return "Every year"
	}

	return ""
}
