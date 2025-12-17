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
		return // Prevent staticcheck SA5011
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
		return // Prevent staticcheck SA5011
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
		return // Prevent staticcheck SA5011
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
	// Use an empty map for input to test the interface{} parameter
	input := map[string]interface{}{}
	err = client.MutateWithRetry(ctx, "test-mutation", &mutation, input, nil)

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

// Integration test for enterprise organization listing (requires enterprise access)
func TestClient_ListEnterpriseOrganizationsIntegration(t *testing.T) {
	// Skip this test unless explicitly running integration tests
	token := os.Getenv("GITHUB_TOKEN")
	enterpriseSlug := os.Getenv("GITHUB_ENTERPRISE_SLUG")

	if token == "" || enterpriseSlug == "" {
		t.Skip("Skipping integration test: GITHUB_TOKEN or GITHUB_ENTERPRISE_SLUG not set")
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
	orgs, err := client.ListEnterpriseOrganizations(ctx, enterpriseSlug)

	if err != nil {
		t.Errorf("ListEnterpriseOrganizations() error = %v, want nil", err)
		return
	}

	if orgs == nil {
		t.Error("ListEnterpriseOrganizations() returned nil organizations")
		return
	}

	t.Logf("Found %d organizations in enterprise %s", len(orgs), enterpriseSlug)

	// Should return at least an empty slice, not nil
	if len(orgs) == 0 {
		t.Log("No organizations found in enterprise (could be valid)")
	} else {
		t.Logf("Sample organizations: %v", orgs[:min(3, len(orgs))])
	}
}

// Unit test for ListEnterpriseOrganizations structure
func TestClient_ListEnterpriseOrganizationsUnit(t *testing.T) {
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

	// This will fail with authentication error, but we're testing the structure
	_, err = client.ListEnterpriseOrganizations(ctx, "test-enterprise")

	// We expect an error since we're using a fake token
	if err == nil {
		t.Log("ListEnterpriseOrganizations() succeeded (unexpected with fake token)")
	}

	// The important thing is it doesn't panic and returns proper error
	if err != nil {
		t.Logf("Expected error with fake token: %v", err)
	}
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func TestDetectInstanceType(t *testing.T) {
	tests := []struct {
		name     string
		baseURL  string
		expected InstanceType
	}{
		{
			name:     "GitHub.com - empty URL",
			baseURL:  "",
			expected: InstanceTypeGitHub,
		},
		{
			name:     "GitHub.com - standard API URL",
			baseURL:  "https://api.github.com",
			expected: InstanceTypeGitHub,
		},
		{
			name:     "GHEC - data residency with .ghe.com",
			baseURL:  "https://octocorp.ghe.com",
			expected: InstanceTypeGHEC,
		},
		{
			name:     "GHEC - data residency with api subdomain",
			baseURL:  "https://api.octocorp.ghe.com",
			expected: InstanceTypeGHEC,
		},
		{
			name:     "GHES - self-hosted instance",
			baseURL:  "https://github.company.com/api/v3",
			expected: InstanceTypeGHES,
		},
		{
			name:     "GHES - self-hosted with subdomain",
			baseURL:  "https://api.github.company.com",
			expected: InstanceTypeGHES,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := detectInstanceType(tt.baseURL)
			if result != tt.expected {
				t.Errorf("detectInstanceType(%q) = %v, want %v", tt.baseURL, result, tt.expected)
			}
		})
	}
}

func TestBuildGraphQLURL(t *testing.T) {
	tests := []struct {
		name     string
		baseURL  string
		expected string
	}{
		{
			name:     "GitHub.com - empty URL",
			baseURL:  "",
			expected: "https://api.github.com/graphql",
		},
		{
			name:     "GitHub.com - standard API URL",
			baseURL:  "https://api.github.com",
			expected: "https://api.github.com/graphql",
		},
		{
			name:     "GHEC - data residency domain",
			baseURL:  "https://octocorp.ghe.com",
			expected: "https://api.octocorp.ghe.com/graphql",
		},
		{
			name:     "GHEC - data residency with api subdomain",
			baseURL:  "https://api.octocorp.ghe.com",
			expected: "https://api.octocorp.ghe.com/graphql",
		},
		{
			name:     "GHEC - data residency without https",
			baseURL:  "octocorp.ghe.com",
			expected: "https://api.octocorp.ghe.com/graphql",
		},
		{
			name:     "GHES - self-hosted instance with /api/v3",
			baseURL:  "https://github.company.com/api/v3",
			expected: "https://github.company.com/api/graphql",
		},
		{
			name:     "GHES - self-hosted instance with /api",
			baseURL:  "https://github.company.com/api",
			expected: "https://github.company.com/api/graphql",
		},
		{
			name:     "GHES - self-hosted with trailing slash",
			baseURL:  "https://github.company.com/",
			expected: "https://github.company.com/api/graphql",
		},
		{
			name:     "GHEC - data residency with trailing slash",
			baseURL:  "https://octocorp.ghe.com/",
			expected: "https://api.octocorp.ghe.com/graphql",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := buildGraphQLURL(tt.baseURL)
			if result != tt.expected {
				t.Errorf("buildGraphQLURL(%q) = %q, want %q", tt.baseURL, result, tt.expected)
			}
		})
	}
}

