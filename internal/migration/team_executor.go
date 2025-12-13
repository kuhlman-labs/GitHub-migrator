package migration

import (
	"context"
	"fmt"
	"log/slog"
	"strings"
	"sync"
	"time"

	"github.com/kuhlman-labs/github-migrator/internal/github"
	"github.com/kuhlman-labs/github-migrator/internal/models"
	"github.com/kuhlman-labs/github-migrator/internal/storage"
)

// TeamExecutor handles the execution of team migrations
type TeamExecutor struct {
	storage    *storage.Database
	destClient *github.Client
	logger     *slog.Logger

	// Execution state
	mu        sync.Mutex
	running   bool
	cancelled bool
	progress  *TeamMigrationProgress
}

// TeamMigrationProgress tracks the progress of a team migration execution
type TeamMigrationProgress struct {
	TotalTeams       int        `json:"total_teams"`
	ProcessedTeams   int        `json:"processed_teams"`
	CreatedTeams     int        `json:"created_teams"`
	SkippedTeams     int        `json:"skipped_teams"`
	FailedTeams      int        `json:"failed_teams"`
	TotalReposSynced int        `json:"total_repos_synced"`
	StartedAt        time.Time  `json:"started_at"`
	CompletedAt      *time.Time `json:"completed_at,omitempty"`
	CurrentTeam      string     `json:"current_team,omitempty"`
	Status           string     `json:"status"` // pending, in_progress, completed, cancelled, failed
	Errors           []string   `json:"errors,omitempty"`
}

// NewTeamExecutor creates a new TeamExecutor
func NewTeamExecutor(storage *storage.Database, destClient *github.Client, logger *slog.Logger) *TeamExecutor {
	return &TeamExecutor{
		storage:    storage,
		destClient: destClient,
		logger:     logger,
	}
}

// IsRunning returns true if a team migration is currently running
func (e *TeamExecutor) IsRunning() bool {
	e.mu.Lock()
	defer e.mu.Unlock()
	return e.running
}

// GetProgress returns the current progress of the team migration
func (e *TeamExecutor) GetProgress() *TeamMigrationProgress {
	e.mu.Lock()
	defer e.mu.Unlock()
	if e.progress == nil {
		return nil
	}
	// Return a copy to avoid race conditions
	progressCopy := *e.progress
	return &progressCopy
}

// Cancel cancels the current team migration execution
func (e *TeamExecutor) Cancel() error {
	e.mu.Lock()
	defer e.mu.Unlock()
	if !e.running {
		return fmt.Errorf("no team migration is currently running")
	}
	e.cancelled = true
	return nil
}

