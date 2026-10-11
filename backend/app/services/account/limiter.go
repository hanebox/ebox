package account

import (
	"sync"
	"time"
)

type limiterEntry struct {
	failures     int
	windowStart  time.Time
	blockedUntil time.Time
}

type Limiter struct {
	mu            sync.Mutex
	entries       map[string]*limiterEntry
	maxFailures   int
	window        time.Duration
	blockDuration time.Duration
}

func NewLimiter(maxFailures int, window, blockDuration time.Duration) *Limiter {
	return &Limiter{
		entries:       make(map[string]*limiterEntry),
		maxFailures:   maxFailures,
		window:        window,
		blockDuration: blockDuration,
	}
}

func (l *Limiter) Allow(key string) bool {
	l.mu.Lock()
	defer l.mu.Unlock()

	entry, ok := l.entries[key]
	if !ok {
		return true
	}

	if entry.blockedUntil.After(time.Now()) {
		return false
	}

	if time.Since(entry.windowStart) > l.window {
		delete(l.entries, key)
	}

	return true
}

func (l *Limiter) RecordFailure(key string) {
	l.mu.Lock()
	defer l.mu.Unlock()

	entry, ok := l.entries[key]
	if !ok || time.Since(entry.windowStart) > l.window {
		entry = &limiterEntry{windowStart: time.Now()}
		l.entries[key] = entry
	}

	entry.failures++
	if entry.failures >= l.maxFailures {
		entry.blockedUntil = time.Now().Add(l.blockDuration)
		entry.failures = 0
		entry.windowStart = time.Now()
	}
}

func (l *Limiter) Reset(key string) {
	l.mu.Lock()
	defer l.mu.Unlock()

	delete(l.entries, key)
}

var loginLimiter = NewLimiter(10, 5*time.Minute, 15*time.Minute)

func LoginLimiter() *Limiter {
	return loginLimiter
}
