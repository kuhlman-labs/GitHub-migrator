package elm

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"testing"
)

// fakeTransport is the seam every downstream test fakes. It records every
// command it is asked to run and replays a scripted response.
type fakeTransport struct {
	commands []string
	respond  func(command string) (stdout, stderr string, err error)
	closed   bool
}

func (f *fakeTransport) Run(_ context.Context, command string) (string, string, error) {
	f.commands = append(f.commands, command)
	if f.respond == nil {
		return "{}", "", nil
	}
	return f.respond(command)
}

func (f *fakeTransport) Close() error {
	f.closed = true
	return nil
}

func newTestClient(t *testing.T, transport Transport) *Client {
	t.Helper()
	c, err := NewClient(transport, ClientConfig{TargetAPIURL: "https://api.acme.ghe.com"})
	if err != nil {
		t.Fatalf("NewClient: %v", err)
	}
	return c
}

func TestNewClient_RequiresTransportAndTarget(t *testing.T) {
	if _, err := NewClient(nil, ClientConfig{TargetAPIURL: "https://api.acme.ghe.com"}); err == nil {
		t.Error("expected NewClient to refuse a nil transport")
	}
	if _, err := NewClient(&fakeTransport{}, ClientConfig{}); err == nil {
		t.Error("expected NewClient to refuse an empty target API URL")
	}
	c, err := NewClient(&fakeTransport{}, ClientConfig{TargetAPIURL: "https://api.acme.ghe.com"})
	if err != nil {
		t.Fatalf("NewClient: %v", err)
	}
	if c.cfg.PATName != DefaultPATName {
		t.Errorf("PATName = %q, want the documented default %q", c.cfg.PATName, DefaultPATName)
	}
}

// TestELMClient_CutoverCommand pins the exact emitted command string, so a
// comment-only or no-op edit to the builder cannot pass.
func TestELMClient_CutoverCommand(t *testing.T) {
	fake := &fakeTransport{}
	client := newTestClient(t, fake)

	if err := client.CutoverToDestination(context.Background(), "mig-123"); err != nil {
		t.Fatalf("CutoverToDestination: %v", err)
	}

	want := `API_URL=http://localhost:1738 MIGRATION_MANAGER_HMAC_KEY="$(ghe-config secrets.elm-exporter.elm-exporter-hmac-keys)" ` +
		`elm migration cutover-to-destination --migration-id 'mig-123'`
	if len(fake.commands) != 1 {
		t.Fatalf("expected exactly one command, got %v", fake.commands)
	}
	if fake.commands[0] != want {
		t.Errorf("cutover command\n got: %s\nwant: %s", fake.commands[0], want)
	}
}

func TestELMClient_CreateCommand(t *testing.T) {
	fake := &fakeTransport{respond: func(string) (string, string, error) {
		return `{"id":"mig-abc"}`, "", nil
	}}
	client := newTestClient(t, fake)

	ref, err := client.CreateMigration(context.Background(), CreateMigrationRequest{
		SourceOrg:  "acme",
		SourceRepo: "monolith",
		TargetOrg:  "acme-eu",
		TargetRepo: "monolith",
		Visibility: "private",
	})
	if err != nil {
		t.Fatalf("CreateMigration: %v", err)
	}
	if ref.ID != "mig-abc" {
		t.Errorf("id = %q, want mig-abc", ref.ID)
	}

	want := `API_URL=http://localhost:1738 MIGRATION_MANAGER_HMAC_KEY="$(ghe-config secrets.elm-exporter.elm-exporter-hmac-keys)" ` +
		`elm migration create --source-org 'acme' --source-repo 'monolith' --target-org 'acme-eu' --target-repo 'monolith' ` +
		`--target-api-url 'https://api.acme.ghe.com' --visibility 'private' --pat-name 'system-pat'`
	if fake.commands[0] != want {
		t.Errorf("create command\n got: %s\nwant: %s", fake.commands[0], want)
	}
}

