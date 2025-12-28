package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/microsoft/azure-devops-go-api/azuredevops/v7"
	"github.com/microsoft/azure-devops-go-api/azuredevops/v7/build"
	"github.com/microsoft/azure-devops-go-api/azuredevops/v7/core"
	"github.com/microsoft/azure-devops-go-api/azuredevops/v7/git"
	"github.com/microsoft/azure-devops-go-api/azuredevops/v7/policy"
	"github.com/microsoft/azure-devops-go-api/azuredevops/v7/webapi"
	"github.com/microsoft/azure-devops-go-api/azuredevops/v7/wiki"
	"github.com/microsoft/azure-devops-go-api/azuredevops/v7/workitemtracking"
)

// TestRepoConfig defines an ADO test repository configuration
type TestRepoConfig struct {
	Name        string
	Description string
	Setup       func(ctx context.Context, clients *ADOClients, project, repoID string) error
}

// ADOClients holds all Azure DevOps API clients
type ADOClients struct {
	Connection *azuredevops.Connection
	Core       core.Client
	Git        git.Client
	Build      build.Client
	Work       workitemtracking.Client
	Policy     policy.Client
	Wiki       wiki.Client
}

func main() {
	// Parse command-line flags
	orgURL := flag.String("org", "", "Azure DevOps organization URL (e.g., https://dev.azure.com/myorg) (required)")
	project := flag.String("project", "", "Azure DevOps project name (required)")
	pat := flag.String("pat", os.Getenv("AZURE_DEVOPS_PAT"), "Azure DevOps PAT (or set AZURE_DEVOPS_PAT env var)")
	cleanupOnly := flag.Bool("cleanup", false, "Only cleanup existing test repositories")
	flag.Parse()

	if *orgURL == "" {
		log.Fatal("Organization URL is required: -org <org-url>")
	}

	if *project == "" {
		log.Fatal("Project name is required: -project <project-name>")
	}

	if *pat == "" {
		log.Fatal("Azure DevOps PAT is required: -pat <token> or set AZURE_DEVOPS_PAT env var")
	}

	ctx := context.Background()

	// Create Azure DevOps connection
	connection := azuredevops.NewPatConnection(*orgURL, *pat)

	// Create clients
	clients, err := createClients(ctx, connection)
	if err != nil {
		log.Fatalf("Failed to create Azure DevOps clients: %v", err)
	}

	// Verify project access
	proj, err := clients.Core.GetProject(ctx, core.GetProjectArgs{
		ProjectId:           project,
		IncludeCapabilities: ptr(true),
	})
	if err != nil {
		log.Fatalf("Failed to access project %s: %v", *project, err)
	}

	log.Printf("Successfully connected to project: %s (ID: %s)", *proj.Name, proj.Id.String())

	// Cleanup existing test repos if requested
	if *cleanupOnly {
		cleanupTestRepos(ctx, clients, *project)
		return
	}

	// Create test repositories
	repos := getTestRepoConfigs()
	log.Printf("Creating %d test repositories in project %s...", len(repos), *project)

	// Pass the project object instead of just the name
	for i, config := range repos {
		log.Printf("[%d/%d] Creating repository: %s", i+1, len(repos), config.Name)
		if err := createTestRepo(ctx, clients, proj, config); err != nil {
			log.Printf("  ❌ Failed to create %s: %v", config.Name, err)
		} else {
			log.Printf("  ✅ Successfully created %s", config.Name)
		}
		// Rate limiting: sleep briefly between creations
		time.Sleep(2 * time.Second)
	}

	log.Println("\n🎉 Test repository creation complete!")
	log.Printf("Run discovery against organization: %s, project: %s", *orgURL, *project)
	log.Printf("\nTo cleanup test repos later, run: go run scripts/create-ado-test-repos.go -org %s -project %s -cleanup", *orgURL, *project)
}

// createClients creates all necessary Azure DevOps API clients
func createClients(ctx context.Context, connection *azuredevops.Connection) (*ADOClients, error) {
	coreClient, err := core.NewClient(ctx, connection)
	if err != nil {
		return nil, fmt.Errorf("failed to create core client: %w", err)
	}

	gitClient, err := git.NewClient(ctx, connection)
	if err != nil {
		return nil, fmt.Errorf("failed to create git client: %w", err)
	}

	buildClient, err := build.NewClient(ctx, connection)
	if err != nil {
		return nil, fmt.Errorf("failed to create build client: %w", err)
	}

	workClient, err := workitemtracking.NewClient(ctx, connection)
	if err != nil {
		return nil, fmt.Errorf("failed to create work item client: %w", err)
	}

	policyClient, err := policy.NewClient(ctx, connection)
	if err != nil {
		return nil, fmt.Errorf("failed to create policy client: %w", err)
	}

	wikiClient, err := wiki.NewClient(ctx, connection)
	if err != nil {
		return nil, fmt.Errorf("failed to create wiki client: %w", err)
	}

	return &ADOClients{
		Connection: connection,
		Core:       coreClient,
		Git:        gitClient,
		Build:      buildClient,
		Work:       workClient,
		Policy:     policyClient,
		Wiki:       wikiClient,
	}, nil
}

