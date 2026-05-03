package cmd

import (
	"fmt"
	"os"
	"regexp"
	"strings"

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
		faint := color.New(color.Faint)

		bold.Println("\n  ┌─────────────────────────────┐")
		bold.Println("  │      Envy Audit Report      │")
		bold.Println("  └─────────────────────────────┘")
		fmt.Println()

		// ── SCAN ──────────────────────────────────────────
		faint.Println("  ▸ Running scan...")

		envVars, err := parseEnvFile(envFile)
		if err != nil {
			critical.Printf("  ✘  Could not read %s: %v\n", envFile, err)
			os.Exit(1)
		}

		scanCritical := 0
		scanHigh := 0
		scanMedium := 0

		dangerPats := []struct {
			pattern *regexp.Regexp
			risk    string
		}{
			{regexp.MustCompile(`(?i)(password|passwd|pwd)`), "HIGH"},
			{regexp.MustCompile(`(?i)(secret|private_key|privatekey)`), "HIGH"},
			{regexp.MustCompile(`(?i)(api_key|apikey|access_key)`), "HIGH"},
			{regexp.MustCompile(`(?i)(token|auth_token|jwt)`), "HIGH"},
			{regexp.MustCompile(`(?i)(aws_|gcp_|azure_)`), "CRITICAL"},
			{regexp.MustCompile(`(?i)(database_url|db_url|mongo_uri)`), "HIGH"},
			{regexp.MustCompile(`(?i)(webhook|slack_|discord_)`), "MEDIUM"},
		}

		weakVals := []string{
			"password", "123456", "secret", "test", "admin", "changeme", "1234", "qwerty",
		}

		for key, value := range envVars {
			if value == "" {
				continue
			}
			for _, dp := range dangerPats {
				if dp.pattern.MatchString(key) {
					switch dp.risk {
					case "CRITICAL":
						scanCritical++
					case "HIGH":
						scanHigh++
					case "MEDIUM":
						scanMedium++
					}
					break
				}
			}
			for _, weak := range weakVals {
				if strings.EqualFold(value, weak) {
					scanCritical++
					break
				}
			}
		}

		// ── VALIDATE ──────────────────────────────────────
		faint.Println("  ▸ Running validate...")

		exampleVars, err := parseEnvFile(exampleFile)
		if err != nil {
			critical.Printf("  ✘  Could not read %s: %v\n", exampleFile, err)
			os.Exit(1)
		}

		missing := 0
		placeholder := 0
		empty := 0
		passed := 0

		for key, exampleValue := range exampleVars {
			actualValue, exists := envVars[key]
			if !exists {
				missing++
				continue
			}
			if actualValue == "" {
				empty++
				continue
			}
			if exampleValue != "" && actualValue == exampleValue {
				placeholder++
				continue
			}
			passed++
		}

		// ── DIFF ──────────────────────────────────────────
		faint.Println("  ▸ Running diff...")

		diffCount := 0
		for key := range envVars {
			if _, exists := exampleVars[key]; !exists {
				diffCount++
			}
		}
		for key := range exampleVars {
			if _, exists := envVars[key]; !exists {
				diffCount++
			}
		}
		for key := range envVars {
			if val2, exists := exampleVars[key]; exists {
				if envVars[key] != val2 {
					diffCount++
				}
			}
		}

		// ── SCORE ─────────────────────────────────────────
		score := 100
		score -= scanCritical * 25
		score -= scanHigh * 10
		score -= scanMedium * 5
		score -= missing * 15
		score -= placeholder * 5
		score -= empty * 5
		if score < 0 {
			score = 0
		}

		// ── REPORT ────────────────────────────────────────
		fmt.Println()
		bold.Println("  ┌─────────────────────────────┐")
		bold.Println("  │          Results            │")
		bold.Println("  └─────────────────────────────┘")
		fmt.Println()

		// Scan results
		bold.Print("  Scan        ")
		if scanCritical > 0 {
			critical.Printf("%d CRITICAL  ", scanCritical)
		}
		if scanHigh > 0 {
			high.Printf("%d HIGH  ", scanHigh)
		}
		if scanMedium > 0 {
			medium.Printf("%d MEDIUM  ", scanMedium)
		}
		if scanCritical == 0 && scanHigh == 0 && scanMedium == 0 {
			success.Print("All clear")
		}
		fmt.Println()

		// Validate results
		bold.Print("  Validate    ")
		if missing > 0 {
			critical.Printf("%d MISSING  ", missing)
		}
		if placeholder > 0 {
			medium.Printf("%d PLACEHOLDER  ", placeholder)
		}
		if empty > 0 {
			high.Printf("%d EMPTY  ", empty)
		}
		if passed > 0 {
			success.Printf("%d OK  ", passed)
		}
		fmt.Println()

		// Diff results
		bold.Print("  Diff        ")
		if diffCount > 0 {
			high.Printf("%d difference(s) from %s", diffCount, exampleFile)
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
		if score < 40 {
			scoreColor = critical
			scoreLabel = "Critical — needs immediate attention"
		} else if score < 60 {
			scoreColor = high
			scoreLabel = "Poor — several issues found"
		} else if score < 80 {
			scoreColor = medium
			scoreLabel = "Fair — some improvements needed"
		}

		scoreColor.Printf("  %d/100  %s\n", score, scoreLabel)
		fmt.Println()

		// Recommendations
		if scanCritical > 0 || missing > 0 {
			bold.Println("  Recommendations:")
			if scanCritical > 0 {
				critical.Println("  ✘  Rotate exposed credentials immediately")
			}
			if missing > 0 {
				high.Println("  ⚠  Add missing required variables before deploying")
			}
			if placeholder > 0 {
				medium.Println("  ~  Replace placeholder values with real ones")
			}
			fmt.Println()
		} else {
			success.Println("  ✔  Your environment looks healthy!")
			fmt.Println()
		}
	},
}

func init() {
	rootCmd.AddCommand(auditCmd)
}
