package handlers

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"

	"github.com/kuhlman-labs/github-migrator/internal/azuredevops"
	"github.com/kuhlman-labs/github-migrator/internal/discovery"
	"github.com/kuhlman-labs/github-migrator/internal/models"
	"github.com/kuhlman-labs/github-migrator/internal/source"
	"github.com/kuhlman-labs/github-migrator/internal/storage"
)

// ADOHandler contains Azure DevOps-specific HTTP handlers
type ADOHandler struct {
	*Handler
	adoClient    *azuredevops.Client
	adoProvider  source.Provider
	adoCollector *discovery.ADOCollector // Specialized collector for ADO
}

// NewADOHandler creates a new ADOHandler instance
func NewADOHandler(baseHandler *Handler, adoClient *azuredevops.Client, adoProvider source.Provider) *ADOHandler {
	adoCollector := discovery.NewADOCollector(adoClient, baseHandler.db, baseHandler.logger, adoProvider)
	return &ADOHandler{
		Handler:      baseHandler,
		adoClient:    adoClient,
		adoProvider:  adoProvider,
		adoCollector: adoCollector,
	}
}

// StartADODiscovery handles POST /api/v1/ado/discover
// Discovers ADO organizations or specific projects
func (h *ADOHandler) StartADODiscovery(w http.ResponseWriter, r *http.Request) {
	var req StartADODiscoveryRequest

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

	// Check for existing in-progress discovery
	if err := h.checkNoActiveDiscovery(w); err != nil {
		return
	}

	// Verify ADO client and collector are available before creating progress record
	if h.adoClient == nil || h.adoCollector == nil {
		WriteError(w, ErrServiceUnavailable.WithDetails("ADO client not configured"))
		return
	}

	// Create progress record
	progress, err := h.createADODiscoveryProgress(req)
	if err != nil {
		h.handleProgressCreationError(w, err)
		return
	}

	// Create progress tracker and start discovery
	progressTracker := discovery.NewDBProgressTracker(h.db, h.logger, progress)

	if len(req.Projects) == 0 {
		h.startADOOrgDiscovery(w, req, progress, progressTracker)
	} else {
		h.startADOProjectDiscovery(w, req, progress, progressTracker)
	}
}

// checkNoActiveDiscovery checks if there's an existing discovery in progress
func (h *ADOHandler) checkNoActiveDiscovery(w http.ResponseWriter) error {
	existingProgress, err := h.db.GetActiveDiscoveryProgress()
	if err != nil {
		h.logger.Error("Failed to check for active discovery", "error", err)
		WriteError(w, ErrDatabaseFetch.WithDetails("discovery status"))
		return fmt.Errorf("failed to check discovery status: %w", err)
	}
	if existingProgress != nil {
		WriteError(w, ErrConflict.WithDetails("Another discovery is already in progress"))
		return fmt.Errorf("discovery in progress")
	}
	return nil
}

// createADODiscoveryProgress creates a progress record for ADO discovery
func (h *ADOHandler) createADODiscoveryProgress(req StartADODiscoveryRequest) (*models.DiscoveryProgress, error) {
	var discoveryType, target string
	var totalOrgs int
	if len(req.Projects) == 0 {
		discoveryType = models.DiscoveryTypeADOOrganization
		target = req.Organization
		totalOrgs = 1
	} else {
		discoveryType = models.DiscoveryTypeADOProject
		target = req.Organization + "/" + strings.Join(req.Projects, ",")
		totalOrgs = len(req.Projects)
	}

	progress := &models.DiscoveryProgress{
		DiscoveryType: discoveryType,
		Target:        target,
		TotalOrgs:     totalOrgs,
	}

	if err := h.db.CreateDiscoveryProgress(progress); err != nil {
		return nil, err
	}
	return progress, nil
}

// handleProgressCreationError handles errors during progress creation
func (h *ADOHandler) handleProgressCreationError(w http.ResponseWriter, err error) {
	if errors.Is(err, storage.ErrDiscoveryInProgress) {
		WriteError(w, ErrConflict.WithDetails(err.Error()))
		return
	}
	h.logger.Error("Failed to create discovery progress", "error", err)
	WriteError(w, ErrDatabaseUpdate.WithDetails("discovery progress initialization"))
}

// startADOOrgDiscovery starts organization-wide ADO discovery
// Caller must ensure h.adoCollector and h.adoClient are not nil
func (h *ADOHandler) startADOOrgDiscovery(w http.ResponseWriter, req StartADODiscoveryRequest, progress *models.DiscoveryProgress, tracker *discovery.DBProgressTracker) {
	h.logger.Info("Starting ADO organization discovery",
		"organization", req.Organization,
		"workers", req.Workers,
		"progress_id", progress.ID)

	h.adoCollector.SetProgressTracker(tracker)
	go h.runADOOrgDiscovery(req.Organization, progress.ID, tracker)

	h.sendJSON(w, http.StatusAccepted, map[string]any{
		"message":      "ADO organization discovery started",
		"organization": req.Organization,
		"type":         "organization",
		"progress_id":  progress.ID,
	})
}

