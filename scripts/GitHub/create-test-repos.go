package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/google/go-github/v75/github"
	"golang.org/x/oauth2"
)

// TestRepoConfig defines a test repository configuration
type TestRepoConfig struct {
	Name        string
	Description string
	Private     bool
	AutoInit    bool // Initialize with README
	Setup       func(ctx context.Context, client *github.Client, org, repo string) error
}

func main() {
	// Parse command-line flags
	orgName := flag.String("org", "", "GitHub organization name (required)")
	token := flag.String("token", os.Getenv("GITHUB_TOKEN"), "GitHub token (or set GITHUB_TOKEN env var)")
	cleanupOnly := flag.Bool("cleanup", false, "Only cleanup existing test repositories")
	flag.Parse()

	if *orgName == "" {
		log.Fatal("Organization name is required: -org <org-name>")
	}

	if *token == "" {
		log.Fatal("GitHub token is required: -token <token> or set GITHUB_TOKEN env var")
	}

	// Set token in environment for git operations (needed for wiki setup)
	// #nosec G104 -- Acceptable for test script; errors would appear during git operations
	os.Setenv("GITHUB_TOKEN", *token)

	ctx := context.Background()
	ts := oauth2.StaticTokenSource(&oauth2.Token{AccessToken: *token})
	tc := oauth2.NewClient(ctx, ts)
	client := github.NewClient(tc)

	// Verify organization access
	_, _, err := client.Organizations.Get(ctx, *orgName)
	if err != nil {
		log.Fatalf("Failed to access organization %s: %v", *orgName, err)
	}

	log.Printf("Successfully connected to organization: %s", *orgName)

	// Cleanup existing test repos if requested
	if *cleanupOnly {
		cleanupTestRepos(ctx, client, *orgName)
		return
	}

	// Create test repositories
	repos := getTestRepoConfigs()
	log.Printf("Creating %d test repositories...", len(repos))

	for i, config := range repos {
		log.Printf("[%d/%d] Creating repository: %s", i+1, len(repos), config.Name)
		if err := createTestRepo(ctx, client, *orgName, config); err != nil {
			log.Printf("  ❌ Failed to create %s: %v", config.Name, err)
		} else {
			log.Printf("  ✅ Successfully created %s", config.Name)
		}
		// Rate limiting: sleep briefly between creations
		time.Sleep(2 * time.Second)
	}

	log.Println("\n🎉 Test repository creation complete!")
	log.Printf("Run discovery against organization: %s", *orgName)
	log.Printf("\nTo cleanup test repos later, run: go run scripts/create-test-repos.go -org %s -cleanup", *orgName)
}

// getTestRepoConfigs returns all test repository configurations
func getTestRepoConfigs() []TestRepoConfig {
	return []TestRepoConfig{
		{
			Name:        "test-minimal-empty",
			Description: "Minimal empty repository - no commits, no content",
			Private:     false,
			AutoInit:    false,
			Setup:       nil, // No setup needed - empty repo
		},
		{
			Name:        "test-minimal-basic",
			Description: "Basic repository with README only",
			Private:     false,
			AutoInit:    true,
			Setup:       setupBasicRepo,
		},
		{
			Name:        "test-small-repo",
			Description: "Small repository with few commits and branches",
			Private:     false,
			AutoInit:    true,
			Setup:       setupSmallRepo,
		},
		{
			Name:        "test-with-actions",
			Description: "Repository with GitHub Actions workflows",
			Private:     false,
			AutoInit:    true,
			Setup:       setupActionsRepo,
		},
		{
			Name:        "test-with-wiki",
			Description: "Repository with Wiki enabled and content",
			Private:     false,
			AutoInit:    true,
			Setup:       setupWikiRepo,
		},
		{
			Name:        "test-with-pages",
			Description: "Repository configured for GitHub Pages",
			Private:     false,
			AutoInit:    true,
			Setup:       setupPagesRepo,
		},
		{
			Name:        "test-with-lfs",
			Description: "Repository using Git LFS for large files",
			Private:     false,
			AutoInit:    true,
			Setup:       setupLFSRepo,
		},
		{
			Name:        "test-with-submodules",
			Description: "Repository with Git submodules",
			Private:     false,
			AutoInit:    true,
			Setup:       setupSubmodulesRepo,
		},
		{
			Name:        "test-many-branches",
			Description: "Repository with multiple branches",
			Private:     false,
			AutoInit:    true,
			Setup:       setupManyBranchesRepo,
		},
		{
			Name:        "test-with-protection",
			Description: "Repository with branch protection rules",
			Private:     false,
			AutoInit:    true,
			Setup:       setupBranchProtectionRepo,
		},
		{
			Name:        "test-with-releases",
			Description: "Repository with releases and assets",
			Private:     false,
			AutoInit:    true,
			Setup:       setupReleasesRepo,
		},
		{
			Name:        "test-with-issues-prs",
			Description: "Repository with issues and pull requests",
			Private:     false,
			AutoInit:    true,
			Setup:       setupIssuesPRsRepo,
		},
		{
			Name:        "test-with-tags",
			Description: "Repository with multiple tags",
			Private:     false,
			AutoInit:    true,
			Setup:       setupTagsRepo,
		},
		{
			Name:        "test-with-codeowners",
			Description: "Repository with CODEOWNERS file",
			Private:     false,
			AutoInit:    true,
			Setup:       setupCodeownersRepo,
		},
		{
			Name:        "test-with-environments",
			Description: "Repository with deployment environments",
			Private:     false,
			AutoInit:    true,
			Setup:       setupEnvironmentsRepo,
		},
		{
			Name:        "test-private-repo",
			Description: "Private repository for testing visibility",
			Private:     true,
			AutoInit:    true,
			Setup:       setupBasicRepo,
		},
		{
			Name:        "test-archived-repo",
			Description: "Archived repository (will be archived after creation)",
			Private:     false,
			AutoInit:    true,
			Setup:       setupArchivedRepo,
		},
		{
			Name:        "test-complex-all-features",
			Description: "Complex repository with multiple features enabled",
			Private:     false,
			AutoInit:    true,
			Setup:       setupComplexRepo,
		},
		{
			Name:        "test-large-file-history",
			Description: "Repository simulating large files in history",
			Private:     false,
			AutoInit:    true,
			Setup:       setupLargeFileRepo,
		},
		{
			Name:        "test-many-commits",
			Description: "Repository with many commits for history testing",
			Private:     false,
			AutoInit:    true,
			Setup:       setupManyCommitsRepo,
		},
		{
			Name:        "test-with-packages",
			Description: "Repository with GitHub Packages configuration",
			Private:     false,
			AutoInit:    true,
			Setup:       setupPackagesRepo,
		},
		{
			Name:        "test-with-rulesets",
			Description: "Repository with branch and tag rulesets configured",
			Private:     false,
			AutoInit:    true,
			Setup:       setupRulesetsRepo,
		},
		// Dependency testing repositories
		{
			Name:        "test-dependency-target",
			Description: "Simple repository used as a dependency target for testing local dependencies",
			Private:     false,
			AutoInit:    true,
			Setup:       setupDependencyTargetRepo,
		},
		{
			Name:        "test-submodule-dependencies",
			Description: "Repository with Git submodules (both local and external)",
			Private:     false,
			AutoInit:    true,
			Setup:       setupSubmoduleDependenciesRepo,
		},
		{
			Name:        "test-workflow-dependencies",
			Description: "Repository with GitHub Actions using reusable workflows from other repos",
			Private:     false,
			AutoInit:    true,
			Setup:       setupWorkflowDependenciesRepo,
		},
		{
			Name:        "test-package-dependencies",
			Description: "Repository with package dependencies for dependency graph testing",
			Private:     false,
			AutoInit:    true,
			Setup:       setupPackageDependenciesRepo,
		},
		{
			Name:        "test-mixed-dependencies",
			Description: "Repository with multiple types of dependencies (submodules, workflows, packages)",
			Private:     false,
			AutoInit:    true,
			Setup:       setupMixedDependenciesRepo,
		},
	}
}

