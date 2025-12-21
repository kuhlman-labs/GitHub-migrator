package ado

import (
	"testing"
)

func TestIsADOURL(t *testing.T) {
	tests := []struct {
		name     string
		url      string
		expected bool
	}{
		{
			name:     "dev.azure.com HTTPS",
			url:      "https://dev.azure.com/myorg/myproject/_git/myrepo",
			expected: true,
		},
		{
			name:     "dev.azure.com with auth",
			url:      "https://user@dev.azure.com/myorg/myproject/_git/myrepo",
			expected: true,
		},
		{
			name:     "visualstudio.com",
			url:      "https://myorg.visualstudio.com/myproject/_git/myrepo",
			expected: true,
		},
		{
			name:     "SSH URL",
			url:      "git@ssh.dev.azure.com:v3/myorg/myproject/myrepo",
			expected: true,
		},
		{
			name:     "GitHub URL",
			url:      "https://github.com/owner/repo",
			expected: false,
		},
		{
			name:     "GitLab URL",
			url:      "https://gitlab.com/owner/repo",
			expected: false,
		},
		{
			name:     "empty string",
			url:      "",
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := IsADOURL(tt.url)
			if result != tt.expected {
				t.Errorf("IsADOURL(%q) = %v, want %v", tt.url, result, tt.expected)
			}
		})
	}
}

func TestIsADOHost(t *testing.T) {
	tests := []struct {
		name     string
		host     string
		expected bool
	}{
		{
			name:     "dev.azure.com",
			host:     "dev.azure.com",
			expected: true,
		},
		{
			name:     "ssh.dev.azure.com",
			host:     "ssh.dev.azure.com",
			expected: true,
		},
		{
			name:     "visualstudio.com suffix",
			host:     "myorg.visualstudio.com",
			expected: true,
		},
		{
			name:     "github.com",
			host:     "github.com",
			expected: false,
		},
		{
			name:     "empty string",
			host:     "",
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := IsADOHost(tt.host)
			if result != tt.expected {
				t.Errorf("IsADOHost(%q) = %v, want %v", tt.host, result, tt.expected)
			}
		})
	}
}

func TestParse(t *testing.T) {
	tests := []struct {
		name        string
		url         string
		wantOrg     string
		wantProject string
		wantRepo    string
		wantHost    string
		wantSSH     bool
		wantNil     bool
	}{
		{
			name:        "dev.azure.com basic",
			url:         "https://dev.azure.com/myorg/myproject/_git/myrepo",
			wantOrg:     "myorg",
			wantProject: "myproject",
			wantRepo:    "myrepo",
			wantHost:    "dev.azure.com",
			wantSSH:     false,
		},
		{
			name:        "dev.azure.com with .git suffix",
			url:         "https://dev.azure.com/myorg/myproject/_git/myrepo.git",
			wantOrg:     "myorg",
			wantProject: "myproject",
			wantRepo:    "myrepo",
			wantHost:    "dev.azure.com",
			wantSSH:     false,
		},
		{
			name:        "dev.azure.com with auth",
			url:         "https://user@dev.azure.com/myorg/myproject/_git/myrepo",
			wantOrg:     "myorg",
			wantProject: "myproject",
			wantRepo:    "myrepo",
			wantHost:    "dev.azure.com",
			wantSSH:     false,
		},
		{
			name:        "visualstudio.com",
			url:         "https://contoso.visualstudio.com/DefaultProject/_git/ContosoRepo",
			wantOrg:     "contoso",
			wantProject: "DefaultProject",
			wantRepo:    "ContosoRepo",
			wantHost:    "contoso.visualstudio.com",
			wantSSH:     false,
		},
		{
			name:        "SSH URL",
			url:         "git@ssh.dev.azure.com:v3/myorg/myproject/myrepo",
			wantOrg:     "myorg",
			wantProject: "myproject",
			wantRepo:    "myrepo",
			wantHost:    "ssh.dev.azure.com",
			wantSSH:     true,
		},
		{
			name:        "SSH URL with .git suffix",
			url:         "git@ssh.dev.azure.com:v3/myorg/myproject/myrepo.git",
			wantOrg:     "myorg",
			wantProject: "myproject",
			wantRepo:    "myrepo",
			wantHost:    "ssh.dev.azure.com",
			wantSSH:     true,
		},
		{
			name:    "GitHub URL - not ADO",
			url:     "https://github.com/owner/repo",
			wantNil: true,
		},
		{
			name:    "empty string",
			url:     "",
			wantNil: true,
		},
		{
			name:    "invalid ADO URL - missing _git",
			url:     "https://dev.azure.com/myorg/myproject/myrepo",
			wantNil: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := Parse(tt.url)

			if tt.wantNil {
				if result != nil {
					t.Errorf("Parse(%q) = %+v, want nil", tt.url, result)
				}
				return
			}

			if result == nil {
				t.Fatalf("Parse(%q) = nil, want non-nil", tt.url)
				return // unreachable but satisfies staticcheck SA5011
			}

			if result.Organization != tt.wantOrg {
				t.Errorf("Organization = %q, want %q", result.Organization, tt.wantOrg)
			}
			if result.Project != tt.wantProject {
				t.Errorf("Project = %q, want %q", result.Project, tt.wantProject)
			}
			if result.Repository != tt.wantRepo {
				t.Errorf("Repository = %q, want %q", result.Repository, tt.wantRepo)
			}
			if result.Host != tt.wantHost {
				t.Errorf("Host = %q, want %q", result.Host, tt.wantHost)
			}
			if result.IsSSH != tt.wantSSH {
				t.Errorf("IsSSH = %v, want %v", result.IsSSH, tt.wantSSH)
			}
		})
	}
}

