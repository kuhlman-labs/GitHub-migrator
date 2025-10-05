package logging

import (
	"bytes"
	"context"
	"log/slog"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/brettkuhlman/github-migrator/internal/config"
)

func TestParseLevel(t *testing.T) {
	tests := []struct {
		name     string
		level    string
		expected slog.Level
	}{
		{"debug level", "debug", slog.LevelDebug},
		{"info level", "info", slog.LevelInfo},
		{"warn level", "warn", slog.LevelWarn},
		{"error level", "error", slog.LevelError},
		{"default level", "invalid", slog.LevelInfo},
		{"empty level", "", slog.LevelInfo},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := parseLevel(tt.level)
			if got != tt.expected {
				t.Errorf("parseLevel(%s) = %v, want %v", tt.level, got, tt.expected)
			}
		})
	}
}

func TestNewLogger_JSONFormat(t *testing.T) {
	tmpfile, err := os.CreateTemp("", "log-*.log")
	if err != nil {
		t.Fatal(err)
	}
	defer os.Remove(tmpfile.Name())
	tmpfile.Close()

	cfg := config.LoggingConfig{
		Level:      "info",
		Format:     "json",
		OutputFile: tmpfile.Name(),
		MaxSize:    10,
		MaxBackups: 2,
		MaxAge:     7,
	}

	logger := NewLogger(cfg)
	if logger == nil {
		t.Fatal("NewLogger() returned nil")
	}

	// Test that logger can write
	logger.Info("test message", "key", "value")

	// Read the log file
	content, err := os.ReadFile(tmpfile.Name())
	if err != nil {
		t.Fatal(err)
	}

	// Verify JSON format
	if !strings.Contains(string(content), `"msg":"test message"`) {
		t.Errorf("Expected JSON log format, got: %s", string(content))
	}
}

func TestNewLogger_TextFormat(t *testing.T) {
	tmpfile, err := os.CreateTemp("", "log-*.log")
	if err != nil {
		t.Fatal(err)
	}
	defer os.Remove(tmpfile.Name())
	tmpfile.Close()

	cfg := config.LoggingConfig{
		Level:      "debug",
		Format:     "text",
		OutputFile: tmpfile.Name(),
		MaxSize:    10,
		MaxBackups: 2,
		MaxAge:     7,
	}

	logger := NewLogger(cfg)
	if logger == nil {
		t.Fatal("NewLogger() returned nil")
	}

	// Test that logger can write
	logger.Debug("test debug message")

	// Read the log file
	content, err := os.ReadFile(tmpfile.Name())
	if err != nil {
		t.Fatal(err)
	}

	// Verify text format (should contain the message)
	if !strings.Contains(string(content), "test debug message") {
		t.Errorf("Expected text log format with message, got: %s", string(content))
	}
}

func TestColorHandler(t *testing.T) {
	var buf bytes.Buffer
	handler := NewColorHandler(&buf, &slog.HandlerOptions{Level: slog.LevelInfo})

	if handler == nil {
		t.Fatal("NewColorHandler() returned nil")
	}

	ctx := context.Background()

	// Test Enabled
	if !handler.Enabled(ctx, slog.LevelInfo) {
		t.Error("handler.Enabled() = false, want true for info level")
	}

	if handler.Enabled(ctx, slog.LevelDebug) {
		t.Error("handler.Enabled() = true, want false for debug level when min level is info")
	}

	// Test Handle
	record := slog.NewRecord(time.Now(), slog.LevelInfo, "test message", 0)
	err := handler.Handle(ctx, record)
	if err != nil {
		t.Errorf("handler.Handle() error = %v", err)
	}

	// Test WithAttrs
	newHandler := handler.WithAttrs([]slog.Attr{slog.String("key", "value")})
	if newHandler == nil {
		t.Error("handler.WithAttrs() returned nil")
	}

	// Test WithGroup
	groupHandler := handler.WithGroup("testgroup")
	if groupHandler == nil {
		t.Error("handler.WithGroup() returned nil")
	}
}

func TestMultiHandler(t *testing.T) {
	var buf1, buf2 bytes.Buffer
	handler1 := slog.NewTextHandler(&buf1, &slog.HandlerOptions{Level: slog.LevelInfo})
	handler2 := slog.NewTextHandler(&buf2, &slog.HandlerOptions{Level: slog.LevelDebug})

	multiHandler := NewMultiHandler(handler1, handler2)
	if multiHandler == nil {
		t.Fatal("NewMultiHandler() returned nil")
	}

	ctx := context.Background()

	// Test Enabled - should return true if any handler is enabled
	if !multiHandler.Enabled(ctx, slog.LevelInfo) {
		t.Error("multiHandler.Enabled() = false, want true for info level")
	}

	// Test Handle - should write to both handlers
	record := slog.NewRecord(time.Now(), slog.LevelInfo, "test message", 0)
	err := multiHandler.Handle(ctx, record)
	if err != nil {
		t.Errorf("multiHandler.Handle() error = %v", err)
	}

	// Verify both buffers have content
	if buf1.Len() == 0 {
		t.Error("First handler buffer is empty")
	}
	if buf2.Len() == 0 {
		t.Error("Second handler buffer is empty")
	}

	// Test WithAttrs
	newHandler := multiHandler.WithAttrs([]slog.Attr{slog.String("key", "value")})
	if newHandler == nil {
		t.Error("multiHandler.WithAttrs() returned nil")
	}

	// Test WithGroup
	groupHandler := multiHandler.WithGroup("testgroup")
	if groupHandler == nil {
		t.Error("multiHandler.WithGroup() returned nil")
	}
}
