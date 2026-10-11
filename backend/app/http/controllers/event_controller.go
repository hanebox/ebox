package controllers

import (
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	contractshttp "github.com/goravel/framework/contracts/http"

	"ebox/app/services/realtime"
	"ebox/app/support/httpx"
)

const ssePingInterval = 15 * time.Second

type EventController struct{}

func NewEventController() *EventController {
	return &EventController{}
}

func (c *EventController) Stream(ctx contractshttp.Context) contractshttp.Response {
	hub := realtime.Default()
	subscriber := hub.Subscribe()

	writer := ctx.Response().Writer()
	flusher, ok := writer.(http.Flusher)
	if !ok {
		hub.Unsubscribe(subscriber)

		return httpx.Err(ctx, http.StatusInternalServerError, "Streaming is not supported.")
	}

	defer hub.Unsubscribe(subscriber)

	writer.Header().Set("Content-Type", "text/event-stream")
	writer.Header().Set("Cache-Control", "no-cache")
	writer.Header().Set("Connection", "keep-alive")
	writer.Header().Set("X-Accel-Buffering", "no")
	writer.WriteHeader(http.StatusOK)

	_, _ = writer.Write([]byte("event: ready\ndata: {}\n\n"))
	flusher.Flush()

	ping := time.NewTicker(ssePingInterval)
	defer ping.Stop()

	for {
		select {
		case <-ctx.Done():
			return nil
		case event, ok := <-subscriber.Events():
			if !ok {
				return nil
			}

			payload, err := json.Marshal(event)
			if err != nil {
				continue
			}

			if _, err := fmt.Fprintf(writer, "event: message\ndata: %s\n\n", payload); err != nil {
				return nil
			}
			flusher.Flush()
		case <-ping.C:
			if _, err := writer.Write([]byte(": ping\n\n")); err != nil {
				return nil
			}
			flusher.Flush()
		}
	}
}