// getTestRepoConfigs returns all ADO test repository configurations
func getTestRepoConfigs() []TestRepoConfig {
	return []TestRepoConfig{
		{
			Name:        "test-ado-minimal-empty",
			Description: "Minimal empty Git repository",
			Setup:       nil, // No setup needed
		},
		{
			Name:        "test-ado-basic-repo",
			Description: "Basic repository with initial commit",
			Setup:       setupBasicRepo,
		},
		{
			Name:        "test-ado-with-yaml-pipeline",
			Description: "Repository with YAML pipeline",
			Setup:       setupYAMLPipelineRepo,
		},
		{
			Name:        "test-ado-with-classic-pipeline",
			Description: "Repository with Classic build pipeline",
			Setup:       setupClassicPipelineRepo,
		},
		{
			Name:        "test-ado-with-work-items",
			Description: "Repository with Azure Boards work items",
			Setup:       setupWorkItemsRepo,
		},
		{
			Name:        "test-ado-with-pull-requests",
			Description: "Repository with multiple pull requests",
			Setup:       setupPullRequestsRepo,
		},
		{
			Name:        "test-ado-with-branch-policies",
			Description: "Repository with branch protection policies",
			Setup:       setupBranchPoliciesRepo,
		},
		{
			Name:        "test-ado-with-wiki",
			Description: "Repository with project wiki",
			Setup:       setupWikiRepo,
		},
		{
			Name:        "test-ado-many-branches",
			Description: "Repository with multiple branches",
			Setup:       setupManyBranchesRepo,
		},
		{
			Name:        "test-ado-many-commits",
			Description: "Repository with many commits",
			Setup:       setupManyCommitsRepo,
		},
		{
			Name:        "test-ado-complex-all-features",
			Description: "Complex repository with multiple ADO features",
			Setup:       setupComplexRepo,
		},
		{
			Name:        "test-ado-shared-library",
			Description: "Shared library repository (depended on by other repos)",
			Setup:       setupSharedLibraryRepo,
		},
		{
			Name:        "test-ado-frontend-app",
			Description: "Frontend application with dependencies",
			Setup:       setupFrontendAppRepo,
		},
		{
			Name:        "test-ado-backend-api",
			Description: "Backend API with dependencies on shared library",
			Setup:       setupBackendAPIRepo,
		},
		{
			Name:        "test-ado-monorepo",
			Description: "Monorepo with multiple packages",
			Setup:       setupMonorepoRepo,
		},
	}
}

// createTestRepo creates a single test repository with configuration
func createTestRepo(ctx context.Context, clients *ADOClients, proj *core.TeamProject, config TestRepoConfig) error {
	projectID := proj.Id.String()
	projectName := *proj.Name

	// Check if repository already exists
	existingRepo, err := clients.Git.GetRepository(ctx, git.GetRepositoryArgs{
		RepositoryId: &config.Name,
		Project:      &projectName,
	})
	if err == nil && existingRepo != nil {
		log.Printf("  ⚠️  Repository %s already exists, skipping creation", config.Name)
		return nil
	}

	// Create new Git repository
	// Use the project ID in the TeamProjectReference to avoid issues with spaces in project names
	repoToCreate := git.GitRepositoryCreateOptions{
		Name: &config.Name,
		Project: &core.TeamProjectReference{
			Id:   proj.Id,
			Name: proj.Name,
		},
	}

	repo, err := clients.Git.CreateRepository(ctx, git.CreateRepositoryArgs{
		GitRepositoryToCreate: &repoToCreate,
		Project:               &projectID,
	})
	if err != nil {
		return fmt.Errorf("failed to create repository: %w", err)
	}

	repoIDStr := repo.Id.String()
	log.Printf("  📦 Repository created with ID: %s", repoIDStr)

	// Initialize repository with README
	if err := initializeRepo(ctx, clients, projectName, repoIDStr); err != nil {
		log.Printf("  ⚠️  Failed to initialize repo: %v", err)
		// If initialization fails, don't try setup
		return nil
	}

	// Wait for the main branch to be fully available
	if err := waitForBranch(ctx, clients, projectName, repoIDStr, "main", 15*time.Second); err != nil {
		log.Printf("  ⚠️  Branch 'main' not available after initialization: %v", err)
		// If branch isn't available, skip setup but don't fail the repo creation
		return nil
	}

	log.Printf("  ✅ Branch 'main' is ready")

	// Give Azure DevOps a moment to fully commit the branch state across all APIs
	// The branch may appear in refs API but not be ready for push operations yet
	// Empirically, 2 seconds is not enough - increasing to 5 seconds
	time.Sleep(5 * time.Second)

	// Run setup function if provided
	if config.Setup != nil {
		if err := config.Setup(ctx, clients, projectName, repoIDStr); err != nil {
			return fmt.Errorf("setup failed: %w", err)
		}
	}

	return nil
}

