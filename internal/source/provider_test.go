package source

import (
	"testing"
)

func TestDefaultCloneOptions(t *testing.T) {
	opts := DefaultCloneOptions()

	if !opts.Shallow {
		t.Error("Expected Shallow to be true")
	}
	if opts.Bare {
		t.Error("Expected Bare to be false")
	}
	if opts.IncludeLFS {
		t.Error("Expected IncludeLFS to be false")
	}
	if opts.IncludeSubmodules {
		t.Error("Expected IncludeSubmodules to be false")
	}
}

func TestProviderType_String(t *testing.T) {
	tests := []struct {
		providerType ProviderType
		expected     string
	}{
		{ProviderGitHub, "github"},
		{ProviderGitLab, "gitlab"},
		{ProviderAzureDevOps, "azuredevops"},
	}

	for _, tt := range tests {
		t.Run(string(tt.providerType), func(t *testing.T) {
			if string(tt.providerType) != tt.expected {
				t.Errorf("Expected %s, got %s", tt.expected, tt.providerType)
			}
		})
	}
}

func TestFeature_String(t *testing.T) {
	tests := []struct {
		feature  Feature
		expected string
	}{
		{FeatureLFS, "lfs"},
		{FeatureSubmodules, "submodules"},
		{FeatureWiki, "wiki"},
		{FeaturePages, "pages"},
		{FeatureActions, "actions"},
		{FeatureDiscussions, "discussions"},
	}

	for _, tt := range tests {
		t.Run(string(tt.feature), func(t *testing.T) {
			if string(tt.feature) != tt.expected {
				t.Errorf("Expected %s, got %s", tt.expected, tt.feature)
			}
		})
	}
}
