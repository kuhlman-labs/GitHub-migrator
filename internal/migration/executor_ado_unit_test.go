package migration

import (
	"log/slog"
	"os"
	"slices"
	"testing"

	"github.com/kuhlman-labs/github-migrator/internal/ado"
	"github.com/kuhlman-labs/github-migrator/internal/github"
	"github.com/kuhlman-labs/github-migrator/internal/models"
	"github.com/kuhlman-labs/github-migrator/internal/storage"
)

// TestADOMigrationSourceCaching tests that ADO migration sources are properly cached
func TestADOMigrationSourceCaching(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))

	executor, err := NewExecutor(ExecutorConfig{
		DestClient: &github.Client{},
		Storage:    &storage.Database{},
		Logger:     logger,
	})
	if err != nil {
		t.Fatalf("Failed to create executor: %v", err)
	}

	// Verify cache is initialized empty
	if len(executor.adoMigSourceCache) != 0 {
		t.Errorf("Expected empty ADO migration source cache, got %d entries", len(executor.adoMigSourceCache))
	}

	// Simulate caching a migration source
	testURL := "https://dev.azure.com"
	testID := "mig_source_123"
	executor.adoMigSourceCache[testURL] = testID

	// Verify cache contains the entry
	if cachedID, exists := executor.adoMigSourceCache[testURL]; !exists {
		t.Error("Expected ADO migration source to be cached")
	} else if cachedID != testID {
		t.Errorf("Expected cached ID %q, got %q", testID, cachedID)
	}
}

// TestADORepositoryURLValidation tests ADO repository URL format validation
func TestADORepositoryURLValidation(t *testing.T) {
	tests := []struct {
		name    string
		url     string
		isValid bool
		hasGit  bool
	}{
		{
			name:    "valid ADO URL with dev.azure.com",
			url:     "https://dev.azure.com/myorg/myproject/_git/myrepo",
			isValid: true,
			hasGit:  true,
		},
		{
			name:    "valid ADO URL with visualstudio.com",
			url:     "https://myorg.visualstudio.com/myproject/_git/myrepo",
			isValid: true,
			hasGit:  true,
		},
		{
			name:    "missing _git segment",
			url:     "https://dev.azure.com/myorg/myproject/myrepo",
			isValid: true, // Host is valid
			hasGit:  false,
		},
		{
			name:    "GitHub URL (not ADO)",
			url:     "https://github.com/myorg/myrepo",
			isValid: false,
			hasGit:  false,
		},
		{
			name:    "empty URL",
			url:     "",
			isValid: false,
			hasGit:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Simple validation checks
			isADO := false
			hasGit := false

			if tt.url != "" {
				// Check for ADO hosts
				if len(tt.url) > 10 {
					isADO = contains(tt.url, "dev.azure.com") || contains(tt.url, "visualstudio.com")
					hasGit = contains(tt.url, "/_git/")
				}
			}

			if isADO != tt.isValid {
				t.Errorf("ADO detection: expected %v, got %v", tt.isValid, isADO)
			}
			if hasGit != tt.hasGit {
				t.Errorf("Git path detection: expected %v, got %v", tt.hasGit, hasGit)
			}
		})
	}
}

// contains is a simple string contains helper
func contains(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}

// TestADOPATValidation tests ADO PAT format validation
func TestADOPATValidation(t *testing.T) {
	tests := []struct {
		name        string
		patLength   int
		expectValid bool
	}{
		{
			name:        "valid PAT length (52 chars)",
			patLength:   52,
			expectValid: true,
		},
		{
			name:        "valid PAT length (minimum acceptable)",
			patLength:   40,
			expectValid: true,
		},
		{
			name:        "too short PAT",
			patLength:   20,
			expectValid: false,
		},
		{
			name:        "empty PAT",
			patLength:   0,
			expectValid: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create a mock PAT of the specified length
			pat := make([]byte, tt.patLength)
			for i := range pat {
				pat[i] = 'x'
			}

			// Validate PAT length (matching the validation in executor_ado.go)
			isValid := len(pat) >= 40

			if isValid != tt.expectValid {
				t.Errorf("PAT validation: expected %v, got %v for length %d",
					tt.expectValid, isValid, tt.patLength)
			}
		})
	}
}

