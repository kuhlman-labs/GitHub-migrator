package handlers

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/kuhlman-labs/github-migrator/internal/azuredevops"
	"github.com/kuhlman-labs/github-migrator/internal/discovery"
	"github.com/kuhlman-labs/github-migrator/internal/models"
	"github.com/kuhlman-labs/github-migrator/internal/source"
)

// ADOHandler contains Azure DevOps-specific HTTP handlers
type ADOHandler struct {
	Handler
	adoClient    *azuredevops.Client
	adoProvider  source.Provider
	adoCollector *discovery.ADOCollector // Specialized collector for ADO
}

// NewADOHandler creates a new ADOHandler instance
func NewADOHandler(baseHandler *Handler, adoClient *azuredevops.Client, adoProvider source.Provider) *ADOHandler {
	adoCollector := discovery.NewADOCollector(adoClient, baseHandler.db, baseHandler.logger, adoProvider)
	return &ADOHandler{
		Handler:      *baseHandler,
		adoClient:    adoClient,
		adoProvider:  adoProvider,
		adoCollector: adoCollector,
	}
}

// StartADODiscovery handles POST /api/v1/ado/discover
// Discovers ADO organizations or specific projects
func (h *ADOHandler) StartADODiscovery(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Organization string   `json:"organization"`       // Required: ADO organization name
		Projects     []string `json:"projects,omitempty"` // Optional: specific projects to discover
		Workers      int      `json:"workers,omitempty"`  // Optional: number of parallel workers
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		WriteError(w, ErrInvalidJSON)
		return
	}

	if req.Organization == "" {
		WriteError(w, ErrMissingField.WithField("organization"))
		return
	}

	if req.Workers <= 0 {
		req.Workers = 5 // default
	}

	// Start discovery asynchronously based on scope
	if len(req.Projects) == 0 {
		// Discover entire organization
		h.logger.Info("Starting ADO organization discovery",
			"organization", req.Organization,
			"workers", req.Workers)

		// Only start discovery if collector is configured (allows for testing)
		if h.adoCollector != nil && h.adoClient != nil {
			go func() {
				ctx := context.Background()
				if err := h.adoCollector.DiscoverADOOrganization(ctx, req.Organization); err != nil {
					h.logger.Error("ADO organization discovery failed",
						"organization", req.Organization,
						"error", err)
				} else {
					h.logger.Info("ADO organization discovery completed", "organization", req.Organization)
				}
			}()
		}

		h.sendJSON(w, http.StatusAccepted, map[string]interface{}{
			"message":      "ADO organization discovery started",
			"organization": req.Organization,
			"type":         "organization",
		})
	} else {
		// Discover specific projects
		h.logger.Info("Starting ADO project discovery",
			"organization", req.Organization,
			"projects", req.Projects,
			"workers", req.Workers)

		// Only start discovery if collector is configured (allows for testing)
		if h.adoCollector != nil && h.adoClient != nil {
			go func() {
				ctx := context.Background()
				for _, project := range req.Projects {
					if err := h.adoCollector.DiscoverADOProject(ctx, req.Organization, project); err != nil {
						h.logger.Error("Failed to discover ADO project",
							"organization", req.Organization,
							"project", project,
							"error", err)
						// Continue with other projects
					} else {
						h.logger.Info("ADO project discovery completed",
							"organization", req.Organization,
							"project", project)
					}
				}
			}()
		}

		h.sendJSON(w, http.StatusAccepted, map[string]interface{}{
			"message":      "ADO project discovery started",
			"organization": req.Organization,
			"projects":     req.Projects,
			"type":         "project",
		})
	}
}

// ListADOProjects handles GET /api/v1/ado/projects
// Lists all discovered ADO projects with repository counts
func (h *ADOHandler) ListADOProjects(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	// Get optional organization filter
	organization := r.URL.Query().Get("organization")

	// Query projects from database
	projects, err := h.db.GetADOProjects(ctx, organization)
	if err != nil {
		WriteError(w, ErrDatabaseFetch.WithDetails("ADO projects"))
		return
	}

	// Enrich projects with repository counts and status breakdowns
	type ProjectWithCount struct {
		*models.ADOProject
		RepositoryCount int            `json:"repository_count"`
		StatusCounts    map[string]int `json:"status_counts"`
	}

	enrichedProjects := make([]ProjectWithCount, 0, len(projects))
	for _, project := range projects {
		// Count repositories for this project
		count, err := h.db.CountRepositoriesByADOProject(ctx, project.Organization, project.Name)
		if err != nil {
			h.logger.Error("Failed to count repositories for project",
				"organization", project.Organization,
				"project", project.Name,
				"error", err)
			count = 0 // Continue with 0 count
		}

		// Get status distribution for repos in this project
		statusCounts := make(map[string]int)
		if count > 0 {
			// Query actual status distribution using SQL for efficiency
			var results []struct {
				Status string
				Count  int
			}
			err := h.db.DB().WithContext(ctx).
				Raw(`
					SELECT status, COUNT(*) as count
					FROM repositories
					WHERE ado_project = ?
					AND status != 'wont_migrate'
					GROUP BY status
				`, project.Name).
				Scan(&results).Error

			if err != nil {
				h.logger.Warn("Failed to get status counts for project", "project", project.Name, "error", err)
				// Fallback: assume all pending
				statusCounts["pending"] = count
			} else {
				for _, result := range results {
					statusCounts[result.Status] = result.Count
				}
			}
		}

		enrichedProjects = append(enrichedProjects, ProjectWithCount{
			ADOProject:      &project,
			RepositoryCount: count,
			StatusCounts:    statusCounts,
		})
	}

	h.sendJSON(w, http.StatusOK, map[string]interface{}{
		"projects": enrichedProjects,
		"total":    len(enrichedProjects),
	})
}

