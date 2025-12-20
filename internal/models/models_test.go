package models

import (
	"encoding/json"
	"testing"
	"time"
)

// TestRepository_Organization tests organization extraction from full name
func TestRepository_Organization(t *testing.T) {
	tests := []struct {
		name     string
		fullName string
		expected string
	}{
		{
			name:     "standard org/repo format",
			fullName: "my-org/my-repo",
			expected: "my-org",
		},
		{
			name:     "org with numbers",
			fullName: "org123/repo",
			expected: "org123",
		},
		{
			name:     "org with hyphens",
			fullName: "my-awesome-org/repo-name",
			expected: "my-awesome-org",
		},
		{
			name:     "repo name with slashes (ADO format)",
			fullName: "org/project/repo",
			expected: "org",
		},
		{
			name:     "single word (no slash)",
			fullName: "onlyrepo",
			expected: "onlyrepo",
		},
		{
			name:     "empty string",
			fullName: "",
			expected: "",
		},
		{
			name:     "trailing slash",
			fullName: "org/",
			expected: "org",
		},
		{
			name:     "leading slash",
			fullName: "/repo",
			expected: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo := &Repository{FullName: tt.fullName}
			result := repo.Organization()
			if result != tt.expected {
				t.Errorf("Organization() = %q, want %q", result, tt.expected)
			}
		})
	}
}

// TestRepository_Name tests repository name extraction from full name
func TestRepository_Name(t *testing.T) {
	tests := []struct {
		name     string
		fullName string
		expected string
	}{
		{
			name:     "standard org/repo format",
			fullName: "my-org/my-repo",
			expected: "my-repo",
		},
		{
			name:     "repo with hyphens",
			fullName: "org/my-awesome-repo",
			expected: "my-awesome-repo",
		},
		{
			name:     "repo name with nested path (ADO format)",
			fullName: "org/project/repo",
			expected: "project/repo",
		},
		{
			name:     "single word (no slash)",
			fullName: "onlyrepo",
			expected: "onlyrepo",
		},
		{
			name:     "empty string",
			fullName: "",
			expected: "",
		},
		{
			name:     "trailing slash",
			fullName: "org/",
			expected: "",
		},
		{
			name:     "leading slash",
			fullName: "/repo",
			expected: "repo",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo := &Repository{FullName: tt.fullName}
			result := repo.Name()
			if result != tt.expected {
				t.Errorf("Name() = %q, want %q", result, tt.expected)
			}
		})
	}
}

// TestRepository_SetComplexityBreakdown tests setting complexity breakdown
func TestRepository_SetComplexityBreakdown(t *testing.T) {
	t.Run("set nil breakdown", func(t *testing.T) {
		repo := &Repository{}
		err := repo.SetComplexityBreakdown(nil)
		if err != nil {
			t.Errorf("Unexpected error: %v", err)
		}
		if repo.ComplexityBreakdown != nil {
			t.Error("Expected ComplexityBreakdown to be nil")
		}
	})

	t.Run("set valid breakdown", func(t *testing.T) {
		repo := &Repository{}
		breakdown := &ComplexityBreakdown{
			SizePoints:         5,
			LargeFilesPoints:   4,
			EnvironmentsPoints: 3,
		}
		err := repo.SetComplexityBreakdown(breakdown)
		if err != nil {
			t.Errorf("Unexpected error: %v", err)
		}
		if repo.ComplexityBreakdown == nil {
			t.Fatal("Expected ComplexityBreakdown to be set")
		}
		if *repo.ComplexityBreakdown == "" {
			t.Error("Expected non-empty ComplexityBreakdown JSON")
		}
	})
}

