package migration

import (
	"encoding/json"
	"log/slog"
	"os"
	"testing"

	"github.com/kuhlman-labs/github-migrator/internal/models"
)

func newValidationTestLogger() *slog.Logger {
	return slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelError}))
}

// validStrPtr creates a string pointer (local to avoid redeclaration)
func validStrPtr(s string) *string {
	return &s
}

// validInt64Ptr creates an int64 pointer (local to avoid redeclaration)
func validInt64Ptr(i int64) *int64 {
	return &i
}

func TestValidationMismatch_Structure(t *testing.T) {
	mismatch := ValidationMismatch{
		Field:       "commit_count",
		SourceValue: 100,
		DestValue:   99,
		Critical:    true,
	}

	if mismatch.Field != "commit_count" {
		t.Errorf("Field = %q, want %q", mismatch.Field, "commit_count")
	}
	if mismatch.SourceValue != 100 {
		t.Errorf("SourceValue = %v, want %v", mismatch.SourceValue, 100)
	}
	if mismatch.DestValue != 99 {
		t.Errorf("DestValue = %v, want %v", mismatch.DestValue, 99)
	}
	if !mismatch.Critical {
		t.Error("Critical should be true")
	}
}

func TestValidationMismatch_JSONSerialization(t *testing.T) {
	mismatch := ValidationMismatch{
		Field:       "branch_count",
		SourceValue: 5,
		DestValue:   4,
		Critical:    true,
	}

	data, err := json.Marshal(mismatch)
	if err != nil {
		t.Fatalf("Failed to marshal: %v", err)
	}

	var decoded ValidationMismatch
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("Failed to unmarshal: %v", err)
	}

	if decoded.Field != mismatch.Field {
		t.Errorf("Field mismatch after serialization")
	}
	// Note: SourceValue and DestValue will be float64 after JSON round-trip
	if decoded.Critical != mismatch.Critical {
		t.Errorf("Critical mismatch after serialization")
	}
}

func TestExecutor_compareRepositoryCharacteristics_Additional(t *testing.T) {
	tests := []struct {
		name           string
		source         *models.Repository
		dest           *models.Repository
		wantMismatches int
		wantCritical   bool
	}{
		{
			name: "identical repositories",
			source: &models.Repository{
				DefaultBranch: validStrPtr("main"),
				CommitCount:   100,
				BranchCount:   5,
				TagCount:      10,
				HasWiki:       true,
				HasPages:      false,
			},
			dest: &models.Repository{
				DefaultBranch: validStrPtr("main"),
				CommitCount:   100,
				BranchCount:   5,
				TagCount:      10,
				HasWiki:       true,
				HasPages:      false,
			},
			wantMismatches: 0,
			wantCritical:   false,
		},
		{
			name: "different default branch - critical",
			source: &models.Repository{
				DefaultBranch: validStrPtr("main"),
			},
			dest: &models.Repository{
				DefaultBranch: validStrPtr("master"),
			},
			wantMismatches: 1,
			wantCritical:   true,
		},
		{
			name: "different commit count - critical",
			source: &models.Repository{
				CommitCount: 100,
			},
			dest: &models.Repository{
				CommitCount: 99,
			},
			wantMismatches: 1,
			wantCritical:   true,
		},
		{
			name: "different branch count - critical",
			source: &models.Repository{
				BranchCount: 5,
			},
			dest: &models.Repository{
				BranchCount: 4,
			},
			wantMismatches: 1,
			wantCritical:   true,
		},
		{
			name: "different tag count - not critical",
			source: &models.Repository{
				TagCount: 10,
			},
			dest: &models.Repository{
				TagCount: 9,
			},
			wantMismatches: 1,
			wantCritical:   false,
		},
		{
			name: "different wiki - not critical",
			source: &models.Repository{
				HasWiki: true,
			},
			dest: &models.Repository{
				HasWiki: false,
			},
			wantMismatches: 1,
			wantCritical:   false,
		},
		{
			name: "different pages - not critical",
			source: &models.Repository{
				HasPages: true,
			},
			dest: &models.Repository{
				HasPages: false,
			},
			wantMismatches: 1,
			wantCritical:   false,
		},
		{
			name: "different discussions - not critical",
			source: &models.Repository{
				HasDiscussions: true,
			},
			dest: &models.Repository{
				HasDiscussions: false,
			},
			wantMismatches: 1,
			wantCritical:   false,
		},
		{
			name: "different actions - not critical",
			source: &models.Repository{
				HasActions: true,
			},
			dest: &models.Repository{
				HasActions: false,
			},
			wantMismatches: 1,
			wantCritical:   false,
		},
		{
			name: "different branch protections - not critical",
			source: &models.Repository{
				BranchProtections: 2,
			},
			dest: &models.Repository{
				BranchProtections: 0,
			},
			wantMismatches: 1,
			wantCritical:   false,
		},
		{
			name: "different last commit SHA - critical",
			source: &models.Repository{
				LastCommitSHA: validStrPtr("abc123"),
			},
			dest: &models.Repository{
				LastCommitSHA: validStrPtr("def456"),
			},
			wantMismatches: 1,
			wantCritical:   true,
		},
		{
			name: "multiple mismatches - mixed criticality",
			source: &models.Repository{
				DefaultBranch: validStrPtr("main"),
				CommitCount:   100,
				TagCount:      10,
				HasWiki:       true,
			},
			dest: &models.Repository{
				DefaultBranch: validStrPtr("master"),
				CommitCount:   99,
				TagCount:      9,
				HasWiki:       false,
			},
			wantMismatches: 4,
			wantCritical:   true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			e := &Executor{logger: newValidationTestLogger()}
			mismatches, hasCritical := e.compareRepositoryCharacteristics(tt.source, tt.dest)

			if len(mismatches) != tt.wantMismatches {
				t.Errorf("got %d mismatches, want %d", len(mismatches), tt.wantMismatches)
				for _, m := range mismatches {
					t.Logf("  - %s: source=%v, dest=%v, critical=%v", m.Field, m.SourceValue, m.DestValue, m.Critical)
				}
			}
			if hasCritical != tt.wantCritical {
				t.Errorf("hasCritical = %v, want %v", hasCritical, tt.wantCritical)
			}
		})
	}
}

