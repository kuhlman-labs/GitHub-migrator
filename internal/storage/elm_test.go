package storage

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/kuhlman-labs/github-migrator/internal/models"
)

const elmStatusCutoverReady = "cutover_ready"

// seedELMRepo saves a repository with the given status and source, and attaches an
// elm_migrations row to it. Fixtures are built by construction (direct writes),
// never by calling the control under test.
func seedELMRepo(t *testing.T, db *Database, fullName, status string, sourceID *int64) *models.Repository {
	t.Helper()
	ctx := context.Background()

	repo := createTestRepoWithStatus(fullName, status)
	repo.SourceID = sourceID
	if err := db.SaveRepository(ctx, repo); err != nil {
		t.Fatalf("failed to save repository %s: %v", fullName, err)
	}

	saved, err := db.GetRepository(ctx, fullName)
	if err != nil || saved == nil {
		t.Fatalf("failed to reload repository %s: %v", fullName, err)
	}
	return saved
}

func seedELMMigration(t *testing.T, db *Database, repoID int64, elmID string) {
	t.Helper()
	rec := &models.ELMMigration{
		RepositoryID:   repoID,
		ELMMigrationID: elmID,
		ELMStatus:      "backfilling",
	}
	if err := db.UpsertELMMigration(context.Background(), rec); err != nil {
		t.Fatalf("failed to seed elm migration %s: %v", elmID, err)
	}
}

func createTestSourceID(t *testing.T, db *Database, name string) int64 {
	t.Helper()
	src := createTestSource(name, models.SourceConfigTypeGitHub)
	if err := db.CreateSource(context.Background(), src); err != nil {
		t.Fatalf("failed to create source %s: %v", name, err)
	}
	return src.ID
}

// TestUpdateRepositoryMigrationRoute_RoundTrip proves the route persists to the
// repositories table and that clearing it returns the repository to the GEI default.
func TestUpdateRepositoryMigrationRoute_RoundTrip(t *testing.T) {
	db := setupTestDB(t)
	defer func() { _ = db.Close() }()
	ctx := context.Background()

	repo := createTestRepository("org/elm-repo")
	if err := db.SaveRepository(ctx, repo); err != nil {
		t.Fatalf("SaveRepository() error = %v", err)
	}

	// A freshly discovered repository has no route and reads as GEI.
	fresh, err := db.GetRepository(ctx, "org/elm-repo")
	if err != nil {
		t.Fatalf("GetRepository() error = %v", err)
	}
	if fresh.MigrationRoute != nil {
		t.Errorf("expected nil route on a fresh repository, got %q", *fresh.MigrationRoute)
	}
	if got := fresh.GetMigrationRoute(); got != string(models.MigrationRouteGEI) {
		t.Errorf("fresh repository route = %q, want %q", got, models.MigrationRouteGEI)
	}

	elm := string(models.MigrationRouteELM)
	if err := db.UpdateRepositoryMigrationRoute(ctx, "org/elm-repo", &elm); err != nil {
		t.Fatalf("UpdateRepositoryMigrationRoute(elm) error = %v", err)
	}

	routed, err := db.GetRepository(ctx, "org/elm-repo")
	if err != nil {
		t.Fatalf("GetRepository() error = %v", err)
	}
	if got := routed.GetMigrationRoute(); got != string(models.MigrationRouteELM) {
		t.Errorf("after write, route = %q, want %q", got, models.MigrationRouteELM)
	}
	if !routed.IsELMRouted() {
		t.Error("expected IsELMRouted() to be true after writing the elm route")
	}

	// Clearing returns the repository to the GEI default.
	if err := db.UpdateRepositoryMigrationRoute(ctx, "org/elm-repo", nil); err != nil {
		t.Fatalf("UpdateRepositoryMigrationRoute(nil) error = %v", err)
	}
	cleared, err := db.GetRepository(ctx, "org/elm-repo")
	if err != nil {
		t.Fatalf("GetRepository() error = %v", err)
	}
	if cleared.MigrationRoute != nil {
		t.Errorf("expected route to be cleared to NULL, got %q", *cleared.MigrationRoute)
	}
	if got := cleared.GetMigrationRoute(); got != string(models.MigrationRouteGEI) {
		t.Errorf("after clear, route = %q, want %q", got, models.MigrationRouteGEI)
	}
	if cleared.IsELMRouted() {
		t.Error("expected IsELMRouted() to be false after clearing the route")
	}
}