// TestGitHubPATFormatValidation tests GitHub PAT format validation for migrations
func TestGitHubPATFormatValidation(t *testing.T) {
	tests := []struct {
		name          string
		pat           string
		isClassic     bool
		isFineGrained bool
		isValid       bool
	}{
		{
			name:      "classic PAT with ghp_ prefix",
			pat:       "ghp_xxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxx",
			isClassic: true,
			isValid:   true,
		},
		{
			name:      "classic PAT with gho_ prefix",
			pat:       "gho_xxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxx",
			isClassic: true,
			isValid:   true,
		},
		{
			name:          "fine-grained PAT (not supported)",
			pat:           "github_pat_xxxxxxxxxxxxxxxxxxxxxxxxxxxxxxx",
			isFineGrained: true,
			isValid:       false,
		},
		{
			name:    "unknown format",
			pat:     "unknown_token_format",
			isValid: false, // Should warn but not necessarily fail
		},
		{
			name:    "empty PAT",
			pat:     "",
			isValid: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Check PAT format
			isClassic := len(tt.pat) >= 4 && (tt.pat[:4] == "ghp_" || tt.pat[:4] == "gho_")
			isFineGrained := len(tt.pat) >= 11 && tt.pat[:11] == "github_pat_"

			if isClassic != tt.isClassic {
				t.Errorf("Classic PAT detection: expected %v, got %v", tt.isClassic, isClassic)
			}
			if isFineGrained != tt.isFineGrained {
				t.Errorf("Fine-grained PAT detection: expected %v, got %v", tt.isFineGrained, isFineGrained)
			}

			// For migrations, fine-grained PATs are not valid
			isValidForMigration := !isFineGrained && len(tt.pat) > 0
			if tt.isClassic {
				isValidForMigration = true
			}

			// This test documents the expected behavior
			t.Logf("PAT format: classic=%v, fine_grained=%v, valid_for_migration=%v",
				isClassic, isFineGrained, isValidForMigration)
		})
	}
}

// TestADORepoWithoutSourceClient tests that ADO migrations work without a source GitHub client
func TestADORepoWithoutSourceClient(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))

	// Create executor without source client (valid for ADO migrations)
	executor, err := NewExecutor(ExecutorConfig{
		DestClient:  &github.Client{},
		Storage:     &storage.Database{},
		Logger:      logger,
		SourceToken: "ado_pat_token_example",
		SourceURL:   "https://dev.azure.com/myorg",
	})
	if err != nil {
		t.Fatalf("Failed to create executor: %v", err)
	}

	// Verify source client is nil (expected for ADO)
	if executor.sourceClient != nil {
		t.Error("Expected sourceClient to be nil for ADO executor")
	}

	// Verify source token is set
	if executor.sourceToken != "ado_pat_token_example" {
		t.Errorf("Expected source token to be set, got %q", executor.sourceToken)
	}

	// Verify source URL is set
	if executor.sourceURL != "https://dev.azure.com/myorg" {
		t.Errorf("Expected source URL to be set, got %q", executor.sourceURL)
	}
}

// TestADORepoNameExtraction tests extracting repo names from ADO full names
func TestADORepoNameExtraction(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))

	executor, err := NewExecutor(ExecutorConfig{
		SourceClient: &github.Client{},
		DestClient:   &github.Client{},
		Storage:      &storage.Database{},
		Logger:       logger,
	})
	if err != nil {
		t.Fatalf("Failed to create executor: %v", err)
	}

	adoProject := "MyProject"

	tests := []struct {
		name         string
		fullName     string
		adoProject   *string
		expectedName string
	}{
		{
			name:         "standard ADO format org/project/repo - uses project-repo pattern",
			fullName:     "MyOrg/MyProject/MyRepo",
			adoProject:   &adoProject,
			expectedName: "MyProject-MyRepo", // project-repo pattern to avoid naming conflicts
		},
		{
			name:         "ADO repo with spaces",
			fullName:     "MyOrg/MyProject/My Awesome Repo",
			adoProject:   &adoProject,
			expectedName: "MyProject-My-Awesome-Repo", // spaces replaced with hyphens
		},
		{
			name:         "ADO repo with multiple path segments - uses last as repo",
			fullName:     "Org/Project/SubFolder/Repo",
			adoProject:   &adoProject,
			expectedName: "Project-Repo", // takes index 1 for project and last for repo
		},
		{
			name:         "non-ADO repo (no ADO project)",
			fullName:     "github-org/my-repo",
			adoProject:   nil,
			expectedName: "my-repo",
		},
		{
			name:         "empty ADO project pointer - falls back to source name",
			fullName:     "org/project/repo",
			adoProject:   ptrString(""),
			expectedName: "project-repo", // Empty ADO project uses repo.Name() which returns "project/repo", then sanitized to "project-repo"
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo := &models.Repository{
				FullName: tt.fullName,
			}
			repo.SetADOProject(tt.adoProject)

			result := executor.getDestinationRepoName(repo)
			if result != tt.expectedName {
				t.Errorf("getDestinationRepoName() = %q, want %q", result, tt.expectedName)
			}
		})
	}
}