// TestELMClient_QuotesInterpolatedValues is the counterfactual vehicle for the
// shell-quoting helper. Every value is hostile by construction.
func TestELMClient_QuotesInterpolatedValues(t *testing.T) {
	hostile := []struct {
		name  string
		value string
		want  string
	}{
		{"command chaining", "; rm -rf /", `'; rm -rf /'`},
		{"spaces", "my org", `'my org'`},
		{"command substitution", "$(id)", `'$(id)'`},
		{"backticks", "`id`", "'`id`'"},
		{"embedded single quote", "o'brien", `'o'\''brien'`},
		{"looks like a flag", "--force", `'--force'`},
	}

	for _, tt := range hostile {
		t.Run(tt.name, func(t *testing.T) {
			fake := &fakeTransport{respond: func(string) (string, string, error) {
				return `{"id":"mig-1"}`, "", nil
			}}
			client := newTestClient(t, fake)

			// The same hostile value in every interpolated position, so no clean
			// companion value can carry the assertion.
			if _, err := client.CreateMigration(context.Background(), CreateMigrationRequest{
				SourceOrg:  tt.value,
				SourceRepo: tt.value,
				TargetOrg:  tt.value,
				TargetRepo: tt.value,
				Visibility: tt.value,
			}); err != nil {
				t.Fatalf("CreateMigration: %v", err)
			}

			got := fake.commands[0]
			if strings.Count(got, tt.want) != 5 {
				t.Errorf("expected %q quoted in all five positions, got:\n%s", tt.value, got)
			}
			// The raw, unquoted value must not appear anywhere outside those quotes.
			stripped := strings.ReplaceAll(got, tt.want, "")
			if strings.Contains(stripped, tt.value) {
				t.Errorf("unquoted %q survived into the command:\n%s", tt.value, got)
			}
		})
	}

	// The migration id position goes through the same helper.
	fake := &fakeTransport{}
	client := newTestClient(t, fake)
	if err := client.CancelMigration(context.Background(), "id'; whoami; #"); err != nil {
		t.Fatalf("CancelMigration: %v", err)
	}
	if !strings.HasSuffix(fake.commands[0], `--migration-id 'id'\''; whoami; #'`) {
		t.Errorf("migration id was not quoted:\n%s", fake.commands[0])
	}

	// So does the list page cursor.
	fake2 := &fakeTransport{respond: func(string) (string, string, error) {
		return `{"migrations":[]}`, "", nil
	}}
	client2 := newTestClient(t, fake2)
	if _, err := client2.ListMigrations(context.Background(), "2; rm -rf /"); err != nil {
		t.Fatalf("ListMigrations: %v", err)
	}
	if !strings.HasSuffix(fake2.commands[0], `--page '2; rm -rf /'`) {
		t.Errorf("page cursor was not quoted:\n%s", fake2.commands[0])
	}
}

// TestELMClient_CutoverNeverForces is the counterfactual vehicle for the guarded
// builder: no input, including one that itself looks like --force, can produce a
// bare --force in the cutover command, and the builder refuses the flag outright.
func TestELMClient_CutoverNeverForces(t *testing.T) {
	inputs := []string{"mig-123", "--force", "mig --force", "'; elm migration cutover-to-destination --force #"}

	for _, in := range inputs {
		t.Run(fmt.Sprintf("input=%q", in), func(t *testing.T) {
			fake := &fakeTransport{}
			client := newTestClient(t, fake)

			if err := client.CutoverToDestination(context.Background(), in); err != nil {
				t.Fatalf("CutoverToDestination: %v", err)
			}
			got := fake.commands[0]

			// --force may only ever appear inside a single-quoted argument. Strip
			// every quoted argument and assert nothing is left.
			if strings.Contains(stripQuotedArgs(got), forceFlag) {
				t.Errorf("cutover command contains a bare %s:\n%s", forceFlag, got)
			}
		})
	}

	// The builder itself refuses --force for the cutover verb, so no future caller
	// can add one.
	if _, err := buildCommand(verbCutover, commandArg{forceFlag, "true"}); err == nil {
		t.Errorf("buildCommand accepted %s on the cutover verb", forceFlag)
	}
	// And it refuses any flag outside the verb's allowlist.
	if _, err := buildCommand(verbCutover, commandArg{flagVisibility, "private"}); err == nil {
		t.Errorf("buildCommand accepted a flag outside the cutover allowlist")
	}
	if _, err := buildCommand("migration nuke", commandArg{flagMigrationID, "x"}); err == nil {
		t.Errorf("buildCommand accepted an unknown verb")
	}
}

