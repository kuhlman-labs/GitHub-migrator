// Package testdata provides test fixtures and helper functions for storage layer tests
package testdata

import (
	"fmt"
	"time"

	"github.com/kuhlman-labs/github-migrator/internal/models"
)

// Helper functions for pointer creation
func StringPtr(s string) *string { return &s }
func Int64Ptr(i int64) *int64    { return &i }
func IntPtr(i int) *int          { return &i }
func BoolPtr(b bool) *bool       { return &b }

// CreateTestRepository creates a complete test repository with all related data
func CreateTestRepository(fullName string) *models.Repository {
	totalSize := int64(1024 * 1024) // 1MB
	branch := "main"
	now := time.Now()

	return &models.Repository{
		FullName:        fullName,
		Source:          "ghes",
		SourceURL:       fmt.Sprintf("https://github.com/%s", fullName),
		Status:          string(models.StatusPending),
		Visibility:      "private",
		IsArchived:      false,
		IsFork:          false,
		DiscoveredAt:    now,
		UpdatedAt:       now,
		LastDiscoveryAt: &now,
		GitProperties: &models.RepositoryGitProperties{
			TotalSize:          &totalSize,
			DefaultBranch:      &branch,
			BranchCount:        5,
			CommitCount:        100,
			CommitsLast12Weeks: 25,
			HasLFS:             false,
			HasSubmodules:      false,
			HasLargeFiles:      false,
		},
		Features: &models.RepositoryFeatures{
			HasWiki:        false,
			HasPages:       false,
			HasActions:     true,
			HasPackages:    false,
			HasProjects:    false,
			HasRulesets:    false,
			IssueCount:     10,
			OpenIssueCount: 3,
		},
		Validation: &models.RepositoryValidation{
			ComplexityScore: IntPtr(5),
		},
	}
}

// CreateTestRepositoryMinimal creates a minimal test repository with just core fields
func CreateTestRepositoryMinimal(fullName string) *models.Repository {
	now := time.Now()
	return &models.Repository{
		FullName:     fullName,
		Source:       "ghes",
		SourceURL:    fmt.Sprintf("https://github.com/%s", fullName),
		Status:       string(models.StatusPending),
		Visibility:   "private",
		DiscoveredAt: now,
		UpdatedAt:    now,
	}
}

// CreateTestADORepository creates a test Azure DevOps repository
func CreateTestADORepository(fullName, project string, isGit bool) *models.Repository {
	now := time.Now()

	status := string(models.StatusPending)
	if !isGit {
		status = string(models.StatusRemediationRequired)
	}

	return &models.Repository{
		FullName:        fullName,
		Source:          "azuredevops",
		SourceURL:       fmt.Sprintf("https://dev.azure.com/%s", fullName),
		Status:          status,
		Visibility:      "private",
		DiscoveredAt:    now,
		UpdatedAt:       now,
		LastDiscoveryAt: &now,
		ADOProperties: &models.RepositoryADOProperties{
			Project:       &project,
			IsGit:         isGit,
			HasBoards:     true,
			HasPipelines:  true,
			PipelineCount: 3,
		},
	}
}

// CreateTestRepositoryWithSize creates a test repository with specific size
func CreateTestRepositoryWithSize(fullName string, sizeBytes int64) *models.Repository {
	repo := CreateTestRepositoryMinimal(fullName)
	repo.GitProperties = &models.RepositoryGitProperties{
		TotalSize: &sizeBytes,
	}
	return repo
}

// CreateTestRepositoryWithComplexity creates a test repository with specific complexity score
func CreateTestRepositoryWithComplexity(fullName string, score int) *models.Repository {
	repo := CreateTestRepositoryMinimal(fullName)
	repo.Validation = &models.RepositoryValidation{
		ComplexityScore: &score,
	}
	return repo
}

// CreateTestRepositoryWithLFS creates a test repository with LFS enabled
func CreateTestRepositoryWithLFS(fullName string) *models.Repository {
	repo := CreateTestRepository(fullName)
	repo.GitProperties.HasLFS = true
	return repo
}

// CreateTestRepositoryWithActions creates a test repository with GitHub Actions enabled
func CreateTestRepositoryWithActions(fullName string) *models.Repository {
	repo := CreateTestRepository(fullName)
	repo.Features.HasActions = true
	repo.Features.WorkflowCount = 5
	return repo
}

// CreateTestRepositoryWithValidationIssues creates a test repository with migration blockers
func CreateTestRepositoryWithValidationIssues(fullName string) *models.Repository {
	repo := CreateTestRepository(fullName)
	repo.Validation = &models.RepositoryValidation{
		HasOversizedCommits:    true,
		OversizedCommitDetails: StringPtr(`[{"sha": "abc123", "size": 3000000000}]`),
		HasBlockingFiles:       true,
		BlockingFileDetails:    StringPtr(`[{"path": "large.bin", "size": 500000000}]`),
		ComplexityScore:        IntPtr(25),
	}
	return repo
}

// CreateTestRepositoryInBatch creates a test repository assigned to a batch
func CreateTestRepositoryInBatch(fullName string, batchID int64) *models.Repository {
	repo := CreateTestRepository(fullName)
	repo.BatchID = &batchID
	repo.Priority = 1
	return repo
}

// CreateTestRepositoryMigrated creates a test repository that has been migrated
func CreateTestRepositoryMigrated(fullName string) *models.Repository {
	now := time.Now()
	destURL := fmt.Sprintf("https://github.com/%s", fullName)

	repo := CreateTestRepository(fullName)
	repo.Status = string(models.StatusComplete)
	repo.DestinationURL = &destURL
	repo.DestinationFullName = &fullName
	repo.MigratedAt = &now
	return repo
}

// CreateTestBatch creates a test batch
func CreateTestBatch(name string) *models.Batch {
	now := time.Now()
	return &models.Batch{
		Name:         name,
		Description:  StringPtr("Test batch"),
		Type:         "pilot",
		Status:       "pending",
		MigrationAPI: models.MigrationAPIGEI,
		CreatedAt:    now,
	}
}

// CreateTestBatchWithStatus creates a test batch with specific status
func CreateTestBatchWithStatus(name, status string) *models.Batch {
	batch := CreateTestBatch(name)
	batch.Status = status
	return batch
}
