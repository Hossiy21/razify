package cmd

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/Hossiy21/razify/internal/mcp"
	"github.com/fatih/color"
	"github.com/spf13/cobra"
)

var hookScript = `#!/bin/sh
# Razify Guard — installed by razify guard install
# Runs unified integrity & secret leak check before every commit

if ! command -v razify &> /dev/null && ! command -v npx &> /dev/null; then
    echo "razify guard: razify CLI not found, skipping pre-commit check"
    exit 0
fi

echo "⚡ razify guard: checking configuration integrity..."
if command -v razify &> /dev/null; then
    razify check --quiet
else
    npx --no-install razify check --quiet
fi

if [ $? -ne 0 ]; then
    echo ""
    echo "✘ razify guard: commit blocked due to invalid configuration or secret leak."
    echo "👉 Run 'razify check' to view detailed findings."
    exit 1
fi

exit 0
`

var guardCmd = &cobra.Command{
	Use:   "guard [install|uninstall|status]",
	Short: "Protect your git repo by scanning .env files and guarding AI agent access",
	Args:  cobra.MaximumNArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		agentMode, _ := cmd.Flags().GetBool("agent")
		action := ""
		if len(args) > 0 {
			action = args[0]
		}

		if agentMode {
			runAgentGuard(action)
			return
		}

		if action == "" {
			action = "status"
		}

		switch action {
		case "install":
			installGuard()
		case "uninstall":
			uninstallGuard()
		case "status":
			statusGuard()
		default:
			color.Red("  Unknown action: %s\n", action)
			fmt.Println("  Usage: razify guard [install|uninstall|status] [--agent]")
		}
	},
}

func getGitRoot() (string, error) {
	out, err := exec.Command("git", "rev-parse", "--show-toplevel").Output()
	if err != nil {
		return "", fmt.Errorf("not a git repository")
	}
	return strings.TrimSpace(string(out)), nil
}

func getHookPath() (string, error) {
	root, err := getGitRoot()
	if err != nil {
		return "", err
	}
	return filepath.Join(root, ".git", "hooks", "pre-commit"), nil
}

func installGuard() {
	bold := color.New(color.Bold)
	success := color.New(color.FgGreen, color.Bold)
	errColor := color.New(color.FgRed, color.Bold)

	bold.Println("\nInstalling Razify Guard...")

	hookPath, err := getHookPath()
	if err != nil {
		errColor.Printf("  ✘  %v\n", err)
		os.Exit(2)
	}

	// Check if hook already exists
	if _, err := os.Stat(hookPath); err == nil {
		// Read existing hook
		existing, _ := os.ReadFile(hookPath)
		if strings.Contains(string(existing), "razify guard") {
			color.Yellow("  ⚠  Razify Guard is already installed.\n")
			return
		}
		// Append to existing hook
		f, err := os.OpenFile(hookPath, os.O_APPEND|os.O_WRONLY, 0755)
		if err != nil {
			errColor.Printf("  ✘  Could not modify existing hook: %v\n", err)
			os.Exit(2)
		}
		defer f.Close()
		f.WriteString("\n" + hookScript)
	} else {
		// Create new hook
		err = os.WriteFile(hookPath, []byte(hookScript), 0755)
		if err != nil {
			errColor.Printf("  ✘  Could not install hook: %v\n", err)
			os.Exit(2)
		}
	}

	success.Println("  ✔  Razify Guard installed successfully!")
	fmt.Printf("     Hook location: %s\n", hookPath)
	fmt.Println()
	fmt.Println("  Every git commit in this repo will now be scanned.")
	fmt.Println("  Commits with exposed secrets will be blocked automatically.")
	fmt.Println()
	color.Cyan("  To uninstall: razify guard uninstall\n")
}