// stripQuotedArgs removes every single-quoted region from a command string so an
// assertion can look at the unquoted skeleton only. It follows POSIX rules: a
// single quote toggles quoting, and outside quotes a backslash escapes the next
// character (which is how the `'\”` escape survives as a literal quote).
func stripQuotedArgs(command string) string {
	var out strings.Builder
	runes := []rune(command)
	inQuote := false
	for i := 0; i < len(runes); i++ {
		r := runes[i]
		if r == '\'' {
			inQuote = !inQuote
			continue
		}
		if !inQuote && r == '\\' {
			i++ // the escaped character is a literal, not part of the skeleton
			continue
		}
		if !inQuote {
			out.WriteRune(r)
		}
	}
	return out.String()
}

func TestELMClient_StatusParsesReadiness(t *testing.T) {
	tests := []struct {
		name         string
		stdout       string
		wantState    string
		wantReady    bool
		wantProgress int
	}{
		{
			name:         "backfilling is not ready",
			stdout:       `{"id":"m1","state":"backfilling","phase":"git","progress_percent":42,"cutover_ready":false}`,
			wantState:    StateBackfilling,
			wantReady:    false,
			wantProgress: 42,
		},
		{
			name:         "explicit readiness is honoured",
			stdout:       `{"id":"m1","state":"ready_for_cutover","phase":"git","progress_percent":100,"cutover_ready":true}`,
			wantState:    StateReadyForCutover,
			wantReady:    true,
			wantProgress: 100,
		},
		{
			name:         "absent readiness derives from a recognised state",
			stdout:       `{"id":"m1","state":"ready_for_cutover","progress_percent":100}`,
			wantState:    StateReadyForCutover,
			wantReady:    true,
			wantProgress: 100,
		},
		{
			name:         "absent readiness on a mid-flight state is false",
			stdout:       `{"id":"m1","state":"backfilling","progress_percent":10}`,
			wantState:    StateBackfilling,
			wantReady:    false,
			wantProgress: 10,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			out := tt.stdout
			client := newTestClient(t, &fakeTransport{respond: func(string) (string, string, error) {
				return out, "", nil
			}})

			status, err := client.GetMigrationStatus(context.Background(), "m1")
			if err != nil {
				t.Fatalf("GetMigrationStatus: %v", err)
			}
			if status.State != tt.wantState {
				t.Errorf("state = %q, want %q", status.State, tt.wantState)
			}
			if status.CutoverReady != tt.wantReady {
				t.Errorf("cutover ready = %v, want %v", status.CutoverReady, tt.wantReady)
			}
			if status.ProgressPercent == nil || *status.ProgressPercent != tt.wantProgress {
				t.Errorf("progress = %v, want %d", status.ProgressPercent, tt.wantProgress)
			}
		})
	}
}

// TestELMClient_TransportFailureIsTyped -- failure mode 1.
func TestELMClient_TransportFailureIsTyped(t *testing.T) {
	unreachable := errors.New("dial tcp 10.0.0.1:22: connect: connection refused")
	client := newTestClient(t, &fakeTransport{respond: func(string) (string, string, error) {
		return "", "", unreachable
	}})

	_, err := client.GetMigrationStatus(context.Background(), "m1")

	var transportErr *TransportError
	if !errors.As(err, &transportErr) {
		t.Fatalf("expected *TransportError, got %T: %v", err, err)
	}
	if !errors.Is(err, unreachable) {
		t.Errorf("expected the underlying error to be wrapped, got %v", err)
	}

	var commandErr *CommandError
	var parseErr *ParseError
	if errors.As(err, &commandErr) || errors.As(err, &parseErr) {
		t.Error("a transport failure must not also read as a command or parse failure")
	}
}

