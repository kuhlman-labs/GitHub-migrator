package models

import (
	"testing"
	"time"
)

func ptrString(s string) *string {
	return &s
}

func ptrInt64(i int64) *int64 {
	return &i
}

func TestRepository_GetGitProperties(t *testing.T) {
	now := time.Now()
	repo := &Repository{
		TotalSize:          ptrInt64(1000),
		LargestFile:        ptrString("large.bin"),
		LargestFileSize:    ptrInt64(500),
		LargestCommit:      ptrString("abc123"),
		LargestCommitSize:  ptrInt64(200),
		HasLFS:             true,
		HasSubmodules:      true,
		HasLargeFiles:      true,
		LargeFileCount:     5,
		DefaultBranch:      ptrString("main"),
		BranchCount:        10,
		CommitCount:        100,
		CommitsLast12Weeks: 50,
		LastCommitSHA:      ptrString("def456"),
		LastCommitDate:     &now,
	}

	git := repo.GetGitProperties()

	if *git.TotalSize != 1000 {
		t.Errorf("TotalSize = %d, want 1000", *git.TotalSize)
	}
	if *git.LargestFile != "large.bin" {
		t.Errorf("LargestFile = %q, want %q", *git.LargestFile, "large.bin")
	}
	if !git.HasLFS {
		t.Error("HasLFS should be true")
	}
	if !git.HasSubmodules {
		t.Error("HasSubmodules should be true")
	}
	if git.BranchCount != 10 {
		t.Errorf("BranchCount = %d, want 10", git.BranchCount)
	}
	if git.CommitCount != 100 {
		t.Errorf("CommitCount = %d, want 100", git.CommitCount)
	}
}

func TestRepository_GetGitHubFeatures(t *testing.T) {
	repo := &Repository{
		IsArchived:        true,
		IsFork:            true,
		HasWiki:           true,
		HasPages:          true,
		HasDiscussions:    true,
		HasActions:        true,
		HasProjects:       true,
		HasPackages:       true,
		BranchProtections: 5,
		HasRulesets:       true,
	}

	features := repo.GetGitHubFeatures()

	if !features.IsArchived {
		t.Error("IsArchived should be true")
	}
	if !features.IsFork {
		t.Error("IsFork should be true")
	}
	if !features.HasWiki {
		t.Error("HasWiki should be true")
	}
	if !features.HasPages {
		t.Error("HasPages should be true")
	}
	if !features.HasDiscussions {
		t.Error("HasDiscussions should be true")
	}
	if !features.HasActions {
		t.Error("HasActions should be true")
	}
	if !features.HasProjects {
		t.Error("HasProjects should be true")
	}
	if !features.HasPackages {
		t.Error("HasPackages should be true")
	}
	if features.BranchProtections != 5 {
		t.Errorf("BranchProtections = %d, want 5", features.BranchProtections)
	}
	if !features.HasRulesets {
		t.Error("HasRulesets should be true")
	}
}

func TestRepository_GetSecurityFeatures(t *testing.T) {
	repo := &Repository{
		HasCodeScanning:   true,
		HasDependabot:     true,
		HasSecretScanning: true,
		HasCodeowners:     true,
	}

	security := repo.GetSecurityFeatures()

	if !security.HasCodeScanning {
		t.Error("HasCodeScanning should be true")
	}
	if !security.HasDependabot {
		t.Error("HasDependabot should be true")
	}
	if !security.HasSecretScanning {
		t.Error("HasSecretScanning should be true")
	}
	if !security.HasCodeowners {
		t.Error("HasCodeowners should be true")
	}
}

func TestRepository_GetMigrationState(t *testing.T) {
	repo := &Repository{
		Status:              string(StatusPending),
		BatchID:             ptrInt64(1),
		Priority:            5,
		DestinationURL:      ptrString("https://github.com/dest/repo"),
		DestinationFullName: ptrString("dest/repo"),
		SourceMigrationID:   ptrInt64(123),
		IsSourceLocked:      true,
	}

	state := repo.GetMigrationState()

	if state.Status != string(StatusPending) {
		t.Errorf("Status = %q, want %q", state.Status, string(StatusPending))
	}
	if *state.BatchID != 1 {
		t.Errorf("BatchID = %d, want 1", *state.BatchID)
	}
	if state.Priority != 5 {
		t.Errorf("Priority = %d, want 5", state.Priority)
	}
	if !state.IsSourceLocked {
		t.Error("IsSourceLocked should be true")
	}
}

