// Package models provides domain types and constants for the GitHub migrator.
//
// This file consolidates all status and visibility constants used throughout
// the application. Import these constants instead of defining local ones.
package models

// Visibility constants for repository access control.
// These values align with GitHub's visibility settings.
const (
	VisibilityPublic   = "public"
	VisibilityPrivate  = "private"
	VisibilityInternal = "internal"
)

// ValidVisibilities returns all valid visibility values.
func ValidVisibilities() []string {
	return []string{VisibilityPublic, VisibilityPrivate, VisibilityInternal}
}

// IsValidVisibility checks if a visibility value is valid.
func IsValidVisibility(visibility string) bool {
	for _, v := range ValidVisibilities() {
		if v == visibility {
			return true
		}
	}
	return false
}

// Size category constants for repository classification.
const (
	SizeCategorySmall     = "small"      // < 100MB
	SizeCategoryMedium    = "medium"     // 100MB - 1GB
	SizeCategoryLarge     = "large"      // 1GB - 5GB
	SizeCategoryVeryLarge = "very_large" // > 5GB
	SizeCategoryUnknown   = "unknown"    // Size not determined
)

// Size category thresholds in bytes.
const (
	SizeThreshold100MB = 100 * 1024 * 1024      // 100 MB
	SizeThreshold1GB   = 1024 * 1024 * 1024     // 1 GB
	SizeThreshold5GB   = 5 * 1024 * 1024 * 1024 // 5 GB
)

// GetSizeCategory determines the size category for a given size in bytes.
func GetSizeCategory(sizeBytes int64) string {
	if sizeBytes <= 0 {
		return SizeCategoryUnknown
	}
	if sizeBytes < SizeThreshold100MB {
		return SizeCategorySmall
	}
	if sizeBytes < SizeThreshold1GB {
		return SizeCategoryMedium
	}
	if sizeBytes < SizeThreshold5GB {
		return SizeCategoryLarge
	}
	return SizeCategoryVeryLarge
}

// Complexity category constants for repository classification.
const (
	ComplexitySimple      = "simple"       // Score <= 5
	ComplexityMedium      = "medium"       // Score 6-10
	ComplexityComplex     = "complex"      // Score 11-17
	ComplexityVeryComplex = "very_complex" // Score >= 18
)

// Complexity score thresholds.
const (
	ComplexityThresholdSimple  = 5
	ComplexityThresholdMedium  = 10
	ComplexityThresholdComplex = 17
)

// GetComplexityCategory determines the complexity category for a given score.
func GetComplexityCategory(score int) string {
	if score <= ComplexityThresholdSimple {
		return ComplexitySimple
	}
	if score <= ComplexityThresholdMedium {
		return ComplexityMedium
	}
	if score <= ComplexityThresholdComplex {
		return ComplexityComplex
	}
	return ComplexityVeryComplex
}

// Source type constants for repository origins.
const (
	SourceTypeGitHub      = "github"
	SourceTypeGHES        = "ghes" // GitHub Enterprise Server
	SourceTypeGHEC        = "ghec" // GitHub Enterprise Cloud
	SourceTypeGitLab      = "gitlab"
	SourceTypeAzureDevOps = "azuredevops"
)

// MigrationStatus values are defined elsewhere in this package (models.go).
// Reference them using the existing constants:
//   StatusPending, StatusComplete, StatusMigrationFailed, etc.

// Batch status constants for batch lifecycle.
const (
	BatchStatusPending    = "pending"
	BatchStatusInProgress = "in_progress"
	BatchStatusCompleted  = "completed"
	BatchStatusFailed     = "failed"
	BatchStatusCanceled   = "canceled"
)

// User mapping status constants.
const (
	MappingStatusUnmapped = "unmapped"
	MappingStatusMapped   = "mapped"
	MappingStatusSkipped  = "skipped"
)

// Team mapping migration status constants.
const (
	TeamMigrationPending    = "pending"
	TeamMigrationInProgress = "in_progress"
	TeamMigrationCompleted  = "completed"
	TeamMigrationFailed     = "failed"
)

// Note: Discovery status constants are defined in models.go:
//   DiscoveryStatusInProgress, DiscoveryStatusCompleted, DiscoveryStatusFailed

// Validation status constants.
const (
	ValidationStatusPassed  = "passed"
	ValidationStatusFailed  = "failed"
	ValidationStatusSkipped = "skipped"
)
