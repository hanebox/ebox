package deployment

import (
	"strings"
	"sync"
	"time"

	"ebox/app/facades"
	"ebox/app/models"
	"ebox/app/services/realtime"
)

const maxLogLines = 20000

type logWriter struct {
	mu         sync.Mutex
	deployment *models.Deployment
	lines      []string
	seq        int
	truncated  bool
	stop       chan struct{}
	done       chan struct{}
	once       sync.Once
}

func newLogWriter(deployment *models.Deployment) *logWriter {
	writer := &logWriter{
		deployment: deployment,
		stop:       make(chan struct{}),
		done:       make(chan struct{}),
	}

	if deployment.Log != "" {
		writer.lines = strings.Split(strings.TrimRight(deployment.Log, "\n"), "\n")
		writer.seq = len(writer.lines)
	}

	go writer.loop()

	return writer
}

// Line appends a line to the deployment log and publishes it to the event hub.
func (w *logWriter) Line(line string) {
	trimmed := strings.TrimRight(line, "\r\n")

	w.mu.Lock()
	if len(w.lines) >= maxLogLines {
		if !w.truncated {
			w.truncated = true
			w.lines = append(w.lines, "… log truncated, download the full log after the deployment")
		}
		seq := len(w.lines) - 1
		w.mu.Unlock()

		w.publish(seq, line)

		return
	}

	w.lines = append(w.lines, trimmed)
	w.seq = len(w.lines)
	seq := len(w.lines) - 1
	w.mu.Unlock()

	w.publish(seq, trimmed)
}

func (w *logWriter) publish(seq int, line string) {
	realtime.Default().Publish(realtime.Event{
		Topic: "deployment." + w.deployment.UUID,
		Type:  "log",
		Data:  map[string]any{"seq": seq, "line": line},
	})
}

// Lines returns the number of stored log lines.
func (w *logWriter) Lines() int {
	w.mu.Lock()
	defer w.mu.Unlock()

	return len(w.lines)
}

// Snapshot returns the current line count and the lines after the given
// sequence number.
func (w *logWriter) Snapshot(since int) (int, []string) {
	w.mu.Lock()
	defer w.mu.Unlock()

	if since < 0 {
		since = 0
	}
	if since >= len(w.lines) {
		return len(w.lines), nil
	}

	lines := make([]string, len(w.lines)-since)
	copy(lines, w.lines[since:])

	return len(w.lines), lines
}

func (w *logWriter) loop() {
	defer close(w.done)

	ticker := time.NewTicker(2 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-w.stop:
			return
		case <-ticker.C:
			w.flush()
		}
	}
}

func (w *logWriter) flush() {
	w.mu.Lock()
	content := strings.Join(w.lines, "\n")
	truncated := w.truncated
	w.mu.Unlock()

	if truncated {
		content += "\n…"
	}

	_, _ = facades.Orm().Query().
		Model(&models.Deployment{}).
		Where("id", w.deployment.ID).
		Update("log", content)
}

// Close stops the flush loop and persists the remaining lines.
func (w *logWriter) Close() {
	w.once.Do(func() {
		close(w.stop)
		<-w.done
		w.flush()
	})
}
