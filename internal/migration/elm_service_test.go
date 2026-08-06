package migration

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/kuhlman-labs/github-migrator/internal/config"
	"github.com/kuhlman-labs/github-migrator/internal/elm"
	"github.com/kuhlman-labs/github-migrator/internal/models"
	"github.com/kuhlman-labs/github-migrator/internal/storage"
)

// ---------------------------------------------------------------------------
// Fake elm.Transport
//
// Tests drive a REAL *elm.Client over this fake, so every assertion about what
// the appliance was (and was not) asked to do is made against the actual command
// strings the wrapper emits. A stub client that returned canned results without
// issuing commands could not satisfy the command-log assertions below.
// ---------------------------------------------------------------------------

type fakeELMTransport struct {
	mu       sync.Mutex
	commands []string
	handler  func(command string) (stdout, stderr string, err error)
}

func (f *fakeELMTransport) Run(_ context.Context, command string) (string, string, error) {
	f.mu.Lock()
	f.commands = append(f.commands, command)
	h := f.handler
	f.mu.Unlock()
	if h == nil {
		return "", "", nil
	}
	return h(command)
}

func (f *fakeELMTransport) Close() error { return nil }

func (f *fakeELMTransport) log() []string {
	f.mu.Lock()
	defer f.mu.Unlock()
	out := make([]string, len(f.commands))
	copy(out, f.commands)
	return out
}

// count returns how many recorded commands contain sub.
func (f *fakeELMTransport) count(sub string) int {
	n := 0
	for _, c := range f.log() {
		if strings.Contains(c, sub) {
			n++
		}
	}
	return n
}

// Command fragments the tests assert on.
const (
	cmdCreate  = "elm migration create"
	cmdStart   = "elm migration start"
	cmdStatus  = "elm migration status"
	cmdList    = "elm migration list"
	cmdCutover = "elm migration cutover-to-destination"
)

// elmScript is a scripted appliance: it answers create/start/status/list/cutover
// deterministically, returning each status payload in turn and repeating the last.
type elmScript struct {
	createID string
	statuses []string
	list     string

	mu  sync.Mutex
	idx int
}

func (s *elmScript) nextStatus() string {
	s.mu.Lock()
	defer s.mu.Unlock()
	if len(s.statuses) == 0 {
		return `{"id":"m-1","state":"backfilling","cutover_ready":false}`
	}
	i := s.idx
	if i >= len(s.statuses) {
		i = len(s.statuses) - 1
	}
	s.idx++
	return s.statuses[i]
}

func (s *elmScript) handle(command string) (string, string, error) {
	switch {
	case strings.Contains(command, cmdCreate):
		id := s.createID
		if id == "" {
			id = "m-1"
		}
		return fmt.Sprintf(`{"id":%q}`, id), "", nil
	case strings.Contains(command, cmdStatus):
		return s.nextStatus(), "", nil
	case strings.Contains(command, cmdList):
		if s.list == "" {
			return `{"migrations":[]}`, "", nil
		}
		return s.list, "", nil
	case strings.Contains(command, cmdStart),
		strings.Contains(command, cmdCutover),
		strings.Contains(command, "elm migration cancel"):
		return "", "", nil
	}
	return "", "", fmt.Errorf("unexpected command: %s", command)
}

// ---------------------------------------------------------------------------
// Fixtures
// ---------------------------------------------------------------------------

func newELMTestDB(t *testing.T) *storage.Database {
	t.Helper()
	db, err := storage.NewDatabase(config.DatabaseConfig{Type: "sqlite", DSN: ":memory:"})
	if err != nil {
		t.Fatalf("failed to create test database: %v", err)
	}
	if err := db.Migrate(); err != nil {
		t.Fatalf("failed to run migrations: %v", err)
	}
	t.Cleanup(func() { _ = db.Close() })
	return db
}

func newTestLogger() *slog.Logger {
	return slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelError}))
}

// newELMService builds a service over a real elm.Client and the given transport.
func newELMService(t *testing.T, db *storage.Database, transport elm.Transport, maxSource, maxDest int) *ELMService {
	t.Helper()
	client, err := elm.NewClient(transport, elm.ClientConfig{TargetAPIURL: "https://api.octocorp.ghe.com"})
	if err != nil {
		t.Fatalf("elm.NewClient() error = %v", err)
	}
	svc, err := NewELMService(ELMServiceConfig{
		Storage:                  db,
		Client:                   client,
		Logger:                   newTestLogger(),
		PollInterval:             time.Millisecond,
		MaxConcurrentSource:      maxSource,
		MaxConcurrentDestination: maxDest,
	})
	if err != nil {
		t.Fatalf("NewELMService() error = %v", err)
	}
	return svc
}

