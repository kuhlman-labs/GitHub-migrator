package elm

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"
)

// commandEnvPrefix is the environment every elm invocation needs.
//
// API_URL is fixed at the loopback address because the ELM exporter is bound to
// the appliance itself. MIGRATION_MANAGER_HMAC_KEY is deliberately a command
// substitution rather than an interpolated value: the key is resolved by
// ghe-config INSIDE the admin shell and never transits this process, so it can
// neither be logged nor persisted here. Both are constants, not caller input.
const commandEnvPrefix = `API_URL=http://localhost:1738 MIGRATION_MANAGER_HMAC_KEY="$(ghe-config secrets.elm-exporter.elm-exporter-hmac-keys)"`

// The elm verbs this wrapper knows how to build.
const (
	verbCreate  = "migration create"
	verbStart   = "migration start"
	verbStatus  = "migration status"
	verbList    = "migration list"
	verbCutover = "migration cutover-to-destination"
	verbCancel  = "migration cancel"
)

// Flags. Every one of these is a compile-time constant; no flag name is ever
// built from caller input.
const (
	flagSourceOrg   = "--source-org"
	flagSourceRepo  = "--source-repo"
	flagTargetOrg   = "--target-org"
	flagTargetRepo  = "--target-repo"
	flagTargetAPI   = "--target-api-url"
	flagVisibility  = "--visibility"
	flagPATName     = "--pat-name"
	flagMigrationID = "--migration-id"
	flagPage        = "--page"
)

// forceFlag is named here only so the guard below can refuse it. Cutover is the
// one irreversible step in a live migration and this tool never forces it.
const forceFlag = "--force"

// verbFlags is the allowlist of flags each verb may carry. buildCommand refuses
// anything outside it, which is what makes --force structurally unreachable on
// cutover: the cutover entry permits the migration id and nothing else, and
// there is no code path that adds a flag from caller input.
var verbFlags = map[string]map[string]bool{
	verbCreate: {
		flagSourceOrg:  true,
		flagSourceRepo: true,
		flagTargetOrg:  true,
		flagTargetRepo: true,
		flagTargetAPI:  true,
		flagVisibility: true,
		flagPATName:    true,
	},
	verbStart:   {flagMigrationID: true},
	verbStatus:  {flagMigrationID: true},
	verbList:    {flagPage: true},
	verbCutover: {flagMigrationID: true},
	verbCancel:  {flagMigrationID: true},
}

// commandArg is a flag and its value. The value is ALWAYS shell-quoted.
type commandArg struct {
	flag  string
	value string
}

// ClientConfig is the non-connection half of the elm wrapper configuration.
type ClientConfig struct {
	// TargetAPIURL is the destination (GHE.com data residency) API endpoint.
	TargetAPIURL string
	// PATName is the named credential the appliance uses; the preview CLI
	// documents `system-pat`.
	PATName string
}

// DefaultPATName is the credential name the ELM preview documents.
const DefaultPATName = "system-pat"

// Client is a typed wrapper over the elm CLI surface.
type Client struct {
	transport Transport
	cfg       ClientConfig
}

// NewClient returns a Client bound to transport.
func NewClient(transport Transport, cfg ClientConfig) (*Client, error) {
	if transport == nil {
		return nil, fmt.Errorf("elm: transport is required")
	}
	if strings.TrimSpace(cfg.TargetAPIURL) == "" {
		return nil, fmt.Errorf("elm: target API URL is required")
	}
	if strings.TrimSpace(cfg.PATName) == "" {
		cfg.PATName = DefaultPATName
	}
	return &Client{transport: transport, cfg: cfg}, nil
}

// shellQuote wraps s in single quotes and escapes any embedded single quote, so
// the value reaches elm as one literal argument no matter what it contains.
// Every interpolated value in every command goes through this, without exception.
func shellQuote(s string) string {
	return "'" + strings.ReplaceAll(s, "'", `'\''`) + "'"
}