// ExecuteTeamMigration executes team migration for all mapped teams, or a single team if both
// sourceOrgFilter AND sourceTeamSlugFilter are provided (both required to uniquely identify a team).
// Teams are created WITHOUT members (empty) to support EMU/IdP-managed environments.
// Only repository permissions are applied.
func (e *TeamExecutor) ExecuteTeamMigration(ctx context.Context, sourceOrgFilter string, sourceTeamSlugFilter string, dryRun bool) error {
	e.mu.Lock()
	if e.running {
		e.mu.Unlock()
		return fmt.Errorf("team migration is already running")
	}
	e.running = true
	e.cancelled = false
	e.progress = &TeamMigrationProgress{
		Status:    "in_progress",
		StartedAt: time.Now(),
	}
	e.mu.Unlock()

	// Ensure we clean up running state when done
	defer func() {
		e.mu.Lock()
		e.running = false
		e.mu.Unlock()
	}()

	e.logger.Info("Starting team migration execution",
		"source_org_filter", sourceOrgFilter,
		"source_team_slug_filter", sourceTeamSlugFilter,
		"dry_run", dryRun)

	// Get all mapped teams ready for migration
	mappings, err := e.storage.GetMappedTeamsForMigration(ctx, sourceOrgFilter)
	if err != nil {
		e.setProgressStatus("failed")
		return fmt.Errorf("failed to get mapped teams: %w", err)
	}

	// Filter to single team if specified
	if sourceTeamSlugFilter != "" && sourceOrgFilter != "" {
		var filteredMappings []*models.TeamMapping
		for _, m := range mappings {
			// Check both org and slug to ensure we migrate the correct team
			// (multiple orgs could have teams with the same slug)
			if m.SourceOrg == sourceOrgFilter && m.SourceTeamSlug == sourceTeamSlugFilter {
				filteredMappings = append(filteredMappings, m)
				break
			}
		}
		mappings = filteredMappings
	}

	e.mu.Lock()
	e.progress.TotalTeams = len(mappings)
	e.mu.Unlock()

	if len(mappings) == 0 {
		e.logger.Info("No teams to migrate")
		e.setProgressStatus("completed")
		return nil
	}

	e.logger.Info("Found teams to migrate", "count", len(mappings))

	// Process each team
	for _, mapping := range mappings {
		// Check for cancellation
		e.mu.Lock()
		if e.cancelled {
			e.progress.Status = "cancelled"
			e.mu.Unlock()
			e.logger.Info("Team migration cancelled by user")
			return nil
		}
		e.progress.CurrentTeam = mapping.SourceFullSlug()
		e.mu.Unlock()

		// Check context
		select {
		case <-ctx.Done():
			e.setProgressStatus("cancelled")
			return ctx.Err()
		default:
		}

		// Process this team
		err := e.processTeamMapping(ctx, mapping, dryRun)
		if err != nil {
			e.logger.Error("Failed to process team mapping",
				"team", mapping.SourceFullSlug(),
				"error", err)
			e.addError(fmt.Sprintf("%s: %s", mapping.SourceFullSlug(), err.Error()))
			e.incrementFailed()
		}

		e.incrementProcessed()
	}

	// Set final status and capture values for logging under mutex protection
	now := time.Now()
	e.mu.Lock()
	e.progress.CompletedAt = &now
	e.progress.CurrentTeam = ""
	if e.progress.FailedTeams > 0 {
		e.progress.Status = "completed_with_errors"
	} else {
		e.progress.Status = "completed"
	}
	// Copy values for logging while holding the mutex to avoid data race
	totalTeams := e.progress.TotalTeams
	createdTeams := e.progress.CreatedTeams
	skippedTeams := e.progress.SkippedTeams
	failedTeams := e.progress.FailedTeams
	totalReposSynced := e.progress.TotalReposSynced
	e.mu.Unlock()

	e.logger.Info("Team migration execution completed",
		"total", totalTeams,
		"created", createdTeams,
		"skipped", skippedTeams,
		"failed", failedTeams,
		"repos_synced", totalReposSynced)

	return nil
}