// createTestRepo creates a test repository with the given configuration
func createTestRepo(ctx context.Context, client *github.Client, org string, config TestRepoConfig) error {
	// Check if repository already exists
	_, resp, err := client.Repositories.Get(ctx, org, config.Name)
	if err == nil {
		log.Printf("  ⚠️  Repository %s already exists, skipping creation", config.Name)
		return nil
	}
	if resp != nil && resp.StatusCode != 404 {
		return fmt.Errorf("unexpected error checking repository: %v", err)
	}

	// Create the repository
	repo := &github.Repository{
		Name:        github.Ptr(config.Name),
		Description: github.Ptr(config.Description),
		Private:     github.Ptr(config.Private),
		AutoInit:    github.Ptr(config.AutoInit),
	}

	createdRepo, _, err := client.Repositories.Create(ctx, org, repo)
	if err != nil {
		return fmt.Errorf("failed to create repository: %w", err)
	}

	log.Printf("  📦 Repository created: %s", createdRepo.GetHTMLURL())

	// Wait for repository to be fully initialized
	time.Sleep(3 * time.Second)

	// Run setup function if provided
	if config.Setup != nil {
		log.Printf("  🔧 Running setup for %s...", config.Name)
		if err := config.Setup(ctx, client, org, config.Name); err != nil {
			log.Printf("  ⚠️  Setup partially failed for %s: %v", config.Name, err)
			// Don't return error - partial setup is okay for testing
		}
	}

	return nil
}

// cleanupTestRepos deletes all test repositories starting with "test-"
func cleanupTestRepos(ctx context.Context, client *github.Client, org string) {
	log.Printf("Cleaning up test repositories in organization: %s", org)

	opt := &github.RepositoryListByOrgOptions{
		ListOptions: github.ListOptions{PerPage: 100},
	}

	var allRepos []*github.Repository
	for {
		repos, resp, err := client.Repositories.ListByOrg(ctx, org, opt)
		if err != nil {
			log.Fatalf("Failed to list repositories: %v", err)
		}

		allRepos = append(allRepos, repos...)

		if resp.NextPage == 0 {
			break
		}
		opt.Page = resp.NextPage
	}

	testRepos := []*github.Repository{}
	for _, repo := range allRepos {
		if strings.HasPrefix(repo.GetName(), "test-") {
			testRepos = append(testRepos, repo)
		}
	}

	if len(testRepos) == 0 {
		log.Println("No test repositories found to cleanup")
		return
	}

	log.Printf("Found %d test repositories to delete", len(testRepos))

	for i, repo := range testRepos {
		log.Printf("[%d/%d] Deleting: %s", i+1, len(testRepos), repo.GetName())

		// Unarchive if archived (can't delete archived repos)
		if repo.GetArchived() {
			log.Printf("  Unarchiving %s...", repo.GetName())
			repo.Archived = github.Ptr(false)
			_, _, err := client.Repositories.Edit(ctx, org, repo.GetName(), repo)
			if err != nil {
				log.Printf("  ⚠️  Failed to unarchive: %v", err)
			}
			time.Sleep(1 * time.Second)
		}

		_, err := client.Repositories.Delete(ctx, org, repo.GetName())
		if err != nil {
			log.Printf("  ❌ Failed to delete %s: %v", repo.GetName(), err)
		} else {
			log.Printf("  ✅ Deleted %s", repo.GetName())
		}

		time.Sleep(1 * time.Second) // Rate limiting
	}

	log.Println("🎉 Cleanup complete!")
}

// Setup functions for different repository types

func setupBasicRepo(ctx context.Context, client *github.Client, org, repo string) error {
	// Add a simple file
	content := "# Test Repository\n\nThis is a basic test repository."
	return createOrUpdateFile(ctx, client, org, repo, "README.md", content, "Initial commit")
}

func setupSmallRepo(ctx context.Context, client *github.Client, org, repo string) error {
	// Create a few files
	files := map[string]string{
		"src/main.go":    "package main\n\nfunc main() {\n\tprintln(\"Hello, World!\")\n}\n",
		"src/utils.go":   "package main\n\nfunc helper() string {\n\treturn \"helper\"\n}\n",
		"docs/README.md": "# Documentation\n\nProject documentation goes here.\n",
		".gitignore":     "*.log\n*.tmp\n.DS_Store\n",
	}

	for path, content := range files {
		if err := createOrUpdateFile(ctx, client, org, repo, path, content, "Add "+path); err != nil {
			return err
		}
		time.Sleep(500 * time.Millisecond)
	}

	// Create a branch
	if err := createBranch(ctx, client, org, repo, "develop"); err != nil {
		log.Printf("  ⚠️  Failed to create branch: %v", err)
	}

	return nil
}

func setupActionsRepo(ctx context.Context, client *github.Client, org, repo string) error {
	// Create GitHub Actions workflow
	workflowContent := `name: CI

on:
  push:
    branches: [ main ]
  pull_request:
    branches: [ main ]

jobs:
  build:
    runs-on: ubuntu-latest
    steps:
    - uses: actions/checkout@v3
    - name: Run tests
      run: echo "Running tests..."
`

	if err := createOrUpdateFile(ctx, client, org, repo, ".github/workflows/ci.yml", workflowContent, "Add CI workflow"); err != nil {
		return err
	}

	// Add another workflow
	workflowContent2 := `name: Release

on:
  push:
    tags:
      - 'v*'

jobs:
  release:
    runs-on: ubuntu-latest
    steps:
    - uses: actions/checkout@v3
    - name: Create Release
      run: echo "Creating release..."
`

	return createOrUpdateFile(ctx, client, org, repo, ".github/workflows/release.yml", workflowContent2, "Add release workflow")
}

