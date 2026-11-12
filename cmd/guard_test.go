package cmd

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestGuardCmd_PreCommitLifecycle(t *testing.T) {
	tempDir := t.TempDir()
	gitDir := filepath.Join(tempDir, ".git", "hooks")
	if err := os.MkdirAll(gitDir, 0755); err != nil {
		t.Fatalf("failed to create fake .git/hooks: %v", err)
	}

	hookPath := filepath.Join(gitDir, "pre-commit")

	// Verify initial absence
	if _, err := os.Stat(hookPath); !os.IsNotExist(err) {
		t.Errorf("hook should not exist initially")
	}

	// Write hook
	if err := os.WriteFile(hookPath, []byte(hookScript), 0755); err != nil {
		t.Fatalf("failed to write hook: %v", err)
	}

	data, err := os.ReadFile(hookPath)
	if err != nil {
		t.Fatalf("failed to read hook: %v", err)
	}
	if !strings.Contains(string(data), "razify guard") {
		t.Errorf("expected hook to contain 'razify guard'")
	}
}

func TestGuardCmd_AgentFlagRegistration(t *testing.T) {
	flag := guardCmd.Flags().Lookup("agent")
	if flag == nil {
		t.Fatal("expected --agent flag on guardCmd")
	}
	if flag.DefValue != "false" {
		t.Errorf("expected --agent default 'false', got '%s'", flag.DefValue)
	}
}

func TestGuardCmd_ExecutionOutput(t *testing.T) {
	buf := new(bytes.Buffer)
	rootCmd.SetOut(buf)
	rootCmd.SetErr(buf)

	// Test help / execution
	rootCmd.SetArgs([]string{"guard", "--help"})
	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("guard --help failed: %v", err)
	}

	output := buf.String()
	if !strings.Contains(output, "--agent") {
		t.Errorf("expected guard --help to list --agent, got: %s", output)
	}
}