// TestUpdateRepositoryMigrationRoute_RejectsUnknownRoute asserts COMMITTED STATE,
// not just the error: a rejected write must leave the stored route untouched.
func TestUpdateRepositoryMigrationRoute_RejectsUnknownRoute(t *testing.T) {
	db := setupTestDB(t)
	defer func() { _ = db.Close() }()
	ctx := context.Background()

	repo := createTestRepository("org/guarded")
	if err := db.SaveRepository(ctx, repo); err != nil {
		t.Fatalf("SaveRepository() error = %v", err)
	}

	// Seed the elm route by construction (a direct column write), not via the
	// method under test, so the assertion below lands on the guard's behavior.
	if err := db.DB().Model(&models.Repository{}).
		Where("full_name = ?", "org/guarded").
		Update("migration_route", string(models.MigrationRouteELM)).Error; err != nil {
		t.Fatalf("failed to seed route: %v", err)
	}

	for _, bad := range []string{"gei-plus", "ELM", "", "github"} {
		t.Run("rejects_"+bad, func(t *testing.T) {
			route := bad
			err := db.UpdateRepositoryMigrationRoute(ctx, "org/guarded", &route)
			if err == nil {
				t.Fatalf("expected an error for route %q", bad)
			}

			reloaded, getErr := db.GetRepository(ctx, "org/guarded")
			if getErr != nil {
				t.Fatalf("GetRepository() error = %v", getErr)
			}
			if got := reloaded.GetMigrationRoute(); got != string(models.MigrationRouteELM) {
				t.Errorf("stored route changed to %q after a rejected write; want it left at %q",
					got, models.MigrationRouteELM)
			}
		})
	}
}

// TestUpdateRepositoryMigrationRoute_RepositoryNotFound covers the missing-row branch.
func TestUpdateRepositoryMigrationRoute_RepositoryNotFound(t *testing.T) {
	db := setupTestDB(t)
	defer func() { _ = db.Close() }()

	elm := string(models.MigrationRouteELM)
	if err := db.UpdateRepositoryMigrationRoute(context.Background(), "org/does-not-exist", &elm); err == nil {
		t.Error("expected an error when the repository does not exist")
	}
}

// TestUpdateRepository_RoundTripsMigrationRoute proves the new column survives the
// existing CRUD path, which uses Select("*") and therefore needs no edit.
func TestUpdateRepository_RoundTripsMigrationRoute(t *testing.T) {
	db := setupTestDB(t)
	defer func() { _ = db.Close() }()
	ctx := context.Background()

	repo := createTestRepository("org/crud-route")
	if err := db.SaveRepository(ctx, repo); err != nil {
		t.Fatalf("SaveRepository() error = %v", err)
	}

	loaded, err := db.GetRepository(ctx, "org/crud-route")
	if err != nil {
		t.Fatalf("GetRepository() error = %v", err)
	}

	elm := string(models.MigrationRouteELM)
	loaded.MigrationRoute = &elm
	if err := db.UpdateRepository(ctx, loaded); err != nil {
		t.Fatalf("UpdateRepository() error = %v", err)
	}

	reloaded, err := db.GetRepository(ctx, "org/crud-route")
	if err != nil {
		t.Fatalf("GetRepository() error = %v", err)
	}
	if got := reloaded.GetMigrationRoute(); got != string(models.MigrationRouteELM) {
		t.Errorf("route after UpdateRepository = %q, want %q", got, models.MigrationRouteELM)
	}
}

