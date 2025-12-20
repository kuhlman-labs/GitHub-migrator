package batch

import (
	"testing"

	"github.com/kuhlman-labs/github-migrator/internal/models"
)

// TestCanMigrate_Comprehensive tests the canMigrate helper function with all status combinations
func TestCanMigrate_Comprehensive(t *testing.T) {
	tests := []struct {
		name     string
		status   string
		expected bool
	}{
		// Statuses that CAN be migrated
		{
			name:     "queued for migration - can migrate",
			status:   string(models.StatusQueuedForMigration),
			expected: true,
		},
		{
			name:     "dry run queued - can migrate",
			status:   string(models.StatusDryRunQueued),
			expected: true,
		},
		{
			name:     "dry run failed - can retry",
			status:   string(models.StatusDryRunFailed),
			expected: true,
		},
		{
			name:     "dry run complete - can migrate",
			status:   string(models.StatusDryRunComplete),
			expected: true,
		},
		{
			name:     "migration failed - can retry",
			status:   string(models.StatusMigrationFailed),
			expected: true,
		},
		// Statuses that CANNOT be migrated
		{
			name:     "won't migrate - blocked",
			status:   string(models.StatusWontMigrate),
			expected: false,
		},
		{
			name:     "pending - not yet queued",
			status:   string(models.StatusPending),
			expected: false,
		},
		{
			name:     "complete - already done",
			status:   string(models.StatusComplete),
			expected: false,
		},
		{
			name:     "in progress - already migrating",
			status:   string(models.StatusMigratingContent),
			expected: false,
		},
		{
			name:     "rolled back - needs re-evaluation",
			status:   string(models.StatusRolledBack),
			expected: false,
		},
		{
			name:     "empty status",
			status:   "",
			expected: false,
		},
		{
			name:     "invalid status",
			status:   "invalid_status",
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := canMigrate(tt.status)
			if result != tt.expected {
				t.Errorf("canMigrate(%q) = %v, want %v", tt.status, result, tt.expected)
			}
		})
	}
}

// TestIsTerminalStatus tests the isTerminalStatus helper function
func TestIsTerminalStatus(t *testing.T) {
	tests := []struct {
		name     string
		status   string
		expected bool
	}{
		// Terminal statuses
		{
			name:     "completed - terminal",
			status:   models.BatchStatusCompleted,
			expected: true,
		},
		{
			name:     "failed - terminal",
			status:   models.BatchStatusFailed,
			expected: true,
		},
		{
			name:     "completed with errors - terminal",
			status:   models.BatchStatusCompletedWithErrors,
			expected: true,
		},
		{
			name:     "cancelled - terminal",
			status:   models.BatchStatusCancelled,
			expected: true,
		},
		// Non-terminal statuses
		{
			name:     "pending - not terminal",
			status:   models.BatchStatusPending,
			expected: false,
		},
		{
			name:     "ready - not terminal",
			status:   models.BatchStatusReady,
			expected: false,
		},
		{
			name:     "in progress - not terminal",
			status:   models.BatchStatusInProgress,
			expected: false,
		},
		{
			name:     "empty status",
			status:   "",
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := isTerminalStatus(tt.status)
			if result != tt.expected {
				t.Errorf("isTerminalStatus(%q) = %v, want %v", tt.status, result, tt.expected)
			}
		})
	}
}

// TestCalculateBatchStatusFromRepos tests batch status calculation logic
func TestCalculateBatchStatusFromRepos(t *testing.T) {
	tests := []struct {
		name           string
		repos          []*models.Repository
		expectedStatus string
	}{
		{
			name:           "empty repos - completed (0 == 0)",
			repos:          []*models.Repository{},
			expectedStatus: models.BatchStatusCompleted, // totalRepos == 0 and completedCount == 0, so 0 == 0 is true
		},
		{
			name: "all pending - ready",
			repos: []*models.Repository{
				{Status: string(models.StatusPending)},
				{Status: string(models.StatusPending)},
			},
			expectedStatus: models.BatchStatusReady,
		},
		{
			name: "all complete - completed",
			repos: []*models.Repository{
				{Status: string(models.StatusComplete)},
				{Status: string(models.StatusComplete)},
				{Status: string(models.StatusComplete)},
			},
			expectedStatus: models.BatchStatusCompleted,
		},
		{
			name: "all failed - failed",
			repos: []*models.Repository{
				{Status: string(models.StatusMigrationFailed)},
				{Status: string(models.StatusMigrationFailed)},
			},
			expectedStatus: models.BatchStatusFailed,
		},
		{
			name: "some complete some failed - completed with errors",
			repos: []*models.Repository{
				{Status: string(models.StatusComplete)},
				{Status: string(models.StatusComplete)},
				{Status: string(models.StatusMigrationFailed)},
			},
			expectedStatus: models.BatchStatusCompletedWithErrors,
		},
		{
			name: "some in progress - in progress",
			repos: []*models.Repository{
				{Status: string(models.StatusComplete)},
				{Status: string(models.StatusMigratingContent)},
				{Status: string(models.StatusPending)},
			},
			expectedStatus: models.BatchStatusInProgress,
		},
		{
			name: "dry run queued - in progress",
			repos: []*models.Repository{
				{Status: string(models.StatusDryRunQueued)},
				{Status: string(models.StatusPending)},
			},
			expectedStatus: models.BatchStatusInProgress,
		},
		{
			name: "dry run in progress - in progress",
			repos: []*models.Repository{
				{Status: string(models.StatusDryRunInProgress)},
			},
			expectedStatus: models.BatchStatusInProgress,
		},
		{
			name: "archive generating - in progress",
			repos: []*models.Repository{
				{Status: string(models.StatusArchiveGenerating)},
			},
			expectedStatus: models.BatchStatusInProgress,
		},
		{
			name: "all dry run complete - ready",
			repos: []*models.Repository{
				{Status: string(models.StatusDryRunComplete)},
				{Status: string(models.StatusDryRunComplete)},
			},
			expectedStatus: models.BatchStatusReady,
		},
		{
			name: "dry run failed - completed with errors",
			repos: []*models.Repository{
				{Status: string(models.StatusDryRunComplete)},
				{Status: string(models.StatusDryRunFailed)},
			},
			expectedStatus: models.BatchStatusCompletedWithErrors, // failedCount > 0 triggers CompletedWithErrors
		},
		{
			name: "mixed dry run and migration failed",
			repos: []*models.Repository{
				{Status: string(models.StatusDryRunFailed)},
				{Status: string(models.StatusMigrationFailed)},
			},
			expectedStatus: models.BatchStatusFailed,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := CalculateBatchStatusFromRepos(tt.repos)
			if result != tt.expectedStatus {
				t.Errorf("CalculateBatchStatusFromRepos() = %q, want %q", result, tt.expectedStatus)
			}
		})
	}
}