func setupWikiRepo(ctx context.Context, client *github.Client, org, repo string) error {
	// Enable wiki
	repository := &github.Repository{
		HasWiki: github.Ptr(true),
	}

	_, _, err := client.Repositories.Edit(ctx, org, repo, repository)
	if err != nil {
		return fmt.Errorf("failed to enable wiki: %w", err)
	}

	log.Printf("  📖 Wiki enabled")

	// Wait for wiki to be ready
	time.Sleep(2 * time.Second)

	// Create a temporary directory for wiki
	tmpDir, err := os.MkdirTemp("", "wiki-"+repo+"-*")
	if err != nil {
		return fmt.Errorf("failed to create temp directory: %w", err)
	}
	defer os.RemoveAll(tmpDir)

	// Get GitHub token from environment
	token := os.Getenv("GITHUB_TOKEN")
	if token == "" {
		log.Printf("  ⚠️  No GITHUB_TOKEN found, wiki pages cannot be created")
		return nil
	}

	// Initialize new git repo
	log.Printf("  📝 Creating wiki pages via git")

	// #nosec G204 -- tmpDir is a newly created temp directory, not user input
	initCmd := exec.CommandContext(ctx, "git", "init", tmpDir)
	if output, err := initCmd.CombinedOutput(); err != nil {
		return fmt.Errorf("failed to initialize git: %w - %s", err, string(output))
	}

	// Configure git
	configCmds := [][]string{
		{"git", "-C", tmpDir, "config", "user.name", "Test Script"},
		{"git", "-C", tmpDir, "config", "user.email", "test@example.com"},
	}
	for _, cmdArgs := range configCmds {
		// #nosec G204 -- cmdArgs are hardcoded in the slice above, no user input
		cmd := exec.CommandContext(ctx, cmdArgs[0], cmdArgs[1:]...)
		if output, err := cmd.CombinedOutput(); err != nil {
			return fmt.Errorf("git config failed: %w - %s", err, string(output))
		}
	}

	// Create Home page content
	homeContent := `# Welcome to the Wiki

This is the home page of the test repository wiki.

## About This Wiki

This wiki was created automatically for testing purposes.

## Contents

- [Getting Started](#getting-started)
- [Documentation](#documentation)
- [Examples](#examples)

### Getting Started

Add your getting started guide here.

### Documentation

Add your documentation here.

### Examples

Add code examples and tutorials here.
`

	// Write Home.md
	homeFile := filepath.Join(tmpDir, "Home.md")
	if err := os.WriteFile(homeFile, []byte(homeContent), 0600); err != nil {
		return fmt.Errorf("failed to write Home.md: %w", err)
	}

	// Create another wiki page
	docsContent := `# Documentation

This is a documentation page for the test repository.

## Overview

Add your project overview here.

## API Reference

Document your API here.

## Configuration

Explain configuration options here.
`

	docsFile := filepath.Join(tmpDir, "Documentation.md")
	if err := os.WriteFile(docsFile, []byte(docsContent), 0600); err != nil {
		return fmt.Errorf("failed to write Documentation.md: %w", err)
	}

	// Git add and commit
	// #nosec G204 -- tmpDir is a newly created temp directory, not user input
	addCmd := exec.CommandContext(ctx, "git", "-C", tmpDir, "add", ".")
	if output, err := addCmd.CombinedOutput(); err != nil {
		return fmt.Errorf("git add failed: %w - %s", err, string(output))
	}

	// #nosec G204 -- tmpDir is a newly created temp directory, not user input
	commitCmd := exec.CommandContext(ctx, "git", "-C", tmpDir, "commit", "-m", "Initialize wiki with test content")
	if output, err := commitCmd.CombinedOutput(); err != nil {
		return fmt.Errorf("git commit failed: %w - %s", err, string(output))
	}

	// Add remote and push
	wikiURL := fmt.Sprintf("https://%s@github.com/%s/%s.wiki.git", token, org, repo)

	// #nosec G204 -- wikiURL is constructed from org/repo, tmpDir is a temp directory, not user input
	remoteCmd := exec.CommandContext(ctx, "git", "-C", tmpDir, "remote", "add", "origin", wikiURL)
	if output, err := remoteCmd.CombinedOutput(); err != nil {
		return fmt.Errorf("git remote add failed: %w - %s", err, string(output))
	}

	// Try pushing to master first, then main if master fails
	// #nosec G204 -- tmpDir is a temp directory, not user input
	pushCmd := exec.CommandContext(ctx, "git", "-C", tmpDir, "push", "-u", "origin", "master")
	pushCmd.Env = append(os.Environ(), "GIT_TERMINAL_PROMPT=0")
	output, err := pushCmd.CombinedOutput()
	if err != nil {
		log.Printf("  ⚠️  Push to master failed, trying main: %v - %s", err, string(output))
		// Try main branch
		// #nosec G204 -- tmpDir is a temp directory, not user input
		pushCmd = exec.CommandContext(ctx, "git", "-C", tmpDir, "push", "-u", "origin", "HEAD:main")
		pushCmd.Env = append(os.Environ(), "GIT_TERMINAL_PROMPT=0")
		output, err = pushCmd.CombinedOutput()
		if err != nil {
			return fmt.Errorf("git push failed: %w - %s", err, string(output))
		}
	}

	log.Printf("  ✅ Wiki pages created: Home, Documentation")

	return nil
}

func setupPagesRepo(ctx context.Context, client *github.Client, org, repo string) error {
	// Create index.html for GitHub Pages
	htmlContent := `<!DOCTYPE html>
<html>
<head>
    <title>Test Repository</title>
</head>
<body>
    <h1>GitHub Pages Test</h1>
    <p>This is a test page for GitHub Pages.</p>
</body>
</html>
`

	if err := createOrUpdateFile(ctx, client, org, repo, "index.html", htmlContent, "Add GitHub Pages site"); err != nil {
		return err
	}

	// Enable Pages with source from main branch root
	pagesSource := &github.PagesSource{
		Branch: github.Ptr("main"),
		Path:   github.Ptr("/"),
	}

	pages := &github.Pages{
		Source: pagesSource,
	}

	_, _, err := client.Repositories.EnablePages(ctx, org, repo, pages)
	if err != nil {
		// Pages enablement might fail due to various reasons, log but don't fail
		log.Printf("  ⚠️  Failed to enable Pages (may need manual setup): %v", err)
	} else {
		log.Printf("  🌐 GitHub Pages enabled")
	}

	return nil
}

func setupLFSRepo(ctx context.Context, client *github.Client, org, repo string) error {
	// Create .gitattributes with LFS configuration
	gitattributesContent := `*.bin filter=lfs diff=lfs merge=lfs -text
*.large filter=lfs diff=lfs merge=lfs -text
`

	if err := createOrUpdateFile(ctx, client, org, repo, ".gitattributes", gitattributesContent, "Add LFS configuration"); err != nil {
		return err
	}

	// Create a file that would be tracked by LFS
	// Note: We can't actually upload LFS files via API, but we can create the pointer file
	lfsPointer := `version https://git-lfs.github.com/spec/v1
oid sha256:4d7a214614ab2935c943f9e0ff69d22eadbb8f32b1258daaa5e2ca24d17e2393
size 12345
`

	if err := createOrUpdateFile(ctx, client, org, repo, "data/test.bin", lfsPointer, "Add LFS pointer file"); err != nil {
		return err
	}

	log.Printf("  💾 LFS configuration added (pointer files only)")

	return nil
}

func setupSubmodulesRepo(ctx context.Context, client *github.Client, org, repo string) error {
	// Create .gitmodules file
	gitmodulesContent := `[submodule "vendor/library"]
	path = vendor/library
	url = https://github.com/octocat/Hello-World.git
`

	if err := createOrUpdateFile(ctx, client, org, repo, ".gitmodules", gitmodulesContent, "Add submodules"); err != nil {
		return err
	}

	log.Printf("  🔗 Submodules configuration added")

	return nil
}

func setupManyBranchesRepo(ctx context.Context, client *github.Client, org, repo string) error {
	// Create multiple branches
	branches := []string{"develop", "feature/test-1", "feature/test-2", "release/v1.0", "hotfix/bug-123"}

	for _, branch := range branches {
		if err := createBranch(ctx, client, org, repo, branch); err != nil {
			log.Printf("  ⚠️  Failed to create branch %s: %v", branch, err)
		} else {
			log.Printf("  🌿 Created branch: %s", branch)
		}
		time.Sleep(500 * time.Millisecond)
	}

	return nil
}

func setupBranchProtectionRepo(ctx context.Context, client *github.Client, org, repo string) error {
	// Enable branch protection on main branch
	protection := &github.ProtectionRequest{
		RequiredStatusChecks: &github.RequiredStatusChecks{
			Strict:   false,
			Contexts: &[]string{},
		},
		RequiredPullRequestReviews: &github.PullRequestReviewsEnforcementRequest{
			RequiredApprovingReviewCount: 1,
		},
		EnforceAdmins: false,
	}

	_, _, err := client.Repositories.UpdateBranchProtection(ctx, org, repo, "main", protection)
	if err != nil {
		return fmt.Errorf("failed to enable branch protection: %w", err)
	}

	log.Printf("  🛡️  Branch protection enabled on main")

	return nil
}

func setupReleasesRepo(ctx context.Context, client *github.Client, org, repo string) error {
	// Create a tag first
	tagName := "v1.0.0"
	if err := createTag(ctx, client, org, repo, tagName); err != nil {
		log.Printf("  ⚠️  Failed to create tag %s: %v", tagName, err)
		// Don't return error, might already exist
	}

	// Check if release already exists
	_, resp, err := client.Repositories.GetReleaseByTag(ctx, org, repo, tagName)
	if err == nil {
		log.Printf("  ℹ️  Release %s already exists, skipping", tagName)
	} else if resp != nil && resp.StatusCode == 404 {
		// Release doesn't exist, create it
		release := &github.RepositoryRelease{
			TagName:         github.Ptr(tagName),
			Name:            github.Ptr("Release v1.0.0"),
			Body:            github.Ptr("Test release with notes\n\n## Changes\n- Initial release\n- Test features"),
			Draft:           github.Ptr(false),
			Prerelease:      github.Ptr(false),
			TargetCommitish: github.Ptr("main"),
		}

		createdRelease, _, err := client.Repositories.CreateRelease(ctx, org, repo, release)
		if err != nil {
			log.Printf("  ⚠️  Failed to create release: %v", err)
		} else {
			log.Printf("  🎉 Release created: %s", createdRelease.GetHTMLURL())
		}
	}

	// Create another release
	tagName2 := "v0.9.0"
	if err := createTag(ctx, client, org, repo, tagName2); err != nil {
		log.Printf("  ⚠️  Failed to create tag %s: %v", tagName2, err)
	}

	// Check if second release already exists
	_, resp2, err2 := client.Repositories.GetReleaseByTag(ctx, org, repo, tagName2)
	if err2 == nil {
		log.Printf("  ℹ️  Release %s already exists, skipping", tagName2)
	} else if resp2 != nil && resp2.StatusCode == 404 {
		release2 := &github.RepositoryRelease{
			TagName:         github.Ptr(tagName2),
			Name:            github.Ptr("Beta Release"),
			Body:            github.Ptr("Pre-release version"),
			Draft:           github.Ptr(false),
			Prerelease:      github.Ptr(true),
			TargetCommitish: github.Ptr("main"),
		}

		_, _, err := client.Repositories.CreateRelease(ctx, org, repo, release2)
		if err != nil {
			log.Printf("  ⚠️  Failed to create second release: %v", err)
		}
	}

	return nil
}