// initializeRepo adds initial README to repository
func initializeRepo(ctx context.Context, clients *ADOClients, project, repoID string) error {
	// Create initial commit with README
	branch := "main"
	refName := "refs/heads/" + branch

	readme := fmt.Sprintf("# Test Repository\n\nThis is a test repository created for Azure DevOps migration testing.\n\nCreated: %s\n", time.Now().Format(time.RFC3339))

	oldObjectID := "0000000000000000000000000000000000000000"

	changes := []any{
		git.GitChange{
			ChangeType: &git.VersionControlChangeTypeValues.Add,
			Item: &git.GitItem{
				Path: ptr("/README.md"),
			},
			NewContent: &git.ItemContent{
				Content:     &readme,
				ContentType: &git.ItemContentTypeValues.RawText,
			},
		},
	}

	push := git.GitPush{
		Commits: &[]git.GitCommitRef{
			{
				Comment: ptr("Initial commit"),
				Changes: &changes,
			},
		},
		RefUpdates: &[]git.GitRefUpdate{
			{
				Name:        &refName,
				OldObjectId: &oldObjectID,
			},
		},
	}

	result, err := clients.Git.CreatePush(ctx, git.CreatePushArgs{
		Push:         &push,
		RepositoryId: &repoID,
		Project:      &project,
	})

	if err != nil {
		return fmt.Errorf("failed to create push: %w", err)
	}

	if result == nil || result.RefUpdates == nil || len(*result.RefUpdates) == 0 {
		return fmt.Errorf("push succeeded but no ref updates returned")
	}

	// Log the commit that was created
	if result.Commits != nil && len(*result.Commits) > 0 {
		commitID := (*result.Commits)[0].CommitId
		log.Printf("  📝 Initial commit created: %s", *commitID)
	}

	return nil
}

// cleanupTestRepos deletes all test repositories
func cleanupTestRepos(ctx context.Context, clients *ADOClients, project string) {
	log.Printf("Fetching repositories from project: %s", project)

	repos, err := clients.Git.GetRepositories(ctx, git.GetRepositoriesArgs{
		Project: &project,
	})
	if err != nil {
		log.Fatalf("Failed to list repositories: %v", err)
	}

	log.Printf("Found %d total repositories", len(*repos))

	var testRepos []git.GitRepository
	for _, repo := range *repos {
		if repo.Name != nil && len(*repo.Name) >= 5 && (*repo.Name)[:5] == "test-" {
			testRepos = append(testRepos, repo)
		}
	}

	if len(testRepos) == 0 {
		log.Println("No test repositories found to cleanup")
		return
	}

	log.Printf("Found %d test repositories to delete", len(testRepos))

	for i, repo := range testRepos {
		log.Printf("[%d/%d] Deleting repository: %s", i+1, len(testRepos), *repo.Name)

		err := clients.Git.DeleteRepository(ctx, git.DeleteRepositoryArgs{
			RepositoryId: repo.Id,
			Project:      &project,
		})

		if err != nil {
			log.Printf("  ❌ Failed to delete %s: %v", *repo.Name, err)
		} else {
			log.Printf("  ✅ Successfully deleted %s", *repo.Name)
		}

		time.Sleep(1 * time.Second)
	}

	log.Println("\n🎉 Cleanup complete!")
}

// Setup functions for different repository configurations

func setupBasicRepo(ctx context.Context, clients *ADOClients, project, repoID string) error {
	// Already initialized with README, add a few more files

	files := map[string]string{
		"/src/main.go": "package main\n\nfunc main() {\n\tprintln(\"Hello, ADO!\")\n}\n",
		"/.gitignore":  "*.exe\n*.dll\n*.so\n*.dylib\n",
		"/docs/API.md": "# API Documentation\n\nThis is the API documentation.\n",
	}

	return addFilesToRepo(ctx, clients, project, repoID, "main", files, "Add basic project structure")
}

func setupYAMLPipelineRepo(ctx context.Context, clients *ADOClients, project, repoID string) error {
	// Add azure-pipelines.yml file
	pipelineYAML := `trigger:
  - main

pool:
  vmImage: 'ubuntu-latest'

steps:
- script: echo Hello, YAML Pipeline!
  displayName: 'Run a one-line script'

- script: |
    echo Add other tasks to build, test, and deploy your project.
    echo See https://aka.ms/yaml
  displayName: 'Run a multi-line script'
`

	files := map[string]string{
		"/azure-pipelines.yml": pipelineYAML,
		"/src/app.js":          "console.log('Hello from Node.js');\n",
	}

	if err := addFilesToRepo(ctx, clients, project, repoID, "main", files, "Add YAML pipeline configuration"); err != nil {
		return err
	}

	// Note: Creating an actual pipeline definition requires the pipeline to be manually created
	// or using the Pipelines REST API which is more complex
	log.Printf("  ℹ️  YAML pipeline file added. Create pipeline manually in Azure DevOps UI to test detection.")

	return nil
}

func setupClassicPipelineRepo(ctx context.Context, clients *ADOClients, project, repoID string) error {
	// Classic pipelines can't be easily created via API in a generic way
	// They require detailed build definition configuration

	files := map[string]string{
		"/buildspec.md": "# Build Specification\n\nThis repository is configured for Classic Build Pipelines.\nCreate a Classic Pipeline manually in Azure DevOps UI.\n",
		"/src/app.cs":   "using System;\n\nclass Program\n{\n    static void Main()\n    {\n        Console.WriteLine(\"Hello from C#\");\n    }\n}\n",
	}

	if err := addFilesToRepo(ctx, clients, project, repoID, "main", files, "Add project for Classic pipeline"); err != nil {
		return err
	}

	log.Printf("  ℹ️  Repository ready for Classic Pipeline. Create pipeline manually in Azure DevOps UI.")

	return nil
}