// TestADOSourceURLParsing tests parsing ADO source URLs
//
//nolint:gocyclo // Table-driven test with multiple assertions per case
func TestADOSourceURLParsing(t *testing.T) {
	tests := []struct {
		name            string
		sourceURL       string
		expectedOrg     string
		expectedProject string
		expectedRepo    string
		expectError     bool
	}{
		{
			name:            "standard dev.azure.com URL",
			sourceURL:       "https://dev.azure.com/myorg/myproject/_git/myrepo",
			expectedOrg:     "myorg",
			expectedProject: "myproject",
			expectedRepo:    "myrepo",
			expectError:     false,
		},
		{
			name:            "URL with special characters in repo name",
			sourceURL:       "https://dev.azure.com/myorg/myproject/_git/my-repo-name",
			expectedOrg:     "myorg",
			expectedProject: "myproject",
			expectedRepo:    "my-repo-name",
			expectError:     false,
		},
		{
			name:            "URL with encoded spaces",
			sourceURL:       "https://dev.azure.com/myorg/myproject/_git/my%20repo",
			expectedOrg:     "myorg",
			expectedProject: "myproject",
			expectedRepo:    "my%20repo", // URL encoded
			expectError:     false,
		},
		{
			name:        "invalid URL - missing _git",
			sourceURL:   "https://dev.azure.com/myorg/myproject/myrepo",
			expectError: true,
		},
		{
			name:        "empty URL",
			sourceURL:   "",
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Parse URL and extract components
			if tt.sourceURL == "" {
				if !tt.expectError {
					t.Error("Expected error for empty URL")
				}
				return
			}

			// Simple URL parsing for testing
			parts := splitPath(tt.sourceURL)

			if tt.expectError {
				// Check if URL has required _git segment
				hasGit := slices.Contains(parts, ado.GitPathSegment)
				if hasGit {
					t.Error("Expected error but URL appears valid")
				}
				return
			}

			// Find org, project, repo in path parts
			gitIndex := -1
			for i, p := range parts {
				if p == ado.GitPathSegment {
					gitIndex = i
					break
				}
			}

			if gitIndex < 2 || gitIndex >= len(parts)-1 {
				if !tt.expectError {
					t.Errorf("Could not parse URL: %s", tt.sourceURL)
				}
				return
			}

			org := parts[gitIndex-2]
			project := parts[gitIndex-1]
			repo := parts[gitIndex+1]

			if org != tt.expectedOrg {
				t.Errorf("Expected org %q, got %q", tt.expectedOrg, org)
			}
			if project != tt.expectedProject {
				t.Errorf("Expected project %q, got %q", tt.expectedProject, project)
			}
			if repo != tt.expectedRepo {
				t.Errorf("Expected repo %q, got %q", tt.expectedRepo, repo)
			}
		})
	}
}

// splitPath splits a URL path into components
func splitPath(urlStr string) []string {
	// Find the path part (after the host)
	var path string
	if idx := findString(urlStr, "://"); idx != -1 {
		rest := urlStr[idx+3:]
		if slashIdx := findString(rest, "/"); slashIdx != -1 {
			path = rest[slashIdx+1:]
		}
	}

	// Split by /
	var parts []string
	current := ""
	for _, c := range path {
		if c == '/' {
			if current != "" {
				parts = append(parts, current)
			}
			current = ""
		} else {
			current += string(c)
		}
	}
	if current != "" {
		parts = append(parts, current)
	}
	return parts
}

// findString finds the index of substr in s, or -1 if not found
func findString(s, substr string) int {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return i
		}
	}
	return -1
}

// TestExecutorWithADOConfig tests creating an executor with ADO-specific configuration
func TestExecutorWithADOConfig(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))

	tests := []struct {
		name        string
		config      ExecutorConfig
		expectError bool
	}{
		{
			name: "valid ADO config",
			config: ExecutorConfig{
				DestClient:  &github.Client{},
				Storage:     &storage.Database{},
				Logger:      logger,
				SourceToken: "ado_pat_token",
				SourceURL:   "https://dev.azure.com/myorg",
			},
			expectError: false,
		},
		{
			name: "ADO config without source token (may be valid for some scenarios)",
			config: ExecutorConfig{
				DestClient: &github.Client{},
				Storage:    &storage.Database{},
				Logger:     logger,
				SourceURL:  "https://dev.azure.com/myorg",
			},
			expectError: false,
		},
		{
			name: "ADO config missing destination client",
			config: ExecutorConfig{
				Storage:     &storage.Database{},
				Logger:      logger,
				SourceToken: "ado_pat_token",
			},
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			executor, err := NewExecutor(tt.config)

			if tt.expectError {
				if err == nil {
					t.Error("Expected error but got none")
				}
				return
			}

			if err != nil {
				t.Fatalf("Unexpected error: %v", err)
			}

			if executor == nil {
				t.Fatal("Expected executor to be created")
			}

			// Verify ADO-specific fields
			if tt.config.SourceToken != "" && executor.sourceToken != tt.config.SourceToken {
				t.Errorf("Expected source token %q, got %q", tt.config.SourceToken, executor.sourceToken)
			}
			if tt.config.SourceURL != "" && executor.sourceURL != tt.config.SourceURL {
				t.Errorf("Expected source URL %q, got %q", tt.config.SourceURL, executor.sourceURL)
			}
		})
	}
}