// ensureSource creates the sources row a repository's source_id FK points at.
func ensureSource(t *testing.T, db *storage.Database, sourceID int64) {
	t.Helper()
	err := db.DB().Exec(
		`INSERT OR IGNORE INTO sources (id, name, type, base_url, token, is_active) VALUES (?, ?, ?, ?, ?, ?)`,
		sourceID, fmt.Sprintf("ghes-%d", sourceID), "github", "https://ghes.example.com/api/v3", "t", true,
	).Error
	if err != nil {
		t.Fatalf("failed to seed source %d: %v", sourceID, err)
	}
}

// seedELMRepo persists an ELM-routed GHES repository and returns the stored row.
func seedELMRepo(t *testing.T, db *storage.Database, fullName string, sourceID int64, status models.MigrationStatus) *models.Repository {
	t.Helper()
	ctx := context.Background()
	ensureSource(t, db, sourceID)
	route := string(models.MigrationRouteELM)
	sid := sourceID
	repo := &models.Repository{
		FullName:       fullName,
		Source:         models.SourceGHES,
		SourceURL:      "https://ghes.example.com",
		SourceID:       &sid,
		Status:         string(status),
		MigrationRoute: &route,
		Visibility:     models.VisibilityPrivate,
		DiscoveredAt:   time.Now(),
	}
	if err := db.SaveRepository(ctx, repo); err != nil {
		t.Fatalf("SaveRepository(%s) error = %v", fullName, err)
	}
	stored, err := db.GetRepository(ctx, fullName)
	if err != nil || stored == nil {
		t.Fatalf("GetRepository(%s) error = %v", fullName, err)
	}
	return stored
}

// seedInFlight seeds n repositories that are already syncing, each with its own
// elm_migrations row, all attributed to sourceID. The rows are created directly
// (never by calling StartBackfill), so the ceiling tests exercise the guard rather
// than the code path they are guarding.
func seedInFlight(t *testing.T, db *storage.Database, prefix string, sourceID int64, n int) {
	t.Helper()
	ctx := context.Background()
	for i := 0; i < n; i++ {
		name := fmt.Sprintf("%s/repo-%d", prefix, i)
		repo := seedELMRepo(t, db, name, sourceID, models.StatusSyncing)
		rec := &models.ELMMigration{
			RepositoryID:   repo.ID,
			ELMMigrationID: fmt.Sprintf("%s-m-%d", prefix, i),
			ELMStatus:      elm.StateBackfilling,
		}
		if err := db.UpsertELMMigration(ctx, rec); err != nil {
			t.Fatalf("UpsertELMMigration() error = %v", err)
		}
	}
}

func reloadRepo(t *testing.T, db *storage.Database, fullName string) *models.Repository {
	t.Helper()
	repo, err := db.GetRepository(context.Background(), fullName)
	if err != nil || repo == nil {
		t.Fatalf("GetRepository(%s) error = %v", fullName, err)
	}
	return repo
}

func reloadRecord(t *testing.T, db *storage.Database, repoID int64) *models.ELMMigration {
	t.Helper()
	rec, err := db.GetELMMigration(context.Background(), repoID)
	if err != nil {
		t.Fatalf("GetELMMigration() error = %v", err)
	}
	if rec == nil {
		t.Fatalf("expected an elm_migrations row for repository %d", repoID)
	}
	return rec
}

var elmTestTarget = ELMTarget{Org: "octocorp", Repo: "widgets", Visibility: models.VisibilityPrivate}

// ---------------------------------------------------------------------------
// Cross-boundary end-to-end test
// ---------------------------------------------------------------------------

