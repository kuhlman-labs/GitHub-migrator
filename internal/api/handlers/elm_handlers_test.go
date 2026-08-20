package handlers

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/kuhlman-labs/github-migrator/internal/migration"
	"github.com/kuhlman-labs/github-migrator/internal/models"
)

// fakeELMService records every Cutover call so a refusal can be asserted against
// COMMITTED EFFECT (zero commands dispatched) and not only against the returned
// error. A control that fires and is then rolled back returns a byte-identical
// error; the call log does not lie.
type fakeELMService struct {
	calls []string
	err   error
	// onCutover runs on an admitted cutover so a test can observe the state the
	// real service would have committed.
	onCutover func(repo *models.Repository)
}

func (f *fakeELMService) Cutover(_ context.Context, repo *models.Repository) error {
	f.calls = append(f.calls, repo.FullName)
	if f.err != nil {
		return f.err
	}
	if f.onCutover != nil {
		f.onCutover(repo)
	}
	return nil
}

// setupELMHandler builds a Handler over the given mock and registers svc (which
// may be nil, meaning ELM is not configured). The registration is torn down after
// the test so no state leaks between cases.
func setupELMHandler(t *testing.T, mock *MockDataStore, svc ELMCutoverService) *Handler {
	t.Helper()
	h := setupTestHandlerWithMock(t, mock)
	if svc != nil {
		SetELMService(h, svc)
	}
	t.Cleanup(func() { SetELMService(h, nil) })
	return h
}

// seedRepo stores a repository in the mock under both lookup maps and returns it.
func seedRepo(mock *MockDataStore, fullName string, status models.MigrationStatus, route *string) *models.Repository {
	repo := &models.Repository{
		ID:             42,
		FullName:       fullName,
		Status:         string(status),
		Source:         models.SourceGHES,
		MigrationRoute: route,
	}
	mock.Repos[fullName] = repo
	mock.ReposByID[repo.ID] = repo
	return repo
}

// postAction builds a request as HandleRepositoryAction would have prepared it:
// the repository name already URL-decoded and stashed under cleanFullNameKey.
func postAction(fullName, body string) *http.Request {
	req := httptest.NewRequest(http.MethodPost, "/api/v1/repositories/"+fullName, strings.NewReader(body))
	return req.WithContext(context.WithValue(req.Context(), cleanFullNameKey, fullName))
}

func decodeBody(t *testing.T, w *httptest.ResponseRecorder) map[string]any {
	t.Helper()
	var out map[string]any
	if err := json.Unmarshal(w.Body.Bytes(), &out); err != nil {
		t.Fatalf("response is not JSON: %v (body=%s)", err, w.Body.String())
	}
	return out
}

// ============================================================================
// Route writer
// ============================================================================

// TestSetMigrationRoute_RejectsUnknownRoute is the counterfactual vehicle for the
// route-value validation. It asserts COMMITTED STATE as well as the status code:
// the store used here writes any value straight through, so deleting the handler's
// models.IsValidMigrationRoute guard commits "gei-plus" and fails the reload
// assertion, not merely the status assertion.
func TestSetMigrationRoute_RejectsUnknownRoute(t *testing.T) {
	for _, route := range []string{"gei-plus", "ELM", "live", "gei ", "'; drop table repositories; --"} {
		t.Run(route, func(t *testing.T) {
			mock := NewMockDataStore()
			// Seeded BY CONSTRUCTION with a route the guard is not involved in
			// setting, so the "unchanged" assertion below is meaningful.
			existing := string(models.MigrationRouteELM)
			seedRepo(mock, "octo/app", models.StatusSyncing, &existing)
			h := setupELMHandler(t, mock, nil)

			w := httptest.NewRecorder()
			h.SetMigrationRoute(w, postAction("octo/app", fmt.Sprintf(`{"route":%q}`, route)))

			if w.Code != http.StatusBadRequest {
				t.Errorf("expected 400 for route %q, got %d (body=%s)", route, w.Code, w.Body.String())
			}
			if reason := decodeBody(t, w)["reason"]; reason != elmReasonInvalidRoute {
				t.Errorf("expected reason %q, got %v", elmReasonInvalidRoute, reason)
			}

			// COMMITTED STATE: the stored route must be untouched.
			reloaded := mock.Repos["octo/app"]
			if reloaded.MigrationRoute == nil || *reloaded.MigrationRoute != existing {
				t.Errorf("expected stored route to stay %q, got %v", existing, reloaded.MigrationRoute)
			}
			if reloaded.GetMigrationRoute() != string(models.MigrationRouteELM) {
				t.Errorf("expected repository to still read as elm-routed, got %q", reloaded.GetMigrationRoute())
			}
		})
	}
}

