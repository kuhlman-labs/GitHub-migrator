package models

import "testing"

func TestValidVisibilities(t *testing.T) {
	visibilities := ValidVisibilities()

	if len(visibilities) != 3 {
		t.Errorf("ValidVisibilities() returned %d items, want 3", len(visibilities))
	}

	expected := map[string]bool{
		VisibilityPublic:   true,
		VisibilityPrivate:  true,
		VisibilityInternal: true,
	}

	for _, v := range visibilities {
		if !expected[v] {
			t.Errorf("Unexpected visibility: %s", v)
		}
	}
}

func TestIsValidVisibility(t *testing.T) {
	tests := []struct {
		visibility string
		want       bool
	}{
		{VisibilityPublic, true},
		{VisibilityPrivate, true},
		{VisibilityInternal, true},
		{"PUBLIC", false}, // Case sensitive
		{"Public", false},
		{"unknown", false},
		{"", false},
	}

	for _, tt := range tests {
		t.Run(tt.visibility, func(t *testing.T) {
			got := IsValidVisibility(tt.visibility)
			if got != tt.want {
				t.Errorf("IsValidVisibility(%q) = %v, want %v", tt.visibility, got, tt.want)
			}
		})
	}
}

func TestGetSizeCategory(t *testing.T) {
	tests := []struct {
		name      string
		sizeBytes int64
		want      string
	}{
		{"negative size", -1, SizeCategoryUnknown},
		{"zero size", 0, SizeCategoryUnknown},
		{"1 byte", 1, SizeCategorySmall},
		{"50MB", 50 * 1024 * 1024, SizeCategorySmall},
		{"99MB", 99 * 1024 * 1024, SizeCategorySmall},
		{"exactly 100MB", SizeThreshold100MB, SizeCategoryMedium},
		{"101MB", 101 * 1024 * 1024, SizeCategoryMedium},
		{"500MB", 500 * 1024 * 1024, SizeCategoryMedium},
		{"exactly 1GB", SizeThreshold1GB, SizeCategoryLarge},
		{"2GB", 2 * 1024 * 1024 * 1024, SizeCategoryLarge},
		{"exactly 5GB", SizeThreshold5GB, SizeCategoryVeryLarge},
		{"10GB", 10 * 1024 * 1024 * 1024, SizeCategoryVeryLarge},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := GetSizeCategory(tt.sizeBytes)
			if got != tt.want {
				t.Errorf("GetSizeCategory(%d) = %q, want %q", tt.sizeBytes, got, tt.want)
			}
		})
	}
}

func TestGetComplexityCategory(t *testing.T) {
	tests := []struct {
		name  string
		score int
		want  string
	}{
		{"score 0", 0, ComplexitySimple},
		{"score 5", ComplexityThresholdSimple, ComplexitySimple},
		{"score 6", 6, ComplexityMedium},
		{"score 10", ComplexityThresholdMedium, ComplexityMedium},
		{"score 11", 11, ComplexityComplex},
		{"score 17", ComplexityThresholdComplex, ComplexityComplex},
		{"score 18", 18, ComplexityVeryComplex},
		{"score 100", 100, ComplexityVeryComplex},
		{"negative score", -1, ComplexitySimple},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := GetComplexityCategory(tt.score)
			if got != tt.want {
				t.Errorf("GetComplexityCategory(%d) = %q, want %q", tt.score, got, tt.want)
			}
		})
	}
}

func TestVisibilityConstants(t *testing.T) {
	if VisibilityPublic != "public" {
		t.Errorf("VisibilityPublic = %q, want %q", VisibilityPublic, "public")
	}
	if VisibilityPrivate != "private" {
		t.Errorf("VisibilityPrivate = %q, want %q", VisibilityPrivate, "private")
	}
	if VisibilityInternal != "internal" {
		t.Errorf("VisibilityInternal = %q, want %q", VisibilityInternal, "internal")
	}
}

func TestSizeCategoryConstants(t *testing.T) {
	constants := map[string]string{
		"SizeCategorySmall":     SizeCategorySmall,
		"SizeCategoryMedium":    SizeCategoryMedium,
		"SizeCategoryLarge":     SizeCategoryLarge,
		"SizeCategoryVeryLarge": SizeCategoryVeryLarge,
		"SizeCategoryUnknown":   SizeCategoryUnknown,
	}

	expected := map[string]string{
		"SizeCategorySmall":     "small",
		"SizeCategoryMedium":    "medium",
		"SizeCategoryLarge":     "large",
		"SizeCategoryVeryLarge": "very_large",
		"SizeCategoryUnknown":   "unknown",
	}

	for name, got := range constants {
		if got != expected[name] {
			t.Errorf("%s = %q, want %q", name, got, expected[name])
		}
	}
}

func TestSizeThresholds(t *testing.T) {
	// Test that thresholds are in the correct order
	if SizeThreshold100MB >= SizeThreshold1GB {
		t.Error("SizeThreshold100MB should be less than SizeThreshold1GB")
	}
	if SizeThreshold1GB >= SizeThreshold5GB {
		t.Error("SizeThreshold1GB should be less than SizeThreshold5GB")
	}

	// Test actual values
	if SizeThreshold100MB != 100*1024*1024 {
		t.Errorf("SizeThreshold100MB = %d, want %d", SizeThreshold100MB, 100*1024*1024)
	}
	if SizeThreshold1GB != 1024*1024*1024 {
		t.Errorf("SizeThreshold1GB = %d, want %d", SizeThreshold1GB, 1024*1024*1024)
	}
	if SizeThreshold5GB != 5*1024*1024*1024 {
		t.Errorf("SizeThreshold5GB = %d, want %d", SizeThreshold5GB, 5*1024*1024*1024)
	}
}

