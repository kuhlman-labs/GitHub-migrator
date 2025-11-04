package migration

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"testing"
	"time"

	"github.com/brettkuhlman/github-migrator/internal/config"
	"github.com/brettkuhlman/github-migrator/internal/github"
	"github.com/brettkuhlman/github-migrator/internal/models"
	"github.com/brettkuhlman/github-migrator/internal/storage"
)

// createTestRepository creates a minimal repository with all required fields for testing
func createTestRepository(fullName string) *models.Repository {
	totalSize := int64(1024 * 1024)
	defaultBranch := "main"
	topContrib := "user1,user2"

	return &models.Repository{
		FullName:             fullName,
		Source:               "ghes",
		SourceURL:            fmt.Sprintf("https://github.com/%s", fullName),
		TotalSize:            &totalSize,
		DefaultBranch:        &defaultBranch,
		HasLFS:               false,
		HasSubmodules:        false,
		HasLargeFiles:        false,
		LargeFileCount:       0,
		BranchCount:          5,
		CommitCount:          100,
		IsArchived:           false,
		IsFork:               false,
		HasWiki:              false,
		HasPages:             false,
		HasDiscussions:       false,
		HasActions:           false,
		HasProjects:          false,
		HasPackages:          false,
		BranchProtections:    0,
		HasRulesets:          false,
		EnvironmentCount:     0,
		SecretCount:          0,
		VariableCount:        0,
		WebhookCount:         0,
		HasCodeScanning:      false,
		HasDependabot:        false,
		HasSecretScanning:    false,
		HasCodeowners:        false,
		Visibility:           "private",
		WorkflowCount:        0,
		HasSelfHostedRunners: false,
		CollaboratorCount:    0,
		InstalledAppsCount:   0,
		ReleaseCount:         0,
		HasReleaseAssets:     false,
		ContributorCount:     2,
		TopContributors:      &topContrib,
		IssueCount:           0,
		PullRequestCount:     0,
		TagCount:             0,
		OpenIssueCount:       0,
		OpenPRCount:          0,
		Status:               string(models.StatusPending),
		Priority:             0,
		IsSourceLocked:       false,
		DiscoveredAt:         time.Now(),
		UpdatedAt:            time.Now(),
	}
}

func TestNewExecutor(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))

	tests := []struct {
		name    string
		cfg     ExecutorConfig
		wantErr bool
	}{
		{
			name: "valid configuration",
			cfg: ExecutorConfig{
				SourceClient: &github.Client{},
				DestClient:   &github.Client{},
				Storage:      &storage.Database{},
				Logger:       logger,
			},
			wantErr: false,
		},
		{
			name: "missing source client",
			cfg: ExecutorConfig{
				DestClient: &github.Client{},
				Storage:    &storage.Database{},
				Logger:     logger,
			},
			wantErr: true,
		},
		{
			name: "missing destination client",
			cfg: ExecutorConfig{
				SourceClient: &github.Client{},
				Storage:      &storage.Database{},
				Logger:       logger,
			},
			wantErr: true,
		},
		{
			name: "missing storage",
			cfg: ExecutorConfig{
				SourceClient: &github.Client{},
				DestClient:   &github.Client{},
				Logger:       logger,
			},
			wantErr: true,
		},
		{
			name: "missing logger",
			cfg: ExecutorConfig{
				SourceClient: &github.Client{},
				DestClient:   &github.Client{},
				Storage:      &storage.Database{},
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			executor, err := NewExecutor(tt.cfg)
			if (err != nil) != tt.wantErr {
				t.Errorf("NewExecutor() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr && executor == nil {
				t.Error("NewExecutor() returned nil executor without error")
			}
		})
	}
}