func TestSetMigrationRoute_PersistsLegalRoutes(t *testing.T) {
	for _, route := range []string{string(models.MigrationRouteELM), string(models.MigrationRouteGEI)} {
		t.Run(route, func(t *testing.T) {
			mock := NewMockDataStore()
			seedRepo(mock, "octo/app", models.StatusPending, nil)
			h := setupELMHandler(t, mock, nil)

			w := httptest.NewRecorder()
			h.SetMigrationRoute(w, postAction("octo/app", fmt.Sprintf(`{"route":%q}`, route)))

			if w.Code != http.StatusOK {
				t.Fatalf("expected 200, got %d (body=%s)", w.Code, w.Body.String())
			}
			if got := decodeBody(t, w)["migration_route"]; got != route {
				t.Errorf("expected response route %q, got %v", route, got)
			}
			stored := mock.Repos["octo/app"]
			if stored.MigrationRoute == nil || *stored.MigrationRoute != route {
				t.Fatalf("expected stored route %q, got %v", route, stored.MigrationRoute)
			}
			if stored.GetMigrationRoute() != route {
				t.Errorf("expected GetMigrationRoute %q, got %q", route, stored.GetMigrationRoute())
			}
		})
	}
}

// TestSetMigrationRoute_ClearsRoute proves a null (or empty) route clears the
// column back to the GEI default rather than storing a meaningless value.
func TestSetMigrationRoute_ClearsRoute(t *testing.T) {
	for name, body := range map[string]string{
		"explicit null": `{"route":null}`,
		"empty string":  `{"route":""}`,
		"absent field":  `{}`,
	} {
		t.Run(name, func(t *testing.T) {
			mock := NewMockDataStore()
			elm := string(models.MigrationRouteELM)
			seedRepo(mock, "octo/app", models.StatusPending, &elm)
			h := setupELMHandler(t, mock, nil)

			w := httptest.NewRecorder()
			h.SetMigrationRoute(w, postAction("octo/app", body))

			if w.Code != http.StatusOK {
				t.Fatalf("expected 200, got %d (body=%s)", w.Code, w.Body.String())
			}
			stored := mock.Repos["octo/app"]
			if stored.MigrationRoute != nil {
				t.Errorf("expected route to be cleared, got %q", *stored.MigrationRoute)
			}
			if stored.GetMigrationRoute() != string(models.MigrationRouteGEI) {
				t.Errorf("expected cleared route to read as gei, got %q", stored.GetMigrationRoute())
			}
		})
	}
}

func TestSetMigrationRoute_InvalidJSON(t *testing.T) {
	mock := NewMockDataStore()
	seedRepo(mock, "octo/app", models.StatusPending, nil)
	h := setupELMHandler(t, mock, nil)

	w := httptest.NewRecorder()
	h.SetMigrationRoute(w, postAction("octo/app", `{"route":`))

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400 for malformed JSON, got %d", w.Code)
	}
	if mock.Repos["octo/app"].MigrationRoute != nil {
		t.Error("expected no route to be committed for a malformed body")
	}
}

func TestSetMigrationRoute_StorageErrorIsReported(t *testing.T) {
	mock := NewMockDataStore().WithMigrationRouteError(errors.New("write conflict"))
	seedRepo(mock, "octo/app", models.StatusPending, nil)
	h := setupELMHandler(t, mock, nil)

	w := httptest.NewRecorder()
	h.SetMigrationRoute(w, postAction("octo/app", `{"route":"elm"}`))

	if w.Code != http.StatusInternalServerError {
		t.Errorf("expected 500 when the store refuses the write, got %d", w.Code)
	}
	if mock.Repos["octo/app"].MigrationRoute != nil {
		t.Error("expected no route to be committed when the store fails")
	}
}

