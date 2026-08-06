package elm

import (
	"context"
	"fmt"
	"log/slog"
	"net"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"

	"golang.org/x/crypto/ssh"
	"golang.org/x/crypto/ssh/knownhosts"
)

// Transport runs a single shell command on the GHES appliance and returns its
// output. It is the seam every downstream caller and test goes through: if the
// ELM exporter ever exposes a remote API, only this file changes.
type Transport interface {
	// Run executes command and returns its stdout and stderr. A non-zero exit
	// status is reported as an *ExitError; any other error means the command did
	// not run to completion.
	Run(ctx context.Context, command string) (stdout, stderr string, err error)
	// Close releases the underlying connection.
	Close() error
}

// ExitError reports that the remote command ran to completion with a non-zero
// exit status.
type ExitError struct {
	Code int
}

func (e *ExitError) Error() string {
	return fmt.Sprintf("remote command exited with status %d", e.Code)
}

// SSHConfig describes the connection into the GHES administrative shell.
//
// KnownHostsPath and PrivateKeyPath are both REQUIRED. There is no flag, debug
// mode or test hook that relaxes host-key verification: ssh.InsecureIgnoreHostKey
// appears nowhere in this package.
type SSHConfig struct {
	Host                 string
	Port                 int
	User                 string
	PrivateKeyPath       string
	PrivateKeyPassphrase string
	KnownHostsPath       string
	ConnectTimeout       time.Duration
}

const defaultConnectTimeout = 30 * time.Second

// sshTransport holds one pooled *ssh.Client. An ssh.Session may run at most one
// command, so Run opens a NEW session per command over that pooled client.
type sshTransport struct {
	addr     string
	logger   *slog.Logger
	redactor *redactor

	mu     sync.Mutex
	client *ssh.Client
}

// NewSSHTransport dials the appliance and returns a ready transport.
//
// Construction FAILS CLOSED: a missing or unreadable known_hosts file, a missing,
// unreadable or undecryptable private key, or a host key that does not match
// known_hosts all return an error and a nil Transport. A caller can therefore
// never end up with a transport that skipped verification.
func NewSSHTransport(cfg SSHConfig, logger *slog.Logger) (Transport, error) {
	if logger == nil {
		logger = slog.Default()
	}

	if strings.TrimSpace(cfg.Host) == "" {
		return nil, fmt.Errorf("elm ssh: host is required")
	}
	if strings.TrimSpace(cfg.User) == "" {
		return nil, fmt.Errorf("elm ssh: user is required")
	}

	// Host-key verification is mandatory. An empty path is refused up front so a
	// misconfiguration cannot be mistaken for "verification disabled".
	if strings.TrimSpace(cfg.KnownHostsPath) == "" {
		return nil, fmt.Errorf("elm ssh: known_hosts path is required (host key verification is mandatory)")
	}
	hostKeyCallback, err := knownhosts.New(cfg.KnownHostsPath)
	if err != nil {
		return nil, fmt.Errorf("elm ssh: cannot read known_hosts %q: %w", cfg.KnownHostsPath, err)
	}

	if strings.TrimSpace(cfg.PrivateKeyPath) == "" {
		return nil, fmt.Errorf("elm ssh: private key path is required")
	}
	keyPEM, err := os.ReadFile(cfg.PrivateKeyPath)
	if err != nil {
		return nil, fmt.Errorf("elm ssh: cannot read private key %q: %w", cfg.PrivateKeyPath, err)
	}

	red := newRedactor(keyPEM, cfg.PrivateKeyPassphrase)

	var signer ssh.Signer
	if cfg.PrivateKeyPassphrase != "" {
		signer, err = ssh.ParsePrivateKeyWithPassphrase(keyPEM, []byte(cfg.PrivateKeyPassphrase))
	} else {
		signer, err = ssh.ParsePrivateKey(keyPEM)
	}
	if err != nil {
		// The parse error is redacted before it is surfaced: key material must not
		// reach an error string any more than it may reach a log record.
		return nil, fmt.Errorf("elm ssh: cannot parse private key %q: %s", cfg.PrivateKeyPath, red.redact(err.Error()))
	}

	port := cfg.Port
	if port == 0 {
		port = 22
	}
	timeout := cfg.ConnectTimeout
	if timeout == 0 {
		timeout = defaultConnectTimeout
	}
	addr := net.JoinHostPort(cfg.Host, strconv.Itoa(port))

	clientConfig := &ssh.ClientConfig{
		User:            cfg.User,
		Auth:            []ssh.AuthMethod{ssh.PublicKeys(signer)},
		HostKeyCallback: hostKeyCallback,
		Timeout:         timeout,
	}

	logger.Debug("connecting to elm appliance",
		"addr", addr,
		"user", cfg.User,
		"known_hosts_path", cfg.KnownHostsPath,
		"private_key_path", cfg.PrivateKeyPath,
	)

	client, err := ssh.Dial("tcp", addr, clientConfig)
	if err != nil {
		return nil, fmt.Errorf("elm ssh: cannot connect to %s: %s", addr, red.redact(err.Error()))
	}

	return &sshTransport{
		addr:     addr,
		logger:   logger,
		redactor: red,
		client:   client,
	}, nil
}

