package sandbox_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/Hossiy21/razify/internal/sandbox"
)

func TestBuildDockerArgs_Defaults(t *testing.T) {
	tmpDir := t.TempDir()

	cfg := sandbox.DefaultConfig()
	cfg.WorkspaceDir = tmpDir
	cfg.Command = []string{"go", "test", "./..."}

	profile := sandbox.GetProfileConfig("go")
	injectedEnv := map[string]string{
		"PORT":       "8080",
		"STRIPE_KEY": "razify_fake_stripe_key_12345678",
	}

	args, envMap, err := sandbox.BuildDockerArgs(cfg, nil, "", injectedEnv, profile)
	if err != nil {
		t.Fatalf("BuildDockerArgs failed: %v", err)
	}

	argsStr := strings.Join(args, " ")

	// Base containment
	expectedFlags := []string{
		"run",
		"--rm",
		"--init",
		"--cap-drop=ALL",
		"--security-opt no-new-privileges",
		"--network none",
		"--memory 2g",
		"--cpus 2.0",
		"--pids-limit 256",
		"--read-only",
		"-w /workspace",
		"golang:1.25-alpine",
		"go test ./...",
	}

	for _, flag := range expectedFlags {
		if !strings.Contains(argsStr, flag) {
			t.Errorf("expected flag %q in args: %s", flag, argsStr)
		}
	}

	// Verify secrets are NOT in argv
	if strings.Contains(argsStr, "razify_fake_stripe_key_12345678") {
		t.Fatalf("CRITICAL: secret value leaked in argv: %s", argsStr)
	}
	if strings.Contains(argsStr, "-e STRIPE_KEY=") || strings.Contains(argsStr, "-e PORT=") {
		t.Fatalf("CRITICAL: inline environment assignment in argv: %s", argsStr)
	}

	// Verify -e KEY format
	if !strings.Contains(argsStr, "-e STRIPE_KEY") || !strings.Contains(argsStr, "-e PORT") {
		t.Errorf("expected -e KEY in args: %s", argsStr)
	}

	// Verify envMap contains the values
	if envMap["PORT"] != "8080" || envMap["STRIPE_KEY"] != "razify_fake_stripe_key_12345678" {
		t.Errorf("envMap missing expected values: %+v", envMap)
	}
}

func TestBuildDockerArgs_GitProtectionUnderReadWrite(t *testing.T) {
	tmpDir := t.TempDir()
	gitDir := filepath.Join(tmpDir, ".git")
	_ = os.MkdirAll(gitDir, 0755)

	cfg := sandbox.DefaultConfig()
	cfg.WorkspaceDir = tmpDir
	cfg.ReadOnlyWorkspace = false // --read-write requested
	cfg.Command = []string{"echo", "hi"}

	profile := sandbox.GetProfileConfig("generic")
	args, _, err := sandbox.BuildDockerArgs(cfg, nil, "", nil, profile)
	if err != nil {
		t.Fatalf("BuildDockerArgs failed: %v", err)
	}

	argsStr := strings.Join(args, " ")

	// Workspace should be :rw
	if !strings.Contains(argsStr, "/workspace:rw") {
		t.Errorf("expected workspace to be mounted :rw, got: %s", argsStr)
	}

	// .git MUST remain mounted :ro
	if !strings.Contains(argsStr, "/workspace/.git:ro") {
		t.Fatalf(".git was NOT protected with :ro under --read-write: %s", argsStr)
	}
}

func TestBuildDockerArgs_EnvMasking(t *testing.T) {
	tmpDir := t.TempDir()
	emptyMask := filepath.Join(tmpDir, "empty.mask")
	_ = os.WriteFile(emptyMask, nil, 0644)

	cfg := sandbox.DefaultConfig()
	cfg.WorkspaceDir = tmpDir
	cfg.Command = []string{"ls"}

	profile := sandbox.GetProfileConfig("generic")
	maskedFiles := []string{".env", ".env.local"}

	args, _, err := sandbox.BuildDockerArgs(cfg, maskedFiles, emptyMask, nil, profile)
	if err != nil {
		t.Fatalf("BuildDockerArgs failed: %v", err)
	}

	argsStr := strings.Join(args, " ")

	if !strings.Contains(argsStr, "/workspace/.env:ro") {
		t.Errorf("missing .env masking mount: %s", argsStr)
	}
	if !strings.Contains(argsStr, "/workspace/.env.local:ro") {
		t.Errorf("missing .env.local masking mount: %s", argsStr)
	}
}

func TestBuildDockerArgs_ImagePinning(t *testing.T) {
	tmpDir := t.TempDir()
	pinnedImage := "golang:1.25-alpine@sha256:7198d022fa50873a4369e599eb9e7f72260682e04db149ecba8e8aa2c069bc92"

	cfg := sandbox.DefaultConfig()
	cfg.WorkspaceDir = tmpDir
	cfg.Image = pinnedImage
	cfg.Command = []string{"go", "version"}

	profile := sandbox.GetProfileConfig("go")
	args, _, err := sandbox.BuildDockerArgs(cfg, nil, "", nil, profile)
	if err != nil {
		t.Fatalf("BuildDockerArgs failed: %v", err)
	}

	argsStr := strings.Join(args, " ")
	if !strings.Contains(argsStr, pinnedImage) {
		t.Errorf("expected pinned image digest in args: %s", argsStr)
	}
}