// TestRepository_GetComplexityBreakdown tests getting complexity breakdown
func TestRepository_GetComplexityBreakdown(t *testing.T) {
	t.Run("nil breakdown", func(t *testing.T) {
		repo := &Repository{}
		breakdown, err := repo.GetComplexityBreakdown()
		if err != nil {
			t.Errorf("Unexpected error: %v", err)
		}
		if breakdown != nil {
			t.Error("Expected nil breakdown")
		}
	})

	t.Run("empty string breakdown", func(t *testing.T) {
		emptyStr := ""
		repo := &Repository{ComplexityBreakdown: &emptyStr}
		breakdown, err := repo.GetComplexityBreakdown()
		if err != nil {
			t.Errorf("Unexpected error: %v", err)
		}
		if breakdown != nil {
			t.Error("Expected nil breakdown for empty string")
		}
	})

	t.Run("valid JSON breakdown", func(t *testing.T) {
		jsonStr := `{"size_points":5,"large_files_points":4}`
		repo := &Repository{ComplexityBreakdown: &jsonStr}
		breakdown, err := repo.GetComplexityBreakdown()
		if err != nil {
			t.Errorf("Unexpected error: %v", err)
		}
		if breakdown == nil {
			t.Fatal("Expected non-nil breakdown")
			return // Explicitly unreachable, but satisfies static analysis
		}
		if breakdown.SizePoints != 5 {
			t.Errorf("Expected SizePoints=5, got %d", breakdown.SizePoints)
		}
		if breakdown.LargeFilesPoints != 4 {
			t.Errorf("Expected LargeFilesPoints=4, got %d", breakdown.LargeFilesPoints)
		}
	})

	t.Run("invalid JSON breakdown", func(t *testing.T) {
		invalidJSON := `{invalid json}`
		repo := &Repository{ComplexityBreakdown: &invalidJSON}
		_, err := repo.GetComplexityBreakdown()
		if err == nil {
			t.Error("Expected error for invalid JSON")
		}
	})
}

// TestRepository_MarshalJSON tests custom JSON marshaling
func TestRepository_MarshalJSON(t *testing.T) {
	t.Run("marshal without complexity breakdown", func(t *testing.T) {
		repo := &Repository{
			FullName: "org/repo",
			Source:   "ghes",
			Status:   "pending",
		}
		data, err := json.Marshal(repo)
		if err != nil {
			t.Errorf("Unexpected error: %v", err)
		}

		var result map[string]interface{}
		if err := json.Unmarshal(data, &result); err != nil {
			t.Errorf("Failed to unmarshal result: %v", err)
		}

		if result["full_name"] != "org/repo" {
			t.Errorf("Expected full_name='org/repo', got %v", result["full_name"])
		}
	})

	t.Run("marshal with complexity breakdown", func(t *testing.T) {
		jsonStr := `{"size_points":5}`
		repo := &Repository{
			FullName:            "org/repo",
			Source:              "ghes",
			Status:              "pending",
			ComplexityBreakdown: &jsonStr,
		}
		data, err := json.Marshal(repo)
		if err != nil {
			t.Errorf("Unexpected error: %v", err)
		}

		var result map[string]interface{}
		if err := json.Unmarshal(data, &result); err != nil {
			t.Errorf("Failed to unmarshal result: %v", err)
		}

		// Verify complexity_breakdown is marshaled as object not string
		breakdown, ok := result["complexity_breakdown"].(map[string]interface{})
		if !ok {
			t.Error("Expected complexity_breakdown to be an object")
		} else if breakdown["size_points"] != float64(5) {
			t.Errorf("Expected size_points=5, got %v", breakdown["size_points"])
		}
	})
}

// TestMigrationStatus_Constants tests that status constants are correct
func TestMigrationStatus_Constants(t *testing.T) {
	statuses := []struct {
		status   MigrationStatus
		expected string
	}{
		{StatusPending, "pending"},
		{StatusRemediationRequired, "remediation_required"},
		{StatusDryRunQueued, "dry_run_queued"},
		{StatusDryRunInProgress, "dry_run_in_progress"},
		{StatusDryRunComplete, "dry_run_complete"},
		{StatusDryRunFailed, "dry_run_failed"},
		{StatusPreMigration, "pre_migration"},
		{StatusArchiveGenerating, "archive_generating"},
		{StatusQueuedForMigration, "queued_for_migration"},
		{StatusMigratingContent, "migrating_content"},
		{StatusMigrationComplete, "migration_complete"},
		{StatusMigrationFailed, "migration_failed"},
		{StatusPostMigration, "post_migration"},
		{StatusComplete, "complete"},
		{StatusRolledBack, "rolled_back"},
		{StatusWontMigrate, "wont_migrate"},
	}

	for _, tt := range statuses {
		t.Run(string(tt.status), func(t *testing.T) {
			if string(tt.status) != tt.expected {
				t.Errorf("Expected status %q, got %q", tt.expected, string(tt.status))
			}
		})
	}
}

