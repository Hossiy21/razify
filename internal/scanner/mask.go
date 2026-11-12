package scanner

import "strings"

// MaskValue masks sensitive values while keeping a few prefix and suffix characters
// ONLY for long values (len >= 20). Short values (< 20 chars) are fully redacted with
// asterisks so zero fragments of the secret are leaked.
func MaskValue(value string) string {
	if len(value) < 20 {
		count := len(value)
		if count < 8 {
			count = 8
		}
		return strings.Repeat("*", count)
	}
	// For long values (len >= 20), show 2 prefix and 2 suffix chars for operational context
	middle := len(value) - 4
	return value[:2] + strings.Repeat("*", middle) + value[len(value)-2:]
}

// RiskIcon returns a terminal-friendly indicator for a given risk severity.
func RiskIcon(risk string) string {
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
