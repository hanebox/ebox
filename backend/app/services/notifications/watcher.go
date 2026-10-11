package notifications

import (
	"context"
	"fmt"
	"sync"
	"time"

	"ebox/app/facades"
	"ebox/app/models"
	"ebox/app/services/docker"
	"ebox/app/services/metrics"
	"ebox/app/services/proxy"
)

// Watcher raises events that are not tied to a single action: containers that
// stop unexpectedly, certificates that are about to expire, a full disk and a
// waiting update.
type Watcher struct {
	stop chan struct{}
	once sync.Once
	// raised keeps recently sent events so a flapping container is not announced
	// on every check.
	raised map[string]time.Time
	mu     sync.Mutex
}

const (
	watchInterval     = 5 * time.Minute
	eventQuietPeriod  = 30 * time.Minute
	certificateWindow = 14 * 24 * time.Hour
)

func NewWatcher() *Watcher {
	return &Watcher{stop: make(chan struct{}), raised: map[string]time.Time{}}
}

func (w *Watcher) Signature() string {
	return "ebox:notification-watcher"
}

func (w *Watcher) ShouldRun() bool {
	return true
}

func (w *Watcher) Run() error {
	timer := time.NewTimer(45 * time.Second)
	defer timer.Stop()

	for {
		select {
		case <-w.stop:
			return nil
		case <-timer.C:
		}

		w.check()

		timer.Reset(watchInterval)
	}
}

func (w *Watcher) check() {
	w.checkContainers()
	w.checkCertificates()
	w.checkDisk()
}

// checkContainers announces containers that are not running.
func (w *Watcher) checkContainers() {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	containers, err := docker.Default().List(ctx, docker.ResourceFilter("", ""))
	if err != nil {
		return
	}

	for _, container := range containers {
		resourceType := container.Labels[docker.LabelResourceType]
		if resourceType == "backup" || resourceType == "" {
			continue
		}

		if container.State == "running" {
			w.clear("container:" + container.ID)

			continue
		}

		// Only stopped or dead containers are worth announcing.
		if container.State != "exited" && container.State != "dead" {
			continue
		}

		key := "container:" + container.ID
		if w.recent(key) {
			continue
		}

		name := container.Labels[docker.LabelName]
		if name == "" {
			name = container.Name
		}

		w.raise(key, models.EventContainerDown, "Container stopped: "+name,
			fmt.Sprintf("%s (%s) is %s: %s", name, resourceType, container.State, container.Status),
			map[string]any{
				"resource_type": resourceType,
				"resource_id":   container.Labels[docker.LabelResourceID],
				"container":     container.Name,
				"status":        container.Status,
			})
	}
}

// checkCertificates announces certificates that expire soon.
func (w *Watcher) checkCertificates() {
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()

	certificates, err := proxy.Default().Certificates(ctx)
	if err != nil {
		return
	}

	for _, certificate := range certificates {
		key := "certificate:" + certificate.Domain

		if certificate.NotAfter.IsZero() {
			continue
		}

		remaining := time.Until(certificate.NotAfter.UTC())
		if remaining > certificateWindow {
			w.clear(key)

			continue
		}

		if w.recent(key) {
			continue
		}

		w.raise(key, models.EventCertificateExpiry, "Certificate expiring: "+certificate.Domain,
			fmt.Sprintf("The certificate for %s expires in %d days (%s).", certificate.Domain, certificate.DaysLeft, certificate.NotAfter.UTC().Format("2006-01-02")),
			map[string]any{"domain": certificate.Domain, "expires_at": certificate.NotAfter.UTC(), "days_left": certificate.DaysLeft})
	}
}

// checkDisk announces a nearly full disk.
func (w *Watcher) checkDisk() {
	threshold := 90
	if configured := facades.Config().GetInt("ebox.notify.disk_threshold", 90); configured > 0 && configured < 100 {
		threshold = configured
	}

	sample, ok := metrics.Default().Current()
	if !ok || sample.DiskTotal == 0 {
		return
	}

	percent := int(sample.DiskPercent)
	key := "disk:root"

	if percent < threshold {
		w.clear(key)

		return
	}

	if w.recent(key) {
		return
	}

	w.raise(key, models.EventDiskHigh, "Disk space is running out",
		fmt.Sprintf("The disk holding %s is %d%% full (%s free).", facades.Config().GetString("ebox.data_dir"), percent, humanBytes(sample.DiskTotal-sample.DiskUsed)),
		map[string]any{"percent": percent, "free_bytes": sample.DiskTotal - sample.DiskUsed})
}

// raise sends an event unless it was sent recently.
func (w *Watcher) raise(key, event, title, body string, data map[string]any) {
	w.mu.Lock()
	if raisedAt, ok := w.raised[key]; ok && time.Since(raisedAt) < eventQuietPeriod {
		w.mu.Unlock()

		return
	}
	w.raised[key] = time.Now()
	w.mu.Unlock()

	Default().Notify(event, title, body, data)
}

func (w *Watcher) recent(key string) bool {
	w.mu.Lock()
	defer w.mu.Unlock()

	raisedAt, ok := w.raised[key]

	return ok && time.Since(raisedAt) < eventQuietPeriod
}

func (w *Watcher) clear(key string) {
	w.mu.Lock()
	defer w.mu.Unlock()

	delete(w.raised, key)
}

func (w *Watcher) Shutdown() error {
	w.once.Do(func() { close(w.stop) })

	return nil
}

func humanBytes(bytes uint64) string {
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