// TestBatch_TableName tests batch table name
func TestBatch_TableName(t *testing.T) {
	batch := Batch{}
	if batch.TableName() != "batches" {
		t.Errorf("Expected table name 'batches', got %q", batch.TableName())
	}
}

// TestRepository_TableName tests repository table name
func TestRepository_TableName(t *testing.T) {
	repo := Repository{}
	if repo.TableName() != "repositories" {
		t.Errorf("Expected table name 'repositories', got %q", repo.TableName())
	}
}

// TestMigrationHistory_TableName tests migration history table name
func TestMigrationHistory_TableName(t *testing.T) {
	history := MigrationHistory{}
	if history.TableName() != "migration_history" {
		t.Errorf("Expected table name 'migration_history', got %q", history.TableName())
	}
}

// TestMigrationLog_TableName tests migration log table name
func TestMigrationLog_TableName(t *testing.T) {
	log := MigrationLog{}
	if log.TableName() != "migration_logs" {
		t.Errorf("Expected table name 'migration_logs', got %q", log.TableName())
	}
}

// TestRepositoryDependency_TableName tests dependency table name
func TestRepositoryDependency_TableName(t *testing.T) {
	dep := RepositoryDependency{}
	if dep.TableName() != "repository_dependencies" {
		t.Errorf("Expected table name 'repository_dependencies', got %q", dep.TableName())
	}
}

// TestDependencyType_Constants tests dependency type constants
func TestDependencyType_Constants(t *testing.T) {
	tests := []struct {
		constant string
		expected string
	}{
		{DependencyTypeSubmodule, "submodule"},
		{DependencyTypeWorkflow, "workflow"},
		{DependencyTypeDependencyGraph, "dependency_graph"},
		{DependencyTypePackage, "package"},
	}

	for _, tt := range tests {
		t.Run(tt.expected, func(t *testing.T) {
			if tt.constant != tt.expected {
				t.Errorf("Expected %q, got %q", tt.expected, tt.constant)
			}
		})
	}
}

// TestMigrationAPIConstants tests migration API type constants
func TestMigrationAPIConstants(t *testing.T) {
	if MigrationAPIGEI != "GEI" {
		t.Errorf("Expected MigrationAPIGEI='GEI', got %q", MigrationAPIGEI)
	}
	if MigrationAPIELM != "ELM" {
		t.Errorf("Expected MigrationAPIELM='ELM', got %q", MigrationAPIELM)
	}
}

