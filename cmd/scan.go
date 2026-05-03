package cmd

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"regexp"
	"strings"

	"github.com/fatih/color"
	"github.com/spf13/cobra"
)

type ScanResult struct {
	Line   int    `json:"line"`
	Key    string `json:"key"`
	Value  string `json:"value"`
	Reason string `json:"reason"`
	Risk   string `json:"risk"`
}

type ScanOutput struct {
	File    string       `json:"file"`
	Results []ScanResult `json:"results"`
	Summary ScanSummary  `json:"summary"`
}

type ScanSummary struct {
	Critical int `json:"critical"`
	High     int `json:"high"`
	Medium   int `json:"medium"`
	Total    int `json:"total"`
}

var dangerPatterns = []struct {
	pattern *regexp.Regexp
	reason  string
	risk    string
}{
	{regexp.MustCompile(`(?i)(password|passwd|pwd)`), "Looks like a password field", "HIGH"},
	{regexp.MustCompile(`(?i)(secret|private_key|privatekey)`), "Looks like a secret/private key", "HIGH"},
	{regexp.MustCompile(`(?i)(api_key|apikey|access_key)`), "Looks like an API key", "HIGH"},
	{regexp.MustCompile(`(?i)(token|auth_token|jwt)`), "Looks like an auth token", "HIGH"},
	{regexp.MustCompile(`(?i)(aws_|gcp_|azure_)`), "Cloud provider credential", "CRITICAL"},
	{regexp.MustCompile(`(?i)(database_url|db_url|mongo_uri)`), "Database connection string", "HIGH"},
	{regexp.MustCompile(`(?i)(webhook|slack_|discord_)`), "Webhook URL — can be abused", "MEDIUM"},
}

var weakValues = []string{
	"password", "123456", "secret", "test", "admin", "changeme", "1234", "qwerty",
}

var scanCmd = &cobra.Command{
	Use:   "scan [file]",
	Short: "Scan an .env file for secret leaks and weak values",
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		filename := args[0]
		jsonOutput, _ := cmd.Flags().GetBool("json")

		file, err := os.Open(filename)
		if err != nil {
			fmt.Printf("Error opening %s: %v\n", filename, err)
			os.Exit(1)
		}
		defer file.Close()

		var results []ScanResult
		lineNum := 0

		scanner := bufio.NewScanner(file)
		for scanner.Scan() {
			lineNum++
			line := strings.TrimSpace(scanner.Text())

			if line == "" || strings.HasPrefix(line, "#") {
				continue
			}

			parts := strings.SplitN(line, "=", 2)
			if len(parts) != 2 {
				continue
			}

			key := strings.TrimSpace(parts[0])
			value := strings.TrimSpace(parts[1])

			if value == "" {
				continue
			}

			for _, dp := range dangerPatterns {
				if dp.pattern.MatchString(key) {
					results = append(results, ScanResult{
						Line:   lineNum,
						Key:    key,
						Value:  maskValue(value),
						Reason: dp.reason,
						Risk:   dp.risk,
					})
					break
				}
			}

			for _, weak := range weakValues {
				if strings.EqualFold(value, weak) {
					results = append(results, ScanResult{
						Line:   lineNum,
						Key:    key,
						Value:  maskValue(value),
						Reason: "Weak or default value detected",
						Risk:   "CRITICAL",
					})
					break
				}
			}
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
				os.Exit(1)
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
			os.Exit(1)
		}
	},
}

func maskValue(value string) string {
	if len(value) <= 4 {
		return "****"
	}
	return value[:2] + strings.Repeat("*", len(value)-4) + value[len(value)-2:]
}

func riskIcon(risk string) string {
	switch risk {
	case "CRITICAL":
		return "✘ "
	case "HIGH":
		return "⚠ "
	case "MEDIUM":
		return "~ "
	default:
		return "  "
	}
}

func init() {
	rootCmd.AddCommand(scanCmd)
	scanCmd.Flags().Bool("json", false, "Output results as JSON")
}