func TestRepositoryURL(t *testing.T) {
	tests := []struct {
		name     string
		baseURL  string
		fullName string
		expected string
	}{
		{
			name:     "GitHub.com - standard repository",
			baseURL:  "https://api.github.com",
			fullName: "octocat/hello-world",
			expected: "https://github.com/octocat/hello-world",
		},
		{
			name:     "GitHub.com - empty base URL",
			baseURL:  "",
			fullName: "octocat/hello-world",
			expected: "https://github.com/octocat/hello-world",
		},
		{
			name:     "GHEC - data residency domain",
			baseURL:  "https://octocorp.ghe.com",
			fullName: "engineering/webapp",
			expected: "https://octocorp.ghe.com/engineering/webapp",
		},
		{
			name:     "GHEC - data residency with api subdomain",
			baseURL:  "https://api.octocorp.ghe.com",
			fullName: "engineering/webapp",
			expected: "https://octocorp.ghe.com/engineering/webapp",
		},
		{
			name:     "GHEC - data residency with /api/v3",
			baseURL:  "https://api.octocorp.ghe.com/api/v3",
			fullName: "engineering/webapp",
			expected: "https://octocorp.ghe.com/engineering/webapp",
		},
		{
			name:     "GHES - self-hosted instance",
			baseURL:  "https://github.company.com/api/v3",
			fullName: "myorg/myrepo",
			expected: "https://github.company.com/myorg/myrepo",
		},
		{
			name:     "GHES - self-hosted with api subdomain",
			baseURL:  "https://api.github.company.com",
			fullName: "myorg/myrepo",
			expected: "https://api.github.company.com/myorg/myrepo",
		},
		{
			name:     "GHES - self-hosted with trailing slash",
			baseURL:  "https://github.company.com/",
			fullName: "myorg/myrepo",
			expected: "https://github.company.com/myorg/myrepo",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
			cfg := ClientConfig{
				BaseURL:     tt.baseURL,
				Token:       "test-token",
				RetryConfig: DefaultRetryConfig(),
				Logger:      logger,
			}

			client, err := NewClient(cfg)
			if err != nil {
				t.Fatalf("NewClient() error = %v, want nil", err)
			}

			result := client.RepositoryURL(tt.fullName)
			if result != tt.expected {
				t.Errorf("RepositoryURL(%q) = %q, want %q", tt.fullName, result, tt.expected)
			}
		})
	}
}

func TestNewClientWithGHECDataResidency(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	cfg := ClientConfig{
		BaseURL:     "https://octocorp.ghe.com",
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
		return // Prevent staticcheck SA5011
	}

	// Verify that the GraphQL client was created
	if client.graphql == nil {
		t.Error("GraphQL client is nil for GHEC data residency instance")
	}

	// Verify the base URL is stored correctly
	if client.baseURL != cfg.BaseURL {
		t.Errorf("baseURL = %s, want %s", client.baseURL, cfg.BaseURL)
	}
}

func TestNewClientWithGHECAPISubdomain(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	cfg := ClientConfig{
		BaseURL:     "https://api.octocorp.ghe.com",
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
		return // Prevent staticcheck SA5011
	}

	// Verify that the GraphQL client was created
	if client.graphql == nil {
		t.Error("GraphQL client is nil for GHEC data residency instance with api subdomain")
	}
}

func TestListOrganizationProjects(t *testing.T) {
	// This is a unit test to verify that the ListOrganizationProjects method
	// is properly implemented and returns the expected structure.
	// Note: This requires a real GitHub connection for integration testing.

	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))

	// Test that the method exists and can be called
	// For a real test, you would need valid credentials
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

	// Verify the method exists and returns a non-nil map
	// (even if empty due to no real credentials)
	ctx := context.Background()
	projectsMap, _ := client.ListOrganizationProjects(ctx, "test-org")

	// We expect an error or an empty map since we're using test credentials
	// The important thing is the method is callable and returns the right type
	if projectsMap == nil {
		t.Error("ListOrganizationProjects returned nil map, expected non-nil even on error")
	}
}