// TestUpsertELMMigration_Idempotent asserts repeated polls update the row in place.
func TestUpsertELMMigration_Idempotent(t *testing.T) {
	db := setupTestDB(t)
	defer func() { _ = db.Close() }()
	ctx := context.Background()

	repo := seedELMRepo(t, db, "org/live", string(models.StatusSyncing), nil)

	first := &models.ELMMigration{
		RepositoryID:   repo.ID,
		ELMMigrationID: "elm-1",
		ELMStatus:      "backfilling",
		ELMPhase:       "git",
	}
	if err := db.UpsertELMMigration(ctx, first); err != nil {
		t.Fatalf("first UpsertELMMigration() error = %v", err)
	}

	polled := time.Now().UTC().Truncate(time.Second)
	progress := 42
	second := &models.ELMMigration{
		RepositoryID:    repo.ID,
		ELMMigrationID:  "elm-1",
		ELMStatus:       elmStatusCutoverReady,
		ELMPhase:        "metadata",
		CutoverReady:    true,
		ProgressPercent: &progress,
		LastPolledAt:    &polled,
	}
	if err := db.UpsertELMMigration(ctx, second); err != nil {
		t.Fatalf("second UpsertELMMigration() error = %v", err)
	}

	var count int64
	if err := db.DB().Model(&models.ELMMigration{}).Count(&count).Error; err != nil {
		t.Fatalf("count error = %v", err)
	}
	if count != 1 {
		t.Errorf("expected 1 elm_migrations row after two upserts, got %d", count)
	}

	rec, err := db.GetELMMigration(ctx, repo.ID)
	if err != nil {
		t.Fatalf("GetELMMigration() error = %v", err)
	}
	if rec == nil {
		t.Fatal("expected an ELM migration record")
	}
	if rec.ELMStatus != elmStatusCutoverReady || !rec.CutoverReady {
		t.Errorf("expected the second upsert to win, got status=%q cutover_ready=%v", rec.ELMStatus, rec.CutoverReady)
	}
	if rec.ProgressPercent == nil || *rec.ProgressPercent != 42 {
		t.Errorf("expected progress_percent=42, got %v", rec.ProgressPercent)
	}
	if rec.LastPolledAt == nil {
		t.Error("expected last_polled_at to be persisted")
	}

	byELMID, err := db.GetELMMigrationByELMID(ctx, "elm-1")
	if err != nil {
		t.Fatalf("GetELMMigrationByELMID() error = %v", err)
	}
	if byELMID == nil || byELMID.RepositoryID != repo.ID {
		t.Errorf("GetELMMigrationByELMID() = %v, want the record for repo %d", byELMID, repo.ID)
	}
}

// TestUpsertELMMigration_RejectsIncompleteRecord covers the validation branches.
func TestUpsertELMMigration_RejectsIncompleteRecord(t *testing.T) {
	db := setupTestDB(t)
	defer func() { _ = db.Close() }()
	ctx := context.Background()

	if err := db.UpsertELMMigration(ctx, nil); err == nil {
		t.Error("expected an error for a nil record")
	}
	if err := db.UpsertELMMigration(ctx, &models.ELMMigration{ELMMigrationID: "elm-x"}); err == nil {
		t.Error("expected an error for a record with no repository id")
	}
	if err := db.UpsertELMMigration(ctx, &models.ELMMigration{RepositoryID: 1}); err == nil {
		t.Error("expected an error for a record with no ELM migration id")
	}

	var count int64
	if err := db.DB().Model(&models.ELMMigration{}).Count(&count).Error; err != nil {
		t.Fatalf("count error = %v", err)
	}
	if count != 0 {
		t.Errorf("expected no rows written by rejected upserts, got %d", count)
	}
}

// TestGetELMMigration_MissingReturnsNil covers the absent-record branch.
func TestGetELMMigration_MissingReturnsNil(t *testing.T) {
	db := setupTestDB(t)
	defer func() { _ = db.Close() }()
	ctx := context.Background()

	rec, err := db.GetELMMigration(ctx, 12345)
	if err != nil {
		t.Fatalf("GetELMMigration() error = %v", err)
	}
	if rec != nil {
		t.Errorf("expected nil for a repository with no ELM record, got %+v", rec)
	}

	byID, err := db.GetELMMigrationByELMID(ctx, "no-such-elm-id")
	if err != nil {
		t.Fatalf("GetELMMigrationByELMID() error = %v", err)
	}
	if byID != nil {
		t.Errorf("expected nil for an unknown ELM id, got %+v", byID)
	}
}

// TestDeleteELMMigration removes the record and tolerates a missing one.
func TestDeleteELMMigration(t *testing.T) {
	db := setupTestDB(t)
	defer func() { _ = db.Close() }()
	ctx := context.Background()

	repo := seedELMRepo(t, db, "org/delete-me", string(models.StatusSyncing), nil)
	seedELMMigration(t, db, repo.ID, "elm-del")

	if err := db.DeleteELMMigration(ctx, repo.ID); err != nil {
		t.Fatalf("DeleteELMMigration() error = %v", err)
	}
	rec, err := db.GetELMMigration(ctx, repo.ID)
	if err != nil {
		t.Fatalf("GetELMMigration() error = %v", err)
	}
	if rec != nil {
		t.Error("expected the ELM record to be gone")
	}

	if err := db.DeleteELMMigration(ctx, repo.ID); err != nil {
		t.Errorf("deleting a missing record should not error, got %v", err)
	}
}

