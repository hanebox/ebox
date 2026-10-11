package account

import (
	"testing"
	"time"
)

func TestLimiterBlocksAfterMaxFailures(t *testing.T) {
	limiter := NewLimiter(3, time.Minute, time.Minute)
	key := "127.0.0.1"

	if !limiter.Allow(key) {
		t.Fatal("Allow() = false before any failures")
	}

	limiter.RecordFailure(key)
	limiter.RecordFailure(key)

	if !limiter.Allow(key) {
		t.Fatal("Allow() = false before reaching the failure limit")
	}

	limiter.RecordFailure(key)

	if limiter.Allow(key) {
		t.Fatal("Allow() = true after reaching the failure limit")
	}
}

func TestLimiterResetClearsFailures(t *testing.T) {
	limiter := NewLimiter(1, time.Minute, time.Minute)
	key := "10.0.0.1"

	limiter.RecordFailure(key)
	if limiter.Allow(key) {
		t.Fatal("Allow() = true after a failure with limit 1")
	}

	limiter.Reset(key)
	if !limiter.Allow(key) {
		t.Fatal("Allow() = false after Reset()")
	}
}

func TestLimiterWindowExpiry(t *testing.T) {
	limiter := NewLimiter(100, 10*time.Millisecond, time.Minute)
	key := "10.0.0.2"

	limiter.RecordFailure(key)
	time.Sleep(20 * time.Millisecond)

	if !limiter.Allow(key) {
		t.Fatal("Allow() = false after the failure window expired")
	}
}
