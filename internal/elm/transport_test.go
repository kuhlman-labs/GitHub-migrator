package elm

import (
	"context"
	"crypto/ed25519"
	"crypto/rand"
	"encoding/pem"
	"errors"
	"fmt"
	"log/slog"
	"net"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"golang.org/x/crypto/ssh"
	"golang.org/x/crypto/ssh/knownhosts"
)

// --- test fixtures -----------------------------------------------------------

// generateKey returns a freshly generated ed25519 key pair as an ssh.Signer plus
// its PEM encoding. Every key used in these tests is generated independently, so
// a "non-matching host key" fixture is non-matching BY CONSTRUCTION rather than
// because some control under test produced it.
func generateKey(t *testing.T, passphrase string) (ssh.Signer, []byte) {
	t.Helper()

	_, priv, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatalf("generate key: %v", err)
	}

	var block *pem.Block
	if passphrase == "" {
		block, err = ssh.MarshalPrivateKey(priv, "elm-test")
	} else {
		block, err = ssh.MarshalPrivateKeyWithPassphrase(priv, "elm-test", []byte(passphrase))
	}
	if err != nil {
		t.Fatalf("marshal key: %v", err)
	}

	signer, err := ssh.NewSignerFromKey(priv)
	if err != nil {
		t.Fatalf("signer from key: %v", err)
	}
	return signer, pem.EncodeToMemory(block)
}

func writeFile(t *testing.T, name string, content []byte) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), name)
	if err := os.WriteFile(path, content, 0o600); err != nil {
		t.Fatalf("write %s: %v", name, err)
	}
	return path
}

// execResult is what the test SSH server returns for one exec request.
type execResult struct {
	stdout string
	stderr string
	exit   int
}

// testSSHServer is an in-process SSH server on 127.0.0.1. It is REACHABLE, so a
// test that expects a host-key rejection cannot pass merely because the dial
// failed for an unrelated reason.
type testSSHServer struct {
	host string
	port int

	mu       sync.Mutex
	commands []string
	sessions int
}

func startTestSSHServer(t *testing.T, hostKey ssh.Signer, handler func(command string) execResult) *testSSHServer {
	t.Helper()

	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("listen: %v", err)
	}
	t.Cleanup(func() { _ = ln.Close() })

	addr, ok := ln.Addr().(*net.TCPAddr)
	if !ok {
		t.Fatalf("unexpected listener address type %T", ln.Addr())
	}
	srv := &testSSHServer{host: "127.0.0.1", port: addr.Port}

	serverConfig := &ssh.ServerConfig{NoClientAuth: true}
	serverConfig.AddHostKey(hostKey)

	go func() {
		for {
			conn, err := ln.Accept()
			if err != nil {
				return
			}
			go srv.handleConn(conn, serverConfig, handler)
		}
	}()

	return srv
}

func (s *testSSHServer) handleConn(conn net.Conn, cfg *ssh.ServerConfig, handler func(string) execResult) {
	sconn, chans, reqs, err := ssh.NewServerConn(conn, cfg)
	if err != nil {
		_ = conn.Close()
		return
	}
	defer func() { _ = sconn.Close() }()
	go ssh.DiscardRequests(reqs)

	for newChan := range chans {
		if newChan.ChannelType() != "session" {
			_ = newChan.Reject(ssh.UnknownChannelType, "only sessions")
			continue
		}
		ch, requests, err := newChan.Accept()
		if err != nil {
			return
		}
		s.mu.Lock()
		s.sessions++
		s.mu.Unlock()

		go s.serveSession(ch, requests, handler)
	}
}