func setupWorkItemsRepo(ctx context.Context, clients *ADOClients, project, repoID string) error {
	// Create several work items linked to this repository
	// Only use "Task" type which exists in all process templates (Basic, Agile, Scrum, CMMI)

	workItemTypes := []string{"Task", "Task", "Task"}
	workItemLabels := []string{"Feature Request", "Technical Debt", "Testing"}

	for i, wiType := range workItemTypes {
		title := fmt.Sprintf("Test %s %d - %s", wiType, i+1, workItemLabels[i])
		description := fmt.Sprintf("This is a test %s for migration testing", wiType)

		addOp := webapi.Operation("add")
		doc := []webapi.JsonPatchOperation{
			{
				Op:    &addOp,
				Path:  ptr("/fields/System.Title"),
				Value: title,
			},
			{
				Op:    &addOp,
				Path:  ptr("/fields/System.Description"),
				Value: description,
			},
		}

		_, err := clients.Work.CreateWorkItem(ctx, workitemtracking.CreateWorkItemArgs{
			Document: &doc,
			Project:  &project,
			Type:     &wiType,
		})

		if err != nil {
			log.Printf("  ⚠️  Failed to create work item: %v", err)
		} else {
			log.Printf("  ✅ Created work item: %s", title)
		}

		time.Sleep(500 * time.Millisecond)
	}

	return nil
}

func setupPullRequestsRepo(ctx context.Context, clients *ADOClients, project, repoID string) error {
	// Create a feature branch
	if err := createBranch(ctx, clients, project, repoID, "feature/test-pr", "main"); err != nil {
		return err
	}

	// Add some changes to the feature branch
	files := map[string]string{
		"/feature.txt": "This is a new feature\n",
	}

	if err := addFilesToRepo(ctx, clients, project, repoID, "feature/test-pr", files, "Add feature implementation"); err != nil {
		return err
	}

	// Create pull request
	sourceBranch := "refs/heads/feature/test-pr"
	targetBranch := "refs/heads/main"
	title := "Test Pull Request"
	description := "This is a test pull request for migration testing"

	pr := git.GitPullRequest{
		SourceRefName: &sourceBranch,
		TargetRefName: &targetBranch,
		Title:         &title,
		Description:   &description,
	}

	createdPR, err := clients.Git.CreatePullRequest(ctx, git.CreatePullRequestArgs{
		GitPullRequestToCreate: &pr,
		RepositoryId:           &repoID,
		Project:                &project,
	})

	if err != nil {
		return fmt.Errorf("failed to create pull request: %w", err)
	}

	log.Printf("  ✅ Created pull request #%d", *createdPR.PullRequestId)

	return nil
}

func setupBranchPoliciesRepo(ctx context.Context, clients *ADOClients, project, repoID string) error {
	// Create a develop branch
	if err := createBranch(ctx, clients, project, repoID, "develop", "main"); err != nil {
		return err
	}

	// Add branch policy for main branch
	// Note: This requires detailed policy configuration and may need admin permissions

	log.Printf("  ℹ️  Repository ready for branch policies. Add policies manually in Azure DevOps UI:")
	log.Printf("      - Require minimum number of reviewers")
	log.Printf("      - Check for linked work items")
	log.Printf("      - Build validation")

	return nil
}

func setupWikiRepo(ctx context.Context, clients *ADOClients, project, repoID string) error {
	// Note: Wiki creation is project-level, not repository-level
	// Wikis are often already created by default in ADO projects
	// This is just a placeholder - manual wiki content addition would be needed
	log.Printf("  ℹ️  Wiki setup skipped - wikis are project-level in Azure DevOps")
	log.Printf("      You can manually add wiki content via the Azure DevOps UI")
	return nil
}

func setupManyBranchesRepo(ctx context.Context, clients *ADOClients, project, repoID string) error {
	branches := []string{"feature/branch-1", "feature/branch-2", "feature/branch-3", "release/v1.0", "hotfix/critical-bug"}

	for _, branch := range branches {
		if err := createBranch(ctx, clients, project, repoID, branch, "main"); err != nil {
			log.Printf("  ⚠️  Failed to create branch %s: %v", branch, err)
			continue
		}

		// Add a unique file to each branch
		files := map[string]string{
			fmt.Sprintf("/%s.txt", branch): fmt.Sprintf("Content for %s\n", branch),
		}

		if err := addFilesToRepo(ctx, clients, project, repoID, branch, files, fmt.Sprintf("Add content to %s", branch)); err != nil {
			log.Printf("  ⚠️  Failed to add files to branch %s: %v", branch, err)
		}

		time.Sleep(500 * time.Millisecond)
	}

	return nil
}

