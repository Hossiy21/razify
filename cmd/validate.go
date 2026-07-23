package cmd

import (
	"encoding/json"
	"fmt"
	"net"
	"net/url"
	"os"
	"regexp"
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
		if errStr := validateValue(actualValue, ev.Tags, envVars); errStr != "" {
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

var (
	emailRegex = regexp.MustCompile(`^[a-zA-Z0-9._%+-]+@[a-zA-Z0-9.-]+\.[a-zA-Z]{2,}$`)
	uuidRegex  = regexp.MustCompile(`(?i)^[0-9a-f]{8}-[0-9a-f]{4}-[1-5][0-9a-f]{3}-[89ab][0-9a-f]{3}-[0-9a-f]{12}$`)
)

func validateValue(value string, tags map[string]string, envVars map[string]string) string {
	if tags == nil {
		return ""
	}

	// 1. Type validation
	if t, ok := tags["type"]; ok {
		switch strings.ToLower(t) {
		case "int", "integer":
			if _, err := strconv.Atoi(value); err != nil {
				return "Value must be a valid integer"
			}
		case "bool", "boolean":
			valLower := strings.ToLower(value)
			if valLower != "true" && valLower != "false" && valLower != "1" && valLower != "0" {
				return "Value must be a boolean ('true', 'false', '1', or '0')"
			}
		case "url", "uri":
			if _, err := url.ParseRequestURI(value); err != nil {
				return "Value must be a valid URL (e.g. https://example.com)"
			}
		case "email":
			if !emailRegex.MatchString(value) {
				return "Value must be a valid email address"
			}
		case "port":
			p, err := strconv.Atoi(value)
			if err != nil || p < 1 || p > 65535 {
				return "Value must be a valid network port (1 - 65535)"
			}
		case "uuid":
			if !uuidRegex.MatchString(value) {
				return "Value must be a valid UUID string"
			}
		case "json":
			if !json.Valid([]byte(value)) {
				return "Value must be valid JSON text"
			}
		case "ip":
			if net.ParseIP(value) == nil {
				return "Value must be a valid IPv4 or IPv6 address"
			}
		}
	}

	// 2. Enum validation (e.g. @enum(dev,staging,prod) or @enum=dev,staging,prod)
	if e, ok := tags["enum"]; ok {
		var options []string
		if strings.Contains(e, "|") {
			options = strings.Split(e, "|")
		} else {
			options = strings.Split(e, ",")
		}
		found := false
		for _, opt := range options {
			if strings.TrimSpace(opt) == value {
				found = true
				break
			}
		}
		if !found {
			return fmt.Sprintf("Value '%s' is not in allowed enum options [%s]", value, e)
		}
	}

	// 3. Range validation (e.g. @range(1000-65535) or @range=1-100)
	if r, ok := tags["range"]; ok {
		var parts []string
		if strings.Contains(r, ",") {
			parts = strings.Split(r, ",")
		} else {
			parts = strings.Split(r, "-")
		}
		if len(parts) == 2 {
			min, _ := strconv.Atoi(strings.TrimSpace(parts[0]))
			max, _ := strconv.Atoi(strings.TrimSpace(parts[1]))
			val, err := strconv.Atoi(value)
			if err == nil {
				if val < min || val > max {
					return fmt.Sprintf("Value %d is out of range [%d - %d]", val, min, max)
				}
			}
		}
	}

	// 4. Cross-variable dependencies (e.g. @requires(DB_HOST))
	if reqKey, ok := tags["requires"]; ok {
		if envVars != nil {
			depVal, exists := envVars[reqKey]
			if !exists || strings.TrimSpace(depVal) == "" {
				return fmt.Sprintf("Requires dependent variable '%s' to be set in environment", reqKey)
			}
		}
	}

	return ""
}

func init() {
	rootCmd.AddCommand(validateCmd)
	validateCmd.Flags().Bool("json", false, "Output results as JSON")
}
