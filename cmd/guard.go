package cmd

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/fatih/color"
	"github.com/spf13/cobra"
)

var hookScript = `#!/bin/sh
# Envy Guard — installed by envy guard install
# Scans staged .env files before every commit

# Find envy binary
if ! command -v envy &> /dev/null; then
    echo "envy guard: envy not found in PATH, skipping scan"
    exit 0
fi

# Get all staged files
STAGED=$(git diff --cached --name-only)

FAILED=0

for FILE in $STAGED; do
    # Only scan .env files
    if echo "$FILE" | grep -qE '(^|/)\.env(\.|$)'; then
        echo ""
        echo "envy guard: scanning $FILE..."
        envy scan "$FILE"
        if [ $? -ne 0 ]; then
            FAILED=1
        fi
    fi
done

if [ $FAILED -ne 0 ]; then
    echo ""
    echo "envy guard: commit blocked — fix secrets before committing."
    exit 1
fi

exit 0
`

var guardCmd = &cobra.Command{
	Use:   "guard [install|uninstall|status]",
	Short: "Protect your git repo by scanning .env files before every commit",
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		action := args[0]

		switch action {
		case "install":
			installGuard()
		case "uninstall":
			uninstallGuard()
		case "status":
			statusGuard()
		default:
			color.Red("  Unknown action: %s\n", action)
			fmt.Println("  Usage: envy guard [install|uninstall|status]")
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

	bold.Println("\nInstalling Envy Guard...")

	hookPath, err := getHookPath()
	if err != nil {
		errColor.Printf("  ✘  %v\n", err)
		os.Exit(1)
	}

	// Check if hook already exists
	if _, err := os.Stat(hookPath); err == nil {
		// Read existing hook
		existing, _ := os.ReadFile(hookPath)
		if strings.Contains(string(existing), "envy guard") {
			color.Yellow("  ⚠  Envy Guard is already installed.\n")
			return
		}
		// Append to existing hook
		f, err := os.OpenFile(hookPath, os.O_APPEND|os.O_WRONLY, 0755)
		if err != nil {
			errColor.Printf("  ✘  Could not modify existing hook: %v\n", err)
			os.Exit(1)
		}
		defer f.Close()
		f.WriteString("\n" + hookScript)
	} else {
		// Create new hook
		err = os.WriteFile(hookPath, []byte(hookScript), 0755)
		if err != nil {
			errColor.Printf("  ✘  Could not install hook: %v\n", err)
			os.Exit(1)
		}
	}

	success.Println("  ✔  Envy Guard installed successfully!")
	fmt.Printf("     Hook location: %s\n", hookPath)
	fmt.Println()
	fmt.Println("  Every git commit in this repo will now be scanned.")
	fmt.Println("  Commits with exposed secrets will be blocked automatically.")
	fmt.Println()
	color.Cyan("  To uninstall: envy guard uninstall\n")
}

func uninstallGuard() {
	bold := color.New(color.Bold)
	success := color.New(color.FgGreen, color.Bold)
	errColor := color.New(color.FgRed, color.Bold)

	bold.Println("\nUninstalling Envy Guard...")

	hookPath, err := getHookPath()
	if err != nil {
		errColor.Printf("  ✘  %v\n", err)
		os.Exit(1)
	}

	if _, err := os.Stat(hookPath); os.IsNotExist(err) {
		color.Yellow("  ⚠  No hook found. Envy Guard was not installed.\n")
		return
	}

	existing, _ := os.ReadFile(hookPath)
	if !strings.Contains(string(existing), "envy guard") {
		color.Yellow("  ⚠  Envy Guard hook not found in pre-commit.\n")
		return
	}

	// Remove the hook file entirely
	err = os.Remove(hookPath)
	if err != nil {
		errColor.Printf("  ✘  Could not remove hook: %v\n", err)
		os.Exit(1)
	}

	success.Println("  ✔  Envy Guard uninstalled successfully.")
	fmt.Println("     Your commits are no longer being scanned.")
}

func statusGuard() {
	bold := color.New(color.Bold)
	success := color.New(color.FgGreen, color.Bold)

	bold.Println("\nEnvy Guard Status")
	fmt.Println("  ─────────────────────────────")

	hookPath, err := getHookPath()
	if err != nil {
		color.Red("  ✘  Not a git repository\n")
		return
	}

	if _, err := os.Stat(hookPath); os.IsNotExist(err) {
		color.Yellow("  ⚠  Not installed\n")
		fmt.Println("     Run: envy guard install")
		return
	}

	existing, _ := os.ReadFile(hookPath)
	if strings.Contains(string(existing), "envy guard") {
		success.Println("  ✔  Active — commits are being protected")
		fmt.Printf("     Hook: %s\n", hookPath)
	} else {
		color.Yellow("  ⚠  Pre-commit hook exists but Envy Guard is not in it\n")
		fmt.Println("     Run: envy guard install")
	}
	fmt.Println()
}

func init() {
	rootCmd.AddCommand(guardCmd)
}
