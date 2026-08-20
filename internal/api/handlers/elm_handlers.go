package handlers

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/url"
	"sync"
	"time"

	"github.com/kuhlman-labs/github-migrator/internal/migration"
	"github.com/kuhlman-labs/github-migrator/internal/models"
)

// ELMStore is the narrow slice of storage the ELM handlers need: the ELM
// migration record and the writer for the one migration_route column.
//
// It is deliberately narrower than DataStore. h.db is resolved to it with a type
// assertion (elmStore below) so a Handler built over a data store that knows
// nothing about ELM degrades to 503 rather than failing to compile.
type ELMStore interface {
	GetELMMigration(ctx context.Context, repoID int64) (*models.ELMMigration, error)
	UpdateRepositoryMigrationRoute(ctx context.Context, fullName string, route *string) error
}

// ELMCutoverService is the narrow slice of migration.ELMService the cutover
// handler needs. Cutover is the one irreversible step of a live migration, so the
// service re-checks readiness itself: the handler's 409 below is the OUTER of two
// independent gates, not the only one.
type ELMCutoverService interface {
	Cutover(ctx context.Context, repo *models.Repository) error
}

// elmServices associates an optional ELM service with a Handler.
//
// It lives here rather than as a Handler field because the ELM feature is
// optional wiring layered onto the shared Handler: a deployment with ELM disabled
// registers nothing and every ELM endpoint answers 503. Keyed by *Handler, which
// is created once per server.
var elmServices sync.Map // *Handler -> ELMCutoverService

// SetELMService registers the ELM service used by the cutover endpoint. Passing a
// nil service leaves ELM unconfigured, which is the default.
func SetELMService(h *Handler, svc ELMCutoverService) {
	if h == nil {
		return
	}
	if svc == nil {
		elmServices.Delete(h)
		return
	}
	elmServices.Store(h, svc)
}

// elmService returns the registered ELM service, or false when ELM is not
// configured for this deployment.
func (h *Handler) elmService() (ELMCutoverService, bool) {
	v, ok := elmServices.Load(h)
	if !ok {
		return nil, false
	}
	svc, ok := v.(ELMCutoverService)
	return svc, ok && svc != nil
}

// elmStore returns the data store as an ELMStore, or false when the configured
// data store does not implement the ELM operations.
func (h *Handler) elmStore() (ELMStore, bool) {
	store, ok := h.db.(ELMStore)
	return store, ok
}

// actionCutover is the /cutover repository action suffix, shared by the suffix
// parser and the dispatch switch in HandleRepositoryAction.
const actionCutover = "cutover"

// Machine-readable refusal reasons returned alongside an ELM error response, so
// the dashboard can branch on the cause without parsing prose.
const (
	elmReasonNotReady      = "elm_not_ready"
	elmReasonNoRecord      = "elm_no_record"
	elmReasonNotConfigured = "elm_not_configured"
	elmReasonInvalidRoute  = "invalid_migration_route"
)

// elmErrorResponse is the body of a refused ELM request. It carries the same
// error/details fields as APIError plus a stable machine-readable reason.
type elmErrorResponse struct {
	Error   string `json:"error"`
	Details string `json:"details,omitempty"`
	Reason  string `json:"reason"`
}

// SetMigrationRouteRequest is the body of POST /api/v1/repositories/{fullName...}/migration-route.
//
// Route is a pointer so an explicit JSON null clears the route back to the GEI
// default, which is distinguishable from an absent field.
type SetMigrationRouteRequest struct {
	Route *string `json:"route"`
}

// MigrationRouteResponse is returned after a successful route change.
type MigrationRouteResponse struct {
	FullName       string `json:"full_name"`
	MigrationRoute string `json:"migration_route"`
	Message        string `json:"message"`
}

// ELMStatusResponse is the ELM record as the dashboard consumes it.
type ELMStatusResponse struct {
	FullName        string  `json:"full_name"`
	MigrationRoute  string  `json:"migration_route"`
	Status          string  `json:"status"`
	HasELMMigration bool    `json:"has_elm_migration"`
	ELMMigrationID  string  `json:"elm_migration_id,omitempty"`
	ELMStatus       string  `json:"elm_status,omitempty"`
	ELMPhase        string  `json:"elm_phase,omitempty"`
	ProgressPercent *int    `json:"progress_percent,omitempty"`
	CutoverReady    bool    `json:"cutover_ready"`
	LastPolledAt    *string `json:"last_polled_at,omitempty"`
	LastError       *string `json:"last_error,omitempty"`
}