// processTeamMapping processes a single team mapping
// Handles both initial team creation and re-sync for newly migrated repos
//
//nolint:gocyclo // Complex orchestration logic with multiple API calls and error paths
func (e *TeamExecutor) processTeamMapping(ctx context.Context, mapping *models.TeamMapping, dryRun bool) error {
	if mapping.DestinationOrg == nil || mapping.DestinationTeamSlug == nil {
		return fmt.Errorf("destination org or team slug is not set")
	}

	destOrg := *mapping.DestinationOrg
	destTeamSlug := *mapping.DestinationTeamSlug
	isResync := mapping.TeamCreatedInDest // Team was already created in a previous run

	e.logger.Info("Processing team mapping",
		"source", mapping.SourceFullSlug(),
		"destination", destOrg+"/"+destTeamSlug,
		"dry_run", dryRun,
		"is_resync", isResync,
		"previous_repos_synced", mapping.ReposSynced)

	// Update status to in_progress
	if !dryRun {
		if err := e.storage.UpdateTeamMigrationStatus(ctx, mapping.SourceOrg, mapping.SourceTeamSlug, storage.TeamMigrationStatusInProgress, nil); err != nil {
			e.logger.Warn("Failed to update team migration status to in_progress", "error", err)
		}
	}

	// Step 1: Check if destination team exists
	existingTeam, err := e.destClient.GetTeamBySlug(ctx, destOrg, destTeamSlug)
	if err != nil {
		errMsg := err.Error()
		if !dryRun {
			_ = e.storage.UpdateTeamMigrationStatus(ctx, mapping.SourceOrg, mapping.SourceTeamSlug, storage.TeamMigrationStatusFailed, &errMsg)
		}
		return fmt.Errorf("failed to check if destination team exists: %w", err)
	}

	// Step 2: Create team if it doesn't exist
	teamCreatedNow := false
	if existingTeam == nil {
		if dryRun {
			e.logger.Info("DRY RUN: Would create team",
				"org", destOrg,
				"slug", destTeamSlug)
			// Note: When using PAT auth, would also remove PAT owner from team after creation
			if e.destClient.IsPATAuthenticated() {
				e.logger.Info("DRY RUN: Would remove PAT owner from team after creation (PAT auth detected)")
			}
		} else {
			// Get team name - use destination name if set, otherwise use source name
			teamName := destTeamSlug
			if mapping.DestinationTeamName != nil && *mapping.DestinationTeamName != "" {
				teamName = *mapping.DestinationTeamName
			} else if mapping.SourceTeamName != nil && *mapping.SourceTeamName != "" {
				teamName = *mapping.SourceTeamName
			}

			// Create the team (empty, without members)
			createdTeam, err := e.destClient.CreateTeam(ctx, destOrg, github.CreateTeamInput{
				Name:    teamName,
				Privacy: "closed", // Default to closed
			})
			if err != nil {
				errMsg := err.Error()
				_ = e.storage.UpdateTeamMigrationStatus(ctx, mapping.SourceOrg, mapping.SourceTeamSlug, storage.TeamMigrationStatusFailed, &errMsg)
				return fmt.Errorf("failed to create team: %w", err)
			}

			teamCreatedNow = true
			e.logger.Info("Created team in destination",
				"org", destOrg,
				"slug", destTeamSlug,
				"name", teamName)

			// Mark team as created in destination
			teamCreated := true
			_ = e.storage.UpdateTeamMigrationTracking(ctx, mapping.SourceOrg, mapping.SourceTeamSlug, storage.TeamMigrationTrackingUpdate{
				TeamCreatedInDest: &teamCreated,
			})

			// When using PAT authentication, GitHub automatically adds the PAT owner as a maintainer.
			// We need to remove them to ensure the team is created empty as intended.
			if e.destClient.IsPATAuthenticated() {
				patOwner, err := e.destClient.GetAuthenticatedUserLogin(ctx)
				if err != nil {
					e.logger.Warn("Failed to get PAT owner login, cannot remove from team",
						"error", err,
						"team", destOrg+"/"+createdTeam.Slug)
				} else {
					err = e.destClient.RemoveTeamMembership(ctx, destOrg, createdTeam.Slug, patOwner)
					if err != nil {
						e.logger.Warn("Failed to remove PAT owner from team",
							"error", err,
							"team", destOrg+"/"+createdTeam.Slug,
							"user", patOwner)
					} else {
						e.logger.Info("Removed PAT owner from team to ensure empty team",
							"team", destOrg+"/"+createdTeam.Slug,
							"user", patOwner)
					}
				}
			}
		}
		e.incrementCreated()
	} else {
		// Team exists - ensure we track that it exists in destination
		if !dryRun && !mapping.TeamCreatedInDest {
			teamCreated := true
			_ = e.storage.UpdateTeamMigrationTracking(ctx, mapping.SourceOrg, mapping.SourceTeamSlug, storage.TeamMigrationTrackingUpdate{
				TeamCreatedInDest: &teamCreated,
			})
		}

		if isResync {
			e.logger.Info("Re-syncing team permissions (team already exists)",
				"org", destOrg,
				"slug", destTeamSlug)
		} else {
			e.logger.Info("Team already exists in destination",
				"org", destOrg,
				"slug", destTeamSlug)
		}
		e.incrementSkipped()
	}

	// Step 3: Get source team and its repository permissions
	sourceTeam, err := e.storage.GetTeamByOrgAndSlug(ctx, mapping.SourceOrg, mapping.SourceTeamSlug)
	if err != nil || sourceTeam == nil {
		e.logger.Warn("Could not find source team in database, skipping repo permissions",
			"source_org", mapping.SourceOrg,
			"source_team_slug", mapping.SourceTeamSlug)
		// Still mark as completed if team was created
		if !dryRun {
			_ = e.storage.UpdateTeamMigrationStatus(ctx, mapping.SourceOrg, mapping.SourceTeamSlug, storage.TeamMigrationStatusCompleted, nil)
		}
		return nil
	}

	// Step 4: Update tracking counts
	if !dryRun {
		// Update total source repos count
		totalSourceRepos, err := e.storage.UpdateTeamTotalSourceRepos(ctx, sourceTeam.ID, mapping.SourceOrg, mapping.SourceTeamSlug)
		if err != nil {
			e.logger.Warn("Failed to update total source repos count", "error", err)
		} else {
			e.logger.Debug("Updated total source repos count", "count", totalSourceRepos)
		}

		// Update repos eligible count (migrated repos this team has access to)
		reposEligible, err := e.storage.UpdateTeamReposEligible(ctx, sourceTeam.ID, mapping.SourceOrg, mapping.SourceTeamSlug)
		if err != nil {
			e.logger.Warn("Failed to update repos eligible count", "error", err)
		} else {
			e.logger.Debug("Updated repos eligible count", "count", reposEligible)
		}
	}

	// Step 5: Apply repository permissions for migrated repos
	repos, err := e.storage.GetTeamRepositoriesForMigration(ctx, sourceTeam.ID)
	if err != nil {
		e.logger.Warn("Failed to get team repositories for migration", "error", err)
	} else {
		reposSynced := 0
		for _, repo := range repos {
			// Parse destination full name to get owner and repo
			parts := strings.SplitN(repo.DestFullName, "/", 2)
			if len(parts) != 2 {
				e.logger.Warn("Invalid destination full name", "full_name", repo.DestFullName)
				continue
			}
			destRepoOwner := parts[0]
			destRepoName := parts[1]

			if dryRun {
				e.logger.Info("DRY RUN: Would apply repo permission",
					"team", destOrg+"/"+destTeamSlug,
					"repo", repo.DestFullName,
					"permission", repo.Permission)
			} else {
				err := e.destClient.AddTeamRepoPermission(ctx, destOrg, destTeamSlug, destRepoOwner, destRepoName, repo.Permission)
				if err != nil {
					e.logger.Warn("Failed to apply repo permission",
						"team", destOrg+"/"+destTeamSlug,
						"repo", repo.DestFullName,
						"permission", repo.Permission,
						"error", err)
					continue
				}
				e.logger.Debug("Applied repo permission",
					"team", destOrg+"/"+destTeamSlug,
					"repo", repo.DestFullName,
					"permission", repo.Permission)
			}
			reposSynced++
		}

		if !dryRun {
			_ = e.storage.UpdateTeamReposSynced(ctx, mapping.SourceOrg, mapping.SourceTeamSlug, reposSynced)
		}

		e.addReposSynced(reposSynced)

		// Log summary
		if isResync {
			e.logger.Info("Re-sync completed",
				"team", mapping.SourceFullSlug(),
				"repos_synced", reposSynced,
				"team_created_now", teamCreatedNow)
		} else {
			e.logger.Info("Team migration completed",
				"team", mapping.SourceFullSlug(),
				"repos_synced", reposSynced,
				"repos_eligible", len(repos))
		}
	}

	// Step 6: Mark as completed
	if !dryRun {
		if err := e.storage.UpdateTeamMigrationStatus(ctx, mapping.SourceOrg, mapping.SourceTeamSlug, storage.TeamMigrationStatusCompleted, nil); err != nil {
			e.logger.Warn("Failed to update team migration status to completed", "error", err)
		}
	}

	return nil
}

