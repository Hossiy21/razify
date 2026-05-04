package cmd

import (
	"encoding/json"
	"fmt"
	"net/url"
	"os"
	"strconv"
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

		missing, placeholder, empty, okCount, results, err := RunValidate(envFile, exampleFile)
		if err != nil {
			fmt.Printf("Error validating: %v\n", err)
			os.Exit(1)
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
					Passed:      okCount,
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
			case "INVALID":
				missingColor.Printf("  ✘  [INVALID]     %s\n", r.Key)
				fmt.Printf("      %s\n\n", r.Message)
			}
		}

		fmt.Println()
		bold.Printf("Summary: ")
		okColor.Printf("%d OK  ", okCount)
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

func RunValidate(envFile, exampleFile string) (int, int, int, int, []ValidationResult, error) {
	envVars, err := parseEnvFile(envFile)
	if err != nil {
		return 0, 0, 0, 0, nil, err
	}

	exampleVars, err := ParseEnvWithMetadata(exampleFile)
	if err != nil {
		return 0, 0, 0, 0, nil, err
	}

	var results []ValidationResult
	missing := 0
	empty := 0
	placeholder := 0
	passed := 0

	for _, ev := range exampleVars {
		actualValue, exists := envVars[ev.Key]

		if !exists {
			if ev.Required {
				results = append(results, ValidationResult{
					Key:     ev.Key,
					Status:  "MISSING",
					Message: "Required key not found",
				})
				missing++
			}
			continue
		}

		if actualValue == "" {
			results = append(results, ValidationResult{
				Key:     ev.Key,
				Status:  "EMPTY",
				Message: "Key exists but has no value",
			})
			empty++
			continue
		}

		// Placeholder check
		if ev.Value != "" && actualValue == ev.Value {
			obviousPlaceholders := []string{
				"your_", "change_me", "changeme", "xxx", "example",
				"replace_", "fill_in", "todo", "fixme", "<", ">",
				"placeholder", "your-", "put_your", "insert_",
			}
			isObvious := false
			valueLower := strings.ToLower(actualValue)
			for _, p := range obviousPlaceholders {
				if strings.Contains(valueLower, p) {
					isObvious = true
					break
				}
			}
			if isObvious {
				results = append(results, ValidationResult{
					Key:     ev.Key,
					Status:  "PLACEHOLDER",
					Message: "Value looks like it was never changed from example",
				})
				placeholder++
				continue
			}
		}

		// Advanced validation from tags
		if errStr := validateValue(actualValue, ev.Tags); errStr != "" {
			results = append(results, ValidationResult{
				Key:     ev.Key,
				Status:  "INVALID",
				Message: errStr,
			})
			missing++ // Treat as missing/failed
			continue
		}

		results = append(results, ValidationResult{
			Key:    ev.Key,
			Status: "OK",
		})
		passed++
	}

	return missing, placeholder, empty, passed, results, nil
}

func validateValue(value string, tags map[string]string) string {
	if tags == nil {
		return ""
	}

	// Type validation
	if t, ok := tags["type"]; ok {
		switch t {
		case "int":
			if _, err := strconv.Atoi(value); err != nil {
				return "Value must be an integer"
			}
		case "bool":
			if value != "true" && value != "false" {
				return "Value must be 'true' or 'false'"
			}
		case "url":
			if _, err := url.ParseRequestURI(value); err != nil {
				return "Value must be a valid URL"
			}
		}
	}

	// Range validation (e.g. @range=1-100)
	if r, ok := tags["range"]; ok {
		parts := strings.Split(r, "-")
		if len(parts) == 2 {
			min, _ := strconv.Atoi(parts[0])
			max, _ := strconv.Atoi(parts[1])
			val, err := strconv.Atoi(value)
			if err == nil {
				if val < min || val > max {
					return fmt.Sprintf("Value out of range (%d-%d)", min, max)
				}
			}
		}
	}

	return ""
}

func init() {
	rootCmd.AddCommand(validateCmd)
	validateCmd.Flags().Bool("json", false, "Output results as JSON")
}
