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

type ScanResult = scanner.ScanResult
type ScanOutput = reporter.LegacyScanOutput
type ScanSummary = scanner.ScanSummary

var scanCmd = &cobra.Command{
	Use:   "scan [file]",
	Short: "Scan an .env file for secret leaks and weak values",
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		filename := args[0]
		jsonOutput, _ := cmd.Flags().GetBool("json")
		sarifOutput, _ := cmd.Flags().GetBool("sarif")

		results, err := RunScan(filename)
		if err != nil {
			fmt.Printf("Error scanning %s: %v\n", filename, err)
			os.Exit(reporter.ExitError)
		}

		criticalCount := 0
		highCount := 0
		mediumCount := 0

		for _, r := range results {
			switch r.Risk {
			case "CRITICAL":
				criticalCount++
			case "HIGH":
				highCount++
			case "MEDIUM":
				mediumCount++
			}
		}

		// SARIF output
		if sarifOutput {
			res := reporter.BuildResult(
				filename,
				"",
				schema.ValidateSummary{},
				nil,
				scanner.ScanSummary{
					Critical: criticalCount,
					High:     highCount,
					Medium:   mediumCount,
					Total:    len(results),
				},
				results,
				nil,
			)
			sarifText, err := reporter.RenderSarif(res, CurrentVersion)
			if err != nil {
				fmt.Printf("Error generating SARIF: %v\n", err)
				os.Exit(reporter.ExitError)
			}
			fmt.Println(sarifText)
			if criticalCount > 0 || highCount > 0 {
				os.Exit(getViolationExitCode())
			}
			return
		}

		// JSON output
		if jsonOutput {
			if results == nil {
				results = []ScanResult{}
			}
			out := ScanOutput{
				File:    filename,
				Results: results,
				Summary: ScanSummary{
					Critical: criticalCount,
					High:     highCount,
					Medium:   mediumCount,
					Total:    len(results),
				},
			}
			data, _ := json.MarshalIndent(out, "", "  ")
			fmt.Println(string(data))
			if criticalCount > 0 || highCount > 0 {
				os.Exit(getViolationExitCode())
			}
			return
		}

		// Colors
		critical := color.New(color.FgRed, color.Bold)
		high := color.New(color.FgYellow, color.Bold)
		medium := color.New(color.FgCyan)
		success := color.New(color.FgGreen, color.Bold)
		bold := color.New(color.Bold)

		bold.Printf("\nScanning %s...\n\n", filename)

		if len(results) == 0 {
			success.Println("  ✔  All clear! No secrets or weak values found.")
			return
		}

		for _, r := range results {
			switch r.Risk {
			case "CRITICAL":
				critical.Printf("  ✘  [CRITICAL] Line %d: %s\n", r.Line, r.Key)
			case "HIGH":
				high.Printf("  ⚠  [HIGH]     Line %d: %s\n", r.Line, r.Key)
			case "MEDIUM":
				medium.Printf("  ~  [MEDIUM]   Line %d: %s\n", r.Line, r.Key)
			}
			fmt.Printf("     Value : %s\n", r.Value)
			fmt.Printf("     Reason: %s\n\n", r.Reason)
		}

		bold.Printf("Summary: ")
		critical.Printf("%d CRITICAL  ", criticalCount)
		high.Printf("%d HIGH  ", highCount)
		medium.Printf("%d MEDIUM\n\n", mediumCount)

		if criticalCount > 0 || highCount > 0 {
			critical.Println("  ✘  ACTION REQUIRED: Never commit this file to git!")
			os.Exit(getViolationExitCode())
		}
	},
}

func RunScan(filename string) ([]ScanResult, error) {
	return scanner.RunScan(filename)
}

func ShannonEntropy(data string) float64 {
	return scanner.ShannonEntropy(data)
}

func maskValue(value string) string {
	return scanner.MaskValue(value)
}

func init() {
	rootCmd.AddCommand(scanCmd)
	scanCmd.Flags().Bool("json", false, "Output results as JSON")
	scanCmd.Flags().Bool("sarif", false, "Output results as SARIF v2.1.0")
}