// SetMigrationRoute handles POST /api/v1/repositories/{fullName...}/migration-route.
//
// This is the OPERATOR half of the ELM route contract -- the deliberate,
// human-driven writer of the same repositories.migration_route column the
// discovery analyzer writes and ELMStrategy reads. Only the two legal route
// values are accepted; anything else is refused with 400 and the stored route is
// left exactly as it was.
func (h *Handler) SetMigrationRoute(w http.ResponseWriter, r *http.Request) {
	fullName, err := h.getDecodedRepoName(r)
	if err != nil {
		WriteError(w, ErrMissingField.WithField("fullName"))
		return
	}

	var req SetMigrationRouteRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		WriteError(w, ErrInvalidJSON.WithDetails(err.Error()))
		return
	}

	// Reject anything that is not one of the two legal routes BEFORE touching
	// storage, so a rejected request commits nothing.
	if req.Route != nil && *req.Route != "" && !models.IsValidMigrationRoute(*req.Route) {
		h.sendJSON(w, http.StatusBadRequest, elmErrorResponse{
			Error: "Invalid migration route",
			Details: "route must be \"" + string(models.MigrationRouteGEI) +
				"\" or \"" + string(models.MigrationRouteELM) + "\"",
			Reason: elmReasonInvalidRoute,
		})
		return
	}

	store, ok := h.elmStore()
	if !ok {
		WriteError(w, ErrServiceUnavailable.WithDetails("migration routes are not available on this data store"))
		return
	}

	// An empty string is normalised to nil: both clear the route back to the GEI
	// default rather than storing a meaningless empty value.
	route := req.Route
	if route != nil && *route == "" {
		route = nil
	}

	if err := store.UpdateRepositoryMigrationRoute(r.Context(), fullName, route); err != nil {
		h.logger.Error("Failed to update migration route", "repo", fullName, "error", err)
		WriteError(w, ErrDatabaseUpdate.WithDetails(err.Error()))
		return
	}

	applied := string(models.MigrationRouteGEI)
	if route != nil {
		applied = *route
	}

	h.logger.Info("Migration route updated", "repo", fullName, "migration_route", applied)
	h.sendJSON(w, http.StatusOK, MigrationRouteResponse{
		FullName:       fullName,
		MigrationRoute: applied,
		Message:        "Migration route updated",
	})
}

// TriggerCutover handles POST /api/v1/repositories/{fullName...}/cutover.
//
// Cutover is the single irreversible, downtime-causing step of a live migration:
// ELM archives the source repository, and recovery is a manual operator runbook.
// It is therefore reachable ONLY from this operator-triggered path -- nothing
// transitions into cutover automatically -- and it is refused with 409 and a
// machine-readable reason unless the PERSISTED record reports readiness. A refusal
// dispatches no cutover command.
func (h *Handler) TriggerCutover(w http.ResponseWriter, r *http.Request) {
	fullName, err := h.getDecodedRepoName(r)
	if err != nil {
		WriteError(w, ErrMissingField.WithField("fullName"))
		return
	}

	repo, err := h.db.GetRepository(r.Context(), fullName)
	if err != nil {
		h.logger.Error("Failed to load repository for cutover", "repo", fullName, "error", err)
		WriteError(w, ErrDatabaseFetch.WithDetails(err.Error()))
		return
	}
	if repo == nil {
		WriteError(w, ErrRepositoryNotFound.WithDetails(fullName))
		return
	}

	store, ok := h.elmStore()
	if !ok {
		WriteError(w, ErrServiceUnavailable.WithDetails("live migrations are not available on this data store"))
		return
	}

	rec, err := store.GetELMMigration(r.Context(), repo.ID)
	if err != nil {
		h.logger.Error("Failed to load ELM migration record", "repo", fullName, "error", err)
		WriteError(w, ErrDatabaseFetch.WithDetails(err.Error()))
		return
	}

	// GATE: refuse unless the persisted record reports readiness. This runs before
	// the service is even consulted, so an unready repository never reaches the
	// cutover command.
	if rec == nil {
		h.sendJSON(w, http.StatusConflict, elmErrorResponse{
			Error:   "Repository has no live migration",
			Details: fullName + " is not being migrated with Enterprise Live Migrations",
			Reason:  elmReasonNoRecord,
		})
		return
	}
	if !rec.CutoverReady {
		h.sendJSON(w, http.StatusConflict, elmErrorResponse{
			Error:   "Live migration is not ready for cutover",
			Details: "the backfill for " + fullName + " reports elm status \"" + rec.ELMStatus + "\"",
			Reason:  elmReasonNotReady,
		})
		return
	}

	svc, ok := h.elmService()
	if !ok {
		h.sendJSON(w, http.StatusServiceUnavailable, elmErrorResponse{
			Error:   "Live migrations are not configured",
			Details: "no ELM service is configured for this deployment",
			Reason:  elmReasonNotConfigured,
		})
		return
	}

	// The service re-checks readiness against the persisted record AND a fresh
	// status call before issuing the command; --force is structurally unreachable.
	if err := svc.Cutover(r.Context(), repo); err != nil {
		if errors.Is(err, migration.ErrELMNotReadyForCutover) {
			h.sendJSON(w, http.StatusConflict, elmErrorResponse{
				Error:   "Live migration is not ready for cutover",
				Details: err.Error(),
				Reason:  elmReasonNotReady,
			})
			return
		}
		h.logger.Error("ELM cutover failed", "repo", fullName, "error", err)
		WriteError(w, ErrInternal.WithDetails(err.Error()))
		return
	}

	updated, err := h.db.GetRepository(r.Context(), fullName)
	if err != nil || updated == nil {
		// The cutover was issued; report success even if the re-read failed.
		h.logger.Warn("Cutover issued but repository re-read failed", "repo", fullName, "error", err)
		updated = repo
	}

	h.logger.Info("ELM cutover triggered", "repo", fullName, "status", updated.Status)
	h.sendJSON(w, http.StatusOK, map[string]any{
		"message":    "Cutover started",
		"repository": updated,
	})
}

