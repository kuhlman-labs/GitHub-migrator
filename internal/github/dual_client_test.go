package github

import (
	"log/slog"
	"os"
	"testing"
	"time"
)

func TestNewDualClient_PATOnly(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))

	cfg := DualClientConfig{
		PATConfig: ClientConfig{
			BaseURL:     "https://api.github.com",
			Token:       "ghp_test_token",
			Timeout:     30 * time.Second,
			RetryConfig: DefaultRetryConfig(),
			Logger:      logger,
		},
		Logger: logger,
	}

	dc, err := NewDualClient(cfg)
	if err != nil {
		t.Fatalf("NewDualClient() error = %v, want nil", err)
	}

	if dc == nil {
		t.Fatal("NewDualClient() returned nil")
	}

	if dc.HasAppClient() {
		t.Error("Expected no App client, but HasAppClient() = true")
	}

	// Both API and Migration clients should return the same PAT client
	if dc.APIClient() != dc.MigrationClient() {
		t.Error("Expected APIClient() and MigrationClient() to return same client when no App configured")
	}
}

func TestNewDualClient_RequiresPAT(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))

	cfg := DualClientConfig{
		PATConfig: ClientConfig{
			BaseURL: "https://api.github.com",
			Token:   "", // No PAT token
		},
		Logger: logger,
	}

	_, err := NewDualClient(cfg)
	if err == nil {
		t.Error("Expected error when PAT token is missing, got nil")
	}
}

func TestDualClient_MigrationClient(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))

	cfg := DualClientConfig{
		PATConfig: ClientConfig{
			BaseURL:     "https://api.github.com",
			Token:       "ghp_test_token",
			Timeout:     30 * time.Second,
			RetryConfig: DefaultRetryConfig(),
			Logger:      logger,
		},
		Logger: logger,
	}

	dc, err := NewDualClient(cfg)
	if err != nil {
		t.Fatalf("NewDualClient() error = %v", err)
	}

	migClient := dc.MigrationClient()
	if migClient == nil {
		t.Fatal("MigrationClient() returned nil")
	}

	// Verify it's the PAT client
	if migClient.Token() != "ghp_test_token" {
		t.Errorf("MigrationClient().Token() = %q, want %q", migClient.Token(), "ghp_test_token")
	}
}

func TestDualClient_BaseURL(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	testURL := "https://github.enterprise.com/api/v3"

	cfg := DualClientConfig{
		PATConfig: ClientConfig{
			BaseURL:     testURL,
			Token:       "ghp_test_token",
			Timeout:     30 * time.Second,
			RetryConfig: DefaultRetryConfig(),
			Logger:      logger,
		},
		Logger: logger,
	}

	dc, err := NewDualClient(cfg)
	if err != nil {
		t.Fatalf("NewDualClient() error = %v", err)
	}

	if dc.BaseURL() != testURL {
		t.Errorf("BaseURL() = %q, want %q", dc.BaseURL(), testURL)
	}
}
