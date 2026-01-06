package github

import (
	"context"
	"errors"
	"log/slog"
	"os"
	"testing"
	"time"
)

func TestDefaultRetryConfig(t *testing.T) {
	config := DefaultRetryConfig()

	// MaxAttempts is 5 to allow recovery from secondary rate limits
	if config.MaxAttempts != 5 {
		t.Errorf("MaxAttempts = %d, want 5", config.MaxAttempts)
	}

	if config.InitialBackoff != 1*time.Second {
		t.Errorf("InitialBackoff = %v, want 1s", config.InitialBackoff)
	}

	if config.MaxBackoff != 30*time.Second {
		t.Errorf("MaxBackoff = %v, want 30s", config.MaxBackoff)
	}

	if config.BackoffMultiple != 2.0 {
		t.Errorf("BackoffMultiple = %f, want 2.0", config.BackoffMultiple)
	}
}

func TestNewRetryer(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	rateLimiter := NewRateLimiter(logger)
	config := DefaultRetryConfig()

	retryer := NewRetryer(config, rateLimiter, logger)

	if retryer == nil {
		t.Fatal("NewRetryer() returned nil")
		return // Prevent staticcheck SA5011
	}

	if retryer.config.MaxAttempts != config.MaxAttempts {
		t.Errorf("config.MaxAttempts = %d, want %d", retryer.config.MaxAttempts, config.MaxAttempts)
	}
}

func TestRetryer_DoSuccess(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	rateLimiter := NewRateLimiter(logger)
	config := DefaultRetryConfig()
	retryer := NewRetryer(config, rateLimiter, logger)

	ctx := context.Background()
	callCount := 0

	err := retryer.Do(ctx, "test-operation", func(ctx context.Context) error {
		callCount++
		return nil
	})

	if err != nil {
		t.Errorf("Do() error = %v, want nil", err)
	}

	if callCount != 1 {
		t.Errorf("function called %d times, want 1", callCount)
	}
}

func TestRetryer_DoWithRetryableError(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	rateLimiter := NewRateLimiter(logger)
	config := RetryConfig{
		MaxAttempts:     3,
		InitialBackoff:  10 * time.Millisecond,
		MaxBackoff:      100 * time.Millisecond,
		BackoffMultiple: 2.0,
	}
	retryer := NewRetryer(config, rateLimiter, logger)

	ctx := context.Background()
	callCount := 0
	retryableErr := &APIError{StatusCode: 500, Err: ErrServerError}

	err := retryer.Do(ctx, "test-operation", func(ctx context.Context) error {
		callCount++
		if callCount < 3 {
			return retryableErr
		}
		return nil
	})

	if err != nil {
		t.Errorf("Do() error = %v, want nil", err)
	}

	if callCount != 3 {
		t.Errorf("function called %d times, want 3", callCount)
	}
}

func TestRetryer_DoWithNonRetryableError(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	rateLimiter := NewRateLimiter(logger)
	config := DefaultRetryConfig()
	retryer := NewRetryer(config, rateLimiter, logger)

	ctx := context.Background()
	callCount := 0
	nonRetryableErr := &APIError{StatusCode: 404, Err: ErrNotFound}

	err := retryer.Do(ctx, "test-operation", func(ctx context.Context) error {
		callCount++
		return nonRetryableErr
	})

	if err == nil {
		t.Error("Do() error = nil, want error")
	}

	if callCount != 1 {
		t.Errorf("function called %d times, want 1 (should not retry)", callCount)
	}
}

func TestRetryer_DoMaxAttemptsExceeded(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	rateLimiter := NewRateLimiter(logger)
	config := RetryConfig{
		MaxAttempts:     3,
		InitialBackoff:  10 * time.Millisecond,
		MaxBackoff:      100 * time.Millisecond,
		BackoffMultiple: 2.0,
	}
	retryer := NewRetryer(config, rateLimiter, logger)

	ctx := context.Background()
	callCount := 0
	retryableErr := &APIError{StatusCode: 500, Err: ErrServerError}

	err := retryer.Do(ctx, "test-operation", func(ctx context.Context) error {
		callCount++
		return retryableErr
	})

	if err == nil {
		t.Error("Do() error = nil, want error after max attempts")
	}

	if callCount != 3 {
		t.Errorf("function called %d times, want 3", callCount)
	}
}

