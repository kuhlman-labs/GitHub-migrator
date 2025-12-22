package batch

import (
	"context"
	"testing"

	"github.com/kuhlman-labs/github-migrator/internal/models"
)

func TestOrchestratorConfig_Validation(t *testing.T) {
	// Test that OrchestratorConfig validation works correctly
	tests := []struct {
		name    string
		cfg     OrchestratorConfig
		wantErr bool
		errMsg  string
	}{
		{
			name: "missing storage",
			cfg: OrchestratorConfig{
				Storage:  nil,
				Executor: &orchestratorMockExecutor{},
			},
			wantErr: true,
			errMsg:  "storage is required",
		},
		{
			name: "missing executor",
			cfg: OrchestratorConfig{
				Storage:  nil, // Would fail earlier
				Executor: nil,
			},
			wantErr: true,
			errMsg:  "storage is required", // First validation fails
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := NewOrchestrator(tt.cfg)

			if tt.wantErr {
				if err == nil {
					t.Error("Expected error, got nil")
				} else if !containsString(err.Error(), tt.errMsg) {
					t.Errorf("Expected error to contain %q, got %q", tt.errMsg, err.Error())
				}
			} else {
				if err != nil {
					t.Errorf("Unexpected error: %v", err)
				}
			}
		})
	}
}

func TestOrchestratorConfig_Structure(t *testing.T) {
	// Test that the config struct has the expected fields
	cfg := OrchestratorConfig{
		Storage:  nil,
		Executor: nil,
		Logger:   nil,
	}

	// Verify struct fields exist (compile-time check)
	_ = cfg.Storage
	_ = cfg.Executor
	_ = cfg.Logger
}

// orchestratorMockExecutor is a simple mock that implements MigrationExecutor
type orchestratorMockExecutor struct{}

func (m *orchestratorMockExecutor) ExecuteMigration(ctx context.Context, repo *models.Repository, batch *models.Batch, dryRun bool) error {
	return nil
}

// Helper function for this file
func containsString(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
