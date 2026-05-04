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

		diffs, results, err := RunDiff(file1, file2)
		if err != nil {
			fmt.Printf("Error comparing files: %v\n", err)
			os.Exit(1)
		}

		// Colors
		missing := color.New(color.FgRed, color.Bold)
		added := color.New(color.FgGreen, color.Bold)
		changed := color.New(color.FgYellow, color.Bold)
		bold := color.New(color.Bold)
		success := color.New(color.FgGreen)

		bold.Printf("\nComparing %s → %s\n\n", file1, file2)

		for _, r := range results {
			switch r.Type {
			case "MISSING":
				missing.Printf("  ✘  MISSING in %s: %s\n", file2, r.Key)
			case "ADDED":
				added.Printf("  ✔  ADDED in %s:   %s\n", file2, r.Key)
			case "CHANGED":
				changed.Printf("  ~  CHANGED: %s\n", r.Key)
				fmt.Printf("      %s: %s\n", file1, r.OldValue)
				fmt.Printf("      %s: %s\n", file2, r.NewValue)
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

type DiffResult struct {
	Key      string
	Type     string
	OldValue string
	NewValue string
}

func RunDiff(file1, file2 string) (int, []DiffResult, error) {
	env1, err := parseEnvFile(file1)
	if err != nil {
		return 0, nil, err
	}

	env2, err := parseEnvFile(file2)
	if err != nil {
		return 0, nil, err
	}

	var results []DiffResult
	diffs := 0

	for key := range env1 {
		if _, exists := env2[key]; !exists {
			results = append(results, DiffResult{Key: key, Type: "MISSING"})
			diffs++
		}
	}

	for key := range env2 {
		if _, exists := env1[key]; !exists {
			results = append(results, DiffResult{Key: key, Type: "ADDED"})
			diffs++
		}
	}

	for key := range env1 {
		if val2, exists := env2[key]; exists {
			if env1[key] != val2 {
				results = append(results, DiffResult{
					Key:      key,
					Type:     "CHANGED",
					OldValue: env1[key],
					NewValue: val2,
				})
				diffs++
			}
		}
	}

	return diffs, results, nil
}

func init() {
	rootCmd.AddCommand(diffCmd)
}
