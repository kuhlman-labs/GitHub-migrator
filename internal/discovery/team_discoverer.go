package discovery

import (
	"context"
	"fmt"
	"log/slog"
	"sync"

	"github.com/kuhlman-labs/github-migrator/internal/github"
	"github.com/kuhlman-labs/github-migrator/internal/models"
	"github.com/kuhlman-labs/github-migrator/internal/storage"
)

// TeamDiscoverer handles discovery of teams and their members.
// It is a focused component extracted from the larger Collector struct.
type TeamDiscoverer struct {
	storage *storage.Database
	logger  *slog.Logger
	workers int
}

// NewTeamDiscoverer creates a new TeamDiscoverer.
func NewTeamDiscoverer(storage *storage.Database, logger *slog.Logger, workers int) *TeamDiscoverer {
	if workers <= 0 {
		workers = 5
	}
	return &TeamDiscoverer{
		storage: storage,
		logger:  logger,
		workers: workers,
	}
}

// teamResult holds the result of processing a single team
type teamResult struct {
	teamSaved   bool
	memberCount int
	repoCount   int
	err         error
}

// DiscoverTeams discovers all teams for an organization and their repository associations.
// Uses parallel processing with worker pool for improved performance.
func (d *TeamDiscoverer) DiscoverTeams(ctx context.Context, org string, client *github.Client) error {
	d.logger.Info("Discovering teams for organization", "organization", org)

	// List all teams in the organization
	teams, err := client.ListOrganizationTeams(ctx, org)
	if err != nil {
		return fmt.Errorf("failed to list teams: %w", err)
	}

	d.logger.Info("Found teams", "organization", org, "count", len(teams), "workers", d.workers)

	if len(teams) == 0 {
		return nil
	}

	// Process teams in parallel
	jobs := make(chan *github.TeamInfo, len(teams))
	results := make(chan teamResult, len(teams))
	var wg sync.WaitGroup

	// Start workers
	for i := 0; i < d.workers; i++ {
		wg.Add(1)
		go d.teamWorker(ctx, &wg, i, org, client, jobs, results)
	}

	// Send jobs
	for _, team := range teams {
		jobs <- team
	}
	close(jobs)

	// Wait for completion
	wg.Wait()
	close(results)

	// Collect results
	teamCount := 0
	totalMembers := 0
	totalRepos := 0
	var errors []error

	for result := range results {
		if result.err != nil {
			errors = append(errors, result.err)
		}
		if result.teamSaved {
			teamCount++
		}
		totalMembers += result.memberCount
		totalRepos += result.repoCount
	}

	d.logger.Info("Team discovery complete",
		"organization", org,
		"teams_saved", teamCount,
		"total_members", totalMembers,
		"total_repo_associations", totalRepos,
		"errors", len(errors))

	if len(errors) > 0 {
		return fmt.Errorf("encountered %d errors during team discovery", len(errors))
	}

	return nil
}

// DiscoverTeamsOnly discovers only teams and their members without repository associations.
// Returns (teams saved, members saved, error).
func (d *TeamDiscoverer) DiscoverTeamsOnly(ctx context.Context, org string, client *github.Client, sourceInstance string) (int, int, error) {
	d.logger.Info("Starting teams-only discovery", "organization", org)

	teams, err := client.ListOrganizationTeams(ctx, org)
	if err != nil {
		return 0, 0, fmt.Errorf("failed to list teams: %w", err)
	}

	d.logger.Info("Found teams", "organization", org, "count", len(teams), "workers", d.workers)

	if len(teams) == 0 {
		return 0, 0, nil
	}

	// Process teams in parallel
	jobs := make(chan *github.TeamInfo, len(teams))
	results := make(chan teamResult, len(teams))
	var wg sync.WaitGroup

	// Start workers
	for i := 0; i < d.workers; i++ {
		wg.Add(1)
		go d.teamsOnlyWorker(ctx, &wg, i, org, client, sourceInstance, jobs, results)
	}

	// Send jobs
	for _, team := range teams {
		jobs <- team
	}
	close(jobs)

	// Wait for completion
	wg.Wait()
	close(results)

	// Collect results
	teamCount := 0
	memberCount := 0
	for result := range results {
		if result.teamSaved {
			teamCount++
		}
		memberCount += result.memberCount
	}

	d.logger.Info("Teams-only discovery complete",
		"organization", org,
		"teams_saved", teamCount,
		"members_saved", memberCount)

	return teamCount, memberCount, nil
}

// teamWorker processes teams including repository associations.
func (d *TeamDiscoverer) teamWorker(ctx context.Context, wg *sync.WaitGroup, workerID int, org string, client *github.Client, jobs <-chan *github.TeamInfo, results chan<- teamResult) {
	defer wg.Done()

	for teamInfo := range jobs {
		result := d.processTeamFull(ctx, workerID, org, client, teamInfo)
		results <- result
	}
}

