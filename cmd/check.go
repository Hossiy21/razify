package cmd

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/fatih/color"
	"github.com/spf13/cobra"
)

type CheckOutput struct {
	EnvFile            string             `json:"env_file"`
	ExampleFile        string             `json:"example_file"`
	Status             string             `json:"status"` // "PASSED" or "FAILED"
	ValidationResults  []ValidationResult `json:"validation_results"`
	ValidationSummary  ValidateSummary    `json:"validation_summary"`
	SecretScanResults  []ScanResult       `json:"secret_scan_results"`
	SecretScanSummary  ScanSummary        `json:"secret_scan_summary"`
	RemediationSteps   []string           `json:"remediation_steps"`
}

var checkCmd = &cobra.Command{
	Use:   "check [env-file] [example-file]",
	Short: "Run unified validation, type schema checks, and secret scanning in one command",
	Long: `Check is the flagship Razify command. It runs validation against a template,
type checking, and secret leak scanning in a single sub-10ms pass.
If no files are specified, it auto-detects .env and .env.example in the current directory.`,
	Args: cobra.MaximumNArgs(2),
	Run: func(cmd *cobra.Command, args []string) {
		jsonOutput, _ := cmd.Flags().GetBool("json")
		quietOutput, _ := cmd.Flags().GetBool("quiet")

		envFile := ".env"
		exampleFile := ".env.example"

		if len(args) >= 1 {
			envFile = args[0]
		} else {
			// Auto-detect env file fallbacks
			if !fileExists(envFile) {
				if fileExists(".env.local") {
					envFile = ".env.local"
				} else if fileExists(".env.development") {
					envFile = ".env.development"
				}
			}
		}

		if len(args) == 2 {
			exampleFile = args[1]
		} else {
			// Auto-detect example file fallbacks
			if !fileExists(exampleFile) {
				if fileExists(".env.template") {
					exampleFile = ".env.template"
				} else if fileExists(".env.dist") {
					exampleFile = ".env.dist"
				}
			}
		}

		// Verify files exist before running
		if !fileExists(envFile) {
			if !quietOutput {
				fmt.Printf("✘ Error: Environment file '%s' not found.\n", envFile)
				fmt.Println("👉 Tip: Create a .env file or run 'razify check <env-file> <example-file>' to specify custom paths.")
			}
			os.Exit(1)
		}

		if !fileExists(exampleFile) {
			if !quietOutput {
				fmt.Printf("✘ Error: Template file '%s' not found.\n", exampleFile)
				fmt.Println("👉 Tip: Run 'razify init' to generate a .env.example file.")
			}
			os.Exit(1)
		}

		output, err := RunCheck(envFile, exampleFile)
		if err != nil {
			if !quietOutput {
				fmt.Printf("✘ Error executing check: %v\n", err)
			}
			os.Exit(1)
		}

		// JSON Mode
		if jsonOutput {
			data, _ := json.MarshalIndent(output, "", "  ")
			fmt.Println(string(data))
			if output.Status == "FAILED" {
				os.Exit(1)
			}
			return
		}

		// Quiet Mode
		if quietOutput {
			if output.Status == "FAILED" {
				os.Exit(1)
			}
			return
		}

		// Terminal UI Mode
		PrintBanner()

		bold := color.New(color.Bold)
		green := color.New(color.FgGreen, color.Bold)
		red := color.New(color.FgRed, color.Bold)
		yellow := color.New(color.FgYellow, color.Bold)
		cyan := color.New(color.FgCyan)

		bold.Printf("\n⚡ Razify Integrity Check (%s ↔ %s)\n", envFile, exampleFile)
		fmt.Println("────────────────────────────────────────────────────────────")

		// Section 1: Validation Results
		if output.ValidationSummary.Missing > 0 {
			red.Printf("  ✘ Schema Validation   (%d missing key(s))\n", output.ValidationSummary.Missing)
		} else if output.ValidationSummary.Empty > 0 || output.ValidationSummary.Placeholder > 0 {
			yellow.Printf("  ⚠ Schema Validation   (%d empty/placeholder value(s))\n", output.ValidationSummary.Empty+output.ValidationSummary.Placeholder)
		} else {
			green.Printf("  ✔ Schema Validation   (%d key(s) verified & compliant)\n", output.ValidationSummary.Passed)
		}

		// Section 2: Secret Scan Results
		if output.SecretScanSummary.Critical > 0 || output.SecretScanSummary.High > 0 {
			red.Printf("  ✘ Secret Leak Scan    (%d critical/high issue(s) detected)\n", output.SecretScanSummary.Critical+output.SecretScanSummary.High)
		} else if output.SecretScanSummary.Medium > 0 {
			yellow.Printf("  ⚠ Secret Leak Scan    (%d warning(s) detected)\n", output.SecretScanSummary.Medium)
		} else {
			green.Printf("  ✔ Secret Leak Scan    (0 secret leaks detected)\n")
		}

		// Details if errors exist
		if len(output.ValidationResults) > 0 || len(output.SecretScanResults) > 0 {
			fmt.Println("\n📋 Findings & Diagnostics:")

			for _, v := range output.ValidationResults {
				if v.Status == "MISSING" || v.Status == "INVALID" {
					red.Printf("  ✘ [VALIDATION] Key: %s — %s\n", v.Key, v.Message)
				} else if v.Status == "EMPTY" || v.Status == "PLACEHOLDER" {
					yellow.Printf("  ⚠ [VALIDATION] Key: %s — %s\n", v.Key, v.Message)
				}
			}

			for _, s := range output.SecretScanResults {
				if s.Risk == "CRITICAL" {
					red.Printf("  ✘ [SECRET] Line %d (%s): %s (%s)\n", s.Line, s.Key, s.Reason, s.Value)
				} else if s.Risk == "HIGH" {
					red.Printf("  ✘ [SECRET] Line %d (%s): %s (%s)\n", s.Line, s.Key, s.Reason, s.Value)
				} else {
					cyan.Printf("  ~ [SECRET] Line %d (%s): %s (%s)\n", s.Line, s.Key, s.Reason, s.Value)
				}
			}
		}

		// Remediation steps
		if len(output.RemediationSteps) > 0 {
			fmt.Println("\n💡 Actionable Remediation:")
			for _, step := range output.RemediationSteps {
				bold.Printf("  %s\n", step)
			}
		}

		fmt.Println("────────────────────────────────────────────────────────────")

		if output.Status == "FAILED" {
			red.Println("  ✘ FAILED: Resolve required issues before committing or deploying.")
			os.Exit(1)
		} else {
			green.Println("  ✔ PASSED: Environment configuration is secure and complete!")
		}
	},
}