func TestRetryer_DoWithCancelledContext(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	rateLimiter := NewRateLimiter(logger)
	config := RetryConfig{
		MaxAttempts:     3,
		InitialBackoff:  1 * time.Second,
		MaxBackoff:      5 * time.Second,
		BackoffMultiple: 2.0,
	}
	retryer := NewRetryer(config, rateLimiter, logger)

	ctx, cancel := context.WithCancel(context.Background())
	callCount := 0
	retryableErr := &APIError{StatusCode: 500, Err: ErrServerError}

	// Cancel context after first call
	err := retryer.Do(ctx, "test-operation", func(ctx context.Context) error {
		callCount++
		if callCount == 1 {
			cancel()
		}
		return retryableErr
	})

	if err == nil {
		t.Error("Do() error = nil, want error")
	}

	// Should fail during backoff wait
	if callCount > 2 {
		t.Errorf("function called %d times, expected 1-2 due to context cancellation", callCount)
	}
}

func TestDoWithRetry(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	rateLimiter := NewRateLimiter(logger)
	config := DefaultRetryConfig()
	retryer := NewRetryer(config, rateLimiter, logger)

	ctx := context.Background()
	expectedResult := "success"

	result, err := DoWithRetry(ctx, retryer, "test-operation", func(ctx context.Context) (string, error) {
		return expectedResult, nil
	})

	if err != nil {
		t.Errorf("DoWithRetry() error = %v, want nil", err)
	}

	if result != expectedResult {
		t.Errorf("DoWithRetry() result = %s, want %s", result, expectedResult)
	}
}

func TestDoWithRetryError(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	rateLimiter := NewRateLimiter(logger)
	config := DefaultRetryConfig()
	retryer := NewRetryer(config, rateLimiter, logger)

	ctx := context.Background()
	expectedErr := errors.New("test error")

	result, err := DoWithRetry(ctx, retryer, "test-operation", func(ctx context.Context) (string, error) {
		return "", expectedErr
	})

	if err == nil {
		t.Error("DoWithRetry() error = nil, want error")
	}

	if result != "" {
		t.Errorf("DoWithRetry() result = %s, want empty string", result)
	}
}

func TestNewCircuitBreaker(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	cb := NewCircuitBreaker(5, 1*time.Minute, logger)

	if cb == nil {
		t.Fatal("NewCircuitBreaker() returned nil")
		return // Prevent staticcheck SA5011
	}

	if cb.maxFailures != 5 {
		t.Errorf("maxFailures = %d, want 5", cb.maxFailures)
	}

	if cb.state != StateClosed {
		t.Errorf("initial state = %v, want StateClosed", cb.state)
	}
}

func TestCircuitBreaker_AllowRequest(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	cb := NewCircuitBreaker(3, 100*time.Millisecond, logger)

	// Closed state should allow requests
	if !cb.AllowRequest() {
		t.Error("AllowRequest() = false, want true for closed circuit")
	}

	// Record failures to open circuit
	for range 3 {
		cb.RecordFailure()
	}

	// Open state should block requests
	if cb.AllowRequest() {
		t.Error("AllowRequest() = true, want false for open circuit")
	}

	// Wait for reset timeout
	time.Sleep(150 * time.Millisecond)

	// Should transition to half-open and allow request
	if !cb.AllowRequest() {
		t.Error("AllowRequest() = false, want true after reset timeout")
	}

	if cb.state != StateHalfOpen {
		t.Errorf("state = %v, want StateHalfOpen after timeout", cb.state)
	}
}

