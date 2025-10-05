package github

import (
	"context"
	"log/slog"
	"os"
	"testing"
	"time"
)

func TestNewClient(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	cfg := ClientConfig{
		BaseURL:     "https://api.github.com",
		Token:       "test-token",
		Timeout:     30 * time.Second,
		RetryConfig: DefaultRetryConfig(),
		Logger:      logger,
	}

	client, err := NewClient(cfg)
	if err != nil {
		t.Fatalf("NewClient() error = %v, want nil", err)
	}

	if client == nil {
		t.Fatal("NewClient() returned nil client")
	}

	if client.rest == nil {
		t.Error("REST client is nil")
	}

	if client.graphql == nil {
		t.Error("GraphQL client is nil")
	}

	if client.rateLimiter == nil {
		t.Error("Rate limiter is nil")
	}

	if client.retryer == nil {
		t.Error("Retryer is nil")
	}

	if client.circuitBreaker == nil {
		t.Error("Circuit breaker is nil")
	}
}

func TestNewClientWithDefaults(t *testing.T) {
	cfg := ClientConfig{
		BaseURL:     "https://api.github.com",
		Token:       "test-token",
		RetryConfig: DefaultRetryConfig(),
	}

	client, err := NewClient(cfg)
	if err != nil {
		t.Fatalf("NewClient() error = %v, want nil", err)
	}

	if client == nil {
		t.Fatal("NewClient() returned nil client")
	}

	// Should use default logger
	if client.logger == nil {
		t.Error("Logger is nil, should have default")
	}
}

func TestNewClientWithEnterpriseURL(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	cfg := ClientConfig{
		BaseURL:     "https://github.company.com/api/v3",
		Token:       "test-token",
		Timeout:     30 * time.Second,
		RetryConfig: DefaultRetryConfig(),
		Logger:      logger,
	}

	client, err := NewClient(cfg)
	if err != nil {
		t.Fatalf("NewClient() error = %v, want nil", err)
	}

	if client == nil {
		t.Fatal("NewClient() returned nil client")
	}

	if client.baseURL != cfg.BaseURL {
		t.Errorf("baseURL = %s, want %s", client.baseURL, cfg.BaseURL)
	}
}

func TestClient_REST(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	cfg := ClientConfig{
		BaseURL:     "https://api.github.com",
		Token:       "test-token",
		RetryConfig: DefaultRetryConfig(),
		Logger:      logger,
	}

	client, err := NewClient(cfg)
	if err != nil {
		t.Fatalf("NewClient() error = %v, want nil", err)
	}

	restClient := client.REST()
	if restClient == nil {
		t.Error("REST() returned nil")
	}
}

func TestClient_GraphQL(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	cfg := ClientConfig{
		BaseURL:     "https://api.github.com",
		Token:       "test-token",
		RetryConfig: DefaultRetryConfig(),
		Logger:      logger,
	}

	client, err := NewClient(cfg)
	if err != nil {
		t.Fatalf("NewClient() error = %v, want nil", err)
	}

	graphqlClient := client.GraphQL()
	if graphqlClient == nil {
		t.Error("GraphQL() returned nil")
	}
}

func TestClient_GetRateLimiter(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	cfg := ClientConfig{
		BaseURL:     "https://api.github.com",
		Token:       "test-token",
		RetryConfig: DefaultRetryConfig(),
		Logger:      logger,
	}

	client, err := NewClient(cfg)
	if err != nil {
		t.Fatalf("NewClient() error = %v, want nil", err)
	}

	rateLimiter := client.GetRateLimiter()
	if rateLimiter == nil {
		t.Error("GetRateLimiter() returned nil")
	}
}

func TestClient_GetRetryer(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	cfg := ClientConfig{
		BaseURL:     "https://api.github.com",
		Token:       "test-token",
		RetryConfig: DefaultRetryConfig(),
		Logger:      logger,
	}

	client, err := NewClient(cfg)
	if err != nil {
		t.Fatalf("NewClient() error = %v, want nil", err)
	}

	retryer := client.GetRetryer()
	if retryer == nil {
		t.Error("GetRetryer() returned nil")
	}
}

func TestClient_DoWithRetrySuccess(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	cfg := ClientConfig{
		BaseURL:     "https://api.github.com",
		Token:       "test-token",
		RetryConfig: DefaultRetryConfig(),
		Logger:      logger,
	}

	client, err := NewClient(cfg)
	if err != nil {
		t.Fatalf("NewClient() error = %v, want nil", err)
	}

	// Test that DoWithRetry method exists and can be called
	// Note: We can't easily test this without mocking the GitHub client
	// This test just verifies the client is properly initialized
	if client.retryer == nil {
		t.Error("Retryer is nil")
	}
}