// ============================================================================
// Cutover
// ============================================================================

// TestELMHandlers_CutoverFlow drives the HTTP handler -> service -> storage seam
// and asserts the JSON response shape the dashboard consumes.
func TestELMHandlers_CutoverFlow(t *testing.T) {
	mock := NewMockDataStore()
	route := string(models.MigrationRouteELM)
	repo := seedRepo(mock, "octo/app", models.StatusAwaitingCutover, &route)
	mock.SetELMMigration(&models.ELMMigration{
		RepositoryID:   repo.ID,
		ELMMigrationID: "elm-123",
		ELMStatus:      "cutover_ready",
		CutoverReady:   true,
	})

	svc := &fakeELMService{onCutover: func(r *models.Repository) {
		// Stand in for the service's own commit.
		r.Status = string(models.StatusCuttingOver)
	}}
	h := setupELMHandler(t, mock, svc)

	w := httptest.NewRecorder()
	h.TriggerCutover(w, postAction("octo/app", ""))

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d (body=%s)", w.Code, w.Body.String())
	}
	if len(svc.calls) != 1 || svc.calls[0] != "octo/app" {
		t.Fatalf("expected exactly one cutover for octo/app, got %v", svc.calls)
	}

	body := decodeBody(t, w)
	if body["message"] != "Cutover started" {
		t.Errorf("unexpected message: %v", body["message"])
	}
	repoJSON, ok := body["repository"].(map[string]any)
	if !ok {
		t.Fatalf("expected a repository object in the response, got %v", body["repository"])
	}
	if repoJSON["status"] != string(models.StatusCuttingOver) {
		t.Errorf("expected status %q in the response, got %v", models.StatusCuttingOver, repoJSON["status"])
	}
	if repoJSON["migration_route"] != string(models.MigrationRouteELM) {
		t.Errorf("expected migration_route %q in the response, got %v", models.MigrationRouteELM, repoJSON["migration_route"])
	}
}

// TestELMHandlers_CutoverRefusedWhenNotReady pins the handler-layer readiness
// gate. Because the control's effect is COMMITTED STATE (a dispatched cutover
// command is irreversible), the assertions read state AFTER the call returns: the
// fake service recorded ZERO cutovers and the repository is still
// awaiting_cutover.
func TestELMHandlers_CutoverRefusedWhenNotReady(t *testing.T) {
	mock := NewMockDataStore()
	route := string(models.MigrationRouteELM)
	repo := seedRepo(mock, "octo/app", models.StatusAwaitingCutover, &route)
	// Bad state BY CONSTRUCTION: a record whose cutover_ready is false, written
	// directly rather than produced by the control under test.
	mock.SetELMMigration(&models.ELMMigration{
		RepositoryID:   repo.ID,
		ELMMigrationID: "elm-123",
		ELMStatus:      "backfilling",
		CutoverReady:   false,
	})

	svc := &fakeELMService{onCutover: func(r *models.Repository) {
		r.Status = string(models.StatusCuttingOver)
	}}
	h := setupELMHandler(t, mock, svc)

	w := httptest.NewRecorder()
	h.TriggerCutover(w, postAction("octo/app", ""))

	if w.Code != http.StatusConflict {
		t.Fatalf("expected 409, got %d (body=%s)", w.Code, w.Body.String())
	}
	if reason := decodeBody(t, w)["reason"]; reason != elmReasonNotReady {
		t.Errorf("expected machine-readable reason %q, got %v", elmReasonNotReady, reason)
	}
	// COMMITTED STATE: no command dispatched, repository untouched.
	if len(svc.calls) != 0 {
		t.Errorf("expected ZERO cutover commands to be dispatched, got %v", svc.calls)
	}
	if got := mock.Repos["octo/app"].Status; got != string(models.StatusAwaitingCutover) {
		t.Errorf("expected the repository to stay %q, got %q", models.StatusAwaitingCutover, got)
	}
}