func RunCheck(envFile, exampleFile string) (CheckOutput, error) {
	missing, placeholder, empty, passed, valResults, err := RunValidate(envFile, exampleFile)
	if err != nil {
		return CheckOutput{}, err
	}

	scanResults, err := RunScan(envFile)
	if err != nil {
		return CheckOutput{}, err
	}

	criticalCount := 0
	highCount := 0
	mediumCount := 0
	for _, s := range scanResults {
		switch s.Risk {
		case "CRITICAL":
			criticalCount++
		case "HIGH":
			highCount++
		case "MEDIUM":
			mediumCount++
		}
	}

	status := "PASSED"
	if missing > 0 || criticalCount > 0 || highCount > 0 {
		status = "FAILED"
	}

	var remediation []string
	if missing > 0 {
		remediation = append(remediation, fmt.Sprintf("👉 Run 'razify fix %s %s' to sync missing keys into %s.", envFile, exampleFile, envFile))
	}
	if criticalCount > 0 || highCount > 0 {
		remediation = append(remediation, fmt.Sprintf("👉 Remove or obfuscate exposed API keys in %s before committing.", envFile))
	}
	if placeholder > 0 || empty > 0 {
		remediation = append(remediation, fmt.Sprintf("👉 Replace placeholder values in %s with real development settings.", envFile))
	}

	if valResults == nil {
		valResults = []ValidationResult{}
	}
	if scanResults == nil {
		scanResults = []ScanResult{}
	}

	output := CheckOutput{
		EnvFile:           envFile,
		ExampleFile:       exampleFile,
		Status:            status,
		ValidationResults: valResults,
		ValidationSummary: ValidateSummary{
			Passed:      passed,
			Missing:     missing,
			Empty:       empty,
			Placeholder: placeholder,
		},
		SecretScanResults: scanResults,
		SecretScanSummary: ScanSummary{
			Critical: criticalCount,
			High:     highCount,
			Medium:   mediumCount,
			Total:    len(scanResults),
		},
		RemediationSteps: remediation,
	}

	return output, nil
}

func init() {
	rootCmd.AddCommand(checkCmd)
	checkCmd.Flags().Bool("json", false, "Output results as JSON")
	checkCmd.Flags().Bool("quiet", false, "Suppress output and exit with status code 0 (pass) or 1 (fail)")
}