func TestCircuitBreaker_RecordSuccess(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	cb := NewCircuitBreaker(3, 1*time.Minute, logger)

	// Record some failures
	cb.RecordFailure()
	cb.RecordFailure()

	if cb.failures != 2 {
		t.Errorf("failures = %d, want 2", cb.failures)
	}

	// Record success should reset failures
	cb.RecordSuccess()

	if cb.failures != 0 {
		t.Errorf("failures = %d, want 0 after success", cb.failures)
	}

	if cb.state != StateClosed {
		t.Errorf("state = %v, want StateClosed after success", cb.state)
	}
}

func TestCircuitBreaker_RecordFailure(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	cb := NewCircuitBreaker(3, 1*time.Minute, logger)

	// Record failures
	cb.RecordFailure()
	if cb.state != StateClosed {
		t.Errorf("state = %v, want StateClosed after 1 failure", cb.state)
	}

	cb.RecordFailure()
	if cb.state != StateClosed {
		t.Errorf("state = %v, want StateClosed after 2 failures", cb.state)
	}

	cb.RecordFailure()
	if cb.state != StateOpen {
		t.Errorf("state = %v, want StateOpen after 3 failures", cb.state)
	}
}

func TestCircuitBreaker_Call(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	cb := NewCircuitBreaker(3, 100*time.Millisecond, logger)
	ctx := context.Background()

	// Successful call
	err := cb.Call(ctx, func(ctx context.Context) error {
		return nil
	})

	if err != nil {
		t.Errorf("Call() error = %v, want nil", err)
	}

	// Failed call
	expectedErr := errors.New("test error")
	err = cb.Call(ctx, func(ctx context.Context) error {
		return expectedErr
	})

	if err == nil {
		t.Error("Call() error = nil, want error")
	}

	if cb.failures != 1 {
		t.Errorf("failures = %d, want 1", cb.failures)
	}
}

func TestCircuitBreaker_Reset(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	cb := NewCircuitBreaker(3, 1*time.Minute, logger)

	// Open the circuit
	for range 3 {
		cb.RecordFailure()
	}

	if cb.state != StateOpen {
		t.Errorf("state = %v, want StateOpen", cb.state)
	}

	// Reset the circuit
	cb.Reset()

	if cb.state != StateClosed {
		t.Errorf("state = %v, want StateClosed after reset", cb.state)
	}

	if cb.failures != 0 {
		t.Errorf("failures = %d, want 0 after reset", cb.failures)
	}
}

func TestCircuitBreaker_GetState(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	cb := NewCircuitBreaker(3, 1*time.Minute, logger)

	if state := cb.GetState(); state != StateClosed {
		t.Errorf("GetState() = %v, want StateClosed", state)
	}

	// Open the circuit
	for range 3 {
		cb.RecordFailure()
	}

	if state := cb.GetState(); state != StateOpen {
		t.Errorf("GetState() = %v, want StateOpen", state)
	}
}

func TestRetryer_CalculateRateLimitWait(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	rateLimiter := NewRateLimiter(logger)
	config := DefaultRetryConfig()
	retryer := NewRetryer(config, rateLimiter, logger)

	tests := []struct {
		name        string
		err         error
		minDuration time.Duration
		maxDuration time.Duration
	}{
		{
			name:        "blocked rate limit with reset time",
			err:         errors.New("403 API rate limit exceeded [rate reset in 30s]"),
			minDuration: 30 * time.Second, // Should be at least 30s
			maxDuration: 40 * time.Second, // Plus buffer, shouldn't be too much more
		},
		{
			name:        "blocked rate limit with 2m reset time",
			err:         errors.New("403 API rate limit exceeded [rate reset in 2m]"),
			minDuration: 2 * time.Minute,
			maxDuration: 2*time.Minute + 10*time.Second,
		},
		{
			name:        "error without parseable reset time",
			err:         errors.New("rate limit exceeded"),
			minDuration: SecondaryRateLimitBackoff, // Should use default
			maxDuration: SecondaryRateLimitBackoff + time.Second,
		},
		{
			name:        "very short reset time enforces minimum",
			err:         errors.New("rate limit [rate reset in 1s]"),
			minDuration: MinRateLimitWait, // Should be at least minimum
			maxDuration: MinRateLimitWait + time.Second,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			duration := retryer.calculateRateLimitWait(tt.err)
			if duration < tt.minDuration {
				t.Errorf("calculateRateLimitWait() = %v, want >= %v", duration, tt.minDuration)
			}
			if duration > tt.maxDuration {
				t.Errorf("calculateRateLimitWait() = %v, want <= %v", duration, tt.maxDuration)
			}
		})
	}
}

