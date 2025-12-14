package handlers

import (
	"context"
	"testing"

	"github.com/kuhlman-labs/github-migrator/internal/auth"
	"github.com/kuhlman-labs/github-migrator/internal/models"
)

func TestGetInitiatingUser(t *testing.T) {
	tests := []struct {
		name         string
		setupContext func() context.Context
		expectedUser *string
		expectNil    bool
	}{
		{
			name: "returns username when user is in context",
			setupContext: func() context.Context {
				ctx := context.Background()
				user := &auth.GitHubUser{
					Login: "testuser",
					ID:    123,
				}
				return context.WithValue(ctx, auth.ContextKeyUser, user)
			},
			expectedUser: stringPtr("testuser"),
			expectNil:    false,
		},
		{
			name: "returns nil when no user in context",
			setupContext: func() context.Context {
				return context.Background()
			},
			expectedUser: nil,
			expectNil:    true,
		},
		{
			name: "returns nil when context value is not a GitHubUser",
			setupContext: func() context.Context {
				ctx := context.Background()
				return context.WithValue(ctx, auth.ContextKeyUser, "invalid")
			},
			expectedUser: nil,
			expectNil:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := tt.setupContext()
			result := getInitiatingUser(ctx)

			if tt.expectNil {
				if result != nil {
					t.Errorf("expected nil, got %v", *result)
				}
			} else {
				if result == nil {
					t.Error("expected non-nil result")
				} else if *result != *tt.expectedUser {
					t.Errorf("expected %s, got %s", *tt.expectedUser, *result)
				}
			}
		})
	}
}

// TestDirectMigrationLogging tests creating migration logs directly with user info
func TestDirectMigrationLogging(t *testing.T) {
	// Setup test database using the standard test setup
	db := setupTestDB(t)
	defer db.Close()

	// Create test repository
	repo := &models.Repository{
		FullName:  "test-org/direct-log-test",
		Source:    "github",
		SourceURL: "https://github.com",
		Status:    "pending",
	}

	ctx := context.Background()
	if err := db.SaveRepository(ctx, repo); err != nil {
		t.Fatalf("Failed to save repo: %v", err)
	}

	savedRepo, _ := db.GetRepository(ctx, "test-org/direct-log-test")

	tests := []struct {
		name              string
		setupContext      func() context.Context
		expectedInitiator *string
		operation         string
		phase             string
		message           string
	}{
		{
			name: "logs with authenticated user",
			setupContext: func() context.Context {
				user := &auth.GitHubUser{
					Login: "migration-admin",
					ID:    456,
				}
				return context.WithValue(context.Background(), auth.ContextKeyUser, user)
			},
			expectedInitiator: stringPtr("migration-admin"),
			operation:         "queue",
			phase:             "migration",
			message:           "Migration queued by user",
		},
		{
			name: "logs without user when not authenticated",
			setupContext: func() context.Context {
				return context.Background()
			},
			expectedInitiator: nil,
			operation:         "queue",
			phase:             "dry_run",
			message:           "Dry run queued",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := tt.setupContext()
			initiator := getInitiatingUser(ctx)

			logEntry := &models.MigrationLog{
				RepositoryID: savedRepo.ID,
				Level:        "INFO",
				Phase:        tt.phase,
				Operation:    tt.operation,
				Message:      tt.message,
				InitiatedBy:  initiator,
			}

			if err := db.CreateMigrationLog(ctx, logEntry); err != nil {
				t.Fatalf("Failed to create log: %v", err)
			}

			// Verify log was created with correct initiator
			logs, err := db.GetMigrationLogs(context.Background(), savedRepo.ID, "", tt.phase, 10, 0)
			if err != nil {
				t.Fatalf("Failed to get logs: %v", err)
			}

			// Find our log entry
			var foundLog *models.MigrationLog
			for _, log := range logs {
				if log.Operation == tt.operation && log.Message == tt.message {
					foundLog = log
					break
				}
			}

			if foundLog == nil {
				t.Fatal("Expected to find log entry")
			}

			// Verify initiator
			if tt.expectedInitiator == nil {
				if foundLog.InitiatedBy != nil {
					t.Errorf("Expected nil initiator, got %v", *foundLog.InitiatedBy)
				}
			} else {
				if foundLog.InitiatedBy == nil {
					t.Error("Expected initiator to be set, got nil")
				} else if *foundLog.InitiatedBy != *tt.expectedInitiator {
					t.Errorf("Expected initiator %s, got %s", *tt.expectedInitiator, *foundLog.InitiatedBy)
				}
			}
		})
	}
}

