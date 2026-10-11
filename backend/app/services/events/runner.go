package events

import (
	"context"
	"sync"
	"time"

	"ebox/app/facades"
	"ebox/app/services/docker"
	"ebox/app/services/realtime"
)

const reconnectDelay = 5 * time.Second

// Runner forwards Docker container events to the realtime hub so the UI can
// update status chips without polling.
type Runner struct {
	stop chan struct{}
	once sync.Once
}

func NewRunner() *Runner {
	return &Runner{stop: make(chan struct{})}
}

func (r *Runner) Signature() string {
	return "ebox:docker-events"
}

func (r *Runner) ShouldRun() bool {
	return true
}

func (r *Runner) Run() error {
	for {
		select {
		case <-r.stop:
			return nil
		default:
		}

		if err := r.consume(); err != nil {
			if log := facades.Log(); log != nil {
				log.Debugf("ebox: docker event stream reconnecting: %s", err)
			}
		}

		select {
		case <-r.stop:
			return nil
		case <-time.After(reconnectDelay):
		}
	}
}

func (r *Runner) consume() error {
	if err := docker.Default().Available(context.Background()); err != nil {
		return err
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	stream, err := docker.Default().Events(ctx, docker.Filters{}.Add("type", "container"))
	if err != nil {
		return err
	}

	for {
		select {
		case <-r.stop:
			return nil
		case err, ok := <-stream.Err:
			if !ok {
				return nil
			}
			if err != nil {
				return err
			}
		case message, ok := <-stream.Messages:
			if !ok {
				return nil
			}

			action, interesting := docker.EventAction(message.Action)
			if !interesting {
				continue
			}

			labels := message.Actor.Attributes
			resourceType := labels[docker.LabelResourceType]
			resourceID := labels[docker.LabelResourceID]

			if resourceType == "" || resourceID == "" {
				continue
			}

			payload := map[string]any{
				"action":         action,
				"container_id":   message.Actor.ID,
				"container_name": labels[docker.LabelName],
				"resource_type":  resourceType,
				"resource_id":    resourceID,
				"at":             time.Unix(message.Time, 0).UTC(),
			}

			hub := realtime.Default()
			hub.Publish(realtime.Event{
				Topic: "container." + message.Actor.ID,
				Type:  "state",
				Data:  payload,
			})
			hub.Publish(realtime.Event{
				Topic: resourceType + "." + resourceID,
				Type:  "state",
				Data:  payload,
			})
		}
	}
}

func (r *Runner) Shutdown() error {
	r.once.Do(func() { close(r.stop) })

	return nil
}