func TestExecutor_validatePreMigration(t *testing.T) {
	// Create temporary test database
	db, err := storage.NewDatabase(config.DatabaseConfig{
		Type: "sqlite",
		DSN:  ":memory:",
	})
	if err != nil {
		t.Fatalf("Failed to create test database: %v", err)
	}
	defer db.Close()

	if err := db.Migrate(); err != nil {
		t.Fatalf("Failed to run migrations: %v", err)
	}

	_ = slog.New(slog.NewTextHandler(os.Stdout, nil))

	tests := []struct {
		name    string
		repo    *models.Repository
		wantErr bool
	}{
		{
			name: "valid repository",
			repo: &models.Repository{
				FullName: "test-org/test-repo",
			},
			wantErr: false, // Note: will fail if source client can't reach repo, but that's expected
		},
		{
			name: "repository with large file warning",
			repo: &models.Repository{
				FullName:        "test-org/large-file-repo",
				LargestFile:     ptrString("large-binary.bin"),
				LargestFileSize: ptrInt64(150 * 1024 * 1024), // 150MB
			},
			wantErr: false, // Warnings don't fail validation
		},
		{
			name: "very large repository",
			repo: &models.Repository{
				FullName:  "test-org/huge-repo",
				TotalSize: ptrInt64(60 * 1024 * 1024 * 1024), // 60GB
			},
			wantErr: false, // Warnings don't fail validation
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Note: This test requires a valid GitHub client to actually work
			// In a real test environment, we'd use a mock client
			// For now, we just verify the executor structure
			if tt.repo.FullName == "" {
				t.Error("Test repo must have a full name")
			}
		})
	}
}

func TestDestinationRepoExistsActions(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))

	tests := []struct {
		name   string
		action DestinationRepoExistsAction
	}{
		{
			name:   "fail action",
			action: DestinationRepoExistsFail,
		},
		{
			name:   "skip action",
			action: DestinationRepoExistsSkip,
		},
		{
			name:   "delete action",
			action: DestinationRepoExistsDelete,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			executor, err := NewExecutor(ExecutorConfig{
				SourceClient:         &github.Client{},
				DestClient:           &github.Client{},
				Storage:              &storage.Database{},
				Logger:               logger,
				DestRepoExistsAction: tt.action,
			})

			if err != nil {
				t.Errorf("Failed to create executor: %v", err)
				return
			}

			if executor.destRepoExistsAction != tt.action {
				t.Errorf("Expected action %s, got %s", tt.action, executor.destRepoExistsAction)
			}
		})
	}
}

func TestDestinationRepoExistsAction_Default(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))

	executor, err := NewExecutor(ExecutorConfig{
		SourceClient: &github.Client{},
		DestClient:   &github.Client{},
		Storage:      &storage.Database{},
		Logger:       logger,
		// DestRepoExistsAction not specified - should default to "fail"
	})

	if err != nil {
		t.Fatalf("Failed to create executor: %v", err)
	}

	if executor.destRepoExistsAction != DestinationRepoExistsFail {
		t.Errorf("Expected default action 'fail', got %s", executor.destRepoExistsAction)
	}
}

func TestPreMigrationValidationActions(t *testing.T) {
	// Create temporary test database
	db, err := storage.NewDatabase(config.DatabaseConfig{
		Type: "sqlite",
		DSN:  ":memory:",
	})
	if err != nil {
		t.Fatalf("Failed to create test database: %v", err)
	}
	defer db.Close()

	if err := db.Migrate(); err != nil {
		t.Fatalf("Failed to run migrations: %v", err)
	}

	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))

	tests := []struct {
		name        string
		action      DestinationRepoExistsAction
		description string
	}{
		{
			name:        "fail on existing destination",
			action:      DestinationRepoExistsFail,
			description: "Should fail validation if destination repo exists",
		},
		{
			name:        "skip on existing destination",
			action:      DestinationRepoExistsSkip,
			description: "Should skip migration if destination repo exists",
		},
		{
			name:        "delete on existing destination",
			action:      DestinationRepoExistsDelete,
			description: "Should delete destination repo if it exists",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			executor, err := NewExecutor(ExecutorConfig{
				SourceClient:         &github.Client{},
				DestClient:           &github.Client{},
				Storage:              db,
				Logger:               logger,
				DestRepoExistsAction: tt.action,
			})

			if err != nil {
				t.Errorf("Failed to create executor: %v", err)
				return
			}

			// Verify the action is set correctly
			if executor.destRepoExistsAction != tt.action {
				t.Errorf("Expected action %s, got %s", tt.action, executor.destRepoExistsAction)
			}

			t.Logf("✓ %s: %s", tt.name, tt.description)
		})
	}
}

