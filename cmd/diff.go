package cmd

import (
	"bufio"
	"fmt"
	"os"
	"strings"

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

		fmt.Printf("\nComparing %s → %s\n\n", file1, file2)

		// Keys in file1 but missing in file2
		for key := range env1 {
			if _, exists := env2[key]; !exists {
				fmt.Printf("  MISSING in %s: %s\n", file2, key)
			}
		}

		// Keys in file2 but missing in file1
		for key := range env2 {
			if _, exists := env1[key]; !exists {
				fmt.Printf("  ADDED in %s:   %s\n", file2, key)
			}
		}

		// Keys in both but different values
		for key := range env1 {
			if val2, exists := env2[key]; exists {
				if env1[key] != val2 {
					fmt.Printf("  CHANGED: %s\n    %s: %s\n    %s: %s\n", key, file1, env1[key], file2, val2)
				}
			}
		}

		fmt.Println("\nDone.")
	},
}

func parseEnvFile(filename string) (map[string]string, error) {
	file, err := os.Open(filename)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	result := map[string]string{}
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		parts := strings.SplitN(line, "=", 2)
		if len(parts) == 2 {
			result[strings.TrimSpace(parts[0])] = strings.TrimSpace(parts[1])
		}
	}
	return result, scanner.Err()
}

func init() {
	rootCmd.AddCommand(diffCmd)
}