// TestELMService_BackfillToCutover_EndToEnd drives the whole live-migration
// lifecycle across the transport, the client, the service and REAL SQLite
// storage, asserting the persisted repository status AND the persisted
// elm_migrations row at every step. Per-layer unit tests would let this seam
// break silently.
func TestELMService_BackfillToCutover_EndToEnd(t *testing.T) {
	db := newELMTestDB(t)
	ctx := context.Background()

	script := &elmScript{
		createID: "elm-42",
		statuses: []string{
			`{"id":"elm-42","state":"backfilling","phase":"git","progress_percent":40,"cutover_ready":false}`,
			`{"id":"elm-42","state":"ready_for_cutover","phase":"caught_up","progress_percent":100,"cutover_ready":true}`,
			`{"id":"elm-42","state":"completed","phase":"done","progress_percent":100,"cutover_ready":true}`,
		},
	}
	transport := &fakeELMTransport{handler: script.handle}
	svc := newELMService(t, db, transport, 10, 20)

	repo := seedELMRepo(t, db, "octocorp/widgets", 1, models.StatusQueuedForMigration)

	// --- create + start ---------------------------------------------------
	migrationID, err := svc.StartBackfill(ctx, repo, elmTestTarget)
	if err != nil {
		t.Fatalf("StartBackfill() error = %v", err)
	}
	if migrationID != "elm-42" {
		t.Errorf("StartBackfill() id = %q, want elm-42", migrationID)
	}
	if got := reloadRepo(t, db, repo.FullName).Status; got != string(models.StatusSyncing) {
		t.Errorf("after start, repository status = %q, want syncing", got)
	}
	rec := reloadRecord(t, db, repo.ID)
	if rec.ELMMigrationID != "elm-42" {
		t.Errorf("persisted elm_migration_id = %q, want elm-42", rec.ELMMigrationID)
	}
	if rec.CutoverReady {
		t.Error("a freshly started migration must not be persisted as cutover-ready")
	}
	if transport.count(cmdCreate) != 1 || transport.count(cmdStart) != 1 {
		t.Errorf("expected exactly one create and one start, got log %v", transport.log())
	}

	// --- poll: backfilling ------------------------------------------------
	if err := svc.PollOnce(ctx); err != nil {
		t.Fatalf("PollOnce() error = %v", err)
	}
	if got := reloadRepo(t, db, repo.FullName).Status; got != string(models.StatusSyncing) {
		t.Errorf("while backfilling, repository status = %q, want syncing", got)
	}
	rec = reloadRecord(t, db, repo.ID)
	if rec.CutoverReady {
		t.Error("cutover_ready must stay false while the backfill is still running")
	}
	if rec.LastPolledAt == nil {
		t.Error("last_polled_at must be stamped by a poll")
	}
	if rec.ProgressPercent == nil || *rec.ProgressPercent != 40 {
		t.Errorf("progress_percent = %v, want 40", rec.ProgressPercent)
	}

	// --- poll: ready for cutover -----------------------------------------
	if err := svc.PollOnce(ctx); err != nil {
		t.Fatalf("PollOnce() error = %v", err)
	}
	if got := reloadRepo(t, db, repo.FullName).Status; got != string(models.StatusAwaitingCutover) {
		t.Errorf("when the backfill is caught up, repository status = %q, want awaiting_cutover", got)
	}
	if rec = reloadRecord(t, db, repo.ID); !rec.CutoverReady {
		t.Error("cutover_ready must be persisted true once the appliance reports readiness")
	}

	// --- operator cutover -------------------------------------------------
	current := reloadRepo(t, db, repo.FullName)
	if err := svc.Cutover(ctx, current); err != nil {
		t.Fatalf("Cutover() error = %v", err)
	}
	if got := reloadRepo(t, db, repo.FullName).Status; got != string(models.StatusCuttingOver) {
		t.Errorf("after cutover, repository status = %q, want cutting_over", got)
	}
	if n := transport.count(cmdCutover); n != 1 {
		t.Errorf("expected exactly one cutover command, got %d in %v", n, transport.log())
	}
	for _, c := range transport.log() {
		if strings.Contains(c, "--force") {
			t.Fatalf("--force must never be emitted, got %q", c)
		}
	}

	// --- poll: completed --------------------------------------------------
	if err := svc.PollOnce(ctx); err != nil {
		t.Fatalf("PollOnce() error = %v", err)
	}
	final := reloadRepo(t, db, repo.FullName)
	if final.Status != string(models.StatusMigrationComplete) {
		t.Errorf("final repository status = %q, want migration_complete", final.Status)
	}
	if final.MigratedAt == nil {
		t.Error("migrated_at must be stamped on completion")
	}
}

// ---------------------------------------------------------------------------
// Concurrency ceilings (binding condition 1)
// ---------------------------------------------------------------------------