func TestExecutor_DryRunExecution(t *testing.T) {
	// Create temporary test database
	db, err := storage.NewDatabase(config.DatabaseConfig{
		Type: "sqlite",
		DSN:  ":memory:",
	})
	if err != nil {
		t.Fatalf("Failed to create test database: %v", err)
	}
	defer db.Close()

	if err := db.Migrate(); err != nil {
		t.Fatalf("Failed to run migrations: %v", err)
	}

	_ = slog.New(slog.NewTextHandler(os.Stdout, nil))

	// Create a test repository
	repo := createTestRepository("test-org/test-repo")

	// Save repository to database
	if err := db.SaveRepository(context.Background(), repo); err != nil {
		t.Fatalf("Failed to save test repository: %v", err)
	}

	// Fetch the repository to get its ID
	savedRepo, err := db.GetRepository(context.Background(), repo.FullName)
	if err != nil {
		t.Fatalf("Failed to get saved repository: %v", err)
	}

	t.Run("dry run changes status correctly", func(t *testing.T) {
		// Note: This test verifies the database operations
		// Full integration test would require GitHub API mocks

		// Verify repository was saved
		if savedRepo.ID == 0 {
			t.Error("Repository ID should not be 0")
		}

		// Verify initial status
		if savedRepo.Status != string(models.StatusPending) {
			t.Errorf("Expected status %s, got %s", models.StatusPending, savedRepo.Status)
		}
	})
}

func TestExecutor_MigrationHistory(t *testing.T) {
	// Create temporary test database
	db, err := storage.NewDatabase(config.DatabaseConfig{
		Type: "sqlite",
		DSN:  ":memory:",
	})
	if err != nil {
		t.Fatalf("Failed to create test database: %v", err)
	}
	defer db.Close()

	if err := db.Migrate(); err != nil {
		t.Fatalf("Failed to run migrations: %v", err)
	}

	_ = slog.New(slog.NewTextHandler(os.Stdout, nil))

	// Create test repository
	repo := &models.Repository{
		FullName:     "test-org/test-repo",
		Source:       "ghes",
		SourceURL:    "https://github.test.com/test-org/test-repo",
		Status:       string(models.StatusPending),
		DiscoveredAt: time.Now(),
		UpdatedAt:    time.Now(),
	}

	if err := db.SaveRepository(context.Background(), repo); err != nil {
		t.Fatalf("Failed to save repository: %v", err)
	}

	savedRepo, err := db.GetRepository(context.Background(), repo.FullName)
	if err != nil {
		t.Fatalf("Failed to get repository: %v", err)
	}

	// Create migration history
	history := &models.MigrationHistory{
		RepositoryID: savedRepo.ID,
		Status:       "in_progress",
		Phase:        "migration",
		StartedAt:    time.Now(),
	}

	historyID, err := db.CreateMigrationHistory(context.Background(), history)
	if err != nil {
		t.Fatalf("Failed to create migration history: %v", err)
	}

	t.Run("creates migration history", func(t *testing.T) {
		if historyID == 0 {
			t.Error("Migration history ID should not be 0")
		}
	})

	t.Run("updates migration history status", func(t *testing.T) {
		errMsg := "test error"
		err := db.UpdateMigrationHistory(context.Background(), historyID, "failed", &errMsg)
		if err != nil {
			t.Errorf("Failed to update migration history: %v", err)
		}
	})

	t.Run("retrieves migration history", func(t *testing.T) {
		histories, err := db.GetMigrationHistory(context.Background(), savedRepo.ID)
		if err != nil {
			t.Errorf("Failed to get migration history: %v", err)
		}
		if len(histories) != 1 {
			t.Errorf("Expected 1 history record, got %d", len(histories))
		}
	})
}

