package doctor

import (
	"context"
	"fmt"
	"net"
	"os"
	"os/exec"
	"strings"
	"time"
)

// ToolchainRunner inspects available system developer toolchains with timeouts and fixed argv.
type ToolchainRunner interface {
	Run(ctx context.Context, name string, args ...string) (string, error)
}

// PortChecker verifies whether a declared local TCP port is in use.
type PortChecker interface {
	CheckPort(port int) (bool, error)
}

// ServiceProber tests read-only TCP connectivity to a declared service.
type ServiceProber interface {
	Probe(network, address string, timeout time.Duration) error
}

// GitChecker verifies repository file shielding and git index tracking.
type GitChecker interface {
	CheckIgnore(ctx context.Context, path string) (bool, error)
	IsTracked(ctx context.Context, path string) (bool, error)
}

// FileSystem abstracts filesystem lookups for project type detection and file reading.
type FileSystem interface {
	FileExists(path string) bool
	ReadFile(path string) ([]byte, error)
}

// OSToolchainRunner executes real system binaries with context timeouts.
type OSToolchainRunner struct{}

func (r *OSToolchainRunner) Run(ctx context.Context, name string, args ...string) (string, error) {
	out, err := exec.CommandContext(ctx, name, args...).Output()
	return string(out), err
}

// NetPortChecker tests 127.0.0.1 only (preventing Windows firewall alerts).
// It prefers Dial first to see if an active server is listening, and only falls back
// to a momentary bind on 127.0.0.1 to verify availability, closing immediately.
type NetPortChecker struct{}

func (p *NetPortChecker) CheckPort(port int) (bool, error) {
	target := fmt.Sprintf("127.0.0.1:%d", port)

	// 1. Prefer Dial to detect if an active process is actively accepting connections
	conn, err := net.DialTimeout("tcp", target, 80*time.Millisecond)
	if err == nil {
		_ = conn.Close()
		return true, nil // Port is in use
	}

	// 2. Fall back to a brief Listen bind exclusively on 127.0.0.1
	// Note: Binding 127.0.0.1 avoids OS security dialogs/firewall popups.
	ln, err := net.Listen("tcp", target)
	if err != nil {
		return true, err // Unable to bind: port in use or restricted
	}
	_ = ln.Close()
	return false, nil // Port is available
}

// NetServiceProber conducts auth-less, strictly TCP-only connect probes.
type NetServiceProber struct{}

func (s *NetServiceProber) Probe(network, address string, timeout time.Duration) error {
	conn, err := net.DialTimeout(network, address, timeout)
	if err != nil {
		return err
	}
	return conn.Close()
}

// OSGitChecker runs git commands to test ignore status and index tracking.
type OSGitChecker struct{}

func (g *OSGitChecker) CheckIgnore(ctx context.Context, path string) (bool, error) {
	cmd := exec.CommandContext(ctx, "git", "check-ignore", "-q", path)
	err := cmd.Run()
	if err == nil {
		return true, nil // Exit 0 means ignored
	}
	if exitErr, ok := err.(*exec.ExitError); ok && exitErr.ExitCode() == 1 {
		return false, nil // Exit 1 means not ignored
	}
	return false, err
}

func (g *OSGitChecker) IsTracked(ctx context.Context, path string) (bool, error) {
	out, err := exec.CommandContext(ctx, "git", "ls-files", path).Output()
	if err != nil {
		return false, err
	}
	return strings.TrimSpace(string(out)) != "", nil
}

// OSFileSystem reads files from the local filesystem.
type OSFileSystem struct{}

func (f *OSFileSystem) FileExists(path string) bool {
	info, err := os.Stat(path)
	if os.IsNotExist(err) || info == nil || info.IsDir() {
		return false
	}
	return true
}

func (f *OSFileSystem) ReadFile(path string) ([]byte, error) {
	return os.ReadFile(path)
}
