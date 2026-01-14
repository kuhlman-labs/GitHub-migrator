package storage

import (
	"context"
	"testing"

	"github.com/kuhlman-labs/github-migrator/internal/config"
	"github.com/kuhlman-labs/github-migrator/internal/models"
	"github.com/stretchr/testify/require"
)

func setupUserMappingsTestDB(t *testing.T) *Database {
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

// createTestUserMapping creates a minimal UserMapping for testing
func createTestUserMapping(sourceLogin string) *models.UserMapping {
	return &models.UserMapping{
		SourceLogin:   sourceLogin,
		MappingStatus: string(models.UserMappingStatusUnmapped),
	}
}

// TestListUserMappings_MannequinOrgFilter tests the deprecated MannequinOrg filter on user_mappings table.
// NOTE: Mannequin data should now be stored in the user_mannequins table for multi-org support.
// This test remains for backward compatibility with existing data.
func TestListUserMappings_MannequinOrgFilter(t *testing.T) {
	db := setupUserMappingsTestDB(t)
	ctx := context.Background()

	// Create user mappings with different mannequin orgs (deprecated pattern)
	mapping1 := createTestUserMapping("user1")
	mapping1.MannequinID = strPtr("MDEyOk1hbm5lcXVpbjEyMzQ1")
	mapping1.MannequinLogin = strPtr("user1-mona")
	mapping1.MannequinOrg = strPtr("org-alpha")
	mapping1.MappingStatus = string(models.UserMappingStatusMapped)
	mapping1.DestinationLogin = strPtr("user1-dest")

	mapping2 := createTestUserMapping("user2")
	mapping2.MannequinID = strPtr("MDEyOk1hbm5lcXVpbjY3ODkw")
	mapping2.MannequinLogin = strPtr("user2-mona")
	mapping2.MannequinOrg = strPtr("org-alpha")
	mapping2.MappingStatus = string(models.UserMappingStatusMapped)
	mapping2.DestinationLogin = strPtr("user2-dest")

	mapping3 := createTestUserMapping("user3")
	mapping3.MannequinID = strPtr("MDEyOk1hbm5lcXVpbjExMTEx")
	mapping3.MannequinLogin = strPtr("user3-mona")
	mapping3.MannequinOrg = strPtr("org-beta")
	mapping3.MappingStatus = string(models.UserMappingStatusMapped)
	mapping3.DestinationLogin = strPtr("user3-dest")

	// mapping4 has no mannequin org (nil)
	mapping4 := createTestUserMapping("user4")
	mapping4.MannequinID = strPtr("MDEyOk1hbm5lcXVpbjIyMjIy")
	mapping4.MannequinLogin = strPtr("user4-mona")
	mapping4.MappingStatus = string(models.UserMappingStatusMapped)
	mapping4.DestinationLogin = strPtr("user4-dest")

	// Save all mappings
	for _, m := range []*models.UserMapping{mapping1, mapping2, mapping3, mapping4} {
		err := db.SaveUserMapping(ctx, m)
		require.NoError(t, err, "failed to save user mapping")
	}

	t.Run("filter by mannequin_org returns only matching mappings", func(t *testing.T) {
		mappings, total, err := db.ListUserMappings(ctx, UserMappingFilters{
			MannequinOrg: "org-alpha",
			Limit:        100,
		})
		require.NoError(t, err)
		require.Equal(t, int64(2), total)
		require.Len(t, mappings, 2)

		// Verify the correct mappings were returned
		logins := make(map[string]bool)
		for _, m := range mappings {
			logins[m.SourceLogin] = true
		}
		require.True(t, logins["user1"])
		require.True(t, logins["user2"])
	})

	t.Run("filter by different mannequin_org", func(t *testing.T) {
		mappings, total, err := db.ListUserMappings(ctx, UserMappingFilters{
			MannequinOrg: "org-beta",
			Limit:        100,
		})
		require.NoError(t, err)
		require.Equal(t, int64(1), total)
		require.Len(t, mappings, 1)
		require.Equal(t, "user3", mappings[0].SourceLogin)
	})

	t.Run("no mannequin_org filter returns all mappings", func(t *testing.T) {
		mappings, total, err := db.ListUserMappings(ctx, UserMappingFilters{
			Limit: 100,
		})
		require.NoError(t, err)
		require.Equal(t, int64(4), total)
		require.Len(t, mappings, 4)
	})

	t.Run("filter by non-existent mannequin_org returns empty", func(t *testing.T) {
		mappings, total, err := db.ListUserMappings(ctx, UserMappingFilters{
			MannequinOrg: "org-does-not-exist",
			Limit:        100,
		})
		require.NoError(t, err)
		require.Equal(t, int64(0), total)
		require.Len(t, mappings, 0)
	})

	t.Run("combined filters with mannequin_org", func(t *testing.T) {
		// Filter by both status and mannequin_org
		mappings, total, err := db.ListUserMappings(ctx, UserMappingFilters{
			Status:       string(models.UserMappingStatusMapped),
			MannequinOrg: "org-alpha",
			Limit:        100,
		})
		require.NoError(t, err)
		require.Equal(t, int64(2), total)
		require.Len(t, mappings, 2)
	})

	t.Run("mannequin_org filter with HasMannequin filter", func(t *testing.T) {
		hasMannequin := true
		mappings, total, err := db.ListUserMappings(ctx, UserMappingFilters{
			MannequinOrg: "org-alpha",
			HasMannequin: &hasMannequin,
			Limit:        100,
		})
		require.NoError(t, err)
		require.Equal(t, int64(2), total)
		require.Len(t, mappings, 2)
	})
}

func TestSaveUserMapping_WithMannequinOrg(t *testing.T) {
	db := setupUserMappingsTestDB(t)
	ctx := context.Background()

	t.Run("save mapping with mannequin_org", func(t *testing.T) {
		mapping := createTestUserMapping("test-user")
		mapping.MannequinID = strPtr("MDEyOk1hbm5lcXVpbjk5OTk5")
		mapping.MannequinLogin = strPtr("test-user-mona")
		mapping.MannequinOrg = strPtr("test-org")

		err := db.SaveUserMapping(ctx, mapping)
		require.NoError(t, err)

		// Retrieve and verify
		retrieved, err := db.GetUserMappingBySourceLogin(ctx, "test-user")
		require.NoError(t, err)
		require.NotNil(t, retrieved)
		require.NotNil(t, retrieved.MannequinOrg)
		require.Equal(t, "test-org", *retrieved.MannequinOrg)
	})

	t.Run("update mapping with new mannequin_org", func(t *testing.T) {
		// First create a mapping
		mapping := createTestUserMapping("update-test-user")
		mapping.MannequinID = strPtr("MDEyOk1hbm5lcXVpbjg4ODg4")
		mapping.MannequinLogin = strPtr("update-test-mona")
		mapping.MannequinOrg = strPtr("old-org")

		err := db.SaveUserMapping(ctx, mapping)
		require.NoError(t, err)

		// Update with new org
		mapping.MannequinOrg = strPtr("new-org")
		err = db.SaveUserMapping(ctx, mapping)
		require.NoError(t, err)

		// Verify update
		retrieved, err := db.GetUserMappingBySourceLogin(ctx, "update-test-user")
		require.NoError(t, err)
		require.NotNil(t, retrieved)
		require.NotNil(t, retrieved.MannequinOrg)
		require.Equal(t, "new-org", *retrieved.MannequinOrg)
	})
}