// TestELMService_RefusesBeyondSourceCeiling seeds TWO DISTINCT source_id values
// so a per-source count cannot pass by accidentally counting globally: source 1
// is at its ceiling while source 2 is well below it, and the global total (13) is
// below the destination ceiling (20). A global count would refuse both.
func TestELMService_RefusesBeyondSourceCeiling(t *testing.T) {
	db := newELMTestDB(t)
	ctx := context.Background()

	seedInFlight(t, db, "source-one", 1, 10)
	seedInFlight(t, db, "source-two", 2, 3)

	script := &elmScript{createID: "elm-new"}
	transport := &fakeELMTransport{handler: script.handle}
	svc := newELMService(t, db, transport, 10, 20)

	// At the ceiling for source 1: refused, and the repository stays queued.
	atCeiling := seedELMRepo(t, db, "octocorp/blocked", 1, models.StatusQueuedForMigration)
	_, err := svc.StartBackfill(ctx, atCeiling, elmTestTarget)
	if !errors.Is(err, ErrELMAdmissionCeiling) {
		t.Fatalf("StartBackfill() error = %v, want ErrELMAdmissionCeiling", err)
	}
	if got := reloadRepo(t, db, atCeiling.FullName).Status; got != string(models.StatusQueuedForMigration) {
		t.Errorf("a ceiling refusal must leave the repository queued, got %q", got)
	}
	if n := transport.count(cmdCreate); n != 0 {
		t.Errorf("a refused admission must issue no create command, got %d", n)
	}

	// Paired below-ceiling case: source 2 has capacity, so a create still happens.
	// Without this the control could "pass" by refusing everything.
	belowCeiling := seedELMRepo(t, db, "octocorp/allowed", 2, models.StatusQueuedForMigration)
	if _, err := svc.StartBackfill(ctx, belowCeiling, elmTestTarget); err != nil {
		t.Fatalf("StartBackfill() below the source ceiling error = %v", err)
	}
	if n := transport.count(cmdCreate); n != 1 {
		t.Errorf("expected exactly one create for the below-ceiling repository, got %d", n)
	}
	if got := reloadRepo(t, db, belowCeiling.FullName).Status; got != string(models.StatusSyncing) {
		t.Errorf("admitted repository status = %q, want syncing", got)
	}
}

// TestELMService_RefusesBeyondDestinationCeiling spreads 20 in-flight migrations
// across four sources (5 each) so NO source is at its own ceiling of 10. Only a
// GLOBAL count refuses the next create -- which is exactly the contract, because
// the deployment targets one configured destination.
func TestELMService_RefusesBeyondDestinationCeiling(t *testing.T) {
	db := newELMTestDB(t)
	ctx := context.Background()

	for s := int64(1); s <= 4; s++ {
		seedInFlight(t, db, fmt.Sprintf("source-%d", s), s, 5)
	}

	script := &elmScript{createID: "elm-new"}
	transport := &fakeELMTransport{handler: script.handle}
	svc := newELMService(t, db, transport, 10, 20)

	atCeiling := seedELMRepo(t, db, "octocorp/blocked", 5, models.StatusQueuedForMigration)
	_, err := svc.StartBackfill(ctx, atCeiling, elmTestTarget)
	if !errors.Is(err, ErrELMAdmissionCeiling) {
		t.Fatalf("StartBackfill() error = %v, want ErrELMAdmissionCeiling", err)
	}
	if got := reloadRepo(t, db, atCeiling.FullName).Status; got != string(models.StatusQueuedForMigration) {
		t.Errorf("a ceiling refusal must leave the repository queued, got %q", got)
	}
	if n := transport.count(cmdCreate); n != 0 {
		t.Errorf("a refused admission must issue no create command, got %d", n)
	}

	// Paired below-ceiling case: with the global ceiling raised by one the very
	// same fixture admits a create, so the refusal above is the ceiling and not
	// something incidental.
	roomier := newELMService(t, db, transport, 10, 21)
	if _, err := roomier.StartBackfill(ctx, atCeiling, elmTestTarget); err != nil {
		t.Fatalf("StartBackfill() below the destination ceiling error = %v", err)
	}
	if n := transport.count(cmdCreate); n != 1 {
		t.Errorf("expected exactly one create for the below-ceiling case, got %d", n)
	}
}

