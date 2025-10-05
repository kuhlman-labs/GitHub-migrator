package api

import (
	"log/slog"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"github.com/brettkuhlman/github-migrator/internal/config"
	"github.com/brettkuhlman/github-migrator/internal/storage"
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
		os.RemoveAll(tmpDir)
		t.Fatal(err)
	}

	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))

	server := NewServer(cfg, db, logger)

	cleanup := func() {
		db.Close()
		os.RemoveAll(tmpDir)
	}

	return server, cleanup
}

func TestNewServer(t *testing.T) {
	server, cleanup := setupTestServer(t)
	defer cleanup()

	if server == nil {
		t.Fatal("NewServer() returned nil")
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

	server.handleHealth(w, req)

	resp := w.Result()
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("handleHealth() status = %d, want %d", resp.StatusCode, http.StatusOK)
	}

	contentType := resp.Header.Get("Content-Type")
	if contentType != "application/json" {
		t.Errorf("handleHealth() Content-Type = %s, want application/json", contentType)
	}

	body := w.Body.String()
	expectedBody := `{"status":"healthy"}`
	if body != expectedBody {
		t.Errorf("handleHealth() body = %s, want %s", body, expectedBody)
	}
}

func TestServer_HandleRepositories(t *testing.T) {
	server, cleanup := setupTestServer(t)
	defer cleanup()

	req := httptest.NewRequest(http.MethodGet, "/api/v1/repositories", nil)
	w := httptest.NewRecorder()

	server.handleRepositories(w, req)

	resp := w.Result()
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("handleRepositories() status = %d, want %d", resp.StatusCode, http.StatusOK)
	}

	contentType := resp.Header.Get("Content-Type")
	if contentType != "application/json" {
		t.Errorf("handleRepositories() Content-Type = %s, want application/json", contentType)
	}

	body := w.Body.String()
	expectedBody := `{"repositories":[]}`
	if body != expectedBody {
		t.Errorf("handleRepositories() body = %s, want %s", body, expectedBody)
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
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("GET /health status = %d, want %d", resp.StatusCode, http.StatusOK)
	}

	// Test repositories endpoint
	resp, err = http.Get(ts.URL + "/api/v1/repositories")
	if err != nil {
		t.Fatalf("GET /api/v1/repositories error = %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("GET /api/v1/repositories status = %d, want %d", resp.StatusCode, http.StatusOK)
	}
}