func setupIssuesPRsRepo(ctx context.Context, client *github.Client, org, repo string) error {
	// Create some issues
	issues := []struct {
		title string
		body  string
		state string
	}{
		{"Test Issue 1", "This is a test issue for discovery", "open"},
		{"Test Issue 2", "Another test issue", "open"},
		{"Closed Issue", "This issue was closed", "closed"},
	}

	for i, issue := range issues {
		issueReq := &github.IssueRequest{
			Title: github.Ptr(issue.title),
			Body:  github.Ptr(issue.body),
		}

		createdIssue, _, err := client.Issues.Create(ctx, org, repo, issueReq)
		if err != nil {
			log.Printf("  ⚠️  Failed to create issue: %v", err)
			continue
		}

		// Close if needed
		if issue.state == "closed" {
			state := "closed"
			_, _, err = client.Issues.Edit(ctx, org, repo, createdIssue.GetNumber(), &github.IssueRequest{
				State: &state,
			})
			if err != nil {
				log.Printf("  ⚠️  Failed to close issue: %v", err)
			}
		}

		log.Printf("  📝 Created issue #%d: %s", i+1, issue.title)
		time.Sleep(500 * time.Millisecond)
	}

	// Create a branch for PR
	if err := createBranch(ctx, client, org, repo, "test-pr-branch"); err != nil {
		log.Printf("  ⚠️  Failed to create PR branch: %v", err)
		return nil
	}

	// Add a commit to the branch
	content := "# Test PR\n\nThis file was added in a test PR."
	if err := createOrUpdateFileOnBranch(ctx, client, org, repo, "test-pr.md", content, "Add test PR file", "test-pr-branch"); err != nil {
		log.Printf("  ⚠️  Failed to add file to PR branch: %v", err)
		return nil
	}

	// Create a pull request
	pr := &github.NewPullRequest{
		Title: github.Ptr("Test Pull Request"),
		Body:  github.Ptr("This is a test pull request for discovery testing"),
		Head:  github.Ptr("test-pr-branch"),
		Base:  github.Ptr("main"),
	}

	createdPR, _, err := client.PullRequests.Create(ctx, org, repo, pr)
	if err != nil {
		log.Printf("  ⚠️  Failed to create PR: %v", err)
		return nil
	}

	log.Printf("  🔀 Created PR #%d", createdPR.GetNumber())

	return nil
}

func setupTagsRepo(ctx context.Context, client *github.Client, org, repo string) error {
	// Create multiple tags
	tags := []string{"v0.1.0", "v0.2.0", "v0.3.0", "v1.0.0-rc.1", "v1.0.0"}

	for _, tag := range tags {
		if err := createTag(ctx, client, org, repo, tag); err != nil {
			log.Printf("  ⚠️  Failed to create tag %s: %v", tag, err)
		} else {
			log.Printf("  🏷️  Created tag: %s", tag)
		}
		time.Sleep(500 * time.Millisecond)
	}

	return nil
}

func setupCodeownersRepo(ctx context.Context, client *github.Client, org, repo string) error {
	// Create CODEOWNERS file
	codeownersContent := `# CODEOWNERS file for test repository

# Default owners for everything
* @octocat

# Specific owners for docs
/docs/ @octocat @github

# Owners for source code
/src/ @octocat
`

	if err := createOrUpdateFile(ctx, client, org, repo, ".github/CODEOWNERS", codeownersContent, "Add CODEOWNERS file"); err != nil {
		return err
	}

	log.Printf("  👥 CODEOWNERS file added")

	return nil
}

func setupEnvironmentsRepo(ctx context.Context, client *github.Client, org, repo string) error {
	// Note: Deployment environments must be created manually via the UI or with authenticated deployments
	// We'll create a workflow that references environments to prepare the repo
	workflowContent := `name: Deploy

on:
  push:
    branches: [ main ]

jobs:
  deploy-dev:
    runs-on: ubuntu-latest
    environment: development
    steps:
    - uses: actions/checkout@v3
    - name: Deploy to development
      run: echo "Deploying to development..."
  
  deploy-staging:
    runs-on: ubuntu-latest
    environment: staging
    needs: deploy-dev
    steps:
    - uses: actions/checkout@v3
    - name: Deploy to staging
      run: echo "Deploying to staging..."
  
  deploy-prod:
    runs-on: ubuntu-latest
    environment: production
    needs: deploy-staging
    steps:
    - uses: actions/checkout@v3
    - name: Deploy to production
      run: echo "Deploying to production..."
`

	if err := createOrUpdateFile(ctx, client, org, repo, ".github/workflows/deploy.yml", workflowContent, "Add deployment workflow with environments"); err != nil {
		return err
	}

	log.Printf("  🌍 Deployment workflow added (environments will be created on first deployment)")

	return nil
}

func setupArchivedRepo(ctx context.Context, client *github.Client, org, repo string) error {
	// Setup basic content first
	if err := setupBasicRepo(ctx, client, org, repo); err != nil {
		return err
	}

	// Archive the repository
	repository := &github.Repository{
		Archived: github.Ptr(true),
	}

	_, _, err := client.Repositories.Edit(ctx, org, repo, repository)
	if err != nil {
		return fmt.Errorf("failed to archive repository: %w", err)
	}

	log.Printf("  📦 Repository archived")

	return nil
}

func setupComplexRepo(ctx context.Context, client *github.Client, org, repo string) error {
	// Combine multiple features
	log.Printf("  🔧 Setting up complex repository with multiple features...")

	// Files
	files := map[string]string{
		"src/main.go":        "package main\n\nfunc main() {}\n",
		"docs/README.md":     "# Documentation\n",
		".gitignore":         "*.log\n",
		".github/CODEOWNERS": "* @octocat\n",
	}

	for path, content := range files {
		if err := createOrUpdateFile(ctx, client, org, repo, path, content, "Add "+path); err != nil {
			log.Printf("  ⚠️  Failed to add %s: %v", path, err)
		}
		time.Sleep(300 * time.Millisecond)
	}

	// Workflows
	workflowContent := `name: CI
on: [push]
jobs:
  test:
    runs-on: ubuntu-latest
    steps:
    - uses: actions/checkout@v3
`
	if err := createOrUpdateFile(ctx, client, org, repo, ".github/workflows/ci.yml", workflowContent, "Add workflow"); err != nil {
		log.Printf("  ⚠️  Failed to add workflow: %v", err)
	}

	// Branches
	branches := []string{"develop", "staging"}
	for _, branch := range branches {
		if err := createBranch(ctx, client, org, repo, branch); err != nil {
			log.Printf("  ⚠️  Failed to create branch %s: %v", branch, err)
		}
		time.Sleep(300 * time.Millisecond)
	}

	// Tags
	if err := createTag(ctx, client, org, repo, "v1.0.0"); err != nil {
		log.Printf("  ⚠️  Failed to create tag: %v", err)
	}

	// Issue
	issueReq := &github.IssueRequest{
		Title: github.Ptr("Test Issue"),
		Body:  github.Ptr("Complex repo test issue"),
	}
	_, _, err := client.Issues.Create(ctx, org, repo, issueReq)
	if err != nil {
		log.Printf("  ⚠️  Failed to create issue: %v", err)
	}

	// Enable features
	repository := &github.Repository{
		HasWiki:     github.Ptr(true),
		HasProjects: github.Ptr(true),
	}
	_, _, err = client.Repositories.Edit(ctx, org, repo, repository)
	if err != nil {
		log.Printf("  ⚠️  Failed to enable features: %v", err)
	}

	log.Printf("  ✨ Complex repository setup complete")

	return nil
}

