package codegen

import (
	"bufio"
	"bytes"
	"regexp"
	"strings"
)

var (
	// os.getenv('KEY', 'default') or os.environ.get('KEY', 'default')
	pyGetEnvWithDefaultRegex = regexp.MustCompile(`(?:os\.getenv|os\.environ\.get)\(\s*['"]([A-Za-z0-9_]+)['"]\s*,\s*['"]([^'"]*)['"]\s*\)`)
	pyGetEnvNumericDefault   = regexp.MustCompile(`(?:os\.getenv|os\.environ\.get)\(\s*['"]([A-Za-z0-9_]+)['"]\s*,\s*([0-9]+)\s*\)`)
	pyGetEnvBoolDefault      = regexp.MustCompile(`(?:os\.getenv|os\.environ\.get)\(\s*['"]([A-Za-z0-9_]+)['"]\s*,\s*(True|False)\s*\)`)

	// os.getenv('KEY') or os.environ.get('KEY') without default
	pyGetEnvRegex = regexp.MustCompile(`(?:os\.getenv|os\.environ\.get)\(\s*['"]([A-Za-z0-9_]+)['"]\s*\)`)

	// os.environ['KEY'] or os.environ["KEY"]
	pyEnvironBracketRegex = regexp.MustCompile(`os\.environ\[\s*['"]([A-Za-z0-9_]+)['"]\s*\]`)
)

// ParsePythonFile scans Python source code for environment variable references.
func ParsePythonFile(filePath string, src []byte) ([]ExtractedVar, error) {
	var results []ExtractedVar
	scanner := bufio.NewScanner(bytes.NewReader(src))
	lineNum := 0

	for scanner.Scan() {
		lineNum++
		line := scanner.Text()
		trimmed := strings.TrimSpace(line)

		// Skip comments
		if strings.HasPrefix(trimmed, "#") {
			continue
		}

		loc := Location{
			File: filePath,
			Line: lineNum,
		}

		// 1. With string default
		for _, m := range pyGetEnvWithDefaultRegex.FindAllStringSubmatch(line, -1) {
			key := m[1]
			defVal := m[2]
			inferredType := inferPyTypeFromContext(key, line)
			results = append(results, ExtractedVar{
				Name:         key,
				DefaultValue: defVal,
				InferredType: inferredType,
				IsRequired:   false,
				Category:     inferCategory(key, inferredType),
				Occurrences:  []Location{loc},
			})
		}

		// 2. With numeric default
		for _, m := range pyGetEnvNumericDefault.FindAllStringSubmatch(line, -1) {
			key := m[1]
			defVal := m[2]
			inferredType := inferPyTypeFromContext(key, line)
			if inferredType == "string" {
				inferredType = "int"
			}
			results = append(results, ExtractedVar{
				Name:         key,
				DefaultValue: defVal,
				InferredType: inferredType,
				IsRequired:   false,
				Category:     inferCategory(key, inferredType),
				Occurrences:  []Location{loc},
			})
		}

		// 3. With bool default
		for _, m := range pyGetEnvBoolDefault.FindAllStringSubmatch(line, -1) {
			key := m[1]
			defVal := strings.ToLower(m[2])
			results = append(results, ExtractedVar{
				Name:         key,
				DefaultValue: defVal,
				InferredType: "bool",
				IsRequired:   false,
				Category:     inferCategory(key, "bool"),
				Occurrences:  []Location{loc},
			})
		}

		// 4. Direct bracket access: os.environ['KEY'] (raises KeyError if missing -> required)
		for _, m := range pyEnvironBracketRegex.FindAllStringSubmatch(line, -1) {
			key := m[1]
			inferredType := inferPyTypeFromContext(key, line)
			defVal := inferDefaultValue(key, inferredType)
			results = append(results, ExtractedVar{
				Name:         key,
				DefaultValue: defVal,
				InferredType: inferredType,
				IsRequired:   true,
				Category:     inferCategory(key, inferredType),
				Occurrences:  []Location{loc},
			})
		}

		// 5. Without default: os.getenv('KEY')
		for _, m := range pyGetEnvRegex.FindAllStringSubmatch(line, -1) {
			key := m[1]
			inferredType := inferPyTypeFromContext(key, line)
			defVal := inferDefaultValue(key, inferredType)
			results = append(results, ExtractedVar{
				Name:         key,
				DefaultValue: defVal,
				InferredType: inferredType,
				IsRequired:   false,
				Category:     inferCategory(key, inferredType),
				Occurrences:  []Location{loc},
			})
		}
	}

	return results, scanner.Err()
}

func inferPyTypeFromContext(key, line string) string {
	upper := strings.ToUpper(key)

	// Check line context
	if strings.Contains(line, "int(os.getenv") || strings.Contains(line, "int(os.environ") {
		if strings.Contains(upper, "PORT") {
			return "port"
		}
		return "int"
	}
	if strings.Contains(line, ".lower() == 'true'") || strings.Contains(line, ".lower() == \"true\"") ||
		strings.Contains(line, "bool(os.getenv") {
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
