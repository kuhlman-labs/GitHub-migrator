package discovery

import (
	"context"
	"log/slog"

	"github.com/kuhlman-labs/github-migrator/internal/github"
	"github.com/kuhlman-labs/github-migrator/internal/models"
	"github.com/kuhlman-labs/github-migrator/internal/storage"
)

// TeamMemberSaver provides common functionality for saving team members
type TeamMemberSaver struct {
	storage *storage.Database
	logger  *slog.Logger
}

// NewTeamMemberSaver creates a new TeamMemberSaver
func NewTeamMemberSaver(storage *storage.Database, logger *slog.Logger) *TeamMemberSaver {
	return &TeamMemberSaver{
		storage: storage,
		logger:  logger,
	}
}

// SaveMemberParams contains parameters for saving team members
type SaveMemberParams struct {
	WorkerID       int
	Organization   string
	TeamSlug       string
	TeamID         int64
	Members        []*github.TeamMember
	SourceInstance string
}

// SaveTeamMembersResult contains the result of saving team members
type SaveTeamMembersResult struct {
	SavedCount int
}

// SaveTeamMembers saves team members and associated users
func (s *TeamMemberSaver) SaveTeamMembers(ctx context.Context, params SaveMemberParams) SaveTeamMembersResult {
	result := SaveTeamMembersResult{}

	for _, member := range params.Members {
		// Save team member relationship
		teamMember := &models.GitHubTeamMember{
			TeamID: params.TeamID,
			Login:  member.Login,
			Role:   member.Role,
		}
		if err := s.storage.SaveTeamMember(ctx, teamMember); err != nil {
			s.logger.Warn("Failed to save team member",
				"worker_id", params.WorkerID,
				"organization", params.Organization,
				"team", params.TeamSlug,
				"member", member.Login,
				"error", err)
			continue
		}
		result.SavedCount++

		// Also save the user to github_users table
		user := &models.GitHubUser{
			Login:          member.Login,
			SourceInstance: params.SourceInstance,
		}
		if err := s.storage.SaveUser(ctx, user); err != nil {
			s.logger.Debug("User may already exist", "login", member.Login, "error", err)
		}
	}

	return result
}