// buildCommand assembles one elm invocation. It is the single place a command
// string is produced, so the environment prefix, the flag allowlist and the
// shell quoting apply uniformly.
func buildCommand(verb string, args ...commandArg) (string, error) {
	allowed, ok := verbFlags[verb]
	if !ok {
		return "", fmt.Errorf("elm: unknown verb %q", verb)
	}

	var b strings.Builder
	b.WriteString(commandEnvPrefix)
	b.WriteString(" elm ")
	b.WriteString(verb)

	for _, a := range args {
		if a.flag == forceFlag {
			return "", fmt.Errorf("elm: %s is never emitted", forceFlag)
		}
		if !allowed[a.flag] {
			return "", fmt.Errorf("elm: flag %q is not permitted for verb %q", a.flag, verb)
		}
		b.WriteString(" ")
		b.WriteString(a.flag)
		b.WriteString(" ")
		b.WriteString(shellQuote(a.value))
	}

	return b.String(), nil
}

// run executes command and maps every failure onto exactly one of the three
// typed errors this package exposes.
func (c *Client) run(ctx context.Context, command string) (string, error) {
	stdout, stderr, err := c.transport.Run(ctx, command)
	if err != nil {
		var exitErr *ExitError
		if errors.As(err, &exitErr) {
			return stdout, &CommandError{Command: command, ExitCode: exitErr.Code, Stderr: strings.TrimSpace(stderr)}
		}
		return stdout, &TransportError{Command: command, Err: err}
	}
	return stdout, nil
}

// parseTimestamp reads an optional RFC3339 timestamp.
func parseTimestamp(raw *string) (*time.Time, error) {
	if raw == nil || strings.TrimSpace(*raw) == "" {
		return nil, nil
	}
	parsed, err := time.Parse(time.RFC3339, strings.TrimSpace(*raw))
	if err != nil {
		return nil, fmt.Errorf("unreadable timestamp %q: %w", *raw, err)
	}
	return &parsed, nil
}

// CreateMigration creates (but does not start) a live migration.
func (c *Client) CreateMigration(ctx context.Context, req CreateMigrationRequest) (*MigrationRef, error) {
	command, err := buildCommand(verbCreate,
		commandArg{flagSourceOrg, req.SourceOrg},
		commandArg{flagSourceRepo, req.SourceRepo},
		commandArg{flagTargetOrg, req.TargetOrg},
		commandArg{flagTargetRepo, req.TargetRepo},
		commandArg{flagTargetAPI, c.cfg.TargetAPIURL},
		commandArg{flagVisibility, req.Visibility},
		commandArg{flagPATName, c.cfg.PATName},
	)
	if err != nil {
		return nil, err
	}

	stdout, err := c.run(ctx, command)
	if err != nil {
		return nil, err
	}

	var payload struct {
		ID string `json:"id"`
	}
	if err := json.Unmarshal([]byte(strings.TrimSpace(stdout)), &payload); err != nil {
		return nil, &ParseError{Command: command, Output: stdout, Err: err}
	}
	if strings.TrimSpace(payload.ID) == "" {
		return nil, &ParseError{Command: command, Output: stdout, Err: fmt.Errorf("no migration id in output")}
	}
	return &MigrationRef{ID: payload.ID}, nil
}

// StartMigration begins the backfill for an already-created migration.
func (c *Client) StartMigration(ctx context.Context, migrationID string) error {
	command, err := buildCommand(verbStart, commandArg{flagMigrationID, migrationID})
	if err != nil {
		return err
	}
	_, err = c.run(ctx, command)
	return err
}

// GetMigrationStatus reads the current state of one migration.
func (c *Client) GetMigrationStatus(ctx context.Context, migrationID string) (*MigrationStatus, error) {
	command, err := buildCommand(verbStatus, commandArg{flagMigrationID, migrationID})
	if err != nil {
		return nil, err
	}

	stdout, err := c.run(ctx, command)
	if err != nil {
		return nil, err
	}
	return parseStatus(command, stdout)
}