func TestParseStrict(t *testing.T) {
	tests := []struct {
		name    string
		url     string
		wantErr bool
	}{
		{
			name:    "valid URL",
			url:     "https://dev.azure.com/myorg/myproject/_git/myrepo",
			wantErr: false,
		},
		{
			name:    "empty URL",
			url:     "",
			wantErr: true,
		},
		{
			name:    "not ADO URL",
			url:     "https://github.com/owner/repo",
			wantErr: true,
		},
		{
			name:    "invalid ADO URL format",
			url:     "https://dev.azure.com/invalid",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := ParseStrict(tt.url)

			if tt.wantErr {
				if err == nil {
					t.Errorf("ParseStrict(%q) error = nil, want error", tt.url)
				}
				return
			}

			if err != nil {
				t.Errorf("ParseStrict(%q) error = %v, want nil", tt.url, err)
				return
			}

			if result == nil {
				t.Errorf("ParseStrict(%q) = nil, want non-nil", tt.url)
			}
		})
	}
}

func TestParseFromSourceURL(t *testing.T) {
	tests := []struct {
		name        string
		url         string
		wantOrg     string
		wantProject string
		wantRepo    string
		wantErr     bool
	}{
		{
			name:        "valid source URL",
			url:         "https://dev.azure.com/myorg/myproject/_git/myrepo",
			wantOrg:     "myorg",
			wantProject: "myproject",
			wantRepo:    "myrepo",
			wantErr:     false,
		},
		{
			name:    "empty URL",
			url:     "",
			wantErr: true,
		},
		{
			name:    "missing _git segment",
			url:     "https://dev.azure.com/myorg/myproject/myrepo",
			wantErr: true,
		},
		{
			name:    "invalid URL",
			url:     "://invalid",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := ParseFromSourceURL(tt.url)

			if tt.wantErr {
				if err == nil {
					t.Errorf("ParseFromSourceURL(%q) error = nil, want error", tt.url)
				}
				return
			}

			if err != nil {
				t.Errorf("ParseFromSourceURL(%q) error = %v, want nil", tt.url, err)
				return
			}

			if result.Organization != tt.wantOrg {
				t.Errorf("Organization = %q, want %q", result.Organization, tt.wantOrg)
			}
			if result.Project != tt.wantProject {
				t.Errorf("Project = %q, want %q", result.Project, tt.wantProject)
			}
			if result.Repository != tt.wantRepo {
				t.Errorf("Repository = %q, want %q", result.Repository, tt.wantRepo)
			}
		})
	}
}

func TestParsedURL_FullSlug(t *testing.T) {
	p := &ParsedURL{
		Organization: "myorg",
		Project:      "myproject",
		Repository:   "myrepo",
	}

	expected := "myorg/myproject/myrepo"
	if result := p.FullSlug(); result != expected {
		t.Errorf("FullSlug() = %q, want %q", result, expected)
	}
}

func TestParsedURL_ProjectSlug(t *testing.T) {
	p := &ParsedURL{
		Organization: "myorg",
		Project:      "myproject",
		Repository:   "myrepo",
	}

	expected := "myorg/myproject"
	if result := p.ProjectSlug(); result != expected {
		t.Errorf("ProjectSlug() = %q, want %q", result, expected)
	}
}

// TestParseFromPackageScannerTestCases ensures compatibility with existing test cases
// from package_scanner_test.go
func TestParseFromPackageScannerTestCases(t *testing.T) {
	tests := []struct {
		name        string
		gitURL      string
		expectedOrg string
		expectedPrj string
		expectedRep string
	}{
		{
			name:        "standard ADO HTTPS URL",
			gitURL:      "https://dev.azure.com/contoso/MyProject/_git/MyRepo",
			expectedOrg: "contoso",
			expectedPrj: "MyProject",
			expectedRep: "MyRepo",
		},
		{
			name:        "ADO HTTPS with user prefix",
			gitURL:      "https://user@dev.azure.com/contoso/MyProject/_git/MyRepo",
			expectedOrg: "contoso",
			expectedPrj: "MyProject",
			expectedRep: "MyRepo",
		},
		{
			name:        "legacy VSTS URL",
			gitURL:      "https://contoso.visualstudio.com/DefaultProject/_git/ContosoRepo",
			expectedOrg: "contoso",
			expectedPrj: "DefaultProject",
			expectedRep: "ContosoRepo",
		},
		{
			name:        "ADO SSH URL",
			gitURL:      "git@ssh.dev.azure.com:v3/contoso/MyProject/MyRepo",
			expectedOrg: "contoso",
			expectedPrj: "MyProject",
			expectedRep: "MyRepo",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := Parse(tt.gitURL)
			if result == nil {
				t.Fatalf("Parse(%q) = nil, want non-nil", tt.gitURL)
				return // unreachable but satisfies staticcheck SA5011
			}

			if result.Organization != tt.expectedOrg {
				t.Errorf("Organization = %q, want %q", result.Organization, tt.expectedOrg)
			}
			if result.Project != tt.expectedPrj {
				t.Errorf("Project = %q, want %q", result.Project, tt.expectedPrj)
			}
			if result.Repository != tt.expectedRep {
				t.Errorf("Repository = %q, want %q", result.Repository, tt.expectedRep)
			}
		})
	}
}
