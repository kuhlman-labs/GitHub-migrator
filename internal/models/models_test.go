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

// TestRepository_DestinationRepoName tests destination repository name computation
func TestRepository_DestinationRepoName(t *testing.T) {
	tests := []struct {
		name       string
		fullName   string
		adoProject *string
		expected   string
	}{
		{
			name:       "GitHub repo - standard org/repo format",
			fullName:   "my-org/my-repo",
			adoProject: nil,
			expected:   "my-repo",
		},
		{
			name:       "GitHub repo - repo with spaces",
			fullName:   "my-org/my repo",
			adoProject: nil,
			expected:   "my-repo",
		},
		{
			name:       "ADO repo - org/project/repo format",
			fullName:   "brettkuhlman/Test Project/Test Project 2",
			adoProject: strPtr("Test Project"),
			expected:   "Test-Project-Test-Project-2",
		},
		{
			name:       "ADO repo - simple project and repo",
			fullName:   "org/DevOps/Terraform",
			adoProject: strPtr("DevOps"),
			expected:   "DevOps-Terraform",
		},
		{
			name:       "ADO repo - project with spaces",
			fullName:   "org/My Project/my-repo",
			adoProject: strPtr("My Project"),
			expected:   "My-Project-my-repo",
		},
		{
			name:       "ADO repo - complex nested path",
			fullName:   "brettkuhlman/Test Project/test-ado-frontend-app",
			adoProject: strPtr("Test Project"),
			expected:   "Test-Project-test-ado-frontend-app",
		},
		{
			name:       "single word (no slash)",
			fullName:   "onlyrepo",
			adoProject: nil,
			expected:   "onlyrepo",
		},
		{
			name:       "empty string",
			fullName:   "",
			adoProject: nil,
			expected:   "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo := &Repository{
				FullName: tt.fullName,
			}
			if tt.adoProject != nil {
				repo.ADOProperties = &RepositoryADOProperties{
					Project: tt.adoProject,
				}
			}
			result := repo.DestinationRepoName()
			if result != tt.expected {
				t.Errorf("DestinationRepoName() = %q, want %q", result, tt.expected)
			}
		})
	}
}

// strPtr is a helper to create string pointers for tests
func strPtr(s string) *string {
	return &s
}