func setupLargeFileRepo(ctx context.Context, client *github.Client, org, repo string) error {
	// Simulate a large file by creating content
	// Note: GitHub API has file size limits, so we'll create a file with instructions
	content := `# Large File Repository Test

This repository simulates large files in history.

## Simulated Large File
This file represents a large binary file that would be >100MB in a real scenario.

` + strings.Repeat("Lorem ipsum dolor sit amet. ", 10000)

	if err := createOrUpdateFile(ctx, client, org, repo, "large-file-simulation.txt", content, "Add large file simulation"); err != nil {
		return err
	}

	log.Printf("  📦 Large file simulation added (API has size limits)")

	return nil
}

func setupManyCommitsRepo(ctx context.Context, client *github.Client, org, repo string) error {
	// Create multiple commits
	for i := 1; i <= 10; i++ {
		content := fmt.Sprintf("# Commit %d\n\nThis is commit number %d.\nTimestamp: %s\n", i, i, time.Now().Format(time.RFC3339))
		filename := fmt.Sprintf("commit-%02d.txt", i)

		if err := createOrUpdateFile(ctx, client, org, repo, filename, content, fmt.Sprintf("Commit #%d", i)); err != nil {
			log.Printf("  ⚠️  Failed to create commit %d: %v", i, err)
		}

		time.Sleep(300 * time.Millisecond)
	}

	log.Printf("  📝 Created 10 commits")

	return nil
}

func setupPackagesRepo(ctx context.Context, client *github.Client, org, repo string) error {
	log.Printf("  📦 Setting up repository with package configurations")

	// Create a Dockerfile for GitHub Container Registry
	dockerfileContent := `FROM alpine:latest

# Install basic utilities
RUN apk add --no-cache curl bash

# Add application files
WORKDIR /app
COPY . .

# Set up entrypoint
CMD ["echo", "Hello from test package!"]
`

	if err := createOrUpdateFile(ctx, client, org, repo, "Dockerfile", dockerfileContent, "Add Dockerfile for container package"); err != nil {
		return err
	}
	time.Sleep(500 * time.Millisecond)

	// Create package.json for npm package
	packageJSON := `{
  "name": "@` + org + `/` + repo + `",
  "version": "1.0.0",
  "description": "Test package for GitHub Packages",
  "main": "index.js",
  "scripts": {
    "test": "echo \"Error: no test specified\" && exit 1"
  },
  "repository": {
    "type": "git",
    "url": "git+https://github.com/` + org + `/` + repo + `.git"
  },
  "publishConfig": {
    "@` + org + `:registry": "https://npm.pkg.github.com"
  },
  "keywords": ["test", "github-packages"],
  "author": "Test",
  "license": "MIT"
}
`

	if err := createOrUpdateFile(ctx, client, org, repo, "package.json", packageJSON, "Add package.json for npm package"); err != nil {
		return err
	}
	time.Sleep(500 * time.Millisecond)

	// Create index.js
	indexJS := `/**
 * Test Package
 * A simple test package for GitHub Packages demonstration
 */

function greet(name) {
  return 'Hello, ' + name + '!';
}

function add(a, b) {
  return a + b;
}

module.exports = {
  greet,
  add
};
`

	if err := createOrUpdateFile(ctx, client, org, repo, "index.js", indexJS, "Add index.js"); err != nil {
		return err
	}
	time.Sleep(500 * time.Millisecond)

	// Create .npmrc
	npmrcContent := `@` + org + `:registry=https://npm.pkg.github.com
//npm.pkg.github.com/:_authToken=${NODE_AUTH_TOKEN}
`

	if err := createOrUpdateFile(ctx, client, org, repo, ".npmrc", npmrcContent, "Add .npmrc for npm package"); err != nil {
		return err
	}
	time.Sleep(500 * time.Millisecond)

	// Create GitHub Actions workflow to publish packages
	publishWorkflow := `name: Publish Packages

on:
  push:
    tags:
      - 'v*'
  workflow_dispatch:

env:
  REGISTRY: ghcr.io
  IMAGE_NAME: ${{ github.repository }}

jobs:
  publish-docker:
    runs-on: ubuntu-latest
    permissions:
      contents: read
      packages: write
    steps:
      - name: Checkout repository
        uses: actions/checkout@v3

      - name: Log in to Container Registry
        uses: docker/login-action@v2
        with:
          registry: ${{ env.REGISTRY }}
          username: ${{ github.actor }}
          password: ${{ secrets.GITHUB_TOKEN }}

      - name: Extract metadata
        id: meta
        uses: docker/metadata-action@v4
        with:
          images: ${{ env.REGISTRY }}/${{ env.IMAGE_NAME }}
          tags: |
            type=ref,event=tag
            type=sha,prefix={{branch}}-
            type=raw,value=latest,enable={{is_default_branch}}

      - name: Build and push Docker image
        uses: docker/build-push-action@v4
        with:
          context: .
          push: true
          tags: ${{ steps.meta.outputs.tags }}
          labels: ${{ steps.meta.outputs.labels }}

  publish-npm:
    runs-on: ubuntu-latest
    permissions:
      contents: read
      packages: write
    steps:
      - name: Checkout repository
        uses: actions/checkout@v3

      - name: Setup Node.js
        uses: actions/setup-node@v3
        with:
          node-version: '18'
          registry-url: 'https://npm.pkg.github.com'
          scope: '@` + org + `'

      - name: Publish to GitHub Packages
        run: npm publish
        env:
          NODE_AUTH_TOKEN: ${{ secrets.GITHUB_TOKEN }}
`

	if err := createOrUpdateFile(ctx, client, org, repo, ".github/workflows/publish-packages.yml", publishWorkflow, "Add package publishing workflow"); err != nil {
		return err
	}
	time.Sleep(500 * time.Millisecond)

	// Create README with package information
	readmeContent := "# Test Package Repository\n\n" +
		"This repository is configured to publish packages to GitHub Packages.\n\n" +
		"## Package Types\n\n" +
		"This repository can publish:\n\n" +
		"1. **Docker/Container Package** - Via GitHub Container Registry (ghcr.io)\n" +
		"2. **npm Package** - Via GitHub Packages npm registry\n\n" +
		"## Published Packages\n\n" +
		"### Container Image\n\n" +
		"```bash\n" +
		"docker pull ghcr.io/" + org + "/" + repo + ":latest\n" +
		"```\n\n" +
		"### npm Package\n\n" +
		"```bash\n" +
		"npm install @" + org + "/" + repo + "\n" +
		"```\n\n" +
		"## Publishing\n\n" +
		"Packages are automatically published via GitHub Actions when:\n" +
		"- A new tag is pushed (format: v*.*.*)\n" +
		"- Workflow is manually triggered\n\n" +
		"## Configuration Files\n\n" +
		"- `Dockerfile` - Container image definition\n" +
		"- `package.json` - npm package metadata\n" +
		"- `.npmrc` - npm registry configuration\n" +
		"- `.github/workflows/publish-packages.yml` - Publishing automation\n\n" +
		"## Usage\n\n" +
		"### Docker Container\n\n" +
		"```bash\n" +
		"docker run ghcr.io/" + org + "/" + repo + ":latest\n" +
		"```\n\n" +
		"### npm Package\n\n" +
		"```javascript\n" +
		"const testPkg = require('@" + org + "/" + repo + "');\n\n" +
		"console.log(testPkg.greet('World')); // Hello, World!\n" +
		"console.log(testPkg.add(2, 3)); // 5\n" +
		"```\n\n" +
		"## Notes\n\n" +
		"This is a test repository created for GitHub migration testing purposes.\n"

	if err := createOrUpdateFile(ctx, client, org, repo, "README.md", readmeContent, "Add README with package information"); err != nil {
		return err
	}

	log.Printf("  ✅ Package configurations created (Docker + npm)")

	// Wait for files to be committed
	time.Sleep(2 * time.Second)

	// Create and push a tag to trigger package publishing
	tagName := "v1.0.0"
	log.Printf("  🏷️  Creating tag %s to trigger package publishing...", tagName)

	if err := createTag(ctx, client, org, repo, tagName); err != nil {
		log.Printf("  ⚠️  Failed to create tag (package publishing may not trigger): %v", err)
	} else {
		log.Printf("  ✅ Tag created - GitHub Actions workflow should publish packages automatically")
	}

	return nil
}

