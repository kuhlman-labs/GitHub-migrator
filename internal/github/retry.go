package github

import (
	"context"
	"fmt"
	"log/slog"
	"time"
)

// RetryConfig configures retry behavior
type RetryConfig struct {
	MaxAttempts     int
	InitialBackoff  time.Duration
	MaxBackoff      time.Duration
	BackoffMultiple float64
}

// DefaultRetryConfig returns sensible defaults for retry configuration
func DefaultRetryConfig() RetryConfig {
	return RetryConfig{
		MaxAttempts:     5, // Increased to allow recovery from secondary rate limits
		InitialBackoff:  1 * time.Second,
		MaxBackoff:      30 * time.Second,
		BackoffMultiple: 2.0,
	}
}

// SecondaryRateLimitBackoff is the backoff duration for secondary rate limit errors
// GitHub recommends waiting "a few minutes" - we use 60 seconds as a reasonable default
const SecondaryRateLimitBackoff = 60 * time.Second

// RateLimitResetBuffer is added to the parsed reset time to ensure the rate limit is actually reset
const RateLimitResetBuffer = 5 * time.Second

// MinRateLimitWait is the minimum wait time for rate limit errors
const MinRateLimitWait = 10 * time.Second

// MaxRateLimitWait is the maximum wait time for rate limit errors (15 minutes)
const MaxRateLimitWait = 15 * time.Minute

// Retryer handles retry logic with exponential backoff
type Retryer struct {
	config      RetryConfig
	rateLimiter *RateLimiter
	logger      *slog.Logger
}

// NewRetryer creates a new retryer
func NewRetryer(config RetryConfig, rateLimiter *RateLimiter, logger *slog.Logger) *Retryer {
	return &Retryer{
		config:      config,
		rateLimiter: rateLimiter,
		logger:      logger,
	}
}

// calculateRateLimitWait determines how long to wait for a rate limit to reset
func (r *Retryer) calculateRateLimitWait(err error) time.Duration {
	// Try to parse the reset time from the error message
	resetTime, hasResetTime := ParseRateLimitResetTime(err)

	var waitDuration time.Duration
	if hasResetTime {
		waitDuration = time.Until(resetTime)
		// Add buffer to ensure rate limit is actually reset
		waitDuration += RateLimitResetBuffer
	} else {
		// Default wait time if we can't parse the reset time
		waitDuration = SecondaryRateLimitBackoff
	}

	// Ensure minimum and maximum bounds
	if waitDuration < MinRateLimitWait {
		waitDuration = MinRateLimitWait
	}
	if waitDuration > MaxRateLimitWait {
		waitDuration = MaxRateLimitWait
	}

	return waitDuration
}

// RetryFunc is a function that can be retried
type RetryFunc func(ctx context.Context) error

// Do executes a function with retry logic
func (r *Retryer) Do(ctx context.Context, operation string, fn RetryFunc) error {
	var lastErr error
	backoff := r.config.InitialBackoff

	for attempt := 1; attempt <= r.config.MaxAttempts; attempt++ {
		// Wait for rate limiter before each attempt
		if err := r.rateLimiter.Wait(ctx); err != nil {
			return fmt.Errorf("rate limiter wait failed: %w", err)
		}

		// Execute the function
		err := fn(ctx)
		if err == nil {
			// Success - reset backoff on rate limiter
			r.rateLimiter.ResetBackoff()
			if attempt > 1 {
				r.logger.Info("Operation succeeded after retry",
					"operation", operation,
					"attempt", attempt)
			}
			return nil
		}

		lastErr = err

		// Check if we should retry
		if !IsRetryableError(err) {
			r.logger.Debug("Non-retryable error encountered",
				"operation", operation,
				"attempt", attempt,
				"error", err)
			return err
		}

		// Don't retry on last attempt
		if attempt == r.config.MaxAttempts {
			break
		}

		// Handle blocked rate limit errors (go-github's client-side blocking)
		// These contain the reset time in the error message
		if IsRateLimitBlockedError(err) {
			waitDuration := r.calculateRateLimitWait(err)
			r.logger.Warn("Rate limit blocked, waiting for reset",
				"operation", operation,
				"attempt", attempt,
				"wait_duration", waitDuration)

			select {
			case <-ctx.Done():
				return fmt.Errorf("context cancelled during rate limit wait: %w", ctx.Err())
			case <-time.After(waitDuration):
				// Continue with next attempt after waiting
			}
			continue
		}

		// Handle secondary rate limit errors with longer backoff
		// Secondary limits require waiting "a few minutes" before retrying
		if IsSecondaryRateLimitError(err) {
			r.logger.Warn("Secondary rate limit hit, waiting before retry",
				"operation", operation,
				"attempt", attempt,
				"backoff", SecondaryRateLimitBackoff)

			select {
			case <-ctx.Done():
				return fmt.Errorf("context cancelled during secondary rate limit wait: %w", ctx.Err())
			case <-time.After(SecondaryRateLimitBackoff):
				// Continue with next attempt after waiting
			}
			continue
		}

		// Handle primary rate limit errors
		if IsRateLimitError(err) {
			r.logger.Warn("Rate limit error, waiting before retry",
				"operation", operation,
				"attempt", attempt)
			if err := r.rateLimiter.HandleRateLimitError(ctx); err != nil {
				return fmt.Errorf("rate limit handling failed: %w", err)
			}
			continue
		}

		// Apply exponential backoff for other retryable errors
		r.logger.Info("Retryable error, backing off",
			"operation", operation,
			"attempt", attempt,
			"backoff", backoff,
			"error", err)

		select {
		case <-ctx.Done():
			return fmt.Errorf("context cancelled during retry: %w", ctx.Err())
		case <-time.After(backoff):
			// Calculate next backoff
			backoff = min(time.Duration(float64(backoff)*r.config.BackoffMultiple), r.config.MaxBackoff)
		}
	}

	return fmt.Errorf("operation %s failed after %d attempts: %w",
		operation, r.config.MaxAttempts, lastErr)
}