// TestComplexityBreakdown_Serialization tests complexity breakdown round-trip
func TestComplexityBreakdown_Serialization(t *testing.T) {
	original := &ComplexityBreakdown{
		SizePoints:              5,
		LargeFilesPoints:        4,
		EnvironmentsPoints:      3,
		SecretsPoints:           3,
		PackagesPoints:          3,
		RunnersPoints:           3,
		VariablesPoints:         2,
		DiscussionsPoints:       2,
		ReleasesPoints:          2,
		LFSPoints:               2,
		SubmodulesPoints:        2,
		AppsPoints:              2,
		ProjectsPoints:          2,
		SecurityPoints:          1,
		WebhooksPoints:          1,
		BranchProtectionsPoints: 1,
		RulesetsPoints:          1,
		PublicVisibilityPoints:  1,
		CodeownersPoints:        1,
		ActivityPoints:          4,
		// ADO-specific
		ADOTFVCPoints:            50,
		ADOClassicPipelinePoints: 10,
	}

	repo := &Repository{}
	err := repo.SetComplexityBreakdown(original)
	if err != nil {
		t.Fatalf("Failed to set complexity breakdown: %v", err)
	}

	retrieved, err := repo.GetComplexityBreakdown()
	if err != nil {
		t.Fatalf("Failed to get complexity breakdown: %v", err)
	}

	if retrieved.SizePoints != original.SizePoints {
		t.Errorf("SizePoints: expected %d, got %d", original.SizePoints, retrieved.SizePoints)
	}
	if retrieved.ADOTFVCPoints != original.ADOTFVCPoints {
		t.Errorf("ADOTFVCPoints: expected %d, got %d", original.ADOTFVCPoints, retrieved.ADOTFVCPoints)
	}
	if retrieved.ActivityPoints != original.ActivityPoints {
		t.Errorf("ActivityPoints: expected %d, got %d", original.ActivityPoints, retrieved.ActivityPoints)
	}
}

// TestBatch_Structure tests batch struct initialization
func TestBatch_Structure(t *testing.T) {
	desc := "Test batch description"
	destOrg := "dest-org"

	batch := Batch{
		ID:                 1,
		Name:               "Test Batch",
		Description:        &desc,
		Type:               "pilot",
		Status:             "pending",
		RepositoryCount:    10,
		DestinationOrg:     &destOrg,
		ExcludeReleases:    true,
		ExcludeAttachments: false,
		CreatedAt:          time.Now(),
	}

	if batch.Name != "Test Batch" {
		t.Errorf("Expected name 'Test Batch', got %q", batch.Name)
	}
	if batch.RepositoryCount != 10 {
		t.Errorf("Expected RepositoryCount=10, got %d", batch.RepositoryCount)
	}
	if !batch.ExcludeReleases {
		t.Error("Expected ExcludeReleases to be true")
	}
	if batch.ExcludeAttachments {
		t.Error("Expected ExcludeAttachments to be false")
	}
	if batch.DestinationOrg == nil || *batch.DestinationOrg != "dest-org" {
		t.Error("Expected DestinationOrg to be 'dest-org'")
	}
}

// TestMigrationHistory_Structure tests migration history struct
func TestMigrationHistory_Structure(t *testing.T) {
	now := time.Now()
	errMsg := "test error"

	history := MigrationHistory{
		ID:           1,
		RepositoryID: 42,
		Status:       "failed",
		Phase:        "migration",
		ErrorMessage: &errMsg,
		StartedAt:    now,
	}

	if history.RepositoryID != 42 {
		t.Errorf("Expected RepositoryID=42, got %d", history.RepositoryID)
	}
	if history.Status != "failed" {
		t.Errorf("Expected Status='failed', got %q", history.Status)
	}
	if history.ErrorMessage == nil || *history.ErrorMessage != "test error" {
		t.Error("Expected ErrorMessage to be 'test error'")
	}
}

// TestMigrationLog_Structure tests migration log struct
func TestMigrationLog_Structure(t *testing.T) {
	historyID := int64(1)
	details := "detailed info"
	initiatedBy := "user@example.com"

	log := MigrationLog{
		ID:           1,
		RepositoryID: 42,
		HistoryID:    &historyID,
		Level:        "ERROR",
		Phase:        "archive_generation",
		Operation:    "generate",
		Message:      "Failed to generate archive",
		Details:      &details,
		InitiatedBy:  &initiatedBy,
		Timestamp:    time.Now(),
	}

	if log.Level != "ERROR" {
		t.Errorf("Expected Level='ERROR', got %q", log.Level)
	}
	if log.Phase != "archive_generation" {
		t.Errorf("Expected Phase='archive_generation', got %q", log.Phase)
	}
	if log.InitiatedBy == nil || *log.InitiatedBy != "user@example.com" {
		t.Error("Expected InitiatedBy to be 'user@example.com'")
	}
}
