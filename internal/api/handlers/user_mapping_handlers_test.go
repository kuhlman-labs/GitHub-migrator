package handlers

import (
	"context"
	"encoding/csv"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/kuhlman-labs/github-migrator/internal/models"
	"github.com/kuhlman-labs/github-migrator/internal/storage"
	"github.com/stretchr/testify/require"
)

func TestGenerateGEICSV_RequiresOrg(t *testing.T) {
	cfg := DefaultTestConfig().WithRealDB()
	h, _ := setupTestHandlerWithConfig(t, cfg)

	t.Run("returns error when org is missing", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/api/v1/user-mappings/generate-gei-csv", nil)
		w := httptest.NewRecorder()

		h.GenerateGEICSV(w, req)

		require.Equal(t, http.StatusBadRequest, w.Code)
		require.Contains(t, w.Body.String(), "org parameter is required")
	})
}

func TestGenerateGEICSV_FilterByMannequinOrg(t *testing.T) {
	// Use real DB for accurate filtering tests
	cfg := DefaultTestConfig().WithRealDB()
	h, db := setupTestHandlerWithConfig(t, cfg)

	ctx := context.Background()

	realDB, ok := db.(*storage.Database)
	require.True(t, ok, "expected real database for this test")

	// Create user mappings (without mannequin fields - those are per-org now)
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
		SourceLogin:      "user3",
		DestinationLogin: strPtr("user3-dest"),
		MappingStatus:    string(models.UserMappingStatusMapped),
	}

	// Save all mappings
	for _, m := range []*models.UserMapping{mapping1, mapping2, mapping3} {
		err := realDB.SaveUserMapping(ctx, m)
		require.NoError(t, err, "failed to save user mapping")
	}

	// Create mannequins for org-alpha (user1 and user2)
	mannequin1 := &models.UserMannequin{
		SourceLogin:    "user1",
		MannequinOrg:   "org-alpha",
		MannequinID:    "MDEyOk1hbm5lcXVpbjEyMzQ1",
		MannequinLogin: strPtr("user1-mona"),
	}
	mannequin2 := &models.UserMannequin{
		SourceLogin:    "user2",
		MannequinOrg:   "org-alpha",
		MannequinID:    "MDEyOk1hbm5lcXVpbjY3ODkw",
		MannequinLogin: strPtr("user2-mona"),
	}

	// Create mannequin for org-beta (user3)
	mannequin3 := &models.UserMannequin{
		SourceLogin:    "user3",
		MannequinOrg:   "org-beta",
		MannequinID:    "MDEyOk1hbm5lcXVpbjExMTEx",
		MannequinLogin: strPtr("user3-mona"),
	}

	// Save mannequins
	for _, m := range []*models.UserMannequin{mannequin1, mannequin2, mannequin3} {
		err := realDB.SaveUserMannequin(ctx, m)
		require.NoError(t, err, "failed to save user mannequin")
	}

	t.Run("filter by org returns only matching mappings", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/api/v1/user-mappings/generate-gei-csv?org=org-alpha", nil)
		w := httptest.NewRecorder()

		h.GenerateGEICSV(w, req)

		require.Equal(t, http.StatusOK, w.Code)
		require.Equal(t, "text/csv", w.Header().Get("Content-Type"))
		require.Contains(t, w.Header().Get("Content-Disposition"), "mannequin-mappings-org-alpha.csv")

		// Parse CSV and verify contents
		reader := csv.NewReader(strings.NewReader(w.Body.String()))
		records, err := reader.ReadAll()
		require.NoError(t, err)

		// Should have header + 2 rows for org-alpha
		require.Len(t, records, 3, "expected header + 2 data rows for org-alpha")

		// Verify header
		require.Equal(t, []string{"mannequin-user", "mannequin-id", "target-user"}, records[0])

		// Collect target users
		targetUsers := make(map[string]bool)
		for i := 1; i < len(records); i++ {
			targetUsers[records[i][2]] = true
		}
		require.True(t, targetUsers["user1-dest"])
		require.True(t, targetUsers["user2-dest"])
		require.False(t, targetUsers["user3-dest"]) // Should not be included (org-beta)
	})

	t.Run("filter by different org", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/api/v1/user-mappings/generate-gei-csv?org=org-beta", nil)
		w := httptest.NewRecorder()

		h.GenerateGEICSV(w, req)

		require.Equal(t, http.StatusOK, w.Code)
		require.Contains(t, w.Header().Get("Content-Disposition"), "mannequin-mappings-org-beta.csv")

		reader := csv.NewReader(strings.NewReader(w.Body.String()))
		records, err := reader.ReadAll()
		require.NoError(t, err)

		// Should have header + 1 row for org-beta
		require.Len(t, records, 2, "expected header + 1 data row for org-beta")
		require.Equal(t, "user3-dest", records[1][2])
	})

	t.Run("non-existent org returns empty CSV", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/api/v1/user-mappings/generate-gei-csv?org=org-does-not-exist", nil)
		w := httptest.NewRecorder()

		h.GenerateGEICSV(w, req)

		require.Equal(t, http.StatusOK, w.Code)

		reader := csv.NewReader(strings.NewReader(w.Body.String()))
		records, err := reader.ReadAll()
		require.NoError(t, err)

		// Should only have header
		require.Len(t, records, 1, "expected only header for non-existent org")
	})
}

