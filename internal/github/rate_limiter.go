package github

import (
	"context"
	"log/slog"
	"sync"
	"time"
)

// RateLimiter manages API rate limiting with exponential backoff
type RateLimiter struct {
	mu              sync.Mutex
	lastRequestTime time.Time
	minInterval     time.Duration
	logger          *slog.Logger

	// Rate limit tracking
	coreRemaining int
	coreLimit     int
	coreResetTime time.Time

	// Backoff state
	backoffDuration time.Duration
	maxBackoff      time.Duration
}

// NewRateLimiter creates a new rate limiter
func NewRateLimiter(logger *slog.Logger) *RateLimiter {
	return &RateLimiter{
		minInterval:     100 * time.Millisecond, // Minimum time between requests
		logger:          logger,
		backoffDuration: 1 * time.Second,
		maxBackoff:      5 * time.Minute,
		coreRemaining:   5000, // Default GitHub rate limit
		coreLimit:       5000,
	}
}

// Wait blocks until it's safe to make another API request
func (rl *RateLimiter) Wait(ctx context.Context) error {
	// Check context first before acquiring lock
	select {
	case <-ctx.Done():
		return ctx.Err()
	default:
	}

	rl.mu.Lock()
	defer rl.mu.Unlock()

	now := time.Now()

	// Check if we're in a rate limit reset period
	if now.Before(rl.coreResetTime) && rl.coreRemaining <= 0 {
		waitDuration := time.Until(rl.coreResetTime)
		rl.logger.Warn("Rate limit exceeded, waiting for reset",
			"wait_duration", waitDuration,
			"reset_time", rl.coreResetTime)

		// Release lock during wait
		rl.mu.Unlock()

		// Wait for rate limit reset
		select {
		case <-ctx.Done():
			rl.mu.Lock() // Re-acquire before defer unlock
			return ctx.Err()
		case <-time.After(waitDuration):
			// Rate limit should be reset now
		}

		// Re-acquire lock
		rl.mu.Lock()
		rl.coreRemaining = rl.coreLimit
	}

	// Apply minimum interval between requests
	timeSinceLastRequest := time.Since(rl.lastRequestTime)
	if timeSinceLastRequest < rl.minInterval {
		waitTime := rl.minInterval - timeSinceLastRequest

		// Release lock during wait
		rl.mu.Unlock()

		select {
		case <-ctx.Done():
			rl.mu.Lock() // Re-acquire before defer unlock
			return ctx.Err()
		case <-time.After(waitTime):
		}

		// Re-acquire lock
		rl.mu.Lock()
	}

	rl.lastRequestTime = time.Now()
	return nil
}

// UpdateLimits updates the rate limit information from a GitHub response
func (rl *RateLimiter) UpdateLimits(remaining, limit int, resetTime time.Time) {
	rl.mu.Lock()
	defer rl.mu.Unlock()

	rl.coreRemaining = remaining
	rl.coreLimit = limit
	rl.coreResetTime = resetTime

	if remaining < 100 {
		rl.logger.Warn("GitHub API rate limit running low",
			"remaining", remaining,
			"limit", limit,
			"reset_time", resetTime)
	}

	rl.logger.Debug("Rate limit updated",
		"remaining", remaining,
		"limit", limit,
		"reset_time", resetTime)
}

// GetStatus returns the current rate limit status
func (rl *RateLimiter) GetStatus() (remaining, limit int, resetTime time.Time) {
	rl.mu.Lock()
	defer rl.mu.Unlock()
	return rl.coreRemaining, rl.coreLimit, rl.coreResetTime
}

// StartBackoff initiates exponential backoff
func (rl *RateLimiter) StartBackoff(ctx context.Context) error {
	rl.mu.Lock()

	// Calculate backoff duration
	backoff := min(rl.backoffDuration, rl.maxBackoff)

	rl.logger.Info("Starting backoff", "duration", backoff)

	// Increase backoff for next time (exponential)
	rl.backoffDuration *= 2
	if rl.backoffDuration > rl.maxBackoff {
		rl.backoffDuration = rl.maxBackoff
	}

	rl.mu.Unlock()

	// Wait for backoff duration
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-time.After(backoff):
		return nil
	}
}

// ResetBackoff resets the backoff duration after a successful request
func (rl *RateLimiter) ResetBackoff() {
	rl.mu.Lock()
	defer rl.mu.Unlock()
	rl.backoffDuration = 1 * time.Second
	rl.logger.Debug("Backoff reset")
}

// HandleRateLimitError processes a rate limit error and waits appropriately
func (rl *RateLimiter) HandleRateLimitError(ctx context.Context) error {
	rl.mu.Lock()
	resetTime := rl.coreResetTime
	rl.mu.Unlock()

	if time.Now().Before(resetTime) {
		waitDuration := time.Until(resetTime)
		rl.logger.Warn("Rate limit hit, waiting for reset",
			"wait_duration", waitDuration,
			"reset_time", resetTime)

		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(waitDuration):
			rl.mu.Lock()
			rl.coreRemaining = rl.coreLimit
			rl.mu.Unlock()
			return nil
		}
	}

	// If no reset time is set, use exponential backoff
	return rl.StartBackoff(ctx)
}