func setupManyCommitsRepo(ctx context.Context, clients *ADOClients, project, repoID string) error {
	// Create multiple commits
	for i := 1; i <= 10; i++ {
		files := map[string]string{
			fmt.Sprintf("/commit-%d.txt", i): fmt.Sprintf("This is commit number %d\n", i),
		}

		if err := addFilesToRepo(ctx, clients, project, repoID, "main", files, fmt.Sprintf("Commit %d of 10", i)); err != nil {
			log.Printf("  ⚠️  Failed to create commit %d: %v", i, err)
		}

		time.Sleep(300 * time.Millisecond)
	}

	return nil
}

func setupComplexRepo(ctx context.Context, clients *ADOClients, project, repoID string) error {
	// Combine multiple features
	log.Printf("  🔧 Setting up complex repository with multiple features...")

	// Add YAML pipeline
	pipelineYAML := `trigger:
  - main
  - develop

pool:
  vmImage: 'ubuntu-latest'

stages:
- stage: Build
  jobs:
  - job: BuildJob
    steps:
    - script: echo Building...
      
- stage: Test
  jobs:
  - job: TestJob
    steps:
    - script: echo Testing...
      
- stage: Deploy
  jobs:
  - job: DeployJob
    steps:
    - script: echo Deploying...
`

	files := map[string]string{
		"/azure-pipelines.yml": pipelineYAML,
		"/src/app.go":          "package main\n\nfunc main() {\n\tprintln(\"Complex app\")\n}\n",
		"/docs/README.md":      "# Complex Repository\n\nThis repository has multiple ADO features.\n",
		"/.gitignore":          "*.exe\n*.dll\nbin/\nobj/\n",
		"/tests/test.go":       "package main\n\nimport \"testing\"\n\nfunc TestMain(t *testing.T) {\n\tt.Log(\"Test\")\n}\n",
	}

	if err := addFilesToRepo(ctx, clients, project, repoID, "main", files, "Initial complex repository setup"); err != nil {
		return err
	}

	// Create branches
	branches := []string{"develop", "feature/complex-feature", "release/v1.0"}
	for _, branch := range branches {
		if err := createBranch(ctx, clients, project, repoID, branch, "main"); err != nil {
			log.Printf("  ⚠️  Failed to create branch %s: %v", branch, err)
		}
	}

	// Create pull request
	if err := createBranch(ctx, clients, project, repoID, "feature/pr-test", "main"); err == nil {
		sourceBranch := "refs/heads/feature/pr-test"
		targetBranch := "refs/heads/main"
		title := "Complex Feature PR"
		description := "Testing complex repository features"

		pr := git.GitPullRequest{
			SourceRefName: &sourceBranch,
			TargetRefName: &targetBranch,
			Title:         &title,
			Description:   &description,
		}

		_, err := clients.Git.CreatePullRequest(ctx, git.CreatePullRequestArgs{
			GitPullRequestToCreate: &pr,
			RepositoryId:           &repoID,
			Project:                &project,
		})

		if err != nil {
			log.Printf("  ⚠️  Failed to create PR: %v", err)
		}
	}

	// Create work items (only use Task which exists in all process templates)
	workItemLabels := []string{"Backend", "Frontend", "Database", "DevOps"}
	for i, label := range workItemLabels {
		title := fmt.Sprintf("Complex Repo Task %d - %s", i+1, label)

		addOp := webapi.Operation("add")
		doc := []webapi.JsonPatchOperation{
			{
				Op:    &addOp,
				Path:  ptr("/fields/System.Title"),
				Value: title,
			},
			{
				Op:    &addOp,
				Path:  ptr("/fields/System.State"),
				Value: "To Do",
			},
		}

		_, err := clients.Work.CreateWorkItem(ctx, workitemtracking.CreateWorkItemArgs{
			Document: &doc,
			Project:  &project,
			Type:     ptr("Task"),
		})

		if err != nil {
			log.Printf("  ⚠️  Failed to create work item: %v", err)
		}

		time.Sleep(300 * time.Millisecond)
	}

	log.Printf("  ✅ Complex repository setup complete")

	return nil
}

// setupSharedLibraryRepo creates a shared library that other repos depend on
func setupSharedLibraryRepo(ctx context.Context, clients *ADOClients, project, repoID string) error {
	// Create a shared library with package.json (npm) and pom.xml (Maven)
	packageJSON := `{
  "name": "@test-org/shared-library",
  "version": "1.2.3",
  "description": "Shared utilities library",
  "main": "index.js",
  "scripts": {
    "test": "jest"
  },
  "dependencies": {
    "lodash": "^4.17.21",
    "axios": "^1.6.0"
  },
  "devDependencies": {
    "jest": "^29.0.0"
  }
}`

	pomXML := `<?xml version="1.0" encoding="UTF-8"?>
<project xmlns="http://maven.apache.org/POM/4.0.0">
  <modelVersion>4.0.0</modelVersion>
  <groupId>com.testorg</groupId>
  <artifactId>shared-library</artifactId>
  <version>1.2.3</version>
  <packaging>jar</packaging>
  
  <dependencies>
    <dependency>
      <groupId>org.apache.commons</groupId>
      <artifactId>commons-lang3</artifactId>
      <version>3.12.0</version>
    </dependency>
  </dependencies>
</project>`

	nuspec := `<?xml version="1.0" encoding="utf-8"?>
<package xmlns="http://schemas.microsoft.com/packaging/2010/07/nuspec.xsd">
  <metadata>
    <id>TestOrg.SharedLibrary</id>
    <version>1.2.3</version>
    <authors>TestOrg</authors>
    <description>Shared library for testing</description>
    <dependencies>
      <dependency id="Newtonsoft.Json" version="13.0.3" />
    </dependencies>
  </metadata>
</package>`

	indexJS := `// Shared Library
module.exports = {
  formatDate: (date) => date.toISOString(),
  validateEmail: (email) => /^[^\s@]+@[^\s@]+\.[^\s@]+$/.test(email)
};`

	files := map[string]string{
		"/package.json":                       packageJSON,
		"/pom.xml":                            pomXML,
		"/SharedLibrary.nuspec":               nuspec,
		"/index.js":                           indexJS,
		"/src/main/java/Utils.java":           "public class Utils { /* shared code */ }",
		"/src/SharedLibrary/SharedLibrary.cs": "namespace SharedLibrary { public class Utils { } }",
		"/README.md":                          "# Shared Library\n\nThis library is used by other test repositories.",
	}

	return addFilesToRepo(ctx, clients, project, repoID, "main", files, "Add shared library code")
}

