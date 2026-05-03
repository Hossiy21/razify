package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

type ValidationResult struct {
	Key     string
	Status  string
	Message string
}

var validateCmd = &cobra.Command{
	Use:   "validate [env-file] [example-file]",
	Short: "Validate your .env file against a .env.example template",
	Args:  cobra.ExactArgs(2),
	Run: func(cmd *cobra.Command, args []string) {
		envFile := args[0]
		exampleFile := args[1]

		envVars, err := parseEnvFile(envFile)
		if err != nil {
			fmt.Printf("Error reading %s: %v\n", envFile, err)
			os.Exit(1)
		}

		exampleVars, err := parseEnvFile(exampleFile)
		if err != nil {
			fmt.Printf("Error reading %s: %v\n", exampleFile, err)
			os.Exit(1)
		}

		fmt.Printf("\nValidating %s against %s...\n\n", envFile, exampleFile)

		var results []ValidationResult
		missing := 0
		empty := 0
		passed := 0

		for key, exampleValue := range exampleVars {
			actualValue, exists := envVars[key]

			if !exists {
				results = append(results, ValidationResult{
					Key:     key,
					Status:  "MISSING",
					Message: "Required key not found in " + envFile,
				})
				missing++
				continue
			}

			if actualValue == "" {
				results = append(results, ValidationResult{
					Key:     key,
					Status:  "EMPTY",
					Message: "Key exists but has no value",
				})
				empty++
				continue
			}

			if exampleValue != "" && actualValue == exampleValue {
				results = append(results, ValidationResult{
					Key:     key,
					Status:  "PLACEHOLDER",
					Message: "Value looks like it was never changed from example",
				})
				empty++
				continue
			}

			results = append(results, ValidationResult{
				Key:    key,
				Status: "OK",
			})
			passed++
		}

		for _, r := range results {
			switch r.Status {
			case "MISSING":
				fmt.Printf("  !! [MISSING]     %s\n", r.Key)
				fmt.Printf("      %s\n\n", r.Message)
			case "EMPTY":
				fmt.Printf("  !  [EMPTY]       %s\n", r.Key)
				fmt.Printf("      %s\n\n", r.Message)
			case "PLACEHOLDER":
				fmt.Printf("  ~  [PLACEHOLDER] %s\n", r.Key)
				fmt.Printf("      %s\n\n", r.Message)
			case "OK":
				fmt.Printf("  ok [OK]          %s\n", r.Key)
			}
		}

		fmt.Printf("\nSummary: %d OK   %d MISSING   %d EMPTY/PLACEHOLDER\n",
			passed, missing, empty)

		if missing > 0 {
			fmt.Println("\n  ACTION REQUIRED: Add missing keys before deploying!")
			os.Exit(1)
		} else if empty > 0 {
			fmt.Println("\n  WARNING: Some keys need real values.")
		} else {
			fmt.Println("\n  All required keys are present and set!")
		}
	},
}

func init() {
	rootCmd.AddCommand(validateCmd)
}