func (s *testSSHServer) serveSession(ch ssh.Channel, requests <-chan *ssh.Request, handler func(string) execResult) {
	defer func() { _ = ch.Close() }()
	for req := range requests {
		if req.Type != "exec" {
			_ = req.Reply(false, nil)
			continue
		}
		var payload struct{ Command string }
		if err := ssh.Unmarshal(req.Payload, &payload); err != nil {
			_ = req.Reply(false, nil)
			continue
		}
		_ = req.Reply(true, nil)

		s.mu.Lock()
		s.commands = append(s.commands, payload.Command)
		s.mu.Unlock()

		result := execResult{}
		if handler != nil {
			result = handler(payload.Command)
		}
		_, _ = ch.Write([]byte(result.stdout))
		_, _ = ch.Stderr().Write([]byte(result.stderr))
		_, _ = ch.SendRequest("exit-status", false, ssh.Marshal(struct{ Status uint32 }{uint32(result.exit)})) //nolint:gosec // small non-negative test value
		return
	}
}

func (s *testSSHServer) recorded() ([]string, int) {
	s.mu.Lock()
	defer s.mu.Unlock()
	return append([]string(nil), s.commands...), s.sessions
}

// knownHostsFor writes a known_hosts file pinning key for the server address.
func knownHostsFor(t *testing.T, srv *testSSHServer, key ssh.PublicKey) string {
	t.Helper()
	addr := net.JoinHostPort(srv.host, fmt.Sprintf("%d", srv.port))
	line := knownhosts.Line([]string{addr}, key)
	return writeFile(t, "known_hosts", []byte(line+"\n"))
}

// --- capturing logger --------------------------------------------------------

// captureHandler records every attribute value of every record so a test can
// assert that a secret reached NO log record.
type captureHandler struct {
	mu   sync.Mutex
	text []string
}

func (h *captureHandler) Enabled(context.Context, slog.Level) bool { return true }

func (h *captureHandler) Handle(_ context.Context, r slog.Record) error {
	var b strings.Builder
	b.WriteString(r.Message)
	r.Attrs(func(a slog.Attr) bool {
		b.WriteString(" ")
		b.WriteString(a.Key)
		b.WriteString("=")
		b.WriteString(a.Value.String())
		return true
	})
	h.mu.Lock()
	h.text = append(h.text, b.String())
	h.mu.Unlock()
	return nil
}

func (h *captureHandler) WithAttrs([]slog.Attr) slog.Handler { return h }
func (h *captureHandler) WithGroup(string) slog.Handler      { return h }

func (h *captureHandler) all() string {
	h.mu.Lock()
	defer h.mu.Unlock()
	return strings.Join(h.text, "\n")
}

// --- tests -------------------------------------------------------------------

// TestSSHTransport_RejectsUnknownHostKey is the counterfactual vehicle for
// mandatory host-key verification. The server is reachable and presents key A;
// known_hosts pins an independently generated key B.
func TestSSHTransport_RejectsUnknownHostKey(t *testing.T) {
	serverHostKey, _ := generateKey(t, "")
	_, clientKeyPEM := generateKey(t, "")
	unrelatedHostKey, _ := generateKey(t, "") // key B -- never presented by the server

	srv := startTestSSHServer(t, serverHostKey, nil)

	knownHostsPath := knownHostsFor(t, srv, unrelatedHostKey.PublicKey())
	keyPath := writeFile(t, "id_ed25519", clientKeyPEM)

	transport, err := NewSSHTransport(SSHConfig{
		Host:           srv.host,
		Port:           srv.port,
		User:           "admin",
		PrivateKeyPath: keyPath,
		KnownHostsPath: knownHostsPath,
		ConnectTimeout: 5 * time.Second,
	}, slog.New(slog.DiscardHandler))

	if err == nil {
		_ = transport.Close()
		t.Fatal("expected the connection to be refused for an unknown host key, got a usable transport")
	}
	if transport != nil {
		t.Fatalf("expected a nil transport on host key mismatch, got %#v", transport)
	}
	msg := strings.ToLower(err.Error())
	if !strings.Contains(msg, "knownhosts") && !strings.Contains(msg, "host key") {
		t.Fatalf("expected a host key verification error, got: %v", err)
	}
}