// TestELMService_StartBackfillRefusesIneligibleRepository pins the service's own
// eligibility guard. The strategy gate already keeps unrouted repositories away
// from ELM, but the service must not create a live migration for one if it is
// ever called directly (from the API, say).
func TestELMService_StartBackfillRefusesIneligibleRepository(t *testing.T) {
	db := newELMTestDB(t)
	ctx := context.Background()

	transport := &fakeELMTransport{handler: (&elmScript{}).handle}
	svc := newELMService(t, db, transport, 10, 20)

	// By construction: a GEI-routed repository (no route recorded at all).
	ensureSource(t, db, 1)
	sid := int64(1)
	repo := &models.Repository{
		FullName:     "octocorp/gei-repo",
		Source:       models.SourceGHES,
		SourceURL:    "https://ghes.example.com",
		SourceID:     &sid,
		Status:       string(models.StatusQueuedForMigration),
		Visibility:   models.VisibilityPrivate,
		DiscoveredAt: time.Now(),
	}
	if err := db.SaveRepository(ctx, repo); err != nil {
		t.Fatalf("SaveRepository() error = %v", err)
	}

	if _, err := svc.StartBackfill(ctx, repo, elmTestTarget); err == nil {
		t.Fatal("expected StartBackfill to refuse a repository that is not ELM-routed")
	}
	if n := transport.count(cmdCreate); n != 0 {
		t.Errorf("an ineligible repository must issue no create command, got %d", n)
	}
	if got := reloadRepo(t, db, repo.FullName).Status; got != string(models.StatusQueuedForMigration) {
		t.Errorf("status = %q, want queued_for_migration (unchanged)", got)
	}
}

// ---------------------------------------------------------------------------
// Cutover gating
// ---------------------------------------------------------------------------

// TestELMService_CutoverRefusedWhenNotReady asserts COMMITTED STATE after the
// call, not error identity: the fake transport must have recorded ZERO cutover
// commands and the repository must still be awaiting_cutover.
//
// The bad state is seeded BY CONSTRUCTION -- the persisted record says not ready
// while the appliance would report ready -- so deleting the persisted-readiness
// gate lets the flow run all the way to a real cutover command.
func TestELMService_CutoverRefusedWhenNotReady(t *testing.T) {
	db := newELMTestDB(t)
	ctx := context.Background()

	repo := seedELMRepo(t, db, "octocorp/widgets", 1, models.StatusAwaitingCutover)
	if err := db.UpsertELMMigration(ctx, &models.ELMMigration{
		RepositoryID:   repo.ID,
		ELMMigrationID: "elm-42",
		ELMStatus:      elm.StateBackfilling,
		CutoverReady:   false, // by construction: the persisted record says NOT ready
	}); err != nil {
		t.Fatalf("UpsertELMMigration() error = %v", err)
	}

	script := &elmScript{
		statuses: []string{`{"id":"elm-42","state":"ready_for_cutover","cutover_ready":true}`},
	}
	transport := &fakeELMTransport{handler: script.handle}
	svc := newELMService(t, db, transport, 10, 20)

	err := svc.Cutover(ctx, repo)
	if !errors.Is(err, ErrELMNotReadyForCutover) {
		t.Fatalf("Cutover() error = %v, want ErrELMNotReadyForCutover", err)
	}
	if n := transport.count(cmdCutover); n != 0 {
		t.Errorf("a refused cutover must issue no cutover command, got %d in %v", n, transport.log())
	}
	if got := reloadRepo(t, db, repo.FullName).Status; got != string(models.StatusAwaitingCutover) {
		t.Errorf("after a refused cutover the repository status = %q, want awaiting_cutover", got)
	}
}

// TestELMService_CutoverRefusedWhenFreshStatusNotReady covers the SECOND gate: a
// persisted record that claims readiness is not enough, the appliance must
// re-confirm it. Bad state by construction: the record says ready, the appliance
// says not ready.
func TestELMService_CutoverRefusedWhenFreshStatusNotReady(t *testing.T) {
	db := newELMTestDB(t)
	ctx := context.Background()

	repo := seedELMRepo(t, db, "octocorp/widgets", 1, models.StatusAwaitingCutover)
	if err := db.UpsertELMMigration(ctx, &models.ELMMigration{
		RepositoryID:   repo.ID,
		ELMMigrationID: "elm-42",
		ELMStatus:      elm.StateReadyForCutover,
		CutoverReady:   true, // stale: the appliance no longer agrees
	}); err != nil {
		t.Fatalf("UpsertELMMigration() error = %v", err)
	}

	script := &elmScript{
		statuses: []string{`{"id":"elm-42","state":"ready_for_cutover","cutover_ready":false}`},
	}
	transport := &fakeELMTransport{handler: script.handle}
	svc := newELMService(t, db, transport, 10, 20)

	err := svc.Cutover(ctx, repo)
	if !errors.Is(err, ErrELMNotReadyForCutover) {
		t.Fatalf("Cutover() error = %v, want ErrELMNotReadyForCutover", err)
	}
	if n := transport.count(cmdCutover); n != 0 {
		t.Errorf("a refused cutover must issue no cutover command, got %d in %v", n, transport.log())
	}
	if got := reloadRepo(t, db, repo.FullName).Status; got == string(models.StatusCuttingOver) {
		t.Error("a refused cutover must not move the repository to cutting_over")
	}
	if rec := reloadRecord(t, db, repo.ID); rec.CutoverReady {
		t.Error("the stale readiness must be corrected from the fresh status")
	}
}

