package mcp

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

// Agent guard policy markers and template.
const (
	AgentPolicyMarkerStart = "<!-- razify:agent-guard:start -->"
	AgentPolicyMarkerEnd   = "<!-- razify:agent-guard:end -->"
)

// AgentPolicyContent returns the standardized markdown policy directive for AI coding agents.
func AgentPolicyContent() string {
	return AgentPolicyMarkerStart + `
## Environment & Secret Protection (Razify Agent Guard)
This repository enforces strict environment variable and secret protection via Razify.
- **NEVER** attempt to directly read, open, or inspect '.env', '.env.local', or any '.env.*' secret files.
- Use the 'razify mcp' tools to interact with environment configurations safely:
  - 'schema_inspect': View required keys, types (@type), and schema rules from .env.example.
  - 'env_validate_config': Validate candidate configurations against schema rules without writing secrets to disk.
  - 'env_diff_keys': Check missing or untracked keys between .env and .env.example safely.
  - 'env_health_audit': Audit configuration completeness and health score.
- When executing builds, tests, or commands, run inside container containment using 'razify run --sandbox -- <cmd>'.
` + AgentPolicyMarkerEnd
}

// CheckGitTracking inspects if any .env or .env.* files are tracked in the git index.
func CheckGitTracking(repoRoot string) ([]string, error) {
	cmd := exec.Command("git", "ls-files", ".env", ".env.*")
	cmd.Dir = repoRoot
	out, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("git ls-files failed: %w", err)
	}

	lines := strings.Split(strings.TrimSpace(string(out)), "\n")
	var tracked []string
	for _, l := range lines {
		trimmed := strings.TrimSpace(l)
		if trimmed != "" && !strings.HasSuffix(trimmed, ".example") && !strings.HasSuffix(trimmed, ".template") {
			tracked = append(tracked, trimmed)
		}
	}
	return tracked, nil
}

// CheckGitIgnore returns true if .env is ignored by git rules.
func CheckGitIgnore(repoRoot string) (bool, error) {
	cmd := exec.Command("git", "check-ignore", ".env")
	cmd.Dir = repoRoot
	if err := cmd.Run(); err == nil {
		return true, nil
	}

	// Fallback check in .gitignore file directly
	gitignorePath := filepath.Join(repoRoot, ".gitignore")
	data, err := os.ReadFile(gitignorePath)
	if err != nil {
		if os.IsNotExist(err) {
			return false, nil
		}
		return false, err
	}

	for _, line := range strings.Split(string(data), "\n") {
		trimmed := strings.TrimSpace(line)
		if trimmed == ".env" || trimmed == ".env*" || trimmed == "*.env" {
			return true, nil
		}
	}

	return false, nil
}

// EnsureGitIgnore appends .env rules to .gitignore if not already ignored.
func EnsureGitIgnore(repoRoot string) (bool, error) {
	ignored, _ := CheckGitIgnore(repoRoot)
	if ignored {
		return false, nil
	}

	gitignorePath := filepath.Join(repoRoot, ".gitignore")
	var content bytes.Buffer

	existing, err := os.ReadFile(gitignorePath)
	if err == nil {
		content.Write(existing)
		if len(existing) > 0 && !bytes.HasSuffix(existing, []byte("\n")) {
			content.WriteString("\n")
		}
	}

	content.WriteString("\n# Environment secrets (protected by Razify Agent Guard)\n.env\n.env.*\n!.env.example\n!.env.template\n")

	if err := os.WriteFile(gitignorePath, content.Bytes(), 0644); err != nil {
		return false, fmt.Errorf("failed to write .gitignore: %w", err)
	}
	return true, nil
}

// CheckAgentPolicy returns true if AGENTS.md exists and contains the Razify guard marker.
func CheckAgentPolicy(repoRoot string) (bool, error) {
	candidates := []string{
		filepath.Join(repoRoot, "AGENTS.md"),
		filepath.Join(repoRoot, "CLAUDE.md"),
		filepath.Join(repoRoot, ".cursorrules"),
		filepath.Join(repoRoot, ".github", "copilot-instructions.md"),
	}

	for _, path := range candidates {
		data, err := os.ReadFile(path)
		if err == nil && strings.Contains(string(data), AgentPolicyMarkerStart) {
			return true, nil
		}
	}
	return false, nil
}

// InstallAgentPolicy creates or updates AGENTS.md with the Razify agent guard directives.
func InstallAgentPolicy(repoRoot string) (bool, error) {
	agentsPath := filepath.Join(repoRoot, "AGENTS.md")
	data, err := os.ReadFile(agentsPath)
	if os.IsNotExist(err) {
		initial := "# Agent Guidelines\n\n" + AgentPolicyContent() + "\n"
		if err := os.WriteFile(agentsPath, []byte(initial), 0644); err != nil {
			return false, fmt.Errorf("failed to create AGENTS.md: %w", err)
		}
		return true, nil
	} else if err != nil {
		return false, fmt.Errorf("failed to read AGENTS.md: %w", err)
	}

	content := string(data)
	if strings.Contains(content, AgentPolicyMarkerStart) {
		return false, nil // already installed
	}

	var sb strings.Builder
	sb.WriteString(content)
	if !strings.HasSuffix(content, "\n") {
		sb.WriteString("\n")
	}
	sb.WriteString("\n" + AgentPolicyContent() + "\n")

	if err := os.WriteFile(agentsPath, []byte(sb.String()), 0644); err != nil {
		return false, fmt.Errorf("failed to update AGENTS.md: %w", err)
	}
	return true, nil
}

