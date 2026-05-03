package cmd

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"github.com/fatih/color"
	"github.com/spf13/cobra"
)

type ValidationResult struct {
	Key     string `json:"key"`
	Status  string `json:"status"`
	Message string `json:"message"`
}

type ValidateOutput struct {
	EnvFile     string             `json:"env_file"`
	ExampleFile string             `json:"example_file"`
	Results     []ValidationResult `json:"results"`
	Summary     ValidateSummary    `json:"summary"`
}

type ValidateSummary struct {
	Passed      int `json:"passed"`
	Missing     int `json:"missing"`
	Empty       int `json:"empty"`
	Placeholder int `json:"placeholder"`
}

var validateCmd = &cobra.Command{
	Use:   "validate [env-file] [example-file]",
	Short: "Validate your .env file against a .env.example template",
	Args:  cobra.ExactArgs(2),
	Run: func(cmd *cobra.Command, args []string) {
		envFile := args[0]
		exampleFile := args[1]
		jsonOutput, _ := cmd.Flags().GetBool("json")

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

		var results []ValidationResult
		missing := 0
		empty := 0
		placeholder := 0
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
				placeholder++
				continue
			}

			results = append(results, ValidationResult{
				Key:    key,
				Status: "OK",
			})
			passed++
		}

		// JSON output
		if jsonOutput {
			if results == nil {
				results = []ValidationResult{}
			}
			out := ValidateOutput{
				EnvFile:     envFile,
				ExampleFile: exampleFile,
				Results:     results,
				Summary: ValidateSummary{
					Passed:      passed,
					Missing:     missing,
					Empty:       empty,
					Placeholder: placeholder,
				},
			}
			data, _ := json.MarshalIndent(out, "", "  ")
			fmt.Println(string(data))
			if missing > 0 {
				os.Exit(1)
			}
			return
		}

		// Colors
		missingColor := color.New(color.FgRed, color.Bold)
		emptyColor := color.New(color.FgYellow, color.Bold)
		placeholderColor := color.New(color.FgCyan)
		okColor := color.New(color.FgGreen)
		bold := color.New(color.Bold)

		bold.Printf("\nValidating %s against %s...\n\n", envFile, exampleFile)

		for _, r := range results {
			switch r.Status {
			case "MISSING":
				missingColor.Printf("  ✘  [MISSING]     %s\n", r.Key)
				fmt.Printf("      %s\n\n", r.Message)
			case "EMPTY":
				emptyColor.Printf("  ⚠  [EMPTY]       %s\n", r.Key)
				fmt.Printf("      %s\n\n", r.Message)
			case "PLACEHOLDER":
				placeholderColor.Printf("  ~  [PLACEHOLDER] %s\n", r.Key)
				fmt.Printf("      %s\n\n", r.Message)
			case "OK":
				okColor.Printf("  ✔  [OK]          %s\n", r.Key)
			}
		}

		fmt.Println()
		bold.Printf("Summary: ")
		okColor.Printf("%d OK  ", passed)
		missingColor.Printf("%d MISSING  ", missing)
		emptyColor.Printf("%d EMPTY/PLACEHOLDER\n\n", empty+placeholder)

		if missing > 0 {
			missingColor.Println("  ✘  ACTION REQUIRED: Add missing keys before deploying!")
			os.Exit(1)
		} else if empty > 0 || placeholder > 0 {
			emptyColor.Println("  ⚠  WARNING: Some keys need real values.")
		} else {
			okColor.Println("  ✔  All required keys are present and set!")
		}
	},
}

func parseEnvFileValidate(filename string) (map[string]string, error) {
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
	rootCmd.AddCommand(validateCmd)
	validateCmd.Flags().Bool("json", false, "Output results as JSON")
}