func TestGenerateGEICSV_MultiOrgMannequins(t *testing.T) {
	// Test that the same user can have mannequins in multiple orgs
	cfg := DefaultTestConfig().WithRealDB()
	h, db := setupTestHandlerWithConfig(t, cfg)

	ctx := context.Background()

	realDB, ok := db.(*storage.Database)
	require.True(t, ok, "expected real database for this test")

	// Create a single user mapping
	mapping := &models.UserMapping{
		SourceLogin:      "multi-org-user",
		DestinationLogin: strPtr("multi-org-user-dest"),
		MappingStatus:    string(models.UserMappingStatusMapped),
	}
	err := realDB.SaveUserMapping(ctx, mapping)
	require.NoError(t, err)

	// Create mannequins in multiple orgs for the same user
	mannequinA := &models.UserMannequin{
		SourceLogin:    "multi-org-user",
		MannequinOrg:   "org-A",
		MannequinID:    "MDEyOk1hbm5lcXVpbk9yZ0E=",
		MannequinLogin: strPtr("multi-org-user-mona-A"),
	}
	mannequinB := &models.UserMannequin{
		SourceLogin:    "multi-org-user",
		MannequinOrg:   "org-B",
		MannequinID:    "MDEyOk1hbm5lcXVpbk9yZ0I=",
		MannequinLogin: strPtr("multi-org-user-mona-B"),
	}

	// Save both mannequins
	err = realDB.SaveUserMannequin(ctx, mannequinA)
	require.NoError(t, err)
	err = realDB.SaveUserMannequin(ctx, mannequinB)
	require.NoError(t, err)

	t.Run("CSV for org-A has correct mannequin ID", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/api/v1/user-mappings/generate-gei-csv?org=org-A", nil)
		w := httptest.NewRecorder()

		h.GenerateGEICSV(w, req)

		require.Equal(t, http.StatusOK, w.Code)

		reader := csv.NewReader(strings.NewReader(w.Body.String()))
		records, err := reader.ReadAll()
		require.NoError(t, err)

		require.Len(t, records, 2, "expected header + 1 data row")
		require.Equal(t, "multi-org-user-mona-A", records[1][0])    // mannequin-user
		require.Equal(t, "MDEyOk1hbm5lcXVpbk9yZ0E=", records[1][1]) // mannequin-id
		require.Equal(t, "multi-org-user-dest", records[1][2])      // target-user
	})

	t.Run("CSV for org-B has correct mannequin ID", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/api/v1/user-mappings/generate-gei-csv?org=org-B", nil)
		w := httptest.NewRecorder()

		h.GenerateGEICSV(w, req)

		require.Equal(t, http.StatusOK, w.Code)

		reader := csv.NewReader(strings.NewReader(w.Body.String()))
		records, err := reader.ReadAll()
		require.NoError(t, err)

		require.Len(t, records, 2, "expected header + 1 data row")
		require.Equal(t, "multi-org-user-mona-B", records[1][0])    // mannequin-user
		require.Equal(t, "MDEyOk1hbm5lcXVpbk9yZ0I=", records[1][1]) // mannequin-id
		require.Equal(t, "multi-org-user-dest", records[1][2])      // target-user
	})
}

func TestGetMannequinOrgs(t *testing.T) {
	cfg := DefaultTestConfig().WithRealDB()
	h, db := setupTestHandlerWithConfig(t, cfg)

	ctx := context.Background()

	realDB, ok := db.(*storage.Database)
	require.True(t, ok, "expected real database for this test")

	// Create mannequins in multiple orgs
	mannequins := []*models.UserMannequin{
		{SourceLogin: "user1", MannequinOrg: "alpha-corp", MannequinID: "id1"},
		{SourceLogin: "user2", MannequinOrg: "beta-inc", MannequinID: "id2"},
		{SourceLogin: "user3", MannequinOrg: "alpha-corp", MannequinID: "id3"}, // duplicate org
	}

	for _, m := range mannequins {
		err := realDB.SaveUserMannequin(ctx, m)
		require.NoError(t, err)
	}

	t.Run("returns unique org list", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/api/v1/user-mappings/mannequin-orgs", nil)
		w := httptest.NewRecorder()

		h.GetMannequinOrgs(w, req)

		require.Equal(t, http.StatusOK, w.Code)
		require.Contains(t, w.Body.String(), "alpha-corp")
		require.Contains(t, w.Body.String(), "beta-inc")
	})
}

// strPtr is a helper to create string pointers
func strPtr(s string) *string {
	return &s
}
