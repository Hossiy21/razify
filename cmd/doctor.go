package cmd

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/Hossiy21/razify/internal/doctor"
	"github.com/Hossiy21/razify/internal/reporter"
	"github.com/fatih/color"
	"github.com/spf13/cobra"
)

var doctorCmd = &cobra.Command{
	Use:   "doctor [env-file] [example-file]",
	Short: "Diagnose workspace health, toolchains, port conflicts, and database connectivity",
	Long: `Doctor performs a comprehensive, offline-first health check of your development
environment, verifying .env shielding in git, required toolchains (Go, Node, Docker),
schema integrity, port collisions, and service availability.

Health score is calculated deterministically on a 0-100 point scale with concrete
remediation commands for every diagnostic failure.`,
	Args: cobra.MaximumNArgs(2),
	Run: func(cmd *cobra.Command, args []string) {
		jsonOutput, _ := cmd.Flags().GetBool("json")
		sarifOutput, _ := cmd.Flags().GetBool("sarif")
		quietOutput, _ := cmd.Flags().GetBool("quiet")
		fastMode, _ := cmd.Flags().GetBool("fast")
		probeRemote, _ := cmd.Flags().GetBool("probe-remote")
		timeoutMs, _ := cmd.Flags().GetInt("timeout")
		minScore, _ := cmd.Flags().GetInt("min-score")

		envFile := ".env"
		exampleFile := ".env.example"

		if len(args) >= 1 {
			envFile = args[0]
		} else {
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
			if !fileExists(exampleFile) {
				if fileExists(".env.template") {
					exampleFile = ".env.template"
				}
			}
		}

		opts := doctor.DoctorOptions{
			EnvFile:       envFile,
			ExampleFile:   exampleFile,
			GitignoreFile: ".gitignore",
			Fast:          fastMode,
			ProbeRemote:   probeRemote,
			Timeout:       time.Duration(timeoutMs) * time.Millisecond,
			MinScore:      minScore,
		}

		deps := doctor.NewDefaultDependencies()
		report := doctor.RunDoctor(opts, deps)

		// SARIF Mode
		if sarifOutput {
			sarifText, err := reporter.RenderSarif(report.Result, CurrentVersion)
			if err != nil {
				if !quietOutput {
					fmt.Printf("✘ Error generating SARIF: %v\n", err)
				}
				os.Exit(reporter.ExitError)
			}
			fmt.Println(sarifText)
			if report.Result.Status == "FAILED" {
				os.Exit(getViolationExitCode())
			}
			return
		}

		// JSON Mode
		if jsonOutput {
			data, err := json.MarshalIndent(report, "", "  ")
			if err != nil {
				os.Exit(reporter.ExitError)
			}
			fmt.Println(string(data))
			if report.Result.Status == "FAILED" {
				os.Exit(getViolationExitCode())
			}
			return
		}

		// Quiet Mode
		if quietOutput {
			if report.Result.Status == "FAILED" {
				os.Exit(getViolationExitCode())
			}
			return
		}

		// Terminal UI Mode
		PrintBanner()

		bold := color.New(color.Bold)
		green := color.New(color.FgGreen, color.Bold)
		yellow := color.New(color.FgYellow, color.Bold)
		red := color.New(color.FgRed, color.Bold)
		cyan := color.New(color.FgCyan)
		dim := color.New(color.Faint)

		bold.Println("  🩺 Razify Doctor Diagnostic Report")
		if fastMode {
			dim.Println("  ⚡ Fast mode active: port and service network probes skipped")
		}
		fmt.Println()

		currentCategory := ""
		for _, c := range report.Checks {
			if c.Category != currentCategory {
				currentCategory = c.Category
				bold.Printf("── %s ──\n", strings.ToUpper(currentCategory))
			}

			switch c.Status {
			case doctor.StatusPassed:
				green.Print("  ✔ ")
				fmt.Printf("%-24s %s\n", c.Name+":", c.Message)
			case doctor.StatusWarning:
				yellow.Print("  ⚠ ")
				fmt.Printf("%-24s %s\n", c.Name+":", c.Message)
				if c.Fix != "" {
					cyan.Printf("     └ Fix: %s\n", c.Fix)
				}
			case doctor.StatusFailed:
				red.Print("  ✘ ")
				fmt.Printf("%-24s %s\n", c.Name+":", c.Message)
				if c.Fix != "" {
					cyan.Printf("     └ Fix: %s\n", c.Fix)
				}
			case doctor.StatusSkipped:
				dim.Print("  ↷ ")
				dim.Printf("%-24s %s\n", c.Name+":", c.Message)
			}
		}

		fmt.Println()
		bold.Println("── HEALTH SCORE & SUMMARY ──")

		statusColor := green
		if report.HealthStatus == "UNHEALTHY" {
			statusColor = red
		} else if report.HealthStatus == "DEGRADED" {
			statusColor = yellow
		}

		statusColor.Printf("  Health Score:  %d/100 (%s)\n", report.HealthScore, report.HealthStatus)
		dim.Printf("  Checks:        %d passed, %d warnings, %d failed, %d skipped\n",
			report.PassedCount, report.WarningCount, report.FailedCount, report.SkippedCount)

		if len(report.Remediations) > 0 {
			fmt.Println()
			bold.Println("── RECOMMENDED ACTIONS ──")
			for i, rem := range report.Remediations {
				cyan.Printf("  %d. %s\n", i+1, rem)
			}
		}

		fmt.Println()

		if report.Result.Status == "FAILED" {
			os.Exit(getViolationExitCode())
		}
	},
}

func init() {
	rootCmd.AddCommand(doctorCmd)
	doctorCmd.Flags().Bool("json", false, "Output doctor diagnostics as JSON")
	doctorCmd.Flags().Bool("sarif", false, "Output doctor diagnostics as SARIF v2.1.0")
	doctorCmd.Flags().Bool("quiet", false, "Quiet mode: suppress output, exit with status code 0 (pass) or 1 (fail)")
	doctorCmd.Flags().Bool("fast", false, "Fast mode: skip network and service probes (sub-10ms)")
	doctorCmd.Flags().Bool("probe-remote", false, "Probe remote non-localhost services (default: localhost only)")
	doctorCmd.Flags().Int("timeout", 300, "Connection probe timeout in milliseconds")
	doctorCmd.Flags().Int("min-score", 90, "Minimum health score required to pass (default 90)")
}