func TestExecutor_generateValidationReport(t *testing.T) {
	e := &Executor{logger: newValidationTestLogger()}

	tests := []struct {
		name              string
		mismatches        []ValidationMismatch
		wantTotal         int
		wantCriticalCount int
	}{
		{
			name:              "no mismatches",
			mismatches:        []ValidationMismatch{},
			wantTotal:         0,
			wantCriticalCount: 0,
		},
		{
			name: "one critical mismatch",
			mismatches: []ValidationMismatch{
				{Field: "commit_count", SourceValue: 100, DestValue: 99, Critical: true},
			},
			wantTotal:         1,
			wantCriticalCount: 1,
		},
		{
			name: "one non-critical mismatch",
			mismatches: []ValidationMismatch{
				{Field: "has_wiki", SourceValue: true, DestValue: false, Critical: false},
			},
			wantTotal:         1,
			wantCriticalCount: 0,
		},
		{
			name: "mixed mismatches",
			mismatches: []ValidationMismatch{
				{Field: "commit_count", SourceValue: 100, DestValue: 99, Critical: true},
				{Field: "has_wiki", SourceValue: true, DestValue: false, Critical: false},
				{Field: "branch_count", SourceValue: 5, DestValue: 4, Critical: true},
			},
			wantTotal:         3,
			wantCriticalCount: 2,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			report := e.generateValidationReport(tt.mismatches)

			// Parse JSON report
			var parsed struct {
				TotalMismatches    int                  `json:"total_mismatches"`
				CriticalMismatches int                  `json:"critical_mismatches"`
				Mismatches         []ValidationMismatch `json:"mismatches"`
			}
			if err := json.Unmarshal([]byte(report), &parsed); err != nil {
				t.Fatalf("Failed to parse report JSON: %v\nReport: %s", err, report)
			}

			if parsed.TotalMismatches != tt.wantTotal {
				t.Errorf("TotalMismatches = %d, want %d", parsed.TotalMismatches, tt.wantTotal)
			}
			if parsed.CriticalMismatches != tt.wantCriticalCount {
				t.Errorf("CriticalMismatches = %d, want %d", parsed.CriticalMismatches, tt.wantCriticalCount)
			}
			if len(parsed.Mismatches) != len(tt.mismatches) {
				t.Errorf("Mismatches count = %d, want %d", len(parsed.Mismatches), len(tt.mismatches))
			}
		})
	}
}

func TestExecutor_serializeDestinationData(t *testing.T) {
	e := &Executor{logger: newValidationTestLogger()}

	dest := &models.Repository{
		DefaultBranch:     validStrPtr("main"),
		BranchCount:       5,
		CommitCount:       100,
		TagCount:          10,
		LastCommitSHA:     validStrPtr("abc123"),
		TotalSize:         validInt64Ptr(1024000),
		HasWiki:           true,
		HasPages:          false,
		HasDiscussions:    true,
		HasActions:        true,
		BranchProtections: 2,
		IssueCount:        15,
		PullRequestCount:  8,
	}

	jsonData := e.serializeDestinationData(dest)

	var parsed map[string]any
	if err := json.Unmarshal([]byte(jsonData), &parsed); err != nil {
		t.Fatalf("Failed to parse JSON: %v\nData: %s", err, jsonData)
	}

	// Verify key fields are present
	expectedFields := []string{
		"default_branch", "branch_count", "commit_count", "tag_count",
		"last_commit_sha", "total_size", "has_wiki", "has_pages",
		"has_discussions", "has_actions", "branch_protections",
		"issue_count", "pull_request_count",
	}

	for _, field := range expectedFields {
		if _, ok := parsed[field]; !ok {
			t.Errorf("Missing expected field: %s", field)
		}
	}

	// Verify specific values
	if parsed["default_branch"] != "main" {
		t.Errorf("default_branch = %v, want %v", parsed["default_branch"], "main")
	}
	if parsed["branch_count"].(float64) != 5 {
		t.Errorf("branch_count = %v, want %v", parsed["branch_count"], 5)
	}
	if parsed["has_wiki"].(bool) != true {
		t.Errorf("has_wiki = %v, want %v", parsed["has_wiki"], true)
	}
}

func TestExecutor_serializeDestinationData_NilFields(t *testing.T) {
	e := &Executor{logger: newValidationTestLogger()}

	// Repository with nil optional fields
	dest := &models.Repository{
		BranchCount: 3,
		CommitCount: 50,
	}

	jsonData := e.serializeDestinationData(dest)

	var parsed map[string]any
	if err := json.Unmarshal([]byte(jsonData), &parsed); err != nil {
		t.Fatalf("Failed to parse JSON: %v", err)
	}

	// Nil fields should be omitted or null
	if _, ok := parsed["default_branch"]; ok && parsed["default_branch"] != nil {
		t.Logf("default_branch is present with value: %v", parsed["default_branch"])
	}

	// Non-nil fields should have correct values
	if parsed["branch_count"].(float64) != 3 {
		t.Errorf("branch_count = %v, want %v", parsed["branch_count"], 3)
	}
}
