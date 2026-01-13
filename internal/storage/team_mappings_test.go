package storage

import (
	"context"
	"testing"

	"github.com/kuhlman-labs/github-migrator/internal/config"
	"github.com/kuhlman-labs/github-migrator/internal/models"
	"github.com/stretchr/testify/require"
)

func setupTeamMappingsTestDB(t *testing.T) *Database {
	t.Helper()
	cfg := config.DatabaseConfig{
		Type: "sqlite",
		DSN:  ":memory:",
	}

	db, err := NewDatabase(cfg)
	if err != nil {
		t.Fatalf("failed to create test database: %v", err)
	}

	if err := db.Migrate(); err != nil {
		t.Fatalf("failed to run migrations: %v", err)
	}

	t.Cleanup(func() {
		_ = db.Close()
	})

	return db
}

// createTestTeam creates a minimal GitHubTeam for testing
func createTestTeam(org, slug, name string) *models.GitHubTeam {
	return &models.GitHubTeam{
		Organization: org,
		Slug:         slug,
		Name:         name,
		Privacy:      "closed",
	}
}

// createTestTeamMapping creates a minimal TeamMapping for testing
func createTestTeamMapping(sourceOrg, sourceTeamSlug string) *models.TeamMapping {
	return &models.TeamMapping{
		SourceOrg:      sourceOrg,
		SourceTeamSlug: sourceTeamSlug,
		MappingStatus:  "mapped",
	}
}

func TestListTeamsWithMappings(t *testing.T) {
	db := setupTeamMappingsTestDB(t)
	ctx := context.Background()

	// Create teams
	team1 := createTestTeam("org1", "team-alpha", "Team Alpha")
	team2 := createTestTeam("org1", "team-beta", "Team Beta")
	team3 := createTestTeam("org2", "team-gamma", "Team Gamma")

	for _, team := range []*models.GitHubTeam{team1, team2, team3} {
		err := db.db.WithContext(ctx).Create(team).Error
		require.NoError(t, err, "failed to create team")
	}

	// Create mapping for team1
	mapping1 := createTestTeamMapping("org1", "team-alpha")
	mapping1.DestinationOrg = strPtr("dest-org")
	mapping1.DestinationTeamSlug = strPtr("team-alpha")
	mapping1.DestinationTeamName = strPtr("Team Alpha")
	err := db.SaveTeamMapping(ctx, mapping1)
	require.NoError(t, err, "failed to save team mapping")

	t.Run("list all teams", func(t *testing.T) {
		teams, total, err := db.ListTeamsWithMappings(ctx, TeamWithMappingFilters{
			Limit: 100,
		})
		require.NoError(t, err)
		require.Equal(t, int64(3), total)
		require.Len(t, teams, 3)
	})

	t.Run("filter by organization", func(t *testing.T) {
		teams, total, err := db.ListTeamsWithMappings(ctx, TeamWithMappingFilters{
			Organization: "org1",
			Limit:        100,
		})
		require.NoError(t, err)
		require.Equal(t, int64(2), total)
		require.Len(t, teams, 2)
	})

	t.Run("filter by mapped status", func(t *testing.T) {
		teams, total, err := db.ListTeamsWithMappings(ctx, TeamWithMappingFilters{
			Status: "mapped",
			Limit:  100,
		})
		require.NoError(t, err)
		require.Equal(t, int64(1), total)
		require.Len(t, teams, 1)
		require.Equal(t, "team-alpha", teams[0].Slug)
	})

	t.Run("filter by unmapped status", func(t *testing.T) {
		teams, total, err := db.ListTeamsWithMappings(ctx, TeamWithMappingFilters{
			Status: "unmapped",
			Limit:  100,
		})
		require.NoError(t, err)
		require.Equal(t, int64(2), total)
		require.Len(t, teams, 2)
	})

	t.Run("search by name", func(t *testing.T) {
		teams, total, err := db.ListTeamsWithMappings(ctx, TeamWithMappingFilters{
			Search: "gamma",
			Limit:  100,
		})
		require.NoError(t, err)
		require.Equal(t, int64(1), total)
		require.Len(t, teams, 1)
		require.Equal(t, "team-gamma", teams[0].Slug)
	})

	t.Run("pagination", func(t *testing.T) {
		teams, total, err := db.ListTeamsWithMappings(ctx, TeamWithMappingFilters{
			Limit:  2,
			Offset: 0,
		})
		require.NoError(t, err)
		require.Equal(t, int64(3), total)
		require.Len(t, teams, 2)

		// Get second page
		teams2, _, err := db.ListTeamsWithMappings(ctx, TeamWithMappingFilters{
			Limit:  2,
			Offset: 2,
		})
		require.NoError(t, err)
		require.Len(t, teams2, 1)
	})
}

