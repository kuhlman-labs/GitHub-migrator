package github

import (
	"testing"
)

func TestDependencyGraphManifest(t *testing.T) {
	t.Run("creates dependency graph manifest correctly", func(t *testing.T) {
		manifest := &DependencyGraphManifest{
			Filename: "package.json",
			Dependencies: []DependencyGraphDependency{
				{
					PackageName:    "lodash",
					PackageManager: "npm",
					Requirements:   "^4.17.21",
				},
				{
					PackageName:    "express",
					PackageManager: "npm",
					Requirements:   "^4.18.0",
				},
			},
		}

		if manifest.Filename != "package.json" {
			t.Errorf("Expected filename 'package.json', got %s", manifest.Filename)
		}
		if len(manifest.Dependencies) != 2 {
			t.Errorf("Expected 2 dependencies, got %d", len(manifest.Dependencies))
		}
	})
}

func TestDependencyGraphDependency(t *testing.T) {
	t.Run("creates dependency correctly", func(t *testing.T) {
		dep := &DependencyGraphDependency{
			PackageName:    "react",
			PackageManager: "npm",
			Requirements:   "^18.0.0",
		}

		if dep.PackageName != "react" {
			t.Errorf("Expected package name 'react', got %s", dep.PackageName)
		}
		if dep.PackageManager != "npm" {
			t.Errorf("Expected package manager 'npm', got %s", dep.PackageManager)
		}
		if dep.Requirements != "^18.0.0" {
			t.Errorf("Expected requirements '^18.0.0', got %s", dep.Requirements)
		}
	})
}

func TestNewStr(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "regular string",
			input:    "hello",
			expected: "hello",
		},
		{
			name:     "empty string",
			input:    "",
			expected: "",
		},
		{
			name:     "string with spaces",
			input:    "hello world",
			expected: "hello world",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := newStr(tt.input)
			if result == nil {
				t.Fatal("newStr() returned nil")
				return // Explicitly unreachable, but satisfies static analysis
			}
			if *result != tt.expected {
				t.Errorf("newStr(%q) = %q, want %q", tt.input, *result, tt.expected)
			}
		})
	}
}

func TestDependencyGraphManifest_WithComplexData(t *testing.T) {
	t.Run("handles multiple package managers", func(t *testing.T) {
		manifests := []DependencyGraphManifest{
			{
				Filename: "package.json",
				Dependencies: []DependencyGraphDependency{
					{PackageName: "lodash", PackageManager: "npm", Requirements: "^4.17.21"},
				},
			},
			{
				Filename: "requirements.txt",
				Dependencies: []DependencyGraphDependency{
					{PackageName: "django", PackageManager: "pip", Requirements: ">=4.0"},
					{PackageName: "requests", PackageManager: "pip", Requirements: ">=2.28.0"},
				},
			},
			{
				Filename: "go.mod",
				Dependencies: []DependencyGraphDependency{
					{PackageName: "github.com/gin-gonic/gin", PackageManager: "gomod", Requirements: "v1.9.0"},
				},
			},
		}

		if len(manifests) != 3 {
			t.Errorf("Expected 3 manifests, got %d", len(manifests))
		}

		// Verify npm manifest
		if manifests[0].Filename != "package.json" {
			t.Errorf("Expected first manifest 'package.json', got %s", manifests[0].Filename)
		}

		// Verify pip manifest
		if len(manifests[1].Dependencies) != 2 {
			t.Errorf("Expected 2 pip dependencies, got %d", len(manifests[1].Dependencies))
		}

		// Verify go manifest
		if manifests[2].Dependencies[0].PackageManager != "gomod" {
			t.Errorf("Expected gomod package manager, got %s", manifests[2].Dependencies[0].PackageManager)
		}
	})
}
