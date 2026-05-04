package cmd

import (
	"fmt"
	"os"

	"github.com/fatih/color"
	"github.com/spf13/cobra"
)

var auditCmd = &cobra.Command{
	Use:   "audit [env-file] [example-file]",
	Short: "Full health report of your environment configuration",
	Args:  cobra.ExactArgs(2),
	Run: func(cmd *cobra.Command, args []string) {
		envFile := args[0]
		exampleFile := args[1]

		// Colors
		bold := color.New(color.Bold)
		critical := color.New(color.FgRed, color.Bold)
		high := color.New(color.FgYellow, color.Bold)
		medium := color.New(color.FgCyan)
		success := color.New(color.FgGreen, color.Bold)

		bold.Println("\n  ┌─────────────────────────────┐")
		bold.Println("  │     Razify Audit Report     │")
		bold.Println("  └─────────────────────────────┘")
		fmt.Println()

		report, err := RunAudit(envFile, exampleFile)
		if err != nil {
			fmt.Printf("Error running audit: %v\n", err)
			os.Exit(1)
		}

		// ── REPORT ────────────────────────────────────────
		fmt.Println()
		bold.Println("  ┌─────────────────────────────┐")
		bold.Println("  │          Results            │")
		bold.Println("  └─────────────────────────────┘")
		fmt.Println()

		// Scan results
		bold.Print("  Scan        ")
		if report.ScanCritical > 0 {
			critical.Printf("%d CRITICAL  ", report.ScanCritical)
		}
		if report.ScanHigh > 0 {
			high.Printf("%d HIGH  ", report.ScanHigh)
		}
		if report.ScanMedium > 0 {
			medium.Printf("%d MEDIUM  ", report.ScanMedium)
		}
		if report.ScanCritical == 0 && report.ScanHigh == 0 && report.ScanMedium == 0 {
			success.Print("All clear")
		}
		fmt.Println()

		// Validate results
		bold.Print("  Validate    ")
		if report.Missing > 0 {
			critical.Printf("%d MISSING  ", report.Missing)
		}
		if report.Placeholder > 0 {
			medium.Printf("%d PLACEHOLDER  ", report.Placeholder)
		}
		if report.Empty > 0 {
			high.Printf("%d EMPTY  ", report.Empty)
		}
		if report.Passed > 0 {
			success.Printf("%d OK  ", report.Passed)
		}
		fmt.Println()

		// Diff results
		bold.Print("  Diff        ")
		if report.DiffCount > 0 {
			high.Printf("%d difference(s) from %s", report.DiffCount, exampleFile)
		} else {
			success.Print("No differences")
		}
		fmt.Println()

		// Health score
		fmt.Println()
		bold.Println("  ┌─────────────────────────────┐")
		bold.Println("  │        Health Score         │")
		bold.Println("  └─────────────────────────────┘")
		fmt.Println()

		scoreColor := success
		scoreLabel := "Excellent"
		if report.Score < 40 {
			scoreColor = critical
			scoreLabel = "Critical — needs immediate attention"
		} else if report.Score < 60 {
			scoreColor = high
			scoreLabel = "Poor — several issues found"
		} else if report.Score < 80 {
			scoreColor = medium
			scoreLabel = "Fair — some improvements needed"
		}

		scoreColor.Printf("  %d/100  %s\n", report.Score, scoreLabel)
		fmt.Println()

		// Recommendations
		if report.ScanCritical > 0 || report.Missing > 0 {
			bold.Println("  Recommendations:")
			if report.ScanCritical > 0 {
				critical.Println("  ✘  Rotate exposed credentials immediately")
			}
			if report.Missing > 0 {
				high.Println("  ⚠  Add missing required variables before deploying")
			}
			if report.Placeholder > 0 {
				medium.Println("  ~  Replace placeholder values with real ones")
			}
			fmt.Println()
		} else {
			success.Println("  ✔  Your environment looks healthy!")
			fmt.Println()
		}

		// Silently check for updates
		CheckForUpdates(false)
	},
}

type AuditReport struct {
	Score        int
	ScanCritical int
	ScanHigh     int
	ScanMedium   int
	Missing      int
	Placeholder  int
	Empty        int
	Passed       int
	DiffCount    int
}

func RunAudit(envFile, exampleFile string) (AuditReport, error) {
	report := AuditReport{}

	// 1. Run Scan
	scanResults, err := RunScan(envFile)
	if err != nil {
		return report, err
	}
	for _, r := range scanResults {
		switch r.Risk {
		case "CRITICAL":
			report.ScanCritical++
		case "HIGH":
			report.ScanHigh++
		case "MEDIUM":
			report.ScanMedium++
		}
	}

	// 2. Run Validate
	missing, placeholder, empty, passed, _, err := RunValidate(envFile, exampleFile)
	if err != nil {
		return report, err
	}
	report.Missing = missing
	report.Placeholder = placeholder
	report.Empty = empty
	report.Passed = passed

	// 3. Diff Logic (Legacy but kept for now)
	envVars, _ := parseEnvFile(envFile)
	exampleVars, _ := parseEnvFile(exampleFile)
	for key := range envVars {
		if _, exists := exampleVars[key]; !exists {
			report.DiffCount++
		}
	}
	for key := range exampleVars {
		if _, exists := envVars[key]; !exists {
			report.DiffCount++
		}
	}
	for key, val := range envVars {
		if val2, exists := exampleVars[key]; exists {
			if val != val2 {
				report.DiffCount++
			}
		}
	}

	// 4. Scoring
	// 4. Scoring
	report.Score = 100
	report.Score -= report.ScanCritical * 30
	report.Score -= report.ScanHigh * 5
	report.Score -= report.ScanMedium * 2
	report.Score -= report.Missing * 20
	report.Score -= report.Placeholder * 3
	report.Score -= report.Empty * 3
	if report.Score < 0 {
		report.Score = 0
	}

	return report, nil
}

func init() {
	rootCmd.AddCommand(auditCmd)
}