// TestSSHTransport_AcceptsMatchingHostKey is the paired positive case: the same
// setup with the correct host key pinned connects successfully, so the rejection
// above cannot be passing because connections never work.
func TestSSHTransport_AcceptsMatchingHostKey(t *testing.T) {
	serverHostKey, _ := generateKey(t, "")
	_, clientKeyPEM := generateKey(t, "")

	srv := startTestSSHServer(t, serverHostKey, func(string) execResult {
		return execResult{stdout: "ok\n"}
	})

	transport, err := NewSSHTransport(SSHConfig{
		Host:           srv.host,
		Port:           srv.port,
		User:           "admin",
		PrivateKeyPath: writeFile(t, "id_ed25519", clientKeyPEM),
		KnownHostsPath: knownHostsFor(t, srv, serverHostKey.PublicKey()),
		ConnectTimeout: 5 * time.Second,
	}, slog.New(slog.DiscardHandler))
	if err != nil {
		t.Fatalf("expected the pinned host key to be accepted: %v", err)
	}
	t.Cleanup(func() { _ = transport.Close() })

	stdout, _, err := transport.Run(context.Background(), "echo hi")
	if err != nil {
		t.Fatalf("Run: %v", err)
	}
	if strings.TrimSpace(stdout) != "ok" {
		t.Fatalf("stdout = %q, want %q", stdout, "ok")
	}
}

// TestSSHTransport_RequiresKnownHostsAndKey covers the fail-closed construction
// guards, one subtest per guard.
func TestSSHTransport_RequiresKnownHostsAndKey(t *testing.T) {
	serverHostKey, _ := generateKey(t, "")
	_, clientKeyPEM := generateKey(t, "")
	srv := startTestSSHServer(t, serverHostKey, nil)

	goodKnownHosts := knownHostsFor(t, srv, serverHostKey.PublicKey())
	goodKey := writeFile(t, "id_ed25519", clientKeyPEM)
	missing := filepath.Join(t.TempDir(), "does-not-exist")

	base := SSHConfig{
		Host:           srv.host,
		Port:           srv.port,
		User:           "admin",
		PrivateKeyPath: goodKey,
		KnownHostsPath: goodKnownHosts,
		ConnectTimeout: 5 * time.Second,
	}

	tests := []struct {
		name     string
		mutate   func(*SSHConfig)
		contains string
	}{
		// Each case asserts the SPECIFIC guard's message, not merely "some error".
		// Every one of these inputs also fails somewhere downstream, so a vaguer
		// assertion would stay green with the guard deleted.
		{"empty known_hosts path", func(c *SSHConfig) { c.KnownHostsPath = "" }, "known_hosts path is required"},
		{"unreadable known_hosts path", func(c *SSHConfig) { c.KnownHostsPath = missing }, "cannot read known_hosts"},
		{"empty private key path", func(c *SSHConfig) { c.PrivateKeyPath = "" }, "private key path is required"},
		{"unreadable private key path", func(c *SSHConfig) { c.PrivateKeyPath = missing }, "cannot read private key"},
		{"undecryptable private key", func(c *SSHConfig) {
			_, encrypted := generateKey(t, "a-passphrase-we-will-not-supply")
			c.PrivateKeyPath = writeFile(t, "encrypted_key", encrypted)
		}, "cannot parse private key"},
		{"empty host", func(c *SSHConfig) { c.Host = "" }, "host is required"},
		{"empty user", func(c *SSHConfig) { c.User = "" }, "user is required"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := base
			tt.mutate(&cfg)

			transport, err := NewSSHTransport(cfg, slog.New(slog.DiscardHandler))
			if err == nil {
				_ = transport.Close()
				t.Fatalf("expected construction to fail closed for %s", tt.name)
			}
			if transport != nil {
				t.Fatalf("expected a nil transport for %s, got %#v", tt.name, transport)
			}
			if !strings.Contains(err.Error(), tt.contains) {
				t.Fatalf("error %q does not mention %q", err.Error(), tt.contains)
			}
		})
	}
}