// TestListELMMigrationsInFlight_ExcludesTerminalStatuses asserts the in-flight
// predicate: only repositories in an ELM lifecycle status are counted.
func TestListELMMigrationsInFlight_ExcludesTerminalStatuses(t *testing.T) {
	db := setupTestDB(t)
	defer func() { _ = db.Close() }()
	ctx := context.Background()

	inFlight := map[string]string{
		"org/syncing":   string(models.StatusSyncing),
		"org/awaiting":  string(models.StatusAwaitingCutover),
		"org/cuttingov": string(models.StatusCuttingOver),
	}
	terminal := map[string]string{
		"org/done":   string(models.StatusMigrationComplete),
		"org/failed": string(models.StatusMigrationFailed),
		"org/queued": string(models.StatusPending),
	}

	for name, status := range inFlight {
		repo := seedELMRepo(t, db, name, status, nil)
		seedELMMigration(t, db, repo.ID, "elm-live-"+status)
	}
	for name, status := range terminal {
		repo := seedELMRepo(t, db, name, status, nil)
		seedELMMigration(t, db, repo.ID, "elm-done-"+status)
	}

	records, err := db.ListELMMigrationsInFlight(ctx)
	if err != nil {
		t.Fatalf("ListELMMigrationsInFlight() error = %v", err)
	}
	if len(records) != len(inFlight) {
		t.Errorf("expected %d in-flight records, got %d", len(inFlight), len(records))
	}

	counts, err := db.CountELMMigrationsInFlight(ctx)
	if err != nil {
		t.Fatalf("CountELMMigrationsInFlight() error = %v", err)
	}
	if counts.Global != len(inFlight) {
		t.Errorf("expected global count %d, got %d", len(inFlight), counts.Global)
	}
}

// TestCountELMMigrationsInFlight_GroupsBySourceAndCountsGlobally pins BOTH ceiling
// group keys against a fixture with TWO DISTINCT source_id values plus a NULL one,
// so a per-source count cannot pass by accidentally counting globally.
func TestCountELMMigrationsInFlight_GroupsBySourceAndCountsGlobally(t *testing.T) {
	db := setupTestDB(t)
	defer func() { _ = db.Close() }()
	ctx := context.Background()

	sourceA := createTestSourceID(t, db, "Source A")
	sourceB := createTestSourceID(t, db, "Source B")
	if sourceA == sourceB {
		t.Fatal("fixture requires two distinct source ids")
	}

	// 3 in flight on source A, 1 on source B, 2 with a NULL source_id.
	for i := 0; i < 3; i++ {
		repo := seedELMRepo(t, db, fmtName("a", i), string(models.StatusSyncing), &sourceA)
		seedELMMigration(t, db, repo.ID, fmtName("elm-a", i))
	}
	repoB := seedELMRepo(t, db, fmtName("b", 0), string(models.StatusAwaitingCutover), &sourceB)
	seedELMMigration(t, db, repoB.ID, fmtName("elm-b", 0))
	for i := 0; i < 2; i++ {
		repo := seedELMRepo(t, db, fmtName("null", i), string(models.StatusCuttingOver), nil)
		seedELMMigration(t, db, repo.ID, fmtName("elm-null", i))
	}
	// Not in flight: must not be counted anywhere.
	notInFlight := seedELMRepo(t, db, fmtName("a-done", 0), string(models.StatusMigrationComplete), &sourceA)
	seedELMMigration(t, db, notInFlight.ID, "elm-a-done")

	counts, err := db.CountELMMigrationsInFlight(ctx)
	if err != nil {
		t.Fatalf("CountELMMigrationsInFlight() error = %v", err)
	}

	if got := counts.BySourceID[sourceA]; got != 3 {
		t.Errorf("source A in-flight count = %d, want 3", got)
	}
	if got := counts.BySourceID[sourceB]; got != 1 {
		t.Errorf("source B in-flight count = %d, want 1", got)
	}
	if got := counts.BySourceID[ELMUnknownSourceBucket]; got != 2 {
		t.Errorf("NULL source_id bucket count = %d, want 2 (must not be dropped)", got)
	}
	if counts.Global != 6 {
		t.Errorf("global in-flight count = %d, want 6", counts.Global)
	}

	// CountForSource maps a NULL source_id onto the same well-defined bucket.
	if got := counts.CountForSource(&sourceA); got != 3 {
		t.Errorf("CountForSource(sourceA) = %d, want 3", got)
	}
	if got := counts.CountForSource(nil); got != 2 {
		t.Errorf("CountForSource(nil) = %d, want 2", got)
	}
}

func fmtName(prefix string, i int) string {
	return fmt.Sprintf("%s/repo-%d", prefix, i)
}