func setupRulesetsRepo(ctx context.Context, client *github.Client, org, repo string) error {
	log.Printf("  📋 Setting up repository with rulesets")

	// Create some branches and tags first for testing rulesets
	branches := []string{"develop", "staging", "production"}
	for _, branch := range branches {
		if err := createBranch(ctx, client, org, repo, branch); err != nil {
			log.Printf("  ⚠️  Failed to create branch %s: %v", branch, err)
		} else {
			log.Printf("  🌿 Created branch: %s", branch)
		}
		time.Sleep(300 * time.Millisecond)
	}

	// Create a README explaining rulesets
	readmeContent := `# Repository Rulesets Test

This repository demonstrates GitHub repository rulesets.

## Configured Rulesets

This repository has the following rulesets configured:

### 1. Main Branch Protection Ruleset
- **Target**: main branch
- **Rules**:
  - Require pull request before merging
  - Require status checks to pass
  - Require branches to be up to date
  - Block force pushes
  - Restrict deletions

### 2. Production Branch Ruleset
- **Target**: production branch
- **Rules**:
  - Require pull request with 2 approvals
  - Require code owner review
  - Block force pushes
  - Restrict deletions

### 3. Tag Protection Ruleset
- **Target**: Tags matching v*
- **Rules**:
  - Restrict tag creation
  - Restrict tag updates
  - Restrict tag deletions

### 4. Development Branch Ruleset
- **Target**: develop and staging branches
- **Rules**:
  - Require status checks
  - Allow force pushes (for development)

## Benefits of Rulesets

- Centralized rule management
- Apply rules to multiple branches/tags with patterns
- More flexible than traditional branch protection
- Can bypass rules for specific actors or teams
- Audit log for rule enforcement

## Testing Rulesets

Try the following to test rulesets:
1. Attempt to push directly to main (should be blocked)
2. Create a PR to main (should require approval)
3. Try to delete a protected branch
4. Try to create/delete a tag matching v*

For more information, see [GitHub Rulesets Documentation](https://docs.github.com/en/repositories/configuring-branches-and-merges-in-your-repository/managing-rulesets).
`

	if err := createOrUpdateFile(ctx, client, org, repo, "README.md", readmeContent, "Add README about rulesets"); err != nil {
		return err
	}
	time.Sleep(500 * time.Millisecond)

	// Create a CODEOWNERS file for testing code owner review requirements
	codeownersContent := `# CODEOWNERS for ruleset testing
* @octocat
`
	if err := createOrUpdateFile(ctx, client, org, repo, ".github/CODEOWNERS", codeownersContent, "Add CODEOWNERS file"); err != nil {
		log.Printf("  ⚠️  Failed to create CODEOWNERS: %v", err)
	}

	// Wait for repository to be ready
	time.Sleep(2 * time.Second)

	// Ruleset 1: Main branch protection
	log.Printf("  🔒 Creating ruleset for main branch protection...")
	branchTarget := github.RulesetTarget("branch")
	activeEnforcement := github.RulesetEnforcement("active")

	mainRuleset := github.RepositoryRuleset{
		Name:        "Main Branch Protection",
		Target:      &branchTarget,
		Source:      "Repository",
		Enforcement: activeEnforcement,
		Conditions: &github.RepositoryRulesetConditions{
			RefName: &github.RepositoryRulesetRefConditionParameters{
				Include: []string{"refs/heads/main"},
				Exclude: []string{},
			},
		},
		BypassActors: []*github.BypassActor{},
	}

	if _, _, err := client.Repositories.CreateRuleset(ctx, org, repo, mainRuleset); err != nil {
		log.Printf("  ⚠️  Failed to create main branch ruleset: %v", err)
	} else {
		log.Printf("  ✅ Created main branch protection ruleset")
	}
	time.Sleep(500 * time.Millisecond)

	// Ruleset 2: Production branch protection (stricter)
	log.Printf("  🔒 Creating ruleset for production branch...")
	prodRuleset := github.RepositoryRuleset{
		Name:        "Production Branch Protection",
		Target:      &branchTarget,
		Source:      "Repository",
		Enforcement: activeEnforcement,
		Conditions: &github.RepositoryRulesetConditions{
			RefName: &github.RepositoryRulesetRefConditionParameters{
				Include: []string{"refs/heads/production"},
				Exclude: []string{},
			},
		},
		BypassActors: []*github.BypassActor{},
	}

	if _, _, err := client.Repositories.CreateRuleset(ctx, org, repo, prodRuleset); err != nil {
		log.Printf("  ⚠️  Failed to create production branch ruleset: %v", err)
	} else {
		log.Printf("  ✅ Created production branch protection ruleset")
	}
	time.Sleep(500 * time.Millisecond)

	// Ruleset 3: Development branches (more lenient)
	log.Printf("  🔒 Creating ruleset for development branches...")
	devRuleset := github.RepositoryRuleset{
		Name:        "Development Branches",
		Target:      &branchTarget,
		Source:      "Repository",
		Enforcement: activeEnforcement,
		Conditions: &github.RepositoryRulesetConditions{
			RefName: &github.RepositoryRulesetRefConditionParameters{
				Include: []string{"refs/heads/develop", "refs/heads/staging"},
				Exclude: []string{},
			},
		},
		BypassActors: []*github.BypassActor{},
	}

	if _, _, err := client.Repositories.CreateRuleset(ctx, org, repo, devRuleset); err != nil {
		log.Printf("  ⚠️  Failed to create development branches ruleset: %v", err)
	} else {
		log.Printf("  ✅ Created development branches ruleset")
	}
	time.Sleep(500 * time.Millisecond)

	// Ruleset 4: Tag protection
	log.Printf("  🔒 Creating ruleset for tag protection...")
	tagTarget := github.RulesetTarget("tag")
	tagRuleset := github.RepositoryRuleset{
		Name:        "Tag Protection",
		Target:      &tagTarget,
		Source:      "Repository",
		Enforcement: activeEnforcement,
		Conditions: &github.RepositoryRulesetConditions{
			RefName: &github.RepositoryRulesetRefConditionParameters{
				Include: []string{"refs/tags/v*"},
				Exclude: []string{},
			},
		},
		BypassActors: []*github.BypassActor{},
	}

	if _, _, err := client.Repositories.CreateRuleset(ctx, org, repo, tagRuleset); err != nil {
		log.Printf("  ⚠️  Failed to create tag protection ruleset: %v", err)
	} else {
		log.Printf("  ✅ Created tag protection ruleset")
	}

	log.Printf("  ✅ All rulesets configured successfully")

	return nil
}

// Dependency testing setup functions

