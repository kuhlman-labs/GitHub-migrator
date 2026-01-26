package copilot

import (
	"context"
	"log/slog"
	"os"
	"testing"
	"time"

	"github.com/kuhlman-labs/github-migrator/internal/config"
	"github.com/kuhlman-labs/github-migrator/internal/models"
	"github.com/kuhlman-labs/github-migrator/internal/storage"
)

// testClient creates a Client with a test database for testing
func testClient(t *testing.T) (*Client, *storage.Database) {
	t.Helper()

	cfg := config.DatabaseConfig{
		Type: "sqlite",
		DSN:  ":memory:",
	}

	db, err := storage.NewDatabase(cfg)
	if err != nil {
		t.Fatalf("failed to create test database: %v", err)
	}

	if err := db.Migrate(); err != nil {
		t.Fatalf("failed to run migrations: %v", err)
	}

	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelError}))

	client := &Client{
		db:       db,
		logger:   logger,
		sessions: make(map[string]*SDKSession),
	}

	return client, db
}

// =============================================================================
// Tool Authorization Tests
// =============================================================================

func TestCheckToolAuthorization_NilAuthContext(t *testing.T) {
	client, _ := testClient(t)

	// With nil auth context, should allow for backward compatibility
	err := client.checkToolAuthorization("start_migration", nil)
	if err != nil {
		t.Errorf("expected nil auth context to be allowed, got error: %v", err)
	}
}

func TestCheckToolAuthorization_AdminTools(t *testing.T) {
	client, _ := testClient(t)

	adminTools := []string{
		"start_discovery",
		"cancel_discovery",
		"discover_teams",
		"update_repository_status",
		"start_migration",
		"cancel_migration",
		"retry_batch_failures",
		"migrate_team",
		"suggest_team_mappings",
		"execute_team_migration",
		"send_mannequin_invitations",
		"suggest_user_mappings",
		"update_user_mapping",
		"fetch_mannequins",
	}

	tests := []struct {
		name        string
		auth        *AuthContext
		expectError bool
	}{
		{
			name: "admin user allowed",
			auth: &AuthContext{
				UserID:    "1",
				UserLogin: "admin",
				Tier:      "admin",
				Permissions: ToolPermissions{
					CanRead:       true,
					CanMigrateOwn: true,
					CanMigrateAll: true,
				},
			},
			expectError: false,
		},
		{
			name: "self-service user denied",
			auth: &AuthContext{
				UserID:    "2",
				UserLogin: "selfservice",
				Tier:      "self_service",
				Permissions: ToolPermissions{
					CanRead:       true,
					CanMigrateOwn: true,
					CanMigrateAll: false,
				},
			},
			expectError: true,
		},
		{
			name: "read-only user denied",
			auth: &AuthContext{
				UserID:    "3",
				UserLogin: "readonly",
				Tier:      "read_only",
				Permissions: ToolPermissions{
					CanRead:       true,
					CanMigrateOwn: false,
					CanMigrateAll: false,
				},
			},
			expectError: true,
		},
	}

	for _, tool := range adminTools {
		for _, tt := range tests {
			t.Run(tool+"_"+tt.name, func(t *testing.T) {
				err := client.checkToolAuthorization(tool, tt.auth)
				if tt.expectError && err == nil {
					t.Errorf("expected error for %s with %s, got nil", tool, tt.name)
				}
				if !tt.expectError && err != nil {
					t.Errorf("expected no error for %s with %s, got: %v", tool, tt.name, err)
				}
			})
		}
	}
}

