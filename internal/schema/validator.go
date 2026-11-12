package schema

import (
	"encoding/json"
	"fmt"
	"net"
	"net/url"
	"regexp"
	"strconv"
	"strings"
)

var (
	emailRegex = regexp.MustCompile(`^[a-zA-Z0-9._%+-]+@[a-zA-Z0-9.-]+\.[a-zA-Z]{2,}$`)
	uuidRegex  = regexp.MustCompile(`(?i)^[0-9a-f]{8}-[0-9a-f]{4}-[1-5][0-9a-f]{3}-[89ab][0-9a-f]{3}-[0-9a-f]{12}$`)

	obviousPlaceholders = []string{
		"your_", "change_me", "changeme", "xxx", "example",
		"replace_", "fill_in", "todo", "fixme", "<", ">",
		"placeholder", "your-", "put_your", "insert_",
	}
)

// Validate checks envVars against exampleVars according to schema rules.
func Validate(envVars map[string]string, exampleVars []EnvVar) (ValidateSummary, []ValidationResult) {
	var results []ValidationResult
	summary := ValidateSummary{}

	for _, ev := range exampleVars {
		actualValue, exists := envVars[ev.Key]

		if !exists {
			if ev.Required {
				results = append(results, ValidationResult{
					Key:     ev.Key,
					Status:  "MISSING",
					Message: "Required key not found",
				})
				summary.Missing++
			}
			continue
		}

		if actualValue == "" {
			results = append(results, ValidationResult{
				Key:     ev.Key,
				Status:  "EMPTY",
				Message: "Key exists but has no value",
			})
			summary.Empty++
			continue
		}

		// Placeholder check
		if ev.Value != "" && actualValue == ev.Value {
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
				summary.Placeholder++
				continue
			}
		}

		// Advanced validation from tags
		if errStr := ValidateValue(actualValue, ev.Tags, envVars); errStr != "" {
			results = append(results, ValidationResult{
				Key:     ev.Key,
				Status:  "INVALID",
				Message: errStr,
			})
			summary.Missing++ // Treated as missing/failed in summary count
			continue
		}

		results = append(results, ValidationResult{
			Key:    ev.Key,
			Status: "OK",
		})
		summary.Passed++
	}

	return summary, results
}

// RunValidate parses the files and performs validation, returning summary counts and results.
func RunValidate(envFile, exampleFile string) (int, int, int, int, []ValidationResult, error) {
	envVars, err := ParseEnvFile(envFile)
	if err != nil {
		return 0, 0, 0, 0, nil, err
	}

	exampleVars, err := ParseEnvWithMetadata(exampleFile)
	if err != nil {
		return 0, 0, 0, 0, nil, err
	}

	summary, results := Validate(envVars, exampleVars)
	return summary.Missing, summary.Placeholder, summary.Empty, summary.Passed, results, nil
}

// ValidateValue checks an individual value against schema tags (@type, @enum, @range, @requires).
func ValidateValue(value string, tags map[string]string, envVars map[string]string) string {
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
			return fmt.Sprintf("Value is not in allowed enum options [%s]", e)
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

