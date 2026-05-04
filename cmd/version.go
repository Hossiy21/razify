package cmd

import (
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/fatih/color"
	"github.com/spf13/cobra"
)

var CurrentVersion = "v0.1.0"

var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "Print the version number of Razify",
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Printf("Razify %s\n", CurrentVersion)
		CheckForUpdates(true)
	},
}

func CheckForUpdates(verbose bool) {
	client := http.Client{
		Timeout: 2 * time.Second,
	}

	resp, err := client.Get("https://api.github.com/repos/Hossiy21/razify/releases/latest")
	if err != nil {
		if verbose {
			fmt.Println("Could not check for updates.")
		}
		return
	}
	defer resp.Body.Close()

	var release struct {
		TagName string `json:"tag_name"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&release); err != nil {
		return
	}

	if release.TagName != "" && release.TagName != CurrentVersion {
		yellow := color.New(color.FgYellow, color.Bold)
		fmt.Println()
		yellow.Printf("✨ A new version of Razify is available: %s (Current: %s)\n", release.TagName, CurrentVersion)
		fmt.Println("   Update now: go install github.com/Hossiy21/razify@latest")
		fmt.Println()
	} else if verbose {
		fmt.Println("You are running the latest version.")
	}
}

func init() {
	rootCmd.AddCommand(versionCmd)
}