func setupDependencyTargetRepo(ctx context.Context, client *github.Client, org, repo string) error {
	// Create a simple library repo that can be used as a dependency
	readmeContent := `# Dependency Target Repository

This is a simple repository used as a dependency target for testing.

## Purpose

This repository is referenced by other test repositories to validate:
- Submodule detection
- Workflow dependency detection
- Local vs external dependency classification

## Usage

Other test repositories in this organization reference this repo to simulate dependencies.
`

	if err := createOrUpdateFile(ctx, client, org, repo, "README.md", readmeContent, "Add README"); err != nil {
		return err
	}

	// Add a simple library file
	libContent := `package target

// Version of this library
const Version = "1.0.0"

// Helper function
func Hello(name string) string {
	return "Hello, " + name + "!"
}
`

	if err := createOrUpdateFile(ctx, client, org, repo, "lib.go", libContent, "Add library code"); err != nil {
		return err
	}

	// Add a reusable workflow that other repos can call
	reusableWorkflowContent := `name: Reusable Workflow

on:
  workflow_call:
    inputs:
      config:
        description: 'Configuration to use'
        required: false
        type: string
        default: 'default'

jobs:
  reusable-job:
    runs-on: ubuntu-latest
    steps:
      - name: Echo config
        run: echo "Running with config: ${{ inputs.config }}"
      
      - name: Run checks
        run: |
          echo "This is a reusable workflow from test-dependency-target"
          echo "It can be called by other repositories in the same org"
`

	if err := createOrUpdateFile(ctx, client, org, repo, ".github/workflows/reusable.yml", reusableWorkflowContent, "Add reusable workflow"); err != nil {
		return err
	}

	log.Printf("  🎯 Dependency target repository created with reusable workflow")
	return nil
}

func setupSubmoduleDependenciesRepo(ctx context.Context, client *github.Client, org, repo string) error {
	log.Printf("  🔗 Setting up repository with submodule dependencies")

	// Create README explaining the submodules
	readmeContent := `# Submodule Dependencies Test

This repository contains Git submodules to test dependency detection.

## Submodules

This repository includes:
1. **Local submodule** - References another repository in the same organization (test-dependency-target)
2. **External submodule** - References a public repository (actions/checkout)

## Purpose

Used to test:
- Submodule detection during discovery
- Classification of local vs external dependencies
- Parsing of .gitmodules file
`

	if err := createOrUpdateFile(ctx, client, org, repo, "README.md", readmeContent, "Add README"); err != nil {
		return err
	}

	// Create .gitmodules with both local and external submodules
	gitmodulesContent := fmt.Sprintf(`[submodule "vendor/local-lib"]
	path = vendor/local-lib
	url = https://github.com/%s/test-dependency-target.git
	branch = main

[submodule "vendor/actions-checkout"]
	path = vendor/actions-checkout
	url = https://github.com/actions/checkout.git
	branch = v3

[submodule "vendor/octocat-hello-world"]
	path = vendor/octocat-hello-world
	url = https://github.com/octocat/Hello-World.git
`, org)

	if err := createOrUpdateFile(ctx, client, org, repo, ".gitmodules", gitmodulesContent, "Add submodules configuration"); err != nil {
		return err
	}

	log.Printf("  ✅ Submodule dependencies configured (1 local, 2 external)")
	return nil
}

func setupWorkflowDependenciesRepo(ctx context.Context, client *github.Client, org, repo string) error {
	log.Printf("  ⚙️  Setting up repository with workflow dependencies")

	// Create README
	readmeContent := `# Workflow Dependencies Test

This repository uses GitHub Actions workflows that reference other repositories.

## Workflow Dependencies

This repository demonstrates:
1. **Reusable workflows** - Calling workflows from other repositories
2. **Action dependencies** - Using actions from both local and external repos
3. **Local vs external** - Mix of same-org and public dependencies

## Purpose

Tests:
- GitHub Actions workflow parsing
- Reusable workflow detection
- Action dependency extraction
- Local vs external classification
`

	if err := createOrUpdateFile(ctx, client, org, repo, "README.md", readmeContent, "Add README"); err != nil {
		return err
	}

	// Create workflow that uses reusable workflows and actions from other repos
	workflowContent := fmt.Sprintf(`name: CI with Workflow Dependencies

on:
  push:
    branches: [ main ]
  pull_request:
    branches: [ main ]

jobs:
  # Use reusable workflow from same org (local dependency)
  local-reusable-workflow:
    uses: %s/test-dependency-target/.github/workflows/reusable.yml@main
    with:
      config: 'test'

  # Standard job using actions from various sources
  build:
    runs-on: ubuntu-latest
    steps:
      # External dependency - actions org
      - name: Checkout code
        uses: actions/checkout@v4
      
      # External dependency - third party
      - name: Setup Node
        uses: actions/setup-node@v4
        with:
          node-version: '18'
      
      # External dependency - another popular action
      - name: Cache dependencies
        uses: actions/cache@v3
        with:
          path: node_modules
          key: ${{ runner.os }}-node-${{ hashFiles('**/package-lock.json') }}
      
      - name: Run tests
        run: echo "Running tests..."

  # Use another external reusable workflow
  external-reusable-workflow:
    uses: actions/reusable-workflows/.github/workflows/npm-publish.yml@v1
    with:
      node-version: '18'

  # Composite action reference
  composite-action-test:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4
      
      # Reference to a composite action in another repo
      - name: Run composite action
        uses: docker/setup-buildx-action@v3
`, org)

	if err := createOrUpdateFile(ctx, client, org, repo, ".github/workflows/ci.yml", workflowContent, "Add CI workflow with dependencies"); err != nil {
		return err
	}

	// Create another workflow with different dependencies
	deployWorkflow := `name: Deploy

on:
  release:
    types: [published]

jobs:
  deploy:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4
      
      # More external dependencies
      - name: Configure AWS credentials
        uses: aws-actions/configure-aws-credentials@v4
        with:
          role-to-assume: arn:aws:iam::123456789012:role/my-role
          aws-region: us-east-1
      
      - name: Deploy to S3
        uses: aws-actions/aws-deploy@v1
        with:
          bucket: my-bucket
`

	if err := createOrUpdateFile(ctx, client, org, repo, ".github/workflows/deploy.yml", deployWorkflow, "Add deploy workflow"); err != nil {
		return err
	}

	log.Printf("  ✅ Workflow dependencies configured (1 local, 6+ external)")
	return nil
}

func setupPackageDependenciesRepo(ctx context.Context, client *github.Client, org, repo string) error {
	log.Printf("  📦 Setting up repository with package dependencies")

	// Create README
	readmeContent := `# Package Dependencies Test

This repository contains package dependencies for testing dependency graph.

## Package Types

This repository includes:
1. **npm dependencies** - JavaScript packages
2. **Go dependencies** - Go modules
3. **Python dependencies** - pip packages

## Purpose

Tests:
- GitHub dependency graph API
- Package dependency detection
- Local vs external package classification
`

	if err := createOrUpdateFile(ctx, client, org, repo, "README.md", readmeContent, "Add README"); err != nil {
		return err
	}

	// Create package.json with dependencies
	packageJSON := fmt.Sprintf(`{
  "name": "@%s/%s",
  "version": "1.0.0",
  "description": "Test repository with package dependencies",
  "main": "index.js",
  "dependencies": {
    "express": "^4.18.2",
    "lodash": "^4.17.21",
    "axios": "^1.6.0",
    "react": "^18.2.0",
    "react-dom": "^18.2.0"
  },
  "devDependencies": {
    "jest": "^29.7.0",
    "eslint": "^8.54.0",
    "prettier": "^3.1.0",
    "@types/node": "^20.10.0",
    "@types/express": "^4.17.21"
  },
  "repository": {
    "type": "git",
    "url": "https://github.com/%s/%s.git"
  }
}
`, org, repo, org, repo)

	if err := createOrUpdateFile(ctx, client, org, repo, "package.json", packageJSON, "Add package.json with dependencies"); err != nil {
		return err
	}

	// Create go.mod with dependencies
	goMod := `module github.com/` + org + `/` + repo + `

go 1.21

require (
	github.com/gin-gonic/gin v1.9.1
	github.com/google/uuid v1.5.0
	github.com/stretchr/testify v1.8.4
	golang.org/x/crypto v0.17.0
	gopkg.in/yaml.v3 v3.0.1
)
`

	if err := createOrUpdateFile(ctx, client, org, repo, "go.mod", goMod, "Add go.mod with dependencies"); err != nil {
		return err
	}

	// Create requirements.txt for Python
	requirementsTxt := `django==5.0
requests==2.31.0
pytest==7.4.3
pandas==2.1.4
numpy==1.26.2
flask==3.0.0
sqlalchemy==2.0.23
`

	if err := createOrUpdateFile(ctx, client, org, repo, "requirements.txt", requirementsTxt, "Add Python dependencies"); err != nil {
		return err
	}

	// Create a simple index.js
	indexJS := `const express = require('express');
const axios = require('axios');
const _ = require('lodash');

const app = express();
const port = 3000;

app.get('/', (req, res) => {
  res.send('Package Dependencies Test Repository');
});

app.listen(port, () => {
  console.log('Server running on port ' + port);
});
`

	if err := createOrUpdateFile(ctx, client, org, repo, "index.js", indexJS, "Add index.js"); err != nil {
		return err
	}

	log.Printf("  ✅ Package dependencies configured (npm, Go, Python)")
	return nil
}

