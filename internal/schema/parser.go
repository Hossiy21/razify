package schema

import (
	"bufio"
	"io"
	"os"
	"regexp"
	"strings"
)

var (
	tagParenRegex      = regexp.MustCompile(`@(\w+)\(([^)]+)\)`)
	tagEqRegex         = regexp.MustCompile(`@(\w+)=([\w\-\.,]+)`)
	tagStandaloneRegex = regexp.MustCompile(`@(\w+)\b`)
	tagStripRegex      = regexp.MustCompile(`@\w+=[\w\-\.,]+`)
)

// ParseEnvFile parses a basic key=value .env file.
func ParseEnvFile(filename string) (map[string]string, error) {
	file, err := os.Open(filename)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	return ParseEnvReader(file)
}

// ParseEnvReader parses key=value pairs from an io.Reader.
func ParseEnvReader(r io.Reader) (map[string]string, error) {
	result := make(map[string]string)
	scanner := bufio.NewScanner(r)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		parts := strings.SplitN(line, "=", 2)
		if len(parts) == 2 {
			result[strings.TrimSpace(parts[0])] = strings.TrimSpace(parts[1])
		}
	}
	return result, scanner.Err()
}

// ParseEnvWithMetadata parses a .env or .env.example file extracting comments, categories, and @tags.
func ParseEnvWithMetadata(filename string) ([]EnvVar, error) {
	file, err := os.Open(filename)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	return ParseEnvWithMetadataReader(file)
}

// ParseEnvWithMetadataReader parses .env file lines with comments and schema metadata from an io.Reader.
func ParseEnvWithMetadataReader(r io.Reader) ([]EnvVar, error) {
	var vars []EnvVar
	var lastComment string
	var currentCategory string
	lineNum := 0

	scanner := bufio.NewScanner(r)
	for scanner.Scan() {
		lineNum++
		line := strings.TrimSpace(scanner.Text())

		if line == "" {
			lastComment = ""
			continue
		}

		if strings.HasPrefix(line, "##") {
			currentCategory = strings.TrimSpace(strings.TrimPrefix(line, "##"))
			lastComment = ""
			continue
		}

		if strings.HasPrefix(line, "#") {
			lastComment = strings.TrimSpace(strings.TrimPrefix(line, "#"))
			continue
		}

		parts := strings.SplitN(line, "=", 2)
		if len(parts) != 2 {
			continue
		}

		key := strings.TrimSpace(parts[0])
		valueWithComment := strings.TrimSpace(parts[1])

		value := valueWithComment
		trailingComment := ""

		// Check for trailing comment (naive but common)
		if hashIdx := strings.Index(valueWithComment, "#"); hashIdx != -1 {
			// Basic support for quoted values (to avoid splitting on # inside quotes)
			if !strings.HasPrefix(valueWithComment, "\"") || !strings.HasSuffix(valueWithComment, "\"") {
				value = strings.TrimSpace(valueWithComment[:hashIdx])
				trailingComment = strings.TrimSpace(valueWithComment[hashIdx+1:])
			} else {
				// It's a quoted value, strip quotes and keep the whole thing
				value = strings.TrimPrefix(strings.TrimSuffix(valueWithComment, "\""), "\"")
			}
		}

		tags := make(map[string]string)
		if lastComment != "" {
			// 1. Parentheses syntax: @tag(val) e.g. @type(url), @enum(dev,prod), @requires(DB_HOST)
			for _, m := range tagParenRegex.FindAllStringSubmatch(lastComment, -1) {
				tags[strings.ToLower(m[1])] = strings.TrimSpace(m[2])
			}

			// 2. Equals syntax: @tag=val e.g. @type=int, @range=1-100
			for _, m := range tagEqRegex.FindAllStringSubmatch(lastComment, -1) {
				tags[strings.ToLower(m[1])] = strings.TrimSpace(m[2])
			}

			// 3. Standalone boolean flags e.g. @required
			for _, m := range tagStandaloneRegex.FindAllStringSubmatch(lastComment, -1) {
				tagName := strings.ToLower(m[1])
				if _, exists := tags[tagName]; !exists {
					tags[tagName] = "true"
				}
			}
		}

		// Required if value is empty or has a "your-" prefix, or if @required tag exists
		required := value == "" || strings.HasPrefix(value, "your-")
		if val, ok := tags["required"]; ok {
			required = val == "true" || val == "yes" || val == "1"
		}

		vars = append(vars, EnvVar{
			Key:      key,
			Value:    value,
			Comment:  lastComment,
			Required: required,
			Category: currentCategory,
			Tags:     tags,
			LineNum:  lineNum,
			Ignored:  strings.Contains(lastComment, "razify:ignore") || strings.Contains(trailingComment, "razify:ignore"),
		})

		lastComment = ""
	}

	return vars, scanner.Err()
}

// StripTags removes @tag=value substrings from a comment string.
func StripTags(comment string) string {
	return strings.TrimSpace(tagStripRegex.ReplaceAllString(comment, ""))
}