func TestRepository_GetMigrationExclusions(t *testing.T) {
	repo := &Repository{
		ExcludeReleases:      true,
		ExcludeAttachments:   true,
		ExcludeMetadata:      true,
		ExcludeGitData:       true,
		ExcludeOwnerProjects: true,
	}

	exclusions := repo.GetMigrationExclusions()

	if !exclusions.ExcludeReleases {
		t.Error("ExcludeReleases should be true")
	}
	if !exclusions.ExcludeAttachments {
		t.Error("ExcludeAttachments should be true")
	}
	if !exclusions.ExcludeMetadata {
		t.Error("ExcludeMetadata should be true")
	}
	if !exclusions.ExcludeGitData {
		t.Error("ExcludeGitData should be true")
	}
	if !exclusions.ExcludeOwnerProjects {
		t.Error("ExcludeOwnerProjects should be true")
	}
}

func TestRepository_GetGHESLimitViolations(t *testing.T) {
	repo := &Repository{
		HasOversizedCommits:        true,
		OversizedCommitDetails:     ptrString("commit too large"),
		HasLongRefs:                true,
		LongRefDetails:             ptrString("ref too long"),
		HasBlockingFiles:           true,
		BlockingFileDetails:        ptrString("blocked file"),
		HasLargeFileWarnings:       true,
		LargeFileWarningDetails:    ptrString("large file warning"),
		HasOversizedRepository:     true,
		OversizedRepositoryDetails: ptrString("repo too large"),
	}

	violations := repo.GetGHESLimitViolations()

	if !violations.HasOversizedCommits {
		t.Error("HasOversizedCommits should be true")
	}
	if *violations.OversizedCommitDetails != "commit too large" {
		t.Error("OversizedCommitDetails mismatch")
	}
	if !violations.HasLongRefs {
		t.Error("HasLongRefs should be true")
	}
	if !violations.HasBlockingFiles {
		t.Error("HasBlockingFiles should be true")
	}
	if !violations.HasLargeFileWarnings {
		t.Error("HasLargeFileWarnings should be true")
	}
	if !violations.HasOversizedRepository {
		t.Error("HasOversizedRepository should be true")
	}
}

func TestRepository_GetADOProperties(t *testing.T) {
	repo := &Repository{
		ADOProject:           ptrString("MyProject"),
		ADOIsGit:             true,
		ADOHasBoards:         true,
		ADOHasPipelines:      true,
		ADOHasGHAS:           true,
		ADOPullRequestCount:  10,
		ADOWorkItemCount:     50,
		ADOBranchPolicyCount: 3,
	}

	ado := repo.GetADOProperties()

	if *ado.Project != "MyProject" {
		t.Errorf("Project = %q, want %q", *ado.Project, "MyProject")
	}
	if !ado.IsGit {
		t.Error("IsGit should be true")
	}
	if !ado.HasBoards {
		t.Error("HasBoards should be true")
	}
	if !ado.HasPipelines {
		t.Error("HasPipelines should be true")
	}
	if !ado.HasGHAS {
		t.Error("HasGHAS should be true")
	}
	if ado.PullRequestCount != 10 {
		t.Errorf("PullRequestCount = %d, want 10", ado.PullRequestCount)
	}
}