// TestELMHandlers_CutoverRefusedWithoutRecord covers the second refusal branch: a
// repository with no live migration at all.
func TestELMHandlers_CutoverRefusedWithoutRecord(t *testing.T) {
	mock := NewMockDataStore()
	seedRepo(mock, "octo/app", models.StatusPending, nil)
	svc := &fakeELMService{}
	h := setupELMHandler(t, mock, svc)

	w := httptest.NewRecorder()
	h.TriggerCutover(w, postAction("octo/app", ""))

	if w.Code != http.StatusConflict {
		t.Fatalf("expected 409, got %d (body=%s)", w.Code, w.Body.String())
	}
	if reason := decodeBody(t, w)["reason"]; reason != elmReasonNoRecord {
		t.Errorf("expected reason %q, got %v", elmReasonNoRecord, reason)
	}
	if len(svc.calls) != 0 {
		t.Errorf("expected ZERO cutover commands, got %v", svc.calls)
	}
}

// TestELMHandlers_CutoverWithoutServiceIsUnavailable covers the not-configured
// branch: readiness is satisfied but the deployment has no ELM service.
func TestELMHandlers_CutoverWithoutServiceIsUnavailable(t *testing.T) {
	mock := NewMockDataStore()
	route := string(models.MigrationRouteELM)
	repo := seedRepo(mock, "octo/app", models.StatusAwaitingCutover, &route)
	mock.SetELMMigration(&models.ELMMigration{
		RepositoryID: repo.ID, ELMMigrationID: "elm-123", CutoverReady: true,
	})
	h := setupELMHandler(t, mock, nil)

	w := httptest.NewRecorder()
	h.TriggerCutover(w, postAction("octo/app", ""))

	if w.Code != http.StatusServiceUnavailable {
		t.Fatalf("expected 503, got %d (body=%s)", w.Code, w.Body.String())
	}
	if reason := decodeBody(t, w)["reason"]; reason != elmReasonNotConfigured {
		t.Errorf("expected reason %q, got %v", elmReasonNotConfigured, reason)
	}
	if got := mock.Repos["octo/app"].Status; got != string(models.StatusAwaitingCutover) {
		t.Errorf("expected the repository to stay %q, got %q", models.StatusAwaitingCutover, got)
	}
}

// TestELMHandlers_CutoverSurfacesServiceNotReady covers the INNER gate: the
// persisted record said ready but the service's fresh re-confirmation disagreed.
// That must still read as a 409 not-ready to the dashboard, not a 500.
func TestELMHandlers_CutoverSurfacesServiceNotReady(t *testing.T) {
	mock := NewMockDataStore()
	route := string(models.MigrationRouteELM)
	repo := seedRepo(mock, "octo/app", models.StatusAwaitingCutover, &route)
	mock.SetELMMigration(&models.ELMMigration{
		RepositoryID: repo.ID, ELMMigrationID: "elm-123", CutoverReady: true,
	})
	svc := &fakeELMService{err: fmt.Errorf("%w: stale record", migration.ErrELMNotReadyForCutover)}
	h := setupELMHandler(t, mock, svc)

	w := httptest.NewRecorder()
	h.TriggerCutover(w, postAction("octo/app", ""))

	if w.Code != http.StatusConflict {
		t.Fatalf("expected 409, got %d (body=%s)", w.Code, w.Body.String())
	}
	if reason := decodeBody(t, w)["reason"]; reason != elmReasonNotReady {
		t.Errorf("expected reason %q, got %v", elmReasonNotReady, reason)
	}
	if got := mock.Repos["octo/app"].Status; got != string(models.StatusAwaitingCutover) {
		t.Errorf("expected the repository to stay %q, got %q", models.StatusAwaitingCutover, got)
	}
}

func TestELMHandlers_CutoverServiceFailureIsInternal(t *testing.T) {
	mock := NewMockDataStore()
	route := string(models.MigrationRouteELM)
	repo := seedRepo(mock, "octo/app", models.StatusAwaitingCutover, &route)
	mock.SetELMMigration(&models.ELMMigration{
		RepositoryID: repo.ID, ELMMigrationID: "elm-123", CutoverReady: true,
	})
	svc := &fakeELMService{err: errors.New("ssh transport failed")}
	h := setupELMHandler(t, mock, svc)

	w := httptest.NewRecorder()
	h.TriggerCutover(w, postAction("octo/app", ""))

	if w.Code != http.StatusInternalServerError {
		t.Errorf("expected 500 for a transport failure, got %d (body=%s)", w.Code, w.Body.String())
	}
}

