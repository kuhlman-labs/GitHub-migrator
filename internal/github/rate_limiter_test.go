package github

import (
	"context"
	"log/slog"
	"os"
	"testing"
	"time"
)

func TestNewRateLimiter(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	rl := NewRateLimiter(logger)

	if rl == nil {
		t.Fatal("NewRateLimiter() returned nil")
		return // Prevent staticcheck SA5011
	}

	if rl.minInterval != 100*time.Millisecond {
		t.Errorf("minInterval = %v, want %v", rl.minInterval, 100*time.Millisecond)
	}

	if rl.coreRemaining != 5000 {
		t.Errorf("coreRemaining = %d, want 5000", rl.coreRemaining)
	}

	if rl.coreLimit != 5000 {
		t.Errorf("coreLimit = %d, want 5000", rl.coreLimit)
	}
}

func TestRateLimiter_Wait(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	rl := NewRateLimiter(logger)
	ctx := context.Background()

	// First call should succeed immediately
	start := time.Now()
	err := rl.Wait(ctx)
	elapsed := time.Since(start)

	if err != nil {
		t.Errorf("Wait() error = %v, want nil", err)
	}

	// Should take minimal time for first call
	if elapsed > 50*time.Millisecond {
		t.Errorf("First Wait() took %v, expected minimal time", elapsed)
	}

	// Second call should wait for minInterval
	start = time.Now()
	err = rl.Wait(ctx)
	elapsed = time.Since(start)

	if err != nil {
		t.Errorf("Wait() error = %v, want nil", err)
	}

	// Should wait approximately minInterval
	if elapsed < 90*time.Millisecond || elapsed > 200*time.Millisecond {
		t.Errorf("Second Wait() took %v, expected ~100ms", elapsed)
	}
}

func TestRateLimiter_WaitWithContext(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	rl := NewRateLimiter(logger)

	// Create a context that cancels immediately
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	err := rl.Wait(ctx)
	if err == nil {
		t.Error("Wait() with cancelled context should return error")
	}

	if err != context.Canceled {
		t.Errorf("Wait() error = %v, want %v", err, context.Canceled)
	}
}

func TestRateLimiter_UpdateLimits(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	rl := NewRateLimiter(logger)

	resetTime := time.Now().Add(1 * time.Hour)
	rl.UpdateLimits(100, 5000, resetTime)

	remaining, limit, reset := rl.GetStatus()

	if remaining != 100 {
		t.Errorf("remaining = %d, want 100", remaining)
	}

	if limit != 5000 {
		t.Errorf("limit = %d, want 5000", limit)
	}

	if !reset.Equal(resetTime) {
		t.Errorf("resetTime = %v, want %v", reset, resetTime)
	}
}

func TestRateLimiter_WaitForReset(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	rl := NewRateLimiter(logger)

	// Set rate limit to 0 with reset in near future
	resetTime := time.Now().Add(200 * time.Millisecond)
	rl.UpdateLimits(0, 5000, resetTime)

	ctx := context.Background()
	start := time.Now()
	err := rl.Wait(ctx)
	elapsed := time.Since(start)

	if err != nil {
		t.Errorf("Wait() error = %v, want nil", err)
	}

	// Should wait approximately until reset time
	if elapsed < 150*time.Millisecond {
		t.Errorf("Wait() took %v, expected at least 150ms", elapsed)
	}

	// After reset, remaining should be restored
	remaining, _, _ := rl.GetStatus()
	if remaining != 5000 {
		t.Errorf("remaining after reset = %d, want 5000", remaining)
	}
}

func TestRateLimiter_StartBackoff(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	rl := NewRateLimiter(logger)

	ctx := context.Background()

	// First backoff should be 1 second
	start := time.Now()
	err := rl.StartBackoff(ctx)
	elapsed := time.Since(start)

	if err != nil {
		t.Errorf("StartBackoff() error = %v, want nil", err)
	}

	if elapsed < 900*time.Millisecond || elapsed > 1200*time.Millisecond {
		t.Errorf("First StartBackoff() took %v, expected ~1s", elapsed)
	}

	// Second backoff should be doubled (2 seconds)
	start = time.Now()
	err = rl.StartBackoff(ctx)
	elapsed = time.Since(start)

	if err != nil {
		t.Errorf("StartBackoff() error = %v, want nil", err)
	}

	if elapsed < 1800*time.Millisecond || elapsed > 2200*time.Millisecond {
		t.Errorf("Second StartBackoff() took %v, expected ~2s", elapsed)
	}
}

func TestRateLimiter_ResetBackoff(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	rl := NewRateLimiter(logger)

	// Increase backoff
	rl.backoffDuration = 10 * time.Second

	// Reset backoff
	rl.ResetBackoff()

	if rl.backoffDuration != 1*time.Second {
		t.Errorf("backoffDuration after reset = %v, want 1s", rl.backoffDuration)
	}
}

func TestRateLimiter_HandleRateLimitError(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	rl := NewRateLimiter(logger)

	// Set reset time in near future
	resetTime := time.Now().Add(200 * time.Millisecond)
	rl.UpdateLimits(0, 5000, resetTime)

	ctx := context.Background()
	start := time.Now()
	err := rl.HandleRateLimitError(ctx)
	elapsed := time.Since(start)

	if err != nil {
		t.Errorf("HandleRateLimitError() error = %v, want nil", err)
	}

	// Should wait approximately until reset time
	if elapsed < 150*time.Millisecond {
		t.Errorf("HandleRateLimitError() took %v, expected at least 150ms", elapsed)
	}
}

func TestRateLimiter_HandleRateLimitErrorWithoutReset(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	rl := NewRateLimiter(logger)

	// Set reset time in past (no reset time set)
	resetTime := time.Now().Add(-1 * time.Hour)
	rl.UpdateLimits(0, 5000, resetTime)

	ctx := context.Background()
	start := time.Now()
	err := rl.HandleRateLimitError(ctx)
	elapsed := time.Since(start)

	if err != nil {
		t.Errorf("HandleRateLimitError() error = %v, want nil", err)
	}

	// Should use exponential backoff (1 second)
	if elapsed < 900*time.Millisecond {
		t.Errorf("HandleRateLimitError() took %v, expected at least 900ms", elapsed)
	}
}

func TestRateLimiter_StartBackoffWithCancelledContext(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	rl := NewRateLimiter(logger)

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	err := rl.StartBackoff(ctx)
	if err == nil {
		t.Error("StartBackoff() with cancelled context should return error")
	}

	if err != context.Canceled {
		t.Errorf("StartBackoff() error = %v, want %v", err, context.Canceled)
	}
}

func TestRateLimiter_MaxBackoff(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	rl := NewRateLimiter(logger)
	rl.maxBackoff = 100 * time.Millisecond

	ctx := context.Background()

	// Increase backoff multiple times
	for range 10 {
		_ = rl.StartBackoff(ctx)
	}

	// Backoff should not exceed maxBackoff
	if rl.backoffDuration > rl.maxBackoff {
		t.Errorf("backoffDuration = %v, should not exceed maxBackoff = %v",
			rl.backoffDuration, rl.maxBackoff)
	}
}