// TestSSHTransport_NeverLogsKeyMaterial installs a capturing handler across
// connect and two commands and asserts the private key body and the passphrase
// reach no record. The leak is seeded by construction: one command text embeds
// the key PEM (an interpolation accident) and the server echoes the passphrase
// back on stderr with a non-zero exit (a chatty remote tool).
func TestSSHTransport_NeverLogsKeyMaterial(t *testing.T) {
	const passphrase = "correct-horse-battery-staple"

	serverHostKey, _ := generateKey(t, "")
	_, clientKeyPEM := generateKey(t, passphrase)

	srv := startTestSSHServer(t, serverHostKey, func(command string) execResult {
		if strings.Contains(command, "leaky") {
			return execResult{stderr: "auth failed for passphrase " + passphrase, exit: 3}
		}
		return execResult{stdout: "{}\n"}
	})

	capture := &captureHandler{}
	transport, err := NewSSHTransport(SSHConfig{
		Host:                 srv.host,
		Port:                 srv.port,
		User:                 "admin",
		PrivateKeyPath:       writeFile(t, "id_ed25519", clientKeyPEM),
		PrivateKeyPassphrase: passphrase,
		KnownHostsPath:       knownHostsFor(t, srv, serverHostKey.PublicKey()),
		ConnectTimeout:       5 * time.Second,
	}, slog.New(capture))
	if err != nil {
		t.Fatalf("NewSSHTransport: %v", err)
	}
	t.Cleanup(func() { _ = transport.Close() })

	ctx := context.Background()
	// Command 1: key material interpolated into the command by mistake.
	if _, _, err := transport.Run(ctx, "elm migration status --note "+string(clientKeyPEM)); err != nil {
		t.Fatalf("Run 1: %v", err)
	}
	// Command 2: the remote tool echoes the passphrase on stderr and exits non-zero.
	if _, _, err := transport.Run(ctx, "elm migration leaky"); err == nil {
		t.Fatal("expected a non-zero exit for the leaky command")
	}

	logged := capture.all()
	if logged == "" {
		t.Fatal("no log records captured; the test cannot prove anything about redaction")
	}
	if strings.Contains(logged, passphrase) {
		t.Errorf("passphrase leaked into a log record:\n%s", logged)
	}
	for _, line := range strings.Split(strings.TrimSpace(string(clientKeyPEM)), "\n") {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "-----") || len(line) < minRedactableLength {
			continue
		}
		if strings.Contains(logged, line) {
			t.Fatalf("private key material leaked into a log record:\n%s", logged)
		}
	}
}

// TestSSHTransport_NewSessionPerCommand proves the transport does not reuse a
// session: an ssh.Session may run at most one command, so two sequential
// commands over one pooled client must open two sessions.
func TestSSHTransport_NewSessionPerCommand(t *testing.T) {
	serverHostKey, _ := generateKey(t, "")
	_, clientKeyPEM := generateKey(t, "")

	srv := startTestSSHServer(t, serverHostKey, func(string) execResult {
		return execResult{stdout: "{}\n"}
	})

	transport, err := NewSSHTransport(SSHConfig{
		Host:           srv.host,
		Port:           srv.port,
		User:           "admin",
		PrivateKeyPath: writeFile(t, "id_ed25519", clientKeyPEM),
		KnownHostsPath: knownHostsFor(t, srv, serverHostKey.PublicKey()),
		ConnectTimeout: 5 * time.Second,
	}, slog.New(slog.DiscardHandler))
	if err != nil {
		t.Fatalf("NewSSHTransport: %v", err)
	}
	t.Cleanup(func() { _ = transport.Close() })

	ctx := context.Background()
	for i := 0; i < 2; i++ {
		if _, _, err := transport.Run(ctx, fmt.Sprintf("elm migration list --page '%d'", i)); err != nil {
			t.Fatalf("Run %d: %v", i, err)
		}
	}

	commands, sessions := srv.recorded()
	if len(commands) != 2 {
		t.Fatalf("server saw %d commands, want 2: %v", len(commands), commands)
	}
	if sessions != 2 {
		t.Fatalf("server opened %d sessions, want 2 (one per command)", sessions)
	}
}