// TestELMService_CutoverRequiresRecord covers the missing-record branch.
func TestELMService_CutoverRequiresRecord(t *testing.T) {
	db := newELMTestDB(t)
	ctx := context.Background()

	repo := seedELMRepo(t, db, "octocorp/widgets", 1, models.StatusAwaitingCutover)
	transport := &fakeELMTransport{handler: (&elmScript{}).handle}
	svc := newELMService(t, db, transport, 10, 20)

	if err := svc.Cutover(ctx, repo); !errors.Is(err, ErrELMNoRecord) {
		t.Fatalf("Cutover() error = %v, want ErrELMNoRecord", err)
	}
	if n := transport.count(cmdCutover); n != 0 {
		t.Errorf("expected no cutover command, got %d", n)
	}
}

// ---------------------------------------------------------------------------
// Per-failure-mode poll behavior
// ---------------------------------------------------------------------------

// pollFixture seeds one syncing repository with a persisted record whose
// cutover_ready is set as given, and returns the service and transport.
func pollFixture(t *testing.T, ready bool, handler func(string) (string, string, error)) (*ELMService, *storage.Database, *models.Repository, *fakeELMTransport) {
	t.Helper()
	db := newELMTestDB(t)
	repo := seedELMRepo(t, db, "octocorp/widgets", 1, models.StatusSyncing)
	if err := db.UpsertELMMigration(context.Background(), &models.ELMMigration{
		RepositoryID:   repo.ID,
		ELMMigrationID: "elm-42",
		ELMStatus:      elm.StateBackfilling,
		CutoverReady:   ready,
	}); err != nil {
		t.Fatalf("UpsertELMMigration() error = %v", err)
	}
	transport := &fakeELMTransport{handler: handler}
	return newELMService(t, db, transport, 10, 20), db, repo, transport
}

// TestELMService_TransportFailureLeavesStatusUnchanged: the appliance is
// unreachable, so the failure is retryable -- record it and change nothing else.
func TestELMService_TransportFailureLeavesStatusUnchanged(t *testing.T) {
	svc, db, repo, _ := pollFixture(t, false, func(string) (string, string, error) {
		return "", "", errors.New("dial tcp 10.0.0.1:22: connect: connection refused")
	})

	if err := svc.PollOnce(context.Background()); err != nil {
		t.Fatalf("PollOnce() error = %v", err)
	}
	if got := reloadRepo(t, db, repo.FullName).Status; got != string(models.StatusSyncing) {
		t.Errorf("an unreachable appliance must not change repository status, got %q", got)
	}
	rec := reloadRecord(t, db, repo.ID)
	if rec.LastError == nil {
		t.Error("a transport failure must be recorded in last_error")
	}
	if rec.CutoverReady {
		t.Error("a transport failure must not invent readiness")
	}
}

// TestELMService_NonZeroExitMarksMigrationFailed: the elm CLI ran and reported a
// failure, so the repository moves to migration_failed.
func TestELMService_NonZeroExitMarksMigrationFailed(t *testing.T) {
	svc, db, repo, _ := pollFixture(t, false, func(string) (string, string, error) {
		return "", "migration elm-42 does not exist", &elm.ExitError{Code: 2}
	})

	if err := svc.PollOnce(context.Background()); err != nil {
		t.Fatalf("PollOnce() error = %v", err)
	}
	if got := reloadRepo(t, db, repo.FullName).Status; got != string(models.StatusMigrationFailed) {
		t.Errorf("a non-zero exit must fail the repository, got %q", got)
	}
	rec := reloadRecord(t, db, repo.ID)
	if rec.LastError == nil || !strings.Contains(*rec.LastError, "does not exist") {
		t.Errorf("last_error must carry the appliance stderr, got %v", rec.LastError)
	}
}