func TestListTeamsWithMappings_BooleanDialect(t *testing.T) {
	// This test specifically verifies that the dialect-specific boolean handling works correctly
	// The query uses COALESCE(m.team_created_in_dest, <bool_false>) = <bool_true>
	// which must use dialect-appropriate values (TRUE/FALSE for PostgreSQL, 1/0 for SQLite)
	db := setupTeamMappingsTestDB(t)
	ctx := context.Background()

	// Create a team
	team := createTestTeam("test-org", "test-team", "Test Team")
	err := db.db.WithContext(ctx).Create(team).Error
	require.NoError(t, err)

	// Create a mapping with team_created_in_dest = true
	mapping := &models.TeamMapping{
		SourceOrg:           "test-org",
		SourceTeamSlug:      "test-team",
		MappingStatus:       "mapped",
		MigrationStatus:     "in_progress",
		TeamCreatedInDest:   true,
		DestinationOrg:      strPtr("dest-org"),
		DestinationTeamSlug: strPtr("test-team"),
	}
	err = db.SaveTeamMapping(ctx, mapping)
	require.NoError(t, err)

	// Query the teams - this exercises the COALESCE with boolean comparison
	teams, total, err := db.ListTeamsWithMappings(ctx, TeamWithMappingFilters{
		Organization: "test-org",
		Limit:        100,
	})
	require.NoError(t, err, "ListTeamsWithMappings should not fail with boolean dialect handling")
	require.Equal(t, int64(1), total)
	require.Len(t, teams, 1)

	// Verify the team_created_in_dest value is correctly returned
	require.True(t, teams[0].TeamCreatedInDest, "team_created_in_dest should be true")

	// The sync_status should be 'team_only' since:
	// - migration_status is 'in_progress' (not pending, not failed)
	// - team_created_in_dest is true
	// - repos_eligible is 0 (no repos associated)
	require.Equal(t, "team_only", teams[0].SyncStatus, "sync_status should be 'team_only' for team with no eligible repos")
}

func TestListTeamsWithMappings_TeamCreatedInDestFalse(t *testing.T) {
	db := setupTeamMappingsTestDB(t)
	ctx := context.Background()

	// Create a team
	team := createTestTeam("test-org", "not-created-team", "Not Created Team")
	err := db.db.WithContext(ctx).Create(team).Error
	require.NoError(t, err)

	// Create a mapping with team_created_in_dest = false (default)
	mapping := &models.TeamMapping{
		SourceOrg:           "test-org",
		SourceTeamSlug:      "not-created-team",
		MappingStatus:       "mapped",
		MigrationStatus:     "pending",
		TeamCreatedInDest:   false,
		DestinationOrg:      strPtr("dest-org"),
		DestinationTeamSlug: strPtr("not-created-team"),
	}
	err = db.SaveTeamMapping(ctx, mapping)
	require.NoError(t, err)

	// Query the teams
	teams, total, err := db.ListTeamsWithMappings(ctx, TeamWithMappingFilters{
		Organization: "test-org",
		Limit:        100,
	})
	require.NoError(t, err, "ListTeamsWithMappings should not fail")
	require.Equal(t, int64(1), total)
	require.Len(t, teams, 1)

	// Verify team_created_in_dest is false
	require.False(t, teams[0].TeamCreatedInDest, "team_created_in_dest should be false")

	// sync_status should be 'pending' since migration_status is pending
	require.Equal(t, "pending", teams[0].SyncStatus)
}

func TestListTeamsWithMappings_NoMapping(t *testing.T) {
	// Test that teams without mappings also work correctly (NULL values in COALESCE)
	db := setupTeamMappingsTestDB(t)
	ctx := context.Background()

	// Create a team without any mapping
	team := createTestTeam("orphan-org", "orphan-team", "Orphan Team")
	err := db.db.WithContext(ctx).Create(team).Error
	require.NoError(t, err)

	// Query the teams - this tests COALESCE with NULL mapping values
	teams, total, err := db.ListTeamsWithMappings(ctx, TeamWithMappingFilters{
		Organization: "orphan-org",
		Limit:        100,
	})
	require.NoError(t, err, "ListTeamsWithMappings should handle teams without mappings")
	require.Equal(t, int64(1), total)
	require.Len(t, teams, 1)

	// Verify defaults are applied correctly
	require.Equal(t, "unmapped", teams[0].MappingStatus)
	require.Equal(t, "pending", teams[0].MigrationStatus)
	require.False(t, teams[0].TeamCreatedInDest, "team_created_in_dest should default to false")
	require.Equal(t, "pending", teams[0].SyncStatus)
}

func TestSaveTeamMapping(t *testing.T) {
	db := setupTeamMappingsTestDB(t)
	ctx := context.Background()

	t.Run("create new mapping", func(t *testing.T) {
		mapping := createTestTeamMapping("org1", "team1")
		err := db.SaveTeamMapping(ctx, mapping)
		require.NoError(t, err)
		require.NotZero(t, mapping.ID)
	})

	t.Run("update existing mapping", func(t *testing.T) {
		mapping := createTestTeamMapping("org2", "team2")
		err := db.SaveTeamMapping(ctx, mapping)
		require.NoError(t, err)
		originalID := mapping.ID

		// Update the mapping
		mapping.DestinationOrg = strPtr("new-dest-org")
		err = db.SaveTeamMapping(ctx, mapping)
		require.NoError(t, err)
		require.Equal(t, originalID, mapping.ID, "ID should remain the same on update")
	})
}

func TestGetTeamMapping(t *testing.T) {
	db := setupTeamMappingsTestDB(t)
	ctx := context.Background()

	// Create a mapping
	mapping := createTestTeamMapping("org1", "team1")
	mapping.DestinationOrg = strPtr("dest-org")
	err := db.SaveTeamMapping(ctx, mapping)
	require.NoError(t, err)

	t.Run("get existing mapping", func(t *testing.T) {
		retrieved, err := db.GetTeamMapping(ctx, "org1", "team1")
		require.NoError(t, err)
		require.NotNil(t, retrieved)
		require.Equal(t, "org1", retrieved.SourceOrg)
		require.Equal(t, "team1", retrieved.SourceTeamSlug)
		require.Equal(t, "dest-org", *retrieved.DestinationOrg)
	})

	t.Run("get non-existent mapping", func(t *testing.T) {
		retrieved, err := db.GetTeamMapping(ctx, "nonexistent", "team")
		require.NoError(t, err)
		require.Nil(t, retrieved)
	})
}

// strPtr is a helper to create a pointer to a string
func strPtr(s string) *string {
	return &s
}