// TestBatchStatusConstants tests batch status constants are correct
func TestBatchStatusConstants(t *testing.T) {
	tests := []struct {
		constant string
		expected string
	}{
		{models.BatchStatusPending, "pending"},
		{models.BatchStatusReady, "ready"},
		{models.BatchStatusInProgress, "in_progress"},
		{models.BatchStatusCompleted, "completed"},
		{models.BatchStatusCompletedWithErrors, "completed_with_errors"},
		{models.BatchStatusFailed, "failed"},
		{models.BatchStatusCancelled, "cancelled"},
	}

	for _, tt := range tests {
		t.Run(tt.expected, func(t *testing.T) {
			if tt.constant != tt.expected {
				t.Errorf("Expected %q, got %q", tt.expected, tt.constant)
			}
		})
	}
}

// TestBatchTypeConstants tests batch type constants
func TestBatchTypeConstants(t *testing.T) {
	if models.BatchTypePilot != "pilot" {
		t.Errorf("Expected BatchTypePilot='pilot', got %q", models.BatchTypePilot)
	}
}

// TestCanMigrate_AllStatusTransitions tests valid status transitions for migration
func TestCanMigrate_AllStatusTransitions(t *testing.T) {
	// These are the statuses from which a repository can be migrated
	migratableStatuses := []models.MigrationStatus{
		models.StatusQueuedForMigration,
		models.StatusDryRunQueued,
		models.StatusDryRunFailed,
		models.StatusDryRunComplete,
		models.StatusMigrationFailed,
	}

	for _, status := range migratableStatuses {
		t.Run(string(status), func(t *testing.T) {
			if !canMigrate(string(status)) {
				t.Errorf("Expected status %q to be migratable", status)
			}
		})
	}

	// These are statuses from which a repository CANNOT be migrated
	nonMigratableStatuses := []models.MigrationStatus{
		models.StatusPending,
		models.StatusComplete,
		models.StatusWontMigrate,
		models.StatusRolledBack,
		models.StatusMigratingContent,
		models.StatusArchiveGenerating,
		models.StatusPreMigration,
		models.StatusPostMigration,
		models.StatusMigrationComplete,
	}

	for _, status := range nonMigratableStatuses {
		t.Run(string(status)+"_not_migratable", func(t *testing.T) {
			if canMigrate(string(status)) {
				t.Errorf("Expected status %q to NOT be migratable", status)
			}
		})
	}
}

// TestBatchStatusFlow tests the expected batch status flow
func TestBatchStatusFlow(t *testing.T) {
	// Test the typical status flow
	t.Run("typical batch lifecycle", func(t *testing.T) {
		// 1. Batch starts as pending or ready
		initialStatuses := []string{models.BatchStatusPending, models.BatchStatusReady}
		for _, s := range initialStatuses {
			if isTerminalStatus(s) {
				t.Errorf("Initial status %q should not be terminal", s)
			}
		}

		// 2. Batch goes in_progress when executing
		if isTerminalStatus(models.BatchStatusInProgress) {
			t.Error("in_progress should not be terminal")
		}

		// 3. Batch ends in a terminal state
		terminalStatuses := []string{models.BatchStatusCompleted, models.BatchStatusFailed, models.BatchStatusCompletedWithErrors, models.BatchStatusCancelled}
		for _, s := range terminalStatuses {
			if !isTerminalStatus(s) {
				t.Errorf("Terminal status %q should be marked as terminal", s)
			}
		}
	})
}