func TestComplexityConstants(t *testing.T) {
	constants := map[string]string{
		"ComplexitySimple":      ComplexitySimple,
		"ComplexityMedium":      ComplexityMedium,
		"ComplexityComplex":     ComplexityComplex,
		"ComplexityVeryComplex": ComplexityVeryComplex,
	}

	expected := map[string]string{
		"ComplexitySimple":      "simple",
		"ComplexityMedium":      "medium",
		"ComplexityComplex":     "complex",
		"ComplexityVeryComplex": "very_complex",
	}

	for name, got := range constants {
		if got != expected[name] {
			t.Errorf("%s = %q, want %q", name, got, expected[name])
		}
	}
}

func TestSourceTypeConstants(t *testing.T) {
	if SourceTypeGitHub != "github" {
		t.Errorf("SourceTypeGitHub = %q, want %q", SourceTypeGitHub, "github")
	}
	if SourceTypeGHES != "ghes" {
		t.Errorf("SourceTypeGHES = %q, want %q", SourceTypeGHES, "ghes")
	}
	if SourceTypeGHEC != "ghec" {
		t.Errorf("SourceTypeGHEC = %q, want %q", SourceTypeGHEC, "ghec")
	}
	if SourceTypeGitLab != "gitlab" {
		t.Errorf("SourceTypeGitLab = %q, want %q", SourceTypeGitLab, "gitlab")
	}
	if SourceTypeAzureDevOps != "azuredevops" {
		t.Errorf("SourceTypeAzureDevOps = %q, want %q", SourceTypeAzureDevOps, "azuredevops")
	}
}

func TestBatchStatusConstants(t *testing.T) {
	statuses := []struct {
		name  string
		value string
	}{
		{"BatchStatusPending", BatchStatusPending},
		{"BatchStatusReady", BatchStatusReady},
		{"BatchStatusInProgress", BatchStatusInProgress},
		{"BatchStatusCompleted", BatchStatusCompleted},
		{"BatchStatusCompletedWithErrors", BatchStatusCompletedWithErrors},
		{"BatchStatusFailed", BatchStatusFailed},
		{"BatchStatusCancelled", BatchStatusCancelled},
	}

	expected := map[string]string{
		"BatchStatusPending":             "pending",
		"BatchStatusReady":               "ready",
		"BatchStatusInProgress":          "in_progress",
		"BatchStatusCompleted":           "completed",
		"BatchStatusCompletedWithErrors": "completed_with_errors",
		"BatchStatusFailed":              "failed",
		"BatchStatusCancelled":           "cancelled",
	}

	for _, s := range statuses {
		if s.value != expected[s.name] {
			t.Errorf("%s = %q, want %q", s.name, s.value, expected[s.name])
		}
	}
}

func TestMappingStatusConstants(t *testing.T) {
	if MappingStatusUnmapped != "unmapped" {
		t.Errorf("MappingStatusUnmapped = %q, want %q", MappingStatusUnmapped, "unmapped")
	}
	if MappingStatusMapped != "mapped" {
		t.Errorf("MappingStatusMapped = %q, want %q", MappingStatusMapped, "mapped")
	}
	if MappingStatusSkipped != "skipped" {
		t.Errorf("MappingStatusSkipped = %q, want %q", MappingStatusSkipped, "skipped")
	}
}

func TestTeamMigrationStatusConstants(t *testing.T) {
	if TeamMigrationPending != "pending" {
		t.Errorf("TeamMigrationPending = %q, want %q", TeamMigrationPending, "pending")
	}
	if TeamMigrationInProgress != "in_progress" {
		t.Errorf("TeamMigrationInProgress = %q, want %q", TeamMigrationInProgress, "in_progress")
	}
	if TeamMigrationCompleted != "completed" {
		t.Errorf("TeamMigrationCompleted = %q, want %q", TeamMigrationCompleted, "completed")
	}
	if TeamMigrationFailed != "failed" {
		t.Errorf("TeamMigrationFailed = %q, want %q", TeamMigrationFailed, "failed")
	}
}

func TestValidationStatusConstants(t *testing.T) {
	if ValidationStatusPassed != "passed" {
		t.Errorf("ValidationStatusPassed = %q, want %q", ValidationStatusPassed, "passed")
	}
	if ValidationStatusFailed != "failed" {
		t.Errorf("ValidationStatusFailed = %q, want %q", ValidationStatusFailed, "failed")
	}
	if ValidationStatusSkipped != "skipped" {
		t.Errorf("ValidationStatusSkipped = %q, want %q", ValidationStatusSkipped, "skipped")
	}
}

func TestBatchTypePilot(t *testing.T) {
	if BatchTypePilot != "pilot" {
		t.Errorf("BatchTypePilot = %q, want %q", BatchTypePilot, "pilot")
	}
}
