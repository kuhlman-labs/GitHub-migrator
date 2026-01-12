package api

import (
	"log/slog"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"github.com/kuhlman-labs/github-migrator/internal/config"
	"github.com/kuhlman-labs/github-migrator/internal/storage"
)

func setupTestServer(t *testing.T) (*Server, func()) {
	t.Helper()

	// Create temporary directory for test database
	tmpDir, err := os.MkdirTemp("", "api-test-*")
	if err != nil {
		t.Fatal(err)
	}

	cfg := &config.Config{
		Server: config.ServerConfig{
			Port: 8080,
		},
		Database: config.DatabaseConfig{
			Type: "sqlite",
			DSN:  filepath.Join(tmpDir, "test.db"),
		},
	}

	db, err := storage.NewDatabase(cfg.Database)
	if err != nil {
		_ = os.RemoveAll(tmpDir)
		t.Fatal(err)
	}

	// Run migrations
	if err := db.Migrate(); err != nil {
		_ = db.Close()
		_ = os.RemoveAll(tmpDir)
		t.Fatal(err)
	}

	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelError}))

	server := NewServer(cfg, db, logger, nil, nil)

	cleanup := func() {
		_ = db.Close()
		_ = os.RemoveAll(tmpDir)
	}

	return server, cleanup
}

func TestNewServer(t *testing.T) {
	server, cleanup := setupTestServer(t)
	defer cleanup()

	if server == nil {
		t.Fatal("NewServer() returned nil")
		return // Prevent staticcheck SA5011
	}

	if server.config == nil {
		t.Error("server.config is nil")
	}

	if server.db == nil {
		t.Error("server.db is nil")
	}

	if server.logger == nil {
		t.Error("server.logger is nil")
	}
}

func TestServer_Router(t *testing.T) {
	server, cleanup := setupTestServer(t)
	defer cleanup()

	router := server.Router()
	if router == nil {
		t.Fatal("Router() returned nil")
	}
}

func TestServer_HandleHealth(t *testing.T) {
	server, cleanup := setupTestServer(t)
	defer cleanup()

	req := httptest.NewRequest(http.MethodGet, "/health", nil)
	w := httptest.NewRecorder()

	server.handler.Health(w, req)

	resp := w.Result()
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("Health() status = %d, want %d", resp.StatusCode, http.StatusOK)
	}

	contentType := resp.Header.Get("Content-Type")
	if contentType != "application/json" {
		t.Errorf("Health() Content-Type = %s, want application/json", contentType)
	}
}

func TestServer_HandleRepositories(t *testing.T) {
	server, cleanup := setupTestServer(t)
	defer cleanup()

	req := httptest.NewRequest(http.MethodGet, "/api/v1/repositories", nil)
	w := httptest.NewRecorder()

	server.handler.ListRepositories(w, req)

	resp := w.Result()
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("ListRepositories() status = %d, want %d", resp.StatusCode, http.StatusOK)
	}

	contentType := resp.Header.Get("Content-Type")
	if contentType != "application/json" {
		t.Errorf("ListRepositories() Content-Type = %s, want application/json", contentType)
	}
}

func TestServer_Integration(t *testing.T) {
	server, cleanup := setupTestServer(t)
	defer cleanup()

	// Start test server
	ts := httptest.NewServer(server.Router())
	defer ts.Close()

	// Test health endpoint
	resp, err := http.Get(ts.URL + "/health")
	if err != nil {
		t.Fatalf("GET /health error = %v", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("GET /health status = %d, want %d", resp.StatusCode, http.StatusOK)
	}

	// Test repositories endpoint
	resp, err = http.Get(ts.URL + "/api/v1/repositories")
	if err != nil {
		t.Fatalf("GET /api/v1/repositories error = %v", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("GET /api/v1/repositories status = %d, want %d", resp.StatusCode, http.StatusOK)
	}
}
