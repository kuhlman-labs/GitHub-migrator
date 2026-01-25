package copilot

import (
	"context"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"
)

func TestNewLicenseValidator(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelError}))

	tests := []struct {
		name      string
		baseURL   string
		wantURL   string
	}{
		{
			name:    "default URL",
			baseURL: "",
			wantURL: defaultGitHubAPIURL,
		},
		{
			name:    "custom URL",
			baseURL: "https://api.github.example.com",
			wantURL: "https://api.github.example.com",
		},
		{
			name:    "trailing slash removed",
			baseURL: "https://api.github.example.com/",
			wantURL: "https://api.github.example.com",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			v := NewLicenseValidator(tt.baseURL, logger)
			if v.baseURL != tt.wantURL {
				t.Errorf("baseURL = %v, want %v", v.baseURL, tt.wantURL)
			}
		})
	}
}

func TestLicenseCache(t *testing.T) {
	cache := newLicenseCache()

	t.Run("get non-existent key", func(t *testing.T) {
		_, ok := cache.get("nonexistent")
		if ok {
			t.Error("expected false for non-existent key")
		}
	})

	t.Run("set and get", func(t *testing.T) {
		status := &LicenseStatus{
			Valid:     true,
			HasSeat:   true,
			ExpiresAt: time.Now().Add(5 * time.Minute),
		}
		cache.set("testuser", status)

		got, ok := cache.get("testuser")
		if !ok {
			t.Error("expected true for existing key")
		}
		if !got.Valid {
			t.Error("expected Valid to be true")
		}
	})

	t.Run("expired entry", func(t *testing.T) {
		status := &LicenseStatus{
			Valid:     true,
			HasSeat:   true,
			ExpiresAt: time.Now().Add(-1 * time.Minute), // Already expired
		}
		cache.set("expireduser", status)

		_, ok := cache.get("expireduser")
		if ok {
			t.Error("expected false for expired entry")
		}
	})

	t.Run("invalidate", func(t *testing.T) {
		status := &LicenseStatus{
			Valid:     true,
			ExpiresAt: time.Now().Add(5 * time.Minute),
		}
		cache.set("toberemoved", status)

		cache.invalidate("toberemoved")

		_, ok := cache.get("toberemoved")
		if ok {
			t.Error("expected false after invalidation")
		}
	})
}

func TestCheckLicense(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelError}))

	t.Run("cache hit", func(t *testing.T) {
		v := NewLicenseValidator("", logger)

		// Pre-populate cache
		cachedStatus := &LicenseStatus{
			Valid:     true,
			HasSeat:   true,
			Message:   "Cached",
			ExpiresAt: time.Now().Add(5 * time.Minute),
		}
		v.cache.set("cacheduser", cachedStatus)

		status, err := v.CheckLicense(context.Background(), "cacheduser", "token")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if status.Message != "Cached" {
			t.Errorf("expected cached message, got %v", status.Message)
		}
	})

	t.Run("API call for uncached user", func(t *testing.T) {
		// Create a mock server
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			switch r.URL.Path {
			case "/user":
				w.WriteHeader(http.StatusOK)
				_, _ = w.Write([]byte(`{"id": 1, "login": "testuser"}`))
			case "/user/copilot_seat":
				w.WriteHeader(http.StatusOK)
				_, _ = w.Write([]byte(`{"seat_type": "assigned", "created_at": "2024-01-01T00:00:00Z"}`))
			default:
				w.WriteHeader(http.StatusNotFound)
			}
		}))
		defer server.Close()

		v := NewLicenseValidator(server.URL, logger)

		status, err := v.CheckLicense(context.Background(), "testuser", "testtoken")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if !status.Valid {
			t.Error("expected Valid to be true")
		}
		if !status.HasSeat {
			t.Error("expected HasSeat to be true")
		}
	})

	t.Run("no Copilot seat", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			switch r.URL.Path {
			case "/user":
				w.WriteHeader(http.StatusOK)
				_, _ = w.Write([]byte(`{"id": 1, "login": "testuser"}`))
			case "/user/copilot_seat":
				w.WriteHeader(http.StatusNotFound)
			default:
				w.WriteHeader(http.StatusNotFound)
			}
		}))
		defer server.Close()

		v := NewLicenseValidator(server.URL, logger)

		status, err := v.CheckLicense(context.Background(), "noseatuser", "testtoken")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if status.Valid {
			t.Error("expected Valid to be false for user without seat")
		}
		if status.HasSeat {
			t.Error("expected HasSeat to be false")
		}
	})
}

func TestCheckCLIAvailable(t *testing.T) {
	t.Run("empty path uses default copilot", func(t *testing.T) {
		// Empty path defaults to "copilot" in PATH
		// This may succeed or fail depending on the environment
		available, version, err := CheckCLIAvailable("")
		if available {
			// If copilot is in PATH, we should get a version
			t.Logf("copilot found in PATH, version: %s", version)
		} else {
			// If copilot is not in PATH, we should get an error
			t.Logf("copilot not found in PATH: %v", err)
		}
	})

	t.Run("non-existent path", func(t *testing.T) {
		available, _, err := CheckCLIAvailable("/nonexistent/path/to/copilot-cli-that-does-not-exist")
		if available {
			t.Error("expected not available with non-existent path")
		}
		if err == nil {
			t.Error("expected error for non-existent path")
		}
	})
}
