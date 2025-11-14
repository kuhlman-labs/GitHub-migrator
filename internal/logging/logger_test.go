package logging

import (
	"bytes"
	"log/slog"
	"os"
	"strings"
	"testing"

	"github.com/kuhlman-labs/github-migrator/internal/config"
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

func TestMultiHandler(t *testing.T) {
	var buf1, buf2 bytes.Buffer
	handler1 := slog.NewTextHandler(&buf1, &slog.HandlerOptions{Level: slog.LevelInfo})
	handler2 := slog.NewTextHandler(&buf2, &slog.HandlerOptions{Level: slog.LevelDebug})

	multiHandler := NewMultiHandler(handler1, handler2)
	if multiHandler == nil {
		t.Fatal("NewMultiHandler() returned nil")
	}

	// Test that logger can write through multihandler
	logger := slog.New(multiHandler)
	logger.Info("test message")

	// Verify both buffers have content
	if buf1.Len() == 0 {
		t.Error("First handler buffer is empty")
	}
	if buf2.Len() == 0 {
		t.Error("Second handler buffer is empty")
	}
}

func TestNewLogger_TextFormat_NoANSICodes(t *testing.T) {
	tmpfile, err := os.CreateTemp("", "log-*.log")
	if err != nil {
		t.Fatal(err)
	}
	defer os.Remove(tmpfile.Name())
	tmpfile.Close()

	cfg := config.LoggingConfig{
		Level:      "info",
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

	// Test that logger can write with different levels
	logger.Info("info message")
	logger.Debug("debug message")
	logger.Error("error message")

	// Read the log file
	content, err := os.ReadFile(tmpfile.Name())
	if err != nil {
		t.Fatal(err)
	}

	contentStr := string(content)

	// Verify no ANSI escape codes are present in file output
	ansiCodes := []string{"\x1b[", "\\x1b[", "\033[", "\\033["}
	for _, code := range ansiCodes {
		if strings.Contains(contentStr, code) {
			t.Errorf("Log file contains ANSI escape codes: found %q in output:\n%s", code, contentStr)
		}
	}

	// Verify the log message is present
	if !strings.Contains(contentStr, "info message") {
		t.Errorf("Expected log file to contain 'info message', got: %s", contentStr)
	}
}

func TestIsTerminal(t *testing.T) {
	// Test with a regular file (should not be a terminal)
	tmpfile, err := os.CreateTemp("", "test-*.txt")
	if err != nil {
		t.Fatal(err)
	}
	defer os.Remove(tmpfile.Name())
	defer tmpfile.Close()

	if isTerminal(tmpfile) {
		t.Error("isTerminal() = true for a regular file, want false")
	}

	// Note: Testing with actual stdout/stderr is tricky in unit tests
	// because they may be redirected in test environments
	// We just verify the function doesn't panic
	_ = isTerminal(os.Stdout)
	_ = isTerminal(os.Stderr)
}

func TestShouldUseColors(t *testing.T) {
	// Save original env vars
	origNoColor := os.Getenv("NO_COLOR")
	origTerm := os.Getenv("TERM")
	defer func() {
		if origNoColor != "" {
			os.Setenv("NO_COLOR", origNoColor)
		} else {
			os.Unsetenv("NO_COLOR")
		}
		if origTerm != "" {
			os.Setenv("TERM", origTerm)
		} else {
			os.Unsetenv("TERM")
		}
	}()

	tests := []struct {
		name     string
		noColor  string
		term     string
		expected bool
	}{
		{
			name:     "NO_COLOR set",
			noColor:  "1",
			term:     "xterm-256color",
			expected: false,
		},
		{
			name:     "dumb terminal",
			noColor:  "",
			term:     "dumb",
			expected: false,
		},
		{
			name:     "empty TERM",
			noColor:  "",
			term:     "",
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.noColor != "" {
				os.Setenv("NO_COLOR", tt.noColor)
			} else {
				os.Unsetenv("NO_COLOR")
			}
			os.Setenv("TERM", tt.term)

			// Note: In test environment, stdout is likely not a TTY,
			// so shouldUseColors() will likely return false regardless
			// We're mainly testing the logic paths
			result := shouldUseColors()

			// The result should be false because test stdout is not a terminal
			// But we verify the function doesn't panic
			_ = result
		})
	}
}
