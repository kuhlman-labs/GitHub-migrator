package handlers

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/brettkuhlman/github-migrator/internal/azuredevops"
	"github.com/brettkuhlman/github-migrator/internal/discovery"
	"github.com/brettkuhlman/github-migrator/internal/models"
	"github.com/brettkuhlman/github-migrator/internal/source"
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
		Organization string   `json:"organization"`           // Required: ADO organization name
		Projects     []string `json:"projects,omitempty"`     // Optional: specific projects to discover
		Workers      int      `json:"workers,omitempty"`      // Optional: number of parallel workers
		FullProfile  bool     `json:"full_profile,omitempty"` // Optional: perform full profiling
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.sendError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	if req.Organization == "" {
		h.sendError(w, http.StatusBadRequest, "organization is required")
		return
	}

	if req.Workers <= 0 {
		req.Workers = 5 // default
	}

	ctx := r.Context()

	// Start discovery based on scope
	var err error
	var discoveredCount int

	if len(req.Projects) == 0 {
		// Discover entire organization
		h.logger.Info("Starting ADO organization discovery",
			"organization", req.Organization,
			"workers", req.Workers,
			"full_profile", req.FullProfile)

		err = h.adoCollector.DiscoverADOOrganization(ctx, req.Organization)
	} else {
		// Discover specific projects
		h.logger.Info("Starting ADO project discovery",
			"organization", req.Organization,
			"projects", req.Projects,
			"workers", req.Workers,
			"full_profile", req.FullProfile)

		for _, project := range req.Projects {
			projectErr := h.adoCollector.DiscoverADOProject(ctx, req.Organization, project)
			if projectErr != nil {
				h.logger.Error("Failed to discover ADO project",
					"organization", req.Organization,
					"project", project,
					"error", projectErr)
				// Continue with other projects
				continue
			}
		}
		// Count discovered repositories after project discovery
		discoveredCount, _ = h.db.CountRepositoriesByADOProjects(ctx, req.Organization, req.Projects)
	}

	if err != nil {
		h.sendError(w, http.StatusInternalServerError, "Discovery failed")
		return
	}

	// Count total discovered repositories for the organization
	if discoveredCount == 0 {
		discoveredCount, _ = h.db.CountRepositoriesByADOOrganization(ctx, req.Organization)
	}

	h.sendJSON(w, http.StatusOK, map[string]interface{}{
		"message":      "ADO discovery completed",
		"organization": req.Organization,
		"projects":     req.Projects,
		"repositories": discoveredCount,
		"full_profile": req.FullProfile,
	})
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
		h.sendError(w, http.StatusInternalServerError, "Failed to fetch ADO projects")
		return
	}

	// Enrich projects with repository counts
	type ProjectWithCount struct {
		*models.ADOProject
		RepositoryCount int `json:"repository_count"`
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

		enrichedProjects = append(enrichedProjects, ProjectWithCount{
			ADOProject:      &project,
			RepositoryCount: count,
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
		h.sendError(w, http.StatusBadRequest, "organization and project are required")
		return
	}

	// Get project from database
	project, err := h.db.GetADOProject(ctx, organization, projectName)
	if err != nil {
		h.sendError(w, http.StatusNotFound, fmt.Sprintf("Project not found: %s/%s", organization, projectName))
		return
	}

	// Get repositories for this project
	repositories, err := h.db.GetRepositoriesByADOProject(ctx, organization, projectName)
	if err != nil {
		h.sendError(w, http.StatusInternalServerError, "Failed to fetch repositories")
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
		h.sendError(w, http.StatusInternalServerError, "Failed to count repositories")
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
