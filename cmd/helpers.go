package cmd

import (
	"bufio"
	"os"
	"regexp"
	"strings"
)

type EnvVar struct {
	Key      string
	Value    string
	Comment  string
	Required bool
	Category string
	Tags     map[string]string
	LineNum  int
	Ignored  bool
}

func parseEnvFile(filename string) (map[string]string, error) {
	file, err := os.Open(filename)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	result := map[string]string{}
	scanner := bufio.NewScanner(file)
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

func ParseEnvWithMetadata(filename string) ([]EnvVar, error) {
	file, err := os.Open(filename)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	var vars []EnvVar
	var lastComment string
	var currentCategory string
	lineNum := 0

	scanner := bufio.NewScanner(file)
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
			// Parse tags like @type=int @range=1-100
			tagRegex := regexp.MustCompile(`@(\w+)=([\w\-\.,]+)`)
			matches := tagRegex.FindAllStringSubmatch(lastComment, -1)
			for _, m := range matches {
				tags[m[1]] = m[2]
			}
		}

		// Required if value is empty or has a "your-" prefix, or if @required=true tag exists
		required := value == "" || strings.HasPrefix(value, "your-")
		if val, ok := tags["required"]; ok {
			required = val == "true"
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
