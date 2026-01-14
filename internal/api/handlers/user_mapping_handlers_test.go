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

func TestGenerateGEICSV_FilterByMannequinOrg(t *testing.T) {
	// Use real DB for accurate filtering tests
	cfg := DefaultTestConfig().WithRealDB()
	h, db := setupTestHandlerWithConfig(t, cfg)

	ctx := context.Background()

	// Create user mappings with different mannequin orgs
	realDB, ok := db.(*storage.Database)
	require.True(t, ok, "expected real database for this test")

	// Create mappings for org-alpha
	mapping1 := &models.UserMapping{
		SourceLogin:      "user1",
		DestinationLogin: strPtr("user1-dest"),
		MannequinID:      strPtr("MDEyOk1hbm5lcXVpbjEyMzQ1"),
		MannequinLogin:   strPtr("user1-mona"),
		MannequinOrg:     strPtr("org-alpha"),
		MappingStatus:    string(models.UserMappingStatusMapped),
	}

	mapping2 := &models.UserMapping{
		SourceLogin:      "user2",
		DestinationLogin: strPtr("user2-dest"),
		MannequinID:      strPtr("MDEyOk1hbm5lcXVpbjY3ODkw"),
		MannequinLogin:   strPtr("user2-mona"),
		MannequinOrg:     strPtr("org-alpha"),
		MappingStatus:    string(models.UserMappingStatusMapped),
	}

	// Create mapping for org-beta
	mapping3 := &models.UserMapping{
		SourceLogin:      "user3",
		DestinationLogin: strPtr("user3-dest"),
		MannequinID:      strPtr("MDEyOk1hbm5lcXVpbjExMTEx"),
		MannequinLogin:   strPtr("user3-mona"),
		MannequinOrg:     strPtr("org-beta"),
		MappingStatus:    string(models.UserMappingStatusMapped),
	}

	// Create mapping with no org (nil)
	mapping4 := &models.UserMapping{
		SourceLogin:      "user4",
		DestinationLogin: strPtr("user4-dest"),
		MannequinID:      strPtr("MDEyOk1hbm5lcXVpbjIyMjIy"),
		MannequinLogin:   strPtr("user4-mona"),
		MappingStatus:    string(models.UserMappingStatusMapped),
	}

	// Save all mappings
	for _, m := range []*models.UserMapping{mapping1, mapping2, mapping3, mapping4} {
		err := realDB.SaveUserMapping(ctx, m)
		require.NoError(t, err, "failed to save user mapping")
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
		require.False(t, targetUsers["user4-dest"]) // Should not be included (no org)
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

	t.Run("no org filter returns all mappings", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/api/v1/user-mappings/generate-gei-csv", nil)
		w := httptest.NewRecorder()

		h.GenerateGEICSV(w, req)

		require.Equal(t, http.StatusOK, w.Code)
		require.Contains(t, w.Header().Get("Content-Disposition"), "mannequin-mappings.csv")

		reader := csv.NewReader(strings.NewReader(w.Body.String()))
		records, err := reader.ReadAll()
		require.NoError(t, err)

		// Should have header + 4 rows (all mappings)
		require.Len(t, records, 5, "expected header + 4 data rows for all orgs")
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

func TestGenerateGEICSV_FilenameIncludesOrg(t *testing.T) {
	cfg := DefaultTestConfig().WithRealDB()
	h, _ := setupTestHandlerWithConfig(t, cfg)

	t.Run("filename includes org when filtered", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/api/v1/user-mappings/generate-gei-csv?org=my-company", nil)
		w := httptest.NewRecorder()

		h.GenerateGEICSV(w, req)

		require.Equal(t, http.StatusOK, w.Code)
		require.Equal(t, "attachment; filename=mannequin-mappings-my-company.csv", w.Header().Get("Content-Disposition"))
	})

	t.Run("default filename when no org filter", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/api/v1/user-mappings/generate-gei-csv", nil)
		w := httptest.NewRecorder()

		h.GenerateGEICSV(w, req)

		require.Equal(t, http.StatusOK, w.Code)
		require.Equal(t, "attachment; filename=mannequin-mappings.csv", w.Header().Get("Content-Disposition"))
	})
}

// strPtr is a helper to create string pointers
func strPtr(s string) *string {
	return &s
}