func uninstallGuard() {
	bold := color.New(color.Bold)
	success := color.New(color.FgGreen, color.Bold)
	errColor := color.New(color.FgRed, color.Bold)

	bold.Println("\nUninstalling Razify Guard...")

	hookPath, err := getHookPath()
	if err != nil {
		errColor.Printf("  ✘  %v\n", err)
		os.Exit(2)
	}

	if _, err := os.Stat(hookPath); os.IsNotExist(err) {
		color.Yellow("  ⚠  No hook found. Razify Guard was not installed.\n")
		return
	}

	existing, _ := os.ReadFile(hookPath)
	if !strings.Contains(string(existing), "razify guard") {
		color.Yellow("  ⚠  Razify Guard hook not found in pre-commit.\n")
		return
	}

	// Remove the hook file entirely
	err = os.Remove(hookPath)
	if err != nil {
		errColor.Printf("  ✘  Could not remove hook: %v\n", err)
		os.Exit(2)
	}

	success.Println("  ✔  Razify Guard uninstalled successfully.")
	fmt.Println("     Your commits are no longer being scanned.")
}

func statusGuard() {
	bold := color.New(color.Bold)
	success := color.New(color.FgGreen, color.Bold)

	bold.Println("\nRazify Guard Status")
	fmt.Println("  ─────────────────────────────")

	hookPath, err := getHookPath()
	if err != nil {
		color.Red("  ✘  Not a git repository\n")
		return
	}

	if _, err := os.Stat(hookPath); os.IsNotExist(err) {
		color.Yellow("  ⚠  Not installed\n")
		fmt.Println("     Run: razify guard install")
		return
	}

	existing, _ := os.ReadFile(hookPath)
	if strings.Contains(string(existing), "razify guard") {
		success.Println("  ✔  Active — commits are being protected")
		fmt.Printf("     Hook: %s\n", hookPath)
	} else {
		color.Yellow("  ⚠  Pre-commit hook exists but Razify Guard is not in it\n")
		fmt.Println("     Run: razify guard install")
	}
	fmt.Println()
}

func init() {
	rootCmd.AddCommand(guardCmd)
	guardCmd.Flags().Bool("agent", false, "Protect .env from AI coding agents (manage AGENTS.md policy and MCP configuration)")
}

func runAgentGuard(action string) {
	if action == "" {
		action = "status"
	}

	switch action {
	case "install":
		installAgentGuard()
	case "uninstall":
		uninstallAgentGuard()
	case "status":
		statusAgentGuard()
	default:
		color.Red("  Unknown action: %s\n", action)
		fmt.Println("  Usage: razify guard [install|uninstall|status] --agent")
	}
}

func installAgentGuard() {
	bold := color.New(color.Bold)
	success := color.New(color.FgGreen, color.Bold)
	errColor := color.New(color.FgRed, color.Bold)

	bold.Println("\nInstalling Razify Agent Guard...")

	repoRoot, err := getGitRoot()
	if err != nil {
		repoRoot = "."
	}

	// 1. Ensure .env is ignored
	changed, err := mcp.EnsureGitIgnore(repoRoot)
	if err != nil {
		errColor.Printf("  ✘ Failed to update .gitignore: %v\n", err)
	} else if changed {
		success.Println("  ✔ .gitignore updated (added .env protection)")
	} else {
		success.Println("  ✔ .gitignore already protects .env")
	}

	// 2. Install AGENTS.md policy
	policyChanged, err := mcp.InstallAgentPolicy(repoRoot)
	if err != nil {
		errColor.Printf("  ✘ Failed to update AGENTS.md: %v\n", err)
	} else if policyChanged {
		success.Println("  ✔ AGENTS.md policy installed (AI agents guided to use razify MCP)")
	} else {
		success.Println("  ✔ AGENTS.md already contains active Razify policy")
	}

	// 3. Install standard git pre-commit hook if not installed
	hookPath, err := getHookPath()
	if err == nil {
		if _, err := os.Stat(hookPath); os.IsNotExist(err) {
			_ = os.WriteFile(hookPath, []byte(hookScript), 0755)
			success.Println("  ✔ Git pre-commit integrity hook installed")
		}
	}

	fmt.Println()
	bold.Println("Razify Agent Guard is now active!")
	fmt.Println("Coding agents (Cursor, Claude Desktop, Antigravity, Gemini CLI) will avoid reading .env")
	fmt.Println("and utilize safe schema inspection via Model Context Protocol (MCP).")
	fmt.Println()
	color.Cyan("To view MCP setup: razify guard status --agent\n")
}

