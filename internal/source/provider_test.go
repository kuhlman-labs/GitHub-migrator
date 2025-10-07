package source

import (
	"testing"
)

func TestDefaultCloneOptions(t *testing.T) {
	opts := DefaultCloneOptions()

	// Default options use full clone for accurate git-sizer metrics
	if opts.Shallow {
		t.Error("Expected Shallow to be false (full clone needed for git-sizer)")
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

func TestShallowCloneOptions(t *testing.T) {
	opts := ShallowCloneOptions()

	// Shallow options for fast clones when full history isn't needed
	if !opts.Shallow {
		t.Error("Expected Shallow to be true for shallow clone options")
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