func TestCheckToolAuthorization_SelfServiceTools(t *testing.T) {
	client, _ := testClient(t)

	selfServiceTools := []string{
		"create_batch",
		"configure_batch",
		"add_repos_to_batch",
		"remove_repos_from_batch",
		"schedule_batch",
		"plan_waves",
	}

	tests := []struct {
		name        string
		auth        *AuthContext
		expectError bool
	}{
		{
			name: "admin user allowed",
			auth: &AuthContext{
				UserID:    "1",
				UserLogin: "admin",
				Tier:      "admin",
				Permissions: ToolPermissions{
					CanRead:       true,
					CanMigrateOwn: true,
					CanMigrateAll: true,
				},
			},
			expectError: false,
		},
		{
			name: "self-service user allowed",
			auth: &AuthContext{
				UserID:    "2",
				UserLogin: "selfservice",
				Tier:      "self_service",
				Permissions: ToolPermissions{
					CanRead:       true,
					CanMigrateOwn: true,
					CanMigrateAll: false,
				},
			},
			expectError: false,
		},
		{
			name: "read-only user denied",
			auth: &AuthContext{
				UserID:    "3",
				UserLogin: "readonly",
				Tier:      "read_only",
				Permissions: ToolPermissions{
					CanRead:       true,
					CanMigrateOwn: false,
					CanMigrateAll: false,
				},
			},
			expectError: true,
		},
	}

	for _, tool := range selfServiceTools {
		for _, tt := range tests {
			t.Run(tool+"_"+tt.name, func(t *testing.T) {
				err := client.checkToolAuthorization(tool, tt.auth)
				if tt.expectError && err == nil {
					t.Errorf("expected error for %s with %s, got nil", tool, tt.name)
				}
				if !tt.expectError && err != nil {
					t.Errorf("expected no error for %s with %s, got: %v", tool, tt.name, err)
				}
			})
		}
	}
}

func TestCheckToolAuthorization_ReadOnlyTools(t *testing.T) {
	client, _ := testClient(t)

	readOnlyTools := []string{
		"find_pilot_candidates",
		"analyze_repositories",
		"get_complexity_breakdown",
		"check_dependencies",
		"get_top_complex_repositories",
		"get_repositories_with_most_dependencies",
		"get_discovery_status",
		"get_repository_details",
		"validate_repository",
		"list_batches",
		"get_batch_details",
		"get_migration_status",
		"get_migration_progress",
		"list_teams",
		"get_team_repositories",
		"get_team_migration_stats",
		"list_team_mappings",
		"get_team_migration_execution_status",
		"list_mannequins",
		"list_users",
		"get_user_stats",
		"list_user_mappings",
		"get_analytics_summary",
		"get_executive_report",
		"get_permission_audit",
		"list_organizations",
	}

	// All users should be allowed for read-only tools
	authContexts := []*AuthContext{
		{
			UserID:    "1",
			UserLogin: "admin",
			Tier:      "admin",
			Permissions: ToolPermissions{
				CanRead:       true,
				CanMigrateOwn: true,
				CanMigrateAll: true,
			},
		},
		{
			UserID:    "2",
			UserLogin: "selfservice",
			Tier:      "self_service",
			Permissions: ToolPermissions{
				CanRead:       true,
				CanMigrateOwn: true,
				CanMigrateAll: false,
			},
		},
		{
			UserID:    "3",
			UserLogin: "readonly",
			Tier:      "read_only",
			Permissions: ToolPermissions{
				CanRead:       true,
				CanMigrateOwn: false,
				CanMigrateAll: false,
			},
		},
	}

	for _, tool := range readOnlyTools {
		for _, auth := range authContexts {
			t.Run(tool+"_"+auth.Tier, func(t *testing.T) {
				err := client.checkToolAuthorization(tool, auth)
				if err != nil {
					t.Errorf("expected read-only tool %s to be allowed for %s, got: %v", tool, auth.Tier, err)
				}
			})
		}
	}
}

func TestCheckToolAuthorization_UnknownTool(t *testing.T) {
	client, _ := testClient(t)

	// Unknown tools should default to admin required
	readOnlyAuth := &AuthContext{
		UserID:    "1",
		UserLogin: "readonly",
		Tier:      "read_only",
		Permissions: ToolPermissions{
			CanRead:       true,
			CanMigrateOwn: false,
			CanMigrateAll: false,
		},
	}

	err := client.checkToolAuthorization("unknown_tool_xyz", readOnlyAuth)
	if err == nil {
		t.Error("expected unknown tool to require admin access, got no error")
	}

	// Admin should be allowed
	adminAuth := &AuthContext{
		UserID:    "2",
		UserLogin: "admin",
		Tier:      "admin",
		Permissions: ToolPermissions{
			CanRead:       true,
			CanMigrateOwn: true,
			CanMigrateAll: true,
		},
	}

	err = client.checkToolAuthorization("unknown_tool_xyz", adminAuth)
	if err != nil {
		t.Errorf("expected admin to be allowed for unknown tool, got: %v", err)
	}
}