// DoWithRetryFunc executes a function that returns a value with retry logic
func DoWithRetry[T any](
	ctx context.Context,
	retryer *Retryer,
	operation string,
	fn func(ctx context.Context) (T, error),
) (T, error) {
	var result T
	var lastErr error

	err := retryer.Do(ctx, operation, func(ctx context.Context) error {
		var err error
		result, err = fn(ctx)
		if err != nil {
			lastErr = err
			return err
		}
		return nil
	})

	if err != nil {
		return result, lastErr
	}
	return result, nil
}

// CircuitBreaker implements a circuit breaker pattern
type CircuitBreaker struct {
	maxFailures  int
	resetTimeout time.Duration

	failures     int
	lastFailTime time.Time
	state        CircuitState
	logger       *slog.Logger
}

// CircuitState represents the state of a circuit breaker
type CircuitState int

const (
	// StateClosed means requests are allowed
	StateClosed CircuitState = iota
	// StateOpen means requests are blocked
	StateOpen
	// StateHalfOpen means we're testing if the service recovered
	StateHalfOpen
)

// NewCircuitBreaker creates a new circuit breaker
func NewCircuitBreaker(maxFailures int, resetTimeout time.Duration, logger *slog.Logger) *CircuitBreaker {
	return &CircuitBreaker{
		maxFailures:  maxFailures,
		resetTimeout: resetTimeout,
		state:        StateClosed,
		logger:       logger,
	}
}

// Call executes a function through the circuit breaker
func (cb *CircuitBreaker) Call(ctx context.Context, fn RetryFunc) error {
	if !cb.AllowRequest() {
		return fmt.Errorf("circuit breaker is open")
	}

	err := fn(ctx)
	if err != nil {
		cb.RecordFailure()
		return err
	}

	cb.RecordSuccess()
	return nil
}

// AllowRequest checks if a request should be allowed
func (cb *CircuitBreaker) AllowRequest() bool {
	switch cb.state {
	case StateClosed:
		return true
	case StateOpen:
		// Check if we should transition to half-open
		if time.Since(cb.lastFailTime) > cb.resetTimeout {
			cb.logger.Info("Circuit breaker transitioning to half-open state")
			cb.state = StateHalfOpen
			return true
		}
		return false
	case StateHalfOpen:
		return true
	}
	return false
}

// RecordSuccess records a successful request
func (cb *CircuitBreaker) RecordSuccess() {
	if cb.state == StateHalfOpen {
		cb.logger.Info("Circuit breaker recovered, closing circuit")
	}
	cb.failures = 0
	cb.state = StateClosed
}

// RecordFailure records a failed request
func (cb *CircuitBreaker) RecordFailure() {
	cb.failures++
	cb.lastFailTime = time.Now()

	if cb.failures >= cb.maxFailures {
		cb.logger.Warn("Circuit breaker opened due to excessive failures",
			"failures", cb.failures,
			"max_failures", cb.maxFailures)
		cb.state = StateOpen
	}
}

// GetState returns the current circuit breaker state
func (cb *CircuitBreaker) GetState() CircuitState {
	return cb.state
}

// Reset resets the circuit breaker to closed state
func (cb *CircuitBreaker) Reset() {
	cb.failures = 0
	cb.state = StateClosed
	cb.logger.Info("Circuit breaker manually reset")
}