// GetELMStatus handles GET /api/v1/repositories/{fullName...}/elm.
//
// It reports the route and, when a live migration exists, its phase, progress and
// cutover readiness -- the read the dashboard's ELM section renders and the source
// of truth for whether its 'Cut over' button is enabled.
func (h *Handler) GetELMStatus(w http.ResponseWriter, r *http.Request, fullName string) {
	decoded, err := url.QueryUnescape(fullName)
	if err != nil {
		h.logger.Warn("Failed to decode repository name", "fullName", fullName, "error", err)
		decoded = fullName
	}

	if err := h.CheckRepositoryAccess(r.Context(), decoded); err != nil {
		h.logger.Warn("Repository access denied", "repo", decoded, "action", "elm-status", "error", err)
		WriteError(w, ErrForbidden.WithDetails("insufficient permissions for this repository"))
		return
	}

	repo, err := h.db.GetRepository(r.Context(), decoded)
	if err != nil {
		h.logger.Error("Failed to load repository for ELM status", "repo", decoded, "error", err)
		WriteError(w, ErrDatabaseFetch.WithDetails(err.Error()))
		return
	}
	if repo == nil {
		WriteError(w, ErrRepositoryNotFound.WithDetails(decoded))
		return
	}

	resp := ELMStatusResponse{
		FullName:       repo.FullName,
		MigrationRoute: repo.GetMigrationRoute(),
		Status:         repo.Status,
	}

	store, ok := h.elmStore()
	if !ok {
		// No ELM storage: report the route only. A repository can be routed
		// without a live migration having started yet, so this is not an error.
		h.sendJSON(w, http.StatusOK, resp)
		return
	}

	rec, err := store.GetELMMigration(r.Context(), repo.ID)
	if err != nil {
		h.logger.Error("Failed to load ELM migration record", "repo", decoded, "error", err)
		WriteError(w, ErrDatabaseFetch.WithDetails(err.Error()))
		return
	}
	if rec != nil {
		resp.HasELMMigration = true
		resp.ELMMigrationID = rec.ELMMigrationID
		resp.ELMStatus = rec.ELMStatus
		resp.ELMPhase = rec.ELMPhase
		resp.ProgressPercent = rec.ProgressPercent
		resp.CutoverReady = rec.CutoverReady
		resp.LastError = rec.LastError
		if rec.LastPolledAt != nil {
			polled := rec.LastPolledAt.UTC().Format(time.RFC3339)
			resp.LastPolledAt = &polled
		}
	}

	h.sendJSON(w, http.StatusOK, resp)
}