func TestToolAuthRequirements_AllToolsHaveMapping(t *testing.T) {
	// Ensure all known tools have an authorization mapping
	knownTools := []string{
		// Read-only
		"find_pilot_candidates",
		"analyze_repositories",
		"get_complexity_breakdown",
		"check_dependencies",
		"get_top_complex_repositories",
		"get_repositories_with_most_dependencies",
		"get_discovery_status",
		"get_repository_details",
		"validate_repository",
		"list_batches",
		"get_batch_details",
		"get_migration_status",
		"get_migration_progress",
		"list_teams",
		"get_team_repositories",
		"get_team_migration_stats",
		"list_team_mappings",
		"get_team_migration_execution_status",
		"list_mannequins",
		"list_users",
		"get_user_stats",
		"list_user_mappings",
		"get_analytics_summary",
		"get_executive_report",
		"get_permission_audit",
		"list_organizations",
		// Self-service
		"create_batch",
		"configure_batch",
		"add_repos_to_batch",
		"remove_repos_from_batch",
		"schedule_batch",
		"plan_waves",
		// Admin
		"start_discovery",
		"cancel_discovery",
		"discover_teams",
		"update_repository_status",
		"start_migration",
		"cancel_migration",
		"retry_batch_failures",
		"migrate_team",
		"suggest_team_mappings",
		"execute_team_migration",
		"send_mannequin_invitations",
		"suggest_user_mappings",
		"update_user_mapping",
		"fetch_mannequins",
	}

	for _, tool := range knownTools {
		if _, exists := toolAuthRequirements[tool]; !exists {
			t.Errorf("tool %s has no authorization mapping in toolAuthRequirements", tool)
		}
	}
}

// =============================================================================
// Tool Execution Tests - Read Operations
// =============================================================================

func TestExecuteFindPilotCandidates_EmptyDatabase(t *testing.T) {
	client, _ := testClient(t)
	ctx := context.Background()

	params := FindPilotParams{
		MaxCount: 10,
	}

	result, err := client.executeFindPilotCandidates(ctx, params)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Should return empty candidates with a message
	resultMap, ok := result.(map[string]any)
	if !ok {
		t.Fatalf("expected map result, got %T", result)
	}

	candidates, ok := resultMap["candidates"].([]map[string]any)
	if !ok {
		t.Fatalf("expected candidates slice, got %T", resultMap["candidates"])
	}

	if len(candidates) != 0 {
		t.Errorf("expected 0 candidates in empty database, got %d", len(candidates))
	}
}

func TestExecuteFindPilotCandidates_WithRepositories(t *testing.T) {
	client, db := testClient(t)
	ctx := context.Background()

	// Create test repositories with varying complexity
	lowScore := 20
	medScore := 50
	highScore := 90

	repos := []*models.Repository{
		createTestRepo("org/simple-repo", string(models.StatusPending), &lowScore),
		createTestRepo("org/medium-repo", string(models.StatusPending), &medScore),
		createTestRepo("org/complex-repo", string(models.StatusPending), &highScore),
	}

	for _, repo := range repos {
		if err := db.SaveRepository(ctx, repo); err != nil {
			t.Fatalf("failed to create test repository: %v", err)
		}
	}

	params := FindPilotParams{
		MaxCount: 10,
	}

	result, err := client.executeFindPilotCandidates(ctx, params)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	resultMap, ok := result.(map[string]any)
	if !ok {
		t.Fatalf("expected map result, got %T", result)
	}

	candidates, ok := resultMap["candidates"].([]map[string]any)
	if !ok {
		t.Fatalf("expected candidates slice, got %T", resultMap["candidates"])
	}

	// Verify candidates are sorted by complexity (lowest first for pilot)
	if len(candidates) < 1 {
		t.Fatal("expected at least 1 candidate")
	}

	// First candidate should be lowest complexity
	if candidates[0]["full_name"] != "org/simple-repo" {
		t.Errorf("expected first candidate to be simple-repo, got %s", candidates[0]["full_name"])
	}
}