// processTeamFull processes a single team including repository associations.
func (d *TeamDiscoverer) processTeamFull(ctx context.Context, workerID int, org string, client *github.Client, teamInfo *github.TeamInfo) teamResult {
	result := teamResult{}

	d.logger.Debug("Worker processing team",
		"worker_id", workerID,
		"organization", org,
		"team", teamInfo.Slug)

	// Save the team
	team := &models.GitHubTeam{
		Organization: org,
		Slug:         teamInfo.Slug,
		Name:         teamInfo.Name,
		Privacy:      teamInfo.Privacy,
	}
	if teamInfo.Description != "" {
		team.Description = stringPtr(teamInfo.Description)
	}

	if err := d.storage.SaveTeam(ctx, team); err != nil {
		d.logger.Warn("Failed to save team",
			"worker_id", workerID,
			"organization", org,
			"team", teamInfo.Slug,
			"error", err)
		result.err = err
		return result
	}
	result.teamSaved = true

	// List repositories for this team
	teamRepos, err := client.ListTeamRepositories(ctx, org, teamInfo.Slug)
	if err != nil {
		d.logger.Warn("Failed to list repositories for team",
			"worker_id", workerID,
			"organization", org,
			"team", teamInfo.Slug,
			"error", err)
	} else {
		for _, teamRepo := range teamRepos {
			if err := d.storage.SaveTeamRepository(ctx, team.ID, teamRepo.FullName, teamRepo.Permission); err != nil {
				d.logger.Warn("Failed to save team-repository association",
					"worker_id", workerID,
					"organization", org,
					"team", teamInfo.Slug,
					"repo", teamRepo.FullName,
					"error", err)
			} else {
				result.repoCount++
			}
		}
	}

	// List members using GraphQL
	teamMembers, err := client.ListTeamMembersGraphQL(ctx, org, teamInfo.Slug)
	if err != nil {
		d.logger.Warn("Failed to list members for team",
			"worker_id", workerID,
			"organization", org,
			"team", teamInfo.Slug,
			"error", err)
	} else {
		for _, member := range teamMembers {
			teamMember := &models.GitHubTeamMember{
				TeamID: team.ID,
				Login:  member.Login,
				Role:   member.Role,
			}
			if err := d.storage.SaveTeamMember(ctx, teamMember); err != nil {
				d.logger.Warn("Failed to save team member",
					"worker_id", workerID,
					"organization", org,
					"team", teamInfo.Slug,
					"member", member.Login,
					"error", err)
			} else {
				result.memberCount++
			}
		}
	}

	d.logger.Debug("Worker completed team",
		"worker_id", workerID,
		"organization", org,
		"team", teamInfo.Slug,
		"members", result.memberCount,
		"repos", result.repoCount)

	return result
}

// teamsOnlyWorker processes teams without repository associations.
func (d *TeamDiscoverer) teamsOnlyWorker(ctx context.Context, wg *sync.WaitGroup, workerID int, org string, client *github.Client, sourceInstance string, jobs <-chan *github.TeamInfo, results chan<- teamResult) {
	defer wg.Done()

	for teamInfo := range jobs {
		result := teamResult{}

		d.logger.Debug("Worker processing team",
			"worker_id", workerID,
			"organization", org,
			"team", teamInfo.Slug)

		team := &models.GitHubTeam{
			Organization: org,
			Slug:         teamInfo.Slug,
			Name:         teamInfo.Name,
			Privacy:      teamInfo.Privacy,
		}
		if teamInfo.Description != "" {
			team.Description = stringPtr(teamInfo.Description)
		}

		if err := d.storage.SaveTeam(ctx, team); err != nil {
			d.logger.Warn("Failed to save team",
				"worker_id", workerID,
				"organization", org,
				"team", teamInfo.Slug,
				"error", err)
			results <- result
			continue
		}
		result.teamSaved = true

		// List and save team members
		teamMembers, err := client.ListTeamMembersGraphQL(ctx, org, teamInfo.Slug)
		if err != nil {
			d.logger.Warn("Failed to list members for team",
				"worker_id", workerID,
				"organization", org,
				"team", teamInfo.Slug,
				"error", err)
			results <- result
			continue
		}

		// Use shared helper to save team members
		saver := NewTeamMemberSaver(d.storage, d.logger)
		saveResult := saver.SaveTeamMembers(ctx, SaveMemberParams{
			WorkerID:       workerID,
			Organization:   org,
			TeamSlug:       teamInfo.Slug,
			TeamID:         team.ID,
			Members:        teamMembers,
			SourceInstance: sourceInstance,
		})
		result.memberCount = saveResult.SavedCount

		results <- result
	}
}