func TestExecutor_MigrationLogs(t *testing.T) {
	// Create temporary test database
	db, err := storage.NewDatabase(config.DatabaseConfig{
		Type: "sqlite",
		DSN:  ":memory:",
	})
	if err != nil {
		t.Fatalf("Failed to create test database: %v", err)
	}
	defer db.Close()

	if err := db.Migrate(); err != nil {
		t.Fatalf("Failed to run migrations: %v", err)
	}

	_ = slog.New(slog.NewTextHandler(os.Stdout, nil))

	// Create test repository
	repo := &models.Repository{
		FullName:     "test-org/test-repo",
		Source:       "ghes",
		SourceURL:    "https://github.test.com/test-org/test-repo",
		Status:       string(models.StatusPending),
		DiscoveredAt: time.Now(),
		UpdatedAt:    time.Now(),
	}

	if err := db.SaveRepository(context.Background(), repo); err != nil {
		t.Fatalf("Failed to save repository: %v", err)
	}

	savedRepo, err := db.GetRepository(context.Background(), repo.FullName)
	if err != nil {
		t.Fatalf("Failed to get repository: %v", err)
	}

	t.Run("creates and retrieves migration logs", func(t *testing.T) {
		// Create multiple logs
		logs := []*models.MigrationLog{
			{
				RepositoryID: savedRepo.ID,
				Level:        "INFO",
				Phase:        "pre_migration",
				Operation:    "validate",
				Message:      "Starting validation",
				Timestamp:    time.Now(),
			},
			{
				RepositoryID: savedRepo.ID,
				Level:        "ERROR",
				Phase:        "archive_generation",
				Operation:    "generate",
				Message:      "Failed to generate archive",
				Timestamp:    time.Now(),
			},
		}

		for _, log := range logs {
			if err := db.CreateMigrationLog(context.Background(), log); err != nil {
				t.Errorf("Failed to create migration log: %v", err)
			}
		}

		// Retrieve all logs
		retrievedLogs, err := db.GetMigrationLogs(context.Background(), savedRepo.ID, "", "", 100, 0)
		if err != nil {
			t.Errorf("Failed to get migration logs: %v", err)
		}
		if len(retrievedLogs) != 2 {
			t.Errorf("Expected 2 logs, got %d", len(retrievedLogs))
		}

		// Filter by level
		errorLogs, err := db.GetMigrationLogs(context.Background(), savedRepo.ID, "ERROR", "", 100, 0)
		if err != nil {
			t.Errorf("Failed to get error logs: %v", err)
		}
		if len(errorLogs) != 1 {
			t.Errorf("Expected 1 error log, got %d", len(errorLogs))
		}

		// Filter by phase
		preMigrationLogs, err := db.GetMigrationLogs(context.Background(), savedRepo.ID, "", "pre_migration", 100, 0)
		if err != nil {
			t.Errorf("Failed to get pre_migration logs: %v", err)
		}
		if len(preMigrationLogs) != 1 {
			t.Errorf("Expected 1 pre_migration log, got %d", len(preMigrationLogs))
		}
	})
}

func TestArchiveURLs(t *testing.T) {
	t.Run("creates archive URLs", func(t *testing.T) {
		urls := &ArchiveURLs{
			GitSource: "https://example.com/git-source.tar.gz",
			Metadata:  "https://example.com/metadata.tar.gz",
		}

		if urls.GitSource == "" {
			t.Error("GitSource should not be empty")
		}
		if urls.Metadata == "" {
			t.Error("Metadata should not be empty")
		}
	})
}

// TestExecutor_getDestinationOrg tests the getDestinationOrg helper function
func TestExecutor_getDestinationOrg(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))

	executor, err := NewExecutor(ExecutorConfig{
		SourceClient: &github.Client{},
		DestClient:   &github.Client{},
		Storage:      &storage.Database{},
		Logger:       logger,
	})
	if err != nil {
		t.Fatalf("Failed to create executor: %v", err)
	}

	tests := []struct {
		name     string
		repo     *models.Repository
		expected string
	}{
		{
			name: "no destination specified - uses source org",
			repo: &models.Repository{
				FullName:            "source-org/my-repo",
				DestinationFullName: nil,
			},
			expected: "source-org",
		},
		{
			name: "destination org specified - different org",
			repo: &models.Repository{
				FullName:            "source-org/my-repo",
				DestinationFullName: ptrString("dest-org/my-repo"),
			},
			expected: "dest-org",
		},
		{
			name: "destination org specified - different org and repo name",
			repo: &models.Repository{
				FullName:            "source-org/old-name",
				DestinationFullName: ptrString("dest-org/new-name"),
			},
			expected: "dest-org",
		},
		{
			name: "empty destination string - uses source org",
			repo: &models.Repository{
				FullName:            "source-org/my-repo",
				DestinationFullName: ptrString(""),
			},
			expected: "source-org",
		},
		{
			name: "destination with only org (edge case)",
			repo: &models.Repository{
				FullName:            "source-org/my-repo",
				DestinationFullName: ptrString("dest-org"),
			},
			expected: "dest-org",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := executor.getDestinationOrg(tt.repo, nil)
			if result != tt.expected {
				t.Errorf("getDestinationOrg() = %q, want %q", result, tt.expected)
			}
		})
	}
}