func TestELMHandlers_CutoverUnknownRepository(t *testing.T) {
	mock := NewMockDataStore()
	svc := &fakeELMService{}
	h := setupELMHandler(t, mock, svc)

	w := httptest.NewRecorder()
	h.TriggerCutover(w, postAction("octo/missing", ""))

	if w.Code != http.StatusNotFound {
		t.Errorf("expected 404, got %d (body=%s)", w.Code, w.Body.String())
	}
	if len(svc.calls) != 0 {
		t.Errorf("expected ZERO cutover commands, got %v", svc.calls)
	}
}

func TestELMHandlers_CutoverRecordLookupFailure(t *testing.T) {
	mock := NewMockDataStore().WithELMGetError(errors.New("db down"))
	seedRepo(mock, "octo/app", models.StatusAwaitingCutover, nil)
	svc := &fakeELMService{}
	h := setupELMHandler(t, mock, svc)

	w := httptest.NewRecorder()
	h.TriggerCutover(w, postAction("octo/app", ""))

	if w.Code != http.StatusInternalServerError {
		t.Errorf("expected 500 when the ELM record cannot be read, got %d", w.Code)
	}
	if len(svc.calls) != 0 {
		t.Errorf("expected ZERO cutover commands when readiness is unknown, got %v", svc.calls)
	}
}

// ============================================================================
// ELM status read
// ============================================================================

func TestGetELMStatus_ReportsRecord(t *testing.T) {
	mock := NewMockDataStore()
	route := string(models.MigrationRouteELM)
	repo := seedRepo(mock, "octo/app", models.StatusSyncing, &route)
	polled := time.Date(2026, 8, 6, 12, 0, 0, 0, time.UTC)
	progress := 61
	mock.SetELMMigration(&models.ELMMigration{
		RepositoryID:    repo.ID,
		ELMMigrationID:  "elm-123",
		ELMStatus:       "backfilling",
		ELMPhase:        "git",
		ProgressPercent: &progress,
		CutoverReady:    false,
		LastPolledAt:    &polled,
	})
	h := setupELMHandler(t, mock, nil)

	w := httptest.NewRecorder()
	h.GetELMStatus(w, httptest.NewRequest(http.MethodGet, "/api/v1/repositories/octo/app/elm", nil), "octo/app")

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d (body=%s)", w.Code, w.Body.String())
	}
	var resp ELMStatusResponse
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}
	if !resp.HasELMMigration || resp.ELMMigrationID != "elm-123" {
		t.Errorf("expected the elm migration id to be reported, got %+v", resp)
	}
	if resp.MigrationRoute != string(models.MigrationRouteELM) {
		t.Errorf("expected route elm, got %q", resp.MigrationRoute)
	}
	if resp.ProgressPercent == nil || *resp.ProgressPercent != progress {
		t.Errorf("expected progress %d, got %v", progress, resp.ProgressPercent)
	}
	if resp.CutoverReady {
		t.Error("expected cutover_ready false so the dashboard's Cut over button stays disabled")
	}
	if resp.LastPolledAt == nil || *resp.LastPolledAt != polled.Format(time.RFC3339) {
		t.Errorf("expected last_polled_at %q, got %v", polled.Format(time.RFC3339), resp.LastPolledAt)
	}
}

// TestGetELMStatus_UnroutedRepository proves the read reports the GEI default for
// a repository that has never been routed, rather than erroring.
func TestGetELMStatus_UnroutedRepository(t *testing.T) {
	mock := NewMockDataStore()
	seedRepo(mock, "octo/app", models.StatusPending, nil)
	h := setupELMHandler(t, mock, nil)

	w := httptest.NewRecorder()
	h.GetELMStatus(w, httptest.NewRequest(http.MethodGet, "/api/v1/repositories/octo/app/elm", nil), "octo/app")

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d (body=%s)", w.Code, w.Body.String())
	}
	var resp ELMStatusResponse
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}
	if resp.MigrationRoute != string(models.MigrationRouteGEI) {
		t.Errorf("expected the unrouted default %q, got %q", models.MigrationRouteGEI, resp.MigrationRoute)
	}
	if resp.HasELMMigration || resp.CutoverReady {
		t.Errorf("expected no live migration to be reported, got %+v", resp)
	}
}

