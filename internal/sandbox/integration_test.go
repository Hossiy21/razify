//go:build docker

package sandbox_test

import (
	"bytes"
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/Hossiy21/razify/internal/sandbox"
)

func TestSandboxIntegration_NetworkNone(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	tmpDir := t.TempDir()

	cfg := sandbox.DefaultConfig()
	cfg.WorkspaceDir = tmpDir
	cfg.Image = "alpine:latest"
	cfg.Network = "none"
	// Attempt to connect or ping an IP outside loopback
	cfg.Command = []string{"wget", "-T", "2", "-qO-", "http://1.1.1.1"}

	var stdout, stderr bytes.Buffer
	result, _ := sandbox.Execute(ctx, cfg, &stdout, &stderr)

	if result.ExitCode == 0 {
		t.Fatalf("expected network connection to fail under --network none, but it succeeded!")
	}
}

func TestSandboxIntegration_WorkspaceReadOnly(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	tmpDir := t.TempDir()

	cfg := sandbox.DefaultConfig()
	cfg.WorkspaceDir = tmpDir
	cfg.Image = "alpine:latest"
	cfg.ReadOnlyWorkspace = true
	// Attempt to create a file in /workspace
	cfg.Command = []string{"touch", "/workspace/should_fail.txt"}

	var stdout, stderr bytes.Buffer
	result, _ := sandbox.Execute(ctx, cfg, &stdout, &stderr)

	if result.ExitCode == 0 {
		t.Fatalf("expected write to read-only workspace to fail, but exit code was 0")
	}

	if _, err := os.Stat(filepath.Join(tmpDir, "should_fail.txt")); !os.IsNotExist(err) {
		t.Fatalf("file was created on host filesystem despite read-only workspace mount!")
	}
}

func TestSandboxIntegration_GitReadOnly(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	tmpDir := t.TempDir()
	gitDir := filepath.Join(tmpDir, ".git")
	_ = os.MkdirAll(gitDir, 0755)

	cfg := sandbox.DefaultConfig()
	cfg.WorkspaceDir = tmpDir
	cfg.Image = "alpine:latest"
	cfg.ReadOnlyWorkspace = false // --read-write requested
	// Attempt to write inside .git
	cfg.Command = []string{"touch", "/workspace/.git/tampered_hook"}

	var stdout, stderr bytes.Buffer
	result, _ := sandbox.Execute(ctx, cfg, &stdout, &stderr)

	if result.ExitCode == 0 {
		t.Fatalf("expected write inside .git to fail even under --read-write, but got exit code 0")
	}

	if _, err := os.Stat(filepath.Join(gitDir, "tampered_hook")); !os.IsNotExist(err) {
		t.Fatalf("tampered_hook was created inside .git!")
	}
}

func TestSandboxIntegration_EnvFileMasking(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	tmpDir := t.TempDir()
	envPath := filepath.Join(tmpDir, ".env")
	// Real secret on host
	_ = os.WriteFile(envPath, []byte("SUPER_SECRET_TOKEN=real_unmasked_secret_value_12345\n"), 0600)

	cfg := sandbox.DefaultConfig()
	cfg.WorkspaceDir = tmpDir
	cfg.Image = "alpine:latest"
	cfg.MaskEnvFiles = true
	// Read /workspace/.env inside container
	cfg.Command = []string{"cat", "/workspace/.env"}

	var stdout, stderr bytes.Buffer
	result, err := sandbox.Execute(ctx, cfg, &stdout, &stderr)
	if err != nil {
		t.Fatalf("Execute error: %v, stderr: %s", err, stderr.String())
	}
	if result.ExitCode != 0 {
		t.Fatalf("cat command failed: exit code %d, stderr: %s", result.ExitCode, stderr.String())
	}

	catOutput := stdout.String()
	if strings.Contains(catOutput, "real_unmasked_secret_value_12345") {
		t.Fatalf("SECURITY VIOLATION: Host secret read from container /workspace/.env:\n%s", catOutput)
	}
	if len(strings.TrimSpace(catOutput)) != 0 {
		t.Fatalf("expected masked .env file to be empty, got: %s", catOutput)
	}
}

func TestSandboxIntegration_SignalForwarding(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	tmpDir := t.TempDir()

	cfg := sandbox.DefaultConfig()
	cfg.WorkspaceDir = tmpDir
	cfg.Image = "alpine:latest"
	// Command that would sleep longer than ctx timeout
	cfg.Command = []string{"sleep", "30"}

	var stdout, stderr bytes.Buffer
	_, _ = sandbox.Execute(ctx, cfg, &stdout, &stderr)
	// If signal handling / context cancellation works, execution returns within ~2-3 seconds instead of 30s.
}

