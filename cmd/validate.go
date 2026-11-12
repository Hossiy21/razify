package cmd

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/Hossiy21/razify/internal/reporter"
	"github.com/Hossiy21/razify/internal/scanner"
	"github.com/Hossiy21/razify/internal/schema"
	"github.com/fatih/color"
	"github.com/spf13/cobra"
)

type ValidationResult = schema.ValidationResult
type ValidateSummary = schema.ValidateSummary
type ValidateOutput = reporter.LegacyValidateOutput

var validateCmd = &cobra.Command{
	Use:   "validate [env-file] [example-file]",
	Short: "Validate your .env file against a .env.example template",
	Args:  cobra.ExactArgs(2),
	Run: func(cmd *cobra.Command, args []string) {
		envFile := args[0]
		exampleFile := args[1]
		jsonOutput, _ := cmd.Flags().GetBool("json")
		sarifOutput, _ := cmd.Flags().GetBool("sarif")

		missing, placeholder, empty, okCount, results, err := RunValidate(envFile, exampleFile)
		if err != nil {
			fmt.Printf("Error validating: %v\n", err)
			os.Exit(reporter.ExitError)
		}

		// SARIF output
		if sarifOutput {
			res := reporter.BuildResult(
				envFile,
				exampleFile,
				schema.ValidateSummary{
					Passed:      okCount,
					Missing:     missing,
					Empty:       empty,
					Placeholder: placeholder,
				},
				results,
				scanner.ScanSummary{},
				nil,
				nil,
			)
			sarifText, err := reporter.RenderSarif(res, CurrentVersion)
			if err != nil {
				fmt.Printf("Error generating SARIF: %v\n", err)
				os.Exit(reporter.ExitError)
			}
			fmt.Println(sarifText)
			if missing > 0 {
				os.Exit(getViolationExitCode())
			}
			return
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
				os.Exit(getViolationExitCode())
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
			os.Exit(getViolationExitCode())
		} else if empty > 0 || placeholder > 0 {
			emptyColor.Println("  ⚠  WARNING: Some keys need real values.")
		} else {
			okColor.Println("  ✔  All required keys are present and set!")
		}
	},
}

func RunValidate(envFile, exampleFile string) (int, int, int, int, []ValidationResult, error) {
	return schema.RunValidate(envFile, exampleFile)
}

func validateValue(value string, tags map[string]string, envVars map[string]string) string {
	return schema.ValidateValue(value, tags, envVars)
}

func init() {
	rootCmd.AddCommand(validateCmd)
	validateCmd.Flags().Bool("json", false, "Output results as JSON")
	validateCmd.Flags().Bool("sarif", false, "Output results as SARIF v2.1.0")
}