// TestMigrationLogWithoutHistoryID tests creating logs without history linkage
func TestMigrationLogWithoutHistoryID(t *testing.T) {
	// Setup test database using the standard test setup
	db := setupTestDB(t)
	defer db.Close()

	ctx := context.Background()

	// Create test repository
	repo := &models.Repository{
		FullName:  "test-org/history-test",
		Source:    "github",
		SourceURL: "https://github.com",
		Status:    "pending",
	}
	if err := db.SaveRepository(ctx, repo); err != nil {
		t.Fatalf("Failed to save repository: %v", err)
	}

	savedRepo, _ := db.GetRepository(ctx, "test-org/history-test")

	// Setup authenticated context
	user := &auth.GitHubUser{
		Login: "batch-admin",
		ID:    789,
	}
	authCtx := context.WithValue(context.Background(), auth.ContextKeyUser, user)
	initiator := getInitiatingUser(authCtx)

	// Create a log entry with initiated_by but without history_id
	logEntry := &models.MigrationLog{
		RepositoryID: savedRepo.ID,
		Level:        "INFO",
		Phase:        "migration",
		Operation:    "batch_start",
		Message:      "Migration started via batch",
		InitiatedBy:  initiator,
	}

	if err := db.CreateMigrationLog(authCtx, logEntry); err != nil {
		t.Fatalf("Failed to create log: %v", err)
	}

	// Verify log was created with user
	logs, err := db.GetMigrationLogs(context.Background(), savedRepo.ID, "", "", 10, 0)
	if err != nil {
		t.Fatalf("Failed to get logs: %v", err)
	}

	if len(logs) == 0 {
		t.Fatal("Expected to find log entry")
	}

	foundLog := logs[0]
	if foundLog.InitiatedBy == nil {
		t.Fatal("Expected initiator to be set")
	}

	if *foundLog.InitiatedBy != "batch-admin" {
		t.Errorf("Expected initiator 'batch-admin', got '%s'", *foundLog.InitiatedBy)
	}
}

func TestMigrationLogging_DifferentOperations(t *testing.T) {
	// Setup test database using the standard test setup
	db := setupTestDB(t)
	defer db.Close()

	ctx := context.Background()

	// Create test repository
	repo := &models.Repository{
		FullName:  "test-org/operations-test",
		Source:    "github",
		SourceURL: "https://github.com",
		Status:    "pending",
	}
	if err := db.SaveRepository(ctx, repo); err != nil {
		t.Fatalf("Failed to save repository: %v", err)
	}

	savedRepo, _ := db.GetRepository(ctx, "test-org/operations-test")

	// Setup authenticated context
	user := &auth.GitHubUser{
		Login: "testuser",
		ID:    999,
	}
	authCtx := context.WithValue(context.Background(), auth.ContextKeyUser, user)

	testCases := []struct {
		operation     string
		phase         string
		message       string
		expectedLevel string
	}{
		{
			operation:     "queue",
			phase:         "migration",
			message:       "Migration queued",
			expectedLevel: "INFO",
		},
		{
			operation:     "queue",
			phase:         "dry_run",
			message:       "Dry run queued",
			expectedLevel: "INFO",
		},
		{
			operation:     "retry",
			phase:         "migration",
			message:       "Migration retry queued",
			expectedLevel: "INFO",
		},
		{
			operation:     "batch_start",
			phase:         "migration",
			message:       "Migration started via batch Test",
			expectedLevel: "INFO",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.operation+"_"+tc.phase, func(t *testing.T) {
			// Create log entry
			initiator := getInitiatingUser(authCtx)
			logEntry := &models.MigrationLog{
				RepositoryID: savedRepo.ID,
				Level:        tc.expectedLevel,
				Phase:        tc.phase,
				Operation:    tc.operation,
				Message:      tc.message,
				InitiatedBy:  initiator,
			}

			if err := db.CreateMigrationLog(authCtx, logEntry); err != nil {
				t.Fatalf("Failed to create log: %v", err)
			}

			// Verify log was created correctly
			logs, err := db.GetMigrationLogs(context.Background(), savedRepo.ID, "", tc.phase, 100, 0)
			if err != nil {
				t.Fatalf("Failed to get logs: %v", err)
			}

			var found *models.MigrationLog
			for _, log := range logs {
				if log.Operation == tc.operation && log.Phase == tc.phase {
					found = log
					break
				}
			}

			if found == nil {
				t.Fatalf("Log not found for operation=%s phase=%s", tc.operation, tc.phase)
			}

			// Verify all fields
			if found.Level != tc.expectedLevel {
				t.Errorf("Expected level %s, got %s", tc.expectedLevel, found.Level)
			}
			if found.Message != tc.message {
				t.Errorf("Expected message '%s', got '%s'", tc.message, found.Message)
			}
			if found.InitiatedBy == nil {
				t.Error("Expected initiator to be set")
			} else if *found.InitiatedBy != "testuser" {
				t.Errorf("Expected initiator 'testuser', got '%s'", *found.InitiatedBy)
			}
		})
	}
}

