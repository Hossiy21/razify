package codegen

import (
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

var defaultExcludes = map[string]bool{
	".git":         true,
	"node_modules": true,
	"vendor":       true,
	".venv":        true,
	"venv":         true,
	"__pycache__":  true,
	"dist":         true,
	"build":        true,
	".next":        true,
	"target":       true,
	"bin":          true,
	"obj":          true,
}

// Extract walks the root directory and extracts environment variable references from Go, JS/TS, and Python files.
func Extract(opts ExtractOptions) (*ExtractionReport, error) {
	if opts.RootDir == "" {
		opts.RootDir = "."
	}

	excludes := make(map[string]bool)
	for k, v := range defaultExcludes {
		excludes[k] = v
	}
	for _, e := range opts.ExcludeDirs {
		excludes[e] = true
	}

	report := &ExtractionReport{
		LanguagesDetected: []string{},
	}

	langMap := make(map[string]bool)
	var rawVars []ExtractedVar

	err := filepath.WalkDir(opts.RootDir, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}

		name := d.Name()
		if d.IsDir() {
			if excludes[name] || (strings.HasPrefix(name, ".") && name != ".") {
				return filepath.SkipDir
			}
			return nil
		}

		ext := strings.ToLower(filepath.Ext(path))
		switch ext {
		case ".go":
			if strings.HasSuffix(name, "_test.go") {
				return nil
			}
			data, err := os.ReadFile(path)
			if err != nil {
				return nil
			}
			report.FilesScanned++
			langMap["Go"] = true
			vars, err := ParseGoFile(path, data)
			if err == nil {
				rawVars = append(rawVars, vars...)
			}

		case ".ts", ".tsx", ".js", ".jsx", ".mjs", ".cjs":
			if strings.HasSuffix(name, ".d.ts") || strings.HasSuffix(name, ".test.ts") || strings.HasSuffix(name, ".spec.ts") {
				return nil
			}
			data, err := os.ReadFile(path)
			if err != nil {
				return nil
			}
			report.FilesScanned++
			langMap["TypeScript/JavaScript"] = true
			vars, err := ParseJSFile(path, data)
			if err == nil {
				rawVars = append(rawVars, vars...)
			}

		case ".py":
			if strings.HasPrefix(name, "test_") || strings.HasSuffix(name, "_test.py") {
				return nil
			}
			data, err := os.ReadFile(path)
			if err != nil {
				return nil
			}
			report.FilesScanned++
			langMap["Python"] = true
			vars, err := ParsePythonFile(path, data)
			if err == nil {
				rawVars = append(rawVars, vars...)
			}
		}

		return nil
	})

	if err != nil {
		return nil, fmt.Errorf("failed walking directory %s: %w", opts.RootDir, err)
	}

	for l := range langMap {
		report.LanguagesDetected = append(report.LanguagesDetected, l)
	}
	sort.Strings(report.LanguagesDetected)

	report.Variables = mergeAndDeduplicate(rawVars)
	return report, nil
}

func mergeAndDeduplicate(vars []ExtractedVar) []ExtractedVar {
	merged := make(map[string]*ExtractedVar)

	for _, v := range vars {
		existing, ok := merged[v.Name]
		if !ok {
			copyVar := v
			merged[v.Name] = &copyVar
			continue
		}

		// Merge occurrences
		existing.Occurrences = append(existing.Occurrences, v.Occurrences...)

		// Upgrade type priority: port/url/secret/bool/int > string
		if typePriority(v.InferredType) > typePriority(existing.InferredType) {
			existing.InferredType = v.InferredType
		}

		// Keep explicit non-empty default
		if existing.DefaultValue == "" && v.DefaultValue != "" {
			existing.DefaultValue = v.DefaultValue
		}

		// If any caller requires it, mark required
		if v.IsRequired {
			existing.IsRequired = true
		}

		// Prefer specific category over General
		if existing.Category == "General" && v.Category != "General" {
			existing.Category = v.Category
		}
	}

	var result []ExtractedVar
	for _, v := range merged {
		result = append(result, *v)
	}

	// Sort by Category then Name
	sort.Slice(result, func(i, j int) bool {
		catOrder := map[string]int{
			"Server":        1,
			"Database":      2,
			"Security":      3,
			"Feature Flags": 4,
			"General":       5,
		}
		c1, c2 := catOrder[result[i].Category], catOrder[result[j].Category]
		if c1 != c2 {
			return c1 < c2
		}
		return result[i].Name < result[j].Name
	})

	return result
}

func typePriority(t string) int {
	switch t {
	case "port":
		return 6
	case "url":
		return 5
	case "secret":
		return 4
	case "bool":
		return 3
	case "int":
		return 2
	case "email":
		return 2
	default:
		return 1
	}
}

// GenerateEnvExample renders a clean, annotated .env.example string from extracted variables.
func GenerateEnvExample(vars []ExtractedVar) string {
	var sb strings.Builder

	sb.WriteString("# ==============================================================================\n")
	sb.WriteString("# Environment Configuration Schema\n")
	sb.WriteString("# Generated automatically by 'razify init --from-code'\n")
	sb.WriteString("# Annotate comments with @type, @required, @enum, or @range for schema validation\n")
	sb.WriteString("# ==============================================================================\n\n")

	currentCategory := ""
	for _, v := range vars {
		if v.Category != currentCategory {
			currentCategory = v.Category
			sb.WriteString(fmt.Sprintf("# ------------------------------------------------------------------------------\n"))
			sb.WriteString(fmt.Sprintf("# %s Configuration\n", currentCategory))
			sb.WriteString(fmt.Sprintf("# ------------------------------------------------------------------------------\n"))
		}

		// Build tag comment
		var tags []string
		if v.InferredType != "" && v.InferredType != "string" {
			tags = append(tags, fmt.Sprintf("@type %s", v.InferredType))
		}
		if v.IsRequired {
			tags = append(tags, "@required")
		}

		tagStr := ""
		if len(tags) > 0 {
			tagStr = " " + strings.Join(tags, " ")
		}

		desc := v.Description
		if desc == "" {
			desc = formatDescription(v.Name, v.InferredType)
		}

		sb.WriteString(fmt.Sprintf("# %s%s\n", desc, tagStr))

		val := v.DefaultValue
		if val == "" {
			val = inferDefaultValue(v.Name, v.InferredType)
		}
		sb.WriteString(fmt.Sprintf("%s=%s\n\n", v.Name, val))
	}

	return strings.TrimRight(sb.String(), "\n") + "\n"
}

func formatDescription(key, inferredType string) string {
	upper := strings.ToUpper(key)
	switch inferredType {
	case "port":
		return "HTTP/Service network listening port"
	case "url":
		if strings.Contains(upper, "DB") || strings.Contains(upper, "DATABASE") {
			return "Database connection URL string"
		}
		if strings.Contains(upper, "REDIS") {
			return "Redis cache connection URL string"
		}
		return "Service endpoint URL"
	case "secret":
		return "Secret API credential or token"
	case "bool":
		return "Feature flag toggle (true or false)"
	case "email":
		return "Administrative notification email address"
	default:
		return "Application configuration parameter"
	}
}
