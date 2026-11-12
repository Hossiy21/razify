package scanner

import (
	"fmt"
	"strings"

	"github.com/Hossiy21/razify/internal/schema"
)

// ScanVars scans parsed environment variables for secrets, high-entropy tokens, and weak passwords.
func ScanVars(vars []schema.EnvVar) ([]ScanResult, ScanSummary) {
	var results []ScanResult
	summary := ScanSummary{}

	for _, v := range vars {
		if v.Ignored {
			continue
		}

		key := v.Key
		value := v.Value

		if value == "" {
			continue
		}

		found := false

		// 1. Check for value-based patterns (most accurate)
		for _, sp := range SecretValuePatterns {
			if sp.Pattern.MatchString(value) {
				results = append(results, ScanResult{
					Line:   v.LineNum,
					Key:    key,
					Value:  MaskValue(value),
					Reason: sp.Reason,
					Risk:   sp.Risk,
				})
				found = true
				break
			}
		}

		// 2. Check for weak values
		if !found {
			for _, weak := range WeakValues {
				if strings.EqualFold(value, weak) {
					keyLower := strings.ToLower(key)
					isSecretKey := strings.Contains(keyLower, "password") ||
						strings.Contains(keyLower, "passwd") ||
						strings.Contains(keyLower, "secret") ||
						strings.Contains(keyLower, "token") ||
						strings.Contains(keyLower, "key") ||
						strings.Contains(keyLower, "pwd")
					if !isSecretKey {
						break
					}
					results = append(results, ScanResult{
						Line:   v.LineNum,
						Key:    key,
						Value:  MaskValue(value),
						Reason: "Weak or default value detected",
						Risk:   "CRITICAL",
					})
					found = true
					break
				}
			}
		}

		// 3. Check for key-based patterns
		if !found {
			for _, dp := range DangerPatterns {
				if dp.Pattern.MatchString(key) {
					results = append(results, ScanResult{
						Line:   v.LineNum,
						Key:    key,
						Value:  MaskValue(value),
						Reason: dp.Reason,
						Risk:   dp.Risk,
					})
					found = true
					break
				}
			}
		}

		// 4. Entropy analysis for suspicious keys
		if !found && len(value) > 16 {
			entropy := ShannonEntropy(value)
			if entropy > 4.2 {
				results = append(results, ScanResult{
					Line:   v.LineNum,
					Key:    key,
					Value:  MaskValue(value),
					Reason: fmt.Sprintf("High entropy detected (%.2f) — looks like a random secret", entropy),
					Risk:   "MEDIUM",
				})
			}
		}
	}

	for _, r := range results {
		switch r.Risk {
		case "CRITICAL":
			summary.Critical++
		case "HIGH":
			summary.High++
		case "MEDIUM":
			summary.Medium++
		}
	}
	summary.Total = len(results)

	return results, summary
}

// RunScan parses a file and scans it, returning results.
func RunScan(filename string) ([]ScanResult, error) {
	vars, err := schema.ParseEnvWithMetadata(filename)
	if err != nil {
		return nil, err
	}

	results, _ := ScanVars(vars)
	return results, nil
}

