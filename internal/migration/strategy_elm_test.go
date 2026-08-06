package migration

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"testing"

	"github.com/kuhlman-labs/github-migrator/internal/models"
)

// elmRepo builds an ELM-routed GHES repository fixture.
func elmRepo() *models.Repository {
	route := string(models.MigrationRouteELM)
	return &models.Repository{
		FullName:       "octocorp/widgets",
		Source:         models.SourceGHES,
		MigrationRoute: &route,
	}
}

func TestELMStrategy_SupportsRepository(t *testing.T) {
	strategy := NewELMStrategy(nil, nil)
	gei := string(models.MigrationRouteGEI)

	tests := []struct {
		name string
		repo *models.Repository
		want bool
	}{
		{
			name: "elm-routed GHES repository",
			repo: elmRepo(),
			want: true,
		},
		{
			name: "no route recorded (the GEI default)",
			repo: &models.Repository{FullName: "octocorp/widgets", Source: models.SourceGHES},
			want: false,
		},
		{
			name: "explicitly gei-routed",
			repo: func() *models.Repository {
				r := elmRepo()
				r.MigrationRoute = &gei
				return r
			}(),
			want: false,
		},
		{
			name: "elm-routed but has an ADO project",
			repo: func() *models.Repository {
				r := elmRepo()
				r.SetADOProject(strPtr("MyProject"))
				return r
			}(),
			want: false,
		},
		{
			name: "elm-routed but not a GHES source",
			repo: func() *models.Repository {
				r := elmRepo()
				r.Source = models.SourceGHEC
				return r
			}(),
			want: false,
		},
		{
			name: "nil repository",
			repo: nil,
			want: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := strategy.SupportsRepository(tt.repo); got != tt.want {
				t.Errorf("SupportsRepository() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestELMStrategy_ShouldUnlockSource(t *testing.T) {
	// ELM keeps the source writable until cutover, so there is nothing to unlock.
	if NewELMStrategy(nil, nil).ShouldUnlockSource() {
		t.Error("ELMStrategy.ShouldUnlockSource() = true, want false")
	}
}

// TestELMStrategy_ValidateSourceRequiresService pins the refusal that keeps an
// explicitly ELM-routed repository from silently taking the GEI corridor when ELM
// is not configured.
func TestELMStrategy_ValidateSourceRequiresService(t *testing.T) {
	strategy := NewELMStrategy(nil, nil)
	err := strategy.ValidateSource(context.Background(), elmRepo())
	if !errors.Is(err, ErrELMNotConfigured) {
		t.Fatalf("ValidateSource() error = %v, want ErrELMNotConfigured", err)
	}
}

// TestELMStrategy_DryRunCreatesNoMigration asserts on the fake transport's
// COMMAND LOG that an ELM dry run issues `elm migration list` and no create,
// start or cutover -- an assertion a stub that returned success without issuing
// commands could not satisfy.
func TestELMStrategy_DryRunCreatesNoMigration(t *testing.T) {
	db := newELMTestDB(t)
	ctx := context.Background()

	transport := &fakeELMTransport{handler: (&elmScript{}).handle}
	svc := newELMService(t, db, transport, 10, 20)
	repo := seedELMRepo(t, db, "octocorp/widgets", 1, models.StatusDryRunQueued)

	e := &Executor{storage: db, logger: newTestLogger()}
	if err := e.ExecuteWithStrategyAndELM(ctx, repo, nil, true, svc); err != nil {
		t.Fatalf("ELM dry run error = %v", err)
	}

	log := transport.log()
	if n := transport.count(cmdList); n == 0 {
		t.Errorf("a dry run must prove reachability with `elm migration list`, got %v", log)
	}
	for _, forbidden := range []string{cmdCreate, cmdStart, cmdCutover} {
		if transport.count(forbidden) != 0 {
			t.Errorf("a dry run must issue no %q, got %v", forbidden, log)
		}
	}

	stored := reloadRepo(t, db, repo.FullName)
	if stored.Status != string(models.StatusDryRunComplete) {
		t.Errorf("dry run status = %q, want dry_run_complete", stored.Status)
	}
	if stored.IsSourceLocked {
		t.Error("an ELM dry run must never lock the source")
	}
	if stored.LastDryRunAt == nil {
		t.Error("last_dry_run_at must be stamped by a successful dry run")
	}
}

// TestELMStrategy_DryRunFailsWhenApplianceUnreachable covers the failure branch of
// the preflight itself. Source validation passes (the first `migration list`
// succeeds) and the preflight's own list call is the one that fails, so the RED
// lands on the preflight branch rather than on the earlier validation phase.
func TestELMStrategy_DryRunFailsWhenApplianceUnreachable(t *testing.T) {
	db := newELMTestDB(t)
	ctx := context.Background()

	var lists int
	transport := &fakeELMTransport{handler: func(command string) (string, string, error) {
		if strings.Contains(command, cmdList) {
			lists++
			if lists == 1 {
				return `{"migrations":[]}`, "", nil
			}
			return "", "", errors.New("dial tcp 10.0.0.1:22: connect: connection refused")
		}
		return "", "", fmt.Errorf("unexpected command: %s", command)
	}}
	svc := newELMService(t, db, transport, 10, 20)
	repo := seedELMRepo(t, db, "octocorp/widgets", 1, models.StatusDryRunQueued)

	e := &Executor{storage: db, logger: newTestLogger()}
	err := e.ExecuteWithStrategyAndELM(ctx, repo, nil, true, svc)
	if err == nil {
		t.Fatal("expected an error when the appliance is unreachable")
	}
	if got := reloadRepo(t, db, repo.FullName).Status; got != string(models.StatusDryRunFailed) {
		t.Errorf("status after a failed dry run = %q, want dry_run_failed", got)
	}
	if transport.count(cmdCreate) != 0 {
		t.Error("a failed dry run must still create no migration")
	}
}

// TestELMStrategy_StartMigrationRunsBackfill drives the production path through
// the unified executor and asserts the repository is handed off to the poll loop
// in `syncing` rather than being driven through the GEI polling phases.
func TestELMStrategy_StartMigrationRunsBackfill(t *testing.T) {
	db := newELMTestDB(t)
	ctx := context.Background()

	transport := &fakeELMTransport{handler: (&elmScript{createID: "elm-7"}).handle}
	svc := newELMService(t, db, transport, 10, 20)
	repo := seedELMRepo(t, db, "octocorp/widgets", 1, models.StatusQueuedForMigration)

	e := &Executor{storage: db, logger: newTestLogger()}
	if err := e.ExecuteWithStrategyAndELM(ctx, repo, nil, false, svc); err != nil {
		t.Fatalf("ELM migration error = %v", err)
	}

	if got := reloadRepo(t, db, repo.FullName).Status; got != string(models.StatusSyncing) {
		t.Errorf("status after start = %q, want syncing", got)
	}
	if rec := reloadRecord(t, db, repo.ID); rec.ELMMigrationID != "elm-7" {
		t.Errorf("persisted elm_migration_id = %q, want elm-7", rec.ELMMigrationID)
	}
	// The create command must carry the destination coordinates the executor
	// resolved, single-quoted by the client.
	var createCmd string
	for _, c := range transport.log() {
		if strings.Contains(c, cmdCreate) {
			createCmd = c
		}
	}
	if !strings.Contains(createCmd, "--target-org 'octocorp'") {
		t.Errorf("create command did not carry the destination org: %q", createCmd)
	}
}
