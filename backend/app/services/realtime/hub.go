package realtime

import (
	"strings"
	"sync"
	"time"
)

const subscriberBuffer = 256

type Event struct {
	Topic string    `json:"topic"`
	Type  string    `json:"type"`
	Data  any       `json:"data,omitempty"`
	At    time.Time `json:"at"`
}

type Hub struct {
	mu   sync.RWMutex
	subs map[*Subscriber]struct{}
}

type Subscriber struct {
	ch     chan Event
	topics []string
	once   sync.Once
}

var defaultHub = NewHub()

func Default() *Hub {
	return defaultHub
}

func NewHub() *Hub {
	return &Hub{subs: make(map[*Subscriber]struct{})}
}

func (h *Hub) Subscribe(topics ...string) *Subscriber {
	subscriber := &Subscriber{
		ch:     make(chan Event, subscriberBuffer),
		topics: topics,
	}

	h.mu.Lock()
	h.subs[subscriber] = struct{}{}
	h.mu.Unlock()

	return subscriber
}

func (h *Hub) Unsubscribe(subscriber *Subscriber) {
	h.mu.Lock()
	if _, ok := h.subs[subscriber]; ok {
		delete(h.subs, subscriber)
		subscriber.once.Do(func() { close(subscriber.ch) })
	}
	h.mu.Unlock()
}

func (h *Hub) Publish(event Event) {
	if event.At.IsZero() {
		event.At = time.Now()
	}

	h.mu.RLock()
	defer h.mu.RUnlock()

	for subscriber := range h.subs {
		if !subscriber.matches(event.Topic) {
			continue
		}

		select {
		case subscriber.ch <- event:
		default:
		}
	}
}

func (s *Subscriber) Events() <-chan Event {
	return s.ch
}

func (s *Subscriber) matches(topic string) bool {
	if len(s.topics) == 0 {
		return true
	}

	for _, pattern := range s.topics {
		if pattern == topic {
			return true
		}
		if strings.HasSuffix(pattern, "*") && strings.HasPrefix(topic, strings.TrimSuffix(pattern, "*")) {
			return true
		}
	}

	return false
}