func TestRepository_IsADORepository(t *testing.T) {
	tests := []struct {
		name    string
		project *string
		want    bool
	}{
		{"nil project", nil, false},
		{"empty project", ptrString(""), false},
		{"valid project", ptrString("MyProject"), true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo := &Repository{ADOProject: tt.project}
			got := repo.IsADORepository()
			if got != tt.want {
				t.Errorf("IsADORepository() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestRepository_HasMigrationBlockers(t *testing.T) {
	tests := []struct {
		name                   string
		hasOversizedCommits    bool
		hasLongRefs            bool
		hasBlockingFiles       bool
		hasOversizedRepository bool
		want                   bool
	}{
		{"no blockers", false, false, false, false, false},
		{"oversized commits", true, false, false, false, true},
		{"long refs", false, true, false, false, true},
		{"blocking files", false, false, true, false, true},
		{"oversized repo", false, false, false, true, true},
		{"multiple blockers", true, true, true, true, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo := &Repository{
				HasOversizedCommits:    tt.hasOversizedCommits,
				HasLongRefs:            tt.hasLongRefs,
				HasBlockingFiles:       tt.hasBlockingFiles,
				HasOversizedRepository: tt.hasOversizedRepository,
			}
			got := repo.HasMigrationBlockers()
			if got != tt.want {
				t.Errorf("HasMigrationBlockers() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestRepository_IsMigrationComplete(t *testing.T) {
	tests := []struct {
		status string
		want   bool
	}{
		{string(StatusComplete), true},
		{string(StatusMigrationComplete), true},
		{string(StatusPending), false},
		{string(StatusMigrationFailed), false},
		{string(StatusMigratingContent), false},
	}

	for _, tt := range tests {
		t.Run(tt.status, func(t *testing.T) {
			repo := &Repository{Status: tt.status}
			got := repo.IsMigrationComplete()
			if got != tt.want {
				t.Errorf("IsMigrationComplete() = %v, want %v for status %q", got, tt.want, tt.status)
			}
		})
	}
}

func TestRepository_IsMigrationInProgress(t *testing.T) {
	inProgressStatuses := []string{
		string(StatusPreMigration),
		string(StatusArchiveGenerating),
		string(StatusQueuedForMigration),
		string(StatusMigratingContent),
		string(StatusPostMigration),
	}

	notInProgressStatuses := []string{
		string(StatusPending),
		string(StatusComplete),
		string(StatusMigrationFailed),
	}

	for _, status := range inProgressStatuses {
		t.Run(status+"_true", func(t *testing.T) {
			repo := &Repository{Status: status}
			if !repo.IsMigrationInProgress() {
				t.Errorf("IsMigrationInProgress() = false for status %q, want true", status)
			}
		})
	}

	for _, status := range notInProgressStatuses {
		t.Run(status+"_false", func(t *testing.T) {
			repo := &Repository{Status: status}
			if repo.IsMigrationInProgress() {
				t.Errorf("IsMigrationInProgress() = true for status %q, want false", status)
			}
		})
	}
}

func TestRepository_IsMigrationFailed(t *testing.T) {
	tests := []struct {
		status string
		want   bool
	}{
		{string(StatusMigrationFailed), true},
		{string(StatusRolledBack), true},
		{string(StatusPending), false},
		{string(StatusComplete), false},
	}

	for _, tt := range tests {
		t.Run(tt.status, func(t *testing.T) {
			repo := &Repository{Status: tt.status}
			got := repo.IsMigrationFailed()
			if got != tt.want {
				t.Errorf("IsMigrationFailed() = %v, want %v for status %q", got, tt.want, tt.status)
			}
		})
	}
}

func TestRepository_CanBeMigrated(t *testing.T) {
	eligibleStatuses := []string{
		string(StatusPending),
		string(StatusDryRunComplete),
		string(StatusDryRunFailed),
		string(StatusMigrationFailed),
		string(StatusRolledBack),
		string(StatusDryRunQueued),
	}

	ineligibleStatuses := []string{
		string(StatusWontMigrate),
		string(StatusComplete),
		string(StatusMigrationComplete),
	}

	for _, status := range eligibleStatuses {
		t.Run(status+"_eligible", func(t *testing.T) {
			repo := &Repository{Status: status}
			if !repo.CanBeMigrated() {
				t.Errorf("CanBeMigrated() = false for status %q, want true", status)
			}
		})
	}

	for _, status := range ineligibleStatuses {
		t.Run(status+"_ineligible", func(t *testing.T) {
			repo := &Repository{Status: status}
			if repo.CanBeMigrated() {
				t.Errorf("CanBeMigrated() = true for status %q, want false", status)
			}
		})
	}
}

func TestRepository_CanBeAssignedToBatch(t *testing.T) {
	tests := []struct {
		name       string
		repo       *Repository
		wantCan    bool
		wantReason string
	}{
		{
			name:       "already in batch",
			repo:       &Repository{BatchID: ptrInt64(1), Status: string(StatusPending)},
			wantCan:    false,
			wantReason: "repository is already assigned to a batch",
		},
		{
			name:       "oversized repository",
			repo:       &Repository{HasOversizedRepository: true, Status: string(StatusPending)},
			wantCan:    false,
			wantReason: "repository exceeds GitHub's 40 GiB size limit and requires remediation",
		},
		{
			name:       "ineligible status",
			repo:       &Repository{Status: string(StatusComplete)},
			wantCan:    false,
			wantReason: "repository status is not eligible for batch assignment",
		},
		{
			name:    "eligible pending",
			repo:    &Repository{Status: string(StatusPending)},
			wantCan: true,
		},
		{
			name:    "eligible dry run complete",
			repo:    &Repository{Status: string(StatusDryRunComplete)},
			wantCan: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			canAssign, reason := tt.repo.CanBeAssignedToBatch()
			if canAssign != tt.wantCan {
				t.Errorf("CanBeAssignedToBatch() can = %v, want %v", canAssign, tt.wantCan)
			}
			if reason != tt.wantReason {
				t.Errorf("CanBeAssignedToBatch() reason = %q, want %q", reason, tt.wantReason)
			}
		})
	}
}

func TestRepository_GetOrganization(t *testing.T) {
	tests := []struct {
		fullName string
		want     string
	}{
		{"org/repo", "org"},
		{"my-org/my-repo", "my-org"},
		{"org/project/repo", "org"},
		{"single", "single"},
		{"", ""},
	}

	for _, tt := range tests {
		t.Run(tt.fullName, func(t *testing.T) {
			repo := &Repository{FullName: tt.fullName}
			got := repo.GetOrganization()
			if got != tt.want {
				t.Errorf("GetOrganization() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestRepository_GetRepoName(t *testing.T) {
	tests := []struct {
		fullName string
		want     string
	}{
		{"org/repo", "repo"},
		{"my-org/my-repo", "my-repo"},
		{"org/project/repo", "repo"},
		{"single", "single"},
		{"", ""},
	}

	for _, tt := range tests {
		t.Run(tt.fullName, func(t *testing.T) {
			repo := &Repository{FullName: tt.fullName}
			got := repo.GetRepoName()
			if got != tt.want {
				t.Errorf("GetRepoName() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestRepository_SetGitProperties(t *testing.T) {
	repo := &Repository{}
	props := GitProperties{
		TotalSize:     ptrInt64(5000),
		HasLFS:        true,
		HasSubmodules: true,
		BranchCount:   15,
		CommitCount:   200,
	}

	repo.SetGitProperties(props)

	if *repo.TotalSize != 5000 {
		t.Errorf("TotalSize = %d, want 5000", *repo.TotalSize)
	}
	if !repo.HasLFS {
		t.Error("HasLFS should be true")
	}
	if !repo.HasSubmodules {
		t.Error("HasSubmodules should be true")
	}
	if repo.BranchCount != 15 {
		t.Errorf("BranchCount = %d, want 15", repo.BranchCount)
	}
	if repo.CommitCount != 200 {
		t.Errorf("CommitCount = %d, want 200", repo.CommitCount)
	}
}

func TestRepository_SetGitHubFeatures(t *testing.T) {
	repo := &Repository{}
	features := GitHubFeatures{
		IsArchived:        true,
		HasWiki:           true,
		HasActions:        true,
		BranchProtections: 3,
	}

	repo.SetGitHubFeatures(features)

	if !repo.IsArchived {
		t.Error("IsArchived should be true")
	}
	if !repo.HasWiki {
		t.Error("HasWiki should be true")
	}
	if !repo.HasActions {
		t.Error("HasActions should be true")
	}
	if repo.BranchProtections != 3 {
		t.Errorf("BranchProtections = %d, want 3", repo.BranchProtections)
	}
}

func TestRepository_SetSecurityFeatures(t *testing.T) {
	repo := &Repository{}
	security := SecurityFeatures{
		HasCodeScanning:   true,
		HasDependabot:     true,
		HasSecretScanning: true,
		HasCodeowners:     true,
	}

	repo.SetSecurityFeatures(security)

	if !repo.HasCodeScanning {
		t.Error("HasCodeScanning should be true")
	}
	if !repo.HasDependabot {
		t.Error("HasDependabot should be true")
	}
	if !repo.HasSecretScanning {
		t.Error("HasSecretScanning should be true")
	}
	if !repo.HasCodeowners {
		t.Error("HasCodeowners should be true")
	}
}

func TestRepository_GetComplexityCategoryFromFeatures(t *testing.T) {
	tests := []struct {
		name string
		repo *Repository
		want string
	}{
		{
			name: "simple repo",
			repo: &Repository{},
			want: ComplexitySimple,
		},
		{
			name: "medium - has LFS",
			repo: &Repository{HasLFS: true},
			want: ComplexityMedium,
		},
		{
			name: "complex - multiple features",
			repo: &Repository{HasLFS: true, HasSubmodules: true},
			want: ComplexityComplex,
		},
		{
			name: "very complex - has blockers",
			repo: &Repository{HasOversizedCommits: true},
			want: ComplexityVeryComplex,
		},
		{
			name: "very complex - many features",
			repo: &Repository{HasLFS: true, HasSubmodules: true, HasLargeFiles: true, HasPackages: true},
			want: ComplexityVeryComplex,
		},
		{
			name: "medium - large size",
			repo: &Repository{TotalSize: ptrInt64(2 << 30)}, // 2GB
			want: ComplexityMedium,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.repo.GetComplexityCategoryFromFeatures()
			if got != tt.want {
				t.Errorf("GetComplexityCategoryFromFeatures() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestRepository_NeedsRemediation(t *testing.T) {
	tests := []struct {
		status string
		want   bool
	}{
		{string(StatusRemediationRequired), true},
		{string(StatusPending), false},
		{string(StatusComplete), false},
	}

	for _, tt := range tests {
		t.Run(tt.status, func(t *testing.T) {
			repo := &Repository{Status: tt.status}
			got := repo.NeedsRemediation()
			if got != tt.want {
				t.Errorf("NeedsRemediation() = %v, want %v for status %q", got, tt.want, tt.status)
			}
		})
	}
}

func TestRepository_SetMigrationState(t *testing.T) {
	repo := &Repository{}
	state := MigrationState{
		Status:              string(StatusMigratingContent),
		BatchID:             ptrInt64(42),
		Priority:            10,
		DestinationURL:      ptrString("https://github.com/dest/repo"),
		DestinationFullName: ptrString("dest/repo"),
		SourceMigrationID:   ptrInt64(12345),
		IsSourceLocked:      true,
	}

	repo.SetMigrationState(state)

	if repo.Status != string(StatusMigratingContent) {
		t.Errorf("Status = %q, want %q", repo.Status, string(StatusMigratingContent))
	}
	if repo.BatchID == nil || *repo.BatchID != 42 {
		t.Errorf("BatchID = %v, want 42", repo.BatchID)
	}
	if repo.Priority != 10 {
		t.Errorf("Priority = %d, want 10", repo.Priority)
	}
	if repo.DestinationURL == nil || *repo.DestinationURL != "https://github.com/dest/repo" {
		t.Errorf("DestinationURL = %v, want https://github.com/dest/repo", repo.DestinationURL)
	}
	if repo.DestinationFullName == nil || *repo.DestinationFullName != "dest/repo" {
		t.Errorf("DestinationFullName = %v, want dest/repo", repo.DestinationFullName)
	}
	if repo.SourceMigrationID == nil || *repo.SourceMigrationID != 12345 {
		t.Errorf("SourceMigrationID = %v, want 12345", repo.SourceMigrationID)
	}
	if !repo.IsSourceLocked {
		t.Error("IsSourceLocked should be true")
	}
}

func TestRepository_SetMigrationExclusions(t *testing.T) {
	repo := &Repository{}
	exclusions := MigrationExclusions{
		ExcludeReleases:      true,
		ExcludeAttachments:   true,
		ExcludeMetadata:      true,
		ExcludeGitData:       true,
		ExcludeOwnerProjects: true,
	}

	repo.SetMigrationExclusions(exclusions)

	if !repo.ExcludeReleases {
		t.Error("ExcludeReleases should be true")
	}
	if !repo.ExcludeAttachments {
		t.Error("ExcludeAttachments should be true")
	}
	if !repo.ExcludeMetadata {
		t.Error("ExcludeMetadata should be true")
	}
	if !repo.ExcludeGitData {
		t.Error("ExcludeGitData should be true")
	}
	if !repo.ExcludeOwnerProjects {
		t.Error("ExcludeOwnerProjects should be true")
	}
}

func TestRepository_SetGHESLimitViolations(t *testing.T) {
	repo := &Repository{}
	violations := GHESLimitViolations{
		HasOversizedCommits:        true,
		OversizedCommitDetails:     ptrString("commit abc123 is 150MB"),
		HasLongRefs:                true,
		LongRefDetails:             ptrString("long ref details"),
		HasBlockingFiles:           true,
		BlockingFileDetails:        ptrString("blocked files"),
		HasLargeFileWarnings:       true,
		LargeFileWarningDetails:    ptrString("large file warnings"),
		HasOversizedRepository:     true,
		OversizedRepositoryDetails: ptrString("oversized repo"),
	}

	repo.SetGHESLimitViolations(violations)

	// Test boolean flags
	boolTests := []struct {
		name string
		got  bool
	}{
		{"HasOversizedCommits", repo.HasOversizedCommits},
		{"HasLongRefs", repo.HasLongRefs},
		{"HasBlockingFiles", repo.HasBlockingFiles},
		{"HasLargeFileWarnings", repo.HasLargeFileWarnings},
		{"HasOversizedRepository", repo.HasOversizedRepository},
	}
	for _, tt := range boolTests {
		if !tt.got {
			t.Errorf("%s should be true", tt.name)
		}
	}

	// Test string pointer fields
	stringTests := []struct {
		name string
		got  *string
	}{
		{"OversizedCommitDetails", repo.OversizedCommitDetails},
		{"LongRefDetails", repo.LongRefDetails},
		{"BlockingFileDetails", repo.BlockingFileDetails},
		{"LargeFileWarningDetails", repo.LargeFileWarningDetails},
		{"OversizedRepositoryDetails", repo.OversizedRepositoryDetails},
	}
	for _, tt := range stringTests {
		if tt.got == nil {
			t.Errorf("%s should not be nil", tt.name)
		}
	}
}

func TestRepository_SetADOProperties(t *testing.T) {
	repo := &Repository{}
	ado := ADOProperties{
		Project:           ptrString("MyADOProject"),
		IsGit:             true,
		HasBoards:         true,
		HasPipelines:      true,
		HasGHAS:           true,
		PullRequestCount:  25,
		WorkItemCount:     100,
		BranchPolicyCount: 5,
	}

	repo.SetADOProperties(ado)

	if repo.ADOProject == nil || *repo.ADOProject != "MyADOProject" {
		t.Errorf("ADOProject = %v, want MyADOProject", repo.ADOProject)
	}
	if !repo.ADOIsGit {
		t.Error("ADOIsGit should be true")
	}
	if !repo.ADOHasBoards {
		t.Error("ADOHasBoards should be true")
	}
	if !repo.ADOHasPipelines {
		t.Error("ADOHasPipelines should be true")
	}
	if !repo.ADOHasGHAS {
		t.Error("ADOHasGHAS should be true")
	}
	if repo.ADOPullRequestCount != 25 {
		t.Errorf("ADOPullRequestCount = %d, want 25", repo.ADOPullRequestCount)
	}
	if repo.ADOWorkItemCount != 100 {
		t.Errorf("ADOWorkItemCount = %d, want 100", repo.ADOWorkItemCount)
	}
	if repo.ADOBranchPolicyCount != 5 {
		t.Errorf("ADOBranchPolicyCount = %d, want 5", repo.ADOBranchPolicyCount)
	}
}

func TestRepository_SetADOPipelineDetails(t *testing.T) {
	repo := &Repository{}
	pipeline := ADOPipelineDetails{
		PipelineCount:         10,
		YAMLPipelineCount:     7,
		ClassicPipelineCount:  3,
		PipelineRunCount:      500,
		HasServiceConnections: true,
		HasVariableGroups:     true,
		HasSelfHostedAgents:   true,
	}

	repo.SetADOPipelineDetails(pipeline)

	if repo.ADOPipelineCount != 10 {
		t.Errorf("ADOPipelineCount = %d, want 10", repo.ADOPipelineCount)
	}
	if repo.ADOYAMLPipelineCount != 7 {
		t.Errorf("ADOYAMLPipelineCount = %d, want 7", repo.ADOYAMLPipelineCount)
	}
	if repo.ADOClassicPipelineCount != 3 {
		t.Errorf("ADOClassicPipelineCount = %d, want 3", repo.ADOClassicPipelineCount)
	}
	if repo.ADOPipelineRunCount != 500 {
		t.Errorf("ADOPipelineRunCount = %d, want 500", repo.ADOPipelineRunCount)
	}
	if !repo.ADOHasServiceConnections {
		t.Error("ADOHasServiceConnections should be true")
	}
	if !repo.ADOHasVariableGroups {
		t.Error("ADOHasVariableGroups should be true")
	}
	if !repo.ADOHasSelfHostedAgents {
		t.Error("ADOHasSelfHostedAgents should be true")
	}
}