// TestELMService_UnparseableOutputLeavesCutoverReadyUnchanged: a preview CLI whose
// output drifts must not silently read as a stalled sync. Readiness is seeded true
// BY CONSTRUCTION and must survive the parse failure untouched.
func TestELMService_UnparseableOutputLeavesCutoverReadyUnchanged(t *testing.T) {
	svc, db, repo, _ := pollFixture(t, true, func(string) (string, string, error) {
		return "elm: something entirely new", "", nil
	})

	if err := svc.PollOnce(context.Background()); err != nil {
		t.Fatalf("PollOnce() error = %v", err)
	}
	rec := reloadRecord(t, db, repo.ID)
	if !rec.CutoverReady {
		t.Error("unparseable output must leave cutover_ready UNCHANGED, not write it false")
	}
	if rec.LastError == nil {
		t.Error("a parse failure must be recorded in last_error")
	}
	if got := reloadRepo(t, db, repo.FullName).Status; got != string(models.StatusSyncing) {
		t.Errorf("unparseable output must not change repository status, got %q", got)
	}
}

// ---------------------------------------------------------------------------
// Restart reconciliation
// ---------------------------------------------------------------------------

func TestELMService_ReconcileReadoptsInFlight(t *testing.T) {
	db := newELMTestDB(t)
	ctx := context.Background()

	repo := seedELMRepo(t, db, "octocorp/widgets", 1, models.StatusSyncing)
	if err := db.UpsertELMMigration(ctx, &models.ELMMigration{
		RepositoryID:   repo.ID,
		ELMMigrationID: "elm-42",
		ELMStatus:      elm.StateBackfilling,
	}); err != nil {
		t.Fatalf("UpsertELMMigration() error = %v", err)
	}

	script := &elmScript{list: `{"migrations":[{"id":"elm-42","state":"ready_for_cutover","cutover_ready":true}]}`}
	transport := &fakeELMTransport{handler: script.handle}
	svc := newELMService(t, db, transport, 10, 20)

	if err := svc.Reconcile(ctx); err != nil {
		t.Fatalf("Reconcile() error = %v", err)
	}
	if got := reloadRepo(t, db, repo.FullName).Status; got != string(models.StatusAwaitingCutover) {
		t.Errorf("a re-adopted migration must pick up its appliance state, got %q", got)
	}
	if rec := reloadRecord(t, db, repo.ID); !rec.CutoverReady {
		t.Error("re-adoption must persist the appliance-reported readiness")
	}
}

func TestELMService_ReconcileMarksMissingFailed(t *testing.T) {
	db := newELMTestDB(t)
	ctx := context.Background()

	repo := seedELMRepo(t, db, "octocorp/widgets", 1, models.StatusSyncing)
	if err := db.UpsertELMMigration(ctx, &models.ELMMigration{
		RepositoryID:   repo.ID,
		ELMMigrationID: "elm-gone",
		ELMStatus:      elm.StateBackfilling,
	}); err != nil {
		t.Fatalf("UpsertELMMigration() error = %v", err)
	}

	// The appliance knows about a DIFFERENT migration, so elm-gone is orphaned.
	script := &elmScript{list: `{"migrations":[{"id":"elm-other","state":"backfilling"}]}`}
	transport := &fakeELMTransport{handler: script.handle}
	svc := newELMService(t, db, transport, 10, 20)

	if err := svc.Reconcile(ctx); err != nil {
		t.Fatalf("Reconcile() error = %v", err)
	}
	if got := reloadRepo(t, db, repo.FullName).Status; got != string(models.StatusMigrationFailed) {
		t.Errorf("an orphaned migration must be marked failed, got %q", got)
	}
	rec := reloadRecord(t, db, repo.ID)
	if rec.LastError == nil || !strings.Contains(*rec.LastError, "no longer known") {
		t.Errorf("the failure must carry a reason, got %v", rec.LastError)
	}
}

// ---------------------------------------------------------------------------
// Worker isolation and admission re-queue
// ---------------------------------------------------------------------------

