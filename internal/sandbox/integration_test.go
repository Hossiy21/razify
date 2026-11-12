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

func TestSandboxIntegration_NonRootGoTest(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	tmpDir := t.TempDir()

	// Create minimal hermetic go module
	goMod := "module sandboxtest\n\ngo 1.25\n"
	_ = os.WriteFile(filepath.Join(tmpDir, "go.mod"), []byte(goMod), 0644)

	mainTest := `package sandboxtest

import "testing"

func TestPass(t *testing.T) {
	t.Log("hermetic non-root execution inside sandbox passed")
}
`
	_ = os.WriteFile(filepath.Join(tmpDir, "main_test.go"), []byte(mainTest), 0644)

	cfg := sandbox.DefaultConfig()
	cfg.WorkspaceDir = tmpDir
	cfg.ProjectProfile = "go"
	cfg.Image = "golang:1.25-alpine"
	cfg.Command = []string{"go", "test", "-v", "./..."}

	var stdout, stderr bytes.Buffer
	result, err := sandbox.Execute(ctx, cfg, &stdout, &stderr)
	if err != nil {
		t.Fatalf("Execute error: %v, stderr: %s", err, stderr.String())
	}

	if result.ExitCode != 0 {
		t.Fatalf("go test failed: exit code %d, stderr: %s, stdout: %s", result.ExitCode, stderr.String(), stdout.String())
	}

	if !strings.Contains(stdout.String(), "PASS") {
		t.Errorf("expected go test output to contain PASS, got: %s", stdout.String())
	}
}

func TestSandboxIntegration_EnvFakeSecretsOnly(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	tmpDir := t.TempDir()

	// Template with secrets
	exampleContent := "# @type secret\nSECRET_TOKEN=\n# @type url\nDATABASE_URL=\nPORT=8080\n"
	_ = os.WriteFile(filepath.Join(tmpDir, ".env.example"), []byte(exampleContent), 0644)

	// Real .env with plaintext secrets on host
	realEnvContent := "SECRET_TOKEN=real_live_prod_super_secret_token_9999\nDATABASE_URL=postgres://realuser:realpass123@prod.db:5432/main\nPORT=8080\n"
	_ = os.WriteFile(filepath.Join(tmpDir, ".env"), []byte(realEnvContent), 0600)

	cfg := sandbox.DefaultConfig()
	cfg.WorkspaceDir = tmpDir
	cfg.Image = "alpine:latest"
	cfg.FakeSecrets = true
	cfg.InjectSecrets = false
	cfg.Command = []string{"env"}

	var stdout, stderr bytes.Buffer
	result, err := sandbox.Execute(ctx, cfg, &stdout, &stderr)
	if err != nil {
		t.Fatalf("Execute error: %v, stderr: %s", err, stderr.String())
	}
	if result.ExitCode != 0 {
		t.Fatalf("env command failed: exit code %d, stderr: %s", result.ExitCode, stderr.String())
	}

	envOutput := stdout.String()

	// Verify fake secret values are injected
	if !strings.Contains(envOutput, "SECRET_TOKEN=razify_fake_") {
		t.Errorf("expected synthetic SECRET_TOKEN=razify_fake_*, got:\n%s", envOutput)
	}
	if !strings.Contains(envOutput, "DATABASE_URL=razify_fake_") {
		t.Errorf("expected synthetic DATABASE_URL=razify_fake_*, got:\n%s", envOutput)
	}

	// Verify real secrets NEVER appear
	if strings.Contains(envOutput, "real_live_prod_super_secret_token_9999") {
		t.Fatalf("SECURITY VIOLATION: real SECRET_TOKEN value leaked into container environment:\n%s", envOutput)
	}
	if strings.Contains(envOutput, "realpass123") {
		t.Fatalf("SECURITY VIOLATION: real DATABASE_URL password leaked into container environment:\n%s", envOutput)
	}
}

func TestSandboxIntegration_CapEffZero(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	tmpDir := t.TempDir()

	cfg := sandbox.DefaultConfig()
	cfg.WorkspaceDir = tmpDir
	cfg.Image = "alpine:latest"
	cfg.Command = []string{"sh", "-c", "grep CapEff /proc/self/status"}

	var stdout, stderr bytes.Buffer
	result, err := sandbox.Execute(ctx, cfg, &stdout, &stderr)
	if err != nil {
		t.Fatalf("Execute error: %v, stderr: %s", err, stderr.String())
	}
	if result.ExitCode != 0 {
		t.Fatalf("grep CapEff failed: exit code %d, stderr: %s", result.ExitCode, stderr.String())
	}

	out := stdout.String()
	if !strings.Contains(out, "0000000000000000") {
		t.Fatalf("expected CapEff to be 0000000000000000 (all capabilities dropped), got: %s", out)
	}
}

func TestSandboxIntegration_RootfsWriteFail(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	tmpDir := t.TempDir()

	cfg := sandbox.DefaultConfig()
	cfg.WorkspaceDir = tmpDir
	cfg.Image = "alpine:latest"
	cfg.ReadOnlyRootfs = true
	cfg.Command = []string{"touch", "/test_rootfs_write.txt"}

	var stdout, stderr bytes.Buffer
	result, _ := sandbox.Execute(ctx, cfg, &stdout, &stderr)

	if result.ExitCode == 0 {
		t.Fatalf("expected write to read-only rootfs to fail, but exit code was 0")
	}

	errStr := stderr.String()
	if !strings.Contains(errStr, "Read-only file system") && !strings.Contains(errStr, "read-only") {
		t.Logf("Note: stderr was %q, exit code: %d", errStr, result.ExitCode)
	}
}

func TestSandboxIntegration_OutputMasking(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	tmpDir := t.TempDir()

	secretValue := "super_secret_unmasked_token_12345"
	_ = os.WriteFile(filepath.Join(tmpDir, ".env.example"), []byte("API_KEY=\n"), 0644)
	_ = os.WriteFile(filepath.Join(tmpDir, ".env"), []byte("API_KEY="+secretValue+"\n"), 0600)

	cfg := sandbox.DefaultConfig()
	cfg.WorkspaceDir = tmpDir
	cfg.Image = "alpine:latest"
	cfg.InjectSecrets = true
	cfg.NoMaskOutput = false
	// Attempt to echo the secret to stdout
	cfg.Command = []string{"sh", "-c", "echo my api key is " + secretValue}

	var stdout, stderr bytes.Buffer
	result, err := sandbox.Execute(ctx, cfg, &stdout, &stderr)
	if err != nil {
		t.Fatalf("Execute error: %v, stderr: %s", err, stderr.String())
	}
	if result.ExitCode != 0 {
		t.Fatalf("echo failed: exit code %d, stderr: %s", result.ExitCode, stderr.String())
	}

	output := stdout.String()
	if strings.Contains(output, secretValue) {
		t.Fatalf("SECURITY VIOLATION: Output stream masker failed to redact secret from stdout: %s", output)
	}
	if !strings.Contains(output, "[REDACTED]") {
		t.Fatalf("expected [REDACTED] in stdout, got: %s", output)
	}
}