// setupFrontendAppRepo creates a frontend app with dependencies
func setupFrontendAppRepo(ctx context.Context, clients *ADOClients, project, repoID string) error {
	// Frontend app that depends on shared-library and other packages
	packageJSON := `{
  "name": "@test-org/frontend-app",
  "version": "2.0.0",
  "description": "Frontend application",
  "scripts": {
    "start": "react-scripts start",
    "build": "react-scripts build"
  },
  "dependencies": {
    "@test-org/shared-library": "^1.2.0",
    "react": "^18.2.0",
    "react-dom": "^18.2.0",
    "axios": "^1.6.0",
    "react-router-dom": "^6.20.0"
  },
  "devDependencies": {
    "react-scripts": "5.0.1",
    "@testing-library/react": "^14.0.0"
  }
}`

	yarnLock := `# THIS IS AN AUTOGENERATED FILE. DO NOT EDIT THIS FILE DIRECTLY.
# yarn lockfile v1

"@test-org/shared-library@^1.2.0":
  version "1.2.3"
  resolved "https://registry.yarnpkg.com/@test-org/shared-library/-/shared-library-1.2.3.tgz"

axios@^1.6.0:
  version "1.6.2"
  resolved "https://registry.yarnpkg.com/axios/-/axios-1.6.2.tgz"

react@^18.2.0:
  version "18.2.0"
  resolved "https://registry.yarnpkg.com/react/-/react-18.2.0.tgz"
`

	appJS := `import React from 'react';
import { formatDate } from '@test-org/shared-library';

function App() {
  return (
    <div>
      <h1>Frontend App</h1>
      <p>Today: {formatDate(new Date())}</p>
    </div>
  );
}

export default App;`

	files := map[string]string{
		"/package.json": packageJSON,
		"/yarn.lock":    yarnLock,
		"/src/App.js":   appJS,
		"/README.md":    "# Frontend App\n\nDependencies:\n- @test-org/shared-library\n- react\n- axios",
	}

	return addFilesToRepo(ctx, clients, project, repoID, "main", files, "Add frontend application code")
}

// setupBackendAPIRepo creates a backend API with dependencies
func setupBackendAPIRepo(ctx context.Context, clients *ADOClients, project, repoID string) error {
	// Backend API that depends on shared-library
	packageJSON := `{
  "name": "@test-org/backend-api",
  "version": "3.1.0",
  "description": "Backend API service",
  "main": "server.js",
  "scripts": {
    "start": "node server.js",
    "dev": "nodemon server.js"
  },
  "dependencies": {
    "@test-org/shared-library": "^1.2.0",
    "express": "^4.18.2",
    "axios": "^1.6.0",
    "mongoose": "^8.0.0",
    "jsonwebtoken": "^9.0.2"
  },
  "devDependencies": {
    "nodemon": "^3.0.0",
    "jest": "^29.0.0"
  }
}`

	pomXML := `<?xml version="1.0" encoding="UTF-8"?>
<project xmlns="http://maven.apache.org/POM/4.0.0">
  <modelVersion>4.0.0</modelVersion>
  <groupId>com.testorg</groupId>
  <artifactId>backend-api</artifactId>
  <version>3.1.0</version>
  
  <dependencies>
    <dependency>
      <groupId>com.testorg</groupId>
      <artifactId>shared-library</artifactId>
      <version>1.2.3</version>
    </dependency>
    <dependency>
      <groupId>org.springframework.boot</groupId>
      <artifactId>spring-boot-starter-web</artifactId>
      <version>3.2.0</version>
    </dependency>
  </dependencies>
</project>`

	requirementsTxt := `# Python dependencies
flask==3.0.0
requests==2.31.0
sqlalchemy==2.0.23
pytest==7.4.3`

	goMod := `module github.com/test-org/backend-api

go 1.21

require (
	github.com/gin-gonic/gin v1.9.1
	github.com/lib/pq v1.10.9
	github.com/stretchr/testify v1.8.4
)
`

	serverJS := `const express = require('express');
const { validateEmail } = require('@test-org/shared-library');

const app = express();

app.get('/api/validate', (req, res) => {
  const isValid = validateEmail(req.query.email);
  res.json({ valid: isValid });
});

app.listen(3000);`

	files := map[string]string{
		"/package.json":       packageJSON,
		"/pom.xml":            pomXML,
		"/requirements.txt":   requirementsTxt,
		"/go.mod":             goMod,
		"/server.js":          serverJS,
		"/src/main/Main.java": "public class Main { /* API code */ }",
		"/app.py":             "from flask import Flask\napp = Flask(__name__)",
		"/main.go":            "package main\n\nfunc main() { }",
		"/README.md":          "# Backend API\n\nDependencies:\n- @test-org/shared-library\n- express\n- Spring Boot\n- Flask",
	}

	return addFilesToRepo(ctx, clients, project, repoID, "main", files, "Add backend API code")
}