// parseStatus reads `elm migration status` output.
//
// Readiness is never inferred optimistically: an absent or unrecognised state is
// a ParseError, and when the CLI does not report cutover_ready explicitly the
// value is derived only from a state we recognise. There is no path on which
// unreadable output yields CutoverReady == true.
func parseStatus(command, stdout string) (*MigrationStatus, error) {
	var payload struct {
		ID              *string `json:"id"`
		State           *string `json:"state"`
		Phase           string  `json:"phase"`
		ProgressPercent *int    `json:"progress_percent"`
		CutoverReady    *bool   `json:"cutover_ready"`
		CreatedAt       *string `json:"created_at"`
		UpdatedAt       *string `json:"updated_at"`
	}
	if err := json.Unmarshal([]byte(strings.TrimSpace(stdout)), &payload); err != nil {
		return nil, &ParseError{Command: command, Output: stdout, Err: err}
	}
	if payload.ID == nil || strings.TrimSpace(*payload.ID) == "" {
		return nil, &ParseError{Command: command, Output: stdout, Err: fmt.Errorf("no migration id in output")}
	}
	if payload.State == nil {
		return nil, &ParseError{Command: command, Output: stdout, Err: fmt.Errorf("no state in output")}
	}
	state := strings.TrimSpace(*payload.State)
	if !knownStates[state] {
		return nil, &ParseError{Command: command, Output: stdout, Err: fmt.Errorf("unrecognised state %q", state)}
	}

	status := &MigrationStatus{
		ID:              *payload.ID,
		State:           state,
		Phase:           payload.Phase,
		ProgressPercent: payload.ProgressPercent,
	}
	if payload.CutoverReady != nil {
		status.CutoverReady = *payload.CutoverReady
	} else {
		status.CutoverReady = state == StateReadyForCutover
	}

	var err error
	if status.CreatedAt, err = parseTimestamp(payload.CreatedAt); err != nil {
		return nil, &ParseError{Command: command, Output: stdout, Err: err}
	}
	if status.UpdatedAt, err = parseTimestamp(payload.UpdatedAt); err != nil {
		return nil, &ParseError{Command: command, Output: stdout, Err: err}
	}
	return status, nil
}

// ListMigrations returns one page of migrations known to the appliance. Pass an
// empty cursor for the first page.
func (c *Client) ListMigrations(ctx context.Context, cursor string) (*ListPage, error) {
	args := []commandArg{}
	if cursor != "" {
		args = append(args, commandArg{flagPage, cursor})
	}

	command, err := buildCommand(verbList, args...)
	if err != nil {
		return nil, err
	}

	stdout, err := c.run(ctx, command)
	if err != nil {
		return nil, err
	}

	var page ListPage
	if err := json.Unmarshal([]byte(strings.TrimSpace(stdout)), &page); err != nil {
		return nil, &ParseError{Command: command, Output: stdout, Err: err}
	}
	for _, m := range page.Migrations {
		if strings.TrimSpace(m.ID) == "" {
			return nil, &ParseError{Command: command, Output: stdout, Err: fmt.Errorf("migration entry without an id")}
		}
	}
	return &page, nil
}

// CutoverToDestination performs the irreversible cutover for one migration.
//
// It takes no force parameter, and buildCommand refuses --force for this verb,
// so no caller, config key or API field in this codebase can reach it.
func (c *Client) CutoverToDestination(ctx context.Context, migrationID string) error {
	command, err := c.CutoverCommand(migrationID)
	if err != nil {
		return err
	}
	_, err = c.run(ctx, command)
	return err
}

// CutoverCommand exposes the exact cutover command string so callers (and tests)
// can inspect it without executing it.
func (c *Client) CutoverCommand(migrationID string) (string, error) {
	return buildCommand(verbCutover, commandArg{flagMigrationID, migrationID})
}

// CancelMigration aborts an in-flight migration. The source stays writable, so
// cancelling before cutover has no impact on the source repository.
func (c *Client) CancelMigration(ctx context.Context, migrationID string) error {
	command, err := buildCommand(verbCancel, commandArg{flagMigrationID, migrationID})
	if err != nil {
		return err
	}
	_, err = c.run(ctx, command)
	return err
}