// TestELMClient_NonZeroExitIsTyped -- failure mode 2.
func TestELMClient_NonZeroExitIsTyped(t *testing.T) {
	client := newTestClient(t, &fakeTransport{respond: func(string) (string, string, error) {
		return "", "migration m1 not found\n", &ExitError{Code: 4}
	}})

	_, err := client.GetMigrationStatus(context.Background(), "m1")

	var commandErr *CommandError
	if !errors.As(err, &commandErr) {
		t.Fatalf("expected *CommandError, got %T: %v", err, err)
	}
	if commandErr.ExitCode != 4 {
		t.Errorf("exit code = %d, want 4", commandErr.ExitCode)
	}
	if commandErr.Stderr != "migration m1 not found" {
		t.Errorf("stderr = %q, want the appliance message", commandErr.Stderr)
	}

	var transportErr *TransportError
	if errors.As(err, &transportErr) {
		t.Error("a non-zero exit must not read as a transport failure")
	}
}

// TestELMClient_UnparseableOutputIsTyped -- failure mode 3. The critical
// property is that NO unreadable payload yields CutoverReady == true.
func TestELMClient_UnparseableOutputIsTyped(t *testing.T) {
	drifted := []struct {
		name   string
		stdout string
	}{
		{"not json", "MIGRATION m1 IS 100% DONE"},
		{"empty", ""},
		{"missing id", `{"state":"ready_for_cutover"}`},
		{"missing state", `{"id":"m1","progress_percent":100}`},
		{"unrecognised state", `{"id":"m1","state":"totally-new-state","cutover_ready":true}`},
		{"unreadable timestamp", `{"id":"m1","state":"backfilling","created_at":"yesterday"}`},
	}

	for _, tt := range drifted {
		t.Run(tt.name, func(t *testing.T) {
			out := tt.stdout
			client := newTestClient(t, &fakeTransport{respond: func(string) (string, string, error) {
				return out, "", nil
			}})

			status, err := client.GetMigrationStatus(context.Background(), "m1")

			var parseErr *ParseError
			if !errors.As(err, &parseErr) {
				t.Fatalf("expected *ParseError, got %T: %v (status=%#v)", err, err, status)
			}
			if status != nil {
				t.Fatalf("expected no status on a parse failure, got %#v", status)
			}
		})
	}
}

func TestELMClient_CreateRejectsOutputWithoutID(t *testing.T) {
	for _, stdout := range []string{`{}`, `{"id":""}`, `not json`} {
		out := stdout
		client := newTestClient(t, &fakeTransport{respond: func(string) (string, string, error) {
			return out, "", nil
		}})

		ref, err := client.CreateMigration(context.Background(), CreateMigrationRequest{SourceOrg: "a", SourceRepo: "b"})
		var parseErr *ParseError
		if !errors.As(err, &parseErr) {
			t.Errorf("stdout %q: expected *ParseError, got %T: %v", stdout, err, err)
		}
		if ref != nil {
			t.Errorf("stdout %q: expected no migration ref, got %#v", stdout, ref)
		}
	}
}

func TestELMClient_ListPagination(t *testing.T) {
	fake := &fakeTransport{respond: func(command string) (string, string, error) {
		if strings.Contains(command, "--page") {
			return `{"migrations":[{"id":"m2","state":"ready_for_cutover","cutover_ready":true}]}`, "", nil
		}
		return `{"migrations":[{"id":"m1","state":"backfilling"}],"next_page":"2"}`, "", nil
	}}
	client := newTestClient(t, fake)

	first, err := client.ListMigrations(context.Background(), "")
	if err != nil {
		t.Fatalf("ListMigrations: %v", err)
	}
	if len(first.Migrations) != 1 || first.NextPage != "2" {
		t.Fatalf("unexpected first page: %#v", first)
	}
	// The first page is requested without a --page flag.
	if strings.Contains(fake.commands[0], flagPage) {
		t.Errorf("first page should not carry %s: %s", flagPage, fake.commands[0])
	}

	second, err := client.ListMigrations(context.Background(), first.NextPage)
	if err != nil {
		t.Fatalf("ListMigrations page 2: %v", err)
	}
	if len(second.Migrations) != 1 || second.NextPage != "" {
		t.Fatalf("unexpected second page: %#v", second)
	}
	if !second.Migrations[0].CutoverReady {
		t.Error("expected the second page entry to report cutover readiness")
	}
}