func TestMigrationLogging_SystemVsUser(t *testing.T) {
	// Setup test database using the standard test setup
	db := setupTestDB(t)
	defer db.Close()

	ctx := context.Background()

	// Create test repository
	repo := &models.Repository{
		FullName:  "test-org/system-test",
		Source:    "github",
		SourceURL: "https://github.com",
		Status:    "pending",
	}
	if err := db.SaveRepository(ctx, repo); err != nil {
		t.Fatalf("Failed to save repository: %v", err)
	}

	savedRepo, _ := db.GetRepository(ctx, "test-org/system-test")

	tests := []struct {
		name              string
		context           context.Context
		expectedInitiator *string
		description       string
	}{
		{
			name:              "system initiated action",
			context:           context.Background(),
			expectedInitiator: nil,
			description:       "Actions without user context are system-initiated",
		},
		{
			name: "user initiated action",
			context: func() context.Context {
				user := &auth.GitHubUser{Login: "john-doe", ID: 123}
				return context.WithValue(context.Background(), auth.ContextKeyUser, user)
			}(),
			expectedInitiator: stringPtr("john-doe"),
			description:       "Actions with user context are user-initiated",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			initiator := getInitiatingUser(tt.context)

			logEntry := &models.MigrationLog{
				RepositoryID: savedRepo.ID,
				Level:        "INFO",
				Phase:        "test",
				Operation:    tt.name,
				Message:      tt.description,
				InitiatedBy:  initiator,
			}

			if err := db.CreateMigrationLog(tt.context, logEntry); err != nil {
				t.Fatalf("Failed to create log: %v", err)
			}

			// Verify the log
			logs, err := db.GetMigrationLogs(context.Background(), savedRepo.ID, "", "", 100, 0)
			if err != nil {
				t.Fatalf("Failed to get logs: %v", err)
			}

			var found *models.MigrationLog
			for _, log := range logs {
				if log.Operation == tt.name {
					found = log
					break
				}
			}

			if found == nil {
				t.Fatal("Log not found")
			}

			if tt.expectedInitiator == nil {
				if found.InitiatedBy != nil {
					t.Errorf("Expected nil initiator, got %v", *found.InitiatedBy)
				}
			} else {
				if found.InitiatedBy == nil {
					t.Error("Expected initiator to be set")
				} else if *found.InitiatedBy != *tt.expectedInitiator {
					t.Errorf("Expected initiator '%s', got '%s'", *tt.expectedInitiator, *found.InitiatedBy)
				}
			}
		})
	}
}

// Helper functions - stringPtr is defined in handlers.go