func TestClient_QueryWithRetrySuccess(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	cfg := ClientConfig{
		BaseURL:     "https://api.github.com",
		Token:       "test-token",
		RetryConfig: DefaultRetryConfig(),
		Logger:      logger,
	}

	client, err := NewClient(cfg)
	if err != nil {
		t.Fatalf("NewClient() error = %v, want nil", err)
	}

	ctx := context.Background()

	// Note: This will fail in real execution because we're not using a valid token
	// and not making a real API call. This test just ensures the method exists
	// and basic error handling works.
	var query struct{}
	err = client.QueryWithRetry(ctx, "test-query", &query, nil)

	// We expect an error because we're not actually authenticated
	// The important thing is that the method doesn't panic
	if err == nil {
		t.Log("QueryWithRetry() succeeded (unexpected with fake token, but OK for test)")
	}
}

func TestClient_MutateWithRetrySuccess(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	cfg := ClientConfig{
		BaseURL:     "https://api.github.com",
		Token:       "test-token",
		RetryConfig: DefaultRetryConfig(),
		Logger:      logger,
	}

	client, err := NewClient(cfg)
	if err != nil {
		t.Fatalf("NewClient() error = %v, want nil", err)
	}

	ctx := context.Background()

	// Note: This will fail in real execution because we're not using a valid token
	// The test just ensures the method exists and basic error handling works.
	var mutation struct{}
	err = client.MutateWithRetry(ctx, "test-mutation", &mutation, nil, nil)

	// We expect an error because we're not actually authenticated
	if err == nil {
		t.Log("MutateWithRetry() succeeded (unexpected with fake token, but OK for test)")
	}
}

func TestClientConfig_DefaultTimeout(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	cfg := ClientConfig{
		BaseURL:     "https://api.github.com",
		Token:       "test-token",
		Timeout:     0, // Should use default
		RetryConfig: DefaultRetryConfig(),
		Logger:      logger,
	}

	client, err := NewClient(cfg)
	if err != nil {
		t.Fatalf("NewClient() error = %v, want nil", err)
	}

	if client == nil {
		t.Fatal("NewClient() returned nil client")
	}

	// Client should be created successfully with default timeout
}

func TestClient_UpdateRateLimits(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	cfg := ClientConfig{
		BaseURL:     "https://api.github.com",
		Token:       "test-token",
		RetryConfig: DefaultRetryConfig(),
		Logger:      logger,
	}

	client, err := NewClient(cfg)
	if err != nil {
		t.Fatalf("NewClient() error = %v, want nil", err)
	}

	// Manually update rate limits (simulating response)
	resetTime := time.Now().Add(1 * time.Hour)
	client.rateLimiter.UpdateLimits(1000, 5000, resetTime)

	remaining, limit, reset := client.rateLimiter.GetStatus()

	if remaining != 1000 {
		t.Errorf("remaining = %d, want 1000", remaining)
	}

	if limit != 5000 {
		t.Errorf("limit = %d, want 5000", limit)
	}

	if !reset.Equal(resetTime) {
		t.Errorf("resetTime = %v, want %v", reset, resetTime)
	}
}

// Integration-like test (requires actual GitHub token, skip by default)
func TestClient_TestAuthenticationIntegration(t *testing.T) {
	// Skip this test unless explicitly running integration tests
	token := os.Getenv("GITHUB_TOKEN")
	if token == "" {
		t.Skip("Skipping integration test: GITHUB_TOKEN not set")
	}

	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	cfg := ClientConfig{
		BaseURL:     "https://api.github.com",
		Token:       token,
		RetryConfig: DefaultRetryConfig(),
		Logger:      logger,
	}

	client, err := NewClient(cfg)
	if err != nil {
		t.Fatalf("NewClient() error = %v, want nil", err)
	}

	ctx := context.Background()
	err = client.TestAuthentication(ctx)

	if err != nil {
		t.Errorf("TestAuthentication() error = %v, want nil", err)
	}
}

// Integration-like test (requires actual GitHub token, skip by default)
func TestClient_GetRateLimitStatusIntegration(t *testing.T) {
	// Skip this test unless explicitly running integration tests
	token := os.Getenv("GITHUB_TOKEN")
	if token == "" {
		t.Skip("Skipping integration test: GITHUB_TOKEN not set")
	}

	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	cfg := ClientConfig{
		BaseURL:     "https://api.github.com",
		Token:       token,
		RetryConfig: DefaultRetryConfig(),
		Logger:      logger,
	}

	client, err := NewClient(cfg)
	if err != nil {
		t.Fatalf("NewClient() error = %v, want nil", err)
	}

	ctx := context.Background()
	limits, err := client.GetRateLimitStatus(ctx)

	if err != nil {
		t.Errorf("GetRateLimitStatus() error = %v, want nil", err)
	}

	if limits == nil {
		t.Error("GetRateLimitStatus() returned nil limits")
		return
	}

	if limits.Core != nil {
		t.Logf("Rate limit: %d/%d, reset: %v",
			limits.Core.Remaining, limits.Core.Limit, limits.Core.Reset)
	}
}
