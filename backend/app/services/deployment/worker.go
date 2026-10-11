package deployment

import (
	"sync"
	"time"

	"github.com/goravel/framework/contracts/queue"

	"ebox/app/facades"
)

const workerRestartDelay = 5 * time.Second

// Worker runs the in-process queue worker for the deployments queue.
type Worker struct {
	mu     sync.Mutex
	worker queue.Worker
	stop   chan struct{}
	once   sync.Once
}

func NewWorker() *Worker {
	return &Worker{stop: make(chan struct{})}
}

func (w *Worker) Signature() string {
	return "ebox:deploy-worker"
}

func (w *Worker) ShouldRun() bool {
	return true
}

// Run keeps a queue worker alive for the deployments queue. Errors are logged
// and retried so a queue hiccup never shuts the panel down.
func (w *Worker) Run() error {
	for {
		select {
		case <-w.stop:
			return nil
		default:
		}

		worker := facades.Queue().Worker(queue.Args{
			Connection: facades.Config().GetString("ebox.deploy.connection", "database"),
			Queue:      facades.Config().GetString("ebox.deploy.queue", "deployments"),
			Concurrent: 1,
			Tries:      1,
		})

		w.mu.Lock()
		w.worker = worker
		w.mu.Unlock()

		if err := worker.Run(); err != nil {
			facades.Log().Errorf("ebox: deploy worker stopped: %s", err)
		}

		select {
		case <-w.stop:
			return nil
		case <-time.After(workerRestartDelay):
		}
	}
}

func (w *Worker) Shutdown() error {
	w.once.Do(func() {
		close(w.stop)

		w.mu.Lock()
		worker := w.worker
		w.mu.Unlock()

		if worker != nil {
			_ = worker.Shutdown()
		}
	})

	return nil
}
