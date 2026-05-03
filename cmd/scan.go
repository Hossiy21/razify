package cmd

import (
	"bufio"
	"fmt"
	"os"
	"regexp"
	"strings"

	"github.com/spf13/cobra"
)

type ScanResult struct {
	Line   int
	Key    string
	Value  string
	Reason string
	Risk   string
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

			// Skip empty values
			if value == "" {
				continue
			}

			// Check key against danger patterns
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

			// Check for weak values
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

		// Print results
		fmt.Printf("\nScanning %s...\n\n", filename)

		if len(results) == 0 {
			fmt.Println("  All clear! No secrets or weak values found.")
			return
		}

		critical := 0
		high := 0
		medium := 0

		for _, r := range results {
			icon := riskIcon(r.Risk)
			fmt.Printf("  %s [%s] Line %d: %s\n", icon, r.Risk, r.Line, r.Key)
			fmt.Printf("     Value : %s\n", r.Value)
			fmt.Printf("     Reason: %s\n\n", r.Reason)

			switch r.Risk {
			case "CRITICAL":
				critical++
			case "HIGH":
				high++
			case "MEDIUM":
				medium++
			}
		}

		fmt.Printf("Summary: %d CRITICAL  %d HIGH  %d MEDIUM\n\n", critical, high, medium)

		if critical > 0 {
			fmt.Println("  ACTION REQUIRED: Never commit this file to git!")
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
		return "!!"
	case "HIGH":
		return "! "
	case "MEDIUM":
		return "~ "
	default:
		return "  "
	}
}

func init() {
	rootCmd.AddCommand(scanCmd)
}