func TestRetryer_DoWithBlockedRateLimitError(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	rateLimiter := NewRateLimiter(logger)
	config := RetryConfig{
		MaxAttempts:     3,
		InitialBackoff:  10 * time.Millisecond,
		MaxBackoff:      100 * time.Millisecond,
		BackoffMultiple: 2.0,
	}
	retryer := NewRetryer(config, rateLimiter, logger)

	ctx := context.Background()
	callCount := 0
	blockedErr := errors.New("GET https://api.github.com/orgs/test/repos: 403 API rate limit of 5000 still exceeded until 2026-01-06, not making remote request. [rate reset in 1s]")

	// Use a context with timeout longer than MinRateLimitWait (10s) + buffer
	ctx, cancel := context.WithTimeout(ctx, 15*time.Second)
	defer cancel()

	start := time.Now()
	err := retryer.Do(ctx, "test-operation", func(ctx context.Context) error {
		callCount++
		if callCount < 2 {
			return blockedErr
		}
		return nil
	})
	elapsed := time.Since(start)

	if err != nil {
		t.Errorf("Do() error = %v, want nil after retry", err)
	}

	if callCount < 2 {
		t.Errorf("function called %d times, expected at least 2 (initial + retry)", callCount)
	}

	// Should have waited at least MinRateLimitWait (10s)
	if elapsed < 10*time.Second {
		t.Errorf("Do() completed in %v, expected at least 10s wait for rate limit", elapsed)
	}
}

func TestRetryer_DoWithSecondaryRateLimitError(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	rateLimiter := NewRateLimiter(logger)
	config := RetryConfig{
		MaxAttempts:     3,
		InitialBackoff:  10 * time.Millisecond,
		MaxBackoff:      100 * time.Millisecond,
		BackoffMultiple: 2.0,
	}
	retryer := NewRetryer(config, rateLimiter, logger)

	ctx := context.Background()
	callCount := 0
	secondaryErr := errors.New("You have exceeded a secondary rate limit. Please wait a few minutes before you try again.")

	// Use a short timeout - we're testing that it handles the error correctly,
	// not that it waits the full 60 seconds
	ctx, cancel := context.WithTimeout(ctx, 100*time.Millisecond)
	defer cancel()

	err := retryer.Do(ctx, "test-operation", func(ctx context.Context) error {
		callCount++
		return secondaryErr
	})

	// Should timeout because secondary rate limit wait is 60s
	if err == nil {
		t.Error("Do() error = nil, expected context timeout")
	}

	// Should have been called once before waiting
	if callCount != 1 {
		t.Errorf("function called %d times, expected 1", callCount)
	}
}

func TestRetryer_RateLimitConstants(t *testing.T) {
	// Verify constants have sensible values
	if RateLimitResetBuffer < 1*time.Second {
		t.Errorf("RateLimitResetBuffer = %v, should be at least 1 second", RateLimitResetBuffer)
	}

	if MinRateLimitWait < 5*time.Second {
		t.Errorf("MinRateLimitWait = %v, should be at least 5 seconds", MinRateLimitWait)
	}

	if MaxRateLimitWait < 5*time.Minute {
		t.Errorf("MaxRateLimitWait = %v, should be at least 5 minutes", MaxRateLimitWait)
	}

	if SecondaryRateLimitBackoff < 30*time.Second {
		t.Errorf("SecondaryRateLimitBackoff = %v, should be at least 30 seconds", SecondaryRateLimitBackoff)
	}
}