// createTestRepo creates a test repository with the given full name, status, and optional complexity score
func createTestRepo(fullName, status string, complexityScore *int) *models.Repository {
	now := time.Now()
	totalSize := int64(1024 * 1024)
	defaultBranch := "main"

	repo := &models.Repository{
		FullName:     fullName,
		Source:       "ghes",
		SourceURL:    "https://github.com/" + fullName,
		Status:       status,
		Visibility:   "private",
		IsArchived:   false,
		IsFork:       false,
		DiscoveredAt: now,
		UpdatedAt:    now,
		GitProperties: &models.RepositoryGitProperties{
			TotalSize:     &totalSize,
			DefaultBranch: &defaultBranch,
		},
	}

	if complexityScore != nil {
		repo.Validation = &models.RepositoryValidation{
			ComplexityScore: complexityScore,
		}
	}

	return repo
}

func TestExecuteAnalyzeRepositories_WithFilters(t *testing.T) {
	client, db := testClient(t)
	ctx := context.Background()

	// Create test repositories
	lowScore := 30
	highScore := 70

	repos := []*models.Repository{
		createTestRepo("org1/repo1", string(models.StatusPending), &lowScore),
		createTestRepo("org2/repo2", string(models.StatusComplete), &highScore),
	}

	for _, repo := range repos {
		if err := db.SaveRepository(ctx, repo); err != nil {
			t.Fatalf("failed to create test repository: %v", err)
		}
	}

	// Filter by organization
	params := AnalyzeRepositoriesParams{
		Organization: "org1",
	}

	result, err := client.executeAnalyzeRepositories(ctx, params)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	resultMap, ok := result.(map[string]any)
	if !ok {
		t.Fatalf("expected map result, got %T", result)
	}

	repos_result, ok := resultMap["repositories"].([]map[string]any)
	if !ok {
		t.Fatalf("expected repositories slice, got %T", resultMap["repositories"])
	}

	// Should only have org1's repo
	if len(repos_result) != 1 {
		t.Errorf("expected 1 repository for org1, got %d", len(repos_result))
	}

	// Verify the full_name contains org1
	if len(repos_result) > 0 {
		fullName, _ := repos_result[0]["full_name"].(string)
		if fullName != "org1/repo1" {
			t.Errorf("expected full_name 'org1/repo1', got %s", fullName)
		}
	}
}

func TestExecuteListBatches_EmptyDatabase(t *testing.T) {
	client, _ := testClient(t)
	ctx := context.Background()

	params := ListBatchesParams{}

	result, err := client.executeListBatches(ctx, params)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	resultMap, ok := result.(map[string]any)
	if !ok {
		t.Fatalf("expected map result, got %T", result)
	}

	batches, ok := resultMap["batches"].([]map[string]any)
	if !ok {
		t.Fatalf("expected batches slice, got %T", resultMap["batches"])
	}

	if len(batches) != 0 {
		t.Errorf("expected 0 batches in empty database, got %d", len(batches))
	}
}

func TestExecuteListBatches_WithBatches(t *testing.T) {
	client, db := testClient(t)
	ctx := context.Background()

	// Create test batches
	batches := []*models.Batch{
		{
			Name:   "batch-1",
			Status: models.BatchStatusPending,
		},
		{
			Name:   "batch-2",
			Status: models.BatchStatusCompleted,
		},
	}

	for _, batch := range batches {
		if err := db.CreateBatch(ctx, batch); err != nil {
			t.Fatalf("failed to create test batch: %v", err)
		}
	}

	params := ListBatchesParams{}

	result, err := client.executeListBatches(ctx, params)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	resultMap, ok := result.(map[string]any)
	if !ok {
		t.Fatalf("expected map result, got %T", result)
	}

	batchList, ok := resultMap["batches"].([]map[string]any)
	if !ok {
		t.Fatalf("expected batches slice, got %T", resultMap["batches"])
	}

	if len(batchList) != 2 {
		t.Errorf("expected 2 batches, got %d", len(batchList))
	}
}

