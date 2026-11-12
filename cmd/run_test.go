package cmd

import (
	"bytes"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

func buildTestBinary(t *testing.T) string {
	t.Helper()
	binName := "razify_run_test"
	if runtime.GOOS == "windows" {
		binName += ".exe"
	}
	binPath := filepath.Join(t.TempDir(), binName)
	cmd := exec.Command("go", "build", "-o", binPath, "..")
	if err := cmd.Run(); err != nil {
		t.Fatalf("failed to build razify test binary: %v", err)
	}
	return binPath
}

func TestRunCmd_RequiresSandboxFlag(t *testing.T) {
	binPath := buildTestBinary(t)

	cmd := exec.Command(binPath, "run", "--", "echo", "hello")
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	err := cmd.Run()
	if err == nil {
		t.Fatal("expected razify run without --sandbox to fail with exit code 2")
	}

	exitErr, ok := err.(*exec.ExitError)
	if !ok {
		t.Fatalf("failed to get exit error: %v", err)
	}
	if exitErr.ExitCode() != 2 {
		t.Errorf("expected exit code 2, got %d", exitErr.ExitCode())
	}

	if !strings.Contains(stderr.String(), "requires the --sandbox flag") {
		t.Errorf("expected error message requiring --sandbox flag, got: %s", stderr.String())
	}
}

func TestRunCmd_RequiresCommand(t *testing.T) {
	binPath := buildTestBinary(t)

	cmd := exec.Command(binPath, "run", "--sandbox")
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	err := cmd.Run()
	if err == nil {
		t.Fatal("expected razify run --sandbox without command to fail with exit code 2")
	}

	exitErr, ok := err.(*exec.ExitError)
	if !ok {
		t.Fatalf("failed to get exit error: %v", err)
	}
	if exitErr.ExitCode() != 2 {
		t.Errorf("expected exit code 2, got %d", exitErr.ExitCode())
	}

	if !strings.Contains(stderr.String(), "no command specified to run") {
		t.Errorf("expected error message requiring command, got: %s", stderr.String())
	}
}

func TestRunCmd_ProhibitedGuardrails(t *testing.T) {
	binPath := buildTestBinary(t)

	tests := []struct {
		name       string
		args       []string
		wantErrMsg string
	}{
		{
			name:       "privileged rejected",
			args:       []string{"run", "--sandbox", "--privileged", "--", "echo", "test"},
			wantErrMsg: "--privileged is prohibited",
		},
		{
			name:       "host network rejected",
			args:       []string{"run", "--sandbox", "--network", "host", "--", "echo", "test"},
			wantErrMsg: "--network host is prohibited",
		},
		{
			name:       "host pid rejected",
			args:       []string{"run", "--sandbox", "--pid", "host", "--", "echo", "test"},
			wantErrMsg: "--pid=host is prohibited",
		},
		{
			name:       "unconfined seccomp rejected",
			args:       []string{"run", "--sandbox", "--seccomp", "unconfined", "--", "echo", "test"},
			wantErrMsg: "disabling seccomp is prohibited",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			cmd := exec.Command(binPath, tc.args...)
			var stdout, stderr bytes.Buffer
			cmd.Stdout = &stdout
			cmd.Stderr = &stderr

			err := cmd.Run()
			if err == nil {
				t.Fatalf("expected guardrail error, got exit 0")
			}
			exitErr, ok := err.(*exec.ExitError)
			if !ok || exitErr.ExitCode() != 2 {
				t.Errorf("expected exit code 2, got %v", err)
			}
			if !strings.Contains(stderr.String(), tc.wantErrMsg) {
				t.Errorf("expected stderr to contain %q, got: %s", tc.wantErrMsg, stderr.String())
			}
		})
	}
}

func TestRunCmd_DryRun(t *testing.T) {
	binPath := buildTestBinary(t)

	tmpDir := t.TempDir()
	envPath := filepath.Join(tmpDir, ".env")
	examplePath := filepath.Join(tmpDir, ".env.example")
	mockSecret := "sk_live_" + "000000000000000000000000"
	_ = os.WriteFile(envPath, []byte("PORT=9000\nAPI_KEY="+mockSecret+"\n"), 0600)
	_ = os.WriteFile(examplePath, []byte("PORT=9000\nAPI_KEY=\n"), 0644)

	cmd := exec.Command(
		binPath,
		"run",
		"--sandbox",
		"--dry-run",
		"--env-file", envPath,
		"--example-file", examplePath,
		"--",
		"go", "test", "./...",
	)
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		t.Fatalf("dry-run failed: %v, stderr: %s", err, stderr.String())
	}

	out := stdout.String()
	if !strings.HasPrefix(out, "docker run") {
		t.Errorf("expected docker run prefix, got: %s", out)
	}

	// Verify security defaults in argv
	if !strings.Contains(out, "--network none") {
		t.Errorf("missing --network none in dry-run output: %s", out)
	}
	if !strings.Contains(out, "--cap-drop=ALL") {
		t.Errorf("missing --cap-drop=ALL in dry-run output: %s", out)
	}
	if !strings.Contains(out, "--security-opt no-new-privileges") {
		t.Errorf("missing no-new-privileges in dry-run output: %s", out)
	}
	if !strings.Contains(out, "--memory 2g") {
		t.Errorf("missing memory limit in dry-run output: %s", out)
	}
	if !strings.Contains(out, "--read-only") {
		t.Errorf("missing --read-only in dry-run output: %s", out)
	}

	// Verify raw secret NEVER appears in argv
	if strings.Contains(out, mockSecret) {
		t.Fatalf("SECURITY LEAK: raw secret found in dry-run argv:\n%s", out)
	}

	// Verify fake secret does NOT appear in argv either (only -e API_KEY is present)
	if strings.Contains(out, "-e API_KEY=") {
		t.Fatalf("SECURITY VIOLATION: inline env assignment in argv:\n%s", out)
	}
	if !strings.Contains(out, "-e API_KEY") {
		t.Errorf("expected -e API_KEY in argv:\n%s", out)
	}
}

func TestRunCmd_WarningsOnStderr(t *testing.T) {
	binPath := buildTestBinary(t)

	cmd := exec.Command(
		binPath,
		"run",
		"--sandbox",
		"--dry-run",
		"--network", "bridge",
		"--read-write",
		"--inject-secrets",
		"--",
		"echo", "hello",
	)
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		t.Fatalf("command failed: %v, stderr: %s", err, stderr.String())
	}

	errOut := stderr.String()
	if !strings.Contains(errOut, "Network egress enabled") {
		t.Errorf("expected network warning on stderr, got: %s", errOut)
	}
	if !strings.Contains(errOut, "Workspace mounted read-write") {
		t.Errorf("expected read-write warning on stderr, got: %s", errOut)
	}
	if !strings.Contains(errOut, "Real secrets injected") {
		t.Errorf("expected inject secrets warning on stderr, got: %s", errOut)
	}
}