// setupMonorepoRepo creates a monorepo with multiple packages
func setupMonorepoRepo(ctx context.Context, clients *ADOClients, project, repoID string) error {
	// Monorepo with internal dependencies
	rootPackageJSON := `{
  "name": "@test-org/monorepo",
  "version": "1.0.0",
  "private": true,
  "workspaces": [
    "packages/*"
  ],
  "scripts": {
    "build": "lerna run build",
    "test": "lerna run test"
  },
  "devDependencies": {
    "lerna": "^8.0.0",
    "typescript": "^5.3.0"
  }
}`

	packageAJSON := `{
  "name": "@test-org/package-a",
  "version": "1.0.0",
  "dependencies": {
    "lodash": "^4.17.21"
  }
}`

	packageBJSON := `{
  "name": "@test-org/package-b",
  "version": "1.0.0",
  "dependencies": {
    "@test-org/package-a": "1.0.0",
    "axios": "^1.6.0"
  }
}`

	packageCJSON := `{
  "name": "@test-org/package-c",
  "version": "1.0.0",
  "dependencies": {
    "@test-org/package-a": "1.0.0",
    "@test-org/package-b": "1.0.0",
    "@test-org/shared-library": "^1.2.0"
  }
}`

	lernaJSON := `{
  "version": "1.0.0",
  "npmClient": "yarn",
  "useWorkspaces": true,
  "packages": [
    "packages/*"
  ]
}`

	files := map[string]string{
		"/package.json":                    rootPackageJSON,
		"/lerna.json":                      lernaJSON,
		"/packages/package-a/package.json": packageAJSON,
		"/packages/package-a/index.js":     "module.exports = { utilA: () => {} };",
		"/packages/package-b/package.json": packageBJSON,
		"/packages/package-b/index.js":     "const { utilA } = require('@test-org/package-a');\nmodule.exports = { utilB: () => {} };",
		"/packages/package-c/package.json": packageCJSON,
		"/packages/package-c/index.js":     "const { utilA } = require('@test-org/package-a');\nconst { utilB } = require('@test-org/package-b');",
		"/README.md":                       "# Monorepo\n\nInternal dependencies:\n- package-c depends on package-a and package-b\n- package-b depends on package-a\n- package-c depends on shared-library",
	}

	return addFilesToRepo(ctx, clients, project, repoID, "main", files, "Add monorepo structure with internal dependencies")
}

// Helper functions

// waitForBranch polls the Azure DevOps API until the branch is available or timeout is reached
func waitForBranch(ctx context.Context, clients *ADOClients, project, repoID, branchName string, timeout time.Duration) error {
	refName := "refs/heads/" + branchName
	deadline := time.Now().Add(timeout)
	retryInterval := 1 * time.Second
	attempt := 0

	for time.Now().Before(deadline) {
		attempt++

		// Try to get all refs first to see what's available
		allRefs, err := clients.Git.GetRefs(ctx, git.GetRefsArgs{
			RepositoryId: &repoID,
			Project:      &project,
		})

		if err != nil {
			log.Printf("  🔍 Attempt %d: Error getting refs: %v", attempt, err)
		} else if allRefs != nil && len(allRefs.Value) > 0 {
			// Check if our specific branch is in the list
			for _, ref := range allRefs.Value {
				if ref.Name != nil && *ref.Name == refName {
					log.Printf("  🔍 Found branch '%s' after %d attempts", branchName, attempt)
					return nil
				}
			}
			log.Printf("  🔍 Attempt %d: Found %d refs, but '%s' not among them yet", attempt, len(allRefs.Value), refName)
		} else {
			log.Printf("  🔍 Attempt %d: No refs found yet", attempt)
		}

		// Wait before retrying
		time.Sleep(retryInterval)
	}

	return fmt.Errorf("timeout waiting for branch %s to be available after %d attempts", branchName, attempt)
}