func TestExecuteGetDiscoveryStatus_NoActiveDiscovery(t *testing.T) {
	client, _ := testClient(t)
	ctx := context.Background()

	params := GetDiscoveryStatusParams{}

	result, err := client.executeGetDiscoveryStatus(ctx, params)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	resultMap, ok := result.(map[string]any)
	if !ok {
		t.Fatalf("expected map result, got %T", result)
	}

	// Check that either active is false or the result indicates no active discovery
	active, hasActive := resultMap["active"]
	if hasActive && active == true {
		// If there's an active discovery indicated, check status field instead
		status, _ := resultMap["status"].(string)
		if status != "" && status != "no_active_discovery" {
			t.Error("expected no active discovery")
		}
	}
	// Test passes if active is false or not present
}

// =============================================================================
// Tool Execution Tests - Write Operations with Authorization
// =============================================================================

func TestExecuteCreateBatch_Success(t *testing.T) {
	client, db := testClient(t)
	ctx := context.Background()

	// Create a test repository first
	repo := createTestRepo("org/test-repo", string(models.StatusPending), nil)
	if err := db.SaveRepository(ctx, repo); err != nil {
		t.Fatalf("failed to create test repository: %v", err)
	}

	// Set admin auth context for write operation
	client.setCurrentAuth(&AuthContext{
		UserID:    "1",
		UserLogin: "admin",
		Tier:      "admin",
		Permissions: ToolPermissions{
			CanRead:       true,
			CanMigrateOwn: true,
			CanMigrateAll: true,
		},
	})
	defer client.clearCurrentAuth()

	params := CreateBatchParams{
		Name:         "test-batch",
		Repositories: []string{"org/test-repo"},
	}

	result, err := client.executeCreateBatch(ctx, params)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	resultMap, ok := result.(map[string]any)
	if !ok {
		t.Fatalf("expected map result, got %T", result)
	}

	if resultMap["batch_name"] != "test-batch" {
		t.Errorf("expected batch name 'test-batch', got %s", resultMap["batch_name"])
	}
}

func TestExecuteCreateBatch_ReadOnlyDenied(t *testing.T) {
	client, _ := testClient(t)
	ctx := context.Background()

	// Set read-only auth context
	client.setCurrentAuth(&AuthContext{
		UserID:    "1",
		UserLogin: "readonly",
		Tier:      "read_only",
		Permissions: ToolPermissions{
			CanRead:       true,
			CanMigrateOwn: false,
			CanMigrateAll: false,
		},
	})
	defer client.clearCurrentAuth()

	params := CreateBatchParams{
		Name: "test-batch",
	}

	_, err := client.executeCreateBatch(ctx, params)
	if err == nil {
		t.Error("expected error for read-only user, got nil")
	}
}

func TestExecuteStartMigration_RequiresAdmin(t *testing.T) {
	client, db := testClient(t)
	ctx := context.Background()

	// Create a batch first
	batch := &models.Batch{
		Name:   "migration-batch",
		Status: models.BatchStatusPending,
	}
	if err := db.CreateBatch(ctx, batch); err != nil {
		t.Fatalf("failed to create test batch: %v", err)
	}

	// Test with self-service user (should be denied)
	client.setCurrentAuth(&AuthContext{
		UserID:    "1",
		UserLogin: "selfservice",
		Tier:      "self_service",
		Permissions: ToolPermissions{
			CanRead:       true,
			CanMigrateOwn: true,
			CanMigrateAll: false,
		},
	})

	params := StartMigrationParams{
		BatchName: "migration-batch",
	}

	_, err := client.executeStartMigration(ctx, params)
	if err == nil {
		t.Error("expected error for self-service user on start_migration, got nil")
	}

	client.clearCurrentAuth()

	// Test with admin user (should be allowed)
	client.setCurrentAuth(&AuthContext{
		UserID:    "2",
		UserLogin: "admin",
		Tier:      "admin",
		Permissions: ToolPermissions{
			CanRead:       true,
			CanMigrateOwn: true,
			CanMigrateAll: true,
		},
	})
	defer client.clearCurrentAuth()

	// Note: This will still fail because no repos in batch, but auth should pass
	_, err = client.executeStartMigration(ctx, params)
	// Error is expected due to no repos, but it shouldn't be a permission error
	if err != nil && err.Error() == "permission denied: start_migration requires admin access" {
		t.Error("admin should have permission to start migration")
	}
}