// Run opens a new session on the pooled client and runs one command.
func (t *sshTransport) Run(ctx context.Context, command string) (string, string, error) {
	t.mu.Lock()
	client := t.client
	t.mu.Unlock()

	if client == nil {
		return "", "", fmt.Errorf("elm ssh: transport is closed")
	}

	t.logger.Debug("running elm command", "addr", t.addr, "command", t.redactor.redact(command))

	// A session runs exactly one command, so every Run gets a fresh one.
	session, err := client.NewSession()
	if err != nil {
		return "", "", fmt.Errorf("elm ssh: cannot open session: %s", t.redactor.redact(err.Error()))
	}
	defer func() { _ = session.Close() }()

	var stdout, stderr strings.Builder
	session.Stdout = &stdout
	session.Stderr = &stderr

	done := make(chan error, 1)
	go func() { done <- session.Run(command) }()

	select {
	case <-ctx.Done():
		_ = session.Signal(ssh.SIGKILL)
		return stdout.String(), stderr.String(), ctx.Err()
	case runErr := <-done:
		if runErr != nil {
			var exitErr *ssh.ExitError
			if ok := asExitError(runErr, &exitErr); ok {
				t.logger.Warn("elm command exited non-zero",
					"addr", t.addr,
					"command", t.redactor.redact(command),
					"exit_code", exitErr.ExitStatus(),
					"stderr", t.redactor.redact(stderr.String()),
				)
				return stdout.String(), stderr.String(), &ExitError{Code: exitErr.ExitStatus()}
			}
			t.logger.Warn("elm command failed",
				"addr", t.addr,
				"command", t.redactor.redact(command),
				"error", t.redactor.redact(runErr.Error()),
				"stderr", t.redactor.redact(stderr.String()),
			)
			return stdout.String(), stderr.String(), fmt.Errorf("elm ssh: %s", t.redactor.redact(runErr.Error()))
		}
	}

	return stdout.String(), stderr.String(), nil
}

// Close releases the pooled client. It is safe to call more than once.
func (t *sshTransport) Close() error {
	t.mu.Lock()
	defer t.mu.Unlock()
	if t.client == nil {
		return nil
	}
	err := t.client.Close()
	t.client = nil
	return err
}

// redactor scrubs SSH key material and passphrases out of anything bound for a
// log record or an error string. Commands, stderr and library errors are all
// routed through it, because any of them can end up carrying a secret that a
// caller interpolated by mistake.
type redactor struct {
	secrets []string
}

const redactionPlaceholder = "[REDACTED]"

// minRedactableLength keeps short, low-entropy fragments (PEM armour, a one-word
// passphrase) from turning every log line into placeholders.
const minRedactableLength = 8

func newRedactor(privateKeyPEM []byte, passphrase string) *redactor {
	r := &redactor{}
	if len(privateKeyPEM) > 0 {
		full := string(privateKeyPEM)
		r.add(full)
		r.add(strings.TrimSpace(full))
		// Individual body lines matter too: a partial leak of the base64 payload is
		// still a leak, so each line is redacted independently of the whole.
		for _, line := range strings.Split(full, "\n") {
			line = strings.TrimSpace(line)
			if strings.HasPrefix(line, "-----") {
				continue // PEM armour is public boilerplate, not key material.
			}
			r.add(line)
		}
	}
	r.add(passphrase)
	return r
}

func (r *redactor) add(secret string) {
	if len(secret) < minRedactableLength {
		return
	}
	r.secrets = append(r.secrets, secret)
}

// redact replaces every known secret in s with a placeholder.
func (r *redactor) redact(s string) string {
	if r == nil || s == "" {
		return s
	}
	for _, secret := range r.secrets {
		s = strings.ReplaceAll(s, secret, redactionPlaceholder)
	}
	return s
}

// asExitError is a tiny wrapper so Run reads linearly; ssh.ExitError is returned
// directly by Session.Run rather than wrapped.
func asExitError(err error, target **ssh.ExitError) bool {
	if e, ok := err.(*ssh.ExitError); ok {
		*target = e
		return true
	}
	return false
}