func uninstallAgentGuard() {
	bold := color.New(color.Bold)
	success := color.New(color.FgGreen, color.Bold)
	errColor := color.New(color.FgRed, color.Bold)

	bold.Println("\nUninstalling Razify Agent Guard...")

	repoRoot, err := getGitRoot()
	if err != nil {
		repoRoot = "."
	}

	removed, err := mcp.UninstallAgentPolicy(repoRoot)
	if err != nil {
		errColor.Printf("  ✘ Failed to update AGENTS.md: %v\n", err)
		return
	}

	if removed {
		success.Println("  ✔ AGENTS.md policy removed successfully.")
	} else {
		color.Yellow("  ⚠ Razify policy was not found in AGENTS.md.\n")
	}
	fmt.Println()
}

func statusAgentGuard() {
	bold := color.New(color.Bold)
	success := color.New(color.FgGreen, color.Bold)
	warning := color.New(color.FgYellow, color.Bold)
	errColor := color.New(color.FgRed, color.Bold)

	bold.Println("\nRazify Agent Guard Status")
	fmt.Println("  ──────────────────────────────────────────")

	repoRoot, err := getGitRoot()
	if err != nil {
		color.Yellow("  ⚠ Not a git repository (checking local directory: .)\n")
		repoRoot = "."
	} else {
		fmt.Printf("  Repository: %s\n\n", repoRoot)
	}

	// 1. Git Tracking Check
	tracked, err := mcp.CheckGitTracking(repoRoot)
	if err != nil {
		warning.Printf("  ⚠ Could not query git tracking: %v\n", err)
	} else if len(tracked) > 0 {
		errColor.Printf("  ✘ TRACKED IN GIT: %s\n", strings.Join(tracked, ", "))
		fmt.Println("    High risk: Secrets are tracked in version control and visible to AI tools!")
		fmt.Printf("    Remediation: git rm --cached %s\n", strings.Join(tracked, " "))
	} else {
		success.Println("  ✔ Git tracking: No .env secret files are tracked in git index")
	}

	// 2. Git Ignore Check
	ignored, err := mcp.CheckGitIgnore(repoRoot)
	if err != nil {
		warning.Printf("  ⚠ Could not verify .gitignore: %v\n", err)
	} else if ignored {
		success.Println("  ✔ Git ignore: .env is properly ignored in .gitignore")
	} else {
		warning.Println("  ⚠ Git ignore: .env is NOT ignored in .gitignore")
		fmt.Println("    Remediation: Run 'razify guard install --agent'")
	}

	// 3. Agent Policy Check
	policyActive, _ := mcp.CheckAgentPolicy(repoRoot)
	if policyActive {
		success.Println("  ✔ Agent Policy: AGENTS.md policy is active (AI models guided to MCP)")
	} else {
		warning.Println("  ⚠ Agent Policy: AGENTS.md policy not configured")
		fmt.Println("    Remediation: Run 'razify guard install --agent'")
	}

	// 4. MCP Server Info
	fmt.Println()
	bold.Println("  Agent Model Context Protocol (MCP) Configuration:")
	fmt.Println("  Configure your AI coding assistant to use 'razify mcp' for safe schema access.")
	fmt.Println()
	color.Cyan("  Cursor (.cursor/mcp.json):")
	fmt.Println(mcp.SampleMCPConfigCursor())
	fmt.Println()
	color.Cyan("  Claude Desktop (claude_desktop_config.json):")
	fmt.Println(mcp.SampleMCPConfigClaudeDesktop())
	fmt.Println()
	color.Cyan("  Antigravity / Gemini CLI (mcp_config.json):")
	fmt.Println(mcp.SampleMCPConfigAntigravity())
	fmt.Println()
}