// TestELMService_AwaitingCutoverDoesNotHoldWorkerSlot asserts that a repository
// parked in awaiting_cutover is absent from the selection MigrationWorker makes
// (internal/worker/worker.go queues only queued_for_migration and dry_run_queued),
// so a live migration can wait days on an operator without consuming a slot.
func TestELMService_AwaitingCutoverDoesNotHoldWorkerSlot(t *testing.T) {
	db := newELMTestDB(t)
	ctx := context.Background()

	seedELMRepo(t, db, "octocorp/waiting", 1, models.StatusAwaitingCutover)
	seedELMRepo(t, db, "octocorp/syncing", 1, models.StatusSyncing)
	seedELMRepo(t, db, "octocorp/ready", 1, models.StatusQueuedForMigration)

	// The exact filter the migration worker uses to pick up work.
	repos, err := db.ListRepositories(ctx, map[string]any{
		"status": []string{
			string(models.StatusQueuedForMigration),
			string(models.StatusDryRunQueued),
		},
		"limit": 100,
	})
	if err != nil {
		t.Fatalf("ListRepositories() error = %v", err)
	}

	for _, r := range repos {
		if r.Status == string(models.StatusAwaitingCutover) || r.Status == string(models.StatusSyncing) {
			t.Errorf("repository %s in %s must not be picked up by the migration worker", r.FullName, r.Status)
		}
	}
	if len(repos) != 1 || repos[0].FullName != "octocorp/ready" {
		t.Errorf("expected only the queued repository to be selected, got %d", len(repos))
	}
}

// TestELMService_AdmissionRefusalRequeuesRepository pins the re-queue branch in
// handleStrategyPhaseError: a ceiling refusal is a capacity problem, not a
// migration failure, so the repository goes back to queued rather than needing an
// operator reset.
func TestELMService_AdmissionRefusalRequeuesRepository(t *testing.T) {
	db := newELMTestDB(t)
	ctx := context.Background()

	repo := seedELMRepo(t, db, "octocorp/widgets", 1, models.StatusQueuedForMigration)
	e := &Executor{storage: db, logger: newTestLogger()}
	mc := e.NewMigrationContext(repo, nil, false)
	strategy := NewELMStrategy(e, nil)

	e.handleStrategyPhaseError(ctx, mc, strategy, fmt.Errorf("wrapped: %w", ErrELMAdmissionCeiling))

	if got := reloadRepo(t, db, repo.FullName).Status; got != string(models.StatusQueuedForMigration) {
		t.Errorf("a ceiling refusal must leave the repository queued, got %q", got)
	}

	// Paired case: an ordinary failure still fails the repository, so the branch
	// above cannot pass by never failing anything.
	other := seedELMRepo(t, db, "octocorp/other", 1, models.StatusQueuedForMigration)
	otherMC := e.NewMigrationContext(other, nil, false)
	e.handleStrategyPhaseError(ctx, otherMC, strategy, errors.New("appliance exploded"))
	if got := reloadRepo(t, db, other.FullName).Status; got != string(models.StatusMigrationFailed) {
		t.Errorf("an ordinary failure must fail the repository, got %q", got)
	}
}

// TestELMService_RunStopsWithContext exercises the long-lived loop with an
// injected sub-millisecond interval and asserts it polls and then stops cleanly.
func TestELMService_RunStopsWithContext(t *testing.T) {
	db := newELMTestDB(t)
	repo := seedELMRepo(t, db, "octocorp/widgets", 1, models.StatusSyncing)
	if err := db.UpsertELMMigration(context.Background(), &models.ELMMigration{
		RepositoryID:   repo.ID,
		ELMMigrationID: "elm-42",
		ELMStatus:      elm.StateBackfilling,
	}); err != nil {
		t.Fatalf("UpsertELMMigration() error = %v", err)
	}

	script := &elmScript{
		list:     `{"migrations":[{"id":"elm-42","state":"backfilling"}]}`,
		statuses: []string{`{"id":"elm-42","state":"ready_for_cutover","cutover_ready":true}`},
	}
	transport := &fakeELMTransport{handler: script.handle}
	svc := newELMService(t, db, transport, 10, 20)

	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan struct{})
	go func() { svc.Run(ctx); close(done) }()

	deadline := time.Now().Add(5 * time.Second)
	for reloadRepo(t, db, repo.FullName).Status != string(models.StatusAwaitingCutover) {
		if time.Now().After(deadline) {
			t.Fatal("poll loop did not advance the repository to awaiting_cutover")
		}
	}

	cancel()
	select {
	case <-done:
	case <-time.After(5 * time.Second):
		t.Fatal("Run() did not stop when its context was cancelled")
	}
}