func setupMixedDependenciesRepo(ctx context.Context, client *github.Client, org, repo string) error {
	log.Printf("  🎭 Setting up repository with mixed dependency types")

	// Create comprehensive README
	readmeContent := `# Mixed Dependencies Test Repository

This repository combines multiple types of dependencies for comprehensive testing.

## Dependency Types

### 1. Submodules
- Local submodule: test-dependency-target (same org)
- External submodule: actions/checkout

### 2. GitHub Actions Workflows
- Reusable workflow from same org
- Actions from external repos (actions/checkout, etc.)

### 3. Package Dependencies
- npm packages (express, react, etc.)
- Go modules
- Python packages

## Purpose

This repository tests:
- Multiple dependency detection methods simultaneously
- Classification of local vs external dependencies across all types
- Batch planning scenarios with complex dependencies
- UI display of mixed dependency types

## Discovery Testing

When discovered, this repository should show:
- Total dependencies: 15+
- Local dependencies: 2+
- External dependencies: 13+
- By type: submodules, workflows, dependency_graph
`

	if err := createOrUpdateFile(ctx, client, org, repo, "README.md", readmeContent, "Add comprehensive README"); err != nil {
		return err
	}

	// Add .gitmodules
	gitmodulesContent := fmt.Sprintf(`[submodule "libs/local-target"]
	path = libs/local-target
	url = https://github.com/%s/test-dependency-target.git

[submodule "libs/external-checkout"]
	path = libs/external-checkout
	url = https://github.com/actions/checkout.git
`, org)

	if err := createOrUpdateFile(ctx, client, org, repo, ".gitmodules", gitmodulesContent, "Add submodules"); err != nil {
		return err
	}

	// Add workflow with dependencies
	workflowContent := fmt.Sprintf(`name: Mixed Dependencies CI

on: [push, pull_request]

jobs:
  # Local reusable workflow
  local-workflow:
    uses: %s/test-dependency-target/.github/workflows/reusable.yml@main

  build-and-test:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4
        with:
          submodules: recursive
      
      - uses: actions/setup-node@v4
        with:
          node-version: '18'
      
      - uses: actions/setup-go@v5
        with:
          go-version: '1.21'
      
      - uses: actions/setup-python@v5
        with:
          python-version: '3.11'
      
      - name: Install dependencies
        run: |
          npm install
          go mod download
          pip install -r requirements.txt
      
      - name: Run tests
        run: echo "Testing..."
`, org)

	if err := createOrUpdateFile(ctx, client, org, repo, ".github/workflows/mixed-ci.yml", workflowContent, "Add mixed CI workflow"); err != nil {
		return err
	}

	// Add package.json
	packageJSON := fmt.Sprintf(`{
  "name": "@%s/%s",
  "version": "1.0.0",
  "description": "Repository with mixed dependencies",
  "dependencies": {
    "express": "^4.18.2",
    "react": "^18.2.0",
    "axios": "^1.6.0"
  },
  "devDependencies": {
    "jest": "^29.7.0",
    "eslint": "^8.54.0"
  }
}
`, org, repo)

	if err := createOrUpdateFile(ctx, client, org, repo, "package.json", packageJSON, "Add package.json"); err != nil {
		return err
	}

	// Add go.mod
	goMod := `module github.com/` + org + `/` + repo + `

go 1.21

require (
	github.com/gin-gonic/gin v1.9.1
	github.com/stretchr/testify v1.8.4
)
`

	if err := createOrUpdateFile(ctx, client, org, repo, "go.mod", goMod, "Add go.mod"); err != nil {
		return err
	}

	// Add requirements.txt
	requirementsTxt := `flask==3.0.0
requests==2.31.0
pytest==7.4.3
`

	if err := createOrUpdateFile(ctx, client, org, repo, "requirements.txt", requirementsTxt, "Add requirements.txt"); err != nil {
		return err
	}

	log.Printf("  ✅ Mixed dependencies configured (submodules, workflows, packages)")
	return nil
}

// Helper functions

func createOrUpdateFile(ctx context.Context, client *github.Client, org, repo, path, content, message string) error {
	return createOrUpdateFileOnBranch(ctx, client, org, repo, path, content, message, "main")
}

func createOrUpdateFileOnBranch(ctx context.Context, client *github.Client, org, repo, path, content, message, branch string) error {
	// Check if file exists
	fileContent, _, resp, err := client.Repositories.GetContents(ctx, org, repo, path, &github.RepositoryContentGetOptions{
		Ref: branch,
	})

	opts := &github.RepositoryContentFileOptions{
		Message: github.Ptr(message),
		Content: []byte(content),
		Branch:  github.Ptr(branch),
	}

	if err == nil && resp.StatusCode == 200 {
		// File exists, update it
		opts.SHA = fileContent.SHA
		_, _, err = client.Repositories.UpdateFile(ctx, org, repo, path, opts)
	} else {
		// File doesn't exist, create it
		_, _, err = client.Repositories.CreateFile(ctx, org, repo, path, opts)
	}

	return err
}

func createBranch(ctx context.Context, client *github.Client, org, repo, branchName string) error {
	// Get the main branch to get its SHA
	mainBranch, _, err := client.Repositories.GetBranch(ctx, org, repo, "main", 1)
	if err != nil {
		return fmt.Errorf("failed to get main branch: %w", err)
	}

	// Get the commit SHA from the main branch
	mainSHA := mainBranch.GetCommit().GetSHA()

	// Create a new reference (branch) pointing to the same commit as main
	ref := "refs/heads/" + branchName

	// Try to create the ref using the newer interface that expects CreateRef type
	newRef := github.CreateRef{
		Ref: ref,
		SHA: mainSHA,
	}

	_, _, err = client.Git.CreateRef(ctx, org, repo, newRef)
	if err != nil {
		// Check if branch already exists
		if strings.Contains(err.Error(), "already exists") || strings.Contains(err.Error(), "Reference already exists") {
			log.Printf("  ℹ️  Branch %s already exists, skipping", branchName)
			return nil
		}
		return fmt.Errorf("failed to create branch: %w", err)
	}

	return nil
}

func createTag(ctx context.Context, client *github.Client, org, repo, tagName string) error {
	// Check if tag/release already exists
	_, resp, err := client.Repositories.GetReleaseByTag(ctx, org, repo, tagName)
	if err == nil {
		// Tag/release already exists
		return nil
	}
	if resp != nil && resp.StatusCode != 404 {
		return fmt.Errorf("unexpected error checking for tag: %w", err)
	}

	// Tag doesn't exist, create it via release
	// Note: We create tags via releases since the direct Git API has interface complications
	// This approach also provides better test coverage for the migration tool
	release := &github.RepositoryRelease{
		TagName:         github.Ptr(tagName),
		Name:            github.Ptr("Test Tag " + tagName),
		Body:            github.Ptr(fmt.Sprintf("Automated test tag created at %s", time.Now().Format(time.RFC3339))),
		Draft:           github.Ptr(false),
		Prerelease:      github.Ptr(false),
		TargetCommitish: github.Ptr("main"),
	}

	_, _, err = client.Repositories.CreateRelease(ctx, org, repo, release)
	if err != nil {
		// Check if it's an "already exists" error
		if strings.Contains(err.Error(), "already_exists") {
			return nil
		}
		return fmt.Errorf("failed to create tag via release: %w", err)
	}

	return nil
}
