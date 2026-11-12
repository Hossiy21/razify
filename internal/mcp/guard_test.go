package mcp

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestEnsureGitIgnore(t *testing.T) {
	tempDir := t.TempDir()

	// Initial check: not ignored
	ignored, err := CheckGitIgnore(tempDir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if ignored {
		t.Errorf("expected not ignored initially")
	}

	// Ensure ignore creates .gitignore
	changed, err := EnsureGitIgnore(tempDir)
	if err != nil {
		t.Fatalf("EnsureGitIgnore failed: %v", err)
	}
	if !changed {
		t.Errorf("expected changed=true on initial run")
	}

	// Now check should report ignored
	ignored, err = CheckGitIgnore(tempDir)
	if err != nil {
		t.Fatalf("CheckGitIgnore failed: %v", err)
	}
	if !ignored {
		t.Errorf("expected ignored=true after EnsureGitIgnore")
	}

	// Second run should be idempotent
	changed, err = EnsureGitIgnore(tempDir)
	if err != nil {
		t.Fatalf("EnsureGitIgnore 2nd call failed: %v", err)
	}
	if changed {
		t.Errorf("expected changed=false on second run")
	}
}

func TestAgentPolicyLifecycle(t *testing.T) {
	tempDir := t.TempDir()

	// Initially false
	active, err := CheckAgentPolicy(tempDir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if active {
		t.Errorf("expected policy inactive initially")
	}

	// Install policy
	installed, err := InstallAgentPolicy(tempDir)
	if err != nil {
		t.Fatalf("InstallAgentPolicy failed: %v", err)
	}
	if !installed {
		t.Errorf("expected installed=true")
	}

	// Check active
	active, err = CheckAgentPolicy(tempDir)
	if err != nil {
		t.Fatalf("CheckAgentPolicy failed: %v", err)
	}
	if !active {
		t.Errorf("expected policy active after install")
	}

	// Verify file content
	content, err := os.ReadFile(filepath.Join(tempDir, "AGENTS.md"))
	if err != nil {
		t.Fatalf("failed to read AGENTS.md: %v", err)
	}
	if !strings.Contains(string(content), AgentPolicyMarkerStart) {
		t.Errorf("expected marker start in AGENTS.md")
	}
	if !strings.Contains(string(content), "schema_inspect") {
		t.Errorf("expected tool references in AGENTS.md")
	}

	// Install again should be idempotent
	installedAgain, err := InstallAgentPolicy(tempDir)
	if err != nil {
		t.Fatalf("InstallAgentPolicy idempotent check failed: %v", err)
	}
	if installedAgain {
		t.Errorf("expected installedAgain=false")
	}

	// Uninstall policy
	uninstalled, err := UninstallAgentPolicy(tempDir)
	if err != nil {
		t.Fatalf("UninstallAgentPolicy failed: %v", err)
	}
	if !uninstalled {
		t.Errorf("expected uninstalled=true")
	}

	// Check active again
	active, err = CheckAgentPolicy(tempDir)
	if err != nil {
		t.Fatalf("CheckAgentPolicy after uninstall failed: %v", err)
	}
	if active {
		t.Errorf("expected policy inactive after uninstall")
	}
}

func TestSampleMCPConfigsValidJSON(t *testing.T) {
	configs := []struct {
		name string
		json string
	}{
		{"Cursor", SampleMCPConfigCursor()},
		{"ClaudeDesktop", SampleMCPConfigClaudeDesktop()},
		{"Antigravity", SampleMCPConfigAntigravity()},
	}

	for _, tc := range configs {
		var parsed map[string]any
		if err := json.Unmarshal([]byte(tc.json), &parsed); err != nil {
			t.Errorf("config for %s is not valid JSON: %v", tc.name, err)
		}
		servers, ok := parsed["mcpServers"].(map[string]any)
		if !ok || servers["razify"] == nil {
			t.Errorf("config for %s missing mcpServers.razify", tc.name)
		}
	}
}