// GetADOProject handles GET /api/v1/ado/projects/{organization}/{project}
// Gets a specific ADO project with its repositories
func (h *ADOHandler) GetADOProject(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	// Extract organization and project from URL path
	// This should be set up in the router
	organization := r.PathValue("organization")
	projectName := r.PathValue("project")

	if organization == "" || projectName == "" {
		WriteError(w, ErrMissingField.WithDetails("organization and project are required"))
		return
	}

	// Get project from database
	project, err := h.db.GetADOProject(ctx, organization, projectName)
	if err != nil {
		WriteError(w, ErrNotFound.WithDetails(fmt.Sprintf("Project not found: %s/%s", organization, projectName)))
		return
	}

	// Get repositories for this project
	repositories, err := h.db.GetRepositoriesByADOProject(ctx, organization, projectName)
	if err != nil {
		WriteError(w, ErrDatabaseFetch.WithDetails("repositories"))
		return
	}

	h.sendJSON(w, http.StatusOK, map[string]interface{}{
		"project":      project,
		"repositories": repositories,
		"total":        len(repositories),
	})
}

// ADODiscoveryStatus handles GET /api/v1/ado/discovery/status
// Returns status of ADO discovery across all organizations
func (h *ADOHandler) ADODiscoveryStatus(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	// Get optional organization filter
	organization := r.URL.Query().Get("organization")

	// Count total ADO repositories
	totalCount, err := h.db.CountRepositoriesByADOOrganization(ctx, organization)
	if err != nil {
		WriteError(w, ErrDatabaseFetch.WithDetails("repository count"))
		return
	}

	// Count repositories by status
	// For simplicity, we'll just report totals for now
	statusCounts := make(map[string]int)

	// Count TFVC repositories (need remediation)
	tfvcCount, err := h.db.CountTFVCRepositories(ctx, organization)
	if err != nil {
		h.logger.Error("Failed to count TFVC repositories", "error", err)
		tfvcCount = 0
	}

	// Get project counts
	projects, err := h.db.GetADOProjects(ctx, organization)
	if err != nil {
		h.logger.Error("Failed to fetch ADO projects", "error", err)
		projects = []models.ADOProject{}
	}

	h.sendJSON(w, http.StatusOK, map[string]interface{}{
		"total_repositories": totalCount,
		"total_projects":     len(projects),
		"tfvc_repositories":  tfvcCount,
		"git_repositories":   totalCount - tfvcCount,
		"status_breakdown":   statusCounts,
		"organization":       organization,
	})
}

// RediscoverADORepository handles rediscovery of a single ADO repository
// This is called when a user clicks "Rediscover" on an ADO repo
func (h *ADOHandler) RediscoverADORepository(ctx context.Context, repo *models.Repository) error {
	if repo.ADOProject == nil || *repo.ADOProject == "" {
		return fmt.Errorf("repository is not an ADO repository")
	}

	// Extract organization from full_name (format: org/project/repo)
	// For ADO repos, full_name should be in format "org/project/repo"
	parts := splitADOFullName(repo.FullName)
	if len(parts) < 3 {
		return fmt.Errorf("invalid ADO repository full_name format: %s", repo.FullName)
	}

	organization := parts[0]
	project := *repo.ADOProject
	repoName := parts[2]

	h.logger.Info("Rediscovering ADO repository",
		"organization", organization,
		"project", project,
		"repo", repoName,
		"full_name", repo.FullName)

	// Use the ADO collector to rediscover only this specific repository
	err := h.adoCollector.DiscoverADORepository(ctx, organization, project, repoName)
	if err != nil {
		return fmt.Errorf("failed to rediscover ADO repository: %w", err)
	}

	h.logger.Info("ADO repository rediscovered successfully",
		"organization", organization,
		"project", project,
		"repo", repoName)

	return nil
}

// splitADOFullName splits an ADO full name into parts
// Format: "org/project/repo" -> ["org", "project", "repo"]
func splitADOFullName(fullName string) []string {
	// For ADO repos, we expect "org/project/repo" format
	// We need to handle cases where project or repo names might contain slashes
	// For now, we'll use a simple split and take first 3 parts
	parts := make([]string, 0, 3)
	remainder := fullName

	// Split org (first part before /)
	if idx := findNthSlash(remainder, 0); idx >= 0 {
		parts = append(parts, remainder[:idx])
		remainder = remainder[idx+1:]
	} else {
		return []string{fullName}
	}

	// Split project (second part before /)
	if idx := findNthSlash(remainder, 0); idx >= 0 {
		parts = append(parts, remainder[:idx])
		remainder = remainder[idx+1:]
	} else {
		parts = append(parts, remainder)
		return parts
	}

	// Repo is everything else
	parts = append(parts, remainder)
	return parts
}

// findNthSlash finds the nth occurrence of '/' in a string
func findNthSlash(s string, n int) int {
	count := 0
	for i, c := range s {
		if c == '/' {
			if count == n {
				return i
			}
			count++
		}
	}
	return -1
}