func createBranch(ctx context.Context, clients *ADOClients, project, repoID, branchName, sourceBranch string) error {
	// Get the latest commit from source branch
	// Use same approach as addFilesToRepo - get all refs and search
	sourceRef := "refs/heads/" + sourceBranch

	allRefs, err := clients.Git.GetRefs(ctx, git.GetRefsArgs{
		RepositoryId: &repoID,
		Project:      &project,
	})

	if err != nil {
		return fmt.Errorf("failed to get refs: %w", err)
	}

	// Search for source branch
	var sourceObjectID string
	found := false
	if allRefs != nil {
		for _, ref := range allRefs.Value {
			if ref.Name != nil && *ref.Name == sourceRef {
				sourceObjectID = *ref.ObjectId
				found = true
				break
			}
		}
	}

	if !found {
		// List available refs for debugging
		availableRefs := []string{}
		if allRefs != nil {
			for _, ref := range allRefs.Value {
				if ref.Name != nil {
					availableRefs = append(availableRefs, *ref.Name)
				}
			}
		}
		return fmt.Errorf("source branch %s not found (available refs: %v)", sourceBranch, availableRefs)
	}

	// Create new branch
	newRef := "refs/heads/" + branchName
	refUpdate := git.GitRefUpdate{
		Name:        &newRef,
		OldObjectId: ptr("0000000000000000000000000000000000000000"),
		NewObjectId: &sourceObjectID,
	}

	_, err = clients.Git.UpdateRefs(ctx, git.UpdateRefsArgs{
		RefUpdates:   &[]git.GitRefUpdate{refUpdate},
		RepositoryId: &repoID,
		Project:      &project,
	})

	if err != nil {
		return fmt.Errorf("failed to create branch: %w", err)
	}

	log.Printf("  ✅ Created branch: %s", branchName)
	return nil
}

func addFilesToRepo(ctx context.Context, clients *ADOClients, project, repoID, branch string, files map[string]string, commitMessage string) error {
	// Get current commit SHA for the branch
	refName := "refs/heads/" + branch

	// Azure DevOps API quirk: Filter parameter doesn't work immediately after branch creation
	// So we get ALL refs and search for the one we need
	var targetRef *git.GitRef
	var err error
	maxRetries := 3

	for attempt := 1; attempt <= maxRetries; attempt++ {
		allRefs, err := clients.Git.GetRefs(ctx, git.GetRefsArgs{
			RepositoryId: &repoID,
			Project:      &project,
		})

		if err == nil && allRefs != nil {
			// Search for our specific branch
			for _, ref := range allRefs.Value {
				if ref.Name != nil && *ref.Name == refName {
					targetRef = &ref
					break
				}
			}

			if targetRef != nil {
				break // Success!
			}
		}

		if attempt < maxRetries {
			log.Printf("      ⏱️  Attempt %d: Branch %s not found in refs, waiting 2 seconds...", attempt, branch)
			time.Sleep(2 * time.Second)
		}
	}

	if err != nil {
		return fmt.Errorf("failed to get refs after %d attempts: %w", maxRetries, err)
	}

	if targetRef == nil {
		// Get ALL refs to see what's actually available
		allRefs, _ := clients.Git.GetRefs(ctx, git.GetRefsArgs{
			RepositoryId: &repoID,
			Project:      &project,
		})
		availableRefs := []string{}
		if allRefs != nil {
			for _, ref := range allRefs.Value {
				if ref.Name != nil {
					availableRefs = append(availableRefs, *ref.Name)
				}
			}
		}
		return fmt.Errorf("branch %s not found in refs (available refs: %v)", refName, availableRefs)
	}

	oldObjectID := *targetRef.ObjectId

	// Get existing files to determine if we should Add or Edit
	existingFiles := make(map[string]bool)
	items, err := clients.Git.GetItems(ctx, git.GetItemsArgs{
		RepositoryId: &repoID,
		Project:      &project,
		VersionDescriptor: &git.GitVersionDescriptor{
			Version:     &branch,
			VersionType: &git.GitVersionTypeValues.Branch,
		},
		RecursionLevel: &git.VersionControlRecursionTypeValues.Full,
	})

	if err == nil && items != nil {
		for _, item := range *items {
			if item.Path != nil {
				existingFiles[*item.Path] = true
			}
		}
	}

	// Build changes array as interface{}
	// Use Edit for existing files (like README.md), Add for new files
	var changes []any
	for path, content := range files {
		contentCopy := content // Create a copy for the pointer
		pathCopy := path       // Create a copy for the pointer

		// Determine change type based on whether file exists
		changeType := &git.VersionControlChangeTypeValues.Add
		if existingFiles[path] {
			changeType = &git.VersionControlChangeTypeValues.Edit
		}

		changes = append(changes, git.GitChange{
			ChangeType: changeType,
			Item: &git.GitItem{
				Path: &pathCopy,
			},
			NewContent: &git.ItemContent{
				Content:     &contentCopy,
				ContentType: &git.ItemContentTypeValues.RawText,
			},
		})
	}

	// Create push
	push := git.GitPush{
		Commits: &[]git.GitCommitRef{
			{
				Comment: &commitMessage,
				Changes: &changes,
			},
		},
		RefUpdates: &[]git.GitRefUpdate{
			{
				Name:        &refName,
				OldObjectId: &oldObjectID,
			},
		},
	}

	_, err = clients.Git.CreatePush(ctx, git.CreatePushArgs{
		Push:         &push,
		RepositoryId: &repoID,
		Project:      &project,
	})

	if err != nil {
		return fmt.Errorf("failed to push changes: %w", err)
	}

	return nil
}

// ptr is a helper function to get pointer to a value
func ptr[T any](v T) *T {
	return &v
}