// Helper methods for thread-safe progress updates
func (e *TeamExecutor) setProgressStatus(status string) {
	e.mu.Lock()
	defer e.mu.Unlock()
	e.progress.Status = status
	if status == "completed" || status == "failed" || status == "cancelled" {
		now := time.Now()
		e.progress.CompletedAt = &now
	}
}

func (e *TeamExecutor) incrementProcessed() {
	e.mu.Lock()
	defer e.mu.Unlock()
	e.progress.ProcessedTeams++
}

func (e *TeamExecutor) incrementCreated() {
	e.mu.Lock()
	defer e.mu.Unlock()
	e.progress.CreatedTeams++
}

func (e *TeamExecutor) incrementSkipped() {
	e.mu.Lock()
	defer e.mu.Unlock()
	e.progress.SkippedTeams++
}

func (e *TeamExecutor) incrementFailed() {
	e.mu.Lock()
	defer e.mu.Unlock()
	e.progress.FailedTeams++
}

func (e *TeamExecutor) addReposSynced(count int) {
	e.mu.Lock()
	defer e.mu.Unlock()
	e.progress.TotalReposSynced += count
}

func (e *TeamExecutor) addError(msg string) {
	e.mu.Lock()
	defer e.mu.Unlock()
	e.progress.Errors = append(e.progress.Errors, msg)
}