// TestRepository_SetComplexityBreakdown tests setting complexity breakdown
func TestRepository_SetComplexityBreakdown(t *testing.T) {
	t.Run("set nil breakdown", func(t *testing.T) {
		repo := &Repository{Validation: &RepositoryValidation{}}
		err := repo.SetComplexityBreakdown(nil)
		if err != nil {
			t.Errorf("Unexpected error: %v", err)
		}
		if repo.Validation.ComplexityBreakdown != nil {
			t.Error("Expected ComplexityBreakdown to be nil")
		}
	})

	t.Run("set valid breakdown", func(t *testing.T) {
		repo := &Repository{Validation: &RepositoryValidation{}}
		breakdown := &ComplexityBreakdown{
			SizePoints:         5,
			LargeFilesPoints:   4,
			EnvironmentsPoints: 3,
		}
		err := repo.SetComplexityBreakdown(breakdown)
		if err != nil {
			t.Errorf("Unexpected error: %v", err)
		}
		if repo.Validation.ComplexityBreakdown == nil {
			t.Fatal("Expected ComplexityBreakdown to be set")
		}
		if *repo.Validation.ComplexityBreakdown == "" {
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
		repo := &Repository{Validation: &RepositoryValidation{ComplexityBreakdown: &emptyStr}}
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
		repo := &Repository{Validation: &RepositoryValidation{ComplexityBreakdown: &jsonStr}}
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
		repo := &Repository{Validation: &RepositoryValidation{ComplexityBreakdown: &invalidJSON}}
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

		var result map[string]any
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
			FullName:   "org/repo",
			Source:     "ghes",
			Status:     "pending",
			Validation: &RepositoryValidation{ComplexityBreakdown: &jsonStr},
		}
		data, err := json.Marshal(repo)
		if err != nil {
			t.Errorf("Unexpected error: %v", err)
		}

		var result map[string]any
		if err := json.Unmarshal(data, &result); err != nil {
			t.Errorf("Failed to unmarshal result: %v", err)
		}

		// Verify validation is present and contains complexity_breakdown as an object
		validation, ok := result["validation"].(map[string]any)
		if !ok {
			t.Error("Expected validation to be an object")
			return
		}
		breakdown, ok := validation["complexity_breakdown"].(map[string]any)
		if !ok {
			t.Errorf("Expected complexity_breakdown to be an object in validation, got %T", validation["complexity_breakdown"])
			return
		}
		if breakdown["size_points"] != float64(5) {
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
		{StatusSyncing, "syncing"},
		{StatusAwaitingCutover, "awaiting_cutover"},
		{StatusCuttingOver, "cutting_over"},
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
	if history.Status != BatchStatusFailed {
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

// TestMigrationRoute_Constants asserts the two legal route values render exactly as
// the storage column, API and dashboard expect. A typo here silently unroutes ELM.
func TestMigrationRoute_Constants(t *testing.T) {
	if string(MigrationRouteGEI) != "gei" {
		t.Errorf("Expected MigrationRouteGEI='gei', got %q", string(MigrationRouteGEI))
	}
	if string(MigrationRouteELM) != "elm" {
		t.Errorf("Expected MigrationRouteELM='elm', got %q", string(MigrationRouteELM))
	}
}

// TestGetMigrationRoute_DefaultsToGEI pins the route contract's default: an
// unrouted repository is GEI-routed, so no backfill is needed for existing rows.
func TestGetMigrationRoute_DefaultsToGEI(t *testing.T) {
	empty := ""
	elm := "elm"
	gei := "gei"

	tests := []struct {
		name     string
		route    *string
		expected string
	}{
		{name: "nil route defaults to gei", route: nil, expected: "gei"},
		{name: "empty route defaults to gei", route: &empty, expected: "gei"},
		{name: "explicit gei", route: &gei, expected: "gei"},
		{name: "elm route", route: &elm, expected: "elm"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo := &Repository{MigrationRoute: tt.route}
			if got := repo.GetMigrationRoute(); got != tt.expected {
				t.Errorf("GetMigrationRoute() = %q, want %q", got, tt.expected)
			}
		})
	}
}

// TestRepository_IsELMRouted tests the single route predicate ELMStrategy consults.
func TestRepository_IsELMRouted(t *testing.T) {
	elm := "elm"
	gei := "gei"
	empty := ""

	tests := []struct {
		name     string
		route    *string
		expected bool
	}{
		{name: "nil route is not ELM", route: nil, expected: false},
		{name: "empty route is not ELM", route: &empty, expected: false},
		{name: "gei route is not ELM", route: &gei, expected: false},
		{name: "elm route is ELM", route: &elm, expected: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo := &Repository{MigrationRoute: tt.route}
			if got := repo.IsELMRouted(); got != tt.expected {
				t.Errorf("IsELMRouted() = %v, want %v", got, tt.expected)
			}
		})
	}
}

// TestIsValidMigrationRoute tests the validator used by the operator route writer.
func TestIsValidMigrationRoute(t *testing.T) {
	tests := []struct {
		route    string
		expected bool
	}{
		{"gei", true},
		{"elm", true},
		{"", false},
		{"GEI", false},
		{"ELM", false},
		{"gei-plus", false},
		{"github", false},
	}

	for _, tt := range tests {
		t.Run(tt.route, func(t *testing.T) {
			if got := IsValidMigrationRoute(tt.route); got != tt.expected {
				t.Errorf("IsValidMigrationRoute(%q) = %v, want %v", tt.route, got, tt.expected)
			}
		})
	}
}

// TestIsMigrationInProgress_ELMStatuses asserts the three ELM lifecycle statuses
// count as in progress, so an ELM repository is never treated as idle or re-queued.
func TestIsMigrationInProgress_ELMStatuses(t *testing.T) {
	tests := []struct {
		status   MigrationStatus
		expected bool
	}{
		{StatusSyncing, true},
		{StatusAwaitingCutover, true},
		{StatusCuttingOver, true},
		{StatusPending, false},
		{StatusMigrationComplete, false},
	}

	for _, tt := range tests {
		t.Run(string(tt.status), func(t *testing.T) {
			repo := &Repository{Status: string(tt.status)}
			if got := repo.IsMigrationInProgress(); got != tt.expected {
				t.Errorf("IsMigrationInProgress() for %q = %v, want %v", tt.status, got, tt.expected)
			}
		})
	}
}

// TestCanBeAssignedToBatch_ELMRoutedOversized pins the ELM branch of the batch
// eligibility guard: an oversized ELM-routed repository is assignable (ELM is what
// serves it), while an oversized unrouted repository is still refused.
func TestCanBeAssignedToBatch_ELMRoutedOversized(t *testing.T) {
	elm := "elm"

	oversizedELM := &Repository{
		Status:         string(StatusPending),
		MigrationRoute: &elm,
		Validation:     &RepositoryValidation{HasOversizedRepository: true},
	}
	if ok, reason := oversizedELM.CanBeAssignedToBatch(); !ok {
		t.Errorf("oversized ELM-routed repository should be assignable, got refusal: %q", reason)
	}

	oversizedUnrouted := &Repository{
		Status:     string(StatusPending),
		Validation: &RepositoryValidation{HasOversizedRepository: true},
	}
	ok, reason := oversizedUnrouted.CanBeAssignedToBatch()
	if ok {
		t.Error("oversized unrouted repository should be refused")
	}
	if reason == "" {
		t.Error("expected a refusal reason for the oversized unrouted repository")
	}

	// An ELM route does not bypass the other guards.
	batchID := int64(7)
	alreadyBatched := &Repository{
		Status:         string(StatusPending),
		MigrationRoute: &elm,
		BatchID:        &batchID,
		Validation:     &RepositoryValidation{HasOversizedRepository: true},
	}
	if ok, _ := alreadyBatched.CanBeAssignedToBatch(); ok {
		t.Error("ELM-routed repository already in a batch should still be refused")
	}
}

// TestELMMigration_TableName tests the elm_migrations table name
func TestELMMigration_TableName(t *testing.T) {
	rec := ELMMigration{}
	if rec.TableName() != "elm_migrations" {
		t.Errorf("Expected table name 'elm_migrations', got %q", rec.TableName())
	}
}

// TestRepository_MarshalJSON_MigrationRoute asserts the route reaches the API
// payload the dashboard reads (Repository has a custom MarshalJSON).
func TestRepository_MarshalJSON_MigrationRoute(t *testing.T) {
	elm := "elm"
	repo := &Repository{FullName: "org/repo", MigrationRoute: &elm}

	data, err := json.Marshal(repo)
	if err != nil {
		t.Fatalf("Marshal() error = %v", err)
	}

	var decoded map[string]any
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("Unmarshal() error = %v", err)
	}
	if decoded["migration_route"] != "elm" {
		t.Errorf("Expected migration_route='elm' in JSON, got %v", decoded["migration_route"])
	}

	// An unrouted repository omits the field entirely.
	unrouted := &Repository{FullName: "org/other"}
	data, err = json.Marshal(unrouted)
	if err != nil {
		t.Fatalf("Marshal() error = %v", err)
	}
	decoded = map[string]any{}
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("Unmarshal() error = %v", err)
	}
	if _, present := decoded["migration_route"]; present {
		t.Error("Expected migration_route to be omitted for an unrouted repository")
	}
}
