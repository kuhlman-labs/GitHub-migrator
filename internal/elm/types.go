// Package elm wraps the GitHub Enterprise Live Migrations (ELM) command line
// surface that ships inside the GHES administrative shell.
//
// The ELM exporter is only reachable from the appliance itself: its API is bound
// to http://localhost:1738 and its HMAC key is readable only through ghe-config.
// Everything in this package therefore goes through the Transport seam, which in
// production is an SSH connection into the admin shell and in tests is a fake.
//
// ELM is in public preview, so the exact command surface parsed here is pinned by
// tests rather than assumed to be stable. Unrecognised output is reported as a
// typed parse error rather than silently coerced into a default -- in particular
// cutover readiness is never inferred to be true from output we cannot read.
package elm

import (
	"fmt"
	"time"
)

// Migration lifecycle states reported by the ELM CLI.
const (
	StateQueued          = "queued"
	StateBackfilling     = "backfilling"
	StateReadyForCutover = "ready_for_cutover"
	StateCuttingOver     = "cutting_over"
	StateCompleted       = "completed"
	StateFailed          = "failed"
	StateCanceled        = "canceled"
)

// knownStates enumerates every state this wrapper understands. Output carrying a
// state outside this set is a parse error, not an unknown-but-tolerated value:
// a preview CLI whose vocabulary drifts must produce a red test rather than a
// migration whose readiness we guess at.
var knownStates = map[string]bool{
	StateQueued:          true,
	StateBackfilling:     true,
	StateReadyForCutover: true,
	StateCuttingOver:     true,
	StateCompleted:       true,
	StateFailed:          true,
	StateCanceled:        true,
}

// MigrationRef identifies a live migration created on the appliance.
type MigrationRef struct {
	ID string `json:"id"`
}

// MigrationStatus is the parsed result of `elm migration status`.
type MigrationStatus struct {
	ID              string     `json:"id"`
	State           string     `json:"state"`
	Phase           string     `json:"phase,omitempty"`
	ProgressPercent *int       `json:"progress_percent,omitempty"`
	CutoverReady    bool       `json:"cutover_ready"`
	CreatedAt       *time.Time `json:"created_at,omitempty"`
	UpdatedAt       *time.Time `json:"updated_at,omitempty"`
}

// MigrationSummary is one entry of `elm migration list`.
type MigrationSummary struct {
	ID           string `json:"id"`
	State        string `json:"state"`
	SourceOrg    string `json:"source_org,omitempty"`
	SourceRepo   string `json:"source_repo,omitempty"`
	TargetOrg    string `json:"target_org,omitempty"`
	TargetRepo   string `json:"target_repo,omitempty"`
	CutoverReady bool   `json:"cutover_ready"`
}

// ListPage is one page of `elm migration list`. NextPage is empty on the last page.
type ListPage struct {
	Migrations []MigrationSummary `json:"migrations"`
	NextPage   string             `json:"next_page,omitempty"`
}

// CreateMigrationRequest describes the repository pair to live-migrate.
type CreateMigrationRequest struct {
	SourceOrg  string
	SourceRepo string
	TargetOrg  string
	TargetRepo string
	Visibility string
}

// The three failure modes of every ELM command are distinct types so callers can
// react differently: a transport failure is retryable and must not touch
// migration state, a non-zero exit is an appliance-reported failure, and an
// unparseable payload means the preview CLI drifted.

// TransportError reports that the command never ran to completion -- the SSH
// connection failed, the session could not be opened, or the context expired.
type TransportError struct {
	Command string
	Err     error
}

func (e *TransportError) Error() string {
	return fmt.Sprintf("elm transport failure running %q: %v", e.Command, e.Err)
}

func (e *TransportError) Unwrap() error { return e.Err }

// CommandError reports that the elm CLI ran and exited non-zero.
type CommandError struct {
	Command  string
	ExitCode int
	Stderr   string
}

func (e *CommandError) Error() string {
	return fmt.Sprintf("elm command %q exited %d: %s", e.Command, e.ExitCode, e.Stderr)
}

// ParseError reports that the elm CLI exited zero but produced output this
// wrapper cannot read. Callers MUST NOT treat this as "not ready yet" or as a
// stalled sync; it means the contract changed.
type ParseError struct {
	Command string
	Output  string
	Err     error
}

func (e *ParseError) Error() string {
	return fmt.Sprintf("elm command %q produced unparseable output: %v", e.Command, e.Err)
}

func (e *ParseError) Unwrap() error { return e.Err }