// runADOOrgDiscovery executes organization discovery in background
func (h *ADOHandler) runADOOrgDiscovery(organization string, progressID int64, tracker *discovery.DBProgressTracker) {
	// Create cancellable context
	ctx, cancel := context.WithCancel(context.Background())

	// Register cancel function for this discovery
	h.discoveryMu.Lock()
	h.discoveryCancel[progressID] = cancel
	h.discoveryMu.Unlock()

	// Clean up when done
	defer func() {
		h.discoveryMu.Lock()
		delete(h.discoveryCancel, progressID)
		h.discoveryMu.Unlock()
		cancel()
		tracker.Flush()
	}()

	if err := h.adoCollector.DiscoverADOOrganization(ctx, organization); err != nil {
		// Check if this was a cancellation
		if ctx.Err() == context.Canceled {
			h.logger.Info("ADO organization discovery cancelled", "organization", organization)
			if dbErr := h.db.MarkDiscoveryCancelled(progressID); dbErr != nil {
				h.logger.Error("Failed to mark discovery as cancelled", "error", dbErr)
			}
			return
		}

		h.logger.Error("ADO organization discovery failed", "organization", organization, "error", err)
		if markErr := h.db.MarkDiscoveryFailed(progressID, err.Error()); markErr != nil {
			h.logger.Error("Failed to mark discovery as failed", "error", markErr)
		}
		return
	}

	// Check if cancelled even on success (rare but possible)
	if ctx.Err() == context.Canceled {
		h.logger.Info("ADO organization discovery cancelled", "organization", organization)
		if dbErr := h.db.MarkDiscoveryCancelled(progressID); dbErr != nil {
			h.logger.Error("Failed to mark discovery as cancelled", "error", dbErr)
		}
		return
	}

	h.logger.Info("ADO organization discovery completed", "organization", organization)
	if markErr := h.db.MarkDiscoveryComplete(progressID); markErr != nil {
		h.logger.Error("Failed to mark discovery as complete", "error", markErr)
	}
}

// startADOProjectDiscovery starts project-specific ADO discovery
// Caller must ensure h.adoCollector and h.adoClient are not nil
func (h *ADOHandler) startADOProjectDiscovery(w http.ResponseWriter, req StartADODiscoveryRequest, progress *models.DiscoveryProgress, tracker *discovery.DBProgressTracker) {
	h.logger.Info("Starting ADO project discovery",
		"organization", req.Organization,
		"projects", req.Projects,
		"workers", req.Workers,
		"progress_id", progress.ID)

	h.adoCollector.SetProgressTracker(tracker)
	go h.runADOProjectDiscovery(req.Organization, req.Projects, progress.ID, tracker)

	h.sendJSON(w, http.StatusAccepted, map[string]any{
		"message":      "ADO project discovery started",
		"organization": req.Organization,
		"projects":     req.Projects,
		"type":         "project",
		"progress_id":  progress.ID,
	})
}

// runADOProjectDiscovery executes project discovery in background
func (h *ADOHandler) runADOProjectDiscovery(organization string, projects []string, progressID int64, tracker *discovery.DBProgressTracker) {
	// Create cancellable context
	ctx, cancel := context.WithCancel(context.Background())

	// Register cancel function for this discovery
	h.discoveryMu.Lock()
	h.discoveryCancel[progressID] = cancel
	h.discoveryMu.Unlock()

	// Clean up when done
	defer func() {
		h.discoveryMu.Lock()
		delete(h.discoveryCancel, progressID)
		h.discoveryMu.Unlock()
		cancel()
		tracker.Flush()
	}()

	var lastErr error
	for i, project := range projects {
		// Check for cancellation before starting each project
		if ctx.Err() == context.Canceled {
			h.logger.Info("ADO project discovery cancelled", "organization", organization, "completed_projects", i)
			if dbErr := h.db.MarkDiscoveryCancelled(progressID); dbErr != nil {
				h.logger.Error("Failed to mark discovery as cancelled", "error", dbErr)
			}
			return
		}

		tracker.StartOrg(project, i)
		if err := h.adoCollector.DiscoverADOProject(ctx, organization, project); err != nil {
			// Check if this was a cancellation
			if ctx.Err() == context.Canceled {
				h.logger.Info("ADO project discovery cancelled", "organization", organization, "project", project)
				if dbErr := h.db.MarkDiscoveryCancelled(progressID); dbErr != nil {
					h.logger.Error("Failed to mark discovery as cancelled", "error", dbErr)
				}
				return
			}

			h.logger.Error("Failed to discover ADO project", "organization", organization, "project", project, "error", err)
			tracker.RecordError(err)
			lastErr = err
		} else {
			h.logger.Info("ADO project discovery completed", "organization", organization, "project", project)
		}
		tracker.CompleteOrg(project, 0)
	}

	if lastErr != nil {
		if markErr := h.db.MarkDiscoveryFailed(progressID, lastErr.Error()); markErr != nil {
			h.logger.Error("Failed to mark discovery as failed", "error", markErr)
		}
	} else {
		if markErr := h.db.MarkDiscoveryComplete(progressID); markErr != nil {
			h.logger.Error("Failed to mark discovery as complete", "error", markErr)
		}
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

	h.sendJSON(w, http.StatusOK, map[string]any{
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

	h.sendJSON(w, http.StatusOK, map[string]any{
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

	h.sendJSON(w, http.StatusOK, map[string]any{
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