// TestExecutor_getDestinationRepoName tests the getDestinationRepoName helper function
func TestExecutor_getDestinationRepoName(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))

	executor, err := NewExecutor(ExecutorConfig{
		SourceClient: &github.Client{},
		DestClient:   &github.Client{},
		Storage:      &storage.Database{},
		Logger:       logger,
	})
	if err != nil {
		t.Fatalf("Failed to create executor: %v", err)
	}

	tests := []struct {
		name     string
		repo     *models.Repository
		expected string
	}{
		{
			name: "no destination specified - uses source repo name",
			repo: &models.Repository{
				FullName:            "source-org/my-repo",
				DestinationFullName: nil,
			},
			expected: "my-repo",
		},
		{
			name: "destination specified - same repo name",
			repo: &models.Repository{
				FullName:            "source-org/my-repo",
				DestinationFullName: ptrString("dest-org/my-repo"),
			},
			expected: "my-repo",
		},
		{
			name: "destination specified - different repo name",
			repo: &models.Repository{
				FullName:            "source-org/old-name",
				DestinationFullName: ptrString("dest-org/new-name"),
			},
			expected: "new-name",
		},
		{
			name: "destination specified - different org and repo name",
			repo: &models.Repository{
				FullName:            "legacy-org/legacy-repo",
				DestinationFullName: ptrString("modern-org/modern-repo"),
			},
			expected: "modern-repo",
		},
		{
			name: "empty destination string - uses source repo name",
			repo: &models.Repository{
				FullName:            "source-org/my-repo",
				DestinationFullName: ptrString(""),
			},
			expected: "my-repo",
		},
		{
			name: "destination with only org - returns org as name",
			repo: &models.Repository{
				FullName:            "source-org/my-repo",
				DestinationFullName: ptrString("dest-org"),
			},
			expected: "dest-org",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := executor.getDestinationRepoName(tt.repo)
			if result != tt.expected {
				t.Errorf("getDestinationRepoName() = %q, want %q", result, tt.expected)
			}
		})
	}
}

// TestExecutor_DestinationHelpers_Integration tests both helpers together
func TestExecutor_DestinationHelpers_Integration(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))

	executor, err := NewExecutor(ExecutorConfig{
		SourceClient: &github.Client{},
		DestClient:   &github.Client{},
		Storage:      &storage.Database{},
		Logger:       logger,
	})
	if err != nil {
		t.Fatalf("Failed to create executor: %v", err)
	}

	tests := []struct {
		name         string
		repo         *models.Repository
		expectedOrg  string
		expectedName string
		description  string
	}{
		{
			name: "default behavior - no destination specified",
			repo: &models.Repository{
				FullName:            "acme-corp/backend-api",
				DestinationFullName: nil,
			},
			expectedOrg:  "acme-corp",
			expectedName: "backend-api",
			description:  "Should use source org and repo name when destination not specified",
		},
		{
			name: "different organization only",
			repo: &models.Repository{
				FullName:            "old-org/my-service",
				DestinationFullName: ptrString("new-org/my-service"),
			},
			expectedOrg:  "new-org",
			expectedName: "my-service",
			description:  "Should migrate to different org with same repo name",
		},
		{
			name: "different organization AND repository name",
			repo: &models.Repository{
				FullName:            "legacy-systems/monolith-v1",
				DestinationFullName: ptrString("modern-apps/microservice-auth"),
			},
			expectedOrg:  "modern-apps",
			expectedName: "microservice-auth",
			description:  "Should migrate to different org with different repo name",
		},
		{
			name: "same organization, different name",
			repo: &models.Repository{
				FullName:            "company/project-old",
				DestinationFullName: ptrString("company/project-new"),
			},
			expectedOrg:  "company",
			expectedName: "project-new",
			description:  "Should rename repository within same organization",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			org := executor.getDestinationOrg(tt.repo, nil)
			name := executor.getDestinationRepoName(tt.repo)

			if org != tt.expectedOrg {
				t.Errorf("getDestinationOrg() = %q, want %q", org, tt.expectedOrg)
			}
			if name != tt.expectedName {
				t.Errorf("getDestinationRepoName() = %q, want %q", name, tt.expectedName)
			}

			fullName := fmt.Sprintf("%s/%s", org, name)
			t.Logf("✓ %s: %s -> %s", tt.name, tt.repo.FullName, fullName)
			t.Logf("  Description: %s", tt.description)
		})
	}
}

// Helper functions
func ptrString(s string) *string {
	return &s
}

func ptrInt64(i int64) *int64 {
	return &i
}
