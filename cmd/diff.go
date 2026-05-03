package cmd

import (
	"fmt"
	"os"

	"github.com/fatih/color"
	"github.com/spf13/cobra"
)

var diffCmd = &cobra.Command{
	Use:   "diff [file1] [file2]",
	Short: "Compare two .env files and show differences",
	Args:  cobra.ExactArgs(2),
	Run: func(cmd *cobra.Command, args []string) {
		file1 := args[0]
		file2 := args[1]

		env1, err := parseEnvFile(file1)
		if err != nil {
			fmt.Printf("Error reading %s: %v\n", file1, err)
			os.Exit(1)
		}

		env2, err := parseEnvFile(file2)
		if err != nil {
			fmt.Printf("Error reading %s: %v\n", file2, err)
			os.Exit(1)
		}

		// Colors
		missing := color.New(color.FgRed, color.Bold)
		added := color.New(color.FgGreen, color.Bold)
		changed := color.New(color.FgYellow, color.Bold)
		bold := color.New(color.Bold)
		success := color.New(color.FgGreen)

		bold.Printf("\nComparing %s → %s\n\n", file1, file2)

		diffs := 0

		for key := range env1 {
			if _, exists := env2[key]; !exists {
				missing.Printf("  ✘  MISSING in %s: %s\n", file2, key)
				diffs++
			}
		}

		for key := range env2 {
			if _, exists := env1[key]; !exists {
				added.Printf("  ✔  ADDED in %s:   %s\n", file2, key)
				diffs++
			}
		}

		for key := range env1 {
			if val2, exists := env2[key]; exists {
				if env1[key] != val2 {
					changed.Printf("  ~  CHANGED: %s\n", key)
					fmt.Printf("      %s: %s\n", file1, env1[key])
					fmt.Printf("      %s: %s\n", file2, val2)
					diffs++
				}
			}
		}

		fmt.Println()
		if diffs == 0 {
			success.Println("  ✔  Files are identical.")
		} else {
			bold.Printf("  %d difference(s) found.\n", diffs)
		}
		fmt.Println()
	},
}

func init() {
	rootCmd.AddCommand(diffCmd)
}