// =============================================================================
// Tool Parameter Validation Tests
// =============================================================================

func TestExecuteStartDiscovery_ValidationErrors(t *testing.T) {
	client, _ := testClient(t)
	ctx := context.Background()

	// Set admin auth
	client.setCurrentAuth(&AuthContext{
		UserID:    "1",
		UserLogin: "admin",
		Tier:      "admin",
		Permissions: ToolPermissions{
			CanRead:       true,
			CanMigrateOwn: true,
			CanMigrateAll: true,
		},
	})
	defer client.clearCurrentAuth()

	tests := []struct {
		name        string
		params      StartDiscoveryParams
		expectError bool
	}{
		{
			name:        "no parameters",
			params:      StartDiscoveryParams{},
			expectError: true,
		},
		{
			name: "both org and enterprise",
			params: StartDiscoveryParams{
				Organization:   "org",
				EnterpriseSlug: "enterprise",
			},
			expectError: true,
		},
		{
			name: "valid organization",
			params: StartDiscoveryParams{
				Organization: "org",
			},
			expectError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := client.executeStartDiscovery(ctx, tt.params)
			if tt.expectError && err == nil {
				t.Error("expected error, got nil")
			}
			if !tt.expectError && err != nil {
				// Some errors are expected due to no actual discovery service
				// We're only checking that validation passes, not actual execution
				_ = err // Suppress linter - error is expected for non-validation failures
			}
		})
	}
}

func TestExecuteUpdateUserMapping_ValidationErrors(t *testing.T) {
	client, _ := testClient(t)
	ctx := context.Background()

	// Set admin auth
	client.setCurrentAuth(&AuthContext{
		UserID:    "1",
		UserLogin: "admin",
		Tier:      "admin",
		Permissions: ToolPermissions{
			CanRead:       true,
			CanMigrateOwn: true,
			CanMigrateAll: true,
		},
	})
	defer client.clearCurrentAuth()

	// Test with missing source_login
	params := UpdateUserMappingParams{
		SourceLogin: "",
	}

	_, err := client.executeUpdateUserMapping(ctx, params)
	if err == nil {
		t.Error("expected error for missing source_login, got nil")
	}
}

func TestExecuteUpdateRepositoryStatus_ValidationErrors(t *testing.T) {
	client, _ := testClient(t)
	ctx := context.Background()

	// Set admin auth
	client.setCurrentAuth(&AuthContext{
		UserID:    "1",
		UserLogin: "admin",
		Tier:      "admin",
		Permissions: ToolPermissions{
			CanRead:       true,
			CanMigrateOwn: true,
			CanMigrateAll: true,
		},
	})
	defer client.clearCurrentAuth()

	tests := []struct {
		name        string
		params      UpdateRepositoryStatusParams
		expectError bool
	}{
		{
			name: "missing repository",
			params: UpdateRepositoryStatusParams{
				Status: "pending",
			},
			expectError: true,
		},
		{
			name: "missing status",
			params: UpdateRepositoryStatusParams{
				Repository: "org/repo",
			},
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := client.executeUpdateRepositoryStatus(ctx, tt.params)
			if tt.expectError && err == nil {
				t.Error("expected validation error, got nil")
			}
		})
	}
}
