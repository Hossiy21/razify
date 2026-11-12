package scanner

import "strings"

// MaskValue masks sensitive values while keeping a few prefix and suffix characters for context.
func MaskValue(value string) string {
	if len(value) <= 4 {
		return strings.Repeat("*", 4)
	}
	middle := len(value) - 4
	if middle < 2 {
		middle = 2
	}
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

