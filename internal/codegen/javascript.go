package codegen

import (
	"bufio"
	"bytes"
	"regexp"
	"strings"
)

var (
	// Matches process.env.KEY and process.env['KEY'] / process.env["KEY"]
	jsProcessEnvDotRegex     = regexp.MustCompile(`process\.env\.([A-Za-z0-9_]+)`)
	jsProcessEnvBracketRegex = regexp.MustCompile(`process\.env\[['"]([A-Za-z0-9_]+)['"]\]`)

	// Matches import.meta.env.KEY and import.meta.env['KEY'] / import.meta.env["KEY"]
	jsImportMetaDotRegex     = regexp.MustCompile(`import\.meta\.env\.([A-Za-z0-9_]+)`)
	jsImportMetaBracketRegex = regexp.MustCompile(`import\.meta\.env\[['"]([A-Za-z0-9_]+)['"]\]`)

	// Fallback/default detection e.g. process.env.KEY || 'default' or ?? "default" or || 3000
	jsDefaultStringRegex = regexp.MustCompile(`(?:process\.env\.[A-Za-z0-9_]+|process\.env\[['"][^'"]+['"]\]|import\.meta\.env\.[A-Za-z0-9_]+|import\.meta\.env\[['"][^'"]+['"]\])\s*(?:\|\||\?\?)\s*['"]([^'"]*)['"]`)
	jsDefaultNumberRegex = regexp.MustCompile(`(?:process\.env\.[A-Za-z0-9_]+|process\.env\[['"][^'"]+['"]\]|import\.meta\.env\.[A-Za-z0-9_]+|import\.meta\.env\[['"][^'"]+['"]\])\s*(?:\|\||\?\?)\s*([0-9]+)`)
	jsDefaultBoolRegex   = regexp.MustCompile(`(?:process\.env\.[A-Za-z0-9_]+|process\.env\[['"][^'"]+['"]\]|import\.meta\.env\.[A-Za-z0-9_]+|import\.meta\.env\[['"][^'"]+['"]\])\s*(?:\|\||\?\?)\s*(true|false)`)
)

// ParseJSFile scans TypeScript / JavaScript source code for environment variable references.
func ParseJSFile(filePath string, src []byte) ([]ExtractedVar, error) {
	var results []ExtractedVar
	scanner := bufio.NewScanner(bytes.NewReader(src))
	lineNum := 0

	for scanner.Scan() {
		lineNum++
		line := scanner.Text()
		trimmed := strings.TrimSpace(line)

		// Skip comments
		if strings.HasPrefix(trimmed, "//") || strings.HasPrefix(trimmed, "/*") || strings.HasPrefix(trimmed, "*") {
			continue
		}

		// Find all env keys on this line
		keysOnLine := extractJSKeys(line)
		if len(keysOnLine) == 0 {
			continue
		}

		for _, key := range keysOnLine {
			loc := Location{
				File: filePath,
				Line: lineNum,
			}

			inferredType := inferJSTypeFromContext(key, line)
			category := inferCategory(key, inferredType)
			defaultVal := extractJSDefault(line)
			if defaultVal == "" {
				defaultVal = inferDefaultValue(key, inferredType)
			}

			results = append(results, ExtractedVar{
				Name:         key,
				DefaultValue: defaultVal,
				InferredType: inferredType,
				IsRequired:   false, // JS default or fallback
				Category:     category,
				Occurrences:  []Location{loc},
			})
		}
	}

	return results, scanner.Err()
}

func extractJSKeys(line string) []string {
	seen := make(map[string]bool)
	var keys []string

	addKey := func(k string) {
		if k != "" && !seen[k] {
			seen[k] = true
			keys = append(keys, k)
		}
	}

	for _, m := range jsProcessEnvDotRegex.FindAllStringSubmatch(line, -1) {
		addKey(m[1])
	}
	for _, m := range jsProcessEnvBracketRegex.FindAllStringSubmatch(line, -1) {
		addKey(m[1])
	}
	for _, m := range jsImportMetaDotRegex.FindAllStringSubmatch(line, -1) {
		addKey(m[1])
	}
	for _, m := range jsImportMetaBracketRegex.FindAllStringSubmatch(line, -1) {
		addKey(m[1])
	}

	return keys
}

func extractJSDefault(line string) string {
	if m := jsDefaultStringRegex.FindStringSubmatch(line); len(m) > 1 {
		return m[1]
	}
	if m := jsDefaultNumberRegex.FindStringSubmatch(line); len(m) > 1 {
		return m[1]
	}
	if m := jsDefaultBoolRegex.FindStringSubmatch(line); len(m) > 1 {
		return m[1]
	}
	return ""
}

func inferJSTypeFromContext(key, line string) string {
	upper := strings.ToUpper(key)

	// Check line context
	if strings.Contains(line, "parseInt(") || strings.Contains(line, "Number(") {
		if strings.Contains(upper, "PORT") {
			return "port"
		}
		return "int"
	}
	if strings.Contains(line, "Boolean(") || strings.Contains(line, "=== 'true'") || strings.Contains(line, "=== \"true\"") {
		return "bool"
	}

	// Name heuristics
	if strings.Contains(upper, "PORT") {
		return "port"
	}
	if strings.Contains(upper, "URL") || strings.Contains(upper, "URI") || strings.Contains(upper, "ENDPOINT") {
		return "url"
	}
	if strings.Contains(upper, "KEY") || strings.Contains(upper, "SECRET") || strings.Contains(upper, "TOKEN") ||
		strings.Contains(upper, "PASSWORD") || strings.Contains(upper, "AUTH") || strings.Contains(upper, "PRIVATE") {
		return "secret"
	}
	if strings.Contains(upper, "EMAIL") {
		return "email"
	}
	if strings.HasPrefix(upper, "ENABLE_") || strings.HasPrefix(upper, "DISABLE_") ||
		strings.HasPrefix(upper, "IS_") || strings.HasPrefix(upper, "HAS_") ||
		strings.Contains(upper, "DEBUG") {
		return "bool"
	}

	return "string"
}
