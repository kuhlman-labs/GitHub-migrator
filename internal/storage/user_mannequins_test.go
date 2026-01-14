package storage

import (
	"context"
	"testing"

	"github.com/kuhlman-labs/github-migrator/internal/config"
	"github.com/kuhlman-labs/github-migrator/internal/models"
	"github.com/stretchr/testify/require"
)

func setupMannequinTestDB(t *testing.T) *Database {
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

func TestSaveUserMannequin(t *testing.T) {
	db := setupMannequinTestDB(t)
	ctx := context.Background()

	t.Run("creates new mannequin", func(t *testing.T) {
		mannequin := &models.UserMannequin{
			SourceLogin:    "test-user",
			MannequinOrg:   "test-org",
			MannequinID:    "MDEyOk1hbm5lcXVpbjEyMzQ1",
			MannequinLogin: strPtr("test-user-mona"),
		}

		err := db.SaveUserMannequin(ctx, mannequin)
		require.NoError(t, err)

		// Verify it was saved
		retrieved, err := db.GetUserMannequin(ctx, "test-user", "test-org")
		require.NoError(t, err)
		require.NotNil(t, retrieved)
		require.Equal(t, "test-user", retrieved.SourceLogin)
		require.Equal(t, "test-org", retrieved.MannequinOrg)
		require.Equal(t, "MDEyOk1hbm5lcXVpbjEyMzQ1", retrieved.MannequinID)
	})

	t.Run("updates existing mannequin", func(t *testing.T) {
		mannequin := &models.UserMannequin{
			SourceLogin:    "update-user",
			MannequinOrg:   "update-org",
			MannequinID:    "original-id",
			MannequinLogin: strPtr("original-mona"),
		}

		err := db.SaveUserMannequin(ctx, mannequin)
		require.NoError(t, err)

		// Update with new values
		mannequin.MannequinID = "updated-id"
		status := string(models.ReclaimStatusCompleted)
		mannequin.ReclaimStatus = &status

		err = db.SaveUserMannequin(ctx, mannequin)
		require.NoError(t, err)

		// Verify update
		retrieved, err := db.GetUserMannequin(ctx, "update-user", "update-org")
		require.NoError(t, err)
		require.NotNil(t, retrieved)
		require.Equal(t, "updated-id", retrieved.MannequinID)
		require.NotNil(t, retrieved.ReclaimStatus)
		require.Equal(t, string(models.ReclaimStatusCompleted), *retrieved.ReclaimStatus)
	})
}

func TestMultiOrgMannequins(t *testing.T) {
	db := setupMannequinTestDB(t)
	ctx := context.Background()

	// Create mannequins for the same user in multiple orgs
	mannequinA := &models.UserMannequin{
		SourceLogin:    "multi-org-user",
		MannequinOrg:   "org-A",
		MannequinID:    "id-for-org-A",
		MannequinLogin: strPtr("multi-org-user-mona-A"),
	}
	mannequinB := &models.UserMannequin{
		SourceLogin:    "multi-org-user",
		MannequinOrg:   "org-B",
		MannequinID:    "id-for-org-B",
		MannequinLogin: strPtr("multi-org-user-mona-B"),
	}

	err := db.SaveUserMannequin(ctx, mannequinA)
	require.NoError(t, err)
	err = db.SaveUserMannequin(ctx, mannequinB)
	require.NoError(t, err)

	t.Run("both mannequins exist independently", func(t *testing.T) {
		// Get mannequin from org-A
		retrievedA, err := db.GetUserMannequin(ctx, "multi-org-user", "org-A")
		require.NoError(t, err)
		require.NotNil(t, retrievedA)
		require.Equal(t, "id-for-org-A", retrievedA.MannequinID)
		require.Equal(t, "multi-org-user-mona-A", *retrievedA.MannequinLogin)

		// Get mannequin from org-B
		retrievedB, err := db.GetUserMannequin(ctx, "multi-org-user", "org-B")
		require.NoError(t, err)
		require.NotNil(t, retrievedB)
		require.Equal(t, "id-for-org-B", retrievedB.MannequinID)
		require.Equal(t, "multi-org-user-mona-B", *retrievedB.MannequinLogin)
	})

	t.Run("GetUserMannequinsBySourceLogin returns all orgs", func(t *testing.T) {
		mannequins, err := db.GetUserMannequinsBySourceLogin(ctx, "multi-org-user")
		require.NoError(t, err)
		require.Len(t, mannequins, 2)

		orgs := make(map[string]bool)
		for _, m := range mannequins {
			orgs[m.MannequinOrg] = true
		}
		require.True(t, orgs["org-A"])
		require.True(t, orgs["org-B"])
	})

	t.Run("updating one org doesn't affect another", func(t *testing.T) {
		// Update org-A mannequin
		status := string(models.ReclaimStatusCompleted)
		mannequinA.ReclaimStatus = &status
		err := db.SaveUserMannequin(ctx, mannequinA)
		require.NoError(t, err)

		// org-B should be unchanged
		retrievedB, err := db.GetUserMannequin(ctx, "multi-org-user", "org-B")
		require.NoError(t, err)
		require.Nil(t, retrievedB.ReclaimStatus)

		// org-A should have the update
		retrievedA, err := db.GetUserMannequin(ctx, "multi-org-user", "org-A")
		require.NoError(t, err)
		require.NotNil(t, retrievedA.ReclaimStatus)
		require.Equal(t, string(models.ReclaimStatusCompleted), *retrievedA.ReclaimStatus)
	})
}

func TestListUserMannequins(t *testing.T) {
	db := setupMannequinTestDB(t)
	ctx := context.Background()

	// Create mannequins
	mannequins := []*models.UserMannequin{
		{SourceLogin: "user1", MannequinOrg: "org-alpha", MannequinID: "id1"},
		{SourceLogin: "user2", MannequinOrg: "org-alpha", MannequinID: "id2"},
		{SourceLogin: "user3", MannequinOrg: "org-beta", MannequinID: "id3"},
	}

	for _, m := range mannequins {
		err := db.SaveUserMannequin(ctx, m)
		require.NoError(t, err)
	}

	t.Run("filter by org", func(t *testing.T) {
		result, total, err := db.ListUserMannequins(ctx, UserMannequinFilters{
			MannequinOrg: "org-alpha",
		})
		require.NoError(t, err)
		require.Equal(t, int64(2), total)
		require.Len(t, result, 2)
	})

	t.Run("filter by reclaim status", func(t *testing.T) {
		// Update one mannequin with reclaim status
		status := string(models.ReclaimStatusCompleted)
		err := db.UpdateMannequinReclaimStatus(ctx, "user1", "org-alpha", status, nil)
		require.NoError(t, err)

		result, total, err := db.ListUserMannequins(ctx, UserMannequinFilters{
			ReclaimStatus: string(models.ReclaimStatusCompleted),
		})
		require.NoError(t, err)
		require.Equal(t, int64(1), total)
		require.Len(t, result, 1)
		require.Equal(t, "user1", result[0].SourceLogin)
	})

	t.Run("no filter returns all", func(t *testing.T) {
		result, total, err := db.ListUserMannequins(ctx, UserMannequinFilters{})
		require.NoError(t, err)
		require.Equal(t, int64(3), total)
		require.Len(t, result, 3)
	})
}

func TestGetMannequinOrgs(t *testing.T) {
	db := setupMannequinTestDB(t)
	ctx := context.Background()

	// Create mannequins in multiple orgs
	mannequins := []*models.UserMannequin{
		{SourceLogin: "user1", MannequinOrg: "alpha-corp", MannequinID: "id1"},
		{SourceLogin: "user2", MannequinOrg: "beta-inc", MannequinID: "id2"},
		{SourceLogin: "user3", MannequinOrg: "alpha-corp", MannequinID: "id3"}, // duplicate org
		{SourceLogin: "user4", MannequinOrg: "gamma-llc", MannequinID: "id4"},
	}

	for _, m := range mannequins {
		err := db.SaveUserMannequin(ctx, m)
		require.NoError(t, err)
	}

	t.Run("returns unique orgs sorted", func(t *testing.T) {
		orgs, err := db.GetMannequinOrgs(ctx)
		require.NoError(t, err)
		require.Len(t, orgs, 3)
		require.Equal(t, []string{"alpha-corp", "beta-inc", "gamma-llc"}, orgs)
	})
}

func TestListMappingsWithMannequins(t *testing.T) {
	db := setupMannequinTestDB(t)
	ctx := context.Background()

	// Create user mappings
	mapping1 := &models.UserMapping{
		SourceLogin:      "user1",
		DestinationLogin: strPtr("user1-dest"),
		MappingStatus:    string(models.UserMappingStatusMapped),
	}
	mapping2 := &models.UserMapping{
		SourceLogin:      "user2",
		DestinationLogin: strPtr("user2-dest"),
		MappingStatus:    string(models.UserMappingStatusMapped),
	}
	mapping3 := &models.UserMapping{
		SourceLogin:   "user3",
		MappingStatus: string(models.UserMappingStatusUnmapped),
	}

	for _, m := range []*models.UserMapping{mapping1, mapping2, mapping3} {
		err := db.SaveUserMapping(ctx, m)
		require.NoError(t, err)
	}

	// Create mannequins (user1 and user2 in org-A, user2 also in org-B)
	mannequins := []*models.UserMannequin{
		{SourceLogin: "user1", MannequinOrg: "org-A", MannequinID: "id1-A", MannequinLogin: strPtr("user1-mona")},
		{SourceLogin: "user2", MannequinOrg: "org-A", MannequinID: "id2-A", MannequinLogin: strPtr("user2-mona")},
		{SourceLogin: "user2", MannequinOrg: "org-B", MannequinID: "id2-B", MannequinLogin: strPtr("user2-mona-B")},
		{SourceLogin: "user3", MannequinOrg: "org-A", MannequinID: "id3-A", MannequinLogin: strPtr("user3-mona")},
	}

	for _, m := range mannequins {
		err := db.SaveUserMannequin(ctx, m)
		require.NoError(t, err)
	}

	t.Run("returns joined data for org-A with mapped status", func(t *testing.T) {
		results, err := db.ListMappingsWithMannequins(ctx, "org-A", string(models.UserMappingStatusMapped))
		require.NoError(t, err)
		require.Len(t, results, 2)

		// Check we have the right users
		logins := make(map[string]bool)
		for _, r := range results {
			logins[r.SourceLogin] = true
			// Verify mannequin data comes from org-A
			require.Equal(t, "org-A", r.MannequinOrg)
		}
		require.True(t, logins["user1"])
		require.True(t, logins["user2"])
	})

	t.Run("returns joined data for org-B", func(t *testing.T) {
		results, err := db.ListMappingsWithMannequins(ctx, "org-B", string(models.UserMappingStatusMapped))
		require.NoError(t, err)
		require.Len(t, results, 1)
		require.Equal(t, "user2", results[0].SourceLogin)
		require.Equal(t, "id2-B", results[0].MannequinID)
		require.Equal(t, "user2-mona-B", *results[0].MannequinLogin)
	})

	t.Run("filters by status", func(t *testing.T) {
		results, err := db.ListMappingsWithMannequins(ctx, "org-A", string(models.UserMappingStatusUnmapped))
		require.NoError(t, err)
		require.Len(t, results, 1)
		require.Equal(t, "user3", results[0].SourceLogin)
	})

	t.Run("empty status returns all statuses", func(t *testing.T) {
		results, err := db.ListMappingsWithMannequins(ctx, "org-A", "")
		require.NoError(t, err)
		require.Len(t, results, 3) // user1, user2, user3 all have mannequins in org-A
	})
}

func TestDeleteUserMannequin(t *testing.T) {
	db := setupMannequinTestDB(t)
	ctx := context.Background()

	// Create mannequins
	mannequinA := &models.UserMannequin{
		SourceLogin:  "delete-user",
		MannequinOrg: "org-A",
		MannequinID:  "id-A",
	}
	mannequinB := &models.UserMannequin{
		SourceLogin:  "delete-user",
		MannequinOrg: "org-B",
		MannequinID:  "id-B",
	}

	err := db.SaveUserMannequin(ctx, mannequinA)
	require.NoError(t, err)
	err = db.SaveUserMannequin(ctx, mannequinB)
	require.NoError(t, err)

	t.Run("deletes only specified org", func(t *testing.T) {
		err := db.DeleteUserMannequin(ctx, "delete-user", "org-A")
		require.NoError(t, err)

		// org-A should be gone
		retrievedA, err := db.GetUserMannequin(ctx, "delete-user", "org-A")
		require.NoError(t, err)
		require.Nil(t, retrievedA)

		// org-B should still exist
		retrievedB, err := db.GetUserMannequin(ctx, "delete-user", "org-B")
		require.NoError(t, err)
		require.NotNil(t, retrievedB)
	})
}