// TestSSHTransport_NonZeroExitIsExitError pins the transport's contract with the
// client: a non-zero exit is an *ExitError, everything else is not.
func TestSSHTransport_NonZeroExitIsExitError(t *testing.T) {
	serverHostKey, _ := generateKey(t, "")
	_, clientKeyPEM := generateKey(t, "")

	srv := startTestSSHServer(t, serverHostKey, func(string) execResult {
		return execResult{stderr: "boom", exit: 42}
	})

	transport, err := NewSSHTransport(SSHConfig{
		Host:           srv.host,
		Port:           srv.port,
		User:           "admin",
		PrivateKeyPath: writeFile(t, "id_ed25519", clientKeyPEM),
		KnownHostsPath: knownHostsFor(t, srv, serverHostKey.PublicKey()),
		ConnectTimeout: 5 * time.Second,
	}, slog.New(slog.DiscardHandler))
	if err != nil {
		t.Fatalf("NewSSHTransport: %v", err)
	}
	t.Cleanup(func() { _ = transport.Close() })

	_, stderr, err := transport.Run(context.Background(), "elm migration status")
	var exitErr *ExitError
	if !errors.As(err, &exitErr) {
		t.Fatalf("expected *ExitError, got %T: %v", err, err)
	}
	if exitErr.Code != 42 {
		t.Errorf("exit code = %d, want 42", exitErr.Code)
	}
	if strings.TrimSpace(stderr) != "boom" {
		t.Errorf("stderr = %q, want %q", stderr, "boom")
	}
}

// TestSSHTransport_ClosedTransportRefusesRun covers the closed-transport guard.
func TestSSHTransport_ClosedTransportRefusesRun(t *testing.T) {
	serverHostKey, _ := generateKey(t, "")
	_, clientKeyPEM := generateKey(t, "")
	srv := startTestSSHServer(t, serverHostKey, func(string) execResult { return execResult{} })

	transport, err := NewSSHTransport(SSHConfig{
		Host:           srv.host,
		Port:           srv.port,
		User:           "admin",
		PrivateKeyPath: writeFile(t, "id_ed25519", clientKeyPEM),
		KnownHostsPath: knownHostsFor(t, srv, serverHostKey.PublicKey()),
		ConnectTimeout: 5 * time.Second,
	}, slog.New(slog.DiscardHandler))
	if err != nil {
		t.Fatalf("NewSSHTransport: %v", err)
	}
	if err := transport.Close(); err != nil {
		t.Fatalf("Close: %v", err)
	}
	if err := transport.Close(); err != nil {
		t.Fatalf("second Close should be a no-op: %v", err)
	}
	if _, _, err := transport.Run(context.Background(), "elm migration list"); err == nil {
		t.Fatal("expected Run on a closed transport to fail")
	}
}

// TestRedactor_ScrubsSecrets covers the redaction helper directly, including the
// case where there is nothing to redact.
func TestRedactor_ScrubsSecrets(t *testing.T) {
	_, keyPEM := generateKey(t, "")
	r := newRedactor(keyPEM, "hunter2-hunter2")

	scrubbed := r.redact("prefix " + string(keyPEM) + " suffix")
	for _, line := range strings.Split(strings.TrimSpace(string(keyPEM)), "\n") {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "-----") || len(line) < minRedactableLength {
			continue
		}
		if strings.Contains(scrubbed, line) {
			t.Errorf("key material survived redaction: %q", scrubbed)
		}
	}
	if got := r.redact("passphrase=hunter2-hunter2"); strings.Contains(got, "hunter2-hunter2") {
		t.Errorf("passphrase survived redaction: %q", got)
	}
	if got := r.redact("nothing secret here"); got != "nothing secret here" {
		t.Errorf("redact() mangled a clean string: %q", got)
	}
	if got := r.redact(""); got != "" {
		t.Errorf("redact(\"\") = %q", got)
	}
}
