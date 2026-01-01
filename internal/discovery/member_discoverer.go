package discovery

import (
	"context"
	"fmt"
	"log/slog"
	"net/url"

	"github.com/kuhlman-labs/github-migrator/internal/github"
	"github.com/kuhlman-labs/github-migrator/internal/models"
	"github.com/kuhlman-labs/github-migrator/internal/storage"
)

// hostGitHubCom is already defined in package_scanner.go

// MemberDiscoverer handles discovery of organization members.
// It is a focused component extracted from the larger Collector struct.
type MemberDiscoverer struct {
	storage  *storage.Database
	logger   *slog.Logger
	sourceID *int64 // Optional source ID for multi-source support
}

// NewMemberDiscoverer creates a new MemberDiscoverer.
func NewMemberDiscoverer(storage *storage.Database, logger *slog.Logger) *MemberDiscoverer {
	return &MemberDiscoverer{
		storage: storage,
		logger:  logger,
	}
}

// SetSourceID sets the source ID to associate with discovered users
func (d *MemberDiscoverer) SetSourceID(sourceID *int64) {
	d.sourceID = sourceID
}

// MemberDiscoveryResult contains the results of member discovery.
type MemberDiscoveryResult struct {
	TotalMembers     int
	UsersSaved       int
	MembershipsSaved int
	Errors           []error
}

// DiscoverOrgMembers discovers all members of an organization and saves them as users.
// Also saves org membership to track which orgs each user belongs to.
func (d *MemberDiscoverer) DiscoverOrgMembers(ctx context.Context, org string, client *github.Client, sourceInstance string) (*MemberDiscoveryResult, error) {
	d.logger.Info("Discovering organization members", "organization", org)

	members, err := client.ListOrgMembers(ctx, org)
	if err != nil {
		return nil, fmt.Errorf("failed to list org members: %w", err)
	}

	d.logger.Info("Found organization members", "organization", org, "count", len(members))

	result := &MemberDiscoveryResult{
		TotalMembers: len(members),
	}

	for _, member := range members {
		user := &models.GitHubUser{
			SourceID:       d.sourceID, // Associate with source for multi-source support
			Login:          member.Login,
			Name:           member.Name,
			Email:          member.Email,
			SourceInstance: sourceInstance,
		}
		if member.AvatarURL != "" {
			avatarURL := member.AvatarURL
			user.AvatarURL = &avatarURL
		}

		if err := d.storage.SaveUser(ctx, user); err != nil {
			d.logger.Warn("Failed to save organization member",
				"organization", org,
				"login", member.Login,
				"error", err)
			result.Errors = append(result.Errors, err)
			continue
		}
		result.UsersSaved++

		// Save org membership
		membership := &models.UserOrgMembership{
			UserLogin:    member.Login,
			Organization: org,
			Role:         member.Role,
		}
		if err := d.storage.SaveUserOrgMembership(ctx, membership); err != nil {
			d.logger.Warn("Failed to save org membership",
				"organization", org,
				"login", member.Login,
				"error", err)
		} else {
			result.MembershipsSaved++
		}
	}

	d.logger.Info("Organization member discovery complete",
		"organization", org,
		"total_members", result.TotalMembers,
		"users_saved", result.UsersSaved,
		"memberships_saved", result.MembershipsSaved)

	return result, nil
}

// DiscoverOrgMembersOnly discovers only organization members without any other discovery.
// This is used for standalone user discovery from the Users page.
// Returns the number of users saved.
func (d *MemberDiscoverer) DiscoverOrgMembersOnly(ctx context.Context, org string, client *github.Client, sourceInstance string) (int, error) {
	result, err := d.DiscoverOrgMembers(ctx, org, client, sourceInstance)
	if err != nil {
		return 0, err
	}
	return result.UsersSaved, nil
}

// GetSourceInstance returns the source GitHub instance hostname from a client.
func GetSourceInstance(client *github.Client) string {
	if client == nil {
		return hostGitHubCom
	}

	baseURL := client.BaseURL()
	if baseURL == "" {
		return hostGitHubCom
	}

	parsed, err := url.Parse(baseURL)
	if err != nil {
		return hostGitHubCom
	}

	host := parsed.Host
	if host == "" {
		return hostGitHubCom
	}

	return host
}
