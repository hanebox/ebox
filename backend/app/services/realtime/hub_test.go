package realtime

import (
	"testing"
	"time"
)

func TestPublishDeliversToAllTopicsSubscriber(t *testing.T) {
	hub := NewHub()
	subscriber := hub.Subscribe()
	defer hub.Unsubscribe(subscriber)

	hub.Publish(Event{Topic: "deployment.1", Type: "deployment.status"})

	select {
	case event := <-subscriber.Events():
		if event.Topic != "deployment.1" {
			t.Fatalf("event topic = %q, want %q", event.Topic, "deployment.1")
		}
		if event.At.IsZero() {
			t.Fatal("event timestamp was not set")
		}
	case <-time.After(time.Second):
		t.Fatal("timed out waiting for the event")
	}
}

func TestTopicFiltering(t *testing.T) {
	hub := NewHub()
	matching := hub.Subscribe("deployment.*")
	defer hub.Unsubscribe(matching)

	other := hub.Subscribe("container.*")
	defer hub.Unsubscribe(other)

	hub.Publish(Event{Topic: "deployment.42", Type: "log"})

	select {
	case event := <-matching.Events():
		if event.Topic != "deployment.42" {
			t.Fatalf("matching subscriber got topic %q", event.Topic)
		}
	case <-time.After(time.Second):
		t.Fatal("matching subscriber did not receive the event")
	}

	select {
	case event := <-other.Events():
		t.Fatalf("other subscriber received unrelated event %q", event.Topic)
	default:
	}
}

func TestUnsubscribeClosesChannel(t *testing.T) {
	hub := NewHub()
	subscriber := hub.Subscribe()
	hub.Unsubscribe(subscriber)

	if _, open := <-subscriber.Events(); open {
		t.Fatal("channel still open after Unsubscribe()")
	}

	hub.Unsubscribe(subscriber)
}