func TestGetELMStatus_UnknownRepository(t *testing.T) {
	mock := NewMockDataStore()
	h := setupELMHandler(t, mock, nil)

	w := httptest.NewRecorder()
	h.GetELMStatus(w, httptest.NewRequest(http.MethodGet, "/api/v1/repositories/octo/missing/elm", nil), "octo/missing")

	if w.Code != http.StatusNotFound {
		t.Errorf("expected 404, got %d", w.Code)
	}
}

// TestGetELMStatus_DecodesRepositoryName proves the read path applies the same
// URL-decoding the POST actions get.
func TestGetELMStatus_DecodesRepositoryName(t *testing.T) {
	mock := NewMockDataStore()
	seedRepo(mock, "octo/my app", models.StatusPending, nil)
	h := setupELMHandler(t, mock, nil)

	w := httptest.NewRecorder()
	h.GetELMStatus(w, httptest.NewRequest(http.MethodGet, "/api/v1/repositories/octo/my%20app/elm", nil), "octo/my%20app")

	if w.Code != http.StatusOK {
		t.Fatalf("expected the encoded name to resolve, got %d (body=%s)", w.Code, w.Body.String())
	}
}

// ============================================================================
// Action-suffix routing
// ============================================================================

// TestHandleRepositoryAction_ELMSuffixes proves the two new POST suffixes are
// dispatched through the existing HandleRepositoryAction parsing, so they inherit
// its URL-decode and CheckRepositoryAccess flow.
func TestHandleRepositoryAction_ELMSuffixes(t *testing.T) {
	tests := []struct {
		name     string
		suffix   string
		body     string
		wantCode int
	}{
		{name: "migration-route", suffix: "/migration-route", body: `{"route":"elm"}`, wantCode: http.StatusOK},
		{name: "cutover", suffix: "/cutover", body: "", wantCode: http.StatusConflict},
		{name: "unknown action still 404s", suffix: "/teleport", body: "", wantCode: http.StatusNotFound},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mock := NewMockDataStore()
			seedRepo(mock, "octo/app", models.StatusPending, nil)
			h := setupELMHandler(t, mock, &fakeELMService{})

			req := httptest.NewRequest(http.MethodPost, "/api/v1/repositories/octo/app"+tt.suffix, strings.NewReader(tt.body))
			req.SetPathValue("fullName", "octo/app"+tt.suffix)
			w := httptest.NewRecorder()
			h.HandleRepositoryAction(w, req)

			if w.Code != tt.wantCode {
				t.Errorf("expected %d, got %d (body=%s)", tt.wantCode, w.Code, w.Body.String())
			}
		})
	}
}

// TestHandleRepositoryAction_ELMSuffixesDecodeNames proves an encoded repository
// name reaches the ELM handlers decoded.
func TestHandleRepositoryAction_ELMSuffixesDecodeNames(t *testing.T) {
	mock := NewMockDataStore()
	seedRepo(mock, "octo/my app", models.StatusPending, nil)
	h := setupELMHandler(t, mock, &fakeELMService{})

	req := httptest.NewRequest(http.MethodPost, "/api/v1/repositories/octo/my%20app/migration-route", strings.NewReader(`{"route":"elm"}`))
	req.SetPathValue("fullName", "octo/my%20app/migration-route")
	w := httptest.NewRecorder()
	h.HandleRepositoryAction(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d (body=%s)", w.Code, w.Body.String())
	}
	stored := mock.Repos["octo/my app"]
	if stored.MigrationRoute == nil || *stored.MigrationRoute != string(models.MigrationRouteELM) {
		t.Errorf("expected the decoded repository to be routed to elm, got %v", stored.MigrationRoute)
	}
}