func TestELMClient_ListRejectsEntryWithoutID(t *testing.T) {
	client := newTestClient(t, &fakeTransport{respond: func(string) (string, string, error) {
		return `{"migrations":[{"state":"backfilling"}]}`, "", nil
	}})

	page, err := client.ListMigrations(context.Background(), "")
	var parseErr *ParseError
	if !errors.As(err, &parseErr) {
		t.Fatalf("expected *ParseError, got %T: %v", err, err)
	}
	if page != nil {
		t.Fatalf("expected no page on a parse failure, got %#v", page)
	}
}

func TestELMClient_StartAndCancelCommands(t *testing.T) {
	prefix := commandEnvPrefix + " elm "

	fake := &fakeTransport{}
	client := newTestClient(t, fake)
	if err := client.StartMigration(context.Background(), "m1"); err != nil {
		t.Fatalf("StartMigration: %v", err)
	}
	if err := client.CancelMigration(context.Background(), "m1"); err != nil {
		t.Fatalf("CancelMigration: %v", err)
	}

	want := []string{
		prefix + `migration start --migration-id 'm1'`,
		prefix + `migration cancel --migration-id 'm1'`,
	}
	for i, w := range want {
		if fake.commands[i] != w {
			t.Errorf("command %d\n got: %s\nwant: %s", i, fake.commands[i], w)
		}
	}
}

// TestELMClient_EveryCommandCarriesApplianceEnvironment pins the environment
// prefix on every verb: API_URL is the loopback exporter and the HMAC key is
// resolved by ghe-config inside the admin shell, never interpolated here.
func TestELMClient_EveryCommandCarriesApplianceEnvironment(t *testing.T) {
	fake := &fakeTransport{respond: func(command string) (string, string, error) {
		switch {
		case strings.Contains(command, "migration create"):
			return `{"id":"m1"}`, "", nil
		case strings.Contains(command, "migration list"):
			return `{"migrations":[]}`, "", nil
		case strings.Contains(command, "migration status"):
			return `{"id":"m1","state":"backfilling"}`, "", nil
		}
		return "", "", nil
	}}
	client := newTestClient(t, fake)
	ctx := context.Background()

	if _, err := client.CreateMigration(ctx, CreateMigrationRequest{SourceOrg: "a", SourceRepo: "b"}); err != nil {
		t.Fatal(err)
	}
	if err := client.StartMigration(ctx, "m1"); err != nil {
		t.Fatal(err)
	}
	if _, err := client.GetMigrationStatus(ctx, "m1"); err != nil {
		t.Fatal(err)
	}
	if _, err := client.ListMigrations(ctx, ""); err != nil {
		t.Fatal(err)
	}
	if err := client.CutoverToDestination(ctx, "m1"); err != nil {
		t.Fatal(err)
	}
	if err := client.CancelMigration(ctx, "m1"); err != nil {
		t.Fatal(err)
	}

	if len(fake.commands) != 6 {
		t.Fatalf("expected 6 commands, got %d: %v", len(fake.commands), fake.commands)
	}
	for _, cmd := range fake.commands {
		if !strings.HasPrefix(cmd, commandEnvPrefix+" elm migration ") {
			t.Errorf("command missing the appliance environment prefix:\n%s", cmd)
		}
		// The HMAC key itself is never interpolated -- only the ghe-config lookup is.
		if !strings.Contains(cmd, `$(ghe-config secrets.elm-exporter.elm-exporter-hmac-keys)`) {
			t.Errorf("command does not resolve the HMAC key inside the admin shell:\n%s", cmd)
		}
	}
}