// UninstallAgentPolicy removes Razify agent guard directives from AGENTS.md.
func UninstallAgentPolicy(repoRoot string) (bool, error) {
	agentsPath := filepath.Join(repoRoot, "AGENTS.md")
	data, err := os.ReadFile(agentsPath)
	if err != nil {
		if os.IsNotExist(err) {
			return false, nil
		}
		return false, err
	}

	content := string(data)
	startIdx := strings.Index(content, AgentPolicyMarkerStart)
	endIdx := strings.Index(content, AgentPolicyMarkerEnd)
	if startIdx == -1 || endIdx == -1 || endIdx < startIdx {
		return false, nil
	}

	endIdx += len(AgentPolicyMarkerEnd)
	// Remove markers and slice out policy
	updated := content[:startIdx] + content[endIdx:]
	updated = strings.TrimSpace(updated)

	if updated == "" || updated == "# Agent Guidelines" {
		_ = os.Remove(agentsPath)
		return true, nil
	}

	if err := os.WriteFile(agentsPath, []byte(updated+"\n"), 0644); err != nil {
		return false, fmt.Errorf("failed to write AGENTS.md: %w", err)
	}
	return true, nil
}

// AgentIgnoreFileList returns the slice of native agent ignore files.
var AgentIgnoreFileList = []string{".cursorignore", ".claudeignore", ".aiderignore"}

const AgentIgnoreContent = `# Protected by Razify Agent Guard
.env
.env.*
!.env.example
!.env.sample
!.env.template
`

// EnsureAgentIgnoreFiles creates or updates .cursorignore, .claudeignore, and .aiderignore.
func EnsureAgentIgnoreFiles(repoRoot string) ([]string, error) {
	var createdOrUpdated []string
	for _, filename := range AgentIgnoreFileList {
		p := filepath.Join(repoRoot, filename)
		data, err := os.ReadFile(p)
		if os.IsNotExist(err) {
			if err := os.WriteFile(p, []byte(AgentIgnoreContent), 0644); err != nil {
				return createdOrUpdated, fmt.Errorf("failed to create %s: %w", filename, err)
			}
			createdOrUpdated = append(createdOrUpdated, filename)
			continue
		} else if err != nil {
			return createdOrUpdated, fmt.Errorf("failed to read %s: %w", filename, err)
		}

		content := string(data)
		if !strings.Contains(content, ".env") {
			var sb strings.Builder
			sb.WriteString(content)
			if !strings.HasSuffix(content, "\n") {
				sb.WriteString("\n")
			}
			sb.WriteString("\n" + AgentIgnoreContent)
			if err := os.WriteFile(p, []byte(sb.String()), 0644); err != nil {
				return createdOrUpdated, fmt.Errorf("failed to update %s: %w", filename, err)
			}
			createdOrUpdated = append(createdOrUpdated, filename)
		}
	}
	return createdOrUpdated, nil
}

// RemoveAgentIgnoreFiles removes or cleans up agent ignore files created by Razify.
func RemoveAgentIgnoreFiles(repoRoot string) ([]string, error) {
	var removed []string
	for _, filename := range AgentIgnoreFileList {
		p := filepath.Join(repoRoot, filename)
		data, err := os.ReadFile(p)
		if err != nil {
			continue
		}

		content := strings.TrimSpace(string(data))
		// If the file only contains our ignore content or .env rules, remove it
		if strings.TrimSpace(content) == strings.TrimSpace(AgentIgnoreContent) {
			_ = os.Remove(p)
			removed = append(removed, filename)
		}
	}
	return removed, nil
}

// SampleMCPConfigCursor returns formatted JSON config for Cursor (.cursor/mcp.json).
func SampleMCPConfigCursor() string {
	return `{
  "mcpServers": {
    "razify": {
      "command": "razify",
      "args": ["mcp"]
    }
  }
}`
}

// SampleMCPConfigClaudeDesktop returns formatted JSON config for Claude Desktop.
func SampleMCPConfigClaudeDesktop() string {
	return `{
  "mcpServers": {
    "razify": {
      "command": "razify",
      "args": ["mcp"]
    }
  }
}`
}

// SampleMCPConfigAntigravity returns formatted JSON config for Antigravity / Gemini CLI.
func SampleMCPConfigAntigravity() string {
	return `{
  "mcpServers": {
    "razify": {
      "command": "razify",
      "args": ["mcp"]
    }
  }
}`
}
