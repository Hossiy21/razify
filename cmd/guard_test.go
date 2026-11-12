package cmd

import (
	"bytes"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

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

	rootCmd.SetArgs([]string{"guard", "--help"})
	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("guard --help failed: %v", err)
	}

	output := buf.String()
	if !strings.Contains(output, "--agent") {
		t.Errorf("expected guard --help to list --agent, got: %s", output)
	}
}

func TestGuard_PreCommitLifecycle_TempGitRepo(t *testing.T) {
	tempGitDir := t.TempDir()
	t.Chdir(tempGitDir)

	// Initialize real git repo
	cmd := exec.Command("git", "init")
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("git init failed: %v, out: %s", err, string(out))
	}
	_ = exec.Command("git", "config", "user.email", "test@example.com").Run()
	_ = exec.Command("git", "config", "user.name", "Test").Run()

	// 1. Initial status before install
	hookPath, err := getHookPath()
	if err != nil {
		t.Fatalf("getHookPath failed: %v", err)
	}
	if _, err := os.Stat(hookPath); !os.IsNotExist(err) {
		t.Fatalf("hook should not exist before install")
	}

	// 2. Install guard
	installGuard()

	hookData, err := os.ReadFile(hookPath)
	if err != nil {
		t.Fatalf("failed to read installed hook: %v", err)
	}
	if !strings.Contains(string(hookData), hookMarkerStart) {
		t.Errorf("installed hook missing marker start: %s", string(hookData))
	}
	if !strings.Contains(string(hookData), "razify check --quiet") {
		t.Errorf("installed hook missing 'razify check --quiet': %s", string(hookData))
	}

	// 3. Idempotent install: installing again should not duplicate hook
	installGuard()
	hookDataSecond, err := os.ReadFile(hookPath)
	if err != nil {
		t.Fatalf("failed to read hook after second install: %v", err)
	}
	if string(hookData) != string(hookDataSecond) {
		t.Fatalf("installGuard is not idempotent: content changed on second run")
	}

	// 4. Clean uninstall of standalone hook: removes the file entirely
	uninstallGuard()
	if _, err := os.Stat(hookPath); !os.IsNotExist(err) {
		t.Fatalf("hook file should be completely removed on clean uninstall")
	}

	// 5. Existing user hook preservation on uninstall
	customUserHook := "#!/bin/sh\n\necho \"Running user custom pre-commit test\"\nexit 0\n"
	if err := os.MkdirAll(filepath.Dir(hookPath), 0755); err != nil {
		t.Fatalf("failed to create hooks dir: %v", err)
	}
	if err := os.WriteFile(hookPath, []byte(customUserHook), 0755); err != nil {
		t.Fatalf("failed to write custom user hook: %v", err)
	}

	// Install on top of existing user hook
	installGuard()
	combinedData, err := os.ReadFile(hookPath)
	if err != nil {
		t.Fatalf("failed to read combined hook: %v", err)
	}
	if !strings.Contains(string(combinedData), "Running user custom pre-commit test") {
		t.Fatalf("custom user hook command was wiped during installGuard!")
	}
	if !strings.Contains(string(combinedData), hookMarkerStart) {
		t.Fatalf("razify guard block missing from combined hook!")
	}

	// Uninstall: MUST preserve user's custom hook!
	uninstallGuard()
	afterUninstall, err := os.ReadFile(hookPath)
	if err != nil {
		t.Fatalf("user's hook file was deleted when it contained custom commands!")
	}
	if strings.Contains(string(afterUninstall), hookMarkerStart) || strings.Contains(string(afterUninstall), "razify check") {
		t.Errorf("razify guard was not removed: %s", string(afterUninstall))
	}
	if !strings.Contains(string(afterUninstall), "Running user custom pre-commit test") {
		t.Errorf("user custom hook command was lost during uninstall: %s", string(afterUninstall))
	}

	// 6. Test core.hooksPath configuration
	customHooksDir := ".custom_git_hooks"
	if err := exec.Command("git", "config", "core.hooksPath", customHooksDir).Run(); err != nil {
		t.Fatalf("failed to set core.hooksPath: %v", err)
	}

	customHookPath, err := getHookPath()
	if err != nil {
		t.Fatalf("getHookPath failed with core.hooksPath: %v", err)
	}
	expectedCustomPath := filepath.Join(tempGitDir, customHooksDir, "pre-commit")
	if filepath.Clean(customHookPath) != filepath.Clean(expectedCustomPath) {
		t.Fatalf("getHookPath did not honor core.hooksPath! Expected %s, got %s", expectedCustomPath, customHookPath)
	}

	installGuard()
	if _, err := os.Stat(customHookPath); err != nil {
		t.Fatalf("failed to install hook into core.hooksPath location: %v", err)
	}

	uninstallGuard()
	if _, err := os.Stat(customHookPath); !os.IsNotExist(err) {
		t.Fatalf("hook in core.hooksPath location was not uninstalled!")
	}

	// 7. Test Agent Guard Lifecycle
	runAgentGuard("install")

	// Verify .gitignore
	gitignoreBytes, err := os.ReadFile(filepath.Join(tempGitDir, ".gitignore"))
	if err != nil || !strings.Contains(string(gitignoreBytes), ".env") {
		t.Errorf(".gitignore not updated by agent guard")
	}

	// Verify AGENTS.md
	agentsBytes, err := os.ReadFile(filepath.Join(tempGitDir, "AGENTS.md"))
	if err != nil || !strings.Contains(string(agentsBytes), "Razify Agent Guard") {
		t.Errorf("AGENTS.md not created by agent guard")
	}

	// Verify native agent deny files (.cursorignore, .claudeignore, .aiderignore)
	for _, fn := range []string{".cursorignore", ".claudeignore", ".aiderignore"} {
		data, err := os.ReadFile(filepath.Join(tempGitDir, fn))
		if err != nil {
			t.Errorf("agent ignore file %s missing after install", fn)
		} else if !strings.Contains(string(data), ".env") {
			t.Errorf("agent ignore file %s does not contain .env rule", fn)
		}
	}

	// Uninstall Agent Guard
	runAgentGuard("uninstall")

	// Verify AGENTS.md removed or cleaned
	if _, err := os.Stat(filepath.Join(tempGitDir, "AGENTS.md")); !os.IsNotExist(err) {
		content, _ := os.ReadFile(filepath.Join(tempGitDir, "AGENTS.md"))
		if strings.Contains(string(content), "razify:agent-guard") {
			t.Errorf("AGENTS.md still contains razify marker after uninstall")
		}
	}

	// Verify agent ignore files cleaned up
	for _, fn := range []string{".cursorignore", ".claudeignore", ".aiderignore"} {
		if _, err := os.Stat(filepath.Join(tempGitDir, fn)); !os.IsNotExist(err) {
			t.Errorf("agent ignore file %s was not cleaned up after uninstall", fn)
		}
	}
}
