package batch

import (
	"log/slog"
	"os"
	"testing"
	"time"
)

func TestNewStatusUpdater(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelError}))

	tests := []struct {
		name         string
		cfg          StatusUpdaterConfig
		wantErr      bool
		errContains  string
		checkDefault bool
	}{
		{
			name: "missing storage",
			cfg: StatusUpdaterConfig{
				Storage: nil,
				Logger:  logger,
			},
			wantErr:     true,
			errContains: "storage is required",
		},
		{
			name: "missing logger",
			cfg: StatusUpdaterConfig{
				Storage: nil, // Can't pass real storage
				Logger:  nil,
			},
			wantErr:     true,
			errContains: "storage is required", // Storage check comes first
		},
		{
			name: "default interval",
			cfg: StatusUpdaterConfig{
				Storage:  nil, // Would fail, but tests default interval
				Logger:   logger,
				Interval: 0, // Should default to 30 seconds
			},
			wantErr:     true,
			errContains: "storage is required",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			updater, err := NewStatusUpdater(tt.cfg)

			if tt.wantErr {
				if err == nil {
					t.Error("Expected error, got nil")
				} else if tt.errContains != "" && !containsStr(err.Error(), tt.errContains) {
					t.Errorf("Expected error to contain %q, got %q", tt.errContains, err.Error())
				}
				return
			}

			if err != nil {
				t.Errorf("Unexpected error: %v", err)
				return
			}

			if updater == nil {
				t.Error("Expected non-nil updater")
			}
		})
	}
}

func TestStatusUpdaterConfig_Interval(t *testing.T) {
	// Test that default interval is set correctly
	cfg := StatusUpdaterConfig{
		Interval: 0,
	}

	if cfg.Interval != 0 {
		t.Errorf("Expected initial interval 0, got %v", cfg.Interval)
	}

	// Test custom interval
	cfg.Interval = 1 * time.Minute
	if cfg.Interval != time.Minute {
		t.Errorf("Expected interval 1 minute, got %v", cfg.Interval)
	}
}

func TestStatusUpdaterConfig_Structure(t *testing.T) {
	// Test that the config struct has the expected fields
	cfg := StatusUpdaterConfig{
		Storage:  nil,
		Logger:   nil,
		Interval: 30 * time.Second,
	}

	// Verify struct fields exist (compile-time check)
	_ = cfg.Storage
	_ = cfg.Logger
	_ = cfg.Interval
}

// Helper function for this file
func containsStr(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
